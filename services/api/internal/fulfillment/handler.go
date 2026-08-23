package fulfillment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/storestatus"
	"github.com/gin-gonic/gin"
)

const maxCommandBodyBytes = 4096

type Authenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

type Handler struct {
	auth      Authenticator
	app       Application
	reader    MerchantOrderReader
	soldOut   SoldOutCommander
	storeStat StoreStatusCommander
}

type HandlerOption func(*Handler)

func WithSoldOut(value SoldOutCommander) HandlerOption {
	return func(handler *Handler) { handler.soldOut = value }
}
func WithStoreStatus(value StoreStatusCommander) HandlerOption {
	return func(handler *Handler) { handler.storeStat = value }
}

func NewHandler(auth Authenticator, app Application, reader MerchantOrderReader, options ...HandlerOption) *Handler {
	handler := &Handler{auth: auth, app: app, reader: reader}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler
}

func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/merchant/orders/:id/ready", handler.markReady)
	engine.POST("/api/v1/merchant/orders/:id/redeem", handler.redeem)
	engine.POST("/api/v1/verify/scan", handler.verifyScan)
	engine.POST("/api/v1/verify/code", handler.verifyCode)
	engine.PUT("/api/v1/merchant/products/:id/soldout", handler.setSoldOut)
	engine.PUT("/api/v1/merchant/store-status", handler.setStoreStatus)
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) markReady(ctx *gin.Context) { handler.execute(ctx, CommandMarkReady) }
func (handler *Handler) redeem(ctx *gin.Context)    { handler.execute(ctx, CommandRedeemOrder) }

func (handler *Handler) execute(ctx *gin.Context, kind CommandKind) {
	ctx.Header("Cache-Control", "no-store")
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	orderID, ok := strictID(ctx.Param("id"))
	if !ok {
		writeError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
		return
	}
	fields, ok := strictObject(ctx.Request)
	if !ok || len(fields) != 0 {
		writeInvalid(ctx)
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeInvalid(ctx)
		return
	}
	if handler == nil || handler.app == nil || handler.reader == nil {
		writeUnavailable(ctx)
		return
	}
	handler.executeCommandResponse(ctx, userID, key, Command{Kind: kind, OrderID: orderID})
}

func (handler *Handler) executeCommandResponse(ctx *gin.Context, userID uint64, key string, command Command) {
	result, err := handler.app.Execute(ctx.Request.Context(), WriteMeta{
		ActorUserID: userID, IdempotencyKey: key, RequestID: ctx.GetString("request_id"),
	}, command)
	if err != nil {
		writeApplicationError(ctx, err)
		return
	}
	expectedState := orderquery.StateCompleted
	if command.Kind == CommandMarkReady {
		expectedState = orderquery.StateReadyForPickup
	}
	if result.OrderID == 0 || result.State != expectedState || (command.OrderID > 0 && result.OrderID != command.OrderID) {
		writeUnavailable(ctx)
		return
	}
	detail, err := handler.reader.GetMerchantAtState(ctx.Request.Context(), userID, result.OrderID, result.State)
	if err != nil || !orderquery.WriteOrderResponse(ctx, detail) {
		writeUnavailable(ctx)
	}
}

func (handler *Handler) verifyScan(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userID, authenticated := handler.authenticate(ctx)
	if !authenticated {
		return
	}
	fields, ok := strictObject(ctx.Request)
	var token string
	if !ok || len(fields) != 1 || !decodeField(fields, "token", &token) || !validPlainToken(token) {
		writeInvalid(ctx)
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeInvalid(ctx)
		return
	}
	if handler == nil || handler.app == nil || handler.reader == nil {
		writeUnavailable(ctx)
		return
	}
	handler.executeCommandResponse(ctx, userID, key, Command{Kind: CommandRedeemToken, Token: token})
}

func (handler *Handler) verifyCode(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userID, authenticated := handler.authenticate(ctx)
	if !authenticated {
		return
	}
	fields, ok := strictObject(ctx.Request)
	var code string
	if !ok || len(fields) != 1 || !decodeField(fields, "pickup_number", &code) || len(code) != 4 || !allDigits(code) {
		writeInvalid(ctx)
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeInvalid(ctx)
		return
	}
	if handler == nil || handler.app == nil || handler.reader == nil {
		writeUnavailable(ctx)
		return
	}
	handler.executeCommandResponse(ctx, userID, key, Command{Kind: CommandRedeemCurrentDateCode, PickupNumber: code})
}

func (handler *Handler) setSoldOut(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	productID, ok := strictID(ctx.Param("id"))
	fields, objectOK := strictObject(ctx.Request)
	var date string
	var soldOut bool
	if !ok || !objectOK || len(fields) != 2 || !decodeField(fields, "service_date", &date) || !decodeField(fields, "sold_out", &soldOut) || !strictDate(date) {
		writeInvalid(ctx)
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeInvalid(ctx)
		return
	}
	if handler == nil || handler.soldOut == nil {
		writeUnavailable(ctx)
		return
	}
	err := handler.soldOut.SetSoldOut(ctx.Request.Context(), WriteMeta{ActorUserID: userID, IdempotencyKey: key, RequestID: ctx.GetString("request_id")}, SoldOutCommand{
		ProductID: productID, ServiceDate: date, SoldOut: &soldOut,
	})
	if err != nil {
		writeApplicationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"product_id": strconv.FormatUint(productID, 10), "service_date": date, "sold_out": soldOut})
}

func (handler *Handler) setStoreStatus(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	fields, objectOK := strictObject(ctx.Request)
	var status string
	if !objectOK || len(fields) != 1 || !decodeField(fields, "status", &status) || (status != "open" && status != "closed" && status != "cutoff") {
		writeInvalid(ctx)
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeInvalid(ctx)
		return
	}
	if handler == nil || handler.storeStat == nil {
		writeUnavailable(ctx)
		return
	}
	result, err := handler.storeStat.Apply(ctx.Request.Context(), storestatus.Command{
		UserID: userID, DesiredStatus: storefront.BusinessStatus(status), IdempotencyKey: key, RequestID: ctx.GetString("request_id"),
	})
	if err != nil || result.After != storefront.BusinessStatus(status) {
		writeApplicationError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"store_status": status})
}

func (handler *Handler) authenticate(ctx *gin.Context) (uint64, bool) {
	values := ctx.Request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		writeUnauthenticated(ctx)
		return 0, false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		writeUnauthenticated(ctx)
		return 0, false
	}
	if handler == nil || handler.auth == nil {
		writeUnavailable(ctx)
		return 0, false
	}
	userID, err := handler.auth.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) || (err == nil && userID == 0) {
		writeUnauthenticated(ctx)
		return 0, false
	}
	if err != nil {
		writeUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func strictObject(request *http.Request) (map[string]json.RawMessage, bool) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maxCommandBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxCommandBodyBytes || !utf8.Valid(body) {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return nil, false
	}
	result := make(map[string]json.RawMessage)
	for decoder.More() {
		keyValue, err := decoder.Token()
		key, keyOK := keyValue.(string)
		if err != nil || !keyOK {
			return nil, false
		}
		if _, duplicate := result[key]; duplicate {
			return nil, false
		}
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return nil, false
		}
		result[key] = raw
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return nil, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	return result, true
}

func decodeField(fields map[string]json.RawMessage, key string, target any) bool {
	raw, ok := fields[key]
	return ok && json.Unmarshal(raw, target) == nil
}
func exactIdempotencyKey(values []string) (string, bool) {
	if len(values) != 1 || len(values[0]) == 0 || len(values[0]) > 128 {
		return "", false
	}
	for _, value := range []byte(values[0]) {
		if value < 0x21 || value > 0x7e {
			return "", false
		}
	}
	return values[0], true
}
func strictID(value string) (uint64, bool) {
	if value == "" || value[0] == '0' {
		return 0, false
	}
	id, err := strconv.ParseUint(value, 10, 64)
	return id, err == nil && id > 0
}
func strictDate(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	for index, value := range []byte(value) {
		if index != 4 && index != 7 && (value < '0' || value > '9') {
			return false
		}
	}
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}
func allDigits(value string) bool {
	for _, char := range []byte(value) {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func writeApplicationError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, storestatus.ErrInvalidCommand):
		writeInvalid(ctx)
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrRedemptionInvalid):
		writeError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
	case errors.Is(err, ErrForbidden), errors.Is(err, merchantidentity.ErrForbidden), errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable):
		writeError(ctx, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, ErrTransitionNotAllowed):
		writeError(ctx, http.StatusConflict, "TRANSITION_NOT_ALLOWED", "transition not allowed")
	case errors.Is(err, ErrIdempotencyConflict), errors.Is(err, storestatus.ErrIdempotencyConflict):
		writeError(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict")
	default:
		writeUnavailable(ctx)
	}
}
func writeInvalid(ctx *gin.Context) {
	writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
}
func writeUnauthenticated(ctx *gin.Context) {
	writeError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
}
func writeUnavailable(ctx *gin.Context) {
	writeError(ctx, http.StatusServiceUnavailable, "FULFILLMENT_UNAVAILABLE", "fulfillment temporarily unavailable")
}
func writeError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorResponse{Code: code, Message: message}})
}
