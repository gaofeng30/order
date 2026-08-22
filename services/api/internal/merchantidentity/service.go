package merchantidentity

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

// LoginStart is the PII-minimal result of the pre-provider login transaction.
type LoginStart struct {
	AlreadyBound bool
	Existing     Identity
	OpenID       string
}

// Store owns all durable merchant identity transactions.
type Store interface {
	ReadIdentity(context.Context, uint64) (Identity, error)
	StartLogin(context.Context, uint64, LoginCodeHash, string, time.Time) (LoginStart, error)
	CompleteLogin(context.Context, uint64, string, LoginCodeHash, string, time.Time) (Identity, error)
	RecoverRejectedLogin(context.Context, uint64, LoginCodeHash, string, time.Time, time.Time) (Identity, error)
}

// PhoneProvider exchanges one active-user phone code at most once.
type PhoneProvider interface {
	Exchange(context.Context, string, string) (string, error)
}

// Service coordinates provider exchange with short database transactions.
type Service struct {
	store    Store
	provider PhoneProvider
	now      func() time.Time
}

// NewService constructs the merchant identity application service.
func NewService(store Store, provider PhoneProvider) *Service {
	return newService(store, provider, time.Now)
}

func newService(store Store, provider PhoneProvider, now func() time.Time) *Service {
	return &Service{store: store, provider: provider, now: now}
}

// Identity returns only the current non-sensitive projection.
func (service *Service) Identity(ctx context.Context, userID uint64) (Identity, error) {
	if userID == 0 {
		return Identity{}, ErrUnavailable
	}
	projection, err := service.store.ReadIdentity(ctx, userID)
	if err != nil || !validIdentity(projection, false) {
		return Identity{}, ErrUnavailable
	}
	return projection, nil
}

// Login bypasses the provider for an existing enabled binding, otherwise consumes one code.
func (service *Service) Login(ctx context.Context, userID uint64, code, requestID string) (Identity, error) {
	if userID == 0 || requestID == "" {
		return Identity{}, ErrUnavailable
	}
	codeHash := hashLoginCode(code)
	startedAt := service.now().UTC().Truncate(time.Microsecond)
	start, err := service.store.StartLogin(ctx, userID, codeHash, requestID, startedAt)
	if err != nil {
		return Identity{}, err
	}
	if start.AlreadyBound {
		if !validIdentity(start.Existing, true) {
			return Identity{}, ErrUnavailable
		}
		return start.Existing, nil
	}
	if start.OpenID == "" {
		return Identity{}, ErrUnavailable
	}

	phone, err := service.provider.Exchange(ctx, code, start.OpenID)
	if errors.Is(err, wechat.ErrPhoneCodeRejected) {
		projection, recoverErr := service.store.RecoverRejectedLogin(
			ctx, userID, codeHash, requestID, startedAt, service.now().UTC().Truncate(time.Microsecond),
		)
		if recoverErr != nil {
			return Identity{}, recoverErr
		}
		if !validIdentity(projection, true) {
			return Identity{}, ErrUnavailable
		}
		return projection, nil
	}
	if err != nil || !canonicalPhone(phone) {
		return Identity{}, ErrUnavailable
	}
	projection, err := service.store.CompleteLogin(
		ctx, userID, phone, codeHash, requestID, service.now().UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		return Identity{}, err
	}
	if !validIdentity(projection, true) {
		return Identity{}, ErrUnavailable
	}
	return projection, nil
}

func hashLoginCode(code string) LoginCodeHash {
	return LoginCodeHash(sha256.Sum256([]byte("order:merchant-login-code:v1:\x00" + code)))
}

func validIdentity(projection Identity, requireMerchant bool) bool {
	if requireMerchant && projection.Merchant == nil {
		return false
	}
	if projection.Merchant == nil {
		return true
	}
	return projection.PrimaryPhoneBound &&
		(projection.Merchant.Role == RoleOwner || projection.Merchant.Role == RoleSubaccount) &&
		projection.Merchant.AuthVersion > 0
}

func canonicalPhone(phone string) bool {
	if len(phone) < 2 || len(phone) > 16 || phone[0] != '+' || phone[1] < '1' || phone[1] > '9' {
		return false
	}
	for index := 2; index < len(phone); index++ {
		if phone[index] < '0' || phone[index] > '9' {
			return false
		}
	}
	return true
}
