package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxSessionRequestBytes = 1024
	maxLoginCodeBytes      = 256
)

// SessionIssuer is the only application operation exposed by the HTTP handler.
type SessionIssuer interface {
	Issue(context.Context, string) (IssuedSession, error)
}

// Handler serves only Mini Program session creation.
type Handler struct {
	issuer SessionIssuer
}

// NewHandler constructs the session HTTP handler.
func NewHandler(issuer SessionIssuer) *Handler {
	return &Handler{issuer: issuer}
}

// RegisterRoutes adds only the versioned session-creation route.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/auth/miniprogram/session", handler.create)
}

type sessionResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at"`
}

type sessionErrorEnvelope struct {
	Error sessionErrorResponse `json:"error"`
}

type sessionErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) create(ctx *gin.Context) {
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
	issued, err := handler.issuer.Issue(ctx.Request.Context(), code)
	if errors.Is(err, ErrLoginRejected) {
		ctx.JSON(http.StatusUnauthorized, sessionErrorEnvelope{Error: sessionErrorResponse{
			Code: "MINIPROGRAM_LOGIN_REJECTED", Message: "miniprogram login rejected",
		}})
		return
	}
	if err != nil || issued.AccessToken == "" || issued.ExpiresAt.IsZero() {
		writeSessionUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusCreated, sessionResponse{
		AccessToken: issued.AccessToken,
		TokenType:   "Bearer",
		ExpiresAt:   issued.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
}

func decodeSessionRequest(body []byte) (string, bool) {
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

func writeInvalidRequest(ctx *gin.Context) {
	ctx.JSON(http.StatusBadRequest, sessionErrorEnvelope{Error: sessionErrorResponse{
		Code: "INVALID_REQUEST", Message: "invalid request",
	}})
}

func writeSessionUnavailable(ctx *gin.Context) {
	ctx.JSON(http.StatusServiceUnavailable, sessionErrorEnvelope{Error: sessionErrorResponse{
		Code: "SESSION_UNAVAILABLE", Message: "session temporarily unavailable",
	}})
}
