package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

var errCloudCredentialUnavailable = errors.New("cloud credential unavailable")

// Cover the existing COS adapter's maximum presigned URL lifetime.
const cloudCredentialRefreshAhead = 15 * time.Minute

// NewCloudCredentialProvider selects the approved identity for each cloud
// environment. Business material uses its separate environment-specific source.
func NewCloudCredentialProvider(environment Environment, region string) (common.Provider, error) {
	switch environment {
	case Production:
		return common.DefaultCvmRoleProvider(), nil
	case UAT:
		return newCloudBaseCredentialProvider(os.Getenv("CLOUDBASE_ENV_ID"), os.Getenv("CLOUDBASE_APIKEY"), region, &http.Client{}, time.Now)
	default:
		return nil, errCloudCredentialUnavailable
	}
}

type cloudBaseProvider struct {
	credential *cloudBaseCredential
}

type temporaryCredential struct {
	ID     string `json:"TmpSecretId"`
	Key    string `json:"TmpSecretKey"`
	Token  string `json:"Token"`
	Expiry int64  `json:"ExpiredTime"`
}

type cloudBaseCredential struct {
	endpoint string
	envID    string
	apiKey   string
	client   *http.Client
	now      func() time.Time
	mu       sync.Mutex
	cached   temporaryCredential
	inFlight *credentialExchange
}

type credentialExchange struct {
	done  chan struct{}
	value temporaryCredential
	err   error
}

func newCloudBaseCredentialProvider(envID, apiKey, region string, client *http.Client, now func() time.Time) (common.Provider, error) {
	if !validCloudBaseEnvironmentID(envID) || !validCredentialComponent(apiKey) || !validTencentRegion(region) || client == nil || now == nil {
		return nil, errCloudCredentialUnavailable
	}
	boundedClient := *client
	if boundedClient.Timeout <= 0 || boundedClient.Timeout > secretLoadTimeout {
		boundedClient.Timeout = secretLoadTimeout
	}
	boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return errCloudCredentialUnavailable }
	return &cloudBaseProvider{credential: &cloudBaseCredential{
		endpoint: "https://" + envID + "." + region + ".tcb-api.tencentcloudapi.com/capi/credential",
		envID:    envID, apiKey: apiKey, client: &boundedClient, now: now,
	}}, nil
}

func validCloudBaseEnvironmentID(value string) bool {
	if len(value) > 63 || !strings.HasPrefix(value, "order-uat-") || value[len(value)-1] == '-' {
		return false
	}
	for index := range len(value) {
		if (value[index] < 'a' || value[index] > 'z') && (value[index] < '0' || value[index] > '9') && value[index] != '-' {
			return false
		}
	}
	return true
}

func validCredentialComponent(value string) bool {
	if len(value) == 0 || len(value) > 16<<10 {
		return false
	}
	for index := range len(value) {
		if value[index] < '!' || value[index] > '~' {
			return false
		}
	}
	return true
}

func (provider *cloudBaseProvider) GetCredential() (common.CredentialIface, error) {
	if _, err := provider.credential.load(); err != nil {
		return nil, errCloudCredentialUnavailable
	}
	return provider.credential, nil
}

func (credential *cloudBaseCredential) GetCredential() (string, string, string) {
	value, err := credential.load()
	if err != nil {
		return "", "", ""
	}
	return value.ID, value.Key, value.Token
}

func (credential *cloudBaseCredential) GetSecretId() string {
	id, _, _ := credential.GetCredential()
	return id
}

func (credential *cloudBaseCredential) GetSecretKey() string {
	_, key, _ := credential.GetCredential()
	return key
}

func (credential *cloudBaseCredential) GetToken() string {
	_, _, token := credential.GetCredential()
	return token
}

func (credential *cloudBaseCredential) load() (temporaryCredential, error) {
	ctx, cancel := context.WithTimeout(context.Background(), credential.client.Timeout)
	defer cancel()
	credential.mu.Lock()
	if credential.cached.ID != "" && credential.now().Add(cloudCredentialRefreshAhead).Unix() < credential.cached.Expiry {
		value := credential.cached
		credential.mu.Unlock()
		return value, nil
	}
	if pending := credential.inFlight; pending != nil {
		credential.mu.Unlock()
		select {
		case <-pending.done:
			return pending.value, pending.err
		case <-ctx.Done():
			return temporaryCredential{}, errCloudCredentialUnavailable
		}
	}
	pending := &credentialExchange{done: make(chan struct{})}
	credential.inFlight = pending
	credential.mu.Unlock()
	value, err := credential.exchange(ctx)
	credential.mu.Lock()
	if err == nil {
		credential.cached = value
	} else {
		credential.cached = temporaryCredential{}
	}
	pending.value, pending.err = value, err
	credential.inFlight = nil
	close(pending.done)
	credential.mu.Unlock()
	return value, err
}

func (credential *cloudBaseCredential) exchange(ctx context.Context) (temporaryCredential, error) {
	body, _ := json.Marshal(struct {
		Env string `json:"env"`
	}{Env: credential.envID})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, credential.endpoint, bytes.NewReader(body))
	if err != nil {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+credential.apiKey)
	response, err := credential.client.Do(request)
	if err != nil {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	var result struct {
		Code *int                `json:"code"`
		Data temporaryCredential `json:"data"`
	}
	if response.StatusCode != http.StatusOK || err != nil || len(data) > 64<<10 || json.Unmarshal(data, &result) != nil ||
		result.Code == nil || *result.Code != 0 || !validCredentialComponent(result.Data.ID) || !validCredentialComponent(result.Data.Key) || !validCredentialComponent(result.Data.Token) ||
		result.Data.Expiry <= credential.now().Add(cloudCredentialRefreshAhead).Unix() {
		return temporaryCredential{}, errCloudCredentialUnavailable
	}
	return result.Data, nil
}
