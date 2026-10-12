package merchantidentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

const (
	maxLoginBodyBytes = 1024
	maxPhoneCodeBytes = 256
	maxExtraBodyBytes = 2048
)

// SessionAuthenticator resolves an existing strict Mini Program bearer.
type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

// Application exposes the two merchant identity operations used by HTTP.
type Application interface {
	Identity(context.Context, uint64) (Identity, error)
	Login(context.Context, uint64, string, string) (Identity, error)
	SetExtraPhone(context.Context, WriteMeta, ExtraPhoneCommand) (ExtraPhoneResult, error)
}

// Handler serves only merchant identity endpoints.
type Handler struct {
	authenticator SessionAuthenticator
	application   Application
}

// NewHandler constructs the protected merchant identity handler.
func NewHandler(authenticator SessionAuthenticator, application Application) *Handler {
	return &Handler{authenticator: authenticator, application: application}
}

// RegisterRoutes adds only the two versioned merchant identity routes.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/me/identity", handler.getIdentity)
	engine.POST("/api/v1/me/extra-phone", handler.setExtraPhone)
	engine.POST("/api/v1/me/merchant-login", handler.merchantLogin)
}

type identityResponse struct {
	Identity identityProjectionResponse `json:"identity"`
}

type identityProjectionResponse struct {
	PrimaryPhone    primaryPhoneResponse    `json:"primary_phone"`
	ExtraPhone      extraPhoneResponse      `json:"extra_phone"`
	PricingIdentity pricingIdentityResponse `json:"pricing_identity"`
	Merchant        merchantResponse        `json:"merchant"`
}

type primaryPhoneResponse struct {
	Bound       bool   `json:"bound"`
	MaskedPhone string `json:"masked_phone,omitempty"`
}

type extraPhoneResponse struct {
	Set         bool   `json:"set"`
	MaskedPhone string `json:"masked_phone,omitempty"`
	Name        string `json:"name,omitempty"`
}

type pricingIdentityResponse struct {
	Kind        PricingKind `json:"kind"`
	RatePercent uint8       `json:"rate_percent"`
}

type merchantResponse struct {
	Bound bool `json:"bound"`
	Role  Role `json:"role,omitempty"`
}

type merchantLoginResponse struct {
	Merchant merchantResponse `json:"merchant"`
}

type extraPhoneRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

type extraPhoneResultResponse struct {
	ExtraPhone      extraPhoneResponse      `json:"extra_phone"`
	PricingIdentity pricingIdentityResponse `json:"pricing_identity"`
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) getIdentity(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 1))
	if err != nil || len(body) != 0 {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	projection, err := handler.application.Identity(ctx.Request.Context(), userID)
	if err != nil {
		writeUnavailable(ctx)
		return
	}
	response, ok := publicIdentity(projection)
	if !ok {
		writeUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) merchantLogin(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxLoginBodyBytes+1))
	if err != nil || len(body) > maxLoginBodyBytes || !utf8.Valid(body) {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	code, ok := decodeLoginRequest(body)
	if !ok || strings.TrimSpace(code) == "" || len(code) > maxPhoneCodeBytes {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	projection, err := handler.application.Login(ctx.Request.Context(), userID, code, ctx.GetString("request_id"))
	switch {
	case errors.Is(err, ErrMerchantAccountNotAvailable):
		writeError(ctx, http.StatusForbidden, "MERCHANT_ACCOUNT_NOT_AVAILABLE", "merchant account not available")
		return
	case errors.Is(err, ErrPhoneInUse):
		writeError(ctx, http.StatusConflict, "PHONE_IN_USE", "phone already in use")
		return
	case errors.Is(err, ErrPrimaryPhoneMismatch):
		writeError(ctx, http.StatusConflict, "PRIMARY_PHONE_MISMATCH", "primary phone mismatch")
		return
	case errors.Is(err, ErrPhoneCodeRejected):
		writeError(ctx, http.StatusUnprocessableEntity, "PHONE_CODE_REJECTED", "phone code rejected")
		return
	case err != nil:
		writeUnavailable(ctx)
		return
	}
	if projection.Merchant == nil || (projection.Merchant.Role != RoleOwner && projection.Merchant.Role != RoleSubaccount) || projection.Merchant.AuthVersion == 0 {
		writeUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, merchantLoginResponse{Merchant: merchantResponse{Bound: true, Role: projection.Merchant.Role}})
}

func (handler *Handler) setExtraPhone(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	var request extraPhoneRequest
	if err := httpdto.DecodeStrict(ctx.Request.Body, maxExtraBodyBytes, &request); err != nil {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	idempotencyKey, err := httpdto.IdempotencyKey(ctx.Request)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "invalid idempotency key")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	result, err := handler.application.SetExtraPhone(ctx.Request.Context(), WriteMeta{
		ActorUserID: userID, IdempotencyKey: idempotencyKey, RequestID: ctx.GetString("request_id"),
	}, ExtraPhoneCommand{Phone: request.Phone, Name: request.Name})
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	case errors.Is(err, ErrPrimaryPhoneRequired):
		writeError(ctx, http.StatusConflict, "PRIMARY_PHONE_REQUIRED", "primary phone required")
		return
	case errors.Is(err, ErrIdempotencyConflict):
		writeError(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency key conflicts with another request")
		return
	case err != nil:
		writeUnavailable(ctx)
		return
	}
	if !validPricing(result.Pricing.Kind, result.Pricing.RatePercent) || result.ExtraPhone.MaskedPhone == "" || result.ExtraPhone.Name == "" {
		writeUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, extraPhoneResultResponse{
		ExtraPhone:      extraPhoneResponse{Set: true, MaskedPhone: result.ExtraPhone.MaskedPhone, Name: result.ExtraPhone.Name},
		PricingIdentity: pricingIdentityResponse{Kind: result.Pricing.Kind, RatePercent: result.Pricing.RatePercent},
	})
}

func decodeLoginRequest(body []byte) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return "", false
	}
	found := false
	code := ""
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, keyOK := keyToken.(string)
		if err != nil || !keyOK || key != "code" || found {
			return "", false
		}
		if err := decoder.Decode(&code); err != nil {
			return "", false
		}
		found = true
	}
	last, err := decoder.Token()
	if err != nil || last != json.Delim('}') {
		return "", false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", false
	}
	return code, found
}

func (handler *Handler) authenticate(ctx *gin.Context) (uint64, bool) {
	token, err := httpdto.BearerToken(ctx.Request)
	if err != nil {
		writeError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return 0, false
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) || (err == nil && userID == 0) {
		writeError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return 0, false
	}
	if err != nil {
		writeUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func exactBearer(values []string) (string, bool) {
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", false
	}
	return token, true
}

func publicIdentity(projection Identity) (identityResponse, bool) {
	if projection.PrimaryPhoneBound != (projection.PrimaryPhoneMasked != "") || (!projection.PrimaryPhoneBound && projection.ExtraPhone != nil) || !validPricing(projection.Pricing.Kind, projection.Pricing.RatePercent) {
		return identityResponse{}, false
	}
	response := identityResponse{Identity: identityProjectionResponse{
		PrimaryPhone:    primaryPhoneResponse{Bound: projection.PrimaryPhoneBound, MaskedPhone: projection.PrimaryPhoneMasked},
		ExtraPhone:      extraPhoneResponse{Set: projection.ExtraPhone != nil},
		PricingIdentity: pricingIdentityResponse{Kind: projection.Pricing.Kind, RatePercent: projection.Pricing.RatePercent},
		Merchant:        merchantResponse{Bound: projection.Merchant != nil},
	}}
	if projection.ExtraPhone != nil {
		if projection.ExtraPhone.MaskedPhone == "" || projection.ExtraPhone.Name == "" {
			return identityResponse{}, false
		}
		response.Identity.ExtraPhone.MaskedPhone = projection.ExtraPhone.MaskedPhone
		response.Identity.ExtraPhone.Name = projection.ExtraPhone.Name
	}
	if projection.Merchant != nil {
		if (projection.Merchant.Role != RoleOwner && projection.Merchant.Role != RoleSubaccount) || projection.Merchant.AuthVersion == 0 || !projection.PrimaryPhoneBound {
			return identityResponse{}, false
		}
		response.Identity.Merchant.Role = projection.Merchant.Role
	}
	return response, true
}

func writeUnavailable(ctx *gin.Context) {
	writeError(ctx, http.StatusServiceUnavailable, "MERCHANT_IDENTITY_UNAVAILABLE", "merchant identity temporarily unavailable")
}

func writeError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorResponse{Code: code, Message: message}})
}
