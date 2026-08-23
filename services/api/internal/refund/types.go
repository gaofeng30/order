package refund

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

var (
	ErrInvalidInput         = errors.New("refund invalid input")
	ErrUnauthenticated      = errors.New("refund unauthenticated")
	ErrForbidden            = errors.New("refund forbidden")
	ErrNotFound             = errors.New("refund not found")
	ErrIdempotencyConflict  = errors.New("refund idempotency conflict")
	ErrTransitionNotAllowed = errors.New("refund transition not allowed")
	ErrUnavailable          = errors.New("refund unavailable")
)

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type ProviderState string

const (
	ProviderReady         ProviderState = "READY"
	ProviderCreateClaimed ProviderState = "CREATE_CLAIMED"
	ProviderCreateUnknown ProviderState = "CREATE_UNKNOWN"
	ProviderProcessing    ProviderState = "PROCESSING"
	ProviderSuccess       ProviderState = "SUCCESS"
	ProviderClosed        ProviderState = "CLOSED"
)

type MaterializationState string

const (
	MaterializationAwaitingProvider MaterializationState = "AWAITING_PROVIDER"
	MaterializationApplied          MaterializationState = "APPLIED"
	MaterializationPendingManual    MaterializationState = "PENDING_MANUAL"
)

type Source string

const (
	SourceCallback    Source = "CALLBACK"
	SourceActiveQuery Source = "ACTIVE_QUERY"
)

type ProviderCreateRequest struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id,omitempty"`
	OutRefundNo   string `json:"out_refund_no"`
	Reason        string `json:"reason"`
	NotifyURL     string `json:"notify_url"`
	AmountCents   uint64 `json:"amount_cents"`
	TotalCents    uint64 `json:"total_cents"`
	Currency      string `json:"currency"`
}

type ProviderRefund struct {
	MerchantID    string
	OutTradeNo    string
	TransactionID string
	OutRefundNo   string
	RefundID      string
	State         ProviderState
	AmountCents   uint64
	TotalCents    uint64
	Currency      string
	SuccessTime   time.Time
}

type ExpectedRefund struct {
	MerchantID    string
	OutTradeNo    string
	TransactionID string
	OutRefundNo   string
	AmountCents   uint64
	Currency      string
}

type VerifiedRefund struct {
	Source          Source
	ProviderEventID string
	Refund          ProviderRefund
}

type Provider interface {
	CreateRefund(context.Context, ProviderCreateRequest) (ProviderRefund, error)
	QueryRefund(context.Context, string) (ProviderRefund, error)
}

type NotificationParser interface {
	ParseRefundNotification([]byte, wechatpay.SignatureHeaders) (VerifiedRefund, error)
}

type Refund struct {
	ID                   uint64
	PrepaymentID         uint64
	OrderID              uint64
	State                ProviderState
	MaterializationState MaterializationState
	AmountCents          uint64
	Currency             string
	RequestedAt          time.Time
	ProviderRefundID     string
	PendingReason        string
}

type PendingFilter struct {
	AfterID uint64
	Limit   uint16
}

type RunResult struct {
	Claimed  uint16
	Observed uint16
	Pending  uint16
	Applied  uint16
}

// NotificationEnqueuer is the only cross-module refund-result seam. Implementations
// may return ErrNoConsent to keep the refund transaction independent of consent.
type NotificationEnqueuer interface {
	EnqueueRefundResultInTx(context.Context, *sql.Tx, uint64, uint64, string, time.Time) error
}

var ErrNoConsent = errors.New("refund notification consent unavailable")

type Application interface {
	RequestOrder(context.Context, WriteMeta, uint64, string) (Refund, error)
	RequestPaidPrepayment(context.Context, WriteMeta, uint64, string) (Refund, error)
	IngestRefund(context.Context, VerifiedRefund) error
	RunDue(context.Context, time.Time, uint16) (RunResult, error)
	ListPending(context.Context, uint64, PendingFilter) ([]Refund, error)
}
