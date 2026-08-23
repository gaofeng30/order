package refund

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/httpdto"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/gin-gonic/gin"
)

const (
	maxCancelBodyBytes             = 8 * 1024
	maxRefundNotificationBodyBytes = 64 * 1024
)

type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

type UserOrderReader interface {
	GetUser(context.Context, uint64, uint64) (orderquery.Detail, error)
}

type Handler struct {
	authenticator SessionAuthenticator
	application   Application
	orders        UserOrderReader
	parser        NotificationParser
}

func NewHandler(authenticator SessionAuthenticator, application Application, orders UserOrderReader, parser NotificationParser) *Handler {
	return &Handler{authenticator: authenticator, application: application, orders: orders, parser: parser}
}

// RegisterRoutes mounts refund-owned routes on the already versioned API group.
func (handler *Handler) RegisterRoutes(api *gin.RouterGroup) {
	if handler == nil || api == nil {
		return
	}
	api.POST("/orders/:id/cancel", handler.cancelOrder)
	api.POST("/refunds/wechat/notify", handler.refundNotification)
}

func (handler *Handler) cancelOrder(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	orderID, ok := httpdto.ParseID(ctx.Param("id"))
	if !ok || ctx.Request.URL.RawQuery != "" {
		writeRefundHTTPError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND")
		return
	}
	meta, ok := handler.writeMeta(ctx)
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if !isRefundJSONRequest(ctx.Request) || httpdto.DecodeStrict(ctx.Request.Body, maxCancelBodyBytes, &request) != nil || !validHTTPReason(request.Reason) {
		writeRefundHTTPError(ctx, http.StatusBadRequest, "INVALID_INPUT")
		return
	}
	before, err := handler.orders.GetUser(ctx.Request.Context(), meta.ActorUserID, orderID)
	if err != nil {
		writeUserOrderReadError(ctx, err)
		return
	}
	if !validCanonicalUserOrder(before, orderID, false) {
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return
	}
	result, err := handler.application.RequestOrder(ctx.Request.Context(), meta, orderID, request.Reason)
	if err != nil {
		writeRefundApplicationError(ctx, err)
		return
	}
	detail, err := handler.orders.GetUser(ctx.Request.Context(), meta.ActorUserID, orderID)
	if err != nil {
		writeUserOrderReadError(ctx, err)
		return
	}
	if !validCanonicalUserOrder(detail, orderID, true) || !validRefundView(result, detail) {
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return
	}
	ctx.JSON(http.StatusOK, cancelResponse{Order: projectUserOrder(detail), Refund: projectRefund(result)})
}

func validCanonicalUserOrder(detail orderquery.Detail, orderID uint64, afterCancellation bool) bool {
	if detail.ID != orderID || detail.OrderNo == "" || !validOrderState(detail.State) || !validDateTime(detail.PickupDate, detail.PickupTime) ||
		detail.PickupPoint == "" || detail.PickupNumber == 0 || detail.PickupNumber > 9999 || detail.PayableCents == 0 || detail.MaterializedAt.IsZero() ||
		detail.AvailableActions == nil || detail.Contact.Name == "" || detail.Contact.MaskedPhone == "" ||
		(detail.Identity.Kind != "STAFF" && detail.Identity.Kind != "VISITOR") || detail.Discount.RatePercent < 1 || detail.Discount.RatePercent > 100 ||
		detail.Items == nil || detail.TransactionID == "" || detail.PaidAt.IsZero() || detail.NotificationOptions == nil {
		return false
	}
	if afterCancellation && detail.State != orderquery.StateRefunding && detail.State != orderquery.StateRefunded {
		return false
	}
	if (detail.State == orderquery.StateRefunding || detail.State == orderquery.StateRefunded) && (detail.RedemptionToken != "" || detail.TransitionTimes.RefundingAt.IsZero()) {
		return false
	}
	for _, item := range detail.Items {
		if item.ProductID == 0 || item.Name == "" || item.Quantity == 0 || item.Flavors == nil {
			return false
		}
	}
	return true
}

func validRefundView(value Refund, detail orderquery.Detail) bool {
	if value.ID == 0 || value.OrderID != detail.ID || value.AmountCents != detail.PayableCents || value.Currency != "CNY" || value.RequestedAt.IsZero() {
		return false
	}
	switch value.State {
	case ProviderReady, ProviderCreateClaimed, ProviderCreateUnknown, ProviderProcessing, ProviderSuccess, ProviderClosed:
	default:
		return false
	}
	switch value.MaterializationState {
	case MaterializationAwaitingProvider, MaterializationApplied, MaterializationPendingManual:
		return true
	default:
		return false
	}
}

func validOrderState(state orderquery.State) bool {
	switch state {
	case orderquery.StateReserved, orderquery.StatePreparing, orderquery.StateReadyForPickup, orderquery.StateCompleted, orderquery.StateRefunding, orderquery.StateRefunded:
		return true
	default:
		return false
	}
}

func validDateTime(date, clock string) bool {
	parsedDate, dateErr := time.Parse("2006-01-02", date)
	parsedClock, clockErr := time.Parse("15:04", clock)
	return dateErr == nil && parsedDate.Format("2006-01-02") == date && clockErr == nil && parsedClock.Format("15:04") == clock
}

func (handler *Handler) refundNotification(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	if handler == nil || handler.parser == nil || handler.application == nil || len(ctx.Request.Header.Values("Idempotency-Key")) != 0 || !isRefundJSONRequest(ctx.Request) {
		writeRefundHTTPError(ctx, http.StatusBadRequest, "INVALID_CALLBACK")
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxRefundNotificationBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxRefundNotificationBodyBytes || !utf8.Valid(body) {
		writeRefundHTTPError(ctx, http.StatusBadRequest, "INVALID_CALLBACK")
		return
	}
	headers, ok := exactRefundSignatureHeaders(ctx.Request)
	if !ok {
		writeRefundHTTPError(ctx, http.StatusUnauthorized, "CALLBACK_VERIFICATION_FAILED")
		return
	}
	verified, err := handler.parser.ParseRefundNotification(body, headers)
	if err != nil {
		writeRefundHTTPError(ctx, http.StatusUnauthorized, "CALLBACK_VERIFICATION_FAILED")
		return
	}
	if err := handler.application.IngestRefund(ctx.Request.Context(), verified); err != nil {
		writeRefundApplicationError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (handler *Handler) writeMeta(ctx *gin.Context) (WriteMeta, bool) {
	if handler == nil || handler.authenticator == nil || handler.application == nil || handler.orders == nil {
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	token, err := httpdto.BearerToken(ctx.Request)
	if err != nil {
		writeRefundHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
		return WriteMeta{}, false
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnavailable) {
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	if err != nil || userID == 0 {
		writeRefundHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
		return WriteMeta{}, false
	}
	key, err := httpdto.IdempotencyKey(ctx.Request)
	if err != nil || len(key) > 128 {
		writeRefundHTTPError(ctx, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY")
		return WriteMeta{}, false
	}
	requestID := ctx.GetString("request_id")
	if requestID == "" || len(requestID) > 64 {
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	return WriteMeta{ActorUserID: userID, IdempotencyKey: key, RequestID: requestID}, true
}

func validHTTPReason(value string) bool {
	if value == "" || len(value) > 64 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] == 0x7f {
			return false
		}
	}
	return true
}

func isRefundJSONRequest(request *http.Request) bool {
	values := request.Header.Values("Content-Type")
	if len(values) != 1 {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(values[0])
	return err == nil && mediaType == "application/json"
}

func exactRefundSignatureHeaders(request *http.Request) (wechatpay.SignatureHeaders, bool) {
	serial, serialOK := exactRefundHeader(request, "Wechatpay-Serial", 128)
	signature, signatureOK := exactRefundHeader(request, "Wechatpay-Signature", 1024)
	timestamp, timestampOK := exactRefundHeader(request, "Wechatpay-Timestamp", 32)
	nonce, nonceOK := exactRefundHeader(request, "Wechatpay-Nonce", 128)
	return wechatpay.SignatureHeaders{Serial: serial, Signature: signature, Timestamp: timestamp, Nonce: nonce}, serialOK && signatureOK && timestampOK && nonceOK
}

func exactRefundHeader(request *http.Request, name string, limit int) (string, bool) {
	values := request.Header.Values(name)
	if len(values) != 1 || values[0] == "" || len(values[0]) > limit || !utf8.ValidString(values[0]) || strings.TrimSpace(values[0]) != values[0] || strings.Contains(values[0], ",") {
		return "", false
	}
	for index := 0; index < len(values[0]); index++ {
		if values[0][index] < 0x20 || values[0][index] == 0x7f {
			return "", false
		}
	}
	return values[0], true
}

func writeUserOrderReadError(ctx *gin.Context, err error) {
	if errors.Is(err, orderquery.ErrNotFound) || errors.Is(err, orderquery.ErrForbidden) {
		writeRefundHTTPError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND")
		return
	}
	writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
}

func writeRefundApplicationError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeRefundHTTPError(ctx, http.StatusBadRequest, "INVALID_INPUT")
	case errors.Is(err, ErrUnauthenticated):
		writeRefundHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotFound):
		writeRefundHTTPError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND")
	case errors.Is(err, ErrIdempotencyConflict):
		writeRefundHTTPError(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
	case errors.Is(err, ErrTransitionNotAllowed):
		writeRefundHTTPError(ctx, http.StatusUnprocessableEntity, "CANCELLATION_NOT_ALLOWED")
	default:
		writeRefundHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
	}
}

func writeRefundHTTPError(ctx *gin.Context, status int, code string) {
	ctx.JSON(status, gin.H{"error": gin.H{"code": code, "message": "request could not be completed"}})
}

type orderSummaryResponse struct {
	ID               string              `json:"id"`
	OrderNo          string              `json:"order_no"`
	State            orderquery.State    `json:"state"`
	PickupDate       string              `json:"pickup_date"`
	PickupTime       string              `json:"pickup_time"`
	PickupPoint      string              `json:"pickup_point"`
	PickupNumber     string              `json:"pickup_number"`
	PayableCents     uint64              `json:"payable_cents"`
	MaterializedAt   string              `json:"materialized_at"`
	AvailableActions []orderquery.Action `json:"available_actions"`
}

type userOrderResponse struct {
	orderSummaryResponse
	Contact struct {
		Name        string `json:"name"`
		MaskedPhone string `json:"masked_phone"`
	} `json:"contact"`
	Identity struct {
		Kind string `json:"kind"`
	} `json:"identity"`
	Discount struct {
		RatePercent uint8 `json:"rate_percent"`
	} `json:"discount"`
	Items []struct {
		ProductID      string   `json:"product_id"`
		Name           string   `json:"name"`
		Quantity       uint64   `json:"quantity"`
		UnitPriceCents uint64   `json:"unit_price_cents"`
		LineTotalCents uint64   `json:"line_total_cents"`
		Flavors        []string `json:"flavors"`
		Note           string   `json:"note"`
	} `json:"items"`
	TransactionID   string `json:"transaction_id"`
	PaidAt          string `json:"paid_at"`
	RedemptionToken string `json:"redemption_token,omitempty"`
	TransitionTimes struct {
		PreparingAt string `json:"preparing_at,omitempty"`
		ReadyAt     string `json:"ready_at,omitempty"`
		CompletedAt string `json:"completed_at,omitempty"`
		RefundingAt string `json:"refunding_at,omitempty"`
		RefundedAt  string `json:"refunded_at,omitempty"`
	} `json:"transition_times"`
	NotificationOptions []string `json:"notification_options"`
	OrderNote           string   `json:"order_note"`
}

type refundResponse struct {
	ID               string        `json:"id"`
	OrderID          string        `json:"order_id,omitempty"`
	State            ProviderState `json:"state"`
	AmountCents      uint64        `json:"amount_cents"`
	RequestedAt      string        `json:"requested_at"`
	ProviderRefundID string        `json:"provider_refund_id,omitempty"`
}

type cancelResponse struct {
	Order  userOrderResponse `json:"order"`
	Refund refundResponse    `json:"refund"`
}

func projectUserOrder(detail orderquery.Detail) userOrderResponse {
	actions := detail.AvailableActions
	if actions == nil {
		actions = []orderquery.Action{}
	}
	response := userOrderResponse{orderSummaryResponse: orderSummaryResponse{
		ID: strconv.FormatUint(detail.ID, 10), OrderNo: detail.OrderNo, State: detail.State,
		PickupDate: detail.PickupDate, PickupTime: detail.PickupTime, PickupPoint: detail.PickupPoint,
		PickupNumber: strconv.FormatUint(uint64(detail.PickupNumber)+10000, 10)[1:], PayableCents: detail.PayableCents,
		MaterializedAt: detail.MaterializedAt.UTC().Format(time.RFC3339Nano), AvailableActions: actions,
	}}
	response.Contact.Name, response.Contact.MaskedPhone = detail.Contact.Name, detail.Contact.MaskedPhone
	response.Identity.Kind = detail.Identity.Kind
	response.Discount.RatePercent = detail.Discount.RatePercent
	response.Items = make([]struct {
		ProductID      string   `json:"product_id"`
		Name           string   `json:"name"`
		Quantity       uint64   `json:"quantity"`
		UnitPriceCents uint64   `json:"unit_price_cents"`
		LineTotalCents uint64   `json:"line_total_cents"`
		Flavors        []string `json:"flavors"`
		Note           string   `json:"note"`
	}, 0, len(detail.Items))
	for _, item := range detail.Items {
		flavors := item.Flavors
		if flavors == nil {
			flavors = []string{}
		}
		response.Items = append(response.Items, struct {
			ProductID      string   `json:"product_id"`
			Name           string   `json:"name"`
			Quantity       uint64   `json:"quantity"`
			UnitPriceCents uint64   `json:"unit_price_cents"`
			LineTotalCents uint64   `json:"line_total_cents"`
			Flavors        []string `json:"flavors"`
			Note           string   `json:"note"`
		}{strconv.FormatUint(item.ProductID, 10), item.Name, item.Quantity, item.UnitPriceCents, item.LineTotalCents, flavors, item.Note})
	}
	response.TransactionID = detail.TransactionID
	response.PaidAt = detail.PaidAt.UTC().Format(time.RFC3339Nano)
	response.RedemptionToken = detail.RedemptionToken
	response.TransitionTimes.PreparingAt = optionalRefundTime(detail.TransitionTimes.PreparingAt)
	response.TransitionTimes.ReadyAt = optionalRefundTime(detail.TransitionTimes.ReadyAt)
	response.TransitionTimes.CompletedAt = optionalRefundTime(detail.TransitionTimes.CompletedAt)
	response.TransitionTimes.RefundingAt = optionalRefundTime(detail.TransitionTimes.RefundingAt)
	response.TransitionTimes.RefundedAt = optionalRefundTime(detail.TransitionTimes.RefundedAt)
	response.NotificationOptions = detail.NotificationOptions
	if response.NotificationOptions == nil {
		response.NotificationOptions = []string{}
	}
	response.OrderNote = detail.OrderNote
	return response
}

func projectRefund(value Refund) refundResponse {
	response := refundResponse{
		ID: strconv.FormatUint(value.ID, 10), State: value.State, AmountCents: value.AmountCents,
		RequestedAt: value.RequestedAt.UTC().Format(time.RFC3339Nano), ProviderRefundID: value.ProviderRefundID,
	}
	if value.OrderID > 0 {
		response.OrderID = strconv.FormatUint(value.OrderID, 10)
	}
	return response
}

func optionalRefundTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
