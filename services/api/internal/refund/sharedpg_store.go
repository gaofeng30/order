package refund

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"io"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/jackc/pgx/v5/pgconn"
)

type sharedPGStore struct {
	client         sharedRefundExecutor
	enqueueEnabled bool
	config         Config
}

func newSharedPGStore(client sharedRefundExecutor, config Config, enqueueEnabled bool) *sharedPGStore {
	return &sharedPGStore{client: client, config: config, enqueueEnabled: enqueueEnabled}
}

// Time facts retain the original comparison precision. SQL converts them only
// where the native transaction would bind a timestamptz argument.
func sharedRefundTime(t time.Time) map[string]any {
	return map[string]any{"epoch_seconds": strconv.FormatInt(t.Unix(), 10), "nanosecond": t.Nanosecond(), "zero": t.IsZero(), "canonical": t.Format(time.RFC3339Nano)}
}
func sharedRefundDecimal(id uint64) string { return strconv.FormatUint(id, 10) }
func sharedRefundObject(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, e := d.Token()
	if e != nil || token != json.Delim('{') {
		return nil, ErrUnavailable
	}
	v := make(map[string]json.RawMessage, len(keys))
	for d.More() {
		token, e = d.Token()
		k, ok := token.(string)
		if e != nil || !ok {
			return nil, ErrUnavailable
		}
		if _, exists := v[k]; exists {
			return nil, ErrUnavailable
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return nil, ErrUnavailable
		}
		v[k] = value
	}
	token, e = d.Token()
	if e != nil || token != json.Delim('}') || d.Decode(&struct{}{}) != io.EOF || len(v) != len(keys) {
		return nil, ErrUnavailable
	}
	for _, k := range keys {
		if _, ok := v[k]; !ok {
			return nil, ErrUnavailable
		}
	}
	return v, nil
}
func sharedRefundString(raw json.RawMessage) (string, error) {
	var s string
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &s) != nil {
		return "", ErrUnavailable
	}
	return s, nil
}
func sharedRefundID(raw json.RawMessage) (uint64, error) {
	s, e := sharedRefundString(raw)
	if e != nil {
		return 0, e
	}
	id, e := strconv.ParseUint(s, 10, 64)
	if e != nil || sharedRefundDecimal(id) != s {
		return 0, ErrUnavailable
	}
	return id, nil
}
func sharedRefundHex(raw json.RawMessage) ([]byte, error) {
	s, e := sharedRefundString(raw)
	if e != nil {
		return nil, e
	}
	b, e := hex.DecodeString(s)
	if e != nil || hex.EncodeToString(b) != s {
		return nil, ErrUnavailable
	}
	return b, nil
}
func sharedRefundOutcome(raw json.RawMessage, outcome string, keys ...string) (map[string]json.RawMessage, error) {
	var hint struct {
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal(raw, &hint) != nil {
		return nil, ErrUnavailable
	}
	if hint.Outcome != outcome {
		if _, e := sharedRefundObject(raw, "outcome"); e != nil {
			return nil, e
		}
		switch hint.Outcome {
		case "NOT_FOUND":
			return nil, ErrNotFound
		case "FORBIDDEN":
			return nil, ErrForbidden
		case "IDEMPOTENCY_CONFLICT":
			return nil, ErrIdempotencyConflict
		case "INVALID_INPUT":
			return nil, ErrInvalidInput
		case "TRANSITION_NOT_ALLOWED":
			return nil, ErrTransitionNotAllowed
		case "STALE_QUERY_LEASE":
			return nil, errStaleQueryLease
		default:
			return nil, ErrUnavailable
		}
	}
	return sharedRefundObject(raw, append([]string{"outcome"}, keys...)...)
}
func (s *sharedPGStore) call(ctx context.Context, name string, args map[string]any) (json.RawMessage, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnavailable
	}
	var raw json.RawMessage
	if e := s.client.CallRPC(ctx, name, args, &raw); e != nil {
		var ce *cloudbasepg.Error
		if errors.As(e, &ce) && (ce.SQLState == "40P01" || ce.SQLState == "40001" || ce.SQLState == "55P03") && (name == "order_refund_request_order_v2" || name == "order_refund_request_paid_v2" || name == "order_refund_observe_v1") {
			return nil, &pgconn.PgError{Code: ce.SQLState}
		}
		var pe *pgconn.PgError
		if errors.As(e, &pe) && retryablePostgreSQL(e) && (name == "order_refund_request_order_v2" || name == "order_refund_request_paid_v2" || name == "order_refund_observe_v1") {
			return nil, &pgconn.PgError{Code: pe.Code}
		}
		return nil, ErrUnavailable
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	if json.Unmarshal(raw, &hint) != nil || hint.Outcome == "" {
		return nil, ErrUnavailable
	}
	if hint.Outcome == "SQL_FAILURE" {
		v, e := sharedRefundObject(raw, "outcome", "sqlstate", "constraint")
		if e != nil {
			return nil, e
		}
		code, e := sharedRefundString(v["sqlstate"])
		if e != nil {
			return nil, e
		}
		constraint, e := sharedRefundString(v["constraint"])
		if e != nil {
			return nil, e
		}
		atomic := name == "order_refund_request_order_v2" || name == "order_refund_request_paid_v2" || name == "order_refund_observe_v1"
		if atomic && constraint == "" && (code == "40P01" || code == "40001" || code == "55P03") {
			return nil, &pgconn.PgError{Code: code}
		}
		if atomic && code == "23505" && (constraint == "uq_refunds_prepayment" || constraint == "uq_refunds_order" || constraint == "uq_refunds_out_refund_no" || constraint == "uq_action_audits_command_receipt") {
			return nil, &pgconn.PgError{Code: code, ConstraintName: constraint}
		}
		return nil, ErrUnavailable
	}
	return raw, nil
}
func sharedRefundPGTime(kind string, raw json.RawMessage) (time.Time, error) {
	if kind != "FINITE" {
		return time.Time{}, ErrUnavailable
	}
	str, e := sharedRefundString(raw)
	if e != nil {
		return time.Time{}, e
	}
	us, e := strconv.ParseInt(str, 10, 64)
	if e != nil || strconv.FormatInt(us, 10) != str || us == 9223372036854775807 || us == (-9223372036854775807-1) {
		return time.Time{}, ErrUnavailable
	}
	// Identical to pgx's binary decoder, including its default time.Local.
	return time.Unix(946684800+us/1000000, (us%1000000)*1000), nil
}
func sharedRefundView(raw json.RawMessage) (Refund, error) {
	v, e := sharedRefundObject(raw, "id", "prepayment_id", "order_id", "state", "materialization_state", "amount_cents", "currency", "requested_kind", "requested_microseconds", "provider_refund_id_hex", "pending_reason")
	if e != nil {
		return Refund{}, e
	}
	var r Refund
	for _, f := range []struct {
		k string
		p *uint64
	}{{"id", &r.ID}, {"prepayment_id", &r.PrepaymentID}, {"amount_cents", &r.AmountCents}} {
		if *f.p, e = sharedRefundID(v[f.k]); e != nil {
			return r, e
		}
	}
	if !bytes.Equal(v["order_id"], []byte("null")) {
		if r.OrderID, e = sharedRefundID(v["order_id"]); e != nil {
			return r, e
		}
	}
	state, e := sharedRefundString(v["state"])
	if e != nil {
		return r, e
	}
	r.State = ProviderState(state)
	mat, e := sharedRefundString(v["materialization_state"])
	if e != nil {
		return r, e
	}
	r.MaterializationState = MaterializationState(mat)
	if r.Currency, e = sharedRefundString(v["currency"]); e != nil {
		return r, e
	}
	kind, e := sharedRefundString(v["requested_kind"])
	if e != nil {
		return r, e
	}
	if r.RequestedAt, e = sharedRefundPGTime(kind, v["requested_microseconds"]); e != nil {
		return r, e
	}
	if !bytes.Equal(v["provider_refund_id_hex"], []byte("null")) {
		b, e := sharedRefundHex(v["provider_refund_id_hex"])
		if e != nil {
			return r, e
		}
		r.ProviderRefundID = string(b)
	}
	if !bytes.Equal(v["pending_reason"], []byte("null")) {
		if r.PendingReason, e = sharedRefundString(v["pending_reason"]); e != nil {
			return r, e
		}
	}
	return r, nil
}
func sharedRefundRequestArgs(meta WriteMeta, reason string, now time.Time, notify string) map[string]any {
	key := sha256.Sum256([]byte(meta.IdempotencyKey))
	request := sha256.Sum256([]byte(meta.RequestID))
	return map[string]any{"p_actor": sharedRefundDecimal(meta.ActorUserID), "p_reason_hex": hex.EncodeToString([]byte(reason)), "p_key_hash": hex.EncodeToString(key[:]), "p_request_hash": hex.EncodeToString(request[:]), "p_now": sharedRefundTime(now), "p_notify_url_hex": hex.EncodeToString([]byte(notify))}
}
func sharedRefundRequested(raw json.RawMessage) (Refund, bool, error) {
	v, e := sharedRefundOutcome(raw, "REQUESTED", "receipt", "is_new", "refund_id", "prepayment_id")
	if e != nil {
		return Refund{}, false, e
	}
	str, e := sharedRefundString(v["receipt"])
	if e != nil {
		return Refund{}, false, e
	}
	var r Refund
	if json.Unmarshal([]byte(str), &r) != nil || r.ID == 0 || r.PrepaymentID == 0 || r.AmountCents == 0 || r.Currency != "CNY" {
		return Refund{}, false, ErrUnavailable
	}
	id, e := sharedRefundID(v["refund_id"])
	if e != nil || id != r.ID {
		return Refund{}, false, ErrUnavailable
	}
	id, e = sharedRefundID(v["prepayment_id"])
	if e != nil || id != r.PrepaymentID {
		return Refund{}, false, ErrUnavailable
	}
	var fresh bool
	if bytes.Equal(v["is_new"], []byte("null")) || json.Unmarshal(v["is_new"], &fresh) != nil {
		return Refund{}, false, ErrUnavailable
	}
	return r, fresh, nil
}
func (s *sharedPGStore) requestOrder(ctx context.Context, meta WriteMeta, id uint64, reason string, now time.Time, notify string) (Refund, bool, error) {
	raw, e := s.call(ctx, "order_refund_order_locator_v1", map[string]any{"p_order": sharedRefundDecimal(id)})
	if e != nil {
		return Refund{}, false, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "FOUND", "locator")
	if e != nil {
		return Refund{}, false, e
	}
	facts, e := sharedRefundObject(v["locator"], "order_id", "prepayment_id", "user_id", "pickup_kind", "pickup_microseconds")
	if e != nil {
		return Refund{}, false, e
	}
	locator := make(map[string]any, 4)
	for _, key := range []string{"order_id", "prepayment_id", "user_id"} {
		value, e := sharedRefundID(facts[key])
		if e != nil {
			return Refund{}, false, e
		}
		locator[key] = sharedRefundDecimal(value)
	}
	kind, e := sharedRefundString(facts["pickup_kind"])
	if e != nil {
		return Refund{}, false, e
	}
	pickup, e := sharedRefundPGTime(kind, facts["pickup_microseconds"])
	if e != nil {
		return Refund{}, false, e
	}
	locator["pickup_at"] = sharedRefundTime(pickup)
	args := sharedRefundRequestArgs(meta, reason, now, notify)
	args["p_environment"] = s.config.Environment
	args["p_actor_kind"] = string(meta.ActorKind)
	args["p_locator"] = locator
	for attempt := 0; attempt < 2; attempt++ {
		raw, e = s.call(ctx, "order_refund_request_order_v2", args)
		if e == nil {
			return sharedRefundRequested(raw)
		}
		if attempt == 0 && refundNumberCollision(e) {
			continue
		}
		if !retryablePostgreSQL(e) || attempt == 1 {
			return Refund{}, false, mapStoreError(e)
		}
	}
	return Refund{}, false, ErrUnavailable
}
func (s *sharedPGStore) requestPaidPrepayment(ctx context.Context, meta WriteMeta, id uint64, reason string, now time.Time, notify string) (Refund, bool, error) {
	args := sharedRefundRequestArgs(meta, reason, now, notify)
	args["p_environment"] = s.config.Environment
	args["p_prepayment"] = sharedRefundDecimal(id)
	for attempt := 0; attempt < 2; attempt++ {
		raw, e := s.call(ctx, "order_refund_request_paid_v2", args)
		if e == nil {
			return sharedRefundRequested(raw)
		}
		if attempt == 0 && refundNumberCollision(e) {
			continue
		}
		if !retryablePostgreSQL(e) || attempt == 1 {
			return Refund{}, false, mapStoreError(e)
		}
	}
	return Refund{}, false, ErrUnavailable
}
func (s *sharedPGStore) claimCreate(ctx context.Context, id uint64, owner [16]byte, now time.Time, _ time.Duration) (createClaim, bool, error) {
	raw, e := s.call(ctx, "order_refund_claim_create_v1", map[string]any{"p_refund": sharedRefundDecimal(id), "p_owner_hex": hex.EncodeToString(owner[:]), "p_now": sharedRefundTime(now)})
	if e != nil {
		return createClaim{}, false, ErrUnavailable
	}
	var hint struct{ Outcome string }
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "NOT_CLAIMED" {
		_, e = sharedRefundOutcome(raw, "NOT_CLAIMED")
		return createClaim{}, false, e
	}
	v, e := sharedRefundOutcome(raw, "CLAIMED", "version", "request_json")
	if e != nil {
		return createClaim{}, false, e
	}
	version, e := sharedRefundID(v["version"])
	if e != nil {
		return createClaim{}, false, e
	}
	str, e := sharedRefundString(v["request_json"])
	if e != nil {
		return createClaim{}, false, e
	}
	var req ProviderCreateRequest
	if json.Unmarshal([]byte(str), &req) != nil || !validProviderCreateRequest(req) {
		return createClaim{}, false, ErrUnavailable
	}
	if err := s.sharedRefundEnvironmentProof(ctx, "REFUND", s.config.AppID, s.config.MerchantID, req.OutTradeNo, req.OutRefundNo, "", req.TransactionID, nil); err != nil {
		return createClaim{}, false, err
	}
	return createClaim{refundID: id, request: req, owner: owner, version: version}, true, nil
}
func sharedRefundProviderRequest(r ProviderCreateRequest) map[string]any {
	return map[string]any{"out_trade_no_hex": hex.EncodeToString([]byte(r.OutTradeNo)), "transaction_id_hex": hex.EncodeToString([]byte(r.TransactionID)), "out_refund_no_hex": hex.EncodeToString([]byte(r.OutRefundNo)), "reason_hex": hex.EncodeToString([]byte(r.Reason)), "notify_url_hex": hex.EncodeToString([]byte(r.NotifyURL)), "amount_cents": sharedRefundDecimal(r.AmountCents), "total_cents": sharedRefundDecimal(r.TotalCents), "currency": r.Currency}
}
func sharedRefundObserved(r ProviderRefund) map[string]any {
	return map[string]any{"merchant_id_hex": hex.EncodeToString([]byte(r.MerchantID)), "out_trade_no_hex": hex.EncodeToString([]byte(r.OutTradeNo)), "transaction_id_hex": hex.EncodeToString([]byte(r.TransactionID)), "out_refund_no_hex": hex.EncodeToString([]byte(r.OutRefundNo)), "refund_id_hex": hex.EncodeToString([]byte(r.RefundID)), "state": string(r.State), "amount_cents": sharedRefundDecimal(r.AmountCents), "total_cents": sharedRefundDecimal(r.TotalCents), "currency": r.Currency, "success_time": sharedRefundTime(r.SuccessTime)}
}
func sharedRefundReceiptTime(v map[string]json.RawMessage) (map[string]any, error) {
	kind, e := sharedRefundString(v["stored_kind"])
	if e != nil {
		return nil, e
	}
	var micros any
	if kind == "FINITE" {
		s, e := sharedRefundString(v["stored_microseconds"])
		if e != nil {
			return nil, e
		}
		micros = s
	} else if kind != "POSITIVE_INFINITY" && kind != "NEGATIVE_INFINITY" && kind != "NULL" {
		return nil, ErrUnavailable
	}
	fact := map[string]any{"stored_kind": kind, "stored_microseconds": micros, "json_hex": nil, "codec_status": "SCAN_UNAVAILABLE"}
	at, e := sharedRefundPGTime(kind, v["stored_microseconds"])
	if e != nil {
		return fact, nil
	}
	b, e := at.MarshalJSON()
	if e != nil {
		fact["codec_status"] = "MARSHAL_UNAVAILABLE"
		return fact, nil
	}
	fact["codec_status"] = "VALID"
	fact["json_hex"] = hex.EncodeToString(b)
	return fact, nil
}
func (s *sharedPGStore) finishCreate(ctx context.Context, c createClaim, observed ProviderRefund, providerErr error, now time.Time) (Refund, error) {
	clock := sharedRefundTime(now)
	clock["receipt_time"] = nil
	args := map[string]any{"p_refund": sharedRefundDecimal(c.refundID), "p_owner_hex": hex.EncodeToString(c.owner[:]), "p_version": sharedRefundDecimal(c.version), "p_request": sharedRefundProviderRequest(c.request), "p_provider_error": providerErr != nil, "p_observed": sharedRefundObserved(observed), "p_now": clock}
	raw, e := s.call(ctx, "order_refund_finish_create_v1", args)
	if e != nil {
		return Refund{}, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "NEED_RECEIPT_TIME", "stored_kind", "stored_microseconds")
	if e != nil {
		return Refund{}, e
	}
	material, e := sharedRefundReceiptTime(v)
	if e != nil {
		return Refund{}, e
	}
	clock["receipt_time"] = material
	raw, e = s.call(ctx, "order_refund_finish_create_v1", args)
	if e != nil {
		return Refund{}, ErrUnavailable
	}
	v, e = sharedRefundOutcome(raw, "FINISHED", "refund")
	if e != nil {
		return Refund{}, e
	}
	return sharedRefundView(v["refund"])
}
func (s *sharedPGStore) get(ctx context.Context, id uint64) (Refund, error) {
	raw, e := s.call(ctx, "order_refund_read_v1", map[string]any{"p_refund": sharedRefundDecimal(id)})
	if e != nil {
		return Refund{}, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "FOUND", "refund")
	if e != nil {
		return Refund{}, e
	}
	return sharedRefundView(v["refund"])
}
func (s *sharedPGStore) ingest(ctx context.Context, verified VerifiedRefund, now time.Time, _ NotificationEnqueuer) (bool, error) {
	return s.ingestWithClaim(ctx, verified, now, nil)
}
func (s *sharedPGStore) ingestWithClaim(ctx context.Context, verified VerifiedRefund, now time.Time, claim *queryClaim) (bool, error) {
	if err := s.sharedRefundEnvironmentProof(ctx, "REFUND", s.config.AppID, verified.Refund.MerchantID, verified.Refund.OutTradeNo, verified.Refund.OutRefundNo, verified.Refund.RefundID, verified.Refund.TransactionID, nil); err != nil {
		return false, err
	}
	raw, e := s.call(ctx, "order_refund_observation_locator_v1", map[string]any{"p_out_refund_no_hex": hex.EncodeToString([]byte(verified.Refund.OutRefundNo))})
	if e != nil {
		return false, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "FOUND", "locator")
	if e != nil {
		return false, e
	}
	locator, e := sharedRefundObject(v["locator"], "refund_id", "prepayment_id", "order_id")
	if e != nil {
		return false, e
	}
	id, e := sharedRefundID(locator["refund_id"])
	if e != nil {
		return false, e
	}
	if _, e = sharedRefundID(locator["prepayment_id"]); e != nil {
		return false, e
	}
	if !bytes.Equal(locator["order_id"], []byte("null")) {
		if _, e = sharedRefundID(locator["order_id"]); e != nil {
			return false, e
		}
	}
	var claimFact any
	if claim != nil {
		if claim.refundID != id {
			return false, errStaleQueryLease
		}
		claimFact = map[string]any{"refund_id": sharedRefundDecimal(claim.refundID), "owner_hex": hex.EncodeToString(claim.owner[:]), "version": sharedRefundDecimal(claim.version)}
	}
	dedupe := refundObservationDedupe(verified)
	args := map[string]any{"p_locator": json.RawMessage(v["locator"]), "p_source": string(verified.Source), "p_event_hex": hex.EncodeToString([]byte(verified.ProviderEventID)), "p_observed": sharedRefundObserved(verified.Refund), "p_dedupe_hash": hex.EncodeToString(dedupe[:]), "p_claim": claimFact, "p_now": sharedRefundTime(now), "p_enqueue": s.enqueueEnabled}
	for attempt := 0; attempt < 2; attempt++ {
		raw, e = s.call(ctx, "order_refund_observe_v1", args)
		if e == nil {
			v, e = sharedRefundOutcome(raw, "OBSERVED", "applied")
			if e != nil {
				return false, e
			}
			var applied bool
			if bytes.Equal(v["applied"], []byte("null")) || json.Unmarshal(v["applied"], &applied) != nil {
				return false, ErrUnavailable
			}
			return applied, nil
		}
		if !retryablePostgreSQL(e) || attempt == 1 {
			return false, mapStoreError(e)
		}
	}
	return false, ErrUnavailable
}
func (s *sharedPGStore) listDueIDs(ctx context.Context, cutoff, now time.Time, limit uint16) ([]uint64, error) {
	raw, e := s.call(ctx, "order_refund_due_v1", map[string]any{"p_cutoff": sharedRefundTime(cutoff), "p_lease_now": sharedRefundTime(now), "p_limit": limit})
	if e != nil {
		return nil, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "FOUND", "ids")
	if e != nil {
		return nil, e
	}
	var values []json.RawMessage
	if json.Unmarshal(v["ids"], &values) != nil || bytes.Equal(v["ids"], []byte("null")) || len(values) > int(limit) {
		return nil, ErrUnavailable
	}
	var ids []uint64
	for _, value := range values {
		id, e := sharedRefundID(value)
		if e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func (s *sharedPGStore) claimOneQuery(ctx context.Context, id uint64, owner [16]byte, cutoff, now time.Time, _ time.Duration) (queryClaim, bool, error) {
	raw, e := s.call(ctx, "order_refund_claim_query_v1", map[string]any{"p_refund": sharedRefundDecimal(id), "p_owner_hex": hex.EncodeToString(owner[:]), "p_cutoff": sharedRefundTime(cutoff), "p_now": sharedRefundTime(now)})
	if e != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	var hint struct{ Outcome string }
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "NOT_CLAIMED" {
		_, e = sharedRefundOutcome(raw, "NOT_CLAIMED")
		return queryClaim{}, false, e
	}
	v, e := sharedRefundOutcome(raw, "CLAIMED", "version", "out_refund_no_hex")
	if e != nil {
		return queryClaim{}, false, e
	}
	version, e := sharedRefundID(v["version"])
	if e != nil {
		return queryClaim{}, false, e
	}
	out, e := sharedRefundHex(v["out_refund_no_hex"])
	if e != nil {
		return queryClaim{}, false, e
	}
	claim := queryClaim{refundID: id, outRefundNo: string(out), owner: owner, version: version}
	if err := s.sharedRefundEnvironmentProof(ctx, "REFUND", s.config.AppID, s.config.MerchantID, "", string(out), "", "", &claim); err != nil {
		return queryClaim{}, false, err
	}
	return claim, true, nil
}
func (s *sharedPGStore) finishQuery(ctx context.Context, claim queryClaim, observed ProviderRefund, providerErr error, now time.Time, _ NotificationEnqueuer) (bool, bool, error) {
	if providerErr == nil {
		applied, e := s.ingestWithClaim(ctx, VerifiedRefund{Source: SourceActiveQuery, Refund: observed}, now, &claim)
		if errors.Is(e, errStaleQueryLease) {
			return false, false, nil
		}
		return e == nil, applied, e
	}
	raw, e := s.call(ctx, "order_refund_release_query_v1", map[string]any{"p_refund": sharedRefundDecimal(claim.refundID), "p_owner_hex": hex.EncodeToString(claim.owner[:]), "p_version": sharedRefundDecimal(claim.version), "p_now": sharedRefundTime(now)})
	if e != nil {
		return false, false, ErrUnavailable
	}
	_, e = sharedRefundOutcome(raw, "RELEASED")
	return false, false, e
}
func (s *sharedPGStore) listPending(ctx context.Context, owner uint64, filter PendingFilter) ([]Refund, error) {
	raw, e := s.call(ctx, "order_refund_pending_v1", map[string]any{"p_owner": sharedRefundDecimal(owner), "p_after": sharedRefundDecimal(filter.AfterID), "p_limit": filter.Limit})
	if e != nil {
		return nil, ErrUnavailable
	}
	v, e := sharedRefundOutcome(raw, "FOUND", "refunds")
	if e != nil {
		return nil, e
	}
	var values []json.RawMessage
	if json.Unmarshal(v["refunds"], &values) != nil || bytes.Equal(v["refunds"], []byte("null")) || len(values) > int(filter.Limit) {
		return nil, ErrUnavailable
	}
	var out []Refund
	for _, value := range values {
		r, e := sharedRefundView(value)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, nil
}

var _ refundStore = (*sharedPGStore)(nil)

func (s *sharedPGStore) sharedRefundEnvironmentProof(ctx context.Context, kind, appid, merchant, trade, refund, provider, parentProvider string, expectedClaim *queryClaim) error {
	config := s.config
	if appid != config.AppID || merchant != config.MerchantID {
		return ErrUnavailable
	}
	h := func(value string) string { return hex.EncodeToString([]byte(value)) }
	entry := map[string]string{"ordinal": "1", "kind": kind, "appid_hex": h(appid), "mchid_hex": h(merchant), "out_trade_no_hex": h(trade), "out_refund_no_hex": h(refund), "provider_id_hex": h(provider), "payment_provider_id_hex": h(parentProvider)}
	raw, err := s.call(ctx, "order_billing_ownership_facts_v1", map[string]any{"p_environment": config.Environment, "p_entries": []map[string]string{entry}, "p_evidence_cutoff": "0"})
	if err != nil {
		return ErrUnavailable
	}
	reply, err := sharedRefundObject(raw, "outcome", "evidence_cutoff", "entries")
	if err != nil {
		return ErrUnavailable
	}
	outcome, err := sharedRefundString(reply["outcome"])
	if err != nil || outcome != "FOUND" {
		return ErrUnavailable
	}
	cutoff, err := sharedRefundString(reply["evidence_cutoff"])
	if err != nil {
		return ErrUnavailable
	}
	n, err := strconv.ParseUint(cutoff, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != cutoff {
		return ErrUnavailable
	}
	var entries []json.RawMessage
	if json.Unmarshal(reply["entries"], &entries) != nil || len(entries) != 1 {
		return ErrUnavailable
	}
	keys := []string{"entry", "payment_proof", "refund_proof", "confirmations"}
	if expectedClaim != nil {
		keys = append(keys, "resolved_parent_hex", "query_claim")
	}
	fact, err := sharedRefundObject(entries[0], keys...)
	if err != nil {
		return ErrUnavailable
	}
	echo, err := sharedRefundObject(fact["entry"], "ordinal", "kind", "appid_hex", "mchid_hex", "out_trade_no_hex", "out_refund_no_hex", "provider_id_hex", "payment_provider_id_hex")
	if err != nil {
		return ErrUnavailable
	}
	for key, want := range entry {
		got, err := sharedRefundString(echo[key])
		if err != nil || got != want {
			return ErrUnavailable
		}
	}
	if expectedClaim != nil {
		parent, err := sharedRefundHex(fact["resolved_parent_hex"])
		if err != nil || len(parent) == 0 {
			return ErrUnavailable
		}
		lease, err := sharedRefundObject(fact["query_claim"], "version", "owner_hex")
		if err != nil {
			return ErrUnavailable
		}
		version, err := sharedRefundID(lease["version"])
		if err != nil || version != expectedClaim.version {
			return ErrUnavailable
		}
		owner, err := sharedRefundHex(lease["owner_hex"])
		if err != nil || !bytes.Equal(owner, expectedClaim.owner[:]) {
			return ErrUnavailable
		}
		trade = string(parent)
	}
	proofEnvs := map[string]string{}
	proofStates := map[string]string{}
	for _, key := range []string{"payment_proof", "refund_proof"} {
		proof, err := sharedRefundObject(fact[key], "state", "digest_hex", "environment")
		if err != nil {
			return ErrUnavailable
		}
		state, err := sharedRefundString(proof["state"])
		if err != nil {
			return ErrUnavailable
		}
		digest, err := sharedRefundString(proof["digest_hex"])
		if err != nil {
			return ErrUnavailable
		}
		env, err := sharedRefundString(proof["environment"])
		if err != nil {
			return ErrUnavailable
		}
		if state == "CORRUPT" || state == "CONFLICT" || (state != "VALID" && state != "ABSENT") {
			return ErrUnavailable
		}
		if state == "VALID" {
			b, e := hex.DecodeString(digest)
			if e != nil || len(b) != 32 || hex.EncodeToString(b) != digest || (env != "" && env != "uat" && env != "production") {
				return ErrUnavailable
			}
		} else if digest != "" || env != "" {
			return ErrUnavailable
		}
		proofEnvs[key] = env
		proofStates[key] = state
	}
	if expectedClaim != nil && (proofStates["payment_proof"] != "VALID" || proofStates["refund_proof"] != "VALID") {
		return ErrUnavailable
	}
	environments := map[string]bool{}
	ownProof := false
	number, expectedKind := trade, "payment"
	if kind == "REFUND" {
		number, expectedKind = refund, "refund"
	}
	if env, k, ok := wechatpay.ParseMerchantNumber(number); ok {
		if k != expectedKind {
			return ErrUnavailable
		}
		environments[env] = true
		ownProof = true
	}
	if kind == "PAYMENT" {
		if env := proofEnvs["payment_proof"]; env != "" {
			environments[env] = true
		}
	} else {
		if parentEnv, k, ok := wechatpay.ParseMerchantNumber(trade); ok {
			if k != "payment" {
				return ErrUnavailable
			}
			environments[parentEnv] = true
		}
		if proofEnvs["payment_proof"] != "" {
			environments[proofEnvs["payment_proof"]] = true
		}
		if proofStates["refund_proof"] == "VALID" && proofStates["payment_proof"] == "VALID" && proofEnvs["refund_proof"] != "" {
			environments[proofEnvs["refund_proof"]] = true
			if proofEnvs["payment_proof"] == proofEnvs["refund_proof"] {
				ownProof = true
			}
		}
	}
	var confirmations []json.RawMessage
	if json.Unmarshal(fact["confirmations"], &confirmations) != nil || confirmations == nil {
		return ErrUnavailable
	}
	leaves := map[uint64]string{}
	seen := map[uint64]bool{}
	for _, raw := range confirmations {
		c, err := sharedRefundObject(raw, "audit_id", "environment", "source_ref_hex", "evidence_sha256", "supersedes")
		if err != nil {
			return ErrUnavailable
		}
		values := map[string]string{}
		for key, value := range c {
			text, err := sharedRefundString(value)
			if err != nil {
				return ErrUnavailable
			}
			values[key] = text
		}
		id, e1 := strconv.ParseUint(values["audit_id"], 10, 64)
		parent, e2 := strconv.ParseUint(values["supersedes"], 10, 64)
		if e1 != nil || e2 != nil || id == 0 || id > n || parent >= id || seen[id] || strconv.FormatUint(id, 10) != values["audit_id"] || strconv.FormatUint(parent, 10) != values["supersedes"] || (values["environment"] != "uat" && values["environment"] != "production") {
			return ErrUnavailable
		}
		source, e1 := hex.DecodeString(values["source_ref_hex"])
		digest, e2 := hex.DecodeString(values["evidence_sha256"])
		if e1 != nil || e2 != nil || len(source) == 0 || len(digest) != 32 || hex.EncodeToString(source) != values["source_ref_hex"] || hex.EncodeToString(digest) != values["evidence_sha256"] {
			return ErrUnavailable
		}
		if parent != 0 {
			if _, ok := leaves[parent]; !ok {
				return ErrUnavailable
			}
			delete(leaves, parent)
		}
		seen[id] = true
		leaves[id] = values["environment"]
	}
	for _, env := range leaves {
		ownProof = true
		environments[env] = true
	}
	if !ownProof || len(environments) != 1 || !environments[config.Environment] {
		return ErrUnavailable
	}
	return nil
}
