package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
)

type paymentSecretSourceStub struct{ value string }

func (source paymentSecretSourceStub) Get(context.Context, string) (string, error) {
	return source.value, nil
}

func TestComposeProductionWeChatPaymentUsesOfficialAdapterAndRuntimeFacts(t *testing.T) {
	material, err := config.LoadWeChatPayMaterial(context.Background(), paymentSecretSourceStub{value: paymentSecretJSON(t)})
	if err != nil {
		t.Fatal(err)
	}
	provider, paymentConfig, err := composeProductionWeChatPayment("wx-production-app", material)
	if err != nil || provider == nil {
		t.Fatalf("composeProductionWeChatPayment() = %T/%#v/%v", provider, paymentConfig, err)
	}
	if paymentConfig.AppID != "wx-production-app" || paymentConfig.MerchantID != "1900000109" ||
		paymentConfig.Description != "预约点餐" || paymentConfig.PaymentNotifyURL != "https://order.example.com/api/v1/payments/wechat/notify" {
		t.Fatalf("payment config = %#v", paymentConfig)
	}
	var _ paymentorder.PaymentProvider = provider
	var _ paymentorder.NotificationParser = provider
}

func TestComposeProductionWeChatPaymentFailsClosedWithoutAppID(t *testing.T) {
	material, err := config.LoadWeChatPayMaterial(context.Background(), paymentSecretSourceStub{value: paymentSecretJSON(t)})
	if err != nil {
		t.Fatal(err)
	}
	provider, _, err := composeProductionWeChatPayment("", material)
	if err == nil || provider != nil {
		t.Fatalf("composeProductionWeChatPayment(empty appid) = %T/%v", provider, err)
	}
}

func paymentSecretJSON(t *testing.T) string {
	t.Helper()
	merchantKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	providerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	merchantDER, err := x509.MarshalPKCS8PrivateKey(merchantKey)
	if err != nil {
		t.Fatal(err)
	}
	providerDER, err := x509.MarshalPKIXPublicKey(&providerKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	value := map[string]any{
		"merchant_id":                 "1900000109",
		"merchant_certificate_serial": "MERCHANT_SERIAL_001",
		"merchant_private_key_pem":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: merchantDER})),
		"wechatpay_public_keys": []map[string]string{{
			"serial": "PUB_KEY_ID_001", "public_key_pem": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: providerDER})),
		}},
		"api_v3_key": "0123456789abcdef0123456789abcdef",
		"notify_url": "https://order.example.com/api/v1/payments/wechat/notify",
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
