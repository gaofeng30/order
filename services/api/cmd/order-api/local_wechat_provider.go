package main

import (
	"context"
	"errors"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

const (
	localWeChatOpenID = "order-local-openid"
	localWeChatPhone  = "+8613800000000"
)

var errLocalWeChatIdentityMismatch = errors.New("local wechat identity mismatch")

// localWeChatProvider supplies only deterministic, non-production identity facts
// required by local API, MySQL and official DevTools end-to-end tests.
type localWeChatProvider struct{}

func (localWeChatProvider) Exchange(_ context.Context, code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", wechat.ErrLoginRejected
	}
	return localWeChatOpenID, nil
}

func (localWeChatProvider) ExchangePhone(_ context.Context, code, openid string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", wechat.ErrPhoneCodeRejected
	}
	if openid != localWeChatOpenID {
		return "", errLocalWeChatIdentityMismatch
	}
	return localWeChatPhone, nil
}

type localWeChatPhoneProvider struct {
	provider localWeChatProvider
}

func (provider localWeChatPhoneProvider) Exchange(ctx context.Context, code, openid string) (string, error) {
	return provider.provider.ExchangePhone(ctx, code, openid)
}

func composeWeChatProviders(
	environment config.Environment,
	credentials wechat.Credentials,
) (identity.CodeExchanger, identity.PhoneProvider, *wechat.PhoneNumberClient, error) {
	switch environment {
	case config.Development, config.Test:
		local := localWeChatProvider{}
		return local, localWeChatPhoneProvider{provider: local}, nil, nil
	case config.Production:
		phone := wechat.NewPhoneNumberClient(credentials)
		return wechat.NewCode2SessionClient(credentials), phone, phone, nil
	default:
		return nil, nil, nil, errors.New("unsupported runtime environment")
	}
}
