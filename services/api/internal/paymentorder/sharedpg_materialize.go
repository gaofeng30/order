package paymentorder

import (
	"context"
	"encoding/hex"
	"encoding/json"
)

func (a *SharedPGApplication) existingOrder(ctx context.Context, id uint64) (ConfirmResult, bool, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_existing_order_v1", map[string]any{"p_prepayment": decimalID(id)})
	if e != nil {
		return ConfirmResult{}, false, e
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "ABSENT" {
		_, e = sharedPaymentObject(raw, "outcome")
		return ConfirmResult{}, false, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "order_id")
	if e != nil {
		return ConfirmResult{}, false, e
	}
	oid, e := sharedPaymentID(v["order_id"])
	return ConfirmResult{State: ConfirmOrderCreated, OrderID: oid}, e == nil, e
}
func (a *SharedPGApplication) isEnabledOwner(ctx context.Context, id uint64) (bool, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_owner_v1", map[string]any{"p_user": decimalID(id)})
	if e != nil {
		return false, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "enabled")
	if e != nil {
		return false, e
	}
	var enabled bool
	if json.Unmarshal(v["enabled"], &enabled) != nil || sharedPaymentNull(v["enabled"]) {
		return false, ErrUnavailable
	}
	return enabled, nil
}
func (a *SharedPGApplication) listPending(ctx context.Context, filter PendingFilter, page PageQuery) ([]PendingPayment, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_pending_v1", map[string]any{"p_reason_hex": hex.EncodeToString([]byte(filter.Reason)), "p_after": decimalID(page.AfterID), "p_limit": page.Limit})
	if e != nil {
		return nil, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "payments")
	if e != nil {
		return nil, e
	}
	var items []json.RawMessage
	if json.Unmarshal(v["payments"], &items) != nil || items == nil || len(items) > int(page.Limit) {
		return nil, ErrUnavailable
	}
	out := make([]PendingPayment, 0, len(items))
	for _, rawItem := range items {
		v, e := sharedPaymentObject(rawItem, "id", "reason", "created_at")
		if e != nil {
			return nil, e
		}
		var p PendingPayment
		if p.PrepaymentID, e = sharedPaymentID(v["id"]); e != nil {
			return nil, e
		}
		if p.Reason, e = sharedPaymentString(v["reason"]); e != nil {
			return nil, e
		}
		if p.CreatedAt, e = sharedPaymentTime(v["created_at"]); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, nil
}
func (a *SharedPGApplication) applyReadyOnce(ctx context.Context, id uint64, manual bool, owner uint64) (ConfirmResult, bool, error) {
	raw, e := a.sharedPaymentCall(ctx, "order_payment_materialize_v1", map[string]any{"p_prepayment": decimalID(id), "p_manual": manual, "p_owner": decimalID(owner)})
	if e != nil {
		return ConfirmResult{}, false, e
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(raw, &hint)
	switch hint.Outcome {
	case "DEFERRED":
		_, e = sharedPaymentObject(raw, "outcome")
		return ConfirmResult{}, false, e
	case "PENDING":
		_, e = sharedPaymentObject(raw, "outcome")
		return ConfirmResult{State: ConfirmPending}, false, e
	default:
		v, e := sharedPaymentOutcome(raw, "APPLIED", "order_id")
		if e != nil {
			return ConfirmResult{}, false, e
		}
		oid, e := sharedPaymentID(v["order_id"])
		return ConfirmResult{State: ConfirmOrderCreated, OrderID: oid}, e == nil, e
	}
}
