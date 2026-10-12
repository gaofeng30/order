package refund

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type fakeRefundRecord struct {
	request ProviderCreateRequest
	digest  [32]byte
	refund  ProviderRefund
	creates uint64
	queries uint64
}

// FakeProvider is deterministic and concurrency safe. It never performs network I/O.
type FakeProvider struct {
	mu         sync.Mutex
	merchantID string
	secret     [32]byte
	refunds    map[string]*fakeRefundRecord
}

func NewFakeProvider(merchantID string) *FakeProvider {
	return &FakeProvider{merchantID: merchantID, secret: sha256.Sum256([]byte("order-refund-local-fake-v1")), refunds: make(map[string]*fakeRefundRecord)}
}

func (provider *FakeProvider) CreateRefund(_ context.Context, request ProviderCreateRequest) (ProviderRefund, error) {
	if provider == nil || !validProviderCreateRequest(request) {
		return ProviderRefund{}, ErrInvalidInput
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return ProviderRefund{}, ErrInvalidInput
	}
	digest := sha256.Sum256(raw)
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if existing := provider.refunds[request.OutRefundNo]; existing != nil {
		if existing.digest != digest {
			return ProviderRefund{}, ErrIdempotencyConflict
		}
		return existing.refund, nil
	}
	idDigest := sha256.Sum256(append([]byte("fake-refund\x00"), raw...))
	transactionID := request.TransactionID
	if transactionID == "" {
		transactionID = "fake_tx_" + hex.EncodeToString(idDigest[16:24])
	}
	refund := ProviderRefund{
		MerchantID: provider.merchantID, OutTradeNo: request.OutTradeNo, TransactionID: transactionID,
		OutRefundNo: request.OutRefundNo, RefundID: "fake_" + hex.EncodeToString(idDigest[:16]),
		State: ProviderProcessing, AmountCents: request.AmountCents, TotalCents: request.TotalCents, Currency: request.Currency,
	}
	provider.refunds[request.OutRefundNo] = &fakeRefundRecord{request: request, digest: digest, refund: refund, creates: 1}
	return refund, nil
}

func (provider *FakeProvider) QueryRefund(_ context.Context, outRefundNo string) (ProviderRefund, error) {
	if provider == nil || outRefundNo == "" {
		return ProviderRefund{}, ErrInvalidInput
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.refunds[outRefundNo]
	if record == nil {
		return ProviderRefund{}, ErrNotFound
	}
	record.queries++
	return record.refund, nil
}

func (provider *FakeProvider) MarkSuccess(outRefundNo string, at time.Time) error {
	if provider == nil || outRefundNo == "" || at.IsZero() {
		return ErrInvalidInput
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.refunds[outRefundNo]
	if record == nil {
		return ErrNotFound
	}
	record.refund.State = ProviderSuccess
	record.refund.SuccessTime = at.UTC()
	return nil
}

func (provider *FakeProvider) MarkClosed(outRefundNo string) error {
	if provider == nil || outRefundNo == "" {
		return ErrInvalidInput
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	record := provider.refunds[outRefundNo]
	if record == nil {
		return ErrNotFound
	}
	if record.refund.State != ProviderSuccess {
		record.refund.State = ProviderClosed
		record.refund.SuccessTime = time.Time{}
	}
	return nil
}

func (provider *FakeProvider) CreateCount(outRefundNo string) uint64 {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if record := provider.refunds[outRefundNo]; record != nil {
		return record.creates
	}
	return 0
}

func (provider *FakeProvider) QueryCount(outRefundNo string) uint64 {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if record := provider.refunds[outRefundNo]; record != nil {
		return record.queries
	}
	return 0
}

type fakeNotification struct {
	ID     string         `json:"id"`
	Refund ProviderRefund `json:"refund"`
}

func (provider *FakeProvider) RefundNotification(outRefundNo, eventID string) ([]byte, wechatpay.SignatureHeaders, error) {
	refund, err := provider.QueryRefund(context.Background(), outRefundNo)
	if err != nil || eventID == "" || !utf8.ValidString(eventID) {
		return nil, wechatpay.SignatureHeaders{}, ErrInvalidInput
	}
	body, err := json.Marshal(fakeNotification{ID: eventID, Refund: refund})
	if err != nil {
		return nil, wechatpay.SignatureHeaders{}, ErrUnavailable
	}
	return body, wechatpay.SignatureHeaders{Serial: "FAKE", Signature: provider.signature(body), Timestamp: "1787623200", Nonce: "fake-refund-notify"}, nil
}

func (provider *FakeProvider) ParseRefundNotification(body []byte, headers wechatpay.SignatureHeaders) (VerifiedRefund, error) {
	if provider == nil || headers.Serial != "FAKE" || !hmac.Equal([]byte(headers.Signature), []byte(provider.signature(body))) || !utf8.Valid(body) {
		return VerifiedRefund{}, ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var message fakeNotification
	if decoder.Decode(&message) != nil || message.ID == "" || !validProviderRefund(message.Refund) {
		return VerifiedRefund{}, ErrInvalidInput
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return VerifiedRefund{}, ErrInvalidInput
	}
	return VerifiedRefund{Source: SourceCallback, ProviderEventID: message.ID, Refund: message.Refund}, nil
}

func (provider *FakeProvider) signature(body []byte) string {
	mac := hmac.New(sha256.New, provider.secret[:])
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func validProviderCreateRequest(request ProviderCreateRequest) bool {
	return request.OutTradeNo != "" && request.OutRefundNo != "" && request.Reason != "" &&
		request.NotifyURL != "" && request.AmountCents > 0 && request.AmountCents == request.TotalCents && request.Currency == "CNY" &&
		len(request.OutTradeNo) <= 64 && len(request.TransactionID) <= 64 && len(request.OutRefundNo) <= 64 &&
		len(request.Reason) <= 80 && len(request.NotifyURL) <= 2048 && utf8.ValidString(request.OutTradeNo+request.TransactionID+request.OutRefundNo+request.Reason+request.NotifyURL)
}

func validProviderRefund(refund ProviderRefund) bool {
	return refund.MerchantID != "" && refund.OutTradeNo != "" && refund.TransactionID != "" && refund.OutRefundNo != "" &&
		refund.RefundID != "" && refund.AmountCents > 0 && refund.TotalCents >= refund.AmountCents && refund.Currency == "CNY" &&
		(refund.State == ProviderProcessing || refund.State == ProviderSuccess || refund.State == ProviderClosed) &&
		(refund.State != ProviderSuccess || !refund.SuccessTime.IsZero())
}

var _ Provider = (*FakeProvider)(nil)
var _ NotificationParser = (*FakeProvider)(nil)
