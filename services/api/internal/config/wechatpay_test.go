package config

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
)

const weChatPayTestCanary = "wechatpay-secret-canary-must-not-leak"

type recordingWeChatPaySecretSource struct {
	value string
	err   error
	name  string
}

func (source *recordingWeChatPaySecretSource) Get(_ context.Context, name string) (string, error) {
	source.name = name
	return source.value, source.err
}

func TestLoadWeChatPayMaterialBuildsOfficialClientFromSingleSealedSecret(t *testing.T) {
	secret := validWeChatPaySecret(t)
	source := &recordingWeChatPaySecretSource{value: secret}
	material, err := LoadWeChatPayMaterial(context.Background(), source)
	if err != nil {
		t.Fatalf("LoadWeChatPayMaterial() = %v", err)
	}
	if source.name != weChatPaySecretName {
		t.Fatalf("SecretSource name = %q", source.name)
	}
	if material.MerchantID() != "1900000109" || material.NotifyURL() != "https://order.example.com/api/v1/payments/wechat/notify" {
		t.Fatalf("public material facts = %q/%q", material.MerchantID(), material.NotifyURL())
	}
	client, err := material.NewClient("wx-production-app")
	if err != nil || client == nil {
		t.Fatalf("NewClient() = %T/%v", client, err)
	}
}

func TestLoadWeChatPayMaterialRejectsUnsafeOrLeakySecrets(t *testing.T) {
	valid := validWeChatPaySecretMap(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "unknown field", mutate: func(value map[string]any) { value["raw_secret"] = weChatPayTestCanary }},
		{name: "short api key", mutate: func(value map[string]any) { value["api_v3_key"] = "short-" + weChatPayTestCanary }},
		{name: "private key", mutate: func(value map[string]any) { value["merchant_private_key_pem"] = weChatPayTestCanary }},
		{name: "insecure notify", mutate: func(value map[string]any) {
			value["notify_url"] = "http://" + weChatPayTestCanary + "/api/v1/payments/wechat/notify"
		}},
		{name: "duplicate platform serial", mutate: func(value map[string]any) {
			keys := value["wechatpay_public_keys"].([]map[string]string)
			value["wechatpay_public_keys"] = append(keys, keys[0])
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copyValue := cloneWeChatPaySecretMap(t, valid)
			test.mutate(copyValue)
			encoded, err := json.Marshal(copyValue)
			if err != nil {
				t.Fatal(err)
			}
			_, err = LoadWeChatPayMaterial(context.Background(), &recordingWeChatPaySecretSource{value: string(encoded)})
			if err == nil || strings.Contains(err.Error(), weChatPayTestCanary) {
				t.Fatalf("LoadWeChatPayMaterial(invalid) = %v", err)
			}
		})
	}

	providerError := errors.New("provider detail " + weChatPayTestCanary)
	_, err := LoadWeChatPayMaterial(context.Background(), &recordingWeChatPaySecretSource{err: providerError})
	if err == nil || strings.Contains(err.Error(), weChatPayTestCanary) {
		t.Fatalf("LoadWeChatPayMaterial(provider error) = %v", err)
	}
}

func validWeChatPaySecret(t *testing.T) string {
	t.Helper()
	encoded, err := json.Marshal(validWeChatPaySecretMap(t))
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func validWeChatPaySecretMap(t *testing.T) map[string]any {
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
	return map[string]any{
		"merchant_id":                 "1900000109",
		"merchant_certificate_serial": "MERCHANT_SERIAL_001",
		"merchant_private_key_pem":    string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: merchantDER})),
		"wechatpay_public_keys": []map[string]string{{
			"serial": "PUB_KEY_ID_001", "public_key_pem": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: providerDER})),
		}},
		"api_v3_key": "0123456789abcdef0123456789abcdef",
		"notify_url": "https://order.example.com/api/v1/payments/wechat/notify",
	}
}

func cloneWeChatPaySecretMap(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	keys := result["wechatpay_public_keys"].([]any)
	typedKeys := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		entry := key.(map[string]any)
		typedKeys = append(typedKeys, map[string]string{"serial": entry["serial"].(string), "public_key_pem": entry["public_key_pem"].(string)})
	}
	result["wechatpay_public_keys"] = typedKeys
	return result
}
