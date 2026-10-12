package billing

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

type sharedBillingExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type SharedService struct {
	config   Config
	client   sharedBillingExecutor
	provider BillProvider
}

func NewSharedPG(client sharedBillingExecutor, provider BillProvider, config Config) *SharedService {
	return &SharedService{client: client, provider: provider, config: config}
}
func (s *SharedService) valid() bool {
	return s != nil && s.client != nil && cloudBilling(s.config) && validBillingConfig(s.config)
}
func billingTimeWire(at time.Time) map[string]any {
	at = at.UTC()
	return map[string]any{"epoch_seconds": strconv.FormatInt(at.Unix(), 10), "nanosecond": at.Nanosecond(), "zero": at.IsZero(), "canonical_utc": at.Format(time.RFC3339Nano)}
}
func billingDecimal(n uint64) string { return strconv.FormatUint(n, 10) }
func billingObject(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, ErrUnavailable
	}
	v := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		k, ok := token.(string)
		if err != nil || !ok {
			return nil, ErrUnavailable
		}
		if _, exists := v[k]; exists {
			return nil, ErrUnavailable
		}
		var item json.RawMessage
		if d.Decode(&item) != nil {
			return nil, ErrUnavailable
		}
		v[k] = item
	}
	token, err = d.Token()
	if err != nil || token != json.Delim('}') || d.Decode(&struct{}{}) != io.EOF || len(v) != len(keys) {
		return nil, ErrUnavailable
	}
	for _, key := range keys {
		if _, ok := v[key]; !ok {
			return nil, ErrUnavailable
		}
	}
	return v, nil
}
func billingText(raw json.RawMessage) (string, error) {
	var s string
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &s) != nil {
		return "", ErrUnavailable
	}
	return s, nil
}
func billingUint(raw json.RawMessage) (uint64, error) {
	s, err := billingText(raw)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || billingDecimal(n) != s {
		return 0, ErrUnavailable
	}
	return n, nil
}
func billingBytes(raw json.RawMessage) (string, error) {
	s, err := billingText(raw)
	if err != nil {
		return "", err
	}
	b, err := hex.DecodeString(s)
	if err != nil || hex.EncodeToString(b) != s || !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return "", ErrUnavailable
	}
	return string(b), nil
}
func billingScanTime(raw json.RawMessage) (time.Time, error) {
	v, err := billingObject(raw, "kind", "microseconds")
	if err != nil {
		return time.Time{}, err
	}
	kind, err := billingText(v["kind"])
	if err != nil || kind != "finite" {
		return time.Time{}, ErrUnavailable
	}
	value, err := billingText(v["microseconds"])
	if err != nil {
		return time.Time{}, err
	}
	microseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || strconv.FormatInt(microseconds, 10) != value || microseconds == int64(1<<63-1) || microseconds == int64(-1<<63) {
		return time.Time{}, ErrUnavailable
	}
	return time.Unix(946684800+microseconds/1000000, (microseconds%1000000)*1000).In(time.Local), nil
}
func (s *SharedService) call(ctx context.Context, name string, args map[string]any, keys ...string) (map[string]json.RawMessage, error) {
	var raw json.RawMessage
	if s.client.CallRPC(ctx, name, args, &raw) != nil {
		return nil, ErrUnavailable
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal(raw, &hint) != nil {
		return nil, ErrUnavailable
	}
	expected := "FOUND"
	if name == "order_billing_reconcile_v2" {
		expected = "RECONCILED"
	}
	if hint.Outcome != expected {
		if _, err := billingObject(raw, "outcome"); err != nil {
			return nil, err
		}
		if hint.Outcome == "FORBIDDEN" {
			return nil, ErrForbidden
		}
		return nil, ErrUnavailable
	}
	return billingObject(raw, append([]string{"outcome"}, keys...)...)
}
func (s *SharedService) Summary(ctx context.Context, owner uint64, period BillingRange) (Summary, error) {
	if !s.valid() || owner == 0 || !validRange(period) {
		return Summary{}, ErrInvalidInput
	}
	v, err := s.call(ctx, "order_billing_summary_v1", map[string]any{"p_owner": billingDecimal(owner), "p_from": billingTimeWire(period.From), "p_to": billingTimeWire(period.To)}, "summary")
	if err != nil {
		return Summary{}, err
	}
	fields, err := billingObject(v["summary"], "effective_revenue_cents", "effective_orders", "refund_cents", "refund_count", "pending_refunds")
	if err != nil {
		return Summary{}, err
	}
	var result Summary
	for _, f := range []struct {
		key string
		dst *uint64
	}{{"effective_revenue_cents", &result.EffectiveRevenueCents}, {"effective_orders", &result.EffectiveOrders}, {"refund_cents", &result.RefundCents}, {"refund_count", &result.RefundCount}, {"pending_refunds", &result.PendingRefunds}} {
		n, e := billingUint(fields[f.key])
		if e != nil {
			return Summary{}, e
		}
		*f.dst = n
	}
	return result, nil
}
func (s *SharedService) ListPayments(ctx context.Context, owner uint64, period BillingRange, page PageQuery) ([]Payment, uint64, error) {
	if !s.valid() || owner == 0 || !validRange(period) || !validPage(page) {
		return nil, 0, ErrInvalidInput
	}
	v, err := s.call(ctx, "order_billing_payments_v1", map[string]any{"p_owner": billingDecimal(owner), "p_from": billingTimeWire(period.From), "p_to": billingTimeWire(period.To), "p_after": billingDecimal(page.AfterID), "p_limit": page.Limit}, "payments")
	if err != nil {
		return nil, 0, err
	}
	var rows []json.RawMessage
	if json.Unmarshal(v["payments"], &rows) != nil || rows == nil || len(rows) > int(page.Limit) {
		return nil, 0, ErrUnavailable
	}
	items := make([]Payment, 0, len(rows))
	for _, row := range rows {
		f, e := billingObject(row, "order_id", "order_no_hex", "out_trade_no_hex", "transaction_id_hex", "state", "amount_cents", "paid_at")
		if e != nil {
			return nil, 0, e
		}
		var p Payment
		p.OrderID, e = billingUint(f["order_id"])
		if e != nil {
			return nil, 0, e
		}
		p.OrderNo, e = billingBytes(f["order_no_hex"])
		if e != nil {
			return nil, 0, e
		}
		p.OutTradeNo, e = billingBytes(f["out_trade_no_hex"])
		if e != nil {
			return nil, 0, e
		}
		p.TransactionID, e = billingBytes(f["transaction_id_hex"])
		if e != nil {
			return nil, 0, e
		}
		p.State, e = billingText(f["state"])
		if e != nil {
			return nil, 0, e
		}
		p.AmountCents, e = billingUint(f["amount_cents"])
		if e != nil {
			return nil, 0, e
		}
		p.PaidAt, e = billingScanTime(f["paid_at"])
		if e != nil {
			return nil, 0, e
		}
		items = append(items, p)
	}
	return items, nextPaymentCursor(items, page.Limit), nil
}
func (s *SharedService) ListRefunds(ctx context.Context, owner uint64, period BillingRange, page PageQuery) ([]Refund, uint64, error) {
	if !s.valid() || owner == 0 || !validRange(period) || !validPage(page) {
		return nil, 0, ErrInvalidInput
	}
	v, err := s.call(ctx, "order_billing_refunds_v1", map[string]any{"p_owner": billingDecimal(owner), "p_from": billingTimeWire(period.From), "p_to": billingTimeWire(period.To), "p_after": billingDecimal(page.AfterID), "p_limit": page.Limit}, "refunds")
	if err != nil {
		return nil, 0, err
	}
	var rows []json.RawMessage
	if json.Unmarshal(v["refunds"], &rows) != nil || rows == nil || len(rows) > int(page.Limit) {
		return nil, 0, ErrUnavailable
	}
	items := make([]Refund, 0, len(rows))
	for _, row := range rows {
		f, e := billingObject(row, "id", "order_id", "out_refund_no_hex", "provider_refund_id_hex", "state", "amount_cents", "requested_at")
		if e != nil {
			return nil, 0, e
		}
		var p Refund
		p.ID, e = billingUint(f["id"])
		if e != nil {
			return nil, 0, e
		}
		p.OrderID, e = billingUint(f["order_id"])
		if e != nil {
			return nil, 0, e
		}
		p.OutRefundNo, e = billingBytes(f["out_refund_no_hex"])
		if e != nil {
			return nil, 0, e
		}
		p.ProviderRefundID, e = billingBytes(f["provider_refund_id_hex"])
		if e != nil {
			return nil, 0, e
		}
		p.State, e = billingText(f["state"])
		if e != nil {
			return nil, 0, e
		}
		p.AmountCents, e = billingUint(f["amount_cents"])
		if e != nil {
			return nil, 0, e
		}
		p.RequestedAt, e = billingScanTime(f["requested_at"])
		if e != nil {
			return nil, 0, e
		}
		items = append(items, p)
	}
	return items, nextRefundCursor(items, page.Limit), nil
}

func (service *SharedService) ExportCSV(ctx context.Context, ownerUserID uint64, period BillingRange) (io.ReadCloser, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) {
		return nil, ErrInvalidInput
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"order_id", "order_no", "out_trade_no", "transaction_id", "state", "amount_cents", "paid_at"}); err != nil {
		return nil, ErrUnavailable
	}
	cursor := uint64(0)
	for {
		items, next, err := service.ListPayments(ctx, ownerUserID, period, PageQuery{AfterID: cursor, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := writer.Write([]string{strconv.FormatUint(item.OrderID, 10), item.OrderNo, item.OutTradeNo, item.TransactionID, item.State, strconv.FormatUint(item.AmountCents, 10), item.PaidAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}); err != nil {
				return nil, ErrUnavailable
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	writer.Flush()
	if writer.Error() != nil {
		return nil, ErrUnavailable
	}
	return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}

var _ Application = (*SharedService)(nil)
