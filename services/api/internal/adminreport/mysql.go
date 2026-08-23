package adminreport

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type CommandAdapter interface {
	ProcessPending(context.Context, WriteMeta, uint64, PendingAction, string) (any, error)
	RequestRefund(context.Context, WriteMeta, uint64, string) (Order, Refund, error)
	Advance(context.Context, WriteMeta, uint64) (Order, error)
}
type MySQLApplication struct {
	db       *sql.DB
	commands CommandAdapter
}

func NewMySQLApplication(db *sql.DB, commands CommandAdapter) *MySQLApplication {
	return &MySQLApplication{db: db, commands: commands}
}
func (a *MySQLApplication) ProcessPending(ctx context.Context, m WriteMeta, id uint64, act PendingAction, reason string) (any, error) {
	if a.commands == nil {
		return nil, ErrUnavailable
	}
	return a.commands.ProcessPending(ctx, m, id, act, reason)
}
func (a *MySQLApplication) RequestRefund(ctx context.Context, m WriteMeta, id uint64, reason string) (Order, Refund, error) {
	if a.commands == nil {
		return Order{}, Refund{}, ErrUnavailable
	}
	return a.commands.RequestRefund(ctx, m, id, reason)
}
func (a *MySQLApplication) Advance(ctx context.Context, m WriteMeta, id uint64) (Order, error) {
	if a.commands == nil {
		return Order{}, ErrUnavailable
	}
	return a.commands.Advance(ctx, m, id)
}
func authorizeOwner(ctx context.Context, db *sql.DB, userID uint64) error {
	var id uint64
	err := db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}
func (a *MySQLApplication) SearchOrders(ctx context.Context, userID uint64, f OrderFilter) ([]Order, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	query := `SELECT id,CONVERT(order_no USING utf8mb4),state,DATE_FORMAT(pickup_date,'%Y-%m-%d'),TIME_FORMAT(pickup_time,'%H:%i'),meal_period,pickup_point_snapshot,LPAD(pickup_number,4,'0'),payable_cents,original_subtotal_cents,discount_cents,discount_rate_percent,contact_name_snapshot,CONVERT(contact_phone_snapshot USING ascii),CONVERT(transaction_id USING utf8mb4),paid_at,materialized_at FROM orders WHERE id>?`
	args := []any{f.Page.AfterID}
	if f.State != "" {
		query += ` AND state=?`
		args = append(args, normalizeOrderState(f.State))
	}
	if f.Date != "" {
		query += ` AND pickup_date=?`
		args = append(args, f.Date)
	}
	if f.Unclaimed {
		query += ` AND state='READY_FOR_PICKUP' AND pickup_date<DATE(CONVERT_TZ(UTC_TIMESTAMP(6),'+00:00','+08:00'))`
	}
	if f.Query != "" {
		query += ` AND (CONVERT(order_no USING utf8mb4) LIKE CONCAT('%',?,'%') OR LPAD(pickup_number,4,'0')=? OR CONVERT(contact_phone_snapshot USING ascii) LIKE CONCAT('%',?,'%'))`
		args = append(args, f.Query, f.Query, f.Query)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, f.Page.Limit)
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	out := []Order{}
	ids := []uint64{}
	for rows.Next() {
		var o Order
		var phone, state string
		if rows.Scan(&o.ID, &o.OrderNo, &state, &o.PickupDate, &o.PickupTime, &o.MealPeriod, &o.PickupPoint, &o.PickupNumber, &o.PayableCents, &o.SubtotalCents, &o.DiscountCents, &o.DiscountRate, &o.ContactName, &phone, &o.TransactionID, &o.PaidAt, &o.MaterializedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		o.State = stateLabel(state)
		o.MaskedPhone = mask(phone)
		o.AvailableActions = actions(state)
		out = append(out, o)
		ids = append(ids, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	if err := a.loadItems(ctx, out, ids); err != nil {
		return nil, 0, err
	}
	next := uint64(0)
	if len(out) == int(f.Page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func (a *MySQLApplication) loadItems(ctx context.Context, orders []Order, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	index := map[uint64]int{}
	for i, id := range ids {
		args[i] = id
		index[id] = i
	}
	rows, err := a.db.QueryContext(ctx, `SELECT order_id,product_id,product_name_snapshot,quantity,payable_subtotal_cents FROM order_items WHERE order_id IN (`+marks+`) ORDER BY order_id,line_number`, args...)
	if err != nil {
		return ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var orderID uint64
		var item Item
		if rows.Scan(&orderID, &item.ProductID, &item.Name, &item.Quantity, &item.LineTotalCents) != nil {
			return ErrUnavailable
		}
		i, ok := index[orderID]
		if !ok {
			return ErrUnavailable
		}
		orders[i].Items = append(orders[i].Items, item)
	}
	return rows.Err()
}
func (a *MySQLApplication) Stats(ctx context.Context, userID uint64, at time.Time) (Stats, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return Stats{}, err
	}
	var s Stats
	err := a.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(IF(DATE(paid_at)=DATE(?) AND state NOT IN ('REFUNDING','REFUNDED'),payable_cents,0)),0),COALESCE(SUM(DATE(paid_at)=DATE(?) AND state NOT IN ('REFUNDING','REFUNDED')),0),COALESCE(SUM(IF(YEAR(paid_at)=YEAR(?) AND MONTH(paid_at)=MONTH(?) AND state NOT IN ('REFUNDING','REFUNDED'),payable_cents,0)),0),COALESCE(SUM(YEAR(paid_at)=YEAR(?) AND MONTH(paid_at)=MONTH(?) AND state NOT IN ('REFUNDING','REFUNDED')),0),COALESCE(SUM(state='PREPARING'),0) FROM orders`, at, at, at, at, at, at).Scan(&s.TodayRevenueCents, &s.TodayOrders, &s.MonthRevenueCents, &s.MonthOrders, &s.PendingProduction)
	if err != nil {
		return Stats{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_cents),0) FROM refunds WHERE materialization_state='APPLIED' AND DATE(materialized_at)=DATE(?)`, at).Scan(&s.RefundCents) != nil {
		return Stats{}, ErrUnavailable
	}
	rows, err := a.db.QueryContext(ctx, `SELECT oi.product_id,oi.product_name_snapshot,SUM(oi.quantity) FROM order_items oi JOIN orders o ON o.id=oi.order_id WHERE DATE(o.paid_at)=DATE(?) AND o.state NOT IN ('REFUNDING','REFUNDED') GROUP BY oi.product_id,oi.product_name_snapshot ORDER BY SUM(oi.quantity) DESC,oi.product_id LIMIT 10`, at)
	if err != nil {
		return Stats{}, ErrUnavailable
	}
	for rows.Next() {
		var p ProductSale
		if rows.Scan(&p.ProductID, &p.Name, &p.Quantity) != nil {
			rows.Close()
			return Stats{}, ErrUnavailable
		}
		s.ProductSales = append(s.ProductSales, p)
	}
	if rows.Err() != nil {
		rows.Close()
		return Stats{}, ErrUnavailable
	}
	rows.Close()
	return s, nil
}
func (a *MySQLApplication) Payments(ctx context.Context, userID uint64, r BillingRange, p PageQuery) ([]Payment, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT id,CONVERT(order_no USING utf8mb4),state,DATE_FORMAT(pickup_date,'%Y-%m-%d'),TIME_FORMAT(pickup_time,'%H:%i'),meal_period,pickup_point_snapshot,LPAD(pickup_number,4,'0'),payable_cents,original_subtotal_cents,discount_cents,discount_rate_percent,contact_name_snapshot,CONVERT(contact_phone_snapshot USING ascii),CONVERT(transaction_id USING utf8mb4),paid_at,materialized_at FROM orders WHERE id>? AND DATE(paid_at) BETWEEN ? AND ? ORDER BY id LIMIT ?`, p.AfterID, r.From, r.To, p.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	orders := []Order{}
	ids := []uint64{}
	for rows.Next() {
		var o Order
		var state, phone string
		if rows.Scan(&o.ID, &o.OrderNo, &state, &o.PickupDate, &o.PickupTime, &o.MealPeriod, &o.PickupPoint, &o.PickupNumber, &o.PayableCents, &o.SubtotalCents, &o.DiscountCents, &o.DiscountRate, &o.ContactName, &phone, &o.TransactionID, &o.PaidAt, &o.MaterializedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		o.State = stateLabel(state)
		o.MaskedPhone = mask(phone)
		orders = append(orders, o)
		ids = append(ids, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	if err = a.loadItems(ctx, orders, ids); err != nil {
		return nil, 0, err
	}
	out := make([]Payment, 0, len(orders))
	for _, o := range orders {
		out = append(out, Payment{Order: o})
	}
	next := uint64(0)
	if len(out) == int(p.Limit) {
		next = out[len(out)-1].Order.ID
	}
	return out, next, nil
}
func (a *MySQLApplication) Refunds(ctx context.Context, userID uint64, r BillingRange, p PageQuery) ([]Refund, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT r.id,COALESCE(r.order_id,0),COALESCE(CONVERT(o.order_no USING utf8mb4),''),r.provider_state,COALESCE(CONVERT(r.provider_refund_id USING utf8mb4),''),COALESCE(CONVERT(o.transaction_id USING utf8mb4),''),COALESCE(a.name,'系统'),r.reason_code,r.amount_cents,COALESCE(o.paid_at,r.created_at),r.created_at FROM refunds r LEFT JOIN orders o ON o.id=r.order_id LEFT JOIN merchant_accounts a ON a.id=r.requested_by_account_id WHERE r.id>? AND DATE(r.created_at) BETWEEN ? AND ? ORDER BY r.id LIMIT ?`, p.AfterID, r.From, r.To, p.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	out := []Refund{}
	for rows.Next() {
		var x Refund
		if rows.Scan(&x.ID, &x.OrderID, &x.OrderNo, &x.State, &x.ProviderRefundID, &x.TransactionID, &x.Operator, &x.Reason, &x.AmountCents, &x.PaidAt, &x.RequestedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		x.State = refundStateLabel(x.State)
		out = append(out, x)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	next := uint64(0)
	if len(out) == int(p.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func (a *MySQLApplication) Summary(ctx context.Context, userID uint64, r BillingRange) (Summary, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return Summary{}, err
	}
	var s Summary
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(payable_cents),0),COALESCE(SUM(identity_kind='STAFF'),0),COALESCE(SUM(discount_cents),0) FROM orders WHERE DATE(paid_at) BETWEEN ? AND ?`, r.From, r.To).Scan(&s.Count, &s.GrossCents, &s.StaffCount, &s.DiscountCents) != nil {
		return Summary{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(amount_cents),0),COALESCE(SUM(materialization_state<>'APPLIED'),0) FROM refunds WHERE DATE(created_at) BETWEEN ? AND ?`, r.From, r.To).Scan(&s.RefundCount, &s.RefundCents, &s.PendingCount) != nil {
		return Summary{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(expected_amount_cents),0) FROM prepayments WHERE materialization_state='PENDING_MANUAL' AND DATE(created_at) BETWEEN ? AND ?`, r.From, r.To).Scan(&s.UnbuiltCount, &s.UnbuiltCents) != nil {
		return Summary{}, ErrUnavailable
	}
	s.NetCents = int64(s.GrossCents) - int64(s.RefundCents)
	return s, nil
}
func (a *MySQLApplication) ExportCSV(ctx context.Context, userID uint64, r BillingRange) (io.ReadCloser, error) {
	items := []Payment{}
	cursor := uint64(0)
	for {
		page, next, err := a.Payments(ctx, userID, r, PageQuery{AfterID: cursor, Limit: 100})
		if err != nil {
			return nil, err
		}
		items = append(items, page...)
		if next == 0 {
			break
		}
		cursor = next
	}
	var b strings.Builder
	b.WriteString("\ufeff")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"订单号", "取餐号", "支付时间", "微信交易号", "实付金额"})
	for _, p := range items {
		_ = w.Write([]string{p.Order.OrderNo, p.Order.PickupNumber, p.Order.PaidAt.Format(time.RFC3339), p.Order.TransactionID, fmt.Sprintf("%d.%02d", p.Order.PayableCents/100, p.Order.PayableCents%100)})
	}
	w.Flush()
	if w.Error() != nil {
		return nil, ErrUnavailable
	}
	return io.NopCloser(strings.NewReader(b.String())), nil
}
func (a *MySQLApplication) ListPending(ctx context.Context, userID uint64, p PageQuery) ([]Pending, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT p.id,p.quote_id,CONVERT(p.out_trade_no USING utf8mb4),p.materialization_state,p.pending_reason_code,DATE_FORMAT(q.pickup_date,'%Y-%m-%d'),TIME_FORMAT(q.pickup_time,'%H:%i'),q.meal_period,q.contact_name_snapshot,CONVERT(q.contact_phone_snapshot USING ascii),p.expected_amount_cents,COALESCE((SELECT MIN(po.success_time) FROM payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH'),p.updated_at),COALESCE((SELECT CONVERT(po.transaction_id USING utf8mb4) FROM payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH' ORDER BY po.id LIMIT 1),'') FROM prepayments p JOIN quotes q ON q.id=p.quote_id WHERE p.materialization_state='PENDING_MANUAL' AND p.id>? ORDER BY p.id LIMIT ?`, p.AfterID, p.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	out := []Pending{}
	quoteIDs := []uint64{}
	for rows.Next() {
		var x Pending
		var quoteID uint64
		var phone string
		if rows.Scan(&x.ID, &quoteID, &x.OutTradeNo, &x.State, &x.BlockingReason, &x.PickupDate, &x.PickupTime, &x.MealPeriod, &x.ContactName, &phone, &x.AmountCents, &x.PaidAt, &x.TransactionID) != nil {
			return nil, 0, ErrUnavailable
		}
		x.MaskedPhone = mask(phone)
		out = append(out, x)
		quoteIDs = append(quoteIDs, quoteID)
	}
	if len(quoteIDs) > 0 {
		marks := strings.TrimSuffix(strings.Repeat("?,", len(quoteIDs)), ",")
		args := make([]any, len(quoteIDs))
		index := map[uint64]int{}
		for i, id := range quoteIDs {
			args[i] = id
			index[id] = i
		}
		itemRows, e := a.db.QueryContext(ctx, `SELECT quote_id,product_id,product_name_snapshot,quantity,payable_subtotal_cents FROM quote_items WHERE quote_id IN (`+marks+`) ORDER BY quote_id,line_number`, args...)
		if e != nil {
			return nil, 0, ErrUnavailable
		}
		for itemRows.Next() {
			var quoteID uint64
			var item Item
			if itemRows.Scan(&quoteID, &item.ProductID, &item.Name, &item.Quantity, &item.LineTotalCents) != nil {
				itemRows.Close()
				return nil, 0, ErrUnavailable
			}
			out[index[quoteID]].Items = append(out[index[quoteID]].Items, item)
		}
		if itemRows.Err() != nil {
			itemRows.Close()
			return nil, 0, ErrUnavailable
		}
		itemRows.Close()
	}
	next := uint64(0)
	if len(out) == int(p.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func normalizeOrderState(v string) string {
	return map[string]string{"已预约": "RESERVED", "制作中": "PREPARING", "待取餐": "READY_FOR_PICKUP", "已完成": "COMPLETED", "退款中": "REFUNDING", "已退款": "REFUNDED"}[v]
}
func stateLabel(v string) string {
	return map[string]string{"RESERVED": "已预约", "PREPARING": "制作中", "READY_FOR_PICKUP": "待取餐", "COMPLETED": "已完成", "REFUNDING": "退款中", "REFUNDED": "已退款"}[v]
}
func refundStateLabel(v string) string {
	return map[string]string{"READY": "退款中", "CREATE_CLAIMED": "退款中", "CREATE_UNKNOWN": "退款中", "PROCESSING": "退款中", "SUCCESS": "已退款", "CLOSED": "退款失败"}[v]
}
func actions(v string) []string {
	switch v {
	case "PREPARING":
		return []string{"MARK_READY", "REFUND"}
	case "READY_FOR_PICKUP":
		return []string{"REDEEM", "REFUND"}
	case "RESERVED", "COMPLETED":
		return []string{"REFUND"}
	default:
		return []string{}
	}
}
func mask(phone string) string {
	digits := strings.TrimPrefix(phone, "+86")
	if len(digits) < 7 {
		return "***"
	}
	return digits[:3] + "****" + digits[len(digits)-4:]
}
