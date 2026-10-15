package config

import (
	"context"
	"errors"
	"sync"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssm/v20190923"
)

const (
	ssmEndpoint       = "ssm.tencentcloudapi.com"
	ssmCurrentVersion = "SSM_Current"
)

var errProductionSecretUnavailable = errors.New("production secret unavailable")

type credentialProvider interface {
	GetCredential() (common.CredentialIface, error)
}

type ssmSecretClient interface {
	GetSecretValueWithContext(context.Context, *ssm.GetSecretValueRequest) (*ssm.GetSecretValueResponse, error)
}

type ssmClientFactory func(common.CredentialIface, string) (ssmSecretClient, error)

type tencentSSMSecretSource struct {
	region   string
	provider credentialProvider
	factory  ssmClientFactory
	once     sync.Once
	client   ssmSecretClient
	err      error
}

func newCloudSecretSource(environment Environment, region string) SecretSource {
	if environment == UAT || environment == Production {
		return cloudRunSecretSource{}
	}
	provider, _ := NewCloudCredentialProvider(environment, region)
	return newSSMSecretSource(region, provider, defaultSSMClientFactory)
}

func newSSMSecretSource(region string, provider credentialProvider, factory ssmClientFactory) *tencentSSMSecretSource {
	return &tencentSSMSecretSource{region: region, provider: provider, factory: factory}
}

func (source *tencentSSMSecretSource) Get(ctx context.Context, name string) (string, error) {
	if !allowedCloudSecretName(name) {
		return "", errProductionSecretUnavailable
	}
	source.once.Do(func() {
		credential, err := credentialWithContext(ctx, source.provider)
		if err != nil {
			source.err = errProductionSecretUnavailable
			return
		}
		source.client, err = source.factory(credential, source.region)
		if err != nil {
			source.err = errProductionSecretUnavailable
		}
	})
	if source.err != nil || source.client == nil {
		return "", errProductionSecretUnavailable
	}

	request := ssm.NewGetSecretValueRequest()
	request.SecretName = common.StringPtr(name)
	request.VersionId = common.StringPtr(ssmCurrentVersion)
	response, err := source.client.GetSecretValueWithContext(ctx, request)
	if err != nil || response == nil || response.Response == nil || response.Response.SecretString == nil || response.Response.SecretBinary != nil {
		return "", errProductionSecretUnavailable
	}
	return *response.Response.SecretString, nil
}

func allowedCloudSecretName(name string) bool {
	for _, environment := range []Environment{UAT, Production} {
		for _, purpose := range []string{databasePasswordPurpose, miniProgramPurpose, weChatPaySecretPurpose, redemptionTokenPurpose} {
			allowed, _ := environment.SecretName(purpose)
			if name == allowed {
				return true
			}
		}
	}
	return false
}

func credentialWithContext(ctx context.Context, provider credentialProvider) (common.CredentialIface, error) {
	if ctx == nil || provider == nil {
		return nil, errProductionSecretUnavailable
	}
	type result struct {
		credential common.CredentialIface
		err        error
	}
	resultChannel := make(chan result, 1)
	go func() {
		credential, err := provider.GetCredential()
		resultChannel <- result{credential: credential, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, errProductionSecretUnavailable
	case value := <-resultChannel:
		if value.err != nil || value.credential == nil {
			return nil, errProductionSecretUnavailable
		}
		return value.credential, nil
	}
}

func defaultSSMClientFactory(credential common.CredentialIface, region string) (ssmSecretClient, error) {
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = ssmEndpoint
	clientProfile.HttpProfile.ReqMethod = "POST"
	clientProfile.HttpProfile.ReqTimeout = int(secretLoadTimeout.Seconds())
	clientProfile.DisableRegionBreaker = true
	return ssm.NewClient(credential, region, clientProfile)
}
