package config

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

const uatCOSBucket = "6f72-order-uat-d1gtzpzj50615c300-1473835269"
const uatCOSRegion = "ap-shanghai"
const uatCOSPolicy = `{"version":"2.0","statement":[{"effect":"allow","action":["name/cos:PutObject","name/cos:GetObject"],"resource":["qcs::cos:ap-shanghai:uid/1473835269:6f72-order-uat-d1gtzpzj50615c300-1473835269/images/*"]}]}`

// NewUATCOSCredentialProvider is the explicit UAT source-identity route.
func NewUATCOSCredentialProvider(material COSMaterial, secretID, secretKey string) (common.Provider, error) {
	return newUATCOSCredentialProvider(material, secretID, secretKey, nil, time.Now)
}
func newUATCOSCredentialProvider(material COSMaterial, secretID, secretKey string, transport http.RoundTripper, now func() time.Time) (common.Provider, error) {
	if material.Bucket() != uatCOSBucket || material.Region() != uatCOSRegion || !validCredentialComponent(secretID) || !validCredentialComponent(secretKey) || now == nil {
		return nil, errCloudCredentialUnavailable
	}
	p := profile.NewClientProfile()
	p.HttpProfile.Endpoint = "sts.tencentcloudapi.com"
	p.HttpProfile.ReqTimeout = int(secretLoadTimeout.Seconds())
	p.DisableRegionBreaker = true
	p.NetworkFailureMaxRetries = 0
	p.RateLimitExceededMaxRetries = 0
	p.UnsafeRetryOnConnectionFailure = false
	p.Debug = false
	client, err := sts.NewClient(common.NewCredential(secretID, secretKey), uatCOSRegion, p)
	if err != nil {
		return nil, errCloudCredentialUnavailable
	}
	client.WithHttpTransport(uatCOSSTSBoundedTransport{next: transport})
	return &uatCOSProvider{credential: &uatCOSCredential{client: client, now: now}}, nil
}

type uatCOSProvider struct{ credential *uatCOSCredential }
type uatCOSCredential struct {
	client   *sts.Client
	now      func() time.Time
	mu       sync.Mutex
	cached   temporaryCredential
	inFlight *credentialExchange
}

func (p *uatCOSProvider) GetCredential() (common.CredentialIface, error) {
	if p == nil || p.credential == nil {
		return nil, errCloudCredentialUnavailable
	}
	if _, err := p.credential.load(); err != nil {
		return nil, errCloudCredentialUnavailable
	}
	return p.credential, nil
}
func (c *uatCOSCredential) GetCredential() (string, string, string) {
	v, err := c.load()
	if err != nil {
		return "", "", ""
	}
	return v.ID, v.Key, v.Token
}
func (c *uatCOSCredential) GetSecretId() string  { id, _, _ := c.GetCredential(); return id }
func (c *uatCOSCredential) GetSecretKey() string { _, key, _ := c.GetCredential(); return key }
func (c *uatCOSCredential) GetToken() string     { _, _, token := c.GetCredential(); return token }

func (c *uatCOSCredential) load() (temporaryCredential, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretLoadTimeout)
	defer cancel()
	c.mu.Lock()
	if c.cached.ID != "" && c.now().Add(cloudCredentialRefreshAhead).Unix() < c.cached.Expiry {
		v := c.cached
		c.mu.Unlock()
		return v, nil
	}
	if pending := c.inFlight; pending != nil {
		c.mu.Unlock()
		select {
		case <-pending.done:
			return pending.value, pending.err
		case <-ctx.Done():
			return temporaryCredential{}, errCloudCredentialUnavailable
		}
	}
	pending := &credentialExchange{done: make(chan struct{})}
	c.inFlight = pending
	c.mu.Unlock()
	v, err := c.exchange(ctx)
	c.mu.Lock()
	if err == nil {
		c.cached = v
	} else {
		c.cached = temporaryCredential{}
	}
	pending.value, pending.err = v, err
	c.inFlight = nil
	close(pending.done)
	c.mu.Unlock()
	return v, err
}

func (c *uatCOSCredential) exchange(ctx context.Context) (temporaryCredential, error) {
	request := sts.NewGetFederationTokenRequest()
	name, policy, duration := "OrderUATCOSRuntime", url.QueryEscape(uatCOSPolicy), uint64(1800)
	request.Name = &name
	request.Policy = &policy
	request.DurationSeconds = &duration
	response, err := c.client.GetFederationTokenWithContext(ctx, request)
	if err != nil || response == nil || response.Response == nil || response.Response.Credentials == nil || response.Response.ExpiredTime == nil || *response.Response.ExpiredTime > math.MaxInt64 {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	r := response.Response
	tuple := r.Credentials
	if tuple.TmpSecretId == nil || tuple.TmpSecretKey == nil || tuple.Token == nil {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	v := temporaryCredential{ID: *tuple.TmpSecretId, Key: *tuple.TmpSecretKey, Token: *tuple.Token, Expiry: int64(*r.ExpiredTime)}
	if !validCredentialComponent(v.ID) || !validCredentialComponent(v.Key) || !validCredentialComponent(v.Token) || v.Expiry <= c.now().Add(cloudCredentialRefreshAhead).Unix() {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	return v, nil
}

// The SDK parses the response body. Bound it before SDK decoding and reject
// redirects, so neither credentials nor signed requests follow another host.
type uatCOSSTSBoundedTransport struct{ next http.RoundTripper }

func (t uatCOSSTSBoundedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	response, err := next.RoundTrip(r)
	if err != nil || response == nil || response.Body == nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, errCloudCredentialUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode <= 399 {
		return nil, errCloudCredentialUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(data) > 64<<10 {
		return nil, errCloudCredentialUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	response.ContentLength = int64(len(data))
	return response, nil
}
