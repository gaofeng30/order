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
)

var (
	ErrMerchantAccountNotAvailable = errors.New("merchant account not available")
	ErrPhoneInUse                  = errors.New("phone already in use")
	ErrPrimaryPhoneMismatch        = errors.New("primary phone mismatch")
	ErrPhoneCodeRejected           = errors.New("phone code rejected")
	ErrForbidden                   = errors.New("merchant action forbidden")
	ErrUnavailable                 = errors.New("merchant identity unavailable")
)

// MerchantProjection is the only merchant state exposed by identity responses.
type MerchantProjection struct {
	Role        Role
	AuthVersion uint64
}

// Identity is the complete non-sensitive identity projection.
type Identity struct {
	PrimaryPhoneBound bool
	Merchant          *MerchantProjection
}
