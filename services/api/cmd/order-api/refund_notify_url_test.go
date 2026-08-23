package main

import "testing"

func TestProductionRefundNotifyURLUsesValidatedPaymentOrigin(t *testing.T) {
	got, err := productionRefundNotifyURL("https://order.example.com/api/v1/payments/wechat/notify")
	if err != nil || got != "https://order.example.com/api/v1/refunds/wechat/notify" {
		t.Fatalf("productionRefundNotifyURL() = %q/%v", got, err)
	}
	for _, invalid := range []string{
		"http://order.example.com/api/v1/payments/wechat/notify",
		"https://order.example.com/other",
		"https://user@order.example.com/api/v1/payments/wechat/notify",
		"https://order.example.com/api/v1/payments/wechat/notify?x=1",
	} {
		if value, err := productionRefundNotifyURL(invalid); err == nil || value != "" {
			t.Fatalf("productionRefundNotifyURL(%q) = %q/%v", invalid, value, err)
		}
	}
}
