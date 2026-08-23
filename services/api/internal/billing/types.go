package billing

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrInvalidInput    = errors.New("billing invalid input")
	ErrForbidden       = errors.New("billing forbidden")
	ErrUnavailable     = errors.New("billing unavailable")
	ErrBillUnavailable = errors.New("billing provider bill unavailable")
	ErrBillMismatch    = errors.New("billing bill mismatch")
)

type EntryKind string

const (
	EntryPayment EntryKind = "PAYMENT"
	EntryRefund  EntryKind = "REFUND"
)

type BillEntry struct {
	Kind        EntryKind `json:"kind"`
	OutTradeNo  string    `json:"out_trade_no,omitempty"`
	OutRefundNo string    `json:"out_refund_no,omitempty"`
	ProviderID  string    `json:"provider_id"`
	AmountCents uint64    `json:"amount_cents"`
	Currency    string    `json:"currency"`
	State       string    `json:"state"`
	OccurredAt  time.Time `json:"occurred_at,omitempty"`
}

type TransactionBill struct {
	Date    time.Time
	Digest  [32]byte
	Entries []BillEntry
}

type BillProvider interface {
	DownloadTransactionBill(context.Context, time.Time) (TransactionBill, error)
}

type BillingRange struct{ From, To time.Time }
type PageQuery struct {
	AfterID uint64
	Limit   uint16
}

type Payment struct {
	OrderID                                   uint64
	OrderNo, OutTradeNo, TransactionID, State string
	AmountCents                               uint64
	PaidAt                                    time.Time
}

type Refund struct {
	ID, OrderID                          uint64
	OutRefundNo, ProviderRefundID, State string
	AmountCents                          uint64
	RequestedAt                          time.Time
}

type Summary struct {
	EffectiveRevenueCents uint64
	EffectiveOrders       uint64
	RefundCents           uint64
	RefundCount           uint64
	PendingRefunds        uint64
}

type ReconcileResult struct {
	BillDate     time.Time
	Digest       [32]byte
	Matched      uint64
	ProviderOnly []BillEntry
	SystemOnly   []BillEntry
}

type Application interface {
	Summary(context.Context, uint64, BillingRange) (Summary, error)
	ListPayments(context.Context, uint64, BillingRange, PageQuery) ([]Payment, uint64, error)
	ListRefunds(context.Context, uint64, BillingRange, PageQuery) ([]Refund, uint64, error)
	ExportCSV(context.Context, uint64, BillingRange) (io.ReadCloser, error)
	RunReconcile(context.Context, time.Time, uint16) (ReconcileResult, error)
}
