package main

import (
	"context"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/objectstore"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

const cosCredentialLoadTimeout = 5 * time.Second

var errProductionCOSUnavailable = errors.New("production cos object store unavailable")

type cosCredentialProvider interface {
	GetCredential() (common.CredentialIface, error)
}

func composeProductionObjectStore(ctx context.Context, material config.COSMaterial) (*objectstore.Service, error) {
	return composeProductionObjectStoreWithProvider(ctx, material, common.DefaultCvmRoleProvider())
}

func composeProductionObjectStoreWithProvider(ctx context.Context, material config.COSMaterial, provider cosCredentialProvider) (*objectstore.Service, error) {
	if ctx == nil || provider == nil {
		return nil, errProductionCOSUnavailable
	}
	loadContext, cancel := context.WithTimeout(ctx, cosCredentialLoadTimeout)
	defer cancel()
	credential, err := loadCOSCredential(loadContext, provider)
	if err != nil || credential == nil {
		return nil, errProductionCOSUnavailable
	}
	secretID, secretKey, token := credential.GetCredential()
	if secretID == "" || secretKey == "" || token == "" {
		return nil, errProductionCOSUnavailable
	}
	adapter, err := objectstore.NewCOSAdapter(objectstore.COSConfig{
		Bucket: material.Bucket(), Region: material.Region(), PublicOrigin: material.PublicOrigin(),
	}, credential)
	if err != nil {
		return nil, errProductionCOSUnavailable
	}
	return objectstore.NewService(adapter), nil
}

func loadCOSCredential(ctx context.Context, provider cosCredentialProvider) (common.CredentialIface, error) {
	type result struct {
		credential common.CredentialIface
		err        error
	}
	results := make(chan result, 1)
	go func() {
		credential, err := provider.GetCredential()
		results <- result{credential: credential, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, errProductionCOSUnavailable
	case result := <-results:
		return result.credential, result.err
	}
}
