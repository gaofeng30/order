package wechatpay

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAppID          = "wx-test-app"
	testMerchantID     = "1900000109"
	testMerchantSerial = "MERCHANTSERIAL001"
	testProviderSerial = "PUBKEYID_TEST_001"
	testOpenID         = "synthetic-openid"
	testAPIv3Key       = "0123456789abcdef0123456789abcdef"
)

var (
	testKeysOnce    sync.Once
	testMerchantKey *rsa.PrivateKey
	testProviderKey *rsa.PrivateKey
)

func TestNewClientEnforcesRuntimeTransportPolicy(t *testing.T) {
	merchantKey, providerKey := generatedTestKeys(t)
	transactionBody := []byte(`{"appid":"wx-test-app","mchid":"1900000109","out_trade_no":"ORDER_TEST_001","transaction_id":"TX_TEST_001","trade_type":"JSAPI","trade_state":"SUCCESS","trade_state_desc":"synthetic success","success_time":"2027-01-15T08:00:00Z","payer":{"openid":"synthetic-openid"},"amount":{"total":1,"payer_total":1,"currency":"CNY","payer_currency":"CNY"}}`)
	var requests atomic.Int32
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		index := requests.Add(1)
		if request.Host != "api.mch.weixin.qq.com" || request.ProtoMajor != 1 {
			t.Error("NewClient did not use the fixed origin over HTTP/1")
		}
		if !strings.HasPrefix(request.Header.Get("Authorization"), "WECHATPAY2-SHA256-RSA2048 ") {
			t.Error("NewClient request was not signed")
		}
		if strings.Contains(request.URL.Path, "TIMEOUT_TEST") {
			<-request.Context().Done()
			return
		}
		if request.URL.RequestURI() != "/v3/pay/transactions/out-trade-no/ORDER_TEST_001?mchid=1900000109" {
			t.Error("NewClient request target mismatch")
		}
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		nonce := fmt.Sprintf("runtime-response-%d", index)
		writer.Header().Set("Wechatpay-Timestamp", timestamp)
		writer.Header().Set("Wechatpay-Nonce", nonce)
		writer.Header().Set("Wechatpay-Serial", testProviderSerial)
		writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, timestamp+"\n"+nonce+"\n"+string(transactionBody)+"\n"))
		_, _ = writer.Write(transactionBody)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.StartTLS()
	defer server.Close()

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("default HTTP transport is not configurable for the controlled runtime test")
	}
	controlledTransport := defaultTransport.Clone()
	controlledTransport.Proxy = nil
	controlledTransport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	controlledTransport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	controlledTransport.TLSClientConfig.ServerName = "example.com"
	http.DefaultTransport = controlledTransport
	defer func() { http.DefaultTransport = defaultTransport }()

	client, err := NewClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	})
	if err != nil {
		t.Fatal("public client construction failed")
	}
	for range 2 {
		transaction, err := client.QueryTransactionByOutTradeNo(context.Background(), "ORDER_TEST_001")
		if err != nil || transaction.TransactionID != "TX_TEST_001" {
			t.Fatal("public runtime request failed")
		}
	}
	if connections.Load() != 2 {
		t.Fatalf("runtime connections = %d, want 2 fresh connections", connections.Load())
	}

	started := time.Now()
	_, err = client.QueryTransactionByOutTradeNo(context.Background(), "TIMEOUT_TEST")
	elapsed := time.Since(started)
	var providerError *Error
	if !errors.As(err, &providerError) || providerError.Kind() != ErrorTimeout || elapsed < 4500*time.Millisecond || elapsed > 7*time.Second {
		t.Fatal("public runtime timeout policy mismatch")
	}
}

func TestCreateJSAPIPrepaySignsRequestAndRequestPayment(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0).UTC()
	requestNonce := "request-nonce-001"
	paymentNonce := "payment-nonce-002"
	providerNonce := "provider-nonce-003"
	responseBody := []byte(`{"prepay_id":"wx-test-prepay"}`)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.Method != http.MethodPost || request.URL.RequestURI() != "/v3/pay/transactions/jsapi" {
			t.Error("JSAPI endpoint contract mismatch")
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Error("cannot read controlled request")
			return
		}
		wantBody := `{"appid":"wx-test-app","mchid":"1900000109","description":"synthetic meal","out_trade_no":"ORDER_TEST_001","time_expire":"2027-01-16T08:10:00Z","notify_url":"https://merchant.invalid/api/v1/payments/wechat/notify","amount":{"total":1,"currency":"CNY"},"payer":{"openid":"synthetic-openid"}}`
		if string(body) != wantBody || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("Accept") != "application/json" {
			t.Error("JSAPI request body/header contract mismatch")
		}

		authorization := request.Header.Get("Authorization")
		parts := regexp.MustCompile(`^WECHATPAY2-SHA256-RSA2048 mchid="([^"]+)",nonce_str="([^"]+)",signature="([^"]+)",timestamp="([^"]+)",serial_no="([^"]+)"$`).FindStringSubmatch(authorization)
		if len(parts) != 6 || parts[1] != testMerchantID || parts[2] != requestNonce || parts[4] != "1800000000" || parts[5] != testMerchantSerial {
			t.Error("Authorization metadata contract mismatch")
			return
		}
		canonical := "POST\n/v3/pay/transactions/jsapi\n1800000000\nrequest-nonce-001\n" + wantBody + "\n"
		if !verifiesTestSignature(&merchantKey.PublicKey, canonical, parts[3]) {
			t.Error("Authorization signature mismatch")
		}

		writer.Header().Set("Wechatpay-Timestamp", "1800000000")
		writer.Header().Set("Wechatpay-Nonce", providerNonce)
		writer.Header().Set("Wechatpay-Serial", testProviderSerial)
		writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, "1800000000\n"+providerNonce+"\n"+string(responseBody)+"\n"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(responseBody)
	}))
	defer server.Close()

	nonces := []string{requestNonce, paymentNonce}
	client, err := newClient(Config{
		AppID:                     testAppID,
		MerchantID:                testMerchantID,
		MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:        merchantKey,
		WeChatPayPublicKeys:       map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:                  []byte(testAPIv3Key),
	}, server.Client(), server.URL, func() time.Time { return now }, func() (string, error) {
		nonce := nonces[0]
		nonces = nonces[1:]
		return nonce, nil
	})
	if err != nil {
		t.Fatal("controlled client construction failed")
	}

	result, err := client.CreateJSAPIPrepay(context.Background(), JSAPICreateRequest{
		Description: "synthetic meal",
		OutTradeNo:  "ORDER_TEST_001",
		TimeExpire:  "2027-01-16T08:10:00Z",
		NotifyURL:   "https://merchant.invalid/api/v1/payments/wechat/notify",
		Amount:      Amount{Total: 1, Currency: "CNY"},
		Payer:       Payer{OpenID: testOpenID},
	})
	if err != nil {
		t.Fatal("CreateJSAPIPrepay failed")
	}
	if result.PrepayID != "wx-test-prepay" || result.RequestPayment.TimeStamp != "1800000000" ||
		result.RequestPayment.NonceStr != paymentNonce || result.RequestPayment.Package != "prepay_id=wx-test-prepay" ||
		result.RequestPayment.SignType != "RSA" {
		t.Fatal("requestPayment metadata mismatch")
	}
	paymentMessage := strings.Join([]string{testAppID, "1800000000", paymentNonce, "prepay_id=wx-test-prepay", ""}, "\n")
	if !verifiesTestSignature(&merchantKey.PublicKey, paymentMessage, result.RequestPayment.PaySign) {
		t.Fatal("requestPayment signature mismatch")
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

func TestCreateJSAPIPrepayRejectsUntrustedResponses(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0).UTC()
	validBody := []byte(`{"prepay_id":"wx-test-prepay"}`)

	tests := []struct {
		name       string
		serial     string
		timestamp  string
		body       []byte
		signedBody []byte
		want       ErrorKind
	}{
		{name: "unknown serial", serial: "PUBKEYID_UNKNOWN", timestamp: "1800000000", body: validBody, signedBody: validBody, want: ErrorUnknownSerial},
		{name: "stale timestamp", serial: testProviderSerial, timestamp: "1799999699", body: validBody, signedBody: validBody, want: ErrorTimestamp},
		{name: "future timestamp", serial: testProviderSerial, timestamp: "1800000301", body: validBody, signedBody: validBody, want: ErrorTimestamp},
		{name: "tampered raw body", serial: testProviderSerial, timestamp: "1800000000", body: []byte(`{"prepay_id":"tampered"}`), signedBody: validBody, want: ErrorSignature},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			providerNonce := "provider-response-nonce"
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Wechatpay-Timestamp", test.timestamp)
				writer.Header().Set("Wechatpay-Nonce", providerNonce)
				writer.Header().Set("Wechatpay-Serial", test.serial)
				writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, test.timestamp+"\n"+providerNonce+"\n"+string(test.signedBody)+"\n"))
				_, _ = writer.Write(test.body)
			}))
			defer server.Close()

			client, err := newClient(Config{
				AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
				MerchantPrivateKey:  merchantKey,
				WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
				APIv3Key:            []byte(testAPIv3Key),
			}, server.Client(), server.URL, func() time.Time { return now }, func() (string, error) { return "request-nonce", nil })
			if err != nil {
				t.Fatal("controlled client construction failed")
			}
			_, err = client.CreateJSAPIPrepay(context.Background(), JSAPICreateRequest{})
			var providerError *Error
			if !errors.As(err, &providerError) || providerError.Kind() != test.want || !providerError.Retryable() {
				t.Fatal("untrusted response error classification mismatch")
			}
			if strings.Contains(err.Error(), "tampered") || strings.Contains(err.Error(), "wx-test-prepay") {
				t.Fatal("untrusted response material leaked through error")
			}
		})
	}
}

func TestParseTransactionNotificationVerifiesDecryptsAndRejectsInvalidDTOs(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0).UTC()
	apiV3Key := []byte(testAPIv3Key)
	resourceNonce := "nonce-123456"
	associatedData := "transaction"
	resource := `{"appid":"wx-test-app","mchid":"1900000109","out_trade_no":"ORDER_TEST_001","transaction_id":"TX_TEST_001","trade_type":"JSAPI","trade_state":"SUCCESS","trade_state_desc":"synthetic success","bank_type":"TEST","attach":"synthetic","success_time":"2027-01-15T08:00:00Z","payer":{"openid":"synthetic-openid"},"amount":{"total":1,"payer_total":1,"currency":"CNY","payer_currency":"CNY"}}`
	ciphertext := encryptTestResource(t, apiV3Key, resourceNonce, associatedData, []byte(resource))
	outerBody := notificationBody(ciphertext, associatedData, "")

	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            apiV3Key,
	}, http.DefaultClient, "https://provider.invalid", func() time.Time { return now }, func() (string, error) { return "unused-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}
	headers := signedTestHeaders(t, providerKey, outerBody, "1800000000", "callback-nonce")
	notification, err := client.ParseTransactionNotification(outerBody, headers)
	if err != nil {
		t.Fatal("valid transaction notification failed")
	}
	if notification.ID != "NOTICE_TEST_001" || notification.EventType != "TRANSACTION.SUCCESS" ||
		notification.ResourceType != "encrypt-resource" || notification.Transaction.AppID != testAppID ||
		notification.Transaction.MerchantID != testMerchantID || notification.Transaction.OutTradeNo != "ORDER_TEST_001" ||
		notification.Transaction.TransactionID != "TX_TEST_001" || notification.Transaction.TradeState != "SUCCESS" ||
		notification.Transaction.Amount.Total != 1 || notification.Transaction.Amount.Currency != "CNY" {
		t.Fatal("transaction notification DTO mismatch")
	}

	badResources := []struct {
		name           string
		resource       string
		associatedData string
		extraOuter     string
		want           ErrorKind
	}{
		{name: "unknown envelope field", resource: resource, associatedData: associatedData, extraOuter: `,"unexpected":true`, want: ErrorProtocol},
		{name: "duplicate envelope field", resource: resource, associatedData: associatedData, extraOuter: `,"id":"NOTICE_DUPLICATE"`, want: ErrorProtocol},
		{name: "unknown transaction field", resource: strings.TrimSuffix(resource, "}") + `,"unexpected":true}`, associatedData: associatedData, want: ErrorProtocol},
		{name: "duplicate transaction field", resource: strings.TrimSuffix(resource, "}") + `,"appid":"wx-duplicate"}`, associatedData: associatedData, want: ErrorProtocol},
		{name: "missing required transaction id", resource: strings.Replace(resource, `"transaction_id":"TX_TEST_001",`, "", 1), associatedData: associatedData, want: ErrorProtocol},
		{name: "missing success time", resource: strings.Replace(resource, `"success_time":"2027-01-15T08:00:00Z",`, "", 1), associatedData: associatedData, want: ErrorProtocol},
		{name: "wrong associated data", resource: resource, associatedData: "wrong-aad", want: ErrorDecryption},
		{name: "missing payer total", resource: strings.Replace(resource, `"payer_total":1,`, "", 1), associatedData: associatedData, want: ErrorProtocol},
		{name: "missing payer currency", resource: strings.Replace(resource, `,"payer_currency":"CNY"`, "", 1), associatedData: associatedData, want: ErrorProtocol},
	}
	for _, test := range badResources {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixtureCiphertext := ciphertext
			if test.resource != resource {
				fixtureCiphertext = encryptTestResource(t, apiV3Key, resourceNonce, associatedData, []byte(test.resource))
			}
			body := notificationBody(fixtureCiphertext, test.associatedData, test.extraOuter)
			headers := signedTestHeaders(t, providerKey, body, "1800000000", "callback-nonce")
			_, err := client.ParseTransactionNotification(body, headers)
			var providerError *Error
			if !errors.As(err, &providerError) || providerError.Kind() != test.want {
				t.Fatal("invalid notification error classification mismatch")
			}
			if strings.Contains(err.Error(), "ORDER_TEST_001") || strings.Contains(err.Error(), "synthetic-openid") {
				t.Fatal("notification material leaked through error")
			}
		})
	}
}

func TestTypedOperationsUseOfficialEndpointsAndVerifiedResponses(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0).UTC()
	transactionBody := []byte(`{"appid":"wx-test-app","mchid":"1900000109","out_trade_no":"ORDER_TEST_001","transaction_id":"TX_TEST_001","trade_type":"JSAPI","trade_state":"SUCCESS","trade_state_desc":"synthetic success","success_time":"2027-01-15T08:00:00Z","payer":{"openid":"synthetic-openid"},"amount":{"total":1,"payer_total":1,"currency":"CNY","payer_currency":"CNY"}}`)
	refundBody := []byte(`{"refund_id":"REFUND_PROVIDER_001","out_refund_no":"REFUND_TEST_001","transaction_id":"TX_TEST_001","out_trade_no":"ORDER_TEST_001","channel":"ORIGINAL","user_received_account":"synthetic account","create_time":"2027-01-15T08:01:00Z","status":"PROCESSING","funds_account":"AVAILABLE","amount":{"total":1,"refund":1,"payer_total":1,"payer_refund":1,"settlement_refund":1,"settlement_total":1,"discount_refund":0,"currency":"CNY"}}`)
	refundSuccessBody := []byte(`{"refund_id":"REFUND_PROVIDER_001","out_refund_no":"REFUND_TEST_001","transaction_id":"TX_TEST_001","out_trade_no":"ORDER_TEST_001","channel":"ORIGINAL","user_received_account":"synthetic account","create_time":"2027-01-15T08:01:00Z","success_time":"2027-01-15T08:02:00Z","status":"SUCCESS","funds_account":"AVAILABLE","amount":{"total":1,"refund":1,"payer_total":1,"payer_refund":1,"settlement_refund":1,"settlement_total":1,"discount_refund":0,"currency":"CNY"}}`)

	type expectedCall struct {
		method        string
		requestTarget string
		body          string
		response      []byte
		status        int
	}
	expected := []expectedCall{
		{method: http.MethodGet, requestTarget: "/v3/pay/transactions/out-trade-no/ORDER_TEST_001?mchid=1900000109", response: transactionBody, status: http.StatusOK},
		{method: http.MethodGet, requestTarget: "/v3/pay/transactions/id/TX_TEST_001?mchid=1900000109", response: transactionBody, status: http.StatusOK},
		{method: http.MethodPost, requestTarget: "/v3/pay/transactions/out-trade-no/ORDER_TEST_001/close", body: `{"mchid":"1900000109"}`, status: http.StatusNoContent},
		{method: http.MethodPost, requestTarget: "/v3/refund/domestic/refunds", body: `{"out_trade_no":"ORDER_TEST_001","out_refund_no":"REFUND_TEST_001","reason":"synthetic reason","notify_url":"https://merchant.invalid/api/v1/refunds/wechat/notify","amount":{"refund":1,"total":1,"currency":"CNY"}}`, response: refundBody, status: http.StatusOK},
		{method: http.MethodGet, requestTarget: "/v3/refund/domestic/refunds/REFUND_TEST_001", response: refundSuccessBody, status: http.StatusOK},
	}
	var callIndex atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		index := int(callIndex.Add(1) - 1)
		if index >= len(expected) {
			t.Error("unexpected provider call")
			http.Error(writer, "", http.StatusInternalServerError)
			return
		}
		want := expected[index]
		body, err := io.ReadAll(request.Body)
		if err != nil || request.Method != want.method || request.URL.RequestURI() != want.requestTarget || string(body) != want.body {
			t.Error("typed operation wire contract mismatch")
			return
		}
		parts := regexp.MustCompile(`^WECHATPAY2-SHA256-RSA2048 mchid="([^"]+)",nonce_str="([^"]+)",signature="([^"]+)",timestamp="([^"]+)",serial_no="([^"]+)"$`).FindStringSubmatch(request.Header.Get("Authorization"))
		canonical := want.method + "\n" + want.requestTarget + "\n1800000000\noperation-nonce\n" + want.body + "\n"
		if len(parts) != 6 || parts[1] != testMerchantID || parts[2] != "operation-nonce" || parts[4] != "1800000000" || parts[5] != testMerchantSerial || !verifiesTestSignature(&merchantKey.PublicKey, canonical, parts[3]) {
			t.Error("typed operation Authorization mismatch")
			return
		}
		providerNonce := fmt.Sprintf("provider-operation-%d", index)
		writer.Header().Set("Wechatpay-Timestamp", "1800000000")
		writer.Header().Set("Wechatpay-Nonce", providerNonce)
		writer.Header().Set("Wechatpay-Serial", testProviderSerial)
		writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, "1800000000\n"+providerNonce+"\n"+string(want.response)+"\n"))
		writer.WriteHeader(want.status)
		_, _ = writer.Write(want.response)
	}))
	defer server.Close()

	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	}, server.Client(), server.URL, func() time.Time { return now }, func() (string, error) { return "operation-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}

	byOutTradeNo, err := client.QueryTransactionByOutTradeNo(context.Background(), "ORDER_TEST_001")
	if err != nil || byOutTradeNo.TransactionID != "TX_TEST_001" || byOutTradeNo.TradeState != "SUCCESS" {
		t.Fatal("out-trade-no query result mismatch")
	}
	byTransactionID, err := client.QueryTransactionByID(context.Background(), "TX_TEST_001")
	if err != nil || byTransactionID.OutTradeNo != "ORDER_TEST_001" || byTransactionID.Amount.Total != 1 {
		t.Fatal("transaction-id query result mismatch")
	}
	if err := client.CloseTransaction(context.Background(), "ORDER_TEST_001"); err != nil {
		t.Fatal("close transaction failed")
	}
	createdRefund, err := client.CreateRefund(context.Background(), RefundCreateRequest{
		OutTradeNo: "ORDER_TEST_001", OutRefundNo: "REFUND_TEST_001", Reason: "synthetic reason",
		NotifyURL: "https://merchant.invalid/api/v1/refunds/wechat/notify",
		Amount:    RefundRequestAmount{Refund: 1, Total: 1, Currency: "CNY"},
	})
	if err != nil || createdRefund.RefundID != "REFUND_PROVIDER_001" || createdRefund.Status != "PROCESSING" {
		t.Fatal("create refund result mismatch")
	}
	queriedRefund, err := client.QueryRefund(context.Background(), "REFUND_TEST_001")
	if err != nil || queriedRefund.OutTradeNo != "ORDER_TEST_001" || queriedRefund.Amount.Refund != 1 ||
		queriedRefund.Status != "SUCCESS" || queriedRefund.SuccessTime.Unix() != 1_800_000_120 {
		t.Fatal("query refund result mismatch")
	}
	if callIndex.Load() != int32(len(expected)) {
		t.Fatalf("provider calls = %d, want %d", callIndex.Load(), len(expected))
	}
}

func TestClientClassifiesProviderFailures(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0).UTC()
	tests := []struct {
		name          string
		status        int
		body          string
		timeout       bool
		wantKind      ErrorKind
		wantCode      string
		wantRetryable bool
	}{
		{name: "timeout", timeout: true, wantKind: ErrorTimeout, wantRetryable: true},
		{name: "rate limited", status: http.StatusTooManyRequests, body: `{"code":"FREQUENCY_LIMITED","message":"provider-detail-canary"}`, wantKind: ErrorRateLimited, wantCode: "FREQUENCY_LIMITED", wantRetryable: true},
		{name: "server unavailable", status: http.StatusServiceUnavailable, body: `{"code":"SYSTEM_ERROR","message":"provider-detail-canary"}`, wantKind: ErrorProviderUnavailable, wantCode: "SYSTEM_ERROR", wantRetryable: true},
		{name: "provider rejected", status: http.StatusBadRequest, body: `{"code":"PARAM_ERROR","message":"provider-detail-canary"}`, wantKind: ErrorProviderRejected, wantCode: "PARAM_ERROR"},
		{name: "order not found remains unknown", status: http.StatusNotFound, body: `{"code":"ORDER_NOT_EXIST","message":"provider-detail-canary"}`, wantKind: ErrorProviderRejected, wantCode: "ORDER_NOT_EXIST", wantRetryable: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if test.timeout {
					<-request.Context().Done()
					return
				}
				providerNonce := "provider-failure-nonce"
				writer.Header().Set("Wechatpay-Timestamp", "1800000000")
				writer.Header().Set("Wechatpay-Nonce", providerNonce)
				writer.Header().Set("Wechatpay-Serial", testProviderSerial)
				writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, "1800000000\n"+providerNonce+"\n"+test.body+"\n"))
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			httpClient := server.Client()
			if test.timeout {
				httpClient.Timeout = 20 * time.Millisecond
			}
			client, err := newClient(Config{
				AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
				MerchantPrivateKey:  merchantKey,
				WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
				APIv3Key:            []byte(testAPIv3Key),
			}, httpClient, server.URL, func() time.Time { return now }, func() (string, error) { return "failure-nonce", nil })
			if err != nil {
				t.Fatal("controlled client construction failed")
			}
			_, err = client.QueryTransactionByOutTradeNo(context.Background(), "ORDER_TEST_001")
			var providerError *Error
			if !errors.As(err, &providerError) || providerError.Kind() != test.wantKind ||
				providerError.StatusCode() != test.status || providerError.ProviderCode() != test.wantCode ||
				providerError.Retryable() != test.wantRetryable {
				t.Fatal("provider failure classification mismatch")
			}
			if strings.Contains(err.Error(), "provider-detail-canary") || (test.body != "" && strings.Contains(err.Error(), test.body)) {
				t.Fatal("provider failure payload leaked through error")
			}
		})
	}
}

func TestClientRejectsUnsignedProviderFailures(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte(`{"code":"FREQUENCY_LIMITED","message":"provider-detail-canary"}`))
	}))
	defer server.Close()
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	}, server.Client(), server.URL, func() time.Time { return time.Unix(1_800_000_000, 0) }, func() (string, error) { return "failure-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}
	_, err = client.QueryTransactionByOutTradeNo(context.Background(), "ORDER_TEST_001")
	var providerError *Error
	if !errors.As(err, &providerError) || providerError.Kind() != ErrorUnknownSerial || providerError.ProviderCode() != "" {
		t.Fatal("unsigned provider failure was trusted")
	}
}

func TestClientRefusesRedirectWithoutReplayingSignedRequest(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	var calls atomic.Int32
	var redirected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.URL.Path == "/redirected" {
			redirected.Add(1)
			return
		}
		http.Redirect(writer, request, "/redirected", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	}, server.Client(), server.URL, time.Now, func() (string, error) { return "redirect-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}
	if _, err := client.QueryTransactionByOutTradeNo(context.Background(), "ORDER_TEST_001"); err == nil {
		t.Fatal("redirect was accepted")
	}
	if calls.Load() != 1 || redirected.Load() != 0 {
		t.Fatal("signed request followed or replayed a redirect")
	}
}

func TestCloseTransactionRequiresNoContent(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	now := time.Unix(1_800_000_000, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Wechatpay-Timestamp", "1800000000")
		writer.Header().Set("Wechatpay-Nonce", "close-response-nonce")
		writer.Header().Set("Wechatpay-Serial", testProviderSerial)
		writer.Header().Set("Wechatpay-Signature", signTestMessage(t, providerKey, "1800000000\nclose-response-nonce\n\n"))
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	}, server.Client(), server.URL, func() time.Time { return now }, func() (string, error) { return "close-request-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}
	err = client.CloseTransaction(context.Background(), "ORDER_TEST_001")
	var providerError *Error
	if !errors.As(err, &providerError) || providerError.Kind() != ErrorProtocol || providerError.StatusCode() != http.StatusOK {
		t.Fatal("close accepted a non-204 response")
	}
}

func TestCreateRefundRequiresOneTransactionIdentifier(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(writer, "", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey:  merchantKey,
		WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key:            []byte(testAPIv3Key),
	}, server.Client(), server.URL, time.Now, func() (string, error) { return "validation-nonce", nil })
	if err != nil {
		t.Fatal("controlled client construction failed")
	}
	inputs := []RefundCreateRequest{
		{OutRefundNo: "REFUND_TEST_001", Amount: RefundRequestAmount{Refund: 1, Total: 1, Currency: "CNY"}},
		{TransactionID: "TX_TEST_001", OutTradeNo: "ORDER_TEST_001", OutRefundNo: "REFUND_TEST_001", Amount: RefundRequestAmount{Refund: 1, Total: 1, Currency: "CNY"}},
		{TransactionID: " ", OutTradeNo: "ORDER_TEST_001", OutRefundNo: "REFUND_TEST_001", Amount: RefundRequestAmount{Refund: 1, Total: 1, Currency: "CNY"}},
	}
	for _, input := range inputs {
		_, err := client.CreateRefund(context.Background(), input)
		var providerError *Error
		if !errors.As(err, &providerError) || providerError.Kind() != ErrorProtocol || providerError.Retryable() {
			t.Fatal("refund identifier validation mismatch")
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", calls.Load())
	}
}

func encryptTestResource(t *testing.T, key []byte, nonce, associatedData string, plaintext []byte) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal("cannot create controlled test cipher")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal("cannot create controlled test GCM")
	}
	sealed := aead.Seal(nil, []byte(nonce), plaintext, []byte(associatedData))
	return base64.StdEncoding.EncodeToString(sealed)
}

func notificationBody(ciphertext, associatedData, extraOuter string) []byte {
	return []byte(fmt.Sprintf(`{"id":"NOTICE_TEST_001","create_time":"2027-01-15T08:00:00Z","resource_type":"encrypt-resource","event_type":"TRANSACTION.SUCCESS","summary":"synthetic summary","resource":{"original_type":"transaction","algorithm":"AEAD_AES_256_GCM","ciphertext":"%s","associated_data":"%s","nonce":"nonce-123456"}%s}`, ciphertext, associatedData, extraOuter))
}

func signedTestHeaders(t *testing.T, providerKey *rsa.PrivateKey, body []byte, timestamp, nonce string) SignatureHeaders {
	t.Helper()
	return SignatureHeaders{
		Serial:    testProviderSerial,
		Timestamp: timestamp,
		Nonce:     nonce,
		Signature: signTestMessage(t, providerKey, timestamp+"\n"+nonce+"\n"+string(body)+"\n"),
	}
}

func generatedTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PrivateKey) {
	t.Helper()
	testKeysOnce.Do(func() {
		var err error
		testMerchantKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return
		}
		testProviderKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	})
	if testMerchantKey == nil || testProviderKey == nil {
		t.Fatal("cannot generate local RSA test keys")
	}
	return testMerchantKey, testProviderKey
}

func signTestMessage(t *testing.T, key *rsa.PrivateKey, message string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal("cannot sign controlled fixture")
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func verifiesTestSignature(key *rsa.PublicKey, message, encodedSignature string) bool {
	signature, err := base64.StdEncoding.DecodeString(encodedSignature)
	if err != nil {
		return false
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) == nil
}
