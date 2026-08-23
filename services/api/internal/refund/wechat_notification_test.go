package refund

import (
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type refundNotificationClientStub struct {
	gotBody    []byte
	gotHeaders wechatpay.SignatureHeaders
	result     wechatpay.RefundNotification
	err        error
}

func (stub *refundNotificationClientStub) ParseRefundNotification(body []byte, headers wechatpay.SignatureHeaders) (wechatpay.RefundNotification, error) {
	stub.gotBody = append([]byte(nil), body...)
	stub.gotHeaders = headers
	return stub.result, stub.err
}

func TestWeChatNotificationParserMapsTrustedFactsWithoutReplacingThem(t *testing.T) {
	t.Parallel()
	successTime := time.Date(2027, 1, 15, 0, 2, 0, 0, time.UTC)
	stub := &refundNotificationClientStub{result: wechatpay.RefundNotification{
		ID: "REFUND_NOTICE_TEST_001", MerchantID: "wrong-but-verified-mchid",
		Refund: wechatpay.Refund{
			OutTradeNo: "ORDER_TEST_001", TransactionID: "TX_TEST_001", OutRefundNo: "wrong-but-verified-refund",
			RefundID: "REFUND_PROVIDER_001", Status: "SUCCESS", SuccessTime: successTime,
			Amount: wechatpay.RefundAmount{Total: 7, Refund: 3, PayerTotal: 7, PayerRefund: 3, Currency: "CNY"},
		},
	}}
	parser, err := NewWeChatNotificationParser(stub)
	if err != nil {
		t.Fatal("parser construction failed")
	}
	headers := wechatpay.SignatureHeaders{Serial: "SERIAL", Signature: "signature", Timestamp: "timestamp", Nonce: "nonce"}
	got, err := parser.ParseRefundNotification([]byte("verified body"), headers)
	if err != nil {
		t.Fatal("trusted refund mapping failed")
	}
	if string(stub.gotBody) != "verified body" || stub.gotHeaders != headers || got.Source != SourceCallback ||
		got.ProviderEventID != "REFUND_NOTICE_TEST_001" || got.Refund.MerchantID != "wrong-but-verified-mchid" ||
		got.Refund.OutRefundNo != "wrong-but-verified-refund" || got.Refund.AmountCents != 3 || got.Refund.TotalCents != 7 ||
		got.Refund.Currency != "CNY" || got.Refund.State != ProviderSuccess || !got.Refund.SuccessTime.Equal(successTime) {
		t.Fatal("trusted provider facts were changed or incompletely mapped")
	}
}

func TestWeChatNotificationParserRejectsUnusableProviderDTOs(t *testing.T) {
	t.Parallel()
	if _, err := NewWeChatNotificationParser(nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("nil notification client was accepted")
	}
	stub := &refundNotificationClientStub{result: wechatpay.RefundNotification{
		ID: "EVENT", MerchantID: "MCH", Refund: wechatpay.Refund{
			OutTradeNo: "ORDER", TransactionID: "TX", OutRefundNo: "REFUND", RefundID: "WXREFUND",
			Status: "PROCESSING", Amount: wechatpay.RefundAmount{Total: 1, Refund: 1, Currency: "CNY"},
		},
	}}
	parser, err := NewWeChatNotificationParser(stub)
	if err != nil {
		t.Fatal("parser construction failed")
	}
	if _, err := parser.ParseRefundNotification(nil, wechatpay.SignatureHeaders{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("non-terminal callback was accepted")
	}
	stub.err = errors.New("signature failed")
	if _, err := parser.ParseRefundNotification(nil, wechatpay.SignatureHeaders{}); !errors.Is(err, stub.err) {
		t.Fatal("client trust-boundary error was hidden")
	}
}

var _ NotificationParser = (*WeChatNotificationParser)(nil)
