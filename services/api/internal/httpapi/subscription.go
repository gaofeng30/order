package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gin-gonic/gin"
)

const maxConsentRequestBytes = 4096

// SubscriptionAuthenticator resolves the current Mini Program bearer to its user.
type SubscriptionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

// ConsentRecorder persists one user-owned subscription decision.
type ConsentRecorder interface {
	RecordConsent(context.Context, subscription.WriteMeta, subscription.ConsentInput) (subscription.Subscription, error)
}

// SubscriptionHandler owns the frozen subscription HTTP entrypoint.
type SubscriptionHandler struct {
	authenticator         SubscriptionAuthenticator
	recorder              ConsentRecorder
	templateConfigVersion uint64
}

// NewSubscriptionHandler constructs the authenticated subscription route.
func NewSubscriptionHandler(authenticator SubscriptionAuthenticator, recorder ConsentRecorder, templateConfigVersion uint64) *SubscriptionHandler {
	return &SubscriptionHandler{authenticator: authenticator, recorder: recorder, templateConfigVersion: templateConfigVersion}
}

// RegisterRoutes mounts only POST /api/v1/orders/:id/subscriptions.
func (handler *SubscriptionHandler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/orders/:id/subscriptions", handler.record)
}

type consentRequest struct {
	Kind     subscription.Kind     `json:"kind"`
	Decision subscription.Decision `json:"decision"`
}

type consentResponse struct {
	Subscription consentView `json:"subscription"`
}

type consentView struct {
	Kind      subscription.Kind     `json:"kind"`
	Decision  subscription.Decision `json:"decision"`
	Available bool                  `json:"available"`
}

type subscriptionErrorEnvelope struct {
	Error subscriptionErrorResponse `json:"error"`
}

type subscriptionErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *SubscriptionHandler) record(ctx *gin.Context) {
	if handler == nil || handler.authenticator == nil || handler.recorder == nil || handler.templateConfigVersion == 0 {
		writeSubscriptionError(ctx, http.StatusServiceUnavailable, "SUBSCRIPTION_UNAVAILABLE", "subscription temporarily unavailable")
		return
	}
	orderID, validID := httpdto.ParseID(ctx.Param("id"))
	if !validID || ctx.ContentType() != "application/json" {
		writeSubscriptionError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	token, err := httpdto.BearerToken(ctx.Request)
	if err != nil {
		writeSubscriptionError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) || (err == nil && userID == 0) {
		writeSubscriptionError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	if err != nil {
		writeSubscriptionError(ctx, http.StatusServiceUnavailable, "SUBSCRIPTION_UNAVAILABLE", "subscription temporarily unavailable")
		return
	}
	idempotencyKey, err := httpdto.IdempotencyKey(ctx.Request)
	if err != nil {
		writeSubscriptionError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	var request consentRequest
	if err := httpdto.DecodeStrict(ctx.Request.Body, maxConsentRequestBytes, &request); err != nil || !validConsentRequest(request) {
		writeSubscriptionError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	requestID := ctx.GetString(requestIDKey)
	if requestID == "" {
		writeSubscriptionError(ctx, http.StatusServiceUnavailable, "SUBSCRIPTION_UNAVAILABLE", "subscription temporarily unavailable")
		return
	}
	input := subscription.ConsentInput{OrderID: orderID, Kind: request.Kind, Decision: request.Decision, TemplateConfigVersion: handler.templateConfigVersion}
	result, err := handler.recorder.RecordConsent(ctx.Request.Context(), subscription.WriteMeta{ActorUserID: userID, IdempotencyKey: idempotencyKey, RequestID: requestID}, input)
	if err != nil {
		writeConsentDomainError(ctx, err)
		return
	}
	if result.OrderID != orderID || result.Kind != request.Kind || result.Decision != request.Decision || result.Available != (result.Decision == subscription.DecisionAccepted) {
		writeSubscriptionError(ctx, http.StatusServiceUnavailable, "SUBSCRIPTION_UNAVAILABLE", "subscription temporarily unavailable")
		return
	}
	ctx.JSON(http.StatusOK, consentResponse{Subscription: consentView{Kind: result.Kind, Decision: result.Decision, Available: result.Available}})
}

func validConsentRequest(request consentRequest) bool {
	validKind := request.Kind == subscription.KindReady || request.Kind == subscription.KindRefundResult
	validDecision := request.Decision == subscription.DecisionAccepted || request.Decision == subscription.DecisionRejected
	return validKind && validDecision
}

func writeConsentDomainError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, subscription.ErrInvalidInput):
		writeSubscriptionError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	case errors.Is(err, subscription.ErrUnauthenticated):
		writeSubscriptionError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
	case errors.Is(err, subscription.ErrForbidden):
		writeSubscriptionError(ctx, http.StatusForbidden, "FORBIDDEN", "operation forbidden")
	case errors.Is(err, subscription.ErrNotFound):
		writeSubscriptionError(ctx, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, subscription.ErrIdempotencyConflict):
		writeSubscriptionError(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict")
	default:
		writeSubscriptionError(ctx, http.StatusServiceUnavailable, "SUBSCRIPTION_UNAVAILABLE", "subscription temporarily unavailable")
	}
}

func writeSubscriptionError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, subscriptionErrorEnvelope{Error: subscriptionErrorResponse{Code: code, Message: message}})
}
