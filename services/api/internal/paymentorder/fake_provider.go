package paymentorder

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

var errFakeNotificationInvalid = errors.New("paymentorder fake notification invalid")

type fakeTransaction struct {
	request     ProviderCreateRequest
	requestHash [32]byte
	result      ProviderCreateResult
	transaction wechatpay.Transaction
	createCount uint64
	queryCount  uint64
}

// FakeProvider is a deterministic, concurrency-safe local WeChat Pay boundary.
// It never performs network I/O and only becomes paid through MarkPaid.
type FakeProvider struct {
	mu           sync.Mutex
	transactions map[string]*fakeTransaction
	secret       [32]byte
}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{
		transactions: make(map[string]*fakeTransaction),
		secret:       sha256.Sum256([]byte("order-paymentorder-local-fake-v1")),
	}
}

func (provider *FakeProvider) CreateJSAPI(_ context.Context, request ProviderCreateRequest) (ProviderCreateResult, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil || !validProviderCreateRequest(request) {
		return ProviderCreateResult{}, ErrInvalidInput
	}
	digest := sha256.Sum256(requestJSON)
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if provider.transactions == nil {
		return ProviderCreateResult{}, ErrUnavailable
	}
	if existing := provider.transactions[request.OutTradeNo]; existing != nil {
		if existing.requestHash != digest {
			return ProviderCreateResult{}, ErrIdempotencyConflict
		}
		return existing.result, nil
	}
	prepayDigest := sha256.Sum256(append([]byte("fake-prepay\x00"), requestJSON...))
	prepayID := "fake_" + hex.EncodeToString(prepayDigest[:16])
	result := ProviderCreateResult{PrepayID: prepayID, RequestPayment: wechatpay.RequestPayment{
		TimeStamp: "1787623200",
		NonceStr:  hex.EncodeToString(prepayDigest[16:24]),
		Package:   "prepay_id=" + prepayID,
		SignType:  "RSA",
		PaySign:   hex.EncodeToString(prepayDigest[:]),
	}}
	provider.transactions[request.OutTradeNo] = &fakeTransaction{
		request: request, requestHash: digest, result: result, createCount: 1,
		transaction: wechatpay.Transaction{
			AppID: request.AppID, MerchantID: request.MerchantID, OutTradeNo: request.OutTradeNo,
			TradeType: "JSAPI", TradeState: "NOTPAY",
		},
	}
	return result, nil
}

func (provider *FakeProvider) QueryTransaction(_ context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.transactions[outTradeNo]
	if record == nil {
		return wechatpay.Transaction{}, ErrNotFound
	}
	record.queryCount++
	return record.transaction, nil
}

func (provider *FakeProvider) CloseTransaction(_ context.Context, outTradeNo string) error {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.transactions[outTradeNo]
	if record == nil {
		return ErrNotFound
	}
	if record.transaction.TradeState == "SUCCESS" {
		return ErrIdempotencyConflict
	}
	record.transaction.TradeState = "CLOSED"
	return nil
}

func (provider *FakeProvider) MarkPaid(outTradeNo, transactionID string, successAt time.Time) error {
	if transactionID == "" || successAt.IsZero() || !utf8.ValidString(transactionID) {
		return ErrInvalidInput
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.transactions[outTradeNo]
	if record == nil {
		return ErrNotFound
	}
	if record.transaction.TradeState == "SUCCESS" {
		if record.transaction.TransactionID == transactionID && record.transaction.SuccessTime.Equal(successAt.UTC()) {
			return nil
		}
		return ErrIdempotencyConflict
	}
	record.transaction.TradeState = "SUCCESS"
	record.transaction.TransactionID = transactionID
	record.transaction.SuccessTime = successAt.UTC()
	record.transaction.Amount = wechatpay.TransactionAmount{
		Total: record.request.AmountCents, PayerTotal: record.request.AmountCents,
		Currency: record.request.Currency, PayerCurrency: record.request.Currency,
	}
	return nil
}

func (provider *FakeProvider) CreateCount(outTradeNo string) uint64 {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if record := provider.transactions[outTradeNo]; record != nil {
		return record.createCount
	}
	return 0
}

func (provider *FakeProvider) QueryCount(outTradeNo string) uint64 {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if record := provider.transactions[outTradeNo]; record != nil {
		return record.queryCount
	}
	return 0
}

type fakePaymentNotification struct {
	ID          string                `json:"id"`
	Transaction wechatpay.Transaction `json:"transaction"`
}

func (provider *FakeProvider) PaymentNotification(outTradeNo, eventID string) ([]byte, wechatpay.SignatureHeaders, error) {
	if eventID == "" || !utf8.ValidString(eventID) {
		return nil, wechatpay.SignatureHeaders{}, ErrInvalidInput
	}
	transaction, err := provider.QueryTransaction(context.Background(), outTradeNo)
	if err != nil || transaction.TradeState != "SUCCESS" {
		return nil, wechatpay.SignatureHeaders{}, ErrUnavailable
	}
	body, err := json.Marshal(fakePaymentNotification{ID: eventID, Transaction: transaction})
	if err != nil {
		return nil, wechatpay.SignatureHeaders{}, ErrUnavailable
	}
	return body, wechatpay.SignatureHeaders{Serial: "FAKE", Signature: provider.fakeSignature(body), Timestamp: "1787623200", Nonce: "fake-notify"}, nil
}

func (provider *FakeProvider) ParsePaymentNotification(body []byte, headers wechatpay.SignatureHeaders) (VerifiedPayment, error) {
	if provider == nil || headers.Serial != "FAKE" || !hmac.Equal([]byte(headers.Signature), []byte(provider.fakeSignature(body))) {
		return VerifiedPayment{}, errFakeNotificationInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var envelope fakePaymentNotification
	if err := decoder.Decode(&envelope); err != nil || envelope.ID == "" || !utf8.Valid(body) {
		return VerifiedPayment{}, errFakeNotificationInvalid
	}
	if err := requireJSONEnd(decoder); err != nil || envelope.Transaction.TradeState != "SUCCESS" {
		return VerifiedPayment{}, errFakeNotificationInvalid
	}
	return VerifiedPayment{Source: paymentobservation.SourceCallback, ProviderEventID: envelope.ID, Transaction: envelope.Transaction}, nil
}

func (provider *FakeProvider) fakeSignature(body []byte) string {
	mac := hmac.New(sha256.New, provider.secret[:])
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func validProviderCreateRequest(request ProviderCreateRequest) bool {
	return request.AppID != "" && request.MerchantID != "" && request.Description != "" &&
		request.OutTradeNo != "" && request.TimeExpire != "" && request.NotifyURL != "" &&
		request.AmountCents > 0 && request.Currency == "CNY" && request.PayerOpenID != "" &&
		request.QuoteID != "" && request.QuoteDigest != "" &&
		len(request.AppID) <= 64 && len(request.MerchantID) <= 64 && len(request.Description) <= 128 &&
		len(request.OutTradeNo) <= 64 && len(request.TimeExpire) <= 64 && len(request.NotifyURL) <= 2048 &&
		len(request.PayerOpenID) <= 128 && len(request.QuoteID) <= 20 && len(request.QuoteDigest) == 64 &&
		!strings.ContainsRune(request.AppID+request.MerchantID+request.Description+request.OutTradeNo+
			request.TimeExpire+request.NotifyURL+request.Currency+request.PayerOpenID+request.QuoteID+request.QuoteDigest, '\x00') &&
		utf8.ValidString(request.AppID+request.MerchantID+request.Description+request.OutTradeNo+
			request.TimeExpire+request.NotifyURL+request.Currency+request.PayerOpenID+request.QuoteID+request.QuoteDigest)
}

func requireJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errFakeNotificationInvalid
}

var _ PaymentProvider = (*FakeProvider)(nil)
var _ NotificationParser = (*FakeProvider)(nil)
