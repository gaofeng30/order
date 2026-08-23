package main

import (
	"context"
	"errors"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

func TestComposeWeChatProvidersUsesDeterministicLocalIdentityOutsideProduction(t *testing.T) {
	for _, environment := range []config.Environment{config.Development, config.Test} {
		login, phone, productionPhone, err := composeWeChatProviders(environment, wechat.Credentials{AppID: "local-app", AppSecret: "local-secret"})
		if err != nil || login == nil || phone == nil || productionPhone != nil {
			t.Fatalf("environment %q: unexpected composition: login=%T phone=%T production=%v err=%v", environment, login, phone, productionPhone != nil, err)
		}
		openid, err := login.Exchange(context.Background(), "developer-tools-login-code")
		if err != nil || openid != localWeChatOpenID {
			t.Fatalf("environment %q: unexpected local login: openid=%q err=%v", environment, openid, err)
		}
		canonicalPhone, err := phone.Exchange(context.Background(), "developer-tools-phone-code", openid)
		if err != nil || canonicalPhone != localWeChatPhone {
			t.Fatalf("environment %q: unexpected local phone: phone=%q err=%v", environment, canonicalPhone, err)
		}
	}
}

func TestLocalWeChatProviderFailsClosedOnMissingOrMismatchedFacts(t *testing.T) {
	provider := localWeChatProvider{}
	if openid, err := provider.Exchange(context.Background(), " \t"); openid != "" || !errors.Is(err, wechat.ErrLoginRejected) {
		t.Fatalf("blank login code must be rejected: openid=%q err=%v", openid, err)
	}
	if phone, err := provider.ExchangePhone(context.Background(), "", localWeChatOpenID); phone != "" || !errors.Is(err, wechat.ErrPhoneCodeRejected) {
		t.Fatalf("blank phone code must be rejected: phone=%q err=%v", phone, err)
	}
	if phone, err := provider.ExchangePhone(context.Background(), "phone-code", "wrong-openid"); phone != "" || err == nil {
		t.Fatalf("mismatched openid must fail closed: phone=%q err=%v", phone, err)
	}
}

func TestComposeWeChatProvidersKeepsOfficialAdaptersInProduction(t *testing.T) {
	login, phone, productionPhone, err := composeWeChatProviders(config.Production, wechat.Credentials{AppID: "wx-app", AppSecret: "app-secret"})
	if err != nil || login == nil || phone == nil || productionPhone == nil {
		t.Fatalf("unexpected production composition: login=%T phone=%T production=%v err=%v", login, phone, productionPhone != nil, err)
	}
	if _, ok := login.(*wechat.Code2SessionClient); !ok {
		t.Fatalf("production login must use official adapter, got %T", login)
	}
	if official, ok := phone.(*wechat.PhoneNumberClient); !ok || official != productionPhone {
		t.Fatalf("production phone must use the shared official adapter, got %T", phone)
	}
}
