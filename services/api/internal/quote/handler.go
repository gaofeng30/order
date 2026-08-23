package quote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

const maxCreateBodyBytes = 64 * 1024

// SessionAuthenticator resolves one existing Mini Program bearer.
type SessionAuthenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

// Handler exposes only authenticated quote creation and owner reads.
type Handler struct {
	authenticator SessionAuthenticator
	application   Application
}

// NewHandler constructs the quote HTTP adapter.
func NewHandler(authenticator SessionAuthenticator, application Application) *Handler {
	return &Handler{authenticator: authenticator, application: application}
}

// RegisterRoutes adds only the authenticated quote routes.
func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/quotes", handler.create)
	engine.GET("/api/v1/quotes/:id", handler.read)
}

type createRequest struct {
	ContactName string              `json:"contact_name"`
	PickupDate  string              `json:"pickup_date"`
	PickupTime  string              `json:"pickup_time"`
	OrderNote   string              `json:"order_note"`
	Items       []createItemRequest `json:"items"`
}

type createItemRequest struct {
	ProductID string   `json:"product_id"`
	Quantity  int64    `json:"quantity"`
	Flavors   []string `json:"flavors"`
	Note      string   `json:"note"`
}

type quoteEnvelope struct {
	Quote quoteResponse `json:"quote"`
}

type quoteResponse struct {
	ID                    string           `json:"id"`
	Contact               contactResponse  `json:"contact"`
	Identity              identityResponse `json:"identity"`
	Discount              discountResponse `json:"discount"`
	Store                 storeResponse    `json:"store"`
	Pickup                pickupResponse   `json:"pickup"`
	OrderNote             string           `json:"order_note"`
	Items                 []itemResponse   `json:"items"`
	OriginalSubtotalCents int64            `json:"original_subtotal_cents"`
	DiscountCents         int64            `json:"discount_cents"`
	PayableCents          int64            `json:"payable_cents"`
	CreatedAt             string           `json:"created_at"`
	ExpiresAt             string           `json:"expires_at"`
}

type contactResponse struct {
	Name        string `json:"name"`
	MaskedPhone string `json:"masked_phone"`
}

type identityResponse struct {
	Kind IdentityKind `json:"kind"`
}

type discountResponse struct {
	RatePercent int64 `json:"rate_percent"`
}

type storeResponse struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type pickupResponse struct {
	Date       string `json:"date"`
	Time       string `json:"time"`
	MealPeriod string `json:"meal_period"`
	Point      string `json:"point"`
}

type itemResponse struct {
	LineNumber               uint16   `json:"line_number"`
	ProductID                string   `json:"product_id"`
	Name                     string   `json:"name"`
	ImageObjectKey           string   `json:"image_object_key,omitempty"`
	OriginalUnitPriceCents   int64    `json:"original_unit_price_cents"`
	DiscountedUnitPriceCents int64    `json:"discounted_unit_price_cents"`
	Quantity                 int64    `json:"quantity"`
	OriginalSubtotalCents    int64    `json:"original_subtotal_cents"`
	PayableSubtotalCents     int64    `json:"payable_subtotal_cents"`
	Flavors                  []string `json:"flavors"`
	Note                     string   `json:"note"`
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (handler *Handler) create(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	key, ok := exactIdempotencyKey(ctx.Request.Header.Values("Idempotency-Key"))
	if !ok {
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxCreateBodyBytes+1))
	if err != nil || len(body) == 0 || len(body) > maxCreateBodyBytes || !utf8.Valid(body) {
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	input, ok := decodeCreateRequest(body)
	if !ok {
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	if handler == nil || handler.application == nil {
		writeQuoteUnavailable(ctx)
		return
	}
	meta := WriteMeta{ActorUserID: userID, IdempotencyKey: key, RequestID: ctx.GetString("request_id")}
	if !validWriteMeta(meta) {
		writeQuoteUnavailable(ctx)
		return
	}
	result, err := handler.application.Create(ctx.Request.Context(), meta, input)
	if err != nil {
		writeQuoteCreateError(ctx, err)
		return
	}
	if !validApplicationQuote(result.Quote, userID, 0) {
		writeQuoteUnavailable(ctx)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	ctx.JSON(status, quoteEnvelope{Quote: newQuoteResponse(result.Quote)})
}

func (handler *Handler) read(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 1))
	if err != nil || len(body) != 0 {
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	quoteID, ok := strictPositiveID(ctx.Param("id"))
	if !ok {
		writeQuoteError(ctx, http.StatusNotFound, "QUOTE_NOT_FOUND", "quote not found")
		return
	}
	if handler == nil || handler.application == nil {
		writeQuoteUnavailable(ctx)
		return
	}
	result, err := handler.application.Read(ctx.Request.Context(), userID, quoteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) {
			writeQuoteError(ctx, http.StatusNotFound, "QUOTE_NOT_FOUND", "quote not found")
			return
		}
		writeQuoteUnavailable(ctx)
		return
	}
	if !validApplicationQuote(result, userID, quoteID) {
		writeQuoteUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, quoteEnvelope{Quote: newQuoteResponse(result)})
}

func (handler *Handler) authenticate(ctx *gin.Context) (uint64, bool) {
	token, ok := exactBearer(ctx.Request.Header.Values("Authorization"))
	if !ok {
		writeQuoteError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return 0, false
	}
	if handler == nil || handler.authenticator == nil {
		writeQuoteUnavailable(ctx)
		return 0, false
	}
	userID, err := handler.authenticator.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, identity.ErrUnauthenticated) || (err == nil && userID == 0) {
		writeQuoteError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return 0, false
	}
	if err != nil {
		writeQuoteUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func exactBearer(values []string) (string, bool) {
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return "", false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	return token, token != "" && !strings.ContainsAny(token, " \t\r\n")
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

func decodeCreateRequest(body []byte) (CreateInput, bool) {
	if !uniqueJSONObjectKeys(body) {
		return CreateInput{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var request createRequest
	if err := decoder.Decode(&request); err != nil {
		return CreateInput{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return CreateInput{}, false
	}
	if !validContactName(request.ContactName) {
		return CreateInput{}, false
	}
	input := CreateInput{ContactName: request.ContactName, PickupDate: request.PickupDate, PickupTime: request.PickupTime, OrderNote: request.OrderNote, Items: make([]ItemInput, 0, len(request.Items))}
	for _, item := range request.Items {
		productID, ok := strictPositiveID(item.ProductID)
		if !ok {
			return CreateInput{}, false
		}
		input.Items = append(input.Items, ItemInput{ProductID: productID, Quantity: item.Quantity, Flavors: item.Flavors, Note: item.Note})
	}
	return input, true
}

func uniqueJSONObjectKeys(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if !readUniqueJSONValue(decoder) {
		return false
	}
	_, err := decoder.Token()
	return errors.Is(err, io.EOF)
}

func readUniqueJSONValue(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return true
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			key, ok := keyToken.(string)
			if err != nil || !ok {
				return false
			}
			if _, duplicate := seen[key]; duplicate {
				return false
			}
			seen[key] = struct{}{}
			if !readUniqueJSONValue(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim('}')
	case '[':
		for decoder.More() {
			if !readUniqueJSONValue(decoder) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == json.Delim(']')
	default:
		return false
	}
}

func strictPositiveID(value string) (uint64, bool) {
	if value == "" {
		return 0, false
	}
	for index := range value {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil && parsed > 0
}

func newQuoteResponse(value Quote) quoteResponse {
	maskedPhone, _ := maskContactPhone(value.Contact.Phone)
	items := make([]itemResponse, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, itemResponse{
			LineNumber: item.LineNumber, ProductID: strconv.FormatUint(item.ProductID, 10), Name: item.ProductName,
			ImageObjectKey:         item.ImageObjectKey,
			OriginalUnitPriceCents: item.OriginalUnitPriceCents, DiscountedUnitPriceCents: item.DiscountedUnitPriceCents,
			Quantity: item.Quantity, OriginalSubtotalCents: item.OriginalSubtotalCents, PayableSubtotalCents: item.PayableSubtotalCents,
			Flavors: item.Flavors, Note: item.Note,
		})
	}
	return quoteResponse{
		ID:        strconv.FormatUint(value.ID, 10),
		Contact:   contactResponse{Name: value.Contact.Name, MaskedPhone: maskedPhone},
		Identity:  identityResponse{Kind: value.Identity.Kind},
		Discount:  discountResponse{RatePercent: value.Discount.RatePercent},
		Store:     storeResponse{Name: value.Store.Name, Address: value.Store.Address},
		Pickup:    pickupResponse{Date: value.Pickup.Date, Time: value.Pickup.Time, MealPeriod: value.Pickup.Meal, Point: value.Pickup.Point},
		OrderNote: value.OrderNote, Items: items,
		OriginalSubtotalCents: value.OriginalSubtotalCents, DiscountCents: value.DiscountCents, PayableCents: value.PayableCents,
		CreatedAt: value.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999Z"),
		ExpiresAt: value.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999Z"),
	}
}

func maskContactPhone(phone string) (string, bool) {
	if !validPrimaryPhone(phone) {
		return "", false
	}
	digitCount := len(phone) - 1
	visibleCount := 4
	if digitCount <= visibleCount {
		visibleCount = digitCount - 1
	}
	hiddenCount := digitCount - visibleCount
	return "+" + strings.Repeat("*", hiddenCount) + phone[len(phone)-visibleCount:], true
}

func validApplicationQuote(value Quote, userID, expectedQuoteID uint64) bool {
	if value.UserID != userID || value.PayableCents < 1 || len(value.Items) == 0 || len(value.Items) > math.MaxUint16 || (expectedQuoteID != 0 && value.ID != expectedQuoteID) {
		return false
	}
	return validStoredQuote(value, uint16(len(value.Items))) && hashQuoteSnapshot(value) == value.SnapshotDigest
}

func writeQuoteError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorResponse{Code: code, Message: message}})
}

func writeQuoteCreateError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeQuoteError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	case errors.Is(err, ErrPrimaryPhoneRequired):
		writeQuoteError(ctx, http.StatusConflict, "PRIMARY_PHONE_REQUIRED", "primary phone required")
	case errors.Is(err, ErrSelectionUnavailable):
		writeQuoteError(ctx, http.StatusConflict, "QUOTE_SELECTION_UNAVAILABLE", "quote selection unavailable")
	case errors.Is(err, ErrIdempotencyConflict):
		writeQuoteError(ctx, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", "idempotency key conflict")
	case errors.Is(err, ErrPaymentAmountTooSmall):
		writeQuoteError(ctx, http.StatusConflict, "PAYMENT_AMOUNT_TOO_SMALL", "payment amount too small")
	default:
		writeQuoteUnavailable(ctx)
	}
}

func writeQuoteUnavailable(ctx *gin.Context) {
	writeQuoteError(ctx, http.StatusServiceUnavailable, "QUOTE_UNAVAILABLE", "quote temporarily unavailable")
}
