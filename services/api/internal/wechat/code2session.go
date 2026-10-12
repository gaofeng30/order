package wechat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	code2SessionEndpoint = "https://api.weixin.qq.com/sns/jscode2session"
	code2SessionTimeout  = 3 * time.Second
	maxResponseBytes     = 16 * 1024
)

var (
	// ErrLoginRejected identifies an invalid or blocked one-time login code.
	ErrLoginRejected = errors.New("miniprogram login rejected")
	// ErrUnavailable hides provider transport, protocol, quota, and system details.
	ErrUnavailable = errors.New("wechat unavailable")
)

// Credentials are the two structured values required by code2Session.
type Credentials struct {
	AppID     string
	AppSecret string
}

// Code2SessionClient exchanges one login code against the fixed official endpoint.
type Code2SessionClient struct {
	credentials Credentials
	httpClient  *http.Client
	endpoint    string
}

// NewCode2SessionClient constructs the runtime client without a configurable origin.
func NewCode2SessionClient(credentials Credentials) *Code2SessionClient {
	return newCode2SessionClient(
		credentials,
		&http.Client{Transport: newCode2SessionTransport(), Timeout: code2SessionTimeout},
		code2SessionEndpoint,
	)
}

func newCode2SessionTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	return transport
}

func newCode2SessionClient(credentials Credentials, source *http.Client, endpoint string) *Code2SessionClient {
	client := http.Client{}
	if source != nil {
		client = *source
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Code2SessionClient{credentials: credentials, httpClient: &client, endpoint: endpoint}
}

// Exchange returns only the opaque Mini Program user identifier.
func (client *Code2SessionClient) Exchange(ctx context.Context, code string) (string, error) {
	requestURL, err := url.Parse(client.endpoint)
	if err != nil {
		return "", ErrUnavailable
	}
	query := requestURL.Query()
	query.Set("appid", client.credentials.AppID)
	query.Set("secret", client.credentials.AppSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return "", ErrUnavailable
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ErrUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return "", ErrUnavailable
	}
	openid, providerCode, ok := decodeCode2SessionResponse(body)
	if !ok {
		return "", ErrUnavailable
	}
	switch providerCode {
	case 0:
		return openid, nil
	case 40029, 40226:
		return "", ErrLoginRejected
	default:
		return "", ErrUnavailable
	}
}

func decodeCode2SessionResponse(body []byte) (string, int, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return "", 0, false
	}
	fields := make(map[string]json.RawMessage, 5)
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, keyOK := keyToken.(string)
		if err != nil || !keyOK {
			return "", 0, false
		}
		if _, duplicate := fields[key]; duplicate {
			return "", 0, false
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return "", 0, false
		}
		fields[key] = value
	}
	last, err := decoder.Token()
	if err != nil || last != json.Delim('}') {
		return "", 0, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", 0, false
	}

	allowed := map[string]struct{}{
		"openid": {}, "session_key": {}, "unionid": {}, "errcode": {}, "errmsg": {},
	}
	for key := range fields {
		if _, ok := allowed[key]; !ok {
			return "", 0, false
		}
	}

	openid, ok := optionalString(fields, "openid")
	if !ok {
		return "", 0, false
	}
	sessionKey, ok := optionalString(fields, "session_key")
	if !ok {
		return "", 0, false
	}
	if _, ok := optionalString(fields, "unionid"); !ok {
		return "", 0, false
	}
	if _, ok := optionalString(fields, "errmsg"); !ok {
		return "", 0, false
	}
	providerCode := 0
	if raw, exists := fields["errcode"]; exists {
		if err := json.Unmarshal(raw, &providerCode); err != nil {
			return "", 0, false
		}
	}
	if providerCode != 0 {
		return "", providerCode, true
	}
	if openid == "" || sessionKey == "" {
		return "", 0, false
	}
	return openid, 0, true
}

func optionalString(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, exists := fields[key]
	if !exists {
		return "", true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}
