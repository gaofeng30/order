package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

var (
	// ErrPhoneCodeRejected is safe for a provider-rejected phone code.
	ErrPhoneCodeRejected = errors.New("phone code rejected")
	// ErrPhoneInUse means another internal user owns the canonical phone.
	ErrPhoneInUse = errors.New("phone already in use")
	// ErrPrimaryPhoneAlreadyBound means a concurrent request bound another phone first.
	ErrPrimaryPhoneAlreadyBound = errors.New("primary phone already bound")
)

// PhoneUser contains only the provider identity and optional canonical primary phone.
type PhoneUser struct {
	OpenID            string
	PrimaryPhoneBound bool
	PrimaryPhone      string
}

// PhoneBinding is the only successful phone-binding representation.
type PhoneBinding struct {
	MaskedPhone string
}

// PhoneStatus is the current user's minimal primary-phone representation.
type PhoneStatus struct {
	PrimaryPhoneBound bool
	MaskedPhone       string
}

// PhoneProvider resolves one phone code for the authenticated provider identity.
type PhoneProvider interface {
	Exchange(context.Context, string, string) (string, error)
}

// PhoneStore provides the preflight read and the short atomic binding transaction.
type PhoneStore interface {
	FindPhoneUser(context.Context, uint64) (PhoneUser, error)
	BindPrimaryPhone(context.Context, uint64, string, time.Time) (string, error)
}

// PhoneService binds one immutable primary phone to an existing Mini Program user.
type PhoneService struct {
	provider PhoneProvider
	store    PhoneStore
	now      func() time.Time
}

// NewPhoneService constructs the runtime phone-binding service.
func NewPhoneService(provider PhoneProvider, store PhoneStore) *PhoneService {
	return newPhoneService(provider, store, time.Now)
}

func newPhoneService(provider PhoneProvider, store PhoneStore, now func() time.Time) *PhoneService {
	return &PhoneService{provider: provider, store: store, now: now}
}

// Bind returns the existing binding or consumes one provider code before a short binding transaction.
func (service *PhoneService) Bind(ctx context.Context, userID uint64, code string) (PhoneBinding, error) {
	user, err := service.store.FindPhoneUser(ctx, userID)
	if err != nil || userID == 0 || user.OpenID == "" {
		return PhoneBinding{}, ErrUnavailable
	}
	if user.PrimaryPhoneBound {
		return maskedBinding(user.PrimaryPhone)
	}
	if user.PrimaryPhone != "" {
		return PhoneBinding{}, ErrUnavailable
	}

	phone, err := service.provider.Exchange(ctx, code, user.OpenID)
	if errors.Is(err, wechat.ErrPhoneCodeRejected) {
		current, readErr := service.store.FindPhoneUser(ctx, userID)
		if readErr != nil {
			return PhoneBinding{}, ErrUnavailable
		}
		if current.PrimaryPhoneBound {
			return maskedBinding(current.PrimaryPhone)
		}
		if current.PrimaryPhone != "" {
			return PhoneBinding{}, ErrUnavailable
		}
		return PhoneBinding{}, ErrPhoneCodeRejected
	}
	if err != nil {
		return PhoneBinding{}, ErrUnavailable
	}

	boundAt := service.now().UTC().Truncate(time.Microsecond)
	boundPhone, err := service.store.BindPrimaryPhone(ctx, userID, phone, boundAt)
	if err != nil {
		if errors.Is(err, ErrPhoneInUse) || errors.Is(err, ErrPrimaryPhoneAlreadyBound) {
			return PhoneBinding{}, err
		}
		return PhoneBinding{}, ErrUnavailable
	}
	return maskedBinding(boundPhone)
}

// Status reads only the authenticated user's current primary-phone state.
func (service *PhoneService) Status(ctx context.Context, userID uint64) (PhoneStatus, error) {
	user, err := service.store.FindPhoneUser(ctx, userID)
	if err != nil || userID == 0 || user.OpenID == "" {
		return PhoneStatus{}, ErrUnavailable
	}
	if !user.PrimaryPhoneBound {
		if user.PrimaryPhone != "" {
			return PhoneStatus{}, ErrUnavailable
		}
		return PhoneStatus{}, nil
	}
	binding, err := maskedBinding(user.PrimaryPhone)
	if err != nil {
		return PhoneStatus{}, ErrUnavailable
	}
	return PhoneStatus{PrimaryPhoneBound: true, MaskedPhone: binding.MaskedPhone}, nil
}

func maskedBinding(phone string) (PhoneBinding, error) {
	if len(phone) < 2 || phone[0] != '+' || !phoneDigits(phone[1:]) || len(phone) > 16 {
		return PhoneBinding{}, ErrUnavailable
	}
	digitCount := len(phone) - 1
	visibleCount := 4
	if digitCount <= visibleCount {
		visibleCount = digitCount - 1
	}
	hiddenCount := digitCount - visibleCount
	return PhoneBinding{MaskedPhone: "+" + strings.Repeat("*", hiddenCount) + phone[len(phone)-visibleCount:]}, nil
}

func phoneDigits(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return value != ""
}
