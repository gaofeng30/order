package adminreport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidInput = errors.New("admin report invalid input")
	ErrNotFound     = errors.New("admin report not found")
	ErrForbidden    = errors.New("admin report forbidden")
	ErrConflict     = errors.New("admin report conflict")
	ErrUnavailable  = errors.New("admin report unavailable")
)

type WriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type PageQuery struct {
	AfterID uint64
	Limit   uint16
}
type OrderFilter struct {
	State, Date, Query string
	Unclaimed          bool
	Page               PageQuery
}
type Item struct {
	ProductID      uint64
	Name           string
	Quantity       uint32
	LineTotalCents uint64
}
type Order struct {
	ID                                                                uint64
	OrderNo, State, PickupDate, PickupTime, PickupPoint, PickupNumber string
	MealPeriod                                                        string
	PayableCents, SubtotalCents, DiscountCents                        uint64
	DiscountRate                                                      uint8
	ContactName, MaskedPhone, TransactionID                           string
	PaidAt, MaterializedAt                                            time.Time
	Items                                                             []Item
	AvailableActions                                                  []string
}
type Pending struct {
	ID                                                                                                             uint64
	OutTradeNo, State, BlockingReason, PickupDate, PickupTime, MealPeriod, ContactName, MaskedPhone, TransactionID string
	AmountCents                                                                                                    uint64
	PaidAt                                                                                                         time.Time
	Items                                                                                                          []Item
}
type Stats struct {
	TodayRevenueCents, MonthRevenueCents, RefundCents uint64
	TodayOrders, MonthOrders, PendingProduction       uint32
	ProductSales                                      []ProductSale
}
type ProductSale struct {
	ProductID uint64
	Name      string
	Quantity  uint32
}
type BillingRange struct{ From, To string }
type Payment struct{ Order Order }
type Refund struct {
	ID                                                                uint64
	OrderID                                                           uint64
	OrderNo, State, ProviderRefundID, TransactionID, Operator, Reason string
	AmountCents                                                       uint64
	PaidAt, RequestedAt                                               time.Time
}
type Summary struct {
	Count, RefundCount, PendingCount, UnbuiltCount, StaffCount uint32
	GrossCents, RefundCents, UnbuiltCents, DiscountCents       uint64
	NetCents                                                   int64
}
type PendingAction string

const (
	Materialize PendingAction = "MATERIALIZE"
	RefundPaid  PendingAction = "REFUND"
)

// Application is an Adapter over Order/PaymentOrder/Refund/Fulfillment plus derived Billing reads.
// It never owns duplicate transaction facts or summary tables.
type Application interface {
	SearchOrders(context.Context, uint64, OrderFilter) ([]Order, uint64, error)
	Stats(context.Context, uint64, time.Time) (Stats, error)
	Payments(context.Context, uint64, BillingRange, PageQuery) ([]Payment, uint64, error)
	Refunds(context.Context, uint64, BillingRange, PageQuery) ([]Refund, uint64, error)
	Summary(context.Context, uint64, BillingRange) (Summary, error)
	ExportCSV(context.Context, uint64, BillingRange) (io.ReadCloser, error)
	ListPending(context.Context, uint64, PageQuery) ([]Pending, uint64, error)
	ProcessPending(context.Context, WriteMeta, uint64, PendingAction, string) (any, error)
	RequestRefund(context.Context, WriteMeta, uint64, string) (Order, Refund, error)
	Advance(context.Context, WriteMeta, uint64) (Order, error)
}
type Handler struct {
	app Application
	now func() time.Time
}

func NewHandler(app Application) *Handler { return &Handler{app: app, now: time.Now} }
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/orders", h.orders)
	group.GET("/stats", h.stats)
	group.GET("/finance/payments", h.payments)
	group.GET("/finance/refunds", h.refunds)
	group.GET("/finance/summary", h.summary)
	group.GET("/finance/export", h.export)
	group.GET("/pending-payments", h.pending)
	group.POST("/pending-payments/:id", h.processPending)
	group.POST("/orders/:id/refund", h.refund)
	group.PUT("/orders/:id/advance", h.advance)
}

func actor(c *gin.Context) (uint64, bool) { id := c.GetUint64("actor_user_id"); return id, id > 0 }
func page(c *gin.Context) (PageQuery, bool) {
	q := PageQuery{Limit: 50}
	if raw := c.Query("after_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return PageQuery{}, false
		}
		q.AfterID = id
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil || n < 1 || n > 100 {
			return PageQuery{}, false
		}
		q.Limit = uint16(n)
	}
	return q, true
}
func (h *Handler) orders(c *gin.Context) {
	id, ok := actor(c)
	p, pok := page(c)
	if !ok || !pok {
		writeError(c, ErrInvalidInput)
		return
	}
	items, next, err := h.app.SearchOrders(c.Request.Context(), id, OrderFilter{c.Query("state"), c.Query("date"), c.Query("q"), c.Query("unclaimed") == "true", p})
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, orderView(item))
	}
	c.JSON(http.StatusOK, gin.H{"orders": out, "next_after_id": optionalID(next)})
}
func (h *Handler) stats(c *gin.Context) {
	id, ok := actor(c)
	if !ok {
		writeError(c, ErrForbidden)
		return
	}
	s, err := h.app.Stats(c.Request.Context(), id, h.now())
	if err != nil {
		writeError(c, err)
		return
	}
	sales := make([]gin.H, 0, len(s.ProductSales))
	for _, p := range s.ProductSales {
		sales = append(sales, gin.H{"product_id": strconv.FormatUint(p.ProductID, 10), "name": p.Name, "quantity": p.Quantity})
	}
	c.JSON(http.StatusOK, gin.H{"today_revenue_cents": s.TodayRevenueCents, "today_orders": s.TodayOrders, "month_revenue_cents": s.MonthRevenueCents, "month_orders": s.MonthOrders, "refund_cents": s.RefundCents, "pending_production": s.PendingProduction, "product_sales": sales})
}
func billingRange(c *gin.Context) (BillingRange, bool) {
	r := BillingRange{c.Query("from"), c.Query("to")}
	if r.From == "" || r.To == "" || r.From > r.To {
		return BillingRange{}, false
	}
	return r, true
}
func (h *Handler) payments(c *gin.Context) {
	id, ok := actor(c)
	r, rok := billingRange(c)
	p, pok := page(c)
	if !ok || !rok || !pok {
		writeError(c, ErrInvalidInput)
		return
	}
	items, next, err := h.app.Payments(c.Request.Context(), id, r, p)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, orderView(item.Order))
	}
	c.JSON(http.StatusOK, gin.H{"payments": out, "next_after_id": optionalID(next)})
}
func (h *Handler) refunds(c *gin.Context) {
	id, ok := actor(c)
	r, rok := billingRange(c)
	p, pok := page(c)
	if !ok || !rok || !pok {
		writeError(c, ErrInvalidInput)
		return
	}
	items, next, err := h.app.Refunds(c.Request.Context(), id, r, p)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, refundView(item))
	}
	c.JSON(http.StatusOK, gin.H{"refunds": out, "next_after_id": optionalID(next)})
}
func (h *Handler) summary(c *gin.Context) {
	id, ok := actor(c)
	r, rok := billingRange(c)
	if !ok || !rok {
		writeError(c, ErrInvalidInput)
		return
	}
	s, err := h.app.Summary(c.Request.Context(), id, r)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": s.Count, "gross": s.GrossCents, "refundCount": s.RefundCount, "refundAmount": s.RefundCents, "net": s.NetCents, "pendingCount": s.PendingCount, "unbuiltCount": s.UnbuiltCount, "unbuiltAmount": s.UnbuiltCents, "staffCount": s.StaffCount, "discountCut": s.DiscountCents})
}
func (h *Handler) export(c *gin.Context) {
	id, ok := actor(c)
	r, rok := billingRange(c)
	if !ok || !rok {
		writeError(c, ErrInvalidInput)
		return
	}
	stream, err := h.app.ExportCSV(c.Request.Context(), id, r)
	if err != nil {
		writeError(c, err)
		return
	}
	defer stream.Close()
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=payments.csv")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, stream)
}
func (h *Handler) pending(c *gin.Context) {
	id, ok := actor(c)
	p, pok := page(c)
	if !ok || !pok {
		writeError(c, ErrInvalidInput)
		return
	}
	items, next, err := h.app.ListPending(c.Request.Context(), id, p)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, pendingView(item))
	}
	c.JSON(http.StatusOK, gin.H{"prepayments": out, "next_after_id": optionalID(next)})
}

type pendingWrite struct {
	Action PendingAction `json:"action"`
	Reason string        `json:"reason"`
}

func (h *Handler) processPending(c *gin.Context) {
	id, ok := uintID(c.Param("id"))
	var in pendingWrite
	if !ok || !decode(c, &in) || (in.Action != Materialize && in.Action != RefundPaid) || (in.Action == RefundPaid && strings.TrimSpace(in.Reason) == "") {
		writeError(c, ErrInvalidInput)
		return
	}
	meta, mok := meta(c)
	if !mok {
		writeError(c, ErrInvalidInput)
		return
	}
	out, err := h.app.ProcessPending(c.Request.Context(), meta, id, in.Action, strings.TrimSpace(in.Reason))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type refundWrite struct {
	Reason string `json:"reason"`
}

func (h *Handler) refund(c *gin.Context) {
	id, ok := uintID(c.Param("id"))
	var in refundWrite
	if !ok || !decode(c, &in) || strings.TrimSpace(in.Reason) == "" {
		writeError(c, ErrInvalidInput)
		return
	}
	m, mok := meta(c)
	if !mok {
		writeError(c, ErrInvalidInput)
		return
	}
	order, refund, err := h.app.RequestRefund(c.Request.Context(), m, id, strings.TrimSpace(in.Reason))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": orderView(order), "refund": refundView(refund)})
}
func (h *Handler) advance(c *gin.Context) {
	id, ok := uintID(c.Param("id"))
	if !ok || !empty(c) {
		writeError(c, ErrInvalidInput)
		return
	}
	m, mok := meta(c)
	if !mok {
		writeError(c, ErrInvalidInput)
		return
	}
	order, err := h.app.Advance(c.Request.Context(), m, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, orderView(order))
}

func orderView(o Order) gin.H {
	items := make([]gin.H, 0, len(o.Items))
	for _, i := range o.Items {
		items = append(items, gin.H{"product_id": strconv.FormatUint(i.ProductID, 10), "name": i.Name, "quantity": i.Quantity, "line_total_cents": i.LineTotalCents})
	}
	return gin.H{"id": strconv.FormatUint(o.ID, 10), "order_no": o.OrderNo, "state": o.State, "pickup_date": o.PickupDate, "pickup_time": o.PickupTime, "meal_period": o.MealPeriod, "pickup_point": o.PickupPoint, "pickup_number": o.PickupNumber, "payable_cents": o.PayableCents, "subtotal_cents": o.SubtotalCents, "discount_cents": o.DiscountCents, "discount_rate_percent": o.DiscountRate, "contact_name": o.ContactName, "phone_masked": o.MaskedPhone, "transaction_id": o.TransactionID, "paid_at": o.PaidAt, "materialized_at": o.MaterializedAt, "items": items, "available_actions": o.AvailableActions}
}
func pendingView(p Pending) gin.H {
	items := make([]gin.H, 0, len(p.Items))
	for _, i := range p.Items {
		items = append(items, gin.H{"product_id": strconv.FormatUint(i.ProductID, 10), "name": i.Name, "quantity": i.Quantity, "line_total_cents": i.LineTotalCents})
	}
	return gin.H{"id": strconv.FormatUint(p.ID, 10), "out_trade_no": p.OutTradeNo, "transaction_id": p.TransactionID, "state": p.State, "blocking_reason": p.BlockingReason, "pickup_date": p.PickupDate, "pickup_time": p.PickupTime, "meal_period": p.MealPeriod, "contact_name": p.ContactName, "phone_masked": p.MaskedPhone, "amount_cents": p.AmountCents, "paid_at": p.PaidAt, "items": items}
}
func refundView(r Refund) gin.H {
	return gin.H{"id": strconv.FormatUint(r.ID, 10), "order_id": strconv.FormatUint(r.OrderID, 10), "order_no": r.OrderNo, "state": r.State, "provider_refund_id": r.ProviderRefundID, "transaction_id": r.TransactionID, "operator": r.Operator, "reason": r.Reason, "amount_cents": r.AmountCents, "paid_at": r.PaidAt, "requested_at": r.RequestedAt}
}
func optionalID(id uint64) any {
	if id == 0 {
		return nil
	}
	return strconv.FormatUint(id, 10)
}
func uintID(v string) (uint64, bool) {
	id, err := strconv.ParseUint(v, 10, 64)
	return id, err == nil && id > 0
}
func meta(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func decode(c *gin.Context, out any) bool {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 16385))
	if err != nil || len(body) == 0 || len(body) > 16384 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return false
	}
	var x any
	return errors.Is(d.Decode(&x), io.EOF)
}
func empty(c *gin.Context) bool { var v map[string]any; return decode(c, &v) && len(v) == 0 }
func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "ADMIN_REPORT_UNAVAILABLE", "admin data temporarily unavailable"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, ErrForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "forbidden"
	case errors.Is(err, ErrConflict):
		status, code, message = http.StatusConflict, "COMMAND_CONFLICT", "command conflict"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
