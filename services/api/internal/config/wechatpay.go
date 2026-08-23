package config

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

const (
	weChatPaySecretName    = "order-production-wechatpay-api-v3"
	maxWeChatPaySecretSize = 64 * 1024
	weChatPayNotifyPath    = "/api/v1/payments/wechat/notify"
)

// WeChatPayMaterial retains parsed payment credentials without exposing their
// encoded secret values to runtime configuration, logs or callers.
type WeChatPayMaterial struct {
	merchantID string
	notifyURL  string
	client     wechatpay.Config
}

func (material WeChatPayMaterial) MerchantID() string { return material.merchantID }

func (material WeChatPayMaterial) NotifyURL() string { return material.notifyURL }

func (material WeChatPayMaterial) NewClient(appID string) (*wechatpay.Client, error) {
	if appID == "" || material.merchantID == "" || material.notifyURL == "" {
		return nil, loadError{reason: "production_wechatpay_secret_invalid"}
	}
	clientConfig := material.client
	clientConfig.AppID = appID
	client, err := wechatpay.NewClient(clientConfig)
	if err != nil {
		return nil, loadError{reason: "production_wechatpay_secret_invalid"}
	}
	return client, nil
}

type weChatPaySecret struct {
	MerchantID                string                     `json:"merchant_id"`
	MerchantCertificateSerial string                     `json:"merchant_certificate_serial"`
	MerchantPrivateKeyPEM     string                     `json:"merchant_private_key_pem"`
	WeChatPayPublicKeys       []weChatPayPublicKeySecret `json:"wechatpay_public_keys"`
	APIv3Key                  string                     `json:"api_v3_key"`
	NotifyURL                 string                     `json:"notify_url"`
}

type weChatPayPublicKeySecret struct {
	Serial       string `json:"serial"`
	PublicKeyPEM string `json:"public_key_pem"`
}

// LoadWeChatPayMaterial retrieves and strictly parses one sealed production
// secret. Provider errors and secret values are deliberately collapsed into
// stable, non-sensitive error categories.
func LoadWeChatPayMaterial(ctx context.Context, source SecretSource) (WeChatPayMaterial, error) {
	if ctx == nil || source == nil {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_unavailable"}
	}
	loadContext, cancel := context.WithTimeout(ctx, secretLoadTimeout)
	defer cancel()
	encoded, err := source.Get(loadContext, weChatPaySecretName)
	if err != nil {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_unavailable"}
	}
	if encoded == "" || len(encoded) > maxWeChatPaySecretSize || !utf8.ValidString(encoded) {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_invalid"}
	}
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var secret weChatPaySecret
	if decoder.Decode(&secret) != nil || decodeWeChatPayJSONEnd(decoder) != nil {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_invalid"}
	}
	material, err := parseWeChatPaySecret(secret)
	if err != nil {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_invalid"}
	}
	return material, nil
}

// LoadProductionWeChatPayMaterial binds the narrow parser to the existing CVM
// role backed production SecretSource. The region itself is not secret.
func LoadProductionWeChatPayMaterial(ctx context.Context, region string) (WeChatPayMaterial, error) {
	if !validTencentRegion(region) {
		return WeChatPayMaterial{}, loadError{reason: "production_wechatpay_secret_unavailable"}
	}
	return LoadWeChatPayMaterial(ctx, newProductionSecretSource(region))
}

func parseWeChatPaySecret(secret weChatPaySecret) (WeChatPayMaterial, error) {
	if !validDigits(secret.MerchantID, 6, 32) || !validPaymentToken(secret.MerchantCertificateSerial, 128) ||
		!validAPIv3Key(secret.APIv3Key) || !validWeChatPayNotifyURL(secret.NotifyURL) ||
		len(secret.WeChatPayPublicKeys) == 0 || len(secret.WeChatPayPublicKeys) > 8 {
		return WeChatPayMaterial{}, loadError{}
	}
	merchantKey, err := parseMerchantPrivateKey(secret.MerchantPrivateKeyPEM)
	if err != nil {
		return WeChatPayMaterial{}, err
	}
	providerKeys := make(map[string]*rsa.PublicKey, len(secret.WeChatPayPublicKeys))
	for _, encoded := range secret.WeChatPayPublicKeys {
		if !validPaymentToken(encoded.Serial, 128) {
			return WeChatPayMaterial{}, loadError{}
		}
		if _, duplicate := providerKeys[encoded.Serial]; duplicate {
			return WeChatPayMaterial{}, loadError{}
		}
		key, err := parseWeChatPayPublicKey(encoded.PublicKeyPEM)
		if err != nil {
			return WeChatPayMaterial{}, err
		}
		providerKeys[encoded.Serial] = key
	}
	return WeChatPayMaterial{
		merchantID: secret.MerchantID,
		notifyURL:  secret.NotifyURL,
		client: wechatpay.Config{
			MerchantID: secret.MerchantID, MerchantCertificateSerial: secret.MerchantCertificateSerial,
			MerchantPrivateKey: merchantKey, WeChatPayPublicKeys: providerKeys, APIv3Key: []byte(secret.APIv3Key),
		},
	}, nil
}

func parseMerchantPrivateKey(encoded string) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode([]byte(encoded))
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, loadError{}
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, loadError{}
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() != 2048 || key.E != 65537 || key.Validate() != nil {
		return nil, loadError{}
	}
	return key, nil
}

func parseWeChatPayPublicKey(encoded string) (*rsa.PublicKey, error) {
	block, rest := pem.Decode([]byte(encoded))
	if block == nil || block.Type != "PUBLIC KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, loadError{}
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, loadError{}
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok || key.N.BitLen() != 2048 || key.E != 65537 {
		return nil, loadError{}
	}
	return key, nil
}

func validDigits(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func validPaymentToken(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < '0' || character > '9') && (character < 'A' || character > 'Z') &&
			(character < 'a' || character > 'z') && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func validAPIv3Key(value string) bool {
	if len(value) != 32 {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '!' || value[index] > '~' {
			return false
		}
	}
	return true
}

func validWeChatPayNotifyURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil &&
		parsed.Path == weChatPayNotifyPath && parsed.RawQuery == "" && parsed.Fragment == ""
}

func decodeWeChatPayJSONEnd(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return loadError{}
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
