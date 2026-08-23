package merchantidentity_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gin-gonic/gin"
)

func TestFrozenIdentityAndExtraPhoneHTTPContract(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	application := &frozenApplicationStub{identity: merchantidentity.Identity{
		PrimaryPhoneBound:  true,
		PrimaryPhoneMasked: "138****0001",
		ExtraPhone: &merchantidentity.ExtraPhoneProjection{
			MaskedPhone: "139****0002", Name: "张 三",
		},
		Pricing:  merchantidentity.PricingProjection{Kind: merchantidentity.PricingStaff, RatePercent: 80},
		Merchant: &merchantidentity.MerchantProjection{Role: merchantidentity.RoleOwner, AuthVersion: 7},
	}}
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) { ctx.Set("request_id", "request-profile-1"); ctx.Next() })
	merchantidentity.NewHandler(&authenticatorStub{userID: 41}, application).RegisterRoutes(engine)

	identityRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me/identity", nil)
	identityRequest.Header.Set("Authorization", "Bearer opaque")
	identityResponse := httptest.NewRecorder()
	engine.ServeHTTP(identityResponse, identityRequest)
	assertMerchantHTTP(t, identityResponse, http.StatusOK, `{"identity":{"primary_phone":{"bound":true,"masked_phone":"138****0001"},"extra_phone":{"set":true,"masked_phone":"139****0002","name":"张 三"},"pricing_identity":{"kind":"STAFF","rate_percent":80},"merchant":{"bound":true,"role":"OWNER"}}}`)

	extraRequest := httptest.NewRequest(http.MethodPost, "/api/v1/me/extra-phone", bytes.NewBufferString(`{"phone":"13900000002","name":"张 三"}`))
	extraRequest.Header.Set("Content-Type", "application/json")
	extraRequest.Header.Set("Authorization", "Bearer opaque")
	extraRequest.Header.Set("Idempotency-Key", "extra-phone-1")
	extraResponse := httptest.NewRecorder()
	engine.ServeHTTP(extraResponse, extraRequest)
	assertMerchantHTTP(t, extraResponse, http.StatusOK, `{"extra_phone":{"set":true,"masked_phone":"139****0002","name":"张 三"},"pricing_identity":{"kind":"STAFF","rate_percent":80}}`)
	if application.extraMeta.ActorUserID != 41 || application.extraMeta.IdempotencyKey != "extra-phone-1" || application.extraMeta.RequestID != "request-profile-1" || application.extraCommand.Phone != "13900000002" || application.extraCommand.Name != "张 三" {
		t.Fatalf("extra phone command = %#v / %#v", application.extraMeta, application.extraCommand)
	}
}

func TestFrozenExtraPhoneRejectsMissingIdempotencyBeforeApplication(t *testing.T) {
	application := &frozenApplicationStub{}
	engine := merchantHandlerEngine(&authenticatorStub{userID: 41}, application)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/extra-phone", bytes.NewBufferString(`{"phone":"13900000002","name":"张三"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	assertMerchantHTTP(t, response, http.StatusBadRequest, `{"error":{"code":"INVALID_IDEMPOTENCY_KEY","message":"invalid idempotency key"}}`)
	if application.extraCalls != 0 {
		t.Fatal("invalid idempotency key reached application")
	}
}

type frozenApplicationStub struct {
	identity     merchantidentity.Identity
	err          error
	extraMeta    merchantidentity.WriteMeta
	extraCommand merchantidentity.ExtraPhoneCommand
	extraCalls   int
}

func (stub *frozenApplicationStub) Identity(context.Context, uint64) (merchantidentity.Identity, error) {
	return stub.identity, stub.err
}

func (stub *frozenApplicationStub) Login(context.Context, uint64, string, string) (merchantidentity.Identity, error) {
	return stub.identity, stub.err
}

func (stub *frozenApplicationStub) SetExtraPhone(_ context.Context, meta merchantidentity.WriteMeta, command merchantidentity.ExtraPhoneCommand) (merchantidentity.ExtraPhoneResult, error) {
	stub.extraCalls++
	stub.extraMeta = meta
	stub.extraCommand = command
	return merchantidentity.ExtraPhoneResult{ExtraPhone: *stub.identity.ExtraPhone, Pricing: stub.identity.Pricing}, stub.err
}
