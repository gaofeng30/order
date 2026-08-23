package subscription

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestWeChatProviderSendsOfficialSubscriptionMessageRequest(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Fatalf("request method = %q, want POST", request.Method)
		}
		if request.URL.String() != "https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=secret-token" {
			t.Fatalf("request URL = %q", request.URL.String())
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"touser":"openid-private","template_id":"template-private","page":"pages/orders/detail","miniprogram_state":"developer","lang":"zh_CN","data":{"character_string1":{"value":"ORDER-42"},"date2":{"value":"2026-08-25"},"thing4":{"value":"North gate"},"time3":{"value":"12:00"}}}`
		if string(body) != want {
			t.Fatalf("request body = %s, want %s", body, want)
		}
		return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}`), nil
	})
	provider := NewWeChatProvider(
		&http.Client{Transport: transport},
		staticTokenSource("secret-token"),
		staticRecipientResolver("openid-private"),
		staticTemplateResolver{template: ResolvedTemplate{
			TemplateID: "template-private",
			Page:       "pages/orders/detail",
			Data: map[string]string{
				"character_string1": "ORDER-42",
				"date2":             "2026-08-25",
				"time3":             "12:00",
				"thing4":            "North gate",
			},
		}},
		WeChatProviderConfig{MiniProgramState: "developer", Language: "zh_CN"},
	)

	result, err := provider.SendSubscription(context.Background(), readyDelivery())
	if err != nil {
		t.Fatalf("SendSubscription() error = %v", err)
	}
	const wantProviderMessageID = "request_9bc325b2421fc00ce0779be46ffaed1e988c6892ed792cc87564c6d3d0ae34d7"
	if result.ProviderMessageID != wantProviderMessageID {
		t.Fatalf("ProviderMessageID = %q, want %q", result.ProviderMessageID, wantProviderMessageID)
	}
	for _, secret := range []string{"secret-token", "openid-private", "template-private"} {
		if strings.Contains(result.ProviderMessageID, secret) {
			t.Fatalf("ProviderMessageID contains sensitive value %q", secret)
		}
	}
}

func TestWeChatProviderClassifiesOfficialErrorCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		errCode   int
		wantCode  string
		permanent bool
	}{
		{name: "invalid access token", errCode: 40001, wantCode: "WECHAT_ACCESS_TOKEN_INVALID"},
		{name: "invalid recipient", errCode: 40003, wantCode: "WECHAT_INVALID_RECIPIENT", permanent: true},
		{name: "invalid template", errCode: 40037, wantCode: "WECHAT_INVALID_TEMPLATE", permanent: true},
		{name: "user did not subscribe", errCode: 43101, wantCode: "WECHAT_USER_NOT_SUBSCRIBED", permanent: true},
		{name: "subscription capability blocked", errCode: 43107, wantCode: "WECHAT_SUBSCRIPTION_BLOCKED", permanent: true},
		{name: "concurrent send", errCode: 43108, wantCode: "WECHAT_CONCURRENT_SEND"},
		{name: "content rejected", errCode: 45168, wantCode: "WECHAT_CONTENT_REJECTED", permanent: true},
		{name: "invalid template data", errCode: 47003, wantCode: "WECHAT_INVALID_TEMPLATE_DATA", permanent: true},
		{name: "unknown provider error", errCode: 49999, wantCode: "WECHAT_ERROR"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, `{"errcode":`+strconv.Itoa(test.errCode)+`,"errmsg":"raw-private-detail"}`), nil
			}))

			_, err := provider.SendSubscription(context.Background(), readyDelivery())
			var sendError *SendError
			if !errors.As(err, &sendError) {
				t.Fatalf("SendSubscription() error = %T %v, want *SendError", err, err)
			}
			if sendError.Code != test.wantCode || sendError.Permanent != test.permanent {
				t.Fatalf("SendError = %#v, want code %q permanent %t", sendError, test.wantCode, test.permanent)
			}
			if strings.Contains(err.Error(), "raw-private-detail") {
				t.Fatalf("error leaks raw provider response: %q", err)
			}
		})
	}
}

func TestWeChatProviderRejectsInvalidTemplateMappingBeforeNetwork(t *testing.T) {
	t.Parallel()

	called := false
	provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}`), nil
	}))
	provider.templateResolver = staticTemplateResolver{template: ResolvedTemplate{
		TemplateID: "template-private",
		Page:       "pages/orders/detail",
		Data:       map[string]string{"_thing1": "ORDER-42"},
	}}

	_, err := provider.SendSubscription(context.Background(), readyDelivery())
	assertSendError(t, err, "TEMPLATE_INVALID", true)
	if called {
		t.Fatal("provider performed a network request for invalid template mapping")
	}
}

func TestWeChatProviderRejectsMalformedProviderResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		response    func() *http.Response
		wantCode    string
		permanent   bool
		privateText string
	}{
		{
			name: "non success HTTP status",
			response: func() *http.Response {
				return jsonResponse(http.StatusTooManyRequests, `{"private":"raw-private-detail"}`)
			},
			wantCode: "WECHAT_HTTP_STATUS", privateText: "raw-private-detail",
		},
		{
			name: "non JSON content type",
			response: func() *http.Response {
				response := jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}`)
				response.Header.Set("Content-Type", "text/plain")
				return response
			},
			wantCode: "WECHAT_RESPONSE_INVALID",
		},
		{
			name: "oversized body",
			response: func() *http.Response {
				return jsonResponse(http.StatusOK, strings.Repeat("x", maxWeChatResponseBodyBytes+1))
			},
			wantCode: "WECHAT_RESPONSE_INVALID",
		},
		{
			name: "unknown response field",
			response: func() *http.Response {
				return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok","private":"raw-private-detail"}`)
			},
			wantCode: "WECHAT_RESPONSE_INVALID", privateText: "raw-private-detail",
		},
		{
			name: "trailing response",
			response: func() *http.Response {
				return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}{"private":"raw-private-detail"}`)
			},
			wantCode: "WECHAT_RESPONSE_INVALID", privateText: "raw-private-detail",
		},
		{
			name: "missing error code",
			response: func() *http.Response {
				return jsonResponse(http.StatusOK, `{"errmsg":"raw-private-detail"}`)
			},
			wantCode: "WECHAT_RESPONSE_INVALID", privateText: "raw-private-detail",
		},
		{
			name: "error code over bound",
			response: func() *http.Response {
				return jsonResponse(http.StatusOK, `{"errcode":1000000,"errmsg":"raw-private-detail"}`)
			},
			wantCode: "WECHAT_RESPONSE_INVALID", privateText: "raw-private-detail",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return test.response(), nil
			}))
			_, err := provider.SendSubscription(context.Background(), readyDelivery())
			assertSendError(t, err, test.wantCode, test.permanent)
			if test.privateText != "" && strings.Contains(err.Error(), test.privateText) {
				t.Fatalf("error leaks provider response: %q", err)
			}
		})
	}
}

func TestWeChatProviderRedactsDependencyAndTransportErrors(t *testing.T) {
	t.Parallel()

	privateText := "secret-token openid-private template-private raw-private-detail"
	tests := []struct {
		name     string
		mutate   func(*WeChatProvider)
		wantCode string
	}{
		{
			name: "token source",
			mutate: func(provider *WeChatProvider) {
				provider.tokenSource = failingTokenSource{err: errors.New(privateText)}
			},
			wantCode: "TOKEN_UNAVAILABLE",
		},
		{
			name: "recipient resolver",
			mutate: func(provider *WeChatProvider) {
				provider.recipientResolver = failingRecipientResolver{err: errors.New(privateText)}
			},
			wantCode: "RECIPIENT_UNAVAILABLE",
		},
		{
			name: "template resolver",
			mutate: func(provider *WeChatProvider) {
				provider.templateResolver = staticTemplateResolver{err: errors.New(privateText)}
			},
			wantCode: "TEMPLATE_UNAVAILABLE",
		},
		{
			name: "HTTP transport",
			mutate: func(provider *WeChatProvider) {
				provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New(privateText)
				})
			},
			wantCode: "WECHAT_UNAVAILABLE",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, `{"errcode":0,"errmsg":"ok"}`), nil
			}))
			test.mutate(provider)
			_, err := provider.SendSubscription(context.Background(), readyDelivery())
			assertSendError(t, err, test.wantCode, false)
			if strings.Contains(err.Error(), privateText) {
				t.Fatalf("error leaks dependency detail: %q", err)
			}
		})
	}
}

func TestWeChatProviderKeepsAdapterMisconfigurationRetryable(t *testing.T) {
	t.Parallel()

	provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("network must not be called for invalid adapter configuration")
		return nil, nil
	}))
	provider.config.MiniProgramState = "private-state"

	_, err := provider.SendSubscription(context.Background(), readyDelivery())
	assertSendError(t, err, "ADAPTER_UNAVAILABLE", false)
}

func TestWeChatProviderPermanentlyRejectsCorruptedDelivery(t *testing.T) {
	t.Parallel()

	provider := newTestWeChatProvider(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("network must not be called for invalid delivery")
		return nil, nil
	}))
	delivery := readyDelivery()
	delivery.OutboxID = 0

	_, err := provider.SendSubscription(context.Background(), delivery)
	assertSendError(t, err, "DELIVERY_INVALID", true)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type staticTokenSource string

func (source staticTokenSource) AccessToken(context.Context) (string, error) {
	return string(source), nil
}

type failingTokenSource struct{ err error }

func (source failingTokenSource) AccessToken(context.Context) (string, error) {
	return "", source.err
}

type staticRecipientResolver string

func (resolver staticRecipientResolver) OpenID(context.Context, uint64) (string, error) {
	return string(resolver), nil
}

type failingRecipientResolver struct{ err error }

func (resolver failingRecipientResolver) OpenID(context.Context, uint64) (string, error) {
	return "", resolver.err
}

type staticTemplateResolver struct {
	template ResolvedTemplate
	err      error
}

func (resolver staticTemplateResolver) ResolveTemplate(context.Context, Kind, uint64, Message) (ResolvedTemplate, error) {
	return resolver.template, resolver.err
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func readyDelivery() Delivery {
	return Delivery{
		OutboxID:              11,
		OrderID:               42,
		RecipientUserID:       7,
		Kind:                  KindReady,
		TemplateConfigVersion: 3,
		AttemptCount:          1,
		Message: Message{
			OrderNumber: "ORDER-42",
			PickupDate:  "2026-08-25",
			PickupTime:  "12:00",
			PickupPoint: "North gate",
		},
	}
}

func newTestWeChatProvider(transport http.RoundTripper) *WeChatProvider {
	return NewWeChatProvider(
		&http.Client{Transport: transport},
		staticTokenSource("secret-token"),
		staticRecipientResolver("openid-private"),
		staticTemplateResolver{template: ResolvedTemplate{
			TemplateID: "template-private",
			Page:       "pages/orders/detail",
			Data: map[string]string{
				"character_string1": "ORDER-42",
			},
		}},
		WeChatProviderConfig{MiniProgramState: "developer", Language: "zh_CN"},
	)
}

func assertSendError(t *testing.T, err error, wantCode string, permanent bool) {
	t.Helper()
	var sendError *SendError
	if !errors.As(err, &sendError) {
		t.Fatalf("error = %T %v, want *SendError", err, err)
	}
	if sendError.Code != wantCode || sendError.Permanent != permanent {
		t.Fatalf("SendError = %#v, want code %q permanent %t", sendError, wantCode, permanent)
	}
}
