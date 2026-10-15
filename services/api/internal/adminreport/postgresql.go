package adminreport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type CommandAdapter interface {
	ProcessPending(context.Context, WriteMeta, uint64, PendingAction, string) (PendingResult, error)
	RequestRefund(context.Context, WriteMeta, uint64, string) (Order, Refund, error)
}
type PostgreSQLApplication struct {
	db       *sql.DB
	commands CommandAdapter
}

func NewPostgreSQLApplication(db *sql.DB, commands CommandAdapter) *PostgreSQLApplication {
	return &PostgreSQLApplication{db: db, commands: commands}
}
func (a *PostgreSQLApplication) ProcessPending(ctx context.Context, m WriteMeta, id uint64, act PendingAction, reason string) (PendingResult, error) {
	if a.commands == nil {
		return PendingResult{}, ErrUnavailable
	}
	return a.commands.ProcessPending(ctx, m, id, act, reason)
}
func (a *PostgreSQLApplication) RequestRefund(ctx context.Context, m WriteMeta, id uint64, reason string) (Order, Refund, error) {
	if a.commands == nil {
		return Order{}, Refund{}, ErrUnavailable
	}
	return a.commands.RequestRefund(ctx, m, id, reason)
}
func authorizeOwner(ctx context.Context, db *sql.DB, userID uint64) error {
	var id uint64
	err := db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

const orderProjectionColumns = `id,convert_from(order_no,'UTF8'),(CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END),to_char(pickup_date,'YYYY-MM-DD'),to_char(pickup_date+pickup_time,'HH24:MI'),meal_period,pickup_point_snapshot,LPAD(pickup_number::text,4,'0'),payable_cents,original_subtotal_cents,discount_cents,discount_rate_percent,contact_name_snapshot,convert_from(contact_phone_snapshot,'UTF8'),convert_from(transaction_id,'UTF8'),paid_at,materialized_at,order_note`

type orderScanner interface{ Scan(...any) error }

func scanOrderProjection(scanner orderScanner) (Order, error) {
	var order Order
	var phone, state string
	if err := scanner.Scan(&order.ID, &order.OrderNo, &state, &order.PickupDate, &order.PickupTime, &order.MealPeriod, &order.PickupPoint, &order.PickupNumber, &order.PayableCents, &order.SubtotalCents, &order.DiscountCents, &order.DiscountRate, &order.ContactName, &phone, &order.TransactionID, &order.PaidAt, &order.MaterializedAt, &order.OrderNote); err != nil {
		return Order{}, err
	}
	order.State = stateLabel(state)
	order.MaskedPhone = mask(phone)
	order.AvailableActions = actions(state)
	return order, nil
}

func (a *PostgreSQLApplication) GetOrder(ctx context.Context, userID, orderID uint64) (Order, error) {
	if a == nil || a.db == nil || userID == 0 || orderID == 0 {
		return Order{}, ErrInvalidInput
	}
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return Order{}, err
	}
	order, err := scanOrderProjection(a.db.QueryRowContext(ctx, `SELECT `+orderProjectionColumns+` FROM orders WHERE id=$1`, orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, ErrUnavailable
	}
	orders := []Order{order}
	if err := a.loadItems(ctx, orders, []uint64{orderID}); err != nil {
		return Order{}, err
	}
	return orders[0], nil
}

func (a *PostgreSQLApplication) SearchOrders(ctx context.Context, userID uint64, f OrderFilter) ([]Order, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + orderProjectionColumns + ` FROM orders WHERE id>$1`
	args := []any{f.Page.AfterID}
	if f.State != "" {
		args = append(args, normalizeOrderState(f.State))
		query += fmt.Sprintf(" AND (CASE WHEN state='RESERVED' AND pickup_at<=statement_timestamp()+interval '30 minutes' THEN 'PREPARING' ELSE state END)=$%d", len(args))
	}
	if f.Date != "" {
		args = append(args, f.Date)
		query += fmt.Sprintf(" AND pickup_date=$%d", len(args))
	}
	if f.Unclaimed {
		query += " AND state='READY_FOR_PICKUP' AND pickup_date<(clock_timestamp() AT TIME ZONE 'Asia/Shanghai')::date"
	}
	if f.Query != "" {
		escaped := strings.NewReplacer("=", "==", "%", "=%", "_", "=_").Replace(f.Query)
		args = append(args, "%"+escaped+"%", f.Query, "%"+escaped+"%")
		query += fmt.Sprintf(" AND (convert_from(order_no,'UTF8') LIKE $%d ESCAPE '=' OR LPAD(pickup_number::text,4,'0')=$%d OR convert_from(contact_phone_snapshot,'UTF8') LIKE $%d ESCAPE '=')", len(args)-2, len(args)-1, len(args))
	}
	args = append(args, f.Page.Limit)
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d", len(args))
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	out := []Order{}
	ids := []uint64{}
	for rows.Next() {
		o, err := scanOrderProjection(rows)
		if err != nil {
			return nil, 0, ErrUnavailable
		}
		out = append(out, o)
		ids = append(ids, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	if err := rows.Close(); err != nil {
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
func (a *PostgreSQLApplication) loadItems(ctx context.Context, orders []Order, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	marks := make([]string, len(ids))
	args := make([]any, len(ids))
	index := map[uint64]int{}
	for i, id := range ids {
		args[i] = id
		marks[i] = fmt.Sprintf("$%d", i+1)
		index[id] = i
	}
	rows, err := a.db.QueryContext(ctx, `SELECT order_id,product_id,product_name_snapshot,quantity,payable_subtotal_cents,flavors_json,line_note FROM order_items WHERE order_id IN (`+strings.Join(marks, ",")+`) ORDER BY order_id,line_number`, args...)
	if err != nil {
		return ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var orderID uint64
		var item Item
		var rawFlavors []byte
		if rows.Scan(&orderID, &item.ProductID, &item.Name, &item.Quantity, &item.LineTotalCents, &rawFlavors, &item.Note) != nil {
			return ErrUnavailable
		}
		if json.Unmarshal(rawFlavors, &item.Flavors) != nil || item.Flavors == nil {
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
func (a *PostgreSQLApplication) Stats(ctx context.Context, userID uint64, at time.Time) (Stats, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return Stats{}, err
	}
	var s Stats
	err := a.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(payable_cents) FILTER (WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date=($1::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND state='COMPLETED'),0),COUNT(*) FILTER (WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date=($1::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND state='COMPLETED'),COALESCE(SUM(payable_cents) FILTER (WHERE date_trunc('month',paid_at AT TIME ZONE 'Asia/Shanghai')=date_trunc('month',$1::timestamptz AT TIME ZONE 'Asia/Shanghai') AND state='COMPLETED'),0),COUNT(*) FILTER (WHERE date_trunc('month',paid_at AT TIME ZONE 'Asia/Shanghai')=date_trunc('month',$1::timestamptz AT TIME ZONE 'Asia/Shanghai') AND state='COMPLETED'),COUNT(*) FILTER (WHERE (CASE WHEN state='RESERVED' AND pickup_at<=$1::timestamptz+interval '30 minutes' THEN 'PREPARING' ELSE state END)='PREPARING') FROM orders WHERE (CASE WHEN state='RESERVED' AND pickup_at<=$1::timestamptz+interval '30 minutes' THEN 'PREPARING' ELSE state END)='PREPARING' OR (state='COMPLETED' AND paid_at >= (date_trunc('month',$1::timestamptz AT TIME ZONE 'Asia/Shanghai') AT TIME ZONE 'Asia/Shanghai') AND paid_at < ((date_trunc('month',$1::timestamptz AT TIME ZONE 'Asia/Shanghai') + interval '1 month') AT TIME ZONE 'Asia/Shanghai'))`, at).Scan(&s.TodayRevenueCents, &s.TodayOrders, &s.MonthRevenueCents, &s.MonthOrders, &s.PendingProduction)
	if err != nil {
		return Stats{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(r.amount_cents),0) FROM refunds r JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.materialization_state='APPLIED' AND (observed.success_time AT TIME ZONE 'Asia/Shanghai')::date=($1::timestamptz AT TIME ZONE 'Asia/Shanghai')::date`, at).Scan(&s.RefundCents) != nil {
		return Stats{}, ErrUnavailable
	}
	rows, err := a.db.QueryContext(ctx, `SELECT oi.product_id,oi.product_name_snapshot,SUM(oi.quantity) FROM order_items oi JOIN orders o ON o.id=oi.order_id WHERE (o.paid_at AT TIME ZONE 'Asia/Shanghai')::date=($1::timestamptz AT TIME ZONE 'Asia/Shanghai')::date AND o.state='COMPLETED' GROUP BY oi.product_id,oi.product_name_snapshot ORDER BY SUM(oi.quantity) DESC,oi.product_id LIMIT 10`, at)
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
func (a *PostgreSQLApplication) Payments(ctx context.Context, userID uint64, r BillingRange, p PageQuery) ([]Payment, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT `+orderProjectionColumns+` FROM orders WHERE id>$1 AND (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $2 AND $3 ORDER BY id LIMIT $4`, p.AfterID, r.From, r.To, p.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	orders := []Order{}
	ids := []uint64{}
	for rows.Next() {
		o, err := scanOrderProjection(rows)
		if err != nil {
			return nil, 0, ErrUnavailable
		}
		orders = append(orders, o)
		ids = append(ids, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	if err := rows.Close(); err != nil {
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
func (a *PostgreSQLApplication) Refunds(ctx context.Context, userID uint64, r BillingRange, p PageQuery) ([]Refund, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT r.id,COALESCE(r.order_id,0),COALESCE(convert_from(o.order_no,'UTF8'),''),r.provider_state,COALESCE(convert_from(r.provider_refund_id,'UTF8'),''),COALESCE(convert_from(o.transaction_id,'UTF8'),''),COALESCE(a.name,'系统'),r.reason_code,r.amount_cents,COALESCE(o.paid_at,r.created_at),r.created_at FROM refunds r LEFT JOIN orders o ON o.id=r.order_id LEFT JOIN merchant_accounts a ON a.id=r.requested_by_account_id LEFT JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.id>$1 AND ((CASE WHEN r.materialization_state='APPLIED' THEN observed.success_time ELSE r.created_at END) AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $2 AND $3 ORDER BY r.id LIMIT $4`, p.AfterID, r.From, r.To, p.Limit)
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
func (a *PostgreSQLApplication) Summary(ctx context.Context, userID uint64, r BillingRange) (Summary, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return Summary{}, err
	}
	var s Summary
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(payable_cents),0),COUNT(*) FILTER (WHERE identity_kind='STAFF'),COALESCE(SUM(discount_cents),0) FROM orders WHERE (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $1 AND $2`, r.From, r.To).Scan(&s.Count, &s.GrossCents, &s.StaffCount, &s.DiscountCents) != nil {
		return Summary{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(r.amount_cents),0),COUNT(*) FILTER (WHERE r.materialization_state<>'APPLIED') FROM refunds r LEFT JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE ((CASE WHEN r.materialization_state='APPLIED' THEN observed.success_time ELSE r.created_at END) AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $1 AND $2`, r.From, r.To).Scan(&s.RefundCount, &s.RefundCents, &s.PendingCount) != nil {
		return Summary{}, ErrUnavailable
	}
	if a.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(expected_amount_cents),0) FROM prepayments WHERE materialization_state='PENDING_MANUAL' AND (created_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $1 AND $2`, r.From, r.To).Scan(&s.UnbuiltCount, &s.UnbuiltCents) != nil {
		return Summary{}, ErrUnavailable
	}
	s.NetCents = int64(s.GrossCents) - int64(s.RefundCents)
	return s, nil
}
func (a *PostgreSQLApplication) ExportCSV(ctx context.Context, userID uint64, r BillingRange) (io.ReadCloser, error) {
	return exportCSV(ctx, func(ctx context.Context, after uint64) ([]exportPayment, error) {
		if err := authorizeOwner(ctx, a.db, userID); err != nil {
			return nil, err
		}
		rows, err := a.db.QueryContext(ctx, `SELECT id,convert_from(order_no,'UTF8'),LPAD(pickup_number::text,4,'0'),paid_at,convert_from(transaction_id,'UTF8'),payable_cents FROM orders WHERE id>$1 AND (paid_at AT TIME ZONE 'Asia/Shanghai')::date BETWEEN $2 AND $3 ORDER BY id LIMIT 100`, after, r.From, r.To)
		if err != nil {
			return nil, ErrUnavailable
		}
		defer rows.Close()
		out := []exportPayment{}
		for rows.Next() {
			var p exportPayment
			if rows.Scan(&p.ID, &p.OrderNo, &p.PickupNumber, &p.PaidAt, &p.TransactionID, &p.PayableCents) != nil {
				return nil, ErrUnavailable
			}
			out = append(out, p)
		}
		if rows.Err() != nil {
			return nil, ErrUnavailable
		}
		return out, nil
	})
}
func (a *PostgreSQLApplication) ListPending(ctx context.Context, userID uint64, p PageQuery) ([]Pending, uint64, error) {
	if err := authorizeOwner(ctx, a.db, userID); err != nil {
		return nil, 0, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT p.id,p.quote_id,convert_from(p.out_trade_no,'UTF8'),p.materialization_state,p.pending_reason_code,to_char(q.pickup_date,'YYYY-MM-DD'),to_char(q.pickup_date+q.pickup_time,'HH24:MI'),q.meal_period,q.contact_name_snapshot,convert_from(q.contact_phone_snapshot,'UTF8'),p.expected_amount_cents,COALESCE((SELECT MIN(po.success_time) FROM payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH'),p.updated_at),COALESCE((SELECT convert_from(po.transaction_id,'UTF8') FROM payment_observations po WHERE po.prepayment_id=p.id AND po.provider_state='PAID' AND po.validation='MATCH' ORDER BY po.id LIMIT 1),'') FROM prepayments p JOIN quotes q ON q.id=p.quote_id WHERE p.materialization_state='PENDING_MANUAL' AND p.id>$1 ORDER BY p.id LIMIT $2`, p.AfterID, p.Limit)
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
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	if err := rows.Close(); err != nil {
		return nil, 0, ErrUnavailable
	}
	if len(quoteIDs) > 0 {
		marks := make([]string, len(quoteIDs))
		args := make([]any, len(quoteIDs))
		index := map[uint64]int{}
		for i, id := range quoteIDs {
			args[i] = id
			marks[i] = fmt.Sprintf("$%d", i+1)
			index[id] = i
		}
		itemRows, e := a.db.QueryContext(ctx, `SELECT quote_id,product_id,product_name_snapshot,quantity,payable_subtotal_cents FROM quote_items WHERE quote_id IN (`+strings.Join(marks, ",")+`) ORDER BY quote_id,line_number`, args...)
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
