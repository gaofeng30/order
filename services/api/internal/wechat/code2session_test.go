package wechat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAppID      = "wx-test-app-id"
	testAppSecret  = "test-app-secret-canary"
	testLoginCode  = "test-login-code-canary"
	testOpenID     = "test-openid-canary"
	testSessionKey = "test-session-key-canary"
)

func TestCode2SessionWireContract(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.Method != http.MethodGet || request.URL.Path != "/sns/jscode2session" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		want := map[string]string{
			"appid": testAppID, "secret": testAppSecret, "js_code": testLoginCode, "grant_type": "authorization_code",
		}
		if len(query) != len(want) {
			t.Errorf("query key count = %d", len(query))
		}
		for key, value := range want {
			if values, ok := query[key]; !ok || len(values) != 1 || values[0] != value {
				t.Errorf("query[%s] mismatch", key)
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"openid":%q,"session_key":%q,"unionid":"discarded","errcode":0,"errmsg":"ok"}`, testOpenID, testSessionKey)
	}))
	defer server.Close()

	client := newCode2SessionClient(
		Credentials{AppID: testAppID, AppSecret: testAppSecret},
		server.Client(),
		server.URL+"/sns/jscode2session",
	)
	openid, err := client.Exchange(context.Background(), testLoginCode)
	if err != nil || openid != testOpenID {
		t.Fatal("Exchange result mismatch")
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

func TestCode2SessionRuntimeClientIsFixedAndBounded(t *testing.T) {
	t.Parallel()
	client := NewCode2SessionClient(Credentials{AppID: testAppID, AppSecret: testAppSecret})
	if client.endpoint != code2SessionEndpoint {
		t.Fatalf("endpoint = %q", client.endpoint)
	}
	if client.httpClient.Timeout != 3*time.Second {
		t.Fatalf("timeout = %s", client.httpClient.Timeout)
	}
	if client.httpClient.CheckRedirect == nil {
		t.Fatal("redirect policy is absent")
	}
}

func TestCode2SessionAcceptsSuccessWithoutErrcode(t *testing.T) {
	t.Parallel()
	client, closeServer := code2SessionTestClient(t, http.StatusOK, `{"openid":"opaque-user","session_key":"discarded"}`)
	defer closeServer()
	openid, err := client.Exchange(context.Background(), "fresh-code")
	if err != nil || openid != "opaque-user" {
		t.Fatal("Exchange result mismatch")
	}
}

func TestCode2SessionRejectsLoginCodes(t *testing.T) {
	t.Parallel()
	for _, code := range []int{40029, 40226} {
		code := code
		t.Run(fmt.Sprintf("errcode_%d", code), func(t *testing.T) {
			t.Parallel()
			client, closeServer := code2SessionTestClient(t, http.StatusOK, fmt.Sprintf(`{"errcode":%d,"errmsg":"provider detail"}`, code))
			defer closeServer()
			openid, err := client.Exchange(context.Background(), testLoginCode)
			if openid != "" || !errors.Is(err, ErrLoginRejected) {
				t.Fatal("rejected Exchange result mismatch")
			}
			assertProviderCanariesAbsent(t, err)
		})
	}
}

func TestCode2SessionFailsClosed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "system busy", status: http.StatusOK, body: `{"errcode":-1}`},
		{name: "quota", status: http.StatusOK, body: `{"errcode":45011}`},
		{name: "unknown error", status: http.StatusOK, body: `{"errcode":49999}`},
		{name: "http failure", status: http.StatusBadGateway, body: `{"openid":"ignored","session_key":"ignored"}`},
		{name: "malformed", status: http.StatusOK, body: `{"openid":`},
		{name: "trailing json", status: http.StatusOK, body: `{"openid":"x","session_key":"y"}{}`},
		{name: "unknown field", status: http.StatusOK, body: `{"openid":"x","session_key":"y","extra":true}`},
		{name: "duplicate field", status: http.StatusOK, body: `{"openid":"x","openid":"y","session_key":"z"}`},
		{name: "wrong field type", status: http.StatusOK, body: `{"openid":1,"session_key":"y"}`},
		{name: "missing openid", status: http.StatusOK, body: `{"session_key":"y"}`},
		{name: "missing session key", status: http.StatusOK, body: `{"openid":"x"}`},
		{name: "empty success field", status: http.StatusOK, body: `{"openid":"","session_key":"y"}`},
		{name: "oversize", status: http.StatusOK, body: `{"openid":"x","session_key":"` + strings.Repeat("a", 16*1024) + `"}`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, closeServer := code2SessionTestClient(t, test.status, test.body)
			defer closeServer()
			openid, err := client.Exchange(context.Background(), testLoginCode)
			if openid != "" || !errors.Is(err, ErrUnavailable) {
				t.Fatal("unavailable Exchange result mismatch")
			}
			assertProviderCanariesAbsent(t, err)
		})
	}
}

func TestCode2SessionRefusesRedirect(t *testing.T) {
	t.Parallel()
	var redirected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/redirected" {
			redirected.Add(1)
			_, _ = writer.Write([]byte(`{"openid":"wrong","session_key":"wrong"}`))
			return
		}
		http.Redirect(writer, request, "/redirected", http.StatusFound)
	}))
	defer server.Close()
	client := newCode2SessionClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL+"/sns/jscode2session")
	if _, err := client.Exchange(context.Background(), testLoginCode); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Exchange() error = %v", err)
	}
	if redirected.Load() != 0 {
		t.Fatalf("redirect target calls = %d", redirected.Load())
	}
}

func TestCode2SessionTimesOutWithoutRetry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		<-request.Context().Done()
	}))
	defer server.Close()
	httpClient := server.Client()
	httpClient.Timeout = 25 * time.Millisecond
	client := newCode2SessionClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, httpClient, server.URL+"/sns/jscode2session")
	if _, err := client.Exchange(context.Background(), testLoginCode); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Exchange() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

func code2SessionTestClient(t *testing.T, status int, body string) (*Code2SessionClient, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}))
	client := newCode2SessionClient(Credentials{AppID: testAppID, AppSecret: testAppSecret}, server.Client(), server.URL+"/sns/jscode2session")
	return client, server.Close
}

func assertProviderCanariesAbsent(t *testing.T, err error) {
	t.Helper()
	message := err.Error()
	for _, canary := range []string{testAppSecret, testLoginCode, testOpenID, testSessionKey} {
		if strings.Contains(message, canary) {
			t.Fatalf("error contains provider canary")
		}
	}
}
