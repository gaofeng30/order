package wechatpay

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseRefundNotificationVerifiesDecryptsAndPreservesProviderFacts(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey: merchantKey, WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key: []byte(testAPIv3Key),
	}, http.DefaultClient, "https://provider.invalid", func() time.Time { return time.Unix(1_800_000_000, 0).UTC() }, func() (string, error) { return "unused", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}

	resource := `{"mchid":"wrong-but-verified-mchid","transaction_id":"TX_TEST_001","out_trade_no":"ORDER_TEST_001","refund_id":"REFUND_PROVIDER_001","out_refund_no":"wrong-but-verified-refund","refund_status":"SUCCESS","success_time":"2027-01-15T08:02:00+08:00","user_received_account":"synthetic account","amount":{"total":7,"refund":3,"payer_total":7,"payer_refund":3}}`
	body := refundNotificationBody(t, resource, "REFUND.SUCCESS", "")
	headers := signedTestHeaders(t, providerKey, body, "1800000000", "callback-nonce")

	got, err := client.ParseRefundNotification(body, headers)
	if err != nil {
		t.Fatal("valid refund notification failed")
	}
	if got.ID != "REFUND_NOTICE_TEST_001" || got.EventType != "REFUND.SUCCESS" || got.MerchantID != "wrong-but-verified-mchid" ||
		got.Refund.OutTradeNo != "ORDER_TEST_001" || got.Refund.OutRefundNo != "wrong-but-verified-refund" ||
		got.Refund.RefundID != "REFUND_PROVIDER_001" || got.Refund.Status != "SUCCESS" ||
		got.Refund.Amount.Total != 7 || got.Refund.Amount.Refund != 3 || got.Refund.Amount.Currency != "CNY" ||
		!got.Refund.SuccessTime.Equal(time.Date(2027, 1, 15, 0, 2, 0, 0, time.UTC)) {
		t.Fatal("refund notification provider facts were not preserved")
	}

	t.Run("tampered signature", func(t *testing.T) {
		t.Parallel()
		badHeaders := headers
		badHeaders.Signature = "tampered"
		_, err := client.ParseRefundNotification(body, badHeaders)
		assertWeChatPayErrorKind(t, err, ErrorSignature)
	})
	t.Run("unknown decrypted field", func(t *testing.T) {
		t.Parallel()
		badResource := strings.TrimSuffix(resource, "}") + `,"unexpected":true}`
		badBody := refundNotificationBody(t, badResource, "REFUND.SUCCESS", "")
		_, err := client.ParseRefundNotification(badBody, signedTestHeaders(t, providerKey, badBody, "1800000000", "callback-nonce"))
		assertWeChatPayErrorKind(t, err, ErrorProtocol)
	})
	t.Run("event state mismatch", func(t *testing.T) {
		t.Parallel()
		badBody := refundNotificationBody(t, resource, "REFUND.CLOSED", "")
		_, err := client.ParseRefundNotification(badBody, signedTestHeaders(t, providerKey, badBody, "1800000000", "callback-nonce"))
		assertWeChatPayErrorKind(t, err, ErrorProtocol)
	})
	t.Run("missing success time", func(t *testing.T) {
		t.Parallel()
		badResource := strings.Replace(resource, `"success_time":"2027-01-15T08:02:00+08:00",`, "", 1)
		badBody := refundNotificationBody(t, badResource, "REFUND.SUCCESS", "")
		_, err := client.ParseRefundNotification(badBody, signedTestHeaders(t, providerKey, badBody, "1800000000", "callback-nonce"))
		assertWeChatPayErrorKind(t, err, ErrorProtocol)
	})
	t.Run("closed is terminal without success time", func(t *testing.T) {
		t.Parallel()
		closedResource := strings.Replace(resource, `"refund_status":"SUCCESS","success_time":"2027-01-15T08:02:00+08:00",`, `"refund_status":"CLOSED",`, 1)
		closedBody := refundNotificationBody(t, closedResource, "REFUND.CLOSED", "")
		got, err := client.ParseRefundNotification(closedBody, signedTestHeaders(t, providerKey, closedBody, "1800000000", "callback-nonce"))
		if err != nil || got.Refund.Status != "CLOSED" || !got.Refund.SuccessTime.IsZero() {
			t.Fatal("valid closed refund notification failed")
		}
	})
	t.Run("omitted optional associated data", func(t *testing.T) {
		t.Parallel()
		body := refundNotificationBodyWithAAD(t, resource, "REFUND.SUCCESS", "", false, "")
		got, err := client.ParseRefundNotification(body, signedTestHeaders(t, providerKey, body, "1800000000", "callback-nonce"))
		if err != nil || got.Refund.RefundID != "REFUND_PROVIDER_001" {
			t.Fatal("omitted optional callback associated_data was rejected")
		}
	})
}

func refundNotificationBody(t *testing.T, resource, eventType, extraOuter string) []byte {
	t.Helper()
	return refundNotificationBodyWithAAD(t, resource, eventType, "refund", true, extraOuter)
}

func refundNotificationBodyWithAAD(t *testing.T, resource, eventType, associatedData string, includeAssociatedData bool, extraOuter string) []byte {
	t.Helper()
	ciphertext := encryptTestResource(t, []byte(testAPIv3Key), "nonce-123456", associatedData, []byte(resource))
	aadField := ""
	if includeAssociatedData {
		aadField = `,"associated_data":"` + associatedData + `"`
	}
	return []byte(`{"id":"REFUND_NOTICE_TEST_001","create_time":"2027-01-15T08:00:00Z","resource_type":"encrypt-resource","event_type":"` + eventType + `","summary":"synthetic refund","resource":{"original_type":"refund","algorithm":"AEAD_AES_256_GCM","ciphertext":"` + ciphertext + `"` + aadField + `,"nonce":"nonce-123456"}` + extraOuter + `}`)
}

func assertWeChatPayErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var providerError *Error
	if !errors.As(err, &providerError) || providerError.Kind() != want {
		t.Fatalf("error kind = %v, want %v", err, want)
	}
}
