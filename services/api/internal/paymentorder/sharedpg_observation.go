package paymentorder

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"strconv"
	"time"
)

func (a *SharedPGApplication) listDuePrepaymentIDs(ctx context.Context, now time.Time, limit uint16) ([]uint64, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_due_v1", map[string]any{"p_now": now.UTC().Truncate(time.Microsecond), "p_limit": limit})
	if e != nil {
		return nil, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "ids")
	if e != nil {
		return nil, e
	}
	var ids []json.RawMessage
	if json.Unmarshal(v["ids"], &ids) != nil || ids == nil || len(ids) > int(limit) {
		return nil, ErrUnavailable
	}
	out := make([]uint64, 0, len(ids))
	for _, rawID := range ids {
		id, e := sharedPaymentID(rawID)
		if e != nil {
			return nil, e
		}
		out = append(out, id)
	}
	return out, nil
}
func (a *SharedPGApplication) claimQuery(ctx context.Context, id uint64, cutoff time.Time, force bool, owner [16]byte) (queryClaim, bool, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_claim_query_v1", map[string]any{"p_prepayment": decimalID(id), "p_cutoff": cutoff.UTC().Truncate(time.Microsecond), "p_force": force, "p_owner_hex": hex.EncodeToString(owner[:]), "p_lease_ns": strconv.FormatInt(int64(a.config.LeaseDuration), 10)})
	if e != nil {
		return queryClaim{}, false, e
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "NOT_CLAIMED" {
		_, e = sharedPaymentObject(raw, "outcome")
		return queryClaim{}, false, e
	}
	v, e := sharedPaymentOutcome(raw, "CLAIMED", "record", "owner_hex", "version", "expires_at")
	if e != nil {
		return queryClaim{}, false, e
	}
	var c queryClaim
	if c.record, e = sharedPaymentRecord(v["record"]); e != nil {
		return c, false, e
	}
	if c.version, e = sharedPaymentID(v["version"]); e != nil || c.version != c.record.recordVersion {
		return c, false, ErrUnavailable
	}
	b, e := sharedPaymentHex(v["owner_hex"])
	if e != nil || hex.EncodeToString(b) != hex.EncodeToString(owner[:]) || owner == ([16]byte{}) {
		return c, false, ErrUnavailable
	}
	c.owner = owner
	if c.expiresAt, e = sharedPaymentTime(v["expires_at"]); e != nil {
		return c, false, e
	}
	if !paymentRecordInEnvironment(a.config, c.record) {
		return c, false, ErrUnavailable
	}
	return c, true, nil
}
func (a *SharedPGApplication) releaseQuery(ctx context.Context, c queryClaim, next, now time.Time) error {
	return a.finishQuery(ctx, c, next, now, false)
}
func (a *SharedPGApplication) finishClosedQuery(ctx context.Context, c queryClaim, now time.Time) error {
	return a.finishQuery(ctx, c, now, now, true)
}
func (a *SharedPGApplication) finishQuery(ctx context.Context, c queryClaim, next, now time.Time, close bool) error {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_finish_query_v1", map[string]any{"p_prepayment": decimalID(c.record.id), "p_owner_hex": hex.EncodeToString(c.owner[:]), "p_version": decimalID(c.version), "p_now": now.UTC().Truncate(time.Microsecond), "p_next": next.UTC().Truncate(time.Microsecond), "p_close": close})
	if e != nil {
		return ErrUnavailable
	}
	_, e = sharedPaymentOutcome(raw, "FINISHED")
	return e
}
func (a *SharedPGApplication) ingestPaymentOnce(ctx context.Context, verified VerifiedPayment, expectedID uint64, c *queryClaim, receivedAt time.Time) error {
	t := verified.Transaction
	if err := a.sharedPaymentEnvironmentProof(ctx, "PAYMENT", t.AppID, t.MerchantID, t.OutTradeNo, "", t.TransactionID, t.TransactionID); err != nil {
		return err
	}
	at := t.SuccessTime.UTC()
	tx := map[string]any{"appid_hex": hex.EncodeToString([]byte(t.AppID)), "mchid_hex": hex.EncodeToString([]byte(t.MerchantID)), "out_trade_no_hex": hex.EncodeToString([]byte(t.OutTradeNo)), "transaction_id_hex": hex.EncodeToString([]byte(t.TransactionID)), "trade_type_hex": hex.EncodeToString([]byte(t.TradeType)), "trade_state_hex": hex.EncodeToString([]byte(t.TradeState)), "trade_state_desc_hex": hex.EncodeToString([]byte(t.TradeStateDescription)), "bank_type_hex": hex.EncodeToString([]byte(t.BankType)), "attach_hex": hex.EncodeToString([]byte(t.Attach)), "payer_openid_hex": hex.EncodeToString([]byte(t.Payer.OpenID)), "currency_hex": hex.EncodeToString([]byte(t.Amount.Currency)), "payer_currency_hex": hex.EncodeToString([]byte(t.Amount.PayerCurrency)), "total": strconv.FormatInt(t.Amount.Total, 10), "payer_total": strconv.FormatInt(t.Amount.PayerTotal, 10), "success_zero": t.SuccessTime.IsZero(), "success_epoch_seconds": strconv.FormatInt(at.Unix(), 10), "success_nanosecond": at.Nanosecond(), "success_rfc3339nano": at.Format(time.RFC3339Nano)}
	var claim any
	if c != nil {
		claim = map[string]string{"owner_hex": hex.EncodeToString(c.owner[:]), "version": decimalID(c.version)}
	}
	raw, e := a.sharedPaymentCall(ctx, "order_payment_observe_v1", map[string]any{"p_expected_prepayment": decimalID(expectedID), "p_source": string(verified.Source), "p_event_hex": hex.EncodeToString([]byte(verified.ProviderEventID)), "p_transaction": tx, "p_claim": claim, "p_received_at": receivedAt.UTC().Truncate(time.Microsecond), "p_reconcile_ns": strconv.FormatInt(int64(a.config.ReconcileInterval), 10)})
	if e != nil {
		return e
	}
	_, e = sharedPaymentOutcome(raw, "OBSERVED")
	return e
}

func (a *SharedPGApplication) sharedPaymentEnvironmentProof(ctx context.Context, kind, appid, merchant, trade, refund, provider, parentProvider string) error {
	config := a.config
	if appid != config.AppID || merchant != config.MerchantID {
		return ErrUnavailable
	}
	h := func(value string) string { return hex.EncodeToString([]byte(value)) }
	entry := map[string]string{"ordinal": "1", "kind": kind, "appid_hex": h(appid), "mchid_hex": h(merchant), "out_trade_no_hex": h(trade), "out_refund_no_hex": h(refund), "provider_id_hex": h(provider), "payment_provider_id_hex": h(parentProvider)}
	raw, err := a.sharedPaymentCall(ctx, "order_billing_ownership_facts_v1", map[string]any{"p_environment": config.Environment, "p_entries": []map[string]string{entry}, "p_evidence_cutoff": "0"})
	if err != nil {
		return ErrUnavailable
	}
	reply, err := sharedPaymentObject(raw, "outcome", "evidence_cutoff", "entries")
	if err != nil {
		return ErrUnavailable
	}
	outcome, err := sharedPaymentString(reply["outcome"])
	if err != nil || outcome != "FOUND" {
		return ErrUnavailable
	}
	cutoff, err := sharedPaymentString(reply["evidence_cutoff"])
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
	fact, err := sharedPaymentObject(entries[0], "entry", "payment_proof", "refund_proof", "confirmations")
	if err != nil {
		return ErrUnavailable
	}
	echo, err := sharedPaymentObject(fact["entry"], "ordinal", "kind", "appid_hex", "mchid_hex", "out_trade_no_hex", "out_refund_no_hex", "provider_id_hex", "payment_provider_id_hex")
	if err != nil {
		return ErrUnavailable
	}
	for key, want := range entry {
		got, err := sharedPaymentString(echo[key])
		if err != nil || got != want {
			return ErrUnavailable
		}
	}
	proofEnvs := map[string]string{}
	proofStates := map[string]string{}
	for _, key := range []string{"payment_proof", "refund_proof"} {
		proof, err := sharedPaymentObject(fact[key], "state", "digest_hex", "environment")
		if err != nil {
			return ErrUnavailable
		}
		state, err := sharedPaymentString(proof["state"])
		if err != nil {
			return ErrUnavailable
		}
		digest, err := sharedPaymentString(proof["digest_hex"])
		if err != nil {
			return ErrUnavailable
		}
		env, err := sharedPaymentString(proof["environment"])
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
	environments := map[string]bool{}
	number, expectedKind := trade, "payment"
	if kind == "REFUND" {
		number, expectedKind = refund, "refund"
	}
	if env, k, ok := wechatpay.ParseMerchantNumber(number); ok {
		if k != expectedKind {
			return ErrUnavailable
		}
		environments[env] = true
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
		}
	}
	var confirmations []json.RawMessage
	if json.Unmarshal(fact["confirmations"], &confirmations) != nil || confirmations == nil {
		return ErrUnavailable
	}
	leaves := map[uint64]string{}
	seen := map[uint64]bool{}
	for _, raw := range confirmations {
		c, err := sharedPaymentObject(raw, "audit_id", "environment", "source_ref_hex", "evidence_sha256", "supersedes")
		if err != nil {
			return ErrUnavailable
		}
		values := map[string]string{}
		for key, value := range c {
			text, err := sharedPaymentString(value)
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
		environments[env] = true
	}
	if len(environments) != 1 || !environments[config.Environment] {
		return ErrUnavailable
	}
	return nil
}

// The locator supplies only identity; the original complete proof path establishes ownership.
func (a *SharedPGApplication) sharedPaymentRecordEnvironment(ctx context.Context, record prepaymentRecord) bool {
	if paymentRecordInEnvironment(a.config, record) {
		return true
	}
	if !record.validStored() || record.expectedAppID != a.config.AppID || record.expectedMerchantID != a.config.MerchantID {
		return false
	}
	h := func(s string) string { return hex.EncodeToString([]byte(s)) }
	entry := map[string]string{"ordinal": "1", "kind": "PAYMENT", "appid_hex": h(record.expectedAppID), "mchid_hex": h(record.expectedMerchantID), "out_trade_no_hex": h(record.outTradeNo), "out_refund_no_hex": "", "provider_id_hex": "", "payment_provider_id_hex": ""}
	raw, err := a.sharedPaymentCall(ctx, "order_billing_ownership_facts_v1", map[string]any{"p_environment": a.config.Environment, "p_entries": []map[string]string{entry}, "p_evidence_cutoff": "0"})
	if err != nil {
		return false
	}
	reply, err := sharedPaymentObject(raw, "outcome", "evidence_cutoff", "entries")
	if err != nil {
		return false
	}
	outcome, err := sharedPaymentString(reply["outcome"])
	if err != nil || outcome != "FOUND" {
		return false
	}
	var rows []json.RawMessage
	if json.Unmarshal(reply["entries"], &rows) != nil || len(rows) != 1 {
		return false
	}
	fact, err := sharedPaymentObject(rows[0], "entry", "payment_proof", "refund_proof", "confirmations", "resolved_provider_id_hex")
	if err != nil {
		return false
	}
	echo, err := sharedPaymentObject(fact["entry"], "ordinal", "kind", "appid_hex", "mchid_hex", "out_trade_no_hex", "out_refund_no_hex", "provider_id_hex", "payment_provider_id_hex")
	if err != nil {
		return false
	}
	for key, want := range entry {
		got, e := sharedPaymentString(echo[key])
		if e != nil || got != want {
			return false
		}
	}
	id, err := sharedPaymentHex(fact["resolved_provider_id_hex"])
	if err != nil || len(id) == 0 {
		return false
	}
	return a.sharedPaymentEnvironmentProof(ctx, "PAYMENT", record.expectedAppID, record.expectedMerchantID, record.outTradeNo, "", string(id), string(id)) == nil
}
