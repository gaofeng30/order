package main

import (
	"github.com/gaofeng30/order/services/api/internal/config"
	"testing"
)

func TestCloudRunCallbackAndNotificationBinding(t *testing.T) {
	for _, tc := range []struct {
		name, payment, state string
		allowed              bool
	}{
		{"production", "https://orders.example.invalid/api/v1/payments/wechat/notify", "formal", true},
		{"uat callback", "https://order-uat-d1gtzpzj50615c300-1473835269.ap-shanghai.app.tcloudbase.com/api/v1/payments/wechat/notify", "formal", false},
		{"trial notification", "https://orders.example.invalid/api/v1/payments/wechat/notify", "trial", false},
		{"different origin", "https://other.example.invalid/api/v1/payments/wechat/notify", "formal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.CloudRunRuntimeConfig{HTTP: config.Config{Environment: config.Production}, PublicOrigin: "https://orders.example.invalid"}
			if (validateCloudRunBinding(cfg, tc.payment, tc.state) == nil) != tc.allowed {
				t.Fatal("environment callback/state mismatch")
			}
		})
	}
}
