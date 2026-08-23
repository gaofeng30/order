package paymentorder

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
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/gin-gonic/gin"
)

const (
	maxPaymentBodyBytes      = 8 * 1024
	maxNotificationBodyBytes = 64 * 1024
)

type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

type Handler struct {
	authenticator SessionAuthenticator
	application   Application
	parser        NotificationParser
}

func NewHandler(authenticator SessionAuthenticator, application Application, parser NotificationParser) *Handler {
	return &Handler{authenticator: authenticator, application: application, parser: parser}
}

func (handler *Handler) RegisterRoutes(api *gin.RouterGroup) {
	if handler == nil || api == nil {
		return
	}
	api.POST("/orders/prepay", handler.prepare)
	api.POST("/orders/confirm", handler.confirm)
	api.POST("/payments/wechat/notify", handler.paymentNotification)
}

type jsonID uint64

func (id *jsonID) UnmarshalJSON(data []byte) error {
	var encoded string
	if json.Unmarshal(data, &encoded) != nil || encoded == "" || encoded[0] == '0' {
		return ErrInvalidInput
	}
	value, err := strconv.ParseUint(encoded, 10, 64)
	if err != nil || value == 0 || strconv.FormatUint(value, 10) != encoded {
		return ErrInvalidInput
	}
	*id = jsonID(value)
	return nil
}

func (handler *Handler) prepare(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	meta, ok := handler.writeMeta(ctx)
	if !ok {
		return
	}
	var request struct {
		QuoteID jsonID `json:"quote_id"`
	}
	if !isJSONRequest(ctx.Request) || decodeStrict(ctx.Request.Body, maxPaymentBodyBytes, &request) != nil || request.QuoteID == 0 {
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_INPUT")
		return
	}
	result, err := handler.application.Prepare(ctx.Request.Context(), meta, uint64(request.QuoteID))
	if err != nil {
		writeApplicationError(ctx, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	if result.Prepayment.WxRequestPayment == nil && result.Prepayment.State != ProviderPaid {
		status = http.StatusAccepted
	}
	ctx.JSON(status, prepaymentEnvelope(result.Prepayment))
}

func (handler *Handler) confirm(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	meta, ok := handler.writeMeta(ctx)
	if !ok {
		return
	}
	var request struct {
		PrepaymentID jsonID `json:"prepayment_id"`
	}
	if !isJSONRequest(ctx.Request) || decodeStrict(ctx.Request.Body, maxPaymentBodyBytes, &request) != nil || request.PrepaymentID == 0 {
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_INPUT")
		return
	}
	result, err := handler.application.Confirm(ctx.Request.Context(), meta, uint64(request.PrepaymentID))
	if err != nil {
		writeApplicationError(ctx, err)
		return
	}
	if result.State == ConfirmOrderCreated && result.OrderID > 0 {
		ctx.JSON(http.StatusOK, gin.H{"state": result.State, "order_id": strconv.FormatUint(result.OrderID, 10)})
		return
	}
	ctx.JSON(http.StatusAccepted, gin.H{"state": ConfirmPending})
}

func (handler *Handler) paymentNotification(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	if handler.parser == nil || handler.application == nil || len(ctx.Request.Header.Values("Idempotency-Key")) != 0 {
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_CALLBACK")
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxNotificationBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxNotificationBodyBytes || !utf8.Valid(body) {
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_CALLBACK")
		return
	}
	headers, ok := exactSignatureHeaders(ctx.Request)
	if !ok {
		writeHTTPError(ctx, http.StatusUnauthorized, "CALLBACK_VERIFICATION_FAILED")
		return
	}
	verified, err := handler.parser.ParsePaymentNotification(body, headers)
	if err != nil {
		writeHTTPError(ctx, http.StatusUnauthorized, "CALLBACK_VERIFICATION_FAILED")
		return
	}
	if err := handler.application.IngestPayment(ctx.Request.Context(), verified); err != nil {
		writeApplicationError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (handler *Handler) writeMeta(ctx *gin.Context) (WriteMeta, bool) {
	if handler == nil || handler.authenticator == nil || handler.application == nil {
		writeHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	token, ok := exactBearer(ctx.Request)
	if !ok {
		writeHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
		return WriteMeta{}, false
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnavailable) {
		writeHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	if err != nil || userID == 0 {
		writeHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
		return WriteMeta{}, false
	}
	key, ok := exactOpaqueHeader(ctx.Request, "Idempotency-Key")
	if !ok {
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY")
		return WriteMeta{}, false
	}
	requestID := ctx.GetString("request_id")
	if requestID == "" {
		writeHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
		return WriteMeta{}, false
	}
	return WriteMeta{ActorUserID: userID, IdempotencyKey: key, RequestID: requestID}, true
}

func prepaymentEnvelope(prepayment Prepayment) any {
	type responsePrepayment struct {
		ID               string                    `json:"id"`
		State            ProviderState             `json:"state"`
		ExpiresAt        string                    `json:"expires_at"`
		WxRequestPayment *wechatpay.RequestPayment `json:"wx_request_payment,omitempty"`
	}
	return struct {
		Prepayment responsePrepayment `json:"prepayment"`
	}{Prepayment: responsePrepayment{
		ID: strconv.FormatUint(prepayment.ID, 10), State: prepayment.State,
		ExpiresAt:        prepayment.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		WxRequestPayment: prepayment.WxRequestPayment,
	}}
}

func writeApplicationError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeHTTPError(ctx, http.StatusBadRequest, "INVALID_INPUT")
	case errors.Is(err, ErrUnauthenticated):
		writeHTTPError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED")
	case errors.Is(err, ErrForbidden):
		writeHTTPError(ctx, http.StatusForbidden, "FORBIDDEN")
	case errors.Is(err, ErrNotFound):
		writeHTTPError(ctx, http.StatusNotFound, "NOT_FOUND")
	case errors.Is(err, ErrIdempotencyConflict):
		writeHTTPError(ctx, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
	case errors.Is(err, ErrQuoteUnavailable):
		writeHTTPError(ctx, http.StatusConflict, "QUOTE_UNAVAILABLE")
	default:
		writeHTTPError(ctx, http.StatusServiceUnavailable, "UNAVAILABLE")
	}
}

func writeHTTPError(ctx *gin.Context, status int, code string) {
	ctx.JSON(status, gin.H{"error": gin.H{"code": code, "message": "request could not be completed"}})
}

func exactBearer(request *http.Request) (string, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	return token, token != "" && utf8.ValidString(token) && !strings.ContainsAny(token, " \t\r\n,")
}

func isJSONRequest(request *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

func exactOpaqueHeader(request *http.Request, name string) (string, bool) {
	values := request.Header.Values(name)
	if len(values) != 1 || values[0] == "" || len(values[0]) > 128 || !utf8.ValidString(values[0]) || strings.TrimSpace(values[0]) != values[0] || strings.Contains(values[0], ",") {
		return "", false
	}
	for index := 0; index < len(values[0]); index++ {
		if values[0][index] < 0x20 || values[0][index] == 0x7f {
			return "", false
		}
	}
	return values[0], true
}

func exactSignatureHeaders(request *http.Request) (wechatpay.SignatureHeaders, bool) {
	serial, serialOK := exactOpaqueHeader(request, "Wechatpay-Serial")
	signature, signatureOK := exactOpaqueHeader(request, "Wechatpay-Signature")
	timestamp, timestampOK := exactOpaqueHeader(request, "Wechatpay-Timestamp")
	nonce, nonceOK := exactOpaqueHeader(request, "Wechatpay-Nonce")
	return wechatpay.SignatureHeaders{Serial: serial, Signature: signature, Timestamp: timestamp, Nonce: nonce}, serialOK && signatureOK && timestampOK && nonceOK
}

func decodeStrict(reader io.Reader, limit int64, target any) error {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit || !utf8.Valid(body) || !uniqueJSONKeys(body) {
		return ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidInput
	}
	if err := requireJSONEnd(decoder); err != nil {
		return ErrInvalidInput
	}
	return nil
}

func uniqueJSONKeys(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if consumeJSONValue(decoder) != nil {
		return false
	}
	return requireJSONEnd(decoder) == nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, structured := token.(json.Delim)
	if !structured {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return ErrInvalidInput
	}
	seen := map[string]struct{}{}
	for decoder.More() {
		if delimiter == '{' {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return ErrInvalidInput
			}
			if _, duplicate := seen[key]; duplicate {
				return ErrInvalidInput
			}
			seen[key] = struct{}{}
		}
		if err := consumeJSONValue(decoder); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || (delimiter == '{' && end != json.Delim('}')) || (delimiter == '[' && end != json.Delim(']')) {
		return ErrInvalidInput
	}
	return nil
}
