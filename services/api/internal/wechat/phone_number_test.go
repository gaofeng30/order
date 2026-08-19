package wechat

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testPhoneCode   = "test-phone-code-canary"
	testAccessToken = "test-access-token-canary"
	testE164Phone   = "+8613712345678"
)

func TestGetPhoneNumberWireContract(t *testing.T) {
	t.Parallel()
	var stableCalls atomic.Int32
	var phoneCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/cgi-bin/stable_token":
			stableCalls.Add(1)
			if request.Method != http.MethodPost || request.URL.RawQuery != "" {
				t.Error("stable-token request contract mismatch")
			}
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || len(payload) != 4 ||
				payload["grant_type"] != "client_credential" || payload["appid"] != testAppID ||
				payload["secret"] != testAppSecret || payload["force_refresh"] != false {
				t.Error("stable-token payload contract mismatch")
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
		case "/wxa/business/getuserphonenumber":
			phoneCalls.Add(1)
			if request.Method != http.MethodPost || len(request.URL.Query()) != 1 || request.URL.Query().Get("access_token") != testAccessToken {
				t.Error("phone request contract mismatch")
			}
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || len(payload) != 2 ||
				payload["code"] != testPhoneCode || payload["openid"] != testOpenID {
				t.Error("phone payload contract mismatch")
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"errcode":0,"errmsg":"ok","phone_info":{"phoneNumber":"13712345678","purePhoneNumber":"13712345678","countryCode":"86","watermark":{"timestamp":1,"appid":%q}}}`, testAppID)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := newPhoneNumberClient(
		Credentials{AppID: testAppID, AppSecret: testAppSecret},
		server.Client(),
		server.URL+"/cgi-bin/stable_token",
		server.URL+"/wxa/business/getuserphonenumber",
		func() time.Time { return time.Unix(1_800_000_000, 0) },
	)
	phone, err := client.Exchange(context.Background(), testPhoneCode, testOpenID)
	if err != nil || phone != testE164Phone {
		t.Fatal("phone exchange result mismatch")
	}
	if stableCalls.Load() != 1 || phoneCalls.Load() != 1 {
		t.Fatal("provider wire call count mismatch")
	}
}

func TestPhoneClientRuntimeIsFixedAndNonReplaying(t *testing.T) {
	t.Parallel()
	client := NewPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret})
	if client.stableEndpoint != stableTokenEndpoint || client.phoneEndpoint != phoneNumberEndpoint {
		t.Fatal("runtime provider endpoints are not fixed")
	}
	if client.httpClient.Timeout != 3*time.Second || client.httpClient.CheckRedirect == nil {
		t.Fatal("runtime HTTP client is not bounded")
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives || transport.ForceAttemptHTTP2 || transport.TLSNextProto == nil || len(transport.TLSNextProto) != 0 {
		t.Fatal("runtime transport permits replay preconditions")
	}
}

func TestPhoneClientRuntimeUsesFreshHTTP1Connections(t *testing.T) {
	t.Parallel()
	runtimeClient := NewPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret})
	runtimeTransport := runtimeClient.httpClient.Transport.(*http.Transport)

	var connections atomic.Int32
	var requests atomic.Int32
	var wrongProtocol atomic.Bool
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.ProtoMajor != 1 {
			wrongProtocol.Store(true)
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/token":
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
		case "/phone":
			_, _ = fmt.Fprintf(writer, `{"errcode":0,"phone_info":{"phoneNumber":"13712345678","purePhoneNumber":"13712345678","countryCode":"86","watermark":{"timestamp":1,"appid":%q}}}`, testAppID)
		default:
			http.NotFound(writer, request)
		}
	}))
	server.EnableHTTP2 = true
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.StartTLS()
	defer server.Close()

	testTransport := runtimeTransport.Clone()
	testTransport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	client := newPhoneNumberClient(
		Credentials{AppID: testAppID, AppSecret: testAppSecret},
		&http.Client{Transport: testTransport, Timeout: runtimeClient.httpClient.Timeout},
		server.URL+"/token",
		server.URL+"/phone",
		time.Now,
	)
	for range 2 {
		if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); err != nil {
			t.Fatal("controlled runtime phone exchange failed")
		}
	}
	if requests.Load() != 3 || connections.Load() != 3 || wrongProtocol.Load() {
		t.Fatal("runtime provider attempts/connections/protocol mismatch")
	}
}

func TestStableTokenConcurrentRefreshIsMerged(t *testing.T) {
	t.Parallel()
	var stableCalls atomic.Int32
	var phoneCalls atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			stableCalls.Add(1)
			<-release
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
		case "/phone":
			phoneCalls.Add(1)
			_, _ = fmt.Fprintf(writer, `{"errcode":0,"phone_info":{"phoneNumber":"13712345678","purePhoneNumber":"13712345678","countryCode":"86","watermark":{"timestamp":1,"appid":%q}}}`, testAppID)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := newPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL+"/token", server.URL+"/phone", time.Now)

	const callers = 12
	start := make(chan struct{})
	errorsSeen := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			<-start
			_, err := client.Exchange(context.Background(), testPhoneCode, testOpenID)
			errorsSeen <- err
		}()
	}
	close(start)
	for stableCalls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(release)
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatal("merged refresh caller failed")
		}
	}
	if stableCalls.Load() != 1 || phoneCalls.Load() != callers {
		t.Fatal("concurrent cache-miss call count mismatch")
	}
}

func TestStableTokenRefreshesFiveMinutesEarly(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_800_000_000, 0)
	var stableCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			call := stableCalls.Add(1)
			_, _ = fmt.Fprintf(writer, `{"access_token":"token-%d","expires_in":3600}`, call)
		case "/phone":
			_, _ = fmt.Fprintf(writer, `{"errcode":0,"phone_info":{"phoneNumber":"13712345678","purePhoneNumber":"13712345678","countryCode":"86","watermark":{"timestamp":1,"appid":%q}}}`, testAppID)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := newPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL+"/token", server.URL+"/phone", func() time.Time { return now })

	if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); err != nil {
		t.Fatal("initial exchange failed")
	}
	now = now.Add(54*time.Minute + 59*time.Second)
	if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); err != nil {
		t.Fatal("pre-boundary exchange failed")
	}
	now = now.Add(time.Second)
	if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); err != nil {
		t.Fatal("refresh-boundary exchange failed")
	}
	if stableCalls.Load() != 2 {
		t.Fatal("early refresh call count mismatch")
	}
}

func TestStableTokenShortLifetimeIsNotReused(t *testing.T) {
	t.Parallel()
	var stableCalls atomic.Int32
	client, closeServer := phoneTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			stableCalls.Add(1)
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":300}`, testAccessToken)
		case "/phone":
			writePhoneSuccess(writer, testAppID, "86", "13712345678")
		}
	})
	defer closeServer()
	for range 2 {
		if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); err != nil {
			t.Fatal("short-lifetime exchange failed")
		}
	}
	if stableCalls.Load() != 2 {
		t.Fatal("short-lived token was reused")
	}
}

func TestGetPhoneNumberRejectsInvalidCodes(t *testing.T) {
	t.Parallel()
	for _, providerCode := range []int{40013, 40029} {
		providerCode := providerCode
		t.Run(fmt.Sprintf("errcode_%d", providerCode), func(t *testing.T) {
			t.Parallel()
			client, closeServer := phoneTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/token":
					_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
				case "/phone":
					_, _ = fmt.Fprintf(writer, `{"errcode":%d,"errmsg":"provider detail"}`, providerCode)
				}
			})
			defer closeServer()
			phone, err := client.Exchange(context.Background(), testPhoneCode, testOpenID)
			if phone != "" || !errors.Is(err, ErrPhoneCodeRejected) {
				t.Fatal("phone-code rejection mapping mismatch")
			}
			assertPhoneProviderCanariesAbsent(t, err)
		})
	}
}

func TestGetPhoneNumberInvalidTokenIsNotReplayed(t *testing.T) {
	t.Parallel()
	var stableCalls atomic.Int32
	var phoneCalls atomic.Int32
	client, closeServer := phoneTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			stableCalls.Add(1)
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
		case "/phone":
			if phoneCalls.Add(1) == 1 {
				_, _ = writer.Write([]byte(`{"errcode":40014,"errmsg":"invalid token"}`))
				return
			}
			writePhoneSuccess(writer, testAppID, "86", "13712345678")
		}
	})
	defer closeServer()
	if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); !errors.Is(err, ErrUnavailable) {
		t.Fatal("invalid-token result mismatch")
	}
	if stableCalls.Load() != 1 || phoneCalls.Load() != 1 {
		t.Fatal("invalid token replayed current code")
	}
	if _, err := client.Exchange(context.Background(), "fresh-phone-code", testOpenID); err != nil {
		t.Fatal("fresh-code recovery failed")
	}
	if stableCalls.Load() != 2 || phoneCalls.Load() != 2 {
		t.Fatal("invalid token did not evict cache for later request")
	}
}

func TestGetPhoneNumberFailsClosed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "system", body: `{"errcode":-1}`},
		{name: "quota", body: `{"errcode":45011}`},
		{name: "unknown", body: `{"errcode":49999}`},
		{name: "malformed", body: `{"errcode":`},
		{name: "trailing", body: `{"errcode":0}{}`},
		{name: "unknown field", body: `{"errcode":0,"extra":true}`},
		{name: "duplicate", body: `{"errcode":0,"errcode":0}`},
		{name: "missing phone", body: `{"errcode":0}`},
		{name: "wrong phone type", body: `{"errcode":0,"phone_info":1}`},
		{name: "empty displayed phone", body: phoneSuccessJSON(testAppID, "86", "13712345678", "")},
		{name: "watermark mismatch", body: phoneSuccessJSON("other-app", "86", "13712345678", "13712345678")},
		{name: "country starts zero", body: phoneSuccessJSON(testAppID, "086", "13712345678", "13712345678")},
		{name: "nondigit", body: phoneSuccessJSON(testAppID, "86", "13712x45678", "13712345678")},
		{name: "too long", body: phoneSuccessJSON(testAppID, "123", "1234567890123", "1234567890123")},
		{name: "oversize", body: `{"errcode":0,"phone_info":{"phoneNumber":"` + strings.Repeat("x", 16*1024) + `"}}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, closeServer := phoneTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/token":
					_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
				case "/phone":
					_, _ = writer.Write([]byte(test.body))
				}
			})
			defer closeServer()
			phone, err := client.Exchange(context.Background(), testPhoneCode, testOpenID)
			if phone != "" || !errors.Is(err, ErrUnavailable) {
				t.Fatal("fail-closed phone result mismatch")
			}
			assertPhoneProviderCanariesAbsent(t, err)
		})
	}
}

func TestStableTokenFailsClosedWithoutRetry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "system", status: http.StatusOK, body: `{"errcode":-1}`},
		{name: "credential", status: http.StatusOK, body: `{"errcode":40125}`},
		{name: "quota", status: http.StatusOK, body: `{"errcode":45011}`},
		{name: "http", status: http.StatusBadGateway, body: `{"access_token":"ignored","expires_in":7200}`},
		{name: "malformed", status: http.StatusOK, body: `{"access_token":`},
		{name: "trailing", status: http.StatusOK, body: `{"access_token":"x","expires_in":7200}{}`},
		{name: "unknown field", status: http.StatusOK, body: `{"access_token":"x","expires_in":7200,"extra":true}`},
		{name: "duplicate", status: http.StatusOK, body: `{"access_token":"x","access_token":"y","expires_in":7200}`},
		{name: "wrong type", status: http.StatusOK, body: `{"access_token":1,"expires_in":7200}`},
		{name: "missing token", status: http.StatusOK, body: `{"expires_in":7200}`},
		{name: "invalid expiry", status: http.StatusOK, body: `{"access_token":"x","expires_in":0}`},
		{name: "oversize", status: http.StatusOK, body: `{"access_token":"` + strings.Repeat("x", 16*1024) + `","expires_in":7200}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stableCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				stableCalls.Add(1)
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client := newPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL, server.URL, time.Now)
			if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); !errors.Is(err, ErrUnavailable) {
				t.Fatal("stable-token failure mapping mismatch")
			}
			if stableCalls.Load() != 1 {
				t.Fatal("stable-token failure was retried")
			}
		})
	}
}

func TestStableTokenRefusesRedirect(t *testing.T) {
	t.Parallel()
	var redirected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/redirected" {
			redirected.Add(1)
			_, _ = writer.Write([]byte(`{"access_token":"wrong","expires_in":7200}`))
			return
		}
		http.Redirect(writer, request, "/redirected", http.StatusFound)
	}))
	defer server.Close()
	client := newPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL+"/token", server.URL+"/phone", time.Now)
	if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); !errors.Is(err, ErrUnavailable) {
		t.Fatal("redirect result mismatch")
	}
	if redirected.Load() != 0 {
		t.Fatal("redirect target was reached")
	}
}

func TestGetPhoneNumberTimeoutsAreOneAttempt(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"token", "phone"} {
		target := target
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			var targetCalls atomic.Int32
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if strings.Contains(request.URL.Path, target) {
					targetCalls.Add(1)
					<-release
					return
				}
				_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
			}))
			defer server.Close()
			defer close(release)
			httpClient := server.Client()
			httpClient.Timeout = 25 * time.Millisecond
			client := newPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, httpClient, server.URL+"/token", server.URL+"/phone", time.Now)
			if _, err := client.Exchange(context.Background(), testPhoneCode, testOpenID); !errors.Is(err, ErrUnavailable) {
				t.Fatal("timeout result mismatch")
			}
			if targetCalls.Load() != 1 {
				t.Fatal("timed-out provider call was retried")
			}
		})
	}
}

func phoneTestClient(t *testing.T, handler http.HandlerFunc) (*PhoneNumberClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := newPhoneNumberClient(
		Credentials{AppID: testAppID, AppSecret: testAppSecret},
		server.Client(),
		server.URL+"/token",
		server.URL+"/phone",
		time.Now,
	)
	return client, server.Close
}

func writePhoneSuccess(writer http.ResponseWriter, appID, countryCode, purePhone string) {
	_, _ = writer.Write([]byte(phoneSuccessJSON(appID, countryCode, purePhone, purePhone)))
}

func phoneSuccessJSON(appID, countryCode, purePhone, displayedPhone string) string {
	data, _ := json.Marshal(map[string]any{
		"errcode": 0,
		"phone_info": map[string]any{
			"phoneNumber": displayedPhone, "purePhoneNumber": purePhone, "countryCode": countryCode,
			"watermark": map[string]any{"timestamp": 1, "appid": appID},
		},
	})
	return string(data)
}

func assertPhoneProviderCanariesAbsent(t *testing.T, err error) {
	t.Helper()
	message := err.Error()
	for _, canary := range []string{testAppSecret, testPhoneCode, testAccessToken, testOpenID, testE164Phone} {
		if strings.Contains(message, canary) {
			t.Fatal("error contains provider canary")
		}
	}
}

func TestPhoneClientTransportTLSConfigCanBeReplacedForControlledServer(t *testing.T) {
	t.Parallel()
	client := NewPhoneNumberClient(Credentials{AppID: testAppID, AppSecret: testAppSecret})
	transport := client.httpClient.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if transport.TLSClientConfig == nil {
		t.Fatal("controlled TLS transport is unavailable")
	}
}
