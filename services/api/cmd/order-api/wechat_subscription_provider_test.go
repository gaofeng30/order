package main

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/internal/wechat"
)

type phoneUserFinderStub struct {
	gotUserID uint64
	user      identity.PhoneUser
	err       error
}

func (stub *phoneUserFinderStub) FindPhoneUser(_ context.Context, userID uint64) (identity.PhoneUser, error) {
	stub.gotUserID = userID
	return stub.user, stub.err
}

func TestIdentityRecipientResolverReturnsRepositoryOpenID(t *testing.T) {
	t.Parallel()
	repository := &phoneUserFinderStub{user: identity.PhoneUser{OpenID: "opaque-openid"}}
	resolver := identityRecipientResolver{repository: repository}

	openID, err := resolver.OpenID(context.Background(), 42)
	if err != nil || openID != "opaque-openid" {
		t.Fatalf("OpenID() = %q, %v", openID, err)
	}
	if repository.gotUserID != 42 {
		t.Fatalf("FindPhoneUser user id = %d, want 42", repository.gotUserID)
	}
}

func TestIdentityRecipientResolverFailsClosedWithoutLeakingIdentity(t *testing.T) {
	t.Parallel()
	privateText := "openid-private repository-private"
	tests := []struct {
		name       string
		userID     uint64
		repository phoneUserFinder
	}{
		{name: "zero user", repository: &phoneUserFinderStub{user: identity.PhoneUser{OpenID: privateText}}},
		{name: "repository unavailable", userID: 42, repository: &phoneUserFinderStub{err: errors.New(privateText)}},
		{name: "empty openid", userID: 42, repository: &phoneUserFinderStub{}},
		{name: "non canonical openid", userID: 42, repository: &phoneUserFinderStub{user: identity.PhoneUser{OpenID: " " + privateText}}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			resolver := identityRecipientResolver{repository: test.repository}
			openID, err := resolver.OpenID(context.Background(), test.userID)
			if openID != "" || !errors.Is(err, errWeChatSubscriptionRecipient) {
				t.Fatalf("OpenID() = %q, %v", openID, err)
			}
			if err != nil && (err.Error() == privateText || openID == privateText) {
				t.Fatal("recipient error leaks private dependency data")
			}
		})
	}
}

func TestLoadWeChatSubscriptionProviderConfigParsesCompleteEnvironment(t *testing.T) {
	t.Parallel()
	config, err := loadWeChatSubscriptionProviderConfig(mapEnvironmentLookup(validWeChatSubscriptionEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	if config.templateConfigVersion != 7 || config.readyTemplateID != "ready-template" || config.refundResultTemplateID != "refund-template" {
		t.Fatalf("template identity config = %#v", config)
	}
	if config.readyKeys != (readyTemplateKeys{
		orderNumber: "character_string1", pickupDate: "date2", pickupTime: "time3", pickupPoint: "thing4",
	}) {
		t.Fatalf("ready keys = %#v", config.readyKeys)
	}
	if config.refundResultKeys != (refundResultTemplateKeys{orderNumber: "character_string1", result: "phrase2"}) {
		t.Fatalf("refund-result keys = %#v", config.refundResultKeys)
	}
	if config.miniProgramState != "developer" || config.language != "zh_CN" {
		t.Fatalf("provider config = %#v", config)
	}
}

func TestLoadWeChatSubscriptionProviderConfigFailsClosedOnInvalidEnvironment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{name: "missing", mutate: func(values map[string]string) { delete(values, weChatSubscriptionReadyTemplateIDEnv) }},
		{name: "non canonical version", mutate: func(values map[string]string) { values[weChatSubscriptionVersionEnv] = "07" }},
		{name: "zero version", mutate: func(values map[string]string) { values[weChatSubscriptionVersionEnv] = "0" }},
		{name: "invalid key", mutate: func(values map[string]string) { values[weChatSubscriptionReadyPickupPointKeyEnv] = "_thing4" }},
		{name: "invalid template id", mutate: func(values map[string]string) { values[weChatSubscriptionReadyTemplateIDEnv] = "ready template" }},
		{name: "duplicate ready key", mutate: func(values map[string]string) {
			values[weChatSubscriptionReadyPickupTimeKeyEnv] = values[weChatSubscriptionReadyPickupDateKeyEnv]
		}},
		{name: "duplicate refund key", mutate: func(values map[string]string) {
			values[weChatSubscriptionRefundResultKeyEnv] = values[weChatSubscriptionRefundOrderNumberKeyEnv]
		}},
		{name: "duplicate template id", mutate: func(values map[string]string) {
			values[weChatSubscriptionRefundTemplateIDEnv] = values[weChatSubscriptionReadyTemplateIDEnv]
		}},
		{name: "invalid state", mutate: func(values map[string]string) { values[weChatSubscriptionMiniProgramStateEnv] = "private" }},
		{name: "invalid language", mutate: func(values map[string]string) { values[weChatSubscriptionLanguageEnv] = "zh" }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			values := validWeChatSubscriptionEnvironment()
			test.mutate(values)
			config, err := loadWeChatSubscriptionProviderConfig(mapEnvironmentLookup(values))
			if config != (weChatSubscriptionProviderConfig{}) || !errors.Is(err, errWeChatSubscriptionConfig) {
				t.Fatalf("config, error = %#v, %v", config, err)
			}
		})
	}
}

func TestStaticWeChatSubscriptionTemplateResolverMapsImmutableMessages(t *testing.T) {
	t.Parallel()
	config, err := loadWeChatSubscriptionProviderConfig(mapEnvironmentLookup(validWeChatSubscriptionEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	resolver := staticWeChatSubscriptionTemplateResolver{config: config}
	tests := []struct {
		name     string
		kind     subscription.Kind
		message  subscription.Message
		wantID   string
		wantData map[string]string
	}{
		{
			name: "ready", kind: subscription.KindReady, wantID: "ready-template",
			message:  subscription.Message{OrderNumber: "ORDER-42", PickupDate: "2026-08-25", PickupTime: "12:00", PickupPoint: "North gate"},
			wantData: map[string]string{"character_string1": "ORDER-42", "date2": "2026-08-25", "time3": "12:00", "thing4": "North gate"},
		},
		{
			name: "refund result", kind: subscription.KindRefundResult, wantID: "refund-template",
			message:  subscription.Message{OrderNumber: "ORDER-42", RefundResult: "REFUNDED"},
			wantData: map[string]string{"character_string1": "ORDER-42", "phrase2": "REFUNDED"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			resolved, err := resolver.ResolveTemplate(context.Background(), test.kind, 7, test.message)
			if err != nil {
				t.Fatal(err)
			}
			if resolved.TemplateID != test.wantID || resolved.Page != "pages/order-detail/index" || !reflect.DeepEqual(resolved.Data, test.wantData) {
				t.Fatalf("resolved template = %#v", resolved)
			}
		})
	}
}

func TestNewProductionWeChatSubscriptionProviderComposesOfficialDependencies(t *testing.T) {
	t.Parallel()
	config, err := loadWeChatSubscriptionProviderConfig(mapEnvironmentLookup(validWeChatSubscriptionEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newProductionWeChatSubscriptionProvider(
		boundedSubscriptionHTTPClient(),
		wechat.NewPhoneNumberClient(wechat.Credentials{AppID: "wx-app", AppSecret: "app-secret"}),
		identity.NewRepository(nil),
		config,
	)
	if err != nil || provider == nil {
		t.Fatalf("provider = %v, error = %v", provider, err)
	}
}

func TestNewProductionWeChatSubscriptionProviderFailsClosedOnUnsafeComposition(t *testing.T) {
	t.Parallel()
	config, err := loadWeChatSubscriptionProviderConfig(mapEnvironmentLookup(validWeChatSubscriptionEnvironment()))
	if err != nil {
		t.Fatal(err)
	}
	token := wechat.NewPhoneNumberClient(wechat.Credentials{AppID: "wx-app", AppSecret: "app-secret"})
	repository := identity.NewRepository(nil)
	tests := []struct {
		name       string
		client     *http.Client
		token      *wechat.PhoneNumberClient
		repository *identity.Repository
		config     weChatSubscriptionProviderConfig
	}{
		{name: "nil client", token: token, repository: repository, config: config},
		{name: "unbounded client", client: &http.Client{}, token: token, repository: repository, config: config},
		{name: "nil token", client: boundedSubscriptionHTTPClient(), repository: repository, config: config},
		{name: "nil repository", client: boundedSubscriptionHTTPClient(), token: token, config: config},
		{name: "invalid config", client: boundedSubscriptionHTTPClient(), token: token, repository: repository},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider, err := newProductionWeChatSubscriptionProvider(test.client, test.token, test.repository, test.config)
			if provider != nil || !errors.Is(err, errWeChatSubscriptionProvider) {
				t.Fatalf("provider, error = %v, %v", provider, err)
			}
		})
	}
}

func boundedSubscriptionHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func validWeChatSubscriptionEnvironment() map[string]string {
	return map[string]string{
		weChatSubscriptionVersionEnv:              "7",
		weChatSubscriptionReadyTemplateIDEnv:      "ready-template",
		weChatSubscriptionReadyOrderNumberKeyEnv:  "character_string1",
		weChatSubscriptionReadyPickupDateKeyEnv:   "date2",
		weChatSubscriptionReadyPickupTimeKeyEnv:   "time3",
		weChatSubscriptionReadyPickupPointKeyEnv:  "thing4",
		weChatSubscriptionRefundTemplateIDEnv:     "refund-template",
		weChatSubscriptionRefundOrderNumberKeyEnv: "character_string1",
		weChatSubscriptionRefundResultKeyEnv:      "phrase2",
		weChatSubscriptionMiniProgramStateEnv:     "developer",
		weChatSubscriptionLanguageEnv:             "zh_CN",
	}
}

func mapEnvironmentLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
