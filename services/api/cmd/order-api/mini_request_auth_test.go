package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/identity"
)

func TestMiniRequestAuthenticatorRejectsMalformedPresentBearerWithoutCallingSession(t *testing.T) {
	sessions := &miniSessionStub{userID: 42}
	authenticator := miniRequestAuthenticator{sessions: sessions}
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Add("Authorization", "Bearer one")
	request.Header.Add("Authorization", "Bearer two")
	if _, err := authenticator.AuthenticateRequest(context.Background(), request); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("error = %v", err)
	}
	if sessions.calls != 0 {
		t.Fatal("malformed bearer reached session service")
	}
}

func TestMiniRequestAuthenticatorUsesOneExactBearer(t *testing.T) {
	sessions := &miniSessionStub{userID: 42}
	authenticator := miniRequestAuthenticator{sessions: sessions}
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer opaque")
	userID, err := authenticator.AuthenticateRequest(context.Background(), request)
	if err != nil || userID != 42 || sessions.token != "opaque" || sessions.calls != 1 {
		t.Fatalf("result = %d/%v, stub=%#v", userID, err, sessions)
	}
}

type miniSessionStub struct {
	userID uint64
	token  string
	calls  int
}

func (stub *miniSessionStub) Authenticate(_ context.Context, token string) (uint64, error) {
	stub.calls++
	stub.token = token
	return stub.userID, nil
}
