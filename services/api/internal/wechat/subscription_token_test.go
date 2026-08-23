package wechat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAccessTokenReusesPhoneClientStableTokenCache(t *testing.T) {
	t.Parallel()
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/token":
			tokenCalls.Add(1)
			_, _ = fmt.Fprintf(writer, `{"access_token":%q,"expires_in":7200}`, testAccessToken)
		case "/phone":
			_, _ = fmt.Fprintf(writer, `{"errcode":0,"phone_info":{"phoneNumber":"13712345678","purePhoneNumber":"13712345678","countryCode":"86","watermark":{"timestamp":1,"appid":%q}}}`, testAppID)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := newPhoneNumberClient(
		Credentials{AppID: testAppID, AppSecret: testAppSecret},
		server.Client(),
		server.URL+"/token",
		server.URL+"/phone",
		func() time.Time { return time.Unix(1_800_000_000, 0) },
	)
	token, err := client.AccessToken(context.Background())
	if err != nil || token != testAccessToken {
		t.Fatalf("AccessToken() = %q, %v", token, err)
	}
	phone, err := client.Exchange(context.Background(), testPhoneCode, testOpenID)
	if err != nil || phone != testE164Phone {
		t.Fatalf("Exchange() = %q, %v", phone, err)
	}
	if tokenCalls.Load() != 1 {
		t.Fatalf("stable-token calls = %d, want 1", tokenCalls.Load())
	}
}
