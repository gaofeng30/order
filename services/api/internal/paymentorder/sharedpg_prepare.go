package paymentorder

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/jackc/pgx/v5/pgconn"
)

// sharedPaymentObject rejects duplicate, missing and unknown wire members.
func sharedPaymentObject(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, ErrUnavailable
	}
	values := make(map[string]json.RawMessage, len(keys))
	for d.More() {
		token, err = d.Token()
		k, ok := token.(string)
		if err != nil || !ok {
			return nil, ErrUnavailable
		}
		if _, exists := values[k]; exists {
			return nil, ErrUnavailable
		}
		var v json.RawMessage
		if d.Decode(&v) != nil {
			return nil, ErrUnavailable
		}
		values[k] = v
	}
	token, err = d.Token()
	if err != nil || token != json.Delim('}') || d.Decode(&struct{}{}) != io.EOF || len(values) != len(keys) {
		return nil, ErrUnavailable
	}
	for _, k := range keys {
		if _, ok := values[k]; !ok {
			return nil, ErrUnavailable
		}
	}
	return values, nil
}
func sharedPaymentString(raw json.RawMessage) (string, error) {
	var s string
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &s) != nil {
		return "", ErrUnavailable
	}
	return s, nil
}
func sharedPaymentHex(raw json.RawMessage) ([]byte, error) {
	s, e := sharedPaymentString(raw)
	if e != nil {
		return nil, e
	}
	v, e := hex.DecodeString(s)
	if e != nil || hex.EncodeToString(v) != s {
		return nil, ErrUnavailable
	}
	return v, nil
}
func sharedPaymentTime(raw json.RawMessage) (time.Time, error) {
	s, e := sharedPaymentString(raw)
	if e != nil {
		return time.Time{}, e
	}
	at, e := time.Parse(time.RFC3339Nano, s)
	if e != nil {
		return time.Time{}, ErrUnavailable
	}
	return at.UTC(), nil
}
func sharedPaymentID(raw json.RawMessage) (uint64, error) {
	s, e := sharedPaymentString(raw)
	if e != nil {
		return 0, e
	}
	return parseDecimalID(s)
}
func sharedPaymentNull(raw json.RawMessage) bool { return bytes.Equal(raw, []byte("null")) }
func sharedPaymentError(outcome string) error {
	switch outcome {
	case "NOT_FOUND":
		return ErrNotFound
	case "IDEMPOTENCY_CONFLICT":
		return ErrIdempotencyConflict
	case "INVALID_INPUT":
		return ErrInvalidInput
	case "FORBIDDEN":
		return ErrForbidden
	case "QUOTE_UNAVAILABLE":
		return ErrQuoteUnavailable
	case "QUOTE_UNAVAILABLE_RETRY":
		return quote.ErrUnavailable
	default:
		return ErrUnavailable
	}
}
func (a *SharedPGApplication) sharedPaymentCall(ctx context.Context, name string, args map[string]any) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := a.client.CallRPC(ctx, name, args, &raw); err != nil {
		var ce *cloudbasepg.Error
		if (name == "order_payment_prepare_v2" || name == "order_payment_observe_v1" || name == "order_payment_claim_query_v1" || name == "order_payment_materialize_v1" || name == "order_payment_receipt_append_v1") && errors.As(err, &ce) && (ce.SQLState == "40P01" || ce.SQLState == "40001" || ce.SQLState == "55P03") {
			return nil, &pgconn.PgError{Code: ce.SQLState}
		}
		return nil, ErrUnavailable
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal(raw, &hint) != nil || hint.Outcome == "" {
		return nil, ErrUnavailable
	}
	if hint.Outcome == "QUOTE_UNAVAILABLE_RETRY" && name != "order_payment_prepare_v2" && name != "order_payment_materialize_v1" {
		return nil, ErrUnavailable
	}
	if hint.Outcome == "SQL_FAILURE" {
		v, e := sharedPaymentObject(raw, "outcome", "sqlstate", "constraint")
		if e != nil {
			return nil, e
		}
		code, e := sharedPaymentString(v["sqlstate"])
		if e != nil {
			return nil, e
		}
		constraint, e := sharedPaymentString(v["constraint"])
		if e != nil {
			return nil, e
		}
		allowed := name == "order_payment_prepare_v2" || name == "order_payment_observe_v1" || name == "order_payment_claim_query_v1" || name == "order_payment_materialize_v1" || name == "order_payment_receipt_append_v1"
		if allowed && ((code == "40P01" || code == "40001" || code == "55P03") && constraint == "") {
			return nil, &pgconn.PgError{Code: code}
		}
		pg := &pgconn.PgError{Code: code, ConstraintName: constraint}
		if (name == "order_payment_prepare_v2" || name == "order_payment_receipt_append_v1") && isDuplicate(pg) {
			return nil, pg
		}
		return nil, ErrUnavailable
	}
	return raw, nil
}
func sharedPaymentOutcome(raw json.RawMessage, success string, keys ...string) (map[string]json.RawMessage, error) {
	var hint struct {
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal(raw, &hint) != nil {
		return nil, ErrUnavailable
	}
	if hint.Outcome != success {
		if _, e := sharedPaymentObject(raw, "outcome"); e != nil {
			return nil, e
		}
		return nil, sharedPaymentError(hint.Outcome)
	}
	return sharedPaymentObject(raw, append([]string{"outcome"}, keys...)...)
}
func sharedPaymentRecord(raw json.RawMessage) (prepaymentRecord, error) {
	keys := []string{"id", "user_id", "quote_id", "idempotency_key_hash", "out_trade_no", "expected_appid", "expected_mchid", "expected_amount_cents", "currency", "provider_create_request_json", "provider_create_request_digest", "effective_deadline", "provider_state", "create_attempted_at", "wx_request_payment_json", "provider_prepay_id", "last_queried_at", "materialization_state", "pending_reason_code", "materialized_at", "lease_kind", "lease_owner", "lease_expires_at", "record_version", "next_reconcile_at", "created_at", "updated_at"}
	v, e := sharedPaymentObject(raw, keys...)
	if e != nil {
		return prepaymentRecord{}, e
	}
	var r prepaymentRecord
	for _, f := range []struct {
		k string
		p *uint64
	}{{"id", &r.id}, {"user_id", &r.userID}, {"quote_id", &r.quoteID}, {"record_version", &r.recordVersion}} {
		if *f.p, e = sharedPaymentID(v[f.k]); e != nil {
			return r, e
		}
	}
	for _, f := range []struct {
		k string
		p *string
	}{{"currency", &r.currency}} {
		if *f.p, e = sharedPaymentString(v[f.k]); e != nil {
			return r, e
		}
	}
	for _, f := range []struct {
		k string
		p *string
	}{{"out_trade_no", &r.outTradeNo}, {"expected_appid", &r.expectedAppID}, {"expected_mchid", &r.expectedMerchantID}} {
		b, err := sharedPaymentHex(v[f.k])
		if err != nil {
			return r, err
		}
		*f.p = string(b)
	}
	for _, f := range []struct {
		k string
		p *[32]byte
	}{{"idempotency_key_hash", &r.idempotencyKeyHash}, {"provider_create_request_digest", &r.createRequestDigest}} {
		b, err := sharedPaymentHex(v[f.k])
		if err != nil || len(b) != 32 {
			return r, ErrUnavailable
		}
		copy(f.p[:], b)
	}
	amount, err := sharedPaymentString(v["expected_amount_cents"])
	if err != nil {
		return r, err
	}
	r.expectedAmountCents, e = strconv.ParseInt(amount, 10, 64)
	if e != nil || strconv.FormatInt(r.expectedAmountCents, 10) != amount {
		return r, ErrUnavailable
	}
	req, err := sharedPaymentString(v["provider_create_request_json"])
	if err != nil {
		return r, err
	}
	r.createRequestJSON = []byte(req)
	state, err := sharedPaymentString(v["provider_state"])
	if err != nil {
		return r, err
	}
	r.providerState = ProviderState(state)
	state, err = sharedPaymentString(v["materialization_state"])
	if err != nil {
		return r, err
	}
	r.materialState = MaterializationState(state)
	for _, f := range []struct {
		k string
		p *time.Time
	}{{"effective_deadline", &r.effectiveDeadline}, {"created_at", &r.createdAt}, {"updated_at", &r.updatedAt}} {
		if *f.p, e = sharedPaymentTime(v[f.k]); e != nil {
			return r, e
		}
	}
	for _, f := range []struct {
		k string
		p *sql.NullTime
	}{{"create_attempted_at", &r.createAttemptedAt}, {"last_queried_at", &r.lastQueriedAt}, {"materialized_at", &r.materializedAt}, {"lease_expires_at", &r.leaseExpiresAt}, {"next_reconcile_at", &r.nextReconcileAt}} {
		if !sharedPaymentNull(v[f.k]) {
			f.p.Valid = true
			if f.p.Time, e = sharedPaymentTime(v[f.k]); e != nil {
				return r, e
			}
		}
	}
	for _, f := range []struct {
		k string
		p *sql.NullString
	}{{"pending_reason_code", &r.pendingReason}, {"lease_kind", &r.leaseKind}} {
		if !sharedPaymentNull(v[f.k]) {
			f.p.Valid = true
			if f.p.String, e = sharedPaymentString(v[f.k]); e != nil {
				return r, e
			}
		}
	}
	for _, f := range []struct {
		k string
		p *[]byte
	}{{"provider_prepay_id", &r.providerPrepayID}, {"lease_owner", &r.leaseOwner}} {
		if !sharedPaymentNull(v[f.k]) {
			if *f.p, e = sharedPaymentHex(v[f.k]); e != nil {
				return r, e
			}
		}
	}
	if !sharedPaymentNull(v["wx_request_payment_json"]) {
		s, err := sharedPaymentString(v["wx_request_payment_json"])
		if err != nil {
			return r, err
		}
		r.wxRequestPaymentJSON = []byte(s)
	}
	if !r.validStored() {
		return r, ErrUnavailable
	}
	return r, nil
}
func (a *SharedPGApplication) readOpenID(ctx context.Context, user uint64) (string, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_openid_v1", map[string]any{"p_user": decimalID(user)})
	if e != nil {
		return "", e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "openid_hex")
	if e != nil {
		return "", e
	}
	b, e := sharedPaymentHex(v["openid_hex"])
	return string(b), e
}
func (a *SharedPGApplication) readPrepaymentByID(ctx context.Context, id uint64) (prepaymentRecord, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_read_v1", map[string]any{"p_prepayment": decimalID(id)})
	if e != nil {
		return prepaymentRecord{}, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "record")
	if e != nil {
		return prepaymentRecord{}, e
	}
	return sharedPaymentRecord(v["record"])
}
func (a *SharedPGApplication) findPrepareReplay(ctx context.Context, user, quoteID uint64, key [32]byte) (prepaymentRecord, bool, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_prepare_replay_v1", map[string]any{"p_user": decimalID(user), "p_quote": decimalID(quoteID), "p_key_hash": hex.EncodeToString(key[:])})
	if e != nil {
		return prepaymentRecord{}, false, e
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "ABSENT" {
		_, e = sharedPaymentObject(raw, "outcome")
		return prepaymentRecord{}, false, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "record")
	if e != nil {
		return prepaymentRecord{}, false, e
	}
	r, e := sharedPaymentRecord(v["record"])
	if e != nil {
		return r, false, e
	}
	if r.userID != user {
		return r, false, ErrNotFound
	}
	if r.quoteID != quoteID {
		return r, false, ErrIdempotencyConflict
	}
	return r, true, nil
}
func (a *SharedPGApplication) prepareOnce(ctx context.Context, meta WriteMeta, quoteID uint64, key [32]byte, openid string, owner [16]byte, now time.Time) (claimedCreate, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_prepare_v2", map[string]any{"p_user": decimalID(meta.ActorUserID), "p_quote": decimalID(quoteID), "p_key_hash": hex.EncodeToString(key[:]), "p_openid_hex": hex.EncodeToString([]byte(openid)), "p_owner_hex": hex.EncodeToString(owner[:]), "p_now": now.UTC().Truncate(time.Microsecond), "p_lease_ns": strconv.FormatInt(int64(a.config.LeaseDuration), 10), "p_config": map[string]string{"environment": a.config.Environment, "appid_hex": hex.EncodeToString([]byte(a.config.AppID)), "mchid_hex": hex.EncodeToString([]byte(a.config.MerchantID)), "description_hex": hex.EncodeToString([]byte(a.config.Description)), "notify_url_hex": hex.EncodeToString([]byte(a.config.PaymentNotifyURL))}})
	if e != nil {
		return claimedCreate{}, e
	}
	v, e := sharedPaymentOutcome(raw, "CLAIMED", "id", "owner_hex", "version", "request_json")
	if e != nil {
		if errors.Is(e, ErrNotFound) {
			return claimedCreate{}, quote.ErrNotFound
		}
		return claimedCreate{}, e
	}
	var c claimedCreate
	if c.id, e = sharedPaymentID(v["id"]); e != nil {
		return c, e
	}
	if c.version, e = sharedPaymentID(v["version"]); e != nil || c.version != 2 {
		return c, ErrUnavailable
	}
	b, e := sharedPaymentHex(v["owner_hex"])
	if e != nil || !bytes.Equal(b, owner[:]) || owner == ([16]byte{}) {
		return c, ErrUnavailable
	}
	c.owner = owner
	req, e := sharedPaymentString(v["request_json"])
	if e != nil {
		return c, e
	}
	request, canonical, e := canonicalProviderCreateRequest([]byte(req))
	if e != nil || !bytes.Equal(canonical, []byte(req)) || request.QuoteID != decimalID(quoteID) || request.AppID != a.config.AppID || request.MerchantID != a.config.MerchantID || request.PayerOpenID != openid {
		return c, ErrUnavailable
	}
	c.request = request
	return c, nil
}
func (a *SharedPGApplication) finishCreateSuccess(ctx context.Context, c claimedCreate, result ProviderCreateResult, now time.Time) error {
	wx, e := json.Marshal(result.RequestPayment)
	if e != nil {
		return ErrUnavailable
	}
	rows, e := a.finishCreate(ctx, c, now, map[string]string{"prepay_id_hex": hex.EncodeToString([]byte(result.PrepayID)), "request_payment_json": string(wx)})
	if e != nil {
		return ErrUnavailable
	}
	if rows != 1 {
		r, e := a.readPrepaymentByID(ctx, c.id)
		if e == nil && (r.providerState == ProviderPaid || r.providerState == ProviderClosed) {
			return nil
		}
		return ErrUnavailable
	}
	return nil
}
func (a *SharedPGApplication) finishCreateUnknown(ctx context.Context, c claimedCreate, now time.Time) error {
	rows, e := a.finishCreate(ctx, c, now, nil)
	if e != nil || rows != 1 {
		return ErrUnavailable
	}
	return nil
}
func (a *SharedPGApplication) finishCreate(ctx context.Context, c claimedCreate, now time.Time, result any) (int, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_finish_create_v1", map[string]any{"p_prepayment": decimalID(c.id), "p_owner_hex": hex.EncodeToString(c.owner[:]), "p_version": decimalID(c.version), "p_now": now.UTC().Truncate(time.Microsecond), "p_next": now.Add(a.config.ReconcileInterval).UTC().Truncate(time.Microsecond), "p_result": result})
	if e != nil {
		return 0, e
	}
	v, e := sharedPaymentOutcome(raw, "FINISHED", "rows")
	if e != nil {
		return 0, e
	}
	s, e := sharedPaymentString(v["rows"])
	if e != nil || (s != "0" && s != "1") {
		return 0, ErrUnavailable
	}
	n, _ := strconv.Atoi(s)
	return n, nil
}
