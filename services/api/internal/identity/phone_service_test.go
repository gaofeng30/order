package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechat"
)

func TestPhoneServiceBindsAndMasks(t *testing.T) {
	t.Parallel()
	provider := &phoneProviderStub{phone: "+8613712345678"}
	store := &phoneStoreStub{reads: []PhoneUser{{OpenID: "opaque-user"}}, boundPhone: "+8613712345678"}
	service := newPhoneService(provider, store, func() time.Time { return testNow })

	result, err := service.Bind(context.Background(), 42, "fresh-phone-code")
	if err != nil || result.MaskedPhone != "+*********5678" {
		t.Fatal("first phone binding result mismatch")
	}
	if provider.calls != 1 || provider.openID != "opaque-user" || store.bindCalls != 1 || store.boundAt != testNow.UTC().Truncate(time.Microsecond) {
		t.Fatal("first phone binding invocation mismatch")
	}
}

func TestPhoneServiceMasksMinimumE164(t *testing.T) {
	t.Parallel()
	binding, err := maskedBinding("+1")
	if err != nil || binding.MaskedPhone != "+*" {
		t.Fatal("minimum E.164 masking mismatch")
	}
}

func TestPhoneServiceBypassesProviderWhenBound(t *testing.T) {
	t.Parallel()
	provider := &phoneProviderStub{err: errors.New("must not be called")}
	store := &phoneStoreStub{reads: []PhoneUser{{OpenID: "opaque-user", PrimaryPhoneBound: true, PrimaryPhone: "+8613712345678"}}}
	service := newPhoneService(provider, store, func() time.Time { return testNow })

	result, err := service.Bind(context.Background(), 42, "unused-phone-code")
	if err != nil || result.MaskedPhone != "+*********5678" {
		t.Fatal("bound phone retry result mismatch")
	}
	if provider.calls != 0 || store.bindCalls != 0 {
		t.Fatal("bound retry reached provider or bind transaction")
	}
}

func TestPhoneServiceRecoversRejectedSameCode(t *testing.T) {
	t.Parallel()
	provider := &phoneProviderStub{err: wechat.ErrPhoneCodeRejected}
	store := &phoneStoreStub{reads: []PhoneUser{
		{OpenID: "opaque-user"},
		{OpenID: "opaque-user", PrimaryPhoneBound: true, PrimaryPhone: "+8613712345678"},
	}}
	service := newPhoneService(provider, store, func() time.Time { return testNow })

	result, err := service.Bind(context.Background(), 42, "concurrent-phone-code")
	if err != nil || result.MaskedPhone != "+*********5678" {
		t.Fatal("same-code recovery result mismatch")
	}
	if provider.calls != 1 || store.readCalls != 2 || store.bindCalls != 0 {
		t.Fatal("same-code recovery call sequence mismatch")
	}
}

func TestPhoneServiceRejectedCodeWithoutBindingStaysRejected(t *testing.T) {
	t.Parallel()
	provider := &phoneProviderStub{err: wechat.ErrPhoneCodeRejected}
	store := &phoneStoreStub{reads: []PhoneUser{{OpenID: "opaque-user"}, {OpenID: "opaque-user"}}}
	service := newPhoneService(provider, store, func() time.Time { return testNow })

	if _, err := service.Bind(context.Background(), 42, "rejected-phone-code"); !errors.Is(err, ErrPhoneCodeRejected) {
		t.Fatal("unrecovered rejected-code result mismatch")
	}
	if provider.calls != 1 || store.readCalls != 2 || store.bindCalls != 0 {
		t.Fatal("rejected-code call sequence mismatch")
	}
}

func TestPhoneServiceMapsStoreAndProviderFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		provider  error
		readError error
		bindError error
		want      error
	}{
		{name: "provider unavailable", provider: wechat.ErrUnavailable, want: ErrUnavailable},
		{name: "initial read", readError: errors.New("store canary"), want: ErrUnavailable},
		{name: "phone in use", bindError: ErrPhoneInUse, want: ErrPhoneInUse},
		{name: "already bound", bindError: ErrPrimaryPhoneAlreadyBound, want: ErrPrimaryPhoneAlreadyBound},
		{name: "bind unavailable", bindError: errors.New("store canary"), want: ErrUnavailable},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			provider := &phoneProviderStub{phone: "+8613712345678", err: test.provider}
			store := &phoneStoreStub{
				reads: []PhoneUser{{OpenID: "opaque-user"}}, readErr: test.readError,
				boundPhone: "+8613712345678", bindErr: test.bindError,
			}
			service := newPhoneService(provider, store, func() time.Time { return testNow })
			if _, err := service.Bind(context.Background(), 42, "phone-code"); !errors.Is(err, test.want) {
				t.Fatal("phone service failure mapping mismatch")
			}
		})
	}
}

func TestPhoneStatusReadsBoundAndUnboundWithoutWrites(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		user       PhoneUser
		wantBound  bool
		wantMasked string
	}{
		{name: "bound", user: PhoneUser{OpenID: "bound-user", PrimaryPhoneBound: true, PrimaryPhone: "+8613712345678"}, wantBound: true, wantMasked: "+*********5678"},
		{name: "unbound", user: PhoneUser{OpenID: "unbound-user"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			provider := &phoneProviderStub{err: errors.New("provider must not be called")}
			store := &phoneStoreStub{reads: []PhoneUser{test.user}, bindErr: errors.New("bind must not be called")}
			service := newPhoneService(provider, store, func() time.Time { return testNow })

			status, err := service.Status(context.Background(), 42)
			if err != nil || status.PrimaryPhoneBound != test.wantBound || status.MaskedPhone != test.wantMasked {
				t.Fatal("primary-phone status result mismatch")
			}
			if provider.calls != 0 || store.readCalls != 1 || store.bindCalls != 0 {
				t.Fatal("primary-phone status reached provider or write path")
			}
		})
	}
}

func TestPhoneStatusFailsClosedOnInvalidOrUnavailableState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		userID    uint64
		user      PhoneUser
		readError error
	}{
		{name: "zero user", user: PhoneUser{OpenID: "unexpected"}},
		{name: "read unavailable", userID: 42, readError: errors.New("store canary")},
		{name: "missing identity", userID: 42, user: PhoneUser{}},
		{name: "bound empty", userID: 42, user: PhoneUser{OpenID: "opaque-user", PrimaryPhoneBound: true}},
		{name: "bound malformed", userID: 42, user: PhoneUser{OpenID: "opaque-user", PrimaryPhoneBound: true, PrimaryPhone: "not-e164"}},
		{name: "unbound non-empty", userID: 42, user: PhoneUser{OpenID: "opaque-user", PrimaryPhone: "+8613712345678"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			provider := &phoneProviderStub{err: errors.New("provider must not be called")}
			store := &phoneStoreStub{reads: []PhoneUser{test.user}, readErr: test.readError, bindErr: errors.New("bind must not be called")}
			service := newPhoneService(provider, store, func() time.Time { return testNow })

			if _, err := service.Status(context.Background(), test.userID); !errors.Is(err, ErrUnavailable) {
				t.Fatal("invalid primary-phone status did not fail closed")
			}
			if provider.calls != 0 || store.bindCalls != 0 {
				t.Fatal("failed primary-phone status reached provider or write path")
			}
		})
	}
}

func TestPhoneStatusProtectsBindFromInconsistentStoredState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		user PhoneUser
	}{
		{name: "bound empty", user: PhoneUser{OpenID: "opaque-user", PrimaryPhoneBound: true}},
		{name: "bound malformed", user: PhoneUser{OpenID: "opaque-user", PrimaryPhoneBound: true, PrimaryPhone: "not-e164"}},
		{name: "unbound non-empty", user: PhoneUser{OpenID: "opaque-user", PrimaryPhone: "+8613712345678"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			provider := &phoneProviderStub{phone: "+8613712345678"}
			store := &phoneStoreStub{reads: []PhoneUser{test.user}, boundPhone: "+8613712345678"}
			service := newPhoneService(provider, store, func() time.Time { return testNow })

			if _, err := service.Bind(context.Background(), 42, "unused-code"); !errors.Is(err, ErrUnavailable) {
				t.Fatal("inconsistent stored state did not fail closed before bind")
			}
			if provider.calls != 0 || store.readCalls != 1 || store.bindCalls != 0 {
				t.Fatal("inconsistent stored state reached provider or bind transaction")
			}
		})
	}
}

type phoneProviderStub struct {
	phone  string
	err    error
	calls  int
	code   string
	openID string
}

func (provider *phoneProviderStub) Exchange(_ context.Context, code, openID string) (string, error) {
	provider.calls++
	provider.code = code
	provider.openID = openID
	return provider.phone, provider.err
}

type phoneStoreStub struct {
	reads      []PhoneUser
	readErr    error
	readCalls  int
	boundPhone string
	bindErr    error
	bindCalls  int
	boundAt    time.Time
}

func (store *phoneStoreStub) FindPhoneUser(context.Context, uint64) (PhoneUser, error) {
	store.readCalls++
	if store.readErr != nil {
		return PhoneUser{}, store.readErr
	}
	index := store.readCalls - 1
	if index >= len(store.reads) {
		index = len(store.reads) - 1
	}
	return store.reads[index], nil
}

func (store *phoneStoreStub) BindPrimaryPhone(_ context.Context, _ uint64, _ string, at time.Time) (string, error) {
	store.bindCalls++
	store.boundAt = at
	return store.boundPhone, store.bindErr
}
