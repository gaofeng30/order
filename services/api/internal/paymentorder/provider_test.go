package paymentorder

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

func TestFakeProviderCreateIsIntrinsicAndNeverCreatesTwice(t *testing.T) {
	provider := NewFakeProvider()
	request := fakeCreateRequest()
	const workers = 32
	var wait sync.WaitGroup
	wait.Add(workers)
	results := make(chan ProviderCreateResult, workers)
	errorsSeen := make(chan error, workers)
	for range workers {
		go func() {
			defer wait.Done()
			result, err := provider.CreateJSAPI(context.Background(), request)
			results <- result
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("CreateJSAPI() = %v", err)
		}
	}
	for result := range results {
		if result.PrepayID == "" || result.RequestPayment.Package != "prepay_id="+result.PrepayID {
			t.Fatalf("CreateJSAPI() = %#v", result)
		}
	}
	if got := provider.CreateCount(request.OutTradeNo); got != 1 {
		t.Fatalf("CreateCount() = %d, want 1", got)
	}

	conflict := request
	conflict.AmountCents++
	if _, err := provider.CreateJSAPI(context.Background(), conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("CreateJSAPI(conflict) = %v", err)
	}
}

func TestWeChatPayAdapterDelegatesTypedOperationsAndVerifiedNotification(t *testing.T) {
	client := &wechatPayClientStub{
		createResult: wechatpay.JSAPIPrepay{PrepayID: "prepay-1", RequestPayment: wechatpay.RequestPayment{TimeStamp: "1", NonceStr: "n", Package: "prepay_id=prepay-1", SignType: "RSA", PaySign: "s"}},
		queryResult:  wechatpay.Transaction{OutTradeNo: "order-1", TradeState: "NOTPAY"},
		notification: wechatpay.TransactionNotification{ID: "event-1", Transaction: wechatpay.Transaction{OutTradeNo: "order-1", TradeState: "SUCCESS"}},
	}
	adapter, err := NewWeChatPayAdapter(client, "wx-app", "mch-1")
	if err != nil {
		t.Fatal(err)
	}
	request := fakeCreateRequest()
	created, err := adapter.CreateJSAPI(context.Background(), request)
	if err != nil || created.PrepayID != "prepay-1" || client.createInput.OutTradeNo != request.OutTradeNo || client.createInput.Amount.Total != request.AmountCents {
		t.Fatalf("CreateJSAPI() = %#v/%v input=%#v", created, err, client.createInput)
	}
	queried, err := adapter.QueryTransaction(context.Background(), request.OutTradeNo)
	if err != nil || queried.OutTradeNo != "order-1" || client.queryOutTradeNo != request.OutTradeNo {
		t.Fatalf("QueryTransaction() = %#v/%v", queried, err)
	}
	verified, err := adapter.ParsePaymentNotification([]byte("opaque encrypted body"), wechatpay.SignatureHeaders{Serial: "serial"})
	if err != nil || verified.ProviderEventID != "event-1" || verified.Source != paymentobservation.SourceCallback {
		t.Fatalf("ParsePaymentNotification() = %#v/%v", verified, err)
	}
	if err := adapter.CloseTransaction(context.Background(), request.OutTradeNo); err != nil || client.closedOutTradeNo != request.OutTradeNo {
		t.Fatalf("CloseTransaction() = %v", err)
	}
}

func TestFakeProviderQueryTransitionsOnlyWhenTestMarksPaid(t *testing.T) {
	provider := NewFakeProvider()
	request := fakeCreateRequest()
	if _, err := provider.CreateJSAPI(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	transaction, err := provider.QueryTransaction(context.Background(), request.OutTradeNo)
	if err != nil || transaction.TradeState != "NOTPAY" {
		t.Fatalf("QueryTransaction(not paid) = %#v/%v", transaction, err)
	}
	successAt := time.Date(2026, 8, 25, 2, 5, 0, 0, time.UTC)
	if err := provider.MarkPaid(request.OutTradeNo, "wx-transaction-1", successAt); err != nil {
		t.Fatal(err)
	}
	transaction, err = provider.QueryTransaction(context.Background(), request.OutTradeNo)
	if err != nil || transaction.TradeState != "SUCCESS" || !transaction.SuccessTime.Equal(successAt) {
		t.Fatalf("QueryTransaction(paid) = %#v/%v", transaction, err)
	}
}

func TestFakeProviderNotificationIsVerifiedBeforeTypedIngress(t *testing.T) {
	provider := NewFakeProvider()
	request := fakeCreateRequest()
	if _, err := provider.CreateJSAPI(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	successAt := time.Date(2026, 8, 25, 2, 5, 0, 0, time.UTC)
	if err := provider.MarkPaid(request.OutTradeNo, "wx-transaction-1", successAt); err != nil {
		t.Fatal(err)
	}
	body, headers, err := provider.PaymentNotification(request.OutTradeNo, "event-1")
	if err != nil {
		t.Fatal(err)
	}
	verified, err := provider.ParsePaymentNotification(body, headers)
	if err != nil || verified.Source != paymentobservation.SourceCallback || verified.ProviderEventID != "event-1" {
		t.Fatalf("ParsePaymentNotification() = %#v/%v", verified, err)
	}
	body[0] ^= 1
	if _, err := provider.ParsePaymentNotification(body, headers); err == nil {
		t.Fatal("tampered notification was accepted")
	}
}

func fakeCreateRequest() ProviderCreateRequest {
	return ProviderCreateRequest{
		AppID: "wx-app", MerchantID: "mch-1", Description: "预约点餐",
		OutTradeNo: "order-quote-91", TimeExpire: "2026-08-25T02:10:00Z",
		NotifyURL:   "https://example.invalid/api/v1/payments/wechat/notify",
		AmountCents: 980, Currency: "CNY", PayerOpenID: "openid-42",
		QuoteID: "91", QuoteDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

type wechatPayClientStub struct {
	createInput      wechatpay.JSAPICreateRequest
	createResult     wechatpay.JSAPIPrepay
	queryOutTradeNo  string
	queryResult      wechatpay.Transaction
	closedOutTradeNo string
	notification     wechatpay.TransactionNotification
}

func (stub *wechatPayClientStub) CreateJSAPIPrepay(_ context.Context, input wechatpay.JSAPICreateRequest) (wechatpay.JSAPIPrepay, error) {
	stub.createInput = input
	return stub.createResult, nil
}
func (stub *wechatPayClientStub) QueryTransactionByOutTradeNo(_ context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	stub.queryOutTradeNo = outTradeNo
	return stub.queryResult, nil
}
func (stub *wechatPayClientStub) CloseTransaction(_ context.Context, outTradeNo string) error {
	stub.closedOutTradeNo = outTradeNo
	return nil
}
func (stub *wechatPayClientStub) ParseTransactionNotification([]byte, wechatpay.SignatureHeaders) (wechatpay.TransactionNotification, error) {
	return stub.notification, nil
}
