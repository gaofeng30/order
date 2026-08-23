package quote

import (
	"context"
	"database/sql"
	"time"
)

// IdentityKind is the immutable quote-time user classification.
type IdentityKind string

const (
	IdentityStaff   IdentityKind = "STAFF"
	IdentityVisitor IdentityKind = "VISITOR"
)

// WriteMeta is the sole metadata contract for authenticated business mutations.
type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

// ReceiptAction is one stable operation-receipt command name.
type ReceiptAction string

const (
	ReceiptActionQuoteCreate ReceiptAction = "quote.create"
)

// OperationReceipt contains only a request digest and replayable non-PII response.
type OperationReceipt struct {
	RequestDigest [32]byte
	ResponseJSON  []byte
}

// OperationReceiptStore is implemented by the unified Audit module.
// Both methods run in the caller's transaction and the append must be insert-last.
// AppendInTx reports a durable UNIQUE collision as ErrOperationReceiptExists;
// the caller then rolls back all business writes before replaying in a new
// read-only transaction.
type OperationReceiptStore interface {
	ReplayInTx(context.Context, *sql.Tx, WriteMeta, ReceiptAction) (OperationReceipt, bool, error)
	AppendInTx(context.Context, *sql.Tx, WriteMeta, ReceiptAction, OperationReceipt) error
}

// CreateInput contains only caller-owned selection and preference facts.
type CreateInput struct {
	ContactName string
	PickupDate  string
	PickupTime  string
	OrderNote   string
	Items       []ItemInput
}

// ContactSnapshot freezes the caller-supplied name and server-owned bound primary phone.
type ContactSnapshot struct {
	Name  string
	Phone string
}

// ItemInput contains no client-owned price.
type ItemInput struct {
	ProductID uint64
	Quantity  int64
	Flavors   []string
	Note      string
}

// IdentitySnapshot freezes the resolved kind and whitelist source version.
type IdentitySnapshot struct {
	Kind          IdentityKind
	SourceVersion uint64
}

// DiscountSnapshot freezes the applied payable rate and settings version.
type DiscountSnapshot struct {
	RatePercent int64
	Version     uint64
}

// StoreSnapshot freezes the selected store identity.
type StoreSnapshot struct {
	Name    string
	Address string
}

// PickupSnapshot freezes the reservation selection and pickup point.
type PickupSnapshot struct {
	Date  string
	Time  string
	Meal  string
	Point string
}

// ItemSnapshot freezes one priced product line.
type ItemSnapshot struct {
	LineNumber               uint16
	ProductID                uint64
	ProductName              string
	ProductSourceVersion     [32]byte
	ImageObjectKey           string
	OriginalUnitPriceCents   int64
	DiscountedUnitPriceCents int64
	Quantity                 int64
	OriginalSubtotalCents    int64
	PayableSubtotalCents     int64
	Flavors                  []string
	Note                     string
}

// Quote is the complete immutable provider and HTTP read result.
type Quote struct {
	ID                    uint64
	UserID                uint64
	Contact               ContactSnapshot
	Identity              IdentitySnapshot
	Discount              DiscountSnapshot
	Store                 StoreSnapshot
	Pickup                PickupSnapshot
	OrderNote             string
	Items                 []ItemSnapshot
	OriginalSubtotalCents int64
	DiscountCents         int64
	PayableCents          int64
	SnapshotDigest        [32]byte
	CreatedAt             time.Time
	ExpiresAt             time.Time
}

// CreateResult distinguishes a new quote from an exact idempotent replay.
type CreateResult struct {
	Quote   Quote
	Created bool
}

// Application is the deep quote interface consumed by HTTP callers and tests.
type Application interface {
	Create(context.Context, WriteMeta, CreateInput) (CreateResult, error)
	Read(context.Context, uint64, uint64) (Quote, error)
}

// PrepayTransactionSource is the transaction-bound quote seam consumed by a future prepay module.
// The caller owns commit and rollback; this interface creates no prepayment or order records.
type PrepayTransactionSource interface {
	FinalizeForPrepayInTx(context.Context, *sql.Tx, uint64, uint64, time.Time) (Quote, error)
	LoadSnapshotInTx(context.Context, *sql.Tx, uint64) (Quote, error)
}
