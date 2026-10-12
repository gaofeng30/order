package orderquery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gin-gonic/gin"
)

type Authenticator interface {
	Authenticate(context.Context, string) (uint64, error)
}

type Handler struct {
	auth Authenticator
	app  Application
}

func NewHandler(auth Authenticator, app Application) *Handler {
	return &Handler{auth: auth, app: app}
}

func (handler *Handler) RegisterRoutes(engine *gin.Engine) {
	engine.GET("/api/v1/orders", handler.listUser)
	engine.GET("/api/v1/orders/:id", handler.getUser)
	engine.GET("/api/v1/merchant/orders", handler.listMerchant)
	engine.GET("/api/v1/merchant/orders/:id", handler.getMerchant)
}

type errorEnvelope struct {
	Error errorResponse `json:"error"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type summaryResponse struct {
	ID               string   `json:"id"`
	OrderNo          string   `json:"order_no"`
	State            State    `json:"state"`
	PickupDate       string   `json:"pickup_date"`
	PickupTime       string   `json:"pickup_time"`
	PickupPoint      string   `json:"pickup_point"`
	PickupNumber     string   `json:"pickup_number"`
	PayableCents     uint64   `json:"payable_cents"`
	MaterializedAt   string   `json:"materialized_at"`
	AvailableActions []Action `json:"available_actions"`
}

type listResponse struct {
	Orders      []summaryResponse `json:"orders"`
	NextAfterID string            `json:"next_after_id,omitempty"`
}

type contactResponse struct {
	Name        string `json:"name"`
	MaskedPhone string `json:"masked_phone"`
}
type identityResponse struct {
	Kind string `json:"kind"`
}
type discountResponse struct {
	RatePercent uint8 `json:"rate_percent"`
}
type itemResponse struct {
	ProductID      string   `json:"product_id"`
	Name           string   `json:"name"`
	Quantity       uint64   `json:"quantity"`
	UnitPriceCents uint64   `json:"unit_price_cents"`
	LineTotalCents uint64   `json:"line_total_cents"`
	Flavors        []string `json:"flavors"`
	Note           string   `json:"note"`
}
type transitionTimesResponse struct {
	PreparingAt string `json:"preparing_at,omitempty"`
	ReadyAt     string `json:"ready_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	RefundingAt string `json:"refunding_at,omitempty"`
	RefundedAt  string `json:"refunded_at,omitempty"`
}
type detailResponse struct {
	summaryResponse
	Contact             contactResponse         `json:"contact"`
	Identity            identityResponse        `json:"identity"`
	Discount            discountResponse        `json:"discount"`
	Items               []itemResponse          `json:"items"`
	TransactionID       string                  `json:"transaction_id"`
	PaidAt              string                  `json:"paid_at"`
	RedemptionToken     string                  `json:"redemption_token,omitempty"`
	TransitionTimes     transitionTimesResponse `json:"transition_times"`
	NotificationOptions []string                `json:"notification_options"`
	OrderNote           string                  `json:"order_note"`
}
type detailEnvelope struct {
	Order detailResponse `json:"order"`
}

func (handler *Handler) listUser(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	if !emptyBody(ctx.Request) {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	query, ok := parseUserQuery(ctx.Request)
	if !ok {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	if handler == nil || handler.app == nil {
		writeUnavailable(ctx)
		return
	}
	page, err := handler.app.ListUser(ctx.Request.Context(), userID, query)
	if err != nil || !validPage(page) {
		writeUnavailable(ctx)
		return
	}
	writePage(ctx, page)
}

func (handler *Handler) getUser(ctx *gin.Context) {
	handler.get(ctx, false)
}
func (handler *Handler) getMerchant(ctx *gin.Context) {
	handler.get(ctx, true)
}

func (handler *Handler) get(ctx *gin.Context, merchant bool) {
	ctx.Header("Cache-Control", "no-store")
	if !emptyBody(ctx.Request) {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	orderID, ok := strictID(ctx.Param("id"))
	if !ok {
		writeNotFound(ctx)
		return
	}
	if handler == nil || handler.app == nil {
		writeUnavailable(ctx)
		return
	}
	var detail Detail
	var err error
	if merchant {
		detail, err = handler.app.GetMerchant(ctx.Request.Context(), userID, orderID)
	} else {
		detail, err = handler.app.GetUser(ctx.Request.Context(), userID, orderID)
	}
	if errors.Is(err, ErrNotFound) || (!merchant && errors.Is(err, ErrForbidden)) {
		writeNotFound(ctx)
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(ctx, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if err != nil || !validDetail(detail) {
		writeUnavailable(ctx)
		return
	}
	ctx.JSON(http.StatusOK, detailEnvelope{Order: projectDetail(detail)})
}

func (handler *Handler) listMerchant(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	if !emptyBody(ctx.Request) {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	userID, ok := handler.authenticate(ctx)
	if !ok {
		return
	}
	query, ok := parseMerchantQuery(ctx.Request)
	if !ok {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "invalid request")
		return
	}
	if handler == nil || handler.app == nil {
		writeUnavailable(ctx)
		return
	}
	page, err := handler.app.SearchMerchant(ctx.Request.Context(), userID, query)
	if errors.Is(err, ErrForbidden) {
		writeError(ctx, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if err != nil || !validPage(page) {
		writeUnavailable(ctx)
		return
	}
	writePage(ctx, page)
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
	if errors.Is(err, identity.ErrUnauthenticated) || errors.Is(err, ErrUnauthenticated) || (err == nil && userID == 0) {
		writeUnauthenticated(ctx)
		return 0, false
	}
	if err != nil {
		writeUnavailable(ctx)
		return 0, false
	}
	return userID, true
}

func parseUserQuery(request *http.Request) (UserQuery, bool) {
	query := request.URL.Query()
	if !onlySingleKeys(query, "state", "active", "after_id", "limit") {
		return UserQuery{}, false
	}
	result := UserQuery{Limit: 20}
	if value := query.Get("state"); value != "" {
		if !validState(State(value)) {
			return UserQuery{}, false
		}
		result.State = State(value)
	}
	if values, exists := query["active"]; exists {
		if values[0] != "true" || result.State != "" {
			return UserQuery{}, false
		}
		result.Active = true
	}
	var ok bool
	if result.AfterID, ok = optionalID(query.Get("after_id")); !ok {
		return UserQuery{}, false
	}
	if result.Limit, ok = optionalLimit(query.Get("limit"), 20); !ok {
		return UserQuery{}, false
	}
	return result, true
}

func parseMerchantQuery(request *http.Request) (MerchantQuery, bool) {
	query := request.URL.Query()
	if !onlySingleKeys(query, "state", "date", "q", "after_id", "limit") {
		return MerchantQuery{}, false
	}
	result := MerchantQuery{Limit: 20}
	if value := query.Get("state"); value != "" {
		if !validState(State(value)) {
			return MerchantQuery{}, false
		}
		result.State = State(value)
	}
	result.Date = query.Get("date")
	if result.Date != "" && !strictDate(result.Date) {
		return MerchantQuery{}, false
	}
	result.Search = query.Get("q")
	if result.Search != "" && (!utf8.ValidString(result.Search) || len(result.Search) > 64 || strings.TrimSpace(result.Search) != result.Search) {
		return MerchantQuery{}, false
	}
	var ok bool
	if result.AfterID, ok = optionalID(query.Get("after_id")); !ok {
		return MerchantQuery{}, false
	}
	if result.Limit, ok = optionalLimit(query.Get("limit"), 20); !ok {
		return MerchantQuery{}, false
	}
	return result, true
}

func onlySingleKeys(query map[string][]string, allowed ...string) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		set[key] = struct{}{}
	}
	for key, values := range query {
		if _, ok := set[key]; !ok || len(values) != 1 || values[0] == "" {
			return false
		}
	}
	return true
}

func strictID(value string) (uint64, bool) {
	if value == "" || value[0] == '0' {
		return 0, false
	}
	result, err := strconv.ParseUint(value, 10, 64)
	return result, err == nil && result > 0
}
func optionalID(value string) (uint64, bool) {
	if value == "" {
		return 0, true
	}
	return strictID(value)
}
func optionalLimit(value string, fallback uint16) (uint16, bool) {
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	return uint16(parsed), err == nil && parsed >= 1 && parsed <= 100
}
func strictDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}
func emptyBody(request *http.Request) bool {
	body, err := io.ReadAll(io.LimitReader(request.Body, 1))
	return err == nil && len(body) == 0
}

func validState(state State) bool {
	switch state {
	case StateReserved, StatePreparing, StateReadyForPickup, StateCompleted, StateRefunding, StateRefunded:
		return true
	default:
		return false
	}
}
func validSummary(summary Summary) bool {
	return summary.ID > 0 && summary.OrderNo != "" && validState(summary.State) && strictDate(summary.PickupDate) &&
		len(summary.PickupTime) == 5 && summary.PickupPoint != "" && summary.PickupNumber >= 1 && summary.PickupNumber <= 9999 &&
		summary.PayableCents > 0 && !summary.MaterializedAt.IsZero() && summary.AvailableActions != nil
}
func validPage(page Page) bool {
	if page.Orders == nil {
		return false
	}
	for _, order := range page.Orders {
		if !validSummary(order) {
			return false
		}
	}
	return true
}
func validDetail(detail Detail) bool {
	if !validSummary(detail.Summary) || detail.Items == nil || detail.NotificationOptions == nil || detail.Contact.Name == "" ||
		detail.Contact.MaskedPhone == "" || (detail.Identity.Kind != "STAFF" && detail.Identity.Kind != "VISITOR") ||
		detail.Discount.RatePercent < 1 || detail.Discount.RatePercent > 100 || detail.TransactionID == "" || detail.PaidAt.IsZero() {
		return false
	}
	for _, item := range detail.Items {
		if item.ProductID == 0 || item.Name == "" || item.Quantity == 0 || item.Flavors == nil {
			return false
		}
	}
	return detail.State == StateReadyForPickup || detail.RedemptionToken == ""
}

func writePage(ctx *gin.Context, page Page) {
	response := listResponse{Orders: make([]summaryResponse, 0, len(page.Orders))}
	for _, order := range page.Orders {
		response.Orders = append(response.Orders, projectSummary(order))
	}
	if page.NextAfterID > 0 {
		response.NextAfterID = strconv.FormatUint(page.NextAfterID, 10)
	}
	ctx.JSON(http.StatusOK, response)
}
func projectSummary(summary Summary) summaryResponse {
	actions := summary.AvailableActions
	if actions == nil {
		actions = []Action{}
	}
	return summaryResponse{
		ID: strconv.FormatUint(summary.ID, 10), OrderNo: summary.OrderNo, State: summary.State,
		PickupDate: summary.PickupDate, PickupTime: summary.PickupTime, PickupPoint: summary.PickupPoint,
		PickupNumber: strconv.FormatUint(uint64(summary.PickupNumber)+10000, 10)[1:], PayableCents: summary.PayableCents,
		MaterializedAt: summary.MaterializedAt.UTC().Format(time.RFC3339Nano), AvailableActions: actions,
	}
}
func projectDetail(detail Detail) detailResponse {
	items := make([]itemResponse, 0, len(detail.Items))
	for _, item := range detail.Items {
		flavors := item.Flavors
		if flavors == nil {
			flavors = []string{}
		}
		items = append(items, itemResponse{
			ProductID: strconv.FormatUint(item.ProductID, 10), Name: item.Name, Quantity: item.Quantity,
			UnitPriceCents: item.UnitPriceCents, LineTotalCents: item.LineTotalCents, Flavors: flavors, Note: item.Note,
		})
	}
	options := detail.NotificationOptions
	if options == nil {
		options = []string{}
	}
	return detailResponse{
		summaryResponse: projectSummary(detail.Summary),
		Contact:         contactResponse{Name: detail.Contact.Name, MaskedPhone: detail.Contact.MaskedPhone},
		Identity:        identityResponse{Kind: detail.Identity.Kind}, Discount: discountResponse{RatePercent: detail.Discount.RatePercent},
		Items: items, TransactionID: detail.TransactionID, PaidAt: detail.PaidAt.UTC().Format(time.RFC3339Nano),
		RedemptionToken: detail.RedemptionToken, TransitionTimes: projectTransitionTimes(detail.TransitionTimes),
		NotificationOptions: options, OrderNote: detail.OrderNote,
	}
}

// WriteOrderResponse is the single frozen OrderDetail HTTP projection reused by fulfillment commands.
func WriteOrderResponse(ctx *gin.Context, detail Detail) bool {
	if !validDetail(detail) {
		return false
	}
	ctx.JSON(http.StatusOK, detailEnvelope{Order: projectDetail(detail)})
	return true
}
func projectTransitionTimes(value TransitionTimes) transitionTimesResponse {
	response := transitionTimesResponse{}
	if !value.PreparingAt.IsZero() {
		response.PreparingAt = value.PreparingAt.UTC().Format(time.RFC3339Nano)
	}
	if !value.ReadyAt.IsZero() {
		response.ReadyAt = value.ReadyAt.UTC().Format(time.RFC3339Nano)
	}
	if !value.CompletedAt.IsZero() {
		response.CompletedAt = value.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if !value.RefundingAt.IsZero() {
		response.RefundingAt = value.RefundingAt.UTC().Format(time.RFC3339Nano)
	}
	if !value.RefundedAt.IsZero() {
		response.RefundedAt = value.RefundedAt.UTC().Format(time.RFC3339Nano)
	}
	return response
}

func writeNotFound(ctx *gin.Context) {
	writeError(ctx, http.StatusNotFound, "ORDER_NOT_FOUND", "order not found")
}
func writeUnauthenticated(ctx *gin.Context) {
	writeError(ctx, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
}
func writeUnavailable(ctx *gin.Context) {
	writeError(ctx, http.StatusServiceUnavailable, "ORDERS_UNAVAILABLE", "orders temporarily unavailable")
}
func writeError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorEnvelope{Error: errorResponse{Code: code, Message: message}})
}
