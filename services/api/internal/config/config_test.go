package config

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	ssm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssm/v20190923"
)

const configCanarySecret = "config-canary-secret-must-not-leak"

const (
	configMiniProgramAppID     = "wx-config-test-app-id"
	configMiniProgramAppSecret = "config-miniprogram-secret-canary"
)

type fakeSecretSource struct {
	values    map[string]string
	requested []string
	err       error
}

type fakeCredentialProvider struct {
	credential common.CredentialIface
	err        error
	calls      int
}

func (provider *fakeCredentialProvider) GetCredential() (common.CredentialIface, error) {
	provider.calls++
	return provider.credential, provider.err
}

type fakeSSMClient struct {
	request   *ssm.GetSecretValueRequest
	value     string
	response  *ssm.GetSecretValueResponse
	err       error
	returnNil bool
}

func (client *fakeSSMClient) GetSecretValueWithContext(_ context.Context, request *ssm.GetSecretValueRequest) (*ssm.GetSecretValueResponse, error) {
	client.request = request
	if client.err != nil || client.returnNil {
		return nil, client.err
	}
	if client.response != nil {
		return client.response, nil
	}
	return &ssm.GetSecretValueResponse{Response: &ssm.GetSecretValueResponseParams{SecretString: &client.value}}, nil
}

func (source *fakeSecretSource) Get(_ context.Context, name string) (string, error) {
	source.requested = append(source.requested, name)
	if source.err != nil {
		return "", source.err
	}
	return source.values[name], nil
}

func TestLoadDefaultsWithStructuredDevelopmentDatabase(t *testing.T) {
	clearConfigEnv(t)
	setValidDatabaseEnv(t, "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.Environment != Development {
		t.Fatalf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.Database.Host != "127.0.0.1" || cfg.Database.Port != 3306 || cfg.Database.Database != "order_test" || cfg.Database.User != "order_test" || cfg.Database.Password != configCanarySecret || cfg.Database.TLSMode != "disabled" {
		t.Fatalf("structured database config was not preserved")
	}
}

func TestLoadMiniProgramCredentials(t *testing.T) {
	clearConfigEnv(t)
	setValidDatabaseEnv(t, "test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MiniProgram.AppID != configMiniProgramAppID || cfg.MiniProgram.AppSecret != configMiniProgramAppSecret {
		t.Fatal("structured Mini Program credentials were not preserved")
	}
}

func TestLoadProductionSecretsFromCVMSecretSource(t *testing.T) {
	clearConfigEnv(t)
	setValidProductionEnv(t)
	source := &fakeSecretSource{values: map[string]string{
		"order-production-db-password":                   configCanarySecret,
		"order-production-wechat-miniprogram-app-secret": configMiniProgramAppSecret,
	}}

	cfg, err := LoadWithSecretSource(source)
	if err != nil {
		t.Fatalf("LoadWithSecretSource() error = %v", err)
	}
	if cfg.Database.Password != configCanarySecret || cfg.MiniProgram.AppSecret != configMiniProgramAppSecret {
		t.Fatal("production secrets were not preserved")
	}
	wantNames := []string{"order-production-db-password", "order-production-wechat-miniprogram-app-secret"}
	if strings.Join(source.requested, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("requested secrets = %q, want %q", source.requested, wantNames)
	}
}

func TestLoadProductionRejectsInvalidTencentRegion(t *testing.T) {
	for _, value := range []string{"", "guangzhou", "ap-guangzhou/invalid", "AP-GUANGZHOU"} {
		t.Run(value, func(t *testing.T) {
			clearConfigEnv(t)
			setValidProductionEnv(t)
			t.Setenv("ORDER_TENCENT_REGION", value)
			source := &fakeSecretSource{values: map[string]string{
				"order-production-db-password":                   configCanarySecret,
				"order-production-wechat-miniprogram-app-secret": configMiniProgramAppSecret,
			}}

			_, err := LoadWithSecretSource(source)
			assertConfigReason(t, err, "invalid_tencent_region")
			if !strings.Contains(err.Error(), "ORDER_TENCENT_REGION") {
				t.Fatalf("error = %q, want field name", err)
			}
		})
	}
}

func TestLoadProductionRejectsEmptyDatabaseSecret(t *testing.T) {
	clearConfigEnv(t)
	setValidProductionEnv(t)
	source := &fakeSecretSource{values: map[string]string{
		"order-production-db-password":                   "",
		"order-production-wechat-miniprogram-app-secret": configMiniProgramAppSecret,
	}}

	_, err := LoadWithSecretSource(source)
	assertConfigReason(t, err, "production_secret_value_invalid")
	assertNotContains(t, err.Error(), "order-production-db-password", "database SecretName")
}

func TestLoadProductionRejectsMalformedMiniProgramSecretWithoutLeakage(t *testing.T) {
	clearConfigEnv(t)
	setValidProductionEnv(t)
	malformed := "secret-微信-" + configCanarySecret
	source := &fakeSecretSource{values: map[string]string{
		"order-production-db-password":                   configCanarySecret,
		"order-production-wechat-miniprogram-app-secret": malformed,
	}}

	_, err := LoadWithSecretSource(source)
	assertConfigReason(t, err, "production_secret_value_invalid")
	assertNotContains(t, err.Error(), malformed, "Mini Program secret")
	assertNotContains(t, err.Error(), "order-production-wechat-miniprogram-app-secret", "Mini Program SecretName")
}

func TestCVMSSMSecretSourceReadsCurrentTextSecret(t *testing.T) {
	provider := &fakeCredentialProvider{credential: common.NewTokenCredential("temporary-id", "temporary-key", "temporary-token")}
	client := &fakeSSMClient{value: configCanarySecret}
	source := newCVMSSMSecretSource("ap-guangzhou", provider, func(_ common.CredentialIface, region string) (ssmSecretClient, error) {
		if region != "ap-guangzhou" {
			t.Fatalf("client region = %q, want ap-guangzhou", region)
		}
		return client, nil
	})

	value, err := source.Get(context.Background(), "order-production-db-password")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != configCanarySecret {
		t.Fatal("Get() did not preserve the secret value")
	}
	if provider.calls != 1 {
		t.Fatalf("credential provider calls = %d, want 1", provider.calls)
	}
	if client.request == nil || client.request.SecretName == nil || *client.request.SecretName != "order-production-db-password" {
		t.Fatal("GetSecretValue SecretName was not preserved")
	}
	if client.request.VersionId == nil || *client.request.VersionId != "SSM_Current" {
		t.Fatal("GetSecretValue VersionId was not SSM_Current")
	}
}

func TestLoadProductionMapsProviderFailureWithoutLeakage(t *testing.T) {
	clearConfigEnv(t)
	setValidProductionEnv(t)
	providerDetail := "provider-raw-" + configCanarySecret + "-order-production-db-password"

	_, err := LoadWithSecretSource(&fakeSecretSource{err: errors.New(providerDetail)})
	assertConfigReason(t, err, "production_secret_source_unavailable")
	assertNotContains(t, err.Error(), providerDetail, "provider failure")
}

func TestCVMSSMSecretSourceRejectsProviderAndResponseFailuresWithoutLeakage(t *testing.T) {
	providerDetail := "provider-raw-" + configCanarySecret
	binaryValue := "binary-" + configCanarySecret
	tests := []struct {
		name     string
		provider *fakeCredentialProvider
		factory  ssmClientFactory
	}{
		{
			name:     "credential provider",
			provider: &fakeCredentialProvider{err: errors.New(providerDetail)},
			factory: func(common.CredentialIface, string) (ssmSecretClient, error) {
				return &fakeSSMClient{}, nil
			},
		},
		{
			name:     "client construction",
			provider: &fakeCredentialProvider{credential: common.NewTokenCredential("temporary-id", "temporary-key", "temporary-token")},
			factory: func(common.CredentialIface, string) (ssmSecretClient, error) {
				return nil, errors.New(providerDetail)
			},
		},
		{
			name:     "ssm request",
			provider: &fakeCredentialProvider{credential: common.NewTokenCredential("temporary-id", "temporary-key", "temporary-token")},
			factory: func(common.CredentialIface, string) (ssmSecretClient, error) {
				return &fakeSSMClient{err: errors.New(providerDetail)}, nil
			},
		},
		{
			name:     "binary secret",
			provider: &fakeCredentialProvider{credential: common.NewTokenCredential("temporary-id", "temporary-key", "temporary-token")},
			factory: func(common.CredentialIface, string) (ssmSecretClient, error) {
				return &fakeSSMClient{response: &ssm.GetSecretValueResponse{Response: &ssm.GetSecretValueResponseParams{SecretBinary: &binaryValue}}}, nil
			},
		},
		{
			name:     "empty response",
			provider: &fakeCredentialProvider{credential: common.NewTokenCredential("temporary-id", "temporary-key", "temporary-token")},
			factory: func(common.CredentialIface, string) (ssmSecretClient, error) {
				return &fakeSSMClient{returnNil: true}, nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := newCVMSSMSecretSource("ap-guangzhou", test.provider, test.factory)
			_, err := source.Get(context.Background(), "order-production-db-password")
			if !errors.Is(err, errProductionSecretUnavailable) {
				t.Fatalf("Get() error = %v, want stable unavailable error", err)
			}
			for _, forbidden := range []string{providerDetail, binaryValue, "temporary-id", "temporary-key", "temporary-token", "order-production-db-password"} {
				assertNotContains(t, err.Error(), forbidden, test.name)
			}
		})
	}
}

func TestLoadMiniProgramRejectsInvalidValuesWithoutLeakage(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "missing app id", variable: "ORDER_WECHAT_MINIPROGRAM_APP_ID", value: ""},
		{name: "missing app secret", variable: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET", value: ""},
		{name: "app id space", variable: "ORDER_WECHAT_MINIPROGRAM_APP_ID", value: "wx invalid " + configCanarySecret},
		{name: "app secret unicode", variable: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET", value: "secret-微信-" + configCanarySecret},
		{name: "app id too long", variable: "ORDER_WECHAT_MINIPROGRAM_APP_ID", value: strings.Repeat("a", 129) + configCanarySecret},
		{name: "app secret too long", variable: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET", value: strings.Repeat("b", 257) + configCanarySecret},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnv(t)
			setValidDatabaseEnv(t, "development")
			t.Setenv(test.variable, test.value)

			_, err := Load()
			assertConfigReason(t, err, "invalid_miniprogram_field")
			if !strings.Contains(err.Error(), test.variable) {
				t.Fatalf("error = %q, want field name %s", err, test.variable)
			}
			assertNotContains(t, err.Error(), configCanarySecret, test.variable)
		})
	}
}

func TestLoadOverridesHTTPConfiguration(t *testing.T) {
	clearConfigEnv(t)
	setValidDatabaseEnv(t, "test")
	t.Setenv("ORDER_API_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" || cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("HTTP config = %q/%s, want override", cfg.HTTPAddr, cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsRawDSNWithoutLeakingIt(t *testing.T) {
	clearConfigEnv(t)
	setValidDatabaseEnv(t, "development")
	t.Setenv("ORDER_DB_DSN", "user:"+configCanarySecret+"@tcp(example.invalid:3306)/orders")

	_, err := Load()
	assertConfigReason(t, err, "raw_dsn_unsupported")
	assertNotContains(t, err.Error(), configCanarySecret, "raw DSN")
	assertNotContains(t, err.Error(), "example.invalid", "raw DSN host")
}

func TestLoadRejectsProductionEnvironmentSecrets(t *testing.T) {
	for _, variable := range []string{
		"ORDER_DB_PASSWORD",
		"ORDER_DB_DSN",
		"ORDER_WECHAT_MINIPROGRAM_APP_SECRET",
		"TENCENTCLOUD_SECRET_ID",
		"TENCENTCLOUD_SECRET_KEY",
		"TENCENTCLOUD_TOKEN",
		"TENCENTCLOUD_CREDENTIALS_FILE",
	} {
		t.Run(variable, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv("ORDER_ENV", "production")
			t.Setenv(variable, configCanarySecret)

			_, err := Load()
			assertConfigReason(t, err, "production_secret_environment_forbidden")
			assertNotContains(t, err.Error(), configCanarySecret, variable)
		})
	}
}

func TestLoadRejectsInvalidStructuredDatabaseFieldsWithoutLeakage(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "environment", variable: "ORDER_ENV", value: "staging"},
		{name: "host empty", variable: "ORDER_DB_HOST", value: ""},
		{name: "host scheme", variable: "ORDER_DB_HOST", value: "mysql://" + configCanarySecret},
		{name: "port text", variable: "ORDER_DB_PORT", value: "not-a-port"},
		{name: "port zero", variable: "ORDER_DB_PORT", value: "0"},
		{name: "port overflow", variable: "ORDER_DB_PORT", value: strconv.Itoa(1 << 16)},
		{name: "database", variable: "ORDER_DB_NAME", value: ""},
		{name: "user", variable: "ORDER_DB_USER", value: ""},
		{name: "password", variable: "ORDER_DB_PASSWORD", value: ""},
		{name: "tls", variable: "ORDER_DB_TLS_MODE", value: "skip-verify-" + configCanarySecret},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnv(t)
			setValidDatabaseEnv(t, "development")
			t.Setenv(test.variable, test.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want rejection")
			}
			if !strings.Contains(err.Error(), test.variable) && test.variable != "ORDER_ENV" {
				t.Fatalf("error = %q, want field name %s", err, test.variable)
			}
			assertNotContains(t, err.Error(), configCanarySecret, test.variable)
		})
	}
}

func TestLoadRejectsInvalidShutdownTimeout(t *testing.T) {
	clearConfigEnv(t)
	setValidDatabaseEnv(t, "development")
	t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ORDER_API_SHUTDOWN_TIMEOUT") {
		t.Fatalf("Load() error = %v, want named configuration error", err)
	}
}

func TestLoadRejectsNonPositiveShutdownTimeout(t *testing.T) {
	for _, value := range []string{"0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			clearConfigEnv(t)
			setValidDatabaseEnv(t, "development")
			t.Setenv("ORDER_API_SHUTDOWN_TIMEOUT", value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "ORDER_API_SHUTDOWN_TIMEOUT") {
				t.Fatalf("Load() error = %v, want named positive-duration error", err)
			}
		})
	}
}

func setValidDatabaseEnv(t *testing.T, environment string) {
	t.Helper()
	t.Setenv("ORDER_ENV", environment)
	t.Setenv("ORDER_DB_HOST", "127.0.0.1")
	t.Setenv("ORDER_DB_PORT", "3306")
	t.Setenv("ORDER_DB_NAME", "order_test")
	t.Setenv("ORDER_DB_USER", "order_test")
	t.Setenv("ORDER_DB_PASSWORD", configCanarySecret)
	t.Setenv("ORDER_DB_TLS_MODE", "disabled")
	t.Setenv("ORDER_WECHAT_MINIPROGRAM_APP_ID", configMiniProgramAppID)
	t.Setenv("ORDER_WECHAT_MINIPROGRAM_APP_SECRET", configMiniProgramAppSecret)
}

func setValidProductionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ORDER_ENV", "production")
	t.Setenv("ORDER_DB_HOST", "db.internal")
	t.Setenv("ORDER_DB_PORT", "3306")
	t.Setenv("ORDER_DB_NAME", "orders")
	t.Setenv("ORDER_DB_USER", "order_api")
	t.Setenv("ORDER_DB_TLS_MODE", "required")
	t.Setenv("ORDER_WECHAT_MINIPROGRAM_APP_ID", configMiniProgramAppID)
	t.Setenv("ORDER_TENCENT_REGION", "ap-guangzhou")
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"ORDER_ENV",
		"ORDER_DB_HOST",
		"ORDER_DB_PORT",
		"ORDER_DB_NAME",
		"ORDER_DB_USER",
		"ORDER_DB_PASSWORD",
		"ORDER_DB_TLS_MODE",
		"ORDER_DB_DSN",
		"ORDER_API_HTTP_ADDR",
		"ORDER_API_SHUTDOWN_TIMEOUT",
		"ORDER_WECHAT_MINIPROGRAM_APP_ID",
		"ORDER_WECHAT_MINIPROGRAM_APP_SECRET",
		"ORDER_TENCENT_REGION",
		"TENCENTCLOUD_SECRET_ID",
		"TENCENTCLOUD_SECRET_KEY",
		"TENCENTCLOUD_TOKEN",
		"TENCENTCLOUD_CREDENTIALS_FILE",
	} {
		unsetEnv(t, key)
	}
}

func assertConfigReason(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Load() error = nil, want %s", want)
	}
	if got := Reason(err); got != want {
		t.Fatalf("Reason(error) = %q, want %q; error=%v", got, want, err)
	}
}

func assertNotContains(t *testing.T, value, forbidden, label string) {
	t.Helper()
	if strings.Contains(value, forbidden) {
		t.Fatalf("%s leaked %q in %q", label, forbidden, value)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, present := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}
