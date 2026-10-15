package config

import (
	"encoding/base64"
	"testing"
)

func productionRuntimeFixture(t *testing.T) {
	t.Helper()
	const env = "order-prod-d0g4xnw49c5ada5ce"
	for k, v := range map[string]string{
		"ORDER_PUBLIC_ORIGIN": "https://order.suianwuye.com",
		"ORDER_ENV":           "production", "CLOUDBASE_ENV_ID": env, "ORDER_TENCENT_REGION": "ap-shanghai",
		"ORDER_COS_BUCKET": "6f72-" + env + "-1473835269", "ORDER_SHARED_PG_SERVICE_UID": "123",
		"ORDER_WECHAT_MINIPROGRAM_APP_ID": "wx15e136b256d4dc5a", "ORDER_WECHAT_MINIPROGRAM_APP_SECRET": "isolated-app-secret",
		"ORDER_SHARED_PG_USERNAME": "isolated-prod", "ORDER_SHARED_PG_PASSWORD": "isolated-password",
		"ORDER_SHARED_PG_PUBLISHABLE_KEY": "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"role":"anon","aud":"`+env+`"}`)) + ".signature",
		"ORDER_PRODUCTION_COS_SECRET_ID":  "isolated-id", "ORDER_PRODUCTION_COS_SECRET_KEY": "isolated-secret",
	} {
		t.Setenv(k, v)
	}
}

func TestCloudRunProductionIsolation(t *testing.T) {
	t.Run("production ordinary identity", func(t *testing.T) {
		productionRuntimeFixture(t)
		cfg, err := LoadCloudRunRuntime()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.HTTP.Environment != Production || cfg.SharedPG.ServiceUID != "123" {
			t.Fatal("wrong environment or identity")
		}
	})
	for _, tc := range []struct{ key, value string }{
		{"ORDER_PUBLIC_ORIGIN", "https://uat.suianwuye.com"},
		{"ORDER_PUBLIC_ORIGIN", "https://orders.example.invalid"},
		{"ORDER_PUBLIC_ORIGIN", "https://order.suianwuye.com:8443"},
		{"ORDER_PUBLIC_ORIGIN", "https://localhost"},
		{"ORDER_PUBLIC_ORIGIN", "https://127.0.0.1"},
		{"ORDER_PUBLIC_ORIGIN", "https://orders.example.invalid/pc/"},
		{"ORDER_PUBLIC_ORIGIN", "https://orders.example.invalid?x=1"},
		{"CLOUDBASE_ENV_ID", "order-uat-d1gtzpzj50615c300"},
		{"ORDER_COS_BUCKET", "6f72-order-uat-d1gtzpzj50615c300-1473835269"},
		{"ORDER_SHARED_PG_SERVICE_UID", "2108167831809454081"},
		{"ORDER_SHARED_PG_SERVICE_UID", ""},
		{"ORDER_SHARED_PG_SERVICE_UID", "not-numeric"},
		{"ORDER_UAT_COS_SECRET_ID", "cross-environment"},
		{"ORDER_DB_DSN", "forbidden"},
		{"CLOUDBASE_APIKEY", "forbidden"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			productionRuntimeFixture(t)
			t.Setenv(tc.key, tc.value)
			if _, err := LoadCloudRunRuntime(); err == nil {
				t.Fatal("cross-environment or privileged input accepted")
			}
		})
	}
}
