package wechatpay

import (
	"context"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestDownloadTransactionBillUsesSignedOfficialFlowAndStableStrictEntries(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	billBody := []byte(transactionBillFixture(testAppID, testMerchantID))
	billSHA1 := sha1.Sum(billBody)
	applyBody := []byte(fmt.Sprintf(`{"hash_type":"SHA1","hash_value":"%s","download_url":"https://api.mch.weixin.qq.com/v3/billdownload/file?token=controlled-token"}`, hex.EncodeToString(billSHA1[:])))
	var calls atomic.Int32
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		if request.URL.Scheme != "https" || request.URL.Host != "api.mch.weixin.qq.com" || request.Method != http.MethodGet {
			t.Error("bill operation escaped the fixed official HTTPS origin")
		}
		if call == 1 {
			if request.URL.RequestURI() != "/v3/bill/tradebill?bill_date=2027-01-15&bill_type=ALL" {
				t.Error("bill apply request target mismatch")
			}
			assertSignedBillRequest(t, request, merchantKey, "bill-nonce-1")
			return signedHTTPResponse(t, providerKey, request, http.StatusOK, applyBody, "provider-bill-apply"), nil
		}
		if call == 2 {
			if request.URL.RequestURI() != "/v3/billdownload/file?token=controlled-token" {
				t.Error("bill download request target mismatch")
			}
			assertSignedBillRequest(t, request, merchantKey, "bill-nonce-2")
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(billBody))), Request: request}, nil
		}
		t.Error("unexpected extra provider call")
		return nil, fmt.Errorf("unexpected call")
	})
	client := newBillTestClient(t, merchantKey, providerKey, transport)

	got, err := client.DownloadTransactionBill(context.Background(), time.Date(2027, 1, 15, 19, 0, 0, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatal("official bill flow failed")
	}
	if calls.Load() != 2 || !got.Date.Equal(time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)) || got.Digest != sha256.Sum256(billBody) {
		t.Fatal("bill date, digest, or provider call count mismatch")
	}
	if len(got.Entries) != 3 {
		t.Fatalf("bill entries = %d, want 3 provider facts", len(got.Entries))
	}
	payment, refund, processing := got.Entries[0], got.Entries[1], got.Entries[2]
	if payment.Kind != BillEntryPayment || payment.OutTradeNo != "ORDER_TEST_001" || payment.ProviderID != "TX_TEST_001" ||
		payment.State != "SUCCESS" || payment.AmountCents != 700 || payment.Currency != "CNY" ||
		!payment.OccurredAt.Equal(time.Date(2027, 1, 15, 0, 1, 2, 0, time.UTC)) {
		t.Fatal("payment bill entry mismatch")
	}
	if refund.Kind != BillEntryRefund || refund.OutTradeNo != "ORDER_TEST_001" || refund.OutRefundNo != "REFUND_TEST_001" ||
		refund.ProviderID != "REFUND_PROVIDER_001" || refund.State != "SUCCESS" || refund.AmountCents != 300 {
		t.Fatal("successful refund bill entry mismatch")
	}
	if processing.Kind != BillEntryRefund || processing.OutRefundNo != "REFUND_TEST_002" || processing.State != "PROCESSING" || processing.AmountCents != 100 {
		t.Fatal("processing refund provider fact was lost")
	}
}

func TestDownloadTransactionBillRejectsUntrustedOrCorruptDownloads(t *testing.T) {
	t.Parallel()
	merchantKey, providerKey := generatedTestKeys(t)
	validBill := []byte(transactionBillFixture(testAppID, testMerchantID))
	validHash := sha1.Sum(validBill)
	tests := []struct {
		name        string
		downloadURL string
		hashType    string
		hashValue   string
		bill        []byte
		status      int
		wantCalls   int32
	}{
		{name: "non HTTPS", downloadURL: "http://api.mch.weixin.qq.com/v3/billdownload/file?token=x", hashType: "SHA1", hashValue: hex.EncodeToString(validHash[:]), bill: validBill, status: http.StatusOK, wantCalls: 1},
		{name: "foreign host", downloadURL: "https://attacker.invalid/v3/billdownload/file?token=x", hashType: "SHA1", hashValue: hex.EncodeToString(validHash[:]), bill: validBill, status: http.StatusOK, wantCalls: 1},
		{name: "unexpected hash type", downloadURL: "https://api.mch.weixin.qq.com/v3/billdownload/file?token=x", hashType: "SHA256", hashValue: hex.EncodeToString(validHash[:]), bill: validBill, status: http.StatusOK, wantCalls: 1},
		{name: "hash mismatch", downloadURL: "https://api.mch.weixin.qq.com/v3/billdownload/file?token=x", hashType: "SHA1", hashValue: strings.Repeat("0", 40), bill: validBill, status: http.StatusOK, wantCalls: 2},
		{name: "redirect refused", downloadURL: "https://api.mch.weixin.qq.com/v3/billdownload/file?token=x", hashType: "SHA1", hashValue: hex.EncodeToString(validHash[:]), bill: validBill, status: http.StatusFound, wantCalls: 2},
		{name: "oversized body", downloadURL: "https://api.mch.weixin.qq.com/v3/billdownload/file?token=x", hashType: "SHA1", hashValue: hex.EncodeToString(validHash[:]), bill: []byte(strings.Repeat("x", billResponseMaxBytes+1)), status: http.StatusOK, wantCalls: 2},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			applyBody := []byte(fmt.Sprintf(`{"hash_type":"%s","hash_value":"%s","download_url":"%s"}`, test.hashType, test.hashValue, test.downloadURL))
			var calls atomic.Int32
			transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				call := calls.Add(1)
				if call == 1 {
					return signedHTTPResponse(t, providerKey, request, http.StatusOK, applyBody, "provider-apply-error-case"), nil
				}
				response := &http.Response{StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(test.bill))), Request: request}
				if test.status == http.StatusFound {
					response.Header.Set("Location", "https://attacker.invalid/redirect")
				}
				return response, nil
			})
			client := newBillTestClient(t, merchantKey, providerKey, transport)
			if _, err := client.DownloadTransactionBill(context.Background(), time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)); err == nil {
				t.Fatal("unsafe or corrupt bill was accepted")
			}
			if calls.Load() != test.wantCalls {
				t.Fatalf("provider calls = %d, want %d", calls.Load(), test.wantCalls)
			}
		})
	}
}

func TestParseTransactionBillRejectsWrongMerchantAndMalformedMoney(t *testing.T) {
	t.Parallel()
	tests := []string{
		transactionBillFixture(testAppID, "wrong-merchant"),
		strings.Replace(transactionBillFixture(testAppID, testMerchantID), "`0.60%,`7.00,`0.00,`", "`0.60%,`7.001,`0.00,`", 1),
		strings.Replace(transactionBillFixture(testAppID, testMerchantID), "交易时间,公众账号ID", "交易时间,unknown", 1),
	}
	for _, body := range tests {
		if _, err := parseTransactionBill([]byte(body), testAppID, testMerchantID); err == nil {
			t.Fatal("malformed or cross-merchant bill was accepted")
		}
	}
}

func newBillTestClient(t *testing.T, merchantKey, providerKey *rsa.PrivateKey, transport http.RoundTripper) *Client {
	t.Helper()
	var nonceCounter atomic.Int32
	client, err := newClient(Config{
		AppID: testAppID, MerchantID: testMerchantID, MerchantCertificateSerial: testMerchantSerial,
		MerchantPrivateKey: merchantKey, WeChatPayPublicKeys: map[string]*rsa.PublicKey{testProviderSerial: &providerKey.PublicKey},
		APIv3Key: []byte(testAPIv3Key),
	}, &http.Client{Transport: transport}, apiOrigin, func() time.Time { return time.Unix(1_800_000_000, 0).UTC() }, func() (string, error) {
		return fmt.Sprintf("bill-nonce-%d", nonceCounter.Add(1)), nil
	})
	if err != nil {
		t.Fatal("controlled bill client construction failed")
	}
	return client
}

func signedHTTPResponse(t *testing.T, providerKey *rsa.PrivateKey, request *http.Request, status int, body []byte, nonce string) *http.Response {
	t.Helper()
	headers := make(http.Header)
	headers.Set("Wechatpay-Timestamp", "1800000000")
	headers.Set("Wechatpay-Nonce", nonce)
	headers.Set("Wechatpay-Serial", testProviderSerial)
	headers.Set("Wechatpay-Signature", signTestMessage(t, providerKey, "1800000000\n"+nonce+"\n"+string(body)+"\n"))
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(string(body))), Request: request}
}

func assertSignedBillRequest(t *testing.T, request *http.Request, merchantKey *rsa.PrivateKey, nonce string) {
	t.Helper()
	parts := regexp.MustCompile(`^WECHATPAY2-SHA256-RSA2048 mchid="([^"]+)",nonce_str="([^"]+)",signature="([^"]+)",timestamp="([^"]+)",serial_no="([^"]+)"$`).FindStringSubmatch(request.Header.Get("Authorization"))
	canonical := request.Method + "\n" + request.URL.RequestURI() + "\n1800000000\n" + nonce + "\n\n"
	if len(parts) != 6 || parts[1] != testMerchantID || parts[2] != nonce || parts[4] != "1800000000" || parts[5] != testMerchantSerial || !verifiesTestSignature(&merchantKey.PublicKey, canonical, parts[3]) {
		t.Fatal("bill request Authorization mismatch")
	}
}

func transactionBillFixture(appID, merchantID string) string {
	header := "交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注"
	payment := []string{"2027-01-15 08:01:02", appID, merchantID, "", "DEVICE", "TX_TEST_001", "ORDER_TEST_001", "OPENID", "JSAPI", "SUCCESS", "TEST", "CNY", "7.00", "0.00", "", "", "0.00", "0.00", "", "", "synthetic", "attach", "0.01", "0.60%", "7.00", "0.00", ""}
	refund := []string{"2027-01-15 08:03:04", appID, merchantID, "", "DEVICE", "TX_TEST_001", "ORDER_TEST_001", "OPENID", "JSAPI", "REFUND", "TEST", "CNY", "-3.00", "0.00", "REFUND_PROVIDER_001", "REFUND_TEST_001", "3.00", "0.00", "ORIGINAL", "SUCCESS", "synthetic", "attach", "-0.01", "0.60%", "0.00", "3.00", ""}
	processing := []string{"2027-01-15 08:05:06", appID, merchantID, "", "DEVICE", "TX_TEST_002", "ORDER_TEST_002", "OPENID", "JSAPI", "REFUND", "TEST", "CNY", "-1.00", "0.00", "REFUND_PROVIDER_002", "REFUND_TEST_002", "1.00", "0.00", "ORIGINAL", "PROCESSING", "synthetic", "attach", "-0.01", "0.60%", "0.00", "1.00", ""}
	prefix := func(fields []string) string {
		for index := range fields {
			fields[index] = "`" + fields[index]
		}
		return strings.Join(fields, ",")
	}
	return "\ufeff" + header + "\r\n" + prefix(payment) + "\r\n" + prefix(refund) + "\r\n" + prefix(processing) + "\r\n总交易单数,应结订单总金额,退款总金额,充值券退款总金额,手续费总金额,订单总金额,申请退款总金额\r\n`3,`3.00,`3.00,`0.00,`-0.01,`7.00,`4.00\r\n"
}
