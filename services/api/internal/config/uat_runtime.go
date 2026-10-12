package config

import (
	"os"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

// UATRuntimeConfig carries the fixed ordinary-PG and private-COS runtime inputs.
// HTTP.Database is deliberately empty; the UAT entry never opens a native DB.
type UATRuntimeConfig struct {
	HTTP               Config
	EnvID              string
	SharedPG           cloudbasepg.Credentials
	COS                COSMaterial
	COSSourceSecretID  string
	COSSourceSecretKey string
}

func LoadUATRuntime() (UATRuntimeConfig, error) {
	for _, field := range []string{
		"ORDER_DB_HOST", "ORDER_DB_PORT", "ORDER_DB_NAME", "ORDER_DB_USER",
		"ORDER_DB_PASSWORD", "ORDER_DB_TLS_MODE", "ORDER_DB_DSN", "CLOUDBASE_APIKEY",
		"TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY", "TENCENTCLOUD_TOKEN", "TENCENTCLOUD_CREDENTIALS_FILE",
	} {
		if _, present := os.LookupEnv(field); present {
			return UATRuntimeConfig{}, loadError{reason: "uat_runtime_input_forbidden", field: field}
		}
	}
	for _, expected := range []struct{ field, value string }{
		{"ORDER_ENV", "uat"},
		{"CLOUDBASE_ENV_ID", "order-uat-d1gtzpzj50615c300"},
		{"ORDER_TENCENT_REGION", "ap-shanghai"},
		{"ORDER_COS_BUCKET", "6f72-order-uat-d1gtzpzj50615c300-1473835269"},
		{"ORDER_SHARED_PG_SERVICE_UID", "2108167831809454081"},
		{"ORDER_WECHAT_MINIPROGRAM_APP_ID", "wx15e136b256d4dc5a"},
	} {
		if os.Getenv(expected.field) != expected.value {
			return UATRuntimeConfig{}, loadError{reason: "uat_runtime_configuration_invalid", field: expected.field}
		}
	}
	for _, field := range []string{
		"ORDER_SHARED_PG_USERNAME", "ORDER_SHARED_PG_PASSWORD", "ORDER_SHARED_PG_PUBLISHABLE_KEY",
	} {
		value := os.Getenv(field)
		if strings.TrimSpace(value) == "" || !validProductionSecret(value, 16<<10) {
			return UATRuntimeConfig{}, loadError{reason: "uat_runtime_configuration_invalid", field: field}
		}
	}
	for _, field := range []string{"ORDER_UAT_COS_SECRET_ID", "ORDER_UAT_COS_SECRET_KEY"} {
		if !validCredentialComponent(os.Getenv(field)) {
			return UATRuntimeConfig{}, loadError{reason: "uat_runtime_configuration_invalid", field: field}
		}
	}
	miniProgram := wechat.Credentials{
		AppID:     os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_ID"),
		AppSecret: os.Getenv("ORDER_WECHAT_MINIPROGRAM_APP_SECRET"),
	}
	if !validProductionSecret(miniProgram.AppSecret, maxMiniProgramSecret) || validateMiniProgramField("ORDER_WECHAT_MINIPROGRAM_APP_SECRET", miniProgram.AppSecret, maxMiniProgramSecret) != nil {
		return UATRuntimeConfig{}, loadError{reason: "uat_runtime_configuration_invalid", field: "ORDER_WECHAT_MINIPROGRAM_APP_SECRET"}
	}
	cosMaterial, err := ParseCOSMaterial(os.Getenv("ORDER_COS_BUCKET"), os.Getenv("ORDER_TENCENT_REGION"))
	if err != nil {
		return UATRuntimeConfig{}, loadError{reason: "uat_runtime_configuration_invalid"}
	}
	httpConfig := Config{HTTPAddr: defaultHTTPAddr, ShutdownTimeout: defaultShutdownTimeout, Environment: UAT, MiniProgram: miniProgram}
	if value, present := os.LookupEnv("ORDER_API_HTTP_ADDR"); present {
		if value == "" {
			return UATRuntimeConfig{}, loadError{reason: "uat_http_configuration_invalid", field: "ORDER_API_HTTP_ADDR"}
		}
		httpConfig.HTTPAddr = value
	}
	if value, present := os.LookupEnv("ORDER_API_SHUTDOWN_TIMEOUT"); present {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return UATRuntimeConfig{}, loadError{reason: "uat_http_configuration_invalid", field: "ORDER_API_SHUTDOWN_TIMEOUT"}
		}
		httpConfig.ShutdownTimeout = timeout
	}
	return UATRuntimeConfig{
		HTTP: httpConfig, EnvID: os.Getenv("CLOUDBASE_ENV_ID"),
		SharedPG: cloudbasepg.Credentials{
			Username: os.Getenv("ORDER_SHARED_PG_USERNAME"), Password: os.Getenv("ORDER_SHARED_PG_PASSWORD"),
			PublishableKey: os.Getenv("ORDER_SHARED_PG_PUBLISHABLE_KEY"), ServiceUID: os.Getenv("ORDER_SHARED_PG_SERVICE_UID"),
		},
		COS: cosMaterial, COSSourceSecretID: os.Getenv("ORDER_UAT_COS_SECRET_ID"), COSSourceSecretKey: os.Getenv("ORDER_UAT_COS_SECRET_KEY"),
	}, nil
}
