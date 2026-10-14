package billing

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"
)

func (service *SharedService) RunReconcile(ctx context.Context, billDate time.Time, limit uint16) (ReconcileResult, error) {
	if !service.valid() || service.provider == nil || billDate.IsZero() || limit == 0 || limit > 100 {
		return ReconcileResult{}, ErrInvalidInput
	}
	date := normalizedBillDate(billDate)
	if result, done, err := service.completed(ctx, date); err != nil || done {
		return result, err
	}
	bill, err := service.provider.DownloadTransactionBill(ctx, date)
	if err != nil {
		return ReconcileResult{}, ErrBillUnavailable
	}
	if !normalizedBillDate(bill.Date).Equal(date) || !validBillEntries(bill.Entries) {
		return ReconcileResult{}, ErrBillUnavailable
	}
	system, err := service.systemBillEntries(ctx, date)
	if err != nil {
		return ReconcileResult{}, err
	}
	comparison, evidence, unknown, err := reconcileOwnership(ctx, service.config, bill, system, service.sharedOwnershipFacts, service.sharedSystemFacts)
	if err != nil {
		return ReconcileResult{}, err
	}
	comparison.BillDate, comparison.Digest = date, bill.Digest
	if err := service.persistOwnershipReconcile(ctx, comparison, evidence); err != nil {
		return ReconcileResult{}, err
	}
	if unknown || len(comparison.ProviderOnly) > 0 || len(comparison.SystemOnly) > 0 {
		return comparison, ErrBillMismatch
	}
	return comparison, nil
}

func (s *SharedService) systemBillEntries(ctx context.Context, date time.Time) ([]BillEntry, error) {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)).UTC()
	end := date.Add(24 * time.Hour)
	args := map[string]any{"p_from": billingTimeWire(date), "p_to": billingTimeWire(end)}
	entries := make([]BillEntry, 0)
	for _, stage := range []struct {
		name string
		kind EntryKind
	}{{"order_billing_payment_facts_v1", EntryPayment}, {"order_billing_refund_facts_v1", EntryRefund}} {
		v, err := s.call(ctx, stage.name, args, "entries")
		if err != nil {
			return nil, err
		}
		var rows []json.RawMessage
		if json.Unmarshal(v["entries"], &rows) != nil || rows == nil {
			return nil, ErrUnavailable
		}
		for _, row := range rows {
			f, e := billingObject(row, "number_hex", "provider_id_hex", "amount_cents", "currency", "occurred_at")
			if e != nil {
				return nil, e
			}
			entry := BillEntry{Kind: stage.kind, State: "SUCCESS"}
			number, e := billingBytes(f["number_hex"])
			if e != nil {
				return nil, e
			}
			if stage.kind == EntryPayment {
				entry.OutTradeNo = number
			} else {
				entry.OutRefundNo = number
			}
			entry.ProviderID, e = billingBytes(f["provider_id_hex"])
			if e != nil {
				return nil, e
			}
			entry.AmountCents, e = billingUint(f["amount_cents"])
			if e != nil {
				return nil, e
			}
			entry.Currency, e = billingText(f["currency"])
			if e != nil {
				return nil, e
			}
			entry.OccurredAt, e = billingScanTime(f["occurred_at"])
			if e != nil {
				return nil, e
			}
			entries = append(entries, entry)
		}
	}
	sortEntries(entries)
	return entries, nil
}
func (s *SharedService) persistOwnershipReconcile(ctx context.Context, result ReconcileResult, evidence ownershipEvidence) error {
	material := func(entries []BillEntry) []map[string]any {
		v := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			v = append(v, map[string]any{"kind": string(e.Kind), "out_trade_no_hex": hex.EncodeToString([]byte(e.OutTradeNo)), "out_refund_no_hex": hex.EncodeToString([]byte(e.OutRefundNo))})
		}
		return v
	}
	_, err := s.call(ctx, "order_billing_reconcile_v2", map[string]any{"p_environment": s.config.Environment, "p_evidence": evidence, "p_date": result.BillDate.Format("2006-01-02"), "p_digest_hex": hex.EncodeToString(result.Digest[:]), "p_matched": billingDecimal(result.Matched), "p_provider_only": material(result.ProviderOnly), "p_system_only": material(result.SystemOnly)})
	return err
}
