package wechatpay

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiOrigin        = "https://api.mch.weixin.qq.com"
	clientTimeout    = 5 * time.Second
	responseMaxBytes = 1024 * 1024
)

// Config contains the material required at the APIv3 trust boundary.
// Client copies maps and byte slices and never exposes the retained values.
type Config struct {
	AppID                     string
	MerchantID                string
	MerchantCertificateSerial string
	MerchantPrivateKey        *rsa.PrivateKey
	WeChatPayPublicKeys       map[string]*rsa.PublicKey
	APIv3Key                  []byte
}

type nonceSource func() (string, error)

// Client performs signed APIv3 operations against the fixed provider origin.
type Client struct {
	appID              string
	merchantID         string
	merchantSerial     string
	merchantPrivateKey *rsa.PrivateKey
	providerPublicKeys map[string]*rsa.PublicKey
	apiV3Key           []byte
	httpClient         *http.Client
	origin             string
	now                func() time.Time
	nonce              nonceSource
}

// NewClient constructs a bounded runtime client with a fixed provider origin.
func NewClient(config Config) (*Client, error) {
	return newClient(config, &http.Client{Transport: newAPIv3Transport(), Timeout: clientTimeout}, apiOrigin, time.Now, randomNonce)
}

func newAPIv3Transport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	return transport
}

func newClient(config Config, source *http.Client, origin string, now func() time.Time, nonce nonceSource) (*Client, error) {
	if config.AppID == "" || config.MerchantID == "" || config.MerchantCertificateSerial == "" ||
		config.MerchantPrivateKey == nil || len(config.WeChatPayPublicKeys) == 0 || len(config.APIv3Key) != 32 ||
		origin == "" || now == nil || nonce == nil || !safeHeaderToken(config.MerchantID) || !safeHeaderToken(config.MerchantCertificateSerial) {
		return nil, &Error{kind: ErrorInvalidConfig}
	}
	keys := make(map[string]*rsa.PublicKey, len(config.WeChatPayPublicKeys))
	for serial, key := range config.WeChatPayPublicKeys {
		if serial == "" || key == nil || !safeHeaderToken(serial) {
			return nil, &Error{kind: ErrorInvalidConfig}
		}
		keys[serial] = key
	}
	client := http.Client{}
	if source != nil {
		client = *source
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{
		appID: config.AppID, merchantID: config.MerchantID, merchantSerial: config.MerchantCertificateSerial,
		merchantPrivateKey: config.MerchantPrivateKey, providerPublicKeys: keys,
		apiV3Key: append([]byte(nil), config.APIv3Key...), httpClient: &client,
		origin: strings.TrimRight(origin, "/"), now: now, nonce: nonce,
	}, nil
}

func (client *Client) do(ctx context.Context, method, requestTarget string, body []byte) ([]byte, error) {
	timestamp := strconv.FormatInt(client.now().Unix(), 10)
	nonce, err := client.nonce()
	if err != nil || !safeHeaderToken(nonce) {
		return nil, &Error{kind: ErrorProtocol}
	}
	message := method + "\n" + requestTarget + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	signature, err := signSHA256RSA(client.merchantPrivateKey, []byte(message))
	if err != nil {
		return nil, err
	}
	authorization := fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		client.merchantID, nonce, signature, timestamp, client.merchantSerial,
	)
	request, err := http.NewRequestWithContext(ctx, method, client.origin+requestTarget, bytes.NewReader(body))
	if err != nil {
		return nil, &Error{kind: ErrorProtocol}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", authorization)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		var networkError net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
			return nil, &Error{kind: ErrorTimeout}
		}
		return nil, &Error{kind: ErrorTransport}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, responseMaxBytes+1))
	if err != nil || len(responseBody) > responseMaxBytes {
		return nil, &Error{kind: ErrorProtocol}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		kind := ErrorProviderRejected
		switch {
		case response.StatusCode == http.StatusTooManyRequests:
			kind = ErrorRateLimited
		case response.StatusCode >= http.StatusInternalServerError:
			kind = ErrorProviderUnavailable
		}
		return nil, &Error{kind: kind, statusCode: response.StatusCode, providerCode: safeProviderCode(responseBody)}
	}
	if err := client.verify(responseBody, SignatureHeaders{
		Serial: response.Header.Get("Wechatpay-Serial"), Signature: response.Header.Get("Wechatpay-Signature"),
		Timestamp: response.Header.Get("Wechatpay-Timestamp"), Nonce: response.Header.Get("Wechatpay-Nonce"),
	}); err != nil {
		return nil, err
	}
	return responseBody, nil
}
