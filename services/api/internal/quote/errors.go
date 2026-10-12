package quote

import "errors"

var (
	// ErrInvalidInput identifies a caller-owned malformed quote request.
	ErrInvalidInput = errors.New("quote invalid input")
	// ErrPrimaryPhoneRequired identifies a user who has not completed required phone binding.
	ErrPrimaryPhoneRequired = errors.New("quote primary phone required")
	// ErrSelectionUnavailable identifies a currently non-orderable pickup or product selection.
	ErrSelectionUnavailable = errors.New("quote selection unavailable")
	// ErrIdempotencyConflict identifies reuse of a key for different request semantics.
	ErrIdempotencyConflict = errors.New("quote idempotency conflict")
	// ErrNotFound is shared by absent quotes and non-owner reads.
	ErrNotFound = errors.New("quote not found")
	// ErrExpired identifies observed_at at or after the quote's effective deadline.
	ErrExpired = errors.New("quote expired")
	// ErrQuoteStale identifies a current source fact that invalidates the frozen selection without repricing it.
	ErrQuoteStale = errors.New("quote stale")
	// ErrItemUnavailable identifies a product that can no longer be purchased for the frozen pickup date.
	ErrItemUnavailable = errors.New("quote item unavailable")
	// ErrPickupCutoffPassed identifies a frozen pickup selection observed at or after its live cutoff.
	ErrPickupCutoffPassed = errors.New("quote pickup cutoff passed")
	// ErrPaymentAmountTooSmall identifies a payable amount that cannot be sent to WeChat Pay.
	ErrPaymentAmountTooSmall = errors.New("quote payment amount too small")
	// ErrForbidden identifies an authorization rejection at the Quote boundary.
	ErrForbidden = errors.New("quote forbidden")
	// ErrSnapshotInvalid identifies durable corruption in a persisted immutable quote snapshot.
	ErrSnapshotInvalid = errors.New("quote snapshot invalid")
	// ErrUnavailable is the stable PII-free provider failure.
	ErrUnavailable = errors.New("quote unavailable")
	// ErrOperationReceiptExists is returned only by OperationReceiptStore when
	// its durable command-receipt UNIQUE key loses a concurrent first-write race.
	// Quote Create must roll back before replaying it in a new read-only transaction.
	ErrOperationReceiptExists = errors.New("quote operation receipt exists")
)
