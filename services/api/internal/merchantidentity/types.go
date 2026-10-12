package merchantidentity

import "errors"

// Role is the only persisted merchant role vocabulary.
type Role string

// LoginCodeHash is the domain-separated, non-replayable proof used only for
// same-code merchant-login convergence.
type LoginCodeHash [32]byte

const (
	RoleOwner      Role = "OWNER"
	RoleSubaccount Role = "SUBACCOUNT"

	PricingVisitor PricingKind = "VISITOR"
	PricingStaff   PricingKind = "STAFF"
)

var (
	ErrMerchantAccountNotAvailable = errors.New("merchant account not available")
	ErrPhoneInUse                  = errors.New("phone already in use")
	ErrPrimaryPhoneMismatch        = errors.New("primary phone mismatch")
	ErrPhoneCodeRejected           = errors.New("phone code rejected")
	ErrForbidden                   = errors.New("merchant action forbidden")
	ErrInvalidInput                = errors.New("invalid identity input")
	ErrIdempotencyConflict         = errors.New("identity idempotency conflict")
	ErrPrimaryPhoneRequired        = errors.New("primary phone required")
	ErrUnavailable                 = errors.New("merchant identity unavailable")
)

// PricingKind is the only user pricing identity exposed to clients.
type PricingKind string

// PricingProjection contains current display pricing only; Quote repeats the
// same current-fact resolution transactionally before accepting payment.
type PricingProjection struct {
	Kind        PricingKind
	RatePercent uint8
}

// ExtraPhoneProjection is the single optional user-supplied staff claim. The
// phone is always masked before leaving the module.
type ExtraPhoneProjection struct {
	MaskedPhone string
	Name        string
}

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type ExtraPhoneCommand struct {
	Phone string
	Name  string
}

type ExtraPhoneResult struct {
	ExtraPhone ExtraPhoneProjection
	Pricing    PricingProjection
}

// MerchantProjection is the only merchant state exposed by identity responses.
type MerchantProjection struct {
	Role        Role
	AuthVersion uint64
}

// Identity is the complete non-sensitive identity projection.
type Identity struct {
	PrimaryPhoneBound  bool
	PrimaryPhoneMasked string
	ExtraPhone         *ExtraPhoneProjection
	Pricing            PricingProjection
	Merchant           *MerchantProjection
}
