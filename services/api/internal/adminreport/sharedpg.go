package adminreport

import (
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type sharedReportExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGApplication exposes only the original report and command-adapter contract.
type SharedPGApplication struct {
	client   sharedReportExecutor
	commands CommandAdapter
}

func NewSharedPGApplication(client sharedReportExecutor, commands CommandAdapter) *SharedPGApplication {
	return &SharedPGApplication{client: client, commands: commands}
}

var _ Application = (*SharedPGApplication)(nil)

type sharedReportReply struct {
	Outcome string              `json:"outcome"`
	Rows    [][]json.RawMessage `json:"rows"`
	Items   [][]json.RawMessage `json:"items"`
}

func (a *SharedPGApplication) call(ctx context.Context, user uint64, name string, p map[string]any) (sharedReportReply, error) {
	p["p_owner"] = strconv.FormatUint(user, 10)
	var r sharedReportReply
	if a.client == nil || a.client.CallRPC(ctx, name, p, &r) != nil {
		return r, ErrUnavailable
	}
	switch r.Outcome {
	case "FOUND":
		return r, nil
	case "FORBIDDEN":
		return r, ErrForbidden
	case "NOT_FOUND":
		return r, ErrNotFound
	default:
		return r, ErrUnavailable
	}
}
func reportHex(v string) string { return hex.EncodeToString([]byte(v)) }
func reportPage(p PageQuery) map[string]any {
	return map[string]any{"p_after": strconv.FormatUint(p.AfterID, 10), "p_limit": p.Limit}
}
func reportRange(r BillingRange, p PageQuery) map[string]any {
	v := reportPage(p)
	v["p_from_hex"] = reportHex(r.From)
	v["p_to_hex"] = reportHex(r.To)
	return v
}

// A fixed column row feeds the unchanged native projection scanner. SQL validates
// every native Scan before starting the next statement; Go retains labels,
// masking, flavor decoding, widths and Local timestamp reconstruction.
type sharedReportRow []json.RawMessage

func (r sharedReportRow) Scan(dest ...any) error {
	if len(r) != len(dest) {
		return ErrUnavailable
	}
	for i, d := range dest {
		raw := r[i]
		switch v := d.(type) {
		case *string:
			if string(raw) == "null" || json.Unmarshal(raw, v) != nil {
				return ErrUnavailable
			}
		case *uint64, *uint32, *uint8:
			var text string
			if json.Unmarshal(raw, &text) != nil {
				return ErrUnavailable
			}
			bits := 64
			switch d.(type) {
			case *uint32:
				bits = 32
			case *uint8:
				bits = 8
			}
			n, err := strconv.ParseUint(text, 10, bits)
			if err != nil {
				return ErrUnavailable
			}
			switch v := d.(type) {
			case *uint64:
				*v = n
			case *uint32:
				*v = uint32(n)
			case *uint8:
				*v = uint8(n)
			}
		case *time.Time:
			var at struct {
				Seconds    string `json:"seconds"`
				Nanosecond int    `json:"nanosecond"`
			}
			if json.Unmarshal(raw, &at) != nil {
				return ErrUnavailable
			}
			seconds, err := strconv.ParseInt(at.Seconds, 10, 64)
			if err != nil || at.Nanosecond < 0 || at.Nanosecond >= 1000000000 {
				return ErrUnavailable
			}
			*v = time.Unix(seconds, int64(at.Nanosecond)).In(time.Local)
		case *[]byte:
			*v = append((*v)[:0], raw...)
		default:
			return ErrUnavailable
		}
	}
	return nil
}
func sharedReportOrders(r sharedReportReply) ([]Order, error) {
	out := []Order{}
	ids := []uint64{}
	for _, row := range r.Rows {
		o, err := scanOrderProjection(sharedReportRow(row))
		if err != nil {
			return nil, ErrUnavailable
		}
		out = append(out, o)
		ids = append(ids, o.ID)
	}
	index := map[uint64]int{}
	for i, id := range ids {
		index[id] = i
	}
	for _, row := range r.Items {
		var id uint64
		var item Item
		var raw []byte
		if sharedReportRow(row).Scan(&id, &item.ProductID, &item.Name, &item.Quantity, &item.LineTotalCents, &raw, &item.Note) != nil || json.Unmarshal(raw, &item.Flavors) != nil || item.Flavors == nil {
			return nil, ErrUnavailable
		}
		i, ok := index[id]
		if !ok {
			return nil, ErrUnavailable
		}
		out[i].Items = append(out[i].Items, item)
	}
	return out, nil
}
func (a *SharedPGApplication) GetOrder(ctx context.Context, user, order uint64) (Order, error) {
	if a == nil || a.client == nil || user == 0 || order == 0 {
		return Order{}, ErrInvalidInput
	}
	r, err := a.call(ctx, user, "order_admin_report_get_v1", map[string]any{"p_order": strconv.FormatUint(order, 10)})
	if err != nil {
		return Order{}, err
	}
	orders, err := sharedReportOrders(r)
	if err != nil || len(orders) != 1 {
		return Order{}, ErrUnavailable
	}
	return orders[0], nil
}
func (a *SharedPGApplication) SearchOrders(ctx context.Context, user uint64, f OrderFilter) ([]Order, uint64, error) {
	p := reportPage(f.Page)
	p["p_state_hex"] = reportHex(f.State)
	p["p_date_hex"] = reportHex(f.Date)
	p["p_query_hex"] = reportHex(f.Query)
	p["p_unclaimed"] = f.Unclaimed
	r, err := a.call(ctx, user, "order_admin_report_search_v1", p)
	if err != nil {
		return nil, 0, err
	}
	out, err := sharedReportOrders(r)
	if err != nil {
		return nil, 0, err
	}
	next := uint64(0)
	if len(out) == int(f.Page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func (a *SharedPGApplication) Payments(ctx context.Context, user uint64, rng BillingRange, page PageQuery) ([]Payment, uint64, error) {
	r, err := a.call(ctx, user, "order_admin_report_payments_v1", reportRange(rng, page))
	if err != nil {
		return nil, 0, err
	}
	orders, err := sharedReportOrders(r)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Payment, 0, len(orders))
	for _, o := range orders {
		out = append(out, Payment{Order: o})
	}
	next := uint64(0)
	if len(out) == int(page.Limit) {
		next = out[len(out)-1].Order.ID
	}
	return out, next, nil
}
func (a *SharedPGApplication) Stats(ctx context.Context, user uint64, at time.Time) (Stats, error) {
	at = at.UTC()
	p := map[string]any{"p_at": map[string]any{"epoch_seconds": strconv.FormatInt(at.Unix(), 10), "nanosecond": at.Nanosecond(), "zero": at.IsZero(), "canonical_utc": at.Format(time.RFC3339Nano)}}
	r, err := a.call(ctx, user, "order_admin_report_stats_v1", p)
	if err != nil {
		return Stats{}, err
	}
	if len(r.Rows) != 2 {
		return Stats{}, ErrUnavailable
	}
	var s Stats
	if sharedReportRow(r.Rows[0]).Scan(&s.TodayRevenueCents, &s.TodayOrders, &s.MonthRevenueCents, &s.MonthOrders, &s.PendingProduction) != nil || sharedReportRow(r.Rows[1]).Scan(&s.RefundCents) != nil {
		return Stats{}, ErrUnavailable
	}
	for _, row := range r.Items {
		var p ProductSale
		if sharedReportRow(row).Scan(&p.ProductID, &p.Name, &p.Quantity) != nil {
			return Stats{}, ErrUnavailable
		}
		s.ProductSales = append(s.ProductSales, p)
	}
	return s, nil
}
func (a *SharedPGApplication) Refunds(ctx context.Context, user uint64, rng BillingRange, page PageQuery) ([]Refund, uint64, error) {
	r, err := a.call(ctx, user, "order_admin_report_refunds_v1", reportRange(rng, page))
	if err != nil {
		return nil, 0, err
	}
	out := []Refund{}
	for _, row := range r.Rows {
		var x Refund
		if sharedReportRow(row).Scan(&x.ID, &x.OrderID, &x.OrderNo, &x.State, &x.ProviderRefundID, &x.TransactionID, &x.Operator, &x.Reason, &x.AmountCents, &x.PaidAt, &x.RequestedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		x.State = refundStateLabel(x.State)
		out = append(out, x)
	}
	next := uint64(0)
	if len(out) == int(page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func (a *SharedPGApplication) Summary(ctx context.Context, user uint64, rng BillingRange) (Summary, error) {
	r, err := a.call(ctx, user, "order_admin_report_summary_v1", map[string]any{"p_from_hex": reportHex(rng.From), "p_to_hex": reportHex(rng.To)})
	if err != nil {
		return Summary{}, err
	}
	if len(r.Rows) != 3 {
		return Summary{}, ErrUnavailable
	}
	var s Summary
	if sharedReportRow(r.Rows[0]).Scan(&s.Count, &s.GrossCents, &s.StaffCount, &s.DiscountCents) != nil || sharedReportRow(r.Rows[1]).Scan(&s.RefundCount, &s.RefundCents, &s.PendingCount) != nil || sharedReportRow(r.Rows[2]).Scan(&s.UnbuiltCount, &s.UnbuiltCents) != nil {
		return Summary{}, ErrUnavailable
	}
	s.NetCents = int64(s.GrossCents) - int64(s.RefundCents)
	return s, nil
}
func (a *SharedPGApplication) ListPending(ctx context.Context, user uint64, page PageQuery) ([]Pending, uint64, error) {
	r, err := a.call(ctx, user, "order_admin_report_pending_v1", reportPage(page))
	if err != nil {
		return nil, 0, err
	}
	out := []Pending{}
	index := map[uint64]int{}
	for _, row := range r.Rows {
		var x Pending
		var quote uint64
		var phone string
		if sharedReportRow(row).Scan(&x.ID, &quote, &x.OutTradeNo, &x.State, &x.BlockingReason, &x.PickupDate, &x.PickupTime, &x.MealPeriod, &x.ContactName, &phone, &x.AmountCents, &x.PaidAt, &x.TransactionID) != nil {
			return nil, 0, ErrUnavailable
		}
		x.MaskedPhone = mask(phone)
		index[quote] = len(out)
		out = append(out, x)
	}
	for _, row := range r.Items {
		var quote uint64
		var item Item
		if sharedReportRow(row).Scan(&quote, &item.ProductID, &item.Name, &item.Quantity, &item.LineTotalCents) != nil {
			return nil, 0, ErrUnavailable
		}
		i, ok := index[quote]
		if !ok {
			return nil, 0, ErrUnavailable
		}
		out[i].Items = append(out[i].Items, item)
	}
	next := uint64(0)
	if len(out) == int(page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func (a *SharedPGApplication) ExportCSV(ctx context.Context, userID uint64, r BillingRange) (io.ReadCloser, error) {
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
func (a *SharedPGApplication) ProcessPending(ctx context.Context, m WriteMeta, id uint64, act PendingAction, reason string) (PendingResult, error) {
	if a.commands == nil {
		return PendingResult{}, ErrUnavailable
	}
	return a.commands.ProcessPending(ctx, m, id, act, reason)
}
func (a *SharedPGApplication) RequestRefund(ctx context.Context, m WriteMeta, id uint64, reason string) (Order, Refund, error) {
	if a.commands == nil {
		return Order{}, Refund{}, ErrUnavailable
	}
	return a.commands.RequestRefund(ctx, m, id, reason)
}
