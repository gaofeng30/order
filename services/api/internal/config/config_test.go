package config

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const configCanarySecret = "config-canary-secret-must-not-leak"

const (
	configMiniProgramAppID     = "wx-config-test-app-id"
	configMiniProgramAppSecret = "config-miniprogram-secret-canary"
)

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
	for _, variable := range []string{"ORDER_DB_PASSWORD", "ORDER_DB_DSN", "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"} {
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

func TestLoadRejectsProductionUntilSSMChangeExists(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ORDER_ENV", "production")
	t.Setenv("ORDER_DB_HOST", "db.internal")
	t.Setenv("ORDER_DB_PORT", "3306")
	t.Setenv("ORDER_DB_NAME", "orders")
	t.Setenv("ORDER_DB_USER", "order_api")
	t.Setenv("ORDER_DB_TLS_MODE", "required")

	_, err := Load()
	assertConfigReason(t, err, "production_secret_source_unavailable")
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
