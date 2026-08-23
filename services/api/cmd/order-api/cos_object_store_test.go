package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type fakeCOSCredentialProvider struct {
	credential common.CredentialIface
	err        error
	calls      int
}

func (provider *fakeCOSCredentialProvider) GetCredential() (common.CredentialIface, error) {
	provider.calls++
	return provider.credential, provider.err
}

func TestComposeProductionObjectStoreUsesCVMRoleCredential(t *testing.T) {
	material, err := config.ParseCOSMaterial("order-images-1250000000", "ap-guangzhou", "https://images.example.com")
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeCOSCredentialProvider{credential: common.NewTokenCredential("test-id", "test-key", "test-token")}
	service, err := composeProductionObjectStoreWithProvider(context.Background(), material, provider)
	if err != nil {
		t.Fatal(err)
	}
	key := "images/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	publicURL, err := service.PublicURL(context.Background(), key)
	if err != nil || publicURL != "https://images.example.com/"+key || provider.calls != 1 {
		t.Fatalf("url=%q err=%v calls=%d", publicURL, err, provider.calls)
	}
}

func TestComposeProductionObjectStoreCollapsesCredentialFailure(t *testing.T) {
	material, err := config.ParseCOSMaterial("order-images-1250000000", "ap-guangzhou", "https://images.example.com")
	if err != nil {
		t.Fatal(err)
	}
	providerSecret := "metadata-secret-must-not-leak"
	provider := &fakeCOSCredentialProvider{err: errors.New(providerSecret)}
	service, err := composeProductionObjectStoreWithProvider(context.Background(), material, provider)
	if service != nil || !errors.Is(err, errProductionCOSUnavailable) || strings.Contains(err.Error(), providerSecret) {
		t.Fatalf("service=%v err=%v", service, err)
	}
}

func TestComposeProductionObjectStoreRejectsIncompleteCredential(t *testing.T) {
	material, err := config.ParseCOSMaterial("order-images-1250000000", "ap-guangzhou", "https://images.example.com")
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeCOSCredentialProvider{credential: common.NewTokenCredential("test-id", "", "test-token")}
	if service, err := composeProductionObjectStoreWithProvider(context.Background(), material, provider); service != nil || !errors.Is(err, errProductionCOSUnavailable) {
		t.Fatalf("service=%v err=%v", service, err)
	}
}
