package paymentorder

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

var (
	ErrInvalidInput        = errors.New("paymentorder invalid input")
	ErrUnauthenticated     = errors.New("paymentorder unauthenticated")
	ErrForbidden           = errors.New("paymentorder forbidden")
	ErrNotFound            = errors.New("paymentorder not found")
	ErrIdempotencyConflict = errors.New("paymentorder idempotency conflict")
	ErrQuoteUnavailable    = errors.New("paymentorder quote unavailable")
	ErrUnavailable         = errors.New("paymentorder unavailable")
)

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type ProviderState string

const (
	ProviderReady            ProviderState = "READY"
	ProviderCreateClaimed    ProviderState = "CREATE_CLAIMED"
	ProviderCreateUnknown    ProviderState = "CREATE_UNKNOWN"
	ProviderPaymentRequested ProviderState = "PAYMENT_REQUESTED"
	ProviderNotPaid          ProviderState = "NOT_PAID"
	ProviderPaid             ProviderState = "PAID"
	ProviderClosed           ProviderState = "CLOSED"
)

type MaterializationState string

const (
	MaterializationAwaitingPayment MaterializationState = "AWAITING_PAYMENT"
	MaterializationReady           MaterializationState = "READY"
	MaterializationApplied         MaterializationState = "APPLIED"
	MaterializationPendingManual   MaterializationState = "PENDING_MANUAL"
)

type MaterializationMode string

const (
	MaterializationAuto          MaterializationMode = "AUTO"
	MaterializationDelayedManual MaterializationMode = "DELAYED_MANUAL"
)

type Prepayment struct {
	ID                   uint64
	UserID               uint64
	QuoteID              uint64
	State                ProviderState
	MaterializationState MaterializationState
	ExpiresAt            time.Time
	WxRequestPayment     *wechatpay.RequestPayment
}

type PrepareResult struct {
	Prepayment Prepayment
	Created    bool
}

type ConfirmState string

const (
	ConfirmOrderCreated ConfirmState = "ORDER_CREATED"
	ConfirmPending      ConfirmState = "PENDING"
)

type ConfirmResult struct {
	State   ConfirmState
	OrderID uint64
}

type VerifiedPayment struct {
	Source          paymentobservation.Source
	ProviderEventID string
	Transaction     wechatpay.Transaction
}

type PendingFilter struct {
	Reason string
}

type PageQuery struct {
	AfterID uint64
	Limit   uint16
}

type PendingPayment struct {
	PrepaymentID uint64
	Reason       string
	CreatedAt    time.Time
}

type RunResult struct {
	Queried      uint16
	Observed     uint16
	Materialized uint16
	Pending      uint16
}

type Application interface {
	Prepare(context.Context, WriteMeta, uint64) (PrepareResult, error)
	Confirm(context.Context, WriteMeta, uint64) (ConfirmResult, error)
	IngestPayment(context.Context, VerifiedPayment) error
	RunDue(context.Context, time.Time, uint16) (RunResult, error)
	ListPending(context.Context, uint64, PendingFilter, PageQuery) ([]PendingPayment, error)
	MaterializePending(context.Context, WriteMeta, uint64) (ConfirmResult, error)
}

type QuoteSource interface {
	FinalizeForPrepayInTx(context.Context, *sql.Tx, uint64, uint64, time.Time) (quote.Quote, error)
	LoadSnapshotInTx(context.Context, *sql.Tx, uint64) (quote.Quote, error)
}

type ProviderCreateRequest struct {
	AppID       string `json:"appid"`
	MerchantID  string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	TimeExpire  string `json:"time_expire"`
	NotifyURL   string `json:"notify_url"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	PayerOpenID string `json:"payer_openid"`
	QuoteID     string `json:"quote_id"`
	QuoteDigest string `json:"quote_digest"`
}

type ProviderCreateResult struct {
	PrepayID       string
	RequestPayment wechatpay.RequestPayment
}

type PaymentProvider interface {
	CreateJSAPI(context.Context, ProviderCreateRequest) (ProviderCreateResult, error)
	QueryTransaction(context.Context, string) (wechatpay.Transaction, error)
	CloseTransaction(context.Context, string) error
}

type NotificationParser interface {
	ParsePaymentNotification([]byte, wechatpay.SignatureHeaders) (VerifiedPayment, error)
}
