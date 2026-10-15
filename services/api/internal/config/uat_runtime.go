package config

import (
	"os"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

// CloudRunRuntimeConfig carries the fixed ordinary-PG and private-COS runtime inputs.
// HTTP.Database is deliberately empty; the CloudRun entry never opens a native DB.
type CloudRunRuntimeConfig struct {
	PublicOrigin       string
	HTTP               Config
	EnvID              string
	SharedPG           cloudbasepg.Credentials
	COS                COSMaterial
	COSSourceSecretID  string
	COSSourceSecretKey string
}

func LoadCloudRunRuntime() (CloudRunRuntimeConfig, error) {
	environment := Environment(os.Getenv("ORDER_ENV"))
	envID, bucket, sourcePrefix := "order-uat-d1gtzpzj50615c300", uatCOSBucket, "ORDER_UAT_COS_"
	if environment == Production {
		envID, bucket, sourcePrefix = "order-prod-d0g4xnw49c5ada5ce", productionCOSBucket, "ORDER_PRODUCTION_COS_"
		for _, key := range []string{"ORDER_UAT_COS_SECRET_ID", "ORDER_UAT_COS_SECRET_KEY"} {
			if _, present := os.LookupEnv(key); present {
				return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_environment_mismatch", field: key}
			}
		}
	} else if environment != UAT {
		return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_environment_invalid"}
	}
	publicOrigin := "https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com"
	if environment == Production {
		publicOrigin = os.Getenv("ORDER_PUBLIC_ORIGIN")
		if publicOrigin != "https://order.suianwuye.com" {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_public_origin_invalid"}
		}
	} else {
		for _, key := range []string{"ORDER_PRODUCTION_COS_SECRET_ID", "ORDER_PRODUCTION_COS_SECRET_KEY"} {
			if _, present := os.LookupEnv(key); present {
				return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_environment_mismatch", field: key}
			}
		}
	}
	uid := os.Getenv("ORDER_SHARED_PG_SERVICE_UID")
	if !validDigits(uid, 1, 64) || (environment == Production && uid == "2108167831809454081") || (environment == UAT && uid != "2108167831809454081") {
		return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_service_identity_invalid"}
	}

	for _, field := range []string{
		"ORDER_DB_HOST", "ORDER_DB_PORT", "ORDER_DB_NAME", "ORDER_DB_USER",
		"ORDER_DB_PASSWORD", "ORDER_DB_TLS_MODE", "ORDER_DB_DSN", "CLOUDBASE_APIKEY",
		"TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY", "TENCENTCLOUD_TOKEN", "TENCENTCLOUD_CREDENTIALS_FILE",
	} {
		if _, present := os.LookupEnv(field); present {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_input_forbidden", field: field}
		}
	}
	for _, expected := range []struct{ field, value string }{
		{"ORDER_ENV", string(environment)},
		{"CLOUDBASE_ENV_ID", envID},
		{"ORDER_TENCENT_REGION", "ap-shanghai"},
		{"ORDER_COS_BUCKET", bucket},
		{"ORDER_SHARED_PG_SERVICE_UID", uid},
		{"ORDER_WECHAT_MINIPROGRAM_APP_ID", "wx15e136b256d4dc5a"},
	} {
		if os.Getenv(expected.field) != expected.value {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_configuration_invalid", field: expected.field}
		}
	}
	for _, field := range []string{
		"ORDER_SHARED_PG_USERNAME", "ORDER_SHARED_PG_PASSWORD", "ORDER_SHARED_PG_PUBLISHABLE_KEY",
	} {
		value := os.Getenv(field)
		if strings.TrimSpace(value) == "" || !validProductionSecret(value, 16<<10) {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_configuration_invalid", field: field}
		}
	}
	for _, field := range []string{sourcePrefix + "SECRET_ID", sourcePrefix + "SECRET_KEY"} {
		if !validCredentialComponent(os.Getenv(field)) {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_configuration_invalid", field: field}
		}
	}
	miniProgram := wechat.Credentials{
		AppID:     os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_ID"),
		AppSecret: os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_SECRET"),
	}
	if !validProductionSecret(miniProgram.AppSecret, maxMiniProgramSecret) || validateMiniProgramField("ORDER_WECHAT_MINIPROGRAM_APP_SECRET", miniProgram.AppSecret, maxMiniProgramSecret) != nil {
		return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_configuration_invalid", field: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"}
	}
	cosMaterial, err := ParseCOSMaterial(os.Getenv("ORDER_COS_BUCKET"), os.Getenv("ORDER_TENCENT_REGION"))
	if err != nil {
		return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_runtime_configuration_invalid"}
	}
	httpConfig := Config{HTTPAddr: defaultHTTPAddr, ShutdownTimeout: defaultShutdownTimeout, Environment: environment, MiniProgram: miniProgram}
	if value, present := os.LookupEnv("ORDER_API_HTTP_ADDR"); present {
		if value == "" {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_http_configuration_invalid", field: "ORDER_API_HTTP_ADDR"}
		}
		httpConfig.HTTPAddr = value
	}
	if value, present := os.LookupEnv("ORDER_API_SHUTDOWN_TIMEOUT"); present {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return CloudRunRuntimeConfig{}, loadError{reason: "cloudrun_http_configuration_invalid", field: "ORDER_API_SHUTDOWN_TIMEOUT"}
		}
		httpConfig.ShutdownTimeout = timeout
	}
	return CloudRunRuntimeConfig{
		PublicOrigin: publicOrigin, HTTP: httpConfig, EnvID: os.Getenv("CLOUDBASE_ENV_ID"),
		SharedPG: cloudbasepg.Credentials{
			Username: os.Getenv("ORDER_SHARED_PG_USERNAME"), Password: os.Getenv("ORDER_SHARED_PG_PASSWORD"),
			PublishableKey: os.Getenv("ORDER_SHARED_PG_PUBLISHABLE_KEY"), ServiceUID: os.Getenv("ORDER_SHARED_PG_SERVICE_UID"),
		},
		COS: cosMaterial, COSSourceSecretID: os.Getenv(sourcePrefix + "SECRET_ID"), COSSourceSecretKey: os.Getenv(sourcePrefix + "SECRET_KEY"),
	}, nil
}
