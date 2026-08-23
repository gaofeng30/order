package merchantidentity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

func TestMerchantLoginBypassesPhoneProviderForExistingEnabledBinding(t *testing.T) {
	existing := merchantidentity.Identity{
		PrimaryPhoneBound: true,
		Merchant:          &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 5},
	}
	store := &storeStub{start: merchantidentity.LoginStart{AlreadyBound: true, Existing: existing}}
	provider := &phoneProviderStub{}
	service := merchantidentity.NewService(store, provider)

	got, err := service.Login(context.Background(), 71, "fresh-code", "internal-request")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if got.Merchant == nil || got.Merchant.Role != merchantidentity.RoleOwner || got.Merchant.AuthVersion != 5 || !got.PrimaryPhoneBound {
		t.Fatalf("Login() = %#v", got)
	}
	if provider.calls != 0 || store.completeCalls != 0 || store.recoverCalls != 0 {
		t.Fatalf("calls = provider %d, complete %d, recover %d", provider.calls, store.completeCalls, store.recoverCalls)
	}
	if store.startHash == (merchantidentity.LoginCodeHash{}) {
		t.Fatal("existing binding audit did not receive a code hash")
	}
}

func TestMerchantLoginCallsProviderAtMostOnceAndUsesRestrictedRecovery(t *testing.T) {
	tests := []struct {
		name          string
		providerPhone string
		providerErr   error
		recovered     merchantidentity.Identity
		recoverErr    error
		wantErr       error
		wantComplete  int
		wantRecover   int
	}{
		{name: "provider unavailable", providerErr: errors.New("provider unavailable"), wantErr: merchantidentity.ErrUnavailable},
		{name: "invalid canonical phone", providerPhone: "+0", wantErr: merchantidentity.ErrUnavailable},
		{name: "rejected without proof", providerErr: wechat.ErrPhoneCodeRejected, recoverErr: merchantidentity.ErrPhoneCodeRejected, wantErr: merchantidentity.ErrPhoneCodeRejected, wantRecover: 1},
		{
			name: "rejected with committed proof", providerErr: wechat.ErrPhoneCodeRejected,
			recovered: merchantidentity.Identity{
				PrimaryPhoneBound: true,
				Merchant:          &merchantidentity.MerchantProjection{Role: merchantidentity.RoleSubaccount, AuthVersion: 2},
			},
			wantRecover: 1,
		},
		{
			name: "successful exchange", providerPhone: "+7",
			wantComplete: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &storeStub{
				start:      merchantidentity.LoginStart{OpenID: "opaque-provider-subject"},
				complete:   merchantidentity.Identity{PrimaryPhoneBound: true, Merchant: &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 2}},
				recovered:  test.recovered,
				recoverErr: test.recoverErr,
			}
			provider := &phoneProviderStub{phone: test.providerPhone, err: test.providerErr}
			service := merchantidentity.NewService(store, provider)
			_, err := service.Login(context.Background(), 72, "one-use-code", "internal-request")
			if test.wantErr == nil && err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, test.wantErr)
			}
			if provider.calls != 1 || store.completeCalls != test.wantComplete || store.recoverCalls != test.wantRecover {
				t.Fatalf("calls = provider %d complete %d recover %d", provider.calls, store.completeCalls, store.recoverCalls)
			}
			if store.startHash == (merchantidentity.LoginCodeHash{}) || (store.completeCalls == 1 && store.completeHash != store.startHash) || (store.recoverCalls == 1 && store.recoverHash != store.startHash) {
				t.Fatal("merchant login did not preserve one code hash across its database phases")
			}
		})
	}
}

type phoneProviderStub struct {
	phone string
	err   error
	calls int
}

func (stub *phoneProviderStub) Exchange(context.Context, string, string) (string, error) {
	stub.calls++
	return stub.phone, stub.err
}

type storeStub struct {
	identity      merchantidentity.Identity
	identityErr   error
	start         merchantidentity.LoginStart
	startErr      error
	complete      merchantidentity.Identity
	completeErr   error
	recovered     merchantidentity.Identity
	recoverErr    error
	completeCalls int
	recoverCalls  int
	startHash     merchantidentity.LoginCodeHash
	completeHash  merchantidentity.LoginCodeHash
	recoverHash   merchantidentity.LoginCodeHash
}

func (stub *storeStub) ReadIdentity(context.Context, uint64) (merchantidentity.Identity, error) {
	return stub.identity, stub.identityErr
}

func (stub *storeStub) SetExtraPhone(context.Context, merchantidentity.WriteMeta, merchantidentity.ExtraPhoneCommand) (merchantidentity.ExtraPhoneResult, error) {
	return merchantidentity.ExtraPhoneResult{}, nil
}

func (stub *storeStub) StartLogin(_ context.Context, _ uint64, codeHash merchantidentity.LoginCodeHash, _ string, _ time.Time) (merchantidentity.LoginStart, error) {
	stub.startHash = codeHash
	return stub.start, stub.startErr
}

func (stub *storeStub) CompleteLogin(_ context.Context, _ uint64, _ string, codeHash merchantidentity.LoginCodeHash, _ string, _ time.Time) (merchantidentity.Identity, error) {
	stub.completeCalls++
	stub.completeHash = codeHash
	return stub.complete, stub.completeErr
}

func (stub *storeStub) RecoverRejectedLogin(_ context.Context, _ uint64, codeHash merchantidentity.LoginCodeHash, _ string, _, _ time.Time) (merchantidentity.Identity, error) {
	stub.recoverCalls++
	stub.recoverHash = codeHash
	return stub.recovered, stub.recoverErr
}
