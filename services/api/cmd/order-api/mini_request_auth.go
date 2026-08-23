package main

import (
	"context"
	"net/http"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
)

// miniRequestAuthenticator is the single strict bearer adapter shared by
// optional-auth public readers. A present malformed credential never degrades
// to anonymous access.
type miniRequestAuthenticator struct{ sessions miniSessionAuthenticator }

func (authenticator miniRequestAuthenticator) AuthenticateRequest(ctx context.Context, request *http.Request) (uint64, error) {
	if authenticator.sessions == nil {
		return 0, identity.ErrUnauthenticated
	}
	token, err := httpdto.BearerToken(request)
	if err != nil {
		return 0, identity.ErrUnauthenticated
	}
	return authenticator.sessions.Authenticate(ctx, token)
}
