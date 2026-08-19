package identity

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SessionAuthenticator is the integrated hash-only Bearer lookup.
type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

// PhoneBinder is the application operation exposed by the phone route.
type PhoneBinder interface {
	Bind(context.Context, uint64, string) (PhoneBinding, error)
}

// PhoneHandler serves only route-specific primary-phone binding.
type PhoneHandler struct {
	authenticator SessionAuthenticator
	binder        PhoneBinder
}

// NewPhoneHandler constructs the protected phone-binding handler.
func NewPhoneHandler(authenticator SessionAuthenticator, binder PhoneBinder) *PhoneHandler {
	return &PhoneHandler{authenticator: authenticator, binder: binder}
}

// RegisterRoutes adds only the versioned primary-phone route.
func (handler *PhoneHandler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/me/bind-phone", handler.bind)
}

type phoneBindingResponse struct {
	PrimaryPhoneBound bool   `json:"primary_phone_bound"`
	MaskedPhone       string `json:"masked_phone"`
}

func (handler *PhoneHandler) bind(ctx *gin.Context) {
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeInvalidRequest(ctx)
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxSessionRequestBytes+1))
	if err != nil || len(body) > maxSessionRequestBytes {
		writeInvalidRequest(ctx)
		return
	}
	code, ok := decodeSessionRequest(body)
	if !ok || len(code) > maxLoginCodeBytes || strings.TrimSpace(code) == "" {
		writeInvalidRequest(ctx)
		return
	}
	token, ok := exactBearer(ctx.Request.Header.Values("Authorization"))
	if !ok {
		writePhoneError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, ErrUnauthenticated) || (err == nil && userID == 0) {
		writePhoneError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	if err != nil {
		writePhoneUnavailable(ctx)
		return
	}
	binding, err := handler.binder.Bind(ctx.Request.Context(), userID, code)
	switch {
	case errors.Is(err, ErrPhoneCodeRejected):
		writePhoneError(ctx, http.StatusUnprocessableEntity, "PHONE_CODE_REJECTED", "phone code rejected")
	case errors.Is(err, ErrPhoneInUse):
		writePhoneError(ctx, http.StatusConflict, "PHONE_IN_USE", "phone already in use")
	case errors.Is(err, ErrPrimaryPhoneAlreadyBound):
		writePhoneError(ctx, http.StatusConflict, "PRIMARY_PHONE_ALREADY_BOUND", "primary phone already bound")
	case err != nil || binding.MaskedPhone == "":
		writePhoneUnavailable(ctx)
	default:
		ctx.JSON(http.StatusOK, phoneBindingResponse{PrimaryPhoneBound: true, MaskedPhone: binding.MaskedPhone})
	}
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

func writePhoneUnavailable(ctx *gin.Context) {
	writePhoneError(ctx, http.StatusServiceUnavailable, "PHONE_BINDING_UNAVAILABLE", "phone binding temporarily unavailable")
}

func writePhoneError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, sessionErrorEnvelope{Error: sessionErrorResponse{Code: code, Message: message}})
}
