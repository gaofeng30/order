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

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

const (
	maxLoginBodyBytes = 1024
	maxPhoneCodeBytes = 256
)

// SessionAuthenticator resolves an existing strict Mini Program bearer.
type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

// Application exposes the two merchant identity operations used by HTTP.
type Application interface {
	Identity(context.Context, uint64) (Identity, error)
	Login(context.Context, uint64, string, string) (Identity, error)
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
	engine.POST("/api/v1/me/merchant-login", handler.merchantLogin)
}

type identityResponse struct {
	User     identityUserResponse      `json:"user"`
	Merchant *identityMerchantResponse `json:"merchant"`
}

type identityUserResponse struct {
	PrimaryPhoneBound bool `json:"primary_phone_bound"`
}

type identityMerchantResponse struct {
	Role        Role   `json:"role"`
	AuthVersion uint64 `json:"auth_version"`
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
	response, valid := publicIdentity(projection)
	if !valid || response.Merchant == nil {
		writeUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, response)
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
	token, ok := exactBearer(ctx.Request.Header.Values("Authorization"))
	if !ok {
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
	response := identityResponse{User: identityUserResponse{PrimaryPhoneBound: projection.PrimaryPhoneBound}}
	if projection.Merchant == nil {
		return response, true
	}
	if (projection.Merchant.Role != RoleOwner && projection.Merchant.Role != RoleSubaccount) || projection.Merchant.AuthVersion == 0 {
		return identityResponse{}, false
	}
	response.Merchant = &identityMerchantResponse{
		Role: projection.Merchant.Role, AuthVersion: projection.Merchant.AuthVersion,
	}
	return response, true
}

func writeUnavailable(ctx *gin.Context) {
	writeError(ctx, http.StatusServiceUnavailable, "MERCHANT_IDENTITY_UNAVAILABLE", "merchant identity temporarily unavailable")
}

func writeError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorResponse{Code: code, Message: message}})
}
