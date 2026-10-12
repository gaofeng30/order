package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	stableTokenEndpoint = "https://api.weixin.qq.com/cgi-bin/stable_token"
	phoneNumberEndpoint = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"
	phoneClientTimeout  = 3 * time.Second
	tokenRefreshAdvance = 5 * time.Minute
)

var (
	// ErrPhoneCodeRejected identifies a phone code rejected for this Mini Program identity.
	ErrPhoneCodeRejected = errors.New("phone code rejected")
)

type tokenRefresh struct {
	done  chan struct{}
	token string
	err   error
}

// PhoneNumberClient exchanges one user-authorized phone code through fixed official endpoints.
type PhoneNumberClient struct {
	credentials    Credentials
	httpClient     *http.Client
	stableEndpoint string
	phoneEndpoint  string
	now            func() time.Time

	mu        sync.Mutex
	token     string
	refreshAt time.Time
	refresh   *tokenRefresh
}

// NewPhoneNumberClient constructs the runtime client without configurable provider origins.
func NewPhoneNumberClient(credentials Credentials) *PhoneNumberClient {
	return newPhoneNumberClient(
		credentials,
		&http.Client{Transport: newCode2SessionTransport(), Timeout: phoneClientTimeout},
		stableTokenEndpoint,
		phoneNumberEndpoint,
		time.Now,
	)
}

func newPhoneNumberClient(credentials Credentials, source *http.Client, stableEndpoint, phoneEndpoint string, now func() time.Time) *PhoneNumberClient {
	client := http.Client{}
	if source != nil {
		client = *source
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &PhoneNumberClient{
		credentials: credentials, httpClient: &client, stableEndpoint: stableEndpoint, phoneEndpoint: phoneEndpoint, now: now,
	}
}

// Exchange verifies one phone code for the current provider identity and returns only canonical E.164.
func (client *PhoneNumberClient) Exchange(ctx context.Context, code, openid string) (string, error) {
	token, err := client.stableToken(ctx)
	if err != nil {
		return "", ErrUnavailable
	}
	phone, invalidToken, err := client.exchangePhone(ctx, token, code, openid)
	if invalidToken {
		client.evictToken(token)
	}
	if err != nil {
		return "", err
	}
	return phone, nil
}

func (client *PhoneNumberClient) stableToken(ctx context.Context) (string, error) {
	now := client.now()
	client.mu.Lock()
	if client.token != "" && now.Before(client.refreshAt) {
		token := client.token
		client.mu.Unlock()
		return token, nil
	}
	if current := client.refresh; current != nil {
		client.mu.Unlock()
		select {
		case <-current.done:
			return current.token, current.err
		case <-ctx.Done():
			return "", ErrUnavailable
		}
	}
	current := &tokenRefresh{done: make(chan struct{})}
	client.refresh = current
	client.mu.Unlock()

	token, expiresIn, err := client.fetchStableToken(ctx)
	client.mu.Lock()
	if err == nil && expiresIn > tokenRefreshAdvance {
		client.token = token
		client.refreshAt = now.Add(expiresIn - tokenRefreshAdvance)
	}
	current.token = token
	current.err = err
	close(current.done)
	client.refresh = nil
	client.mu.Unlock()
	return token, err
}

func (client *PhoneNumberClient) fetchStableToken(ctx context.Context) (string, time.Duration, error) {
	payload, err := json.Marshal(struct {
		GrantType    string `json:"grant_type"`
		AppID        string `json:"appid"`
		Secret       string `json:"secret"`
		ForceRefresh bool   `json:"force_refresh"`
	}{GrantType: "client_credential", AppID: client.credentials.AppID, Secret: client.credentials.AppSecret, ForceRefresh: false})
	if err != nil {
		return "", 0, ErrUnavailable
	}
	body, err := client.postJSON(ctx, client.stableEndpoint, payload)
	if err != nil {
		return "", 0, ErrUnavailable
	}
	fields, ok := decodeStrictObject(body, map[string]struct{}{
		"access_token": {}, "expires_in": {}, "errcode": {}, "errmsg": {},
	})
	if !ok {
		return "", 0, ErrUnavailable
	}
	providerCode, ok := optionalInteger(fields, "errcode")
	if !ok || providerCode != 0 {
		return "", 0, ErrUnavailable
	}
	if _, ok := optionalJSONText(fields, "errmsg"); !ok {
		return "", 0, ErrUnavailable
	}
	token, exists, ok := requiredJSONText(fields, "access_token")
	if !exists || !ok || token == "" {
		return "", 0, ErrUnavailable
	}
	expires, exists, ok := requiredJSONInteger(fields, "expires_in")
	if !exists || !ok || expires <= 0 {
		return "", 0, ErrUnavailable
	}
	return token, time.Duration(expires) * time.Second, nil
}

func (client *PhoneNumberClient) exchangePhone(ctx context.Context, token, code, openid string) (string, bool, error) {
	requestURL, err := url.Parse(client.phoneEndpoint)
	if err != nil {
		return "", false, ErrUnavailable
	}
	query := requestURL.Query()
	query.Set("access_token", token)
	requestURL.RawQuery = query.Encode()
	payload, err := json.Marshal(struct {
		Code   string `json:"code"`
		OpenID string `json:"openid"`
	}{Code: code, OpenID: openid})
	if err != nil {
		return "", false, ErrUnavailable
	}
	body, err := client.postJSON(ctx, requestURL.String(), payload)
	if err != nil {
		return "", false, ErrUnavailable
	}
	fields, ok := decodeStrictObject(body, map[string]struct{}{
		"errcode": {}, "errmsg": {}, "phone_info": {},
	})
	if !ok {
		return "", false, ErrUnavailable
	}
	providerCode, ok := optionalInteger(fields, "errcode")
	if !ok {
		return "", false, ErrUnavailable
	}
	if _, ok := optionalJSONText(fields, "errmsg"); !ok {
		return "", false, ErrUnavailable
	}
	switch providerCode {
	case 0:
	case 40013, 40029:
		return "", false, ErrPhoneCodeRejected
	case 40001, 40014, 42001:
		return "", true, ErrUnavailable
	default:
		return "", false, ErrUnavailable
	}
	phoneInfo, exists := fields["phone_info"]
	if !exists {
		return "", false, ErrUnavailable
	}
	phoneFields, ok := decodeStrictObject(phoneInfo, map[string]struct{}{
		"phoneNumber": {}, "purePhoneNumber": {}, "countryCode": {}, "watermark": {},
	})
	if !ok {
		return "", false, ErrUnavailable
	}
	displayed, exists, ok := requiredJSONText(phoneFields, "phoneNumber")
	if !exists || !ok || displayed == "" {
		return "", false, ErrUnavailable
	}
	pure, exists, ok := requiredJSONText(phoneFields, "purePhoneNumber")
	if !exists || !ok || pure == "" || !asciiDigits(pure) {
		return "", false, ErrUnavailable
	}
	country, exists, ok := requiredJSONText(phoneFields, "countryCode")
	if !exists || !ok || country == "" || country[0] == '0' || !asciiDigits(country) {
		return "", false, ErrUnavailable
	}
	combined := country + pure
	if len(combined) == 0 || len(combined) > 15 {
		return "", false, ErrUnavailable
	}
	watermark, exists := phoneFields["watermark"]
	if !exists {
		return "", false, ErrUnavailable
	}
	watermarkFields, ok := decodeStrictObject(watermark, map[string]struct{}{"timestamp": {}, "appid": {}})
	if !ok {
		return "", false, ErrUnavailable
	}
	if _, exists, ok := requiredJSONInteger(watermarkFields, "timestamp"); !exists || !ok {
		return "", false, ErrUnavailable
	}
	appID, exists, ok := requiredJSONText(watermarkFields, "appid")
	if !exists || !ok || appID != client.credentials.AppID {
		return "", false, ErrUnavailable
	}
	return "+" + combined, false, nil
}

func (client *PhoneNumberClient) postJSON(ctx context.Context, endpoint string, payload []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, ErrUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return nil, ErrUnavailable
	}
	return body, nil
}

func (client *PhoneNumberClient) evictToken(token string) {
	client.mu.Lock()
	if client.token == token {
		client.token = ""
		client.refreshAt = time.Time{}
	}
	client.mu.Unlock()
}

func decodeStrictObject(body []byte, allowed map[string]struct{}) (map[string]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return nil, false
	}
	fields := make(map[string]json.RawMessage, len(allowed))
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, keyOK := keyToken.(string)
		if err != nil || !keyOK {
			return nil, false
		}
		if _, allowedField := allowed[key]; !allowedField {
			return nil, false
		}
		if _, duplicate := fields[key]; duplicate {
			return nil, false
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false
		}
		fields[key] = value
	}
	last, err := decoder.Token()
	if err != nil || last != json.Delim('}') {
		return nil, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	return fields, true
}

func optionalInteger(fields map[string]json.RawMessage, key string) (int64, bool) {
	raw, exists := fields[key]
	if !exists {
		return 0, true
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}

func requiredJSONInteger(fields map[string]json.RawMessage, key string) (int64, bool, bool) {
	raw, exists := fields[key]
	if !exists {
		return 0, false, true
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, true, false
	}
	return value, true, true
}

func optionalJSONText(fields map[string]json.RawMessage, key string) (string, bool) {
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

func requiredJSONText(fields map[string]json.RawMessage, key string) (string, bool, bool) {
	raw, exists := fields[key]
	if !exists {
		return "", false, true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", true, false
	}
	return value, true, true
}

func asciiDigits(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}
