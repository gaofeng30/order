package billing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

func (service *Service) RunReconcile(ctx context.Context, billDate time.Time, limit uint16) (ReconcileResult, error) {
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
	if cloudBilling(service.config) {
		comparison, evidence, unknown, err := reconcileOwnership(ctx, service.config, bill, system, service.nativeOwnershipFacts, service.nativeSystemFacts)
		if err != nil {
			return ReconcileResult{}, err
		}
		if err = service.persistOwnershipReconcile(ctx, comparison, evidence, unknown); err != nil {
			return ReconcileResult{}, err
		}
		if unknown || len(comparison.ProviderOnly) > 0 || len(comparison.SystemOnly) > 0 {
			return comparison, ErrBillMismatch
		}
		return comparison, nil
	}
	comparison := CompareBill(bill.Entries, system)
	comparison.BillDate, comparison.Digest = date, bill.Digest
	if err := service.persistReconcile(ctx, comparison); err != nil {
		return ReconcileResult{}, err
	}
	if len(comparison.ProviderOnly) > 0 || len(comparison.SystemOnly) > 0 {
		return comparison, ErrBillMismatch
	}
	return comparison, nil
}

func (service *Service) systemBillEntries(ctx context.Context, date time.Time) ([]BillEntry, error) {
	// The provider encodes a calendar bill date at UTC midnight, while its
	// transactions belong to the Shanghai business day.
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)).UTC()
	end := date.Add(24 * time.Hour)
	rows, err := service.db.QueryContext(ctx, `SELECT convert_from(p.out_trade_no,'UTF8'),convert_from(o.transaction_id,'UTF8'),o.payable_cents,p.currency,o.paid_at FROM orders o JOIN prepayments p ON p.id=o.prepayment_id WHERE o.paid_at>=$1 AND o.paid_at<$2 ORDER BY p.out_trade_no`, date, end)
	if err != nil {
		return nil, ErrUnavailable
	}
	entries := make([]BillEntry, 0)
	for rows.Next() {
		var entry BillEntry
		entry.Kind, entry.State = EntryPayment, "SUCCESS"
		if rows.Scan(&entry.OutTradeNo, &entry.ProviderID, &entry.AmountCents, &entry.Currency, &entry.OccurredAt) != nil {
			rows.Close()
			return nil, ErrUnavailable
		}
		entries = append(entries, entry)
	}
	if rows.Err() != nil {
		rows.Close()
		return nil, ErrUnavailable
	}
	rows.Close()
	rows, err = service.db.QueryContext(ctx, `SELECT convert_from(r.out_refund_no,'UTF8'),convert_from(r.provider_refund_id,'UTF8'),r.amount_cents,r.currency,observed.success_time FROM refunds r JOIN (SELECT refund_id,MIN(success_time) AS success_time FROM refund_observations WHERE provider_state='SUCCESS' AND validation='MATCH' AND apply_state='APPLIED' GROUP BY refund_id) observed ON observed.refund_id=r.id WHERE r.materialization_state='APPLIED' AND observed.success_time>=$1 AND observed.success_time<$2 ORDER BY r.out_refund_no`, date, end)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var entry BillEntry
		entry.Kind, entry.State = EntryRefund, "SUCCESS"
		if rows.Scan(&entry.OutRefundNo, &entry.ProviderID, &entry.AmountCents, &entry.Currency, &entry.OccurredAt) != nil {
			return nil, ErrUnavailable
		}
		entries = append(entries, entry)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	sortEntries(entries)
	return entries, nil
}

func (service *Service) persistReconcile(ctx context.Context, result ReconcileResult) error {
	keyMaterial := append([]byte(result.BillDate.Format("2006-01-02")+"\x00"), result.Digest[:]...)
	intrinsic := sha256.Sum256(keyMaterial)
	// Transaction-scoped advisory locking serializes the same bill receipt.
	// Rollback or commit releases it with the transaction, including lost replies.
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SET LOCAL lock_timeout='5s'"); err != nil {
		return ErrUnavailable
	}
	var ignored any
	var acquired bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_advisory_xact_lock($1),TRUE", int64(binary.BigEndian.Uint64(intrinsic[:8]))).Scan(&ignored, &acquired); err != nil || !acquired {
		return ErrUnavailable
	}
	var exists uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND target_key_hash=$1 ORDER BY id LIMIT 1`, intrinsic[:]).Scan(&exists)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	entries := append(append([]BillEntry(nil), result.ProviderOnly...), result.SystemOnly...)
	if err := projectPendingFacts(ctx, tx, entries); err != nil {
		return err
	}
	scope := sha256.Sum256([]byte("PROVIDER:BILLING"))
	evidence, _ := json.Marshal(struct {
		Date         string `json:"date"`
		Digest       string `json:"digest"`
		Matched      uint64 `json:"matched"`
		ProviderOnly uint64 `json:"provider_only"`
		SystemOnly   uint64 `json:"system_only"`
	}{Date: result.BillDate.Format("2006-01-02"), Digest: hex.EncodeToString(result.Digest[:]), Matched: result.Matched, ProviderOnly: uint64(len(result.ProviderOnly)), SystemOnly: uint64(len(result.SystemOnly))})
	reason := "BILL_MATCHED"
	auditResult := "SUCCEEDED"
	if len(result.ProviderOnly) > 0 || len(result.SystemOnly) > 0 {
		reason = "BILL_MISMATCH"
		auditResult = "REJECTED"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,target_key_hash,operation_key_hash,result,reason_code,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',$1,NULL,NULL,'billing.reconcile','BILL',NULL,$2,NULL,$3,$4,$5,$6)`, scope[:], intrinsic[:], auditResult, reason, evidence, service.now().UTC())
	if err != nil {
		return ErrUnavailable
	}
	if err := tx.Commit(); err != nil {
		return ErrUnavailable
	}
	return nil
}

func projectPendingFacts(ctx context.Context, tx *sql.Tx, entries []BillEntry) error {
	paymentIDs := make(map[uint64]struct{})
	refundIDs := make(map[uint64]struct{})
	for _, entry := range entries {
		var id uint64
		var err error
		if entry.Kind == EntryPayment {
			err = tx.QueryRowContext(ctx, `SELECT id FROM prepayments WHERE out_trade_no=$1`, []byte(entry.OutTradeNo)).Scan(&id)
		} else {
			err = tx.QueryRowContext(ctx, `SELECT id FROM refunds WHERE out_refund_no=$1`, []byte(entry.OutRefundNo)).Scan(&id)
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return ErrUnavailable
		}
		if entry.Kind == EntryPayment {
			paymentIDs[id] = struct{}{}
		} else {
			refundIDs[id] = struct{}{}
		}
	}
	orderedPayments := sortedIDs(paymentIDs)
	orderedRefunds := sortedIDs(refundIDs)
	for _, id := range orderedPayments {
		if _, err := tx.ExecContext(ctx, `UPDATE prepayments SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=$1 AND materialization_state IN ('AWAITING_PAYMENT','READY','PENDING_MANUAL')`, id); err != nil {
			return ErrUnavailable
		}
	}
	for _, id := range orderedRefunds {
		if _, err := tx.ExecContext(ctx, `UPDATE refunds SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=$1 AND materialization_state<>'APPLIED'`, id); err != nil {
			return ErrUnavailable
		}
	}
	return nil
}

func sortedIDs(set map[uint64]struct{}) []uint64 {
	ids := make([]uint64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func validBillEntries(entries []BillEntry) bool {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key := entryKey(entry)
		if key == "" || entry.ProviderID == "" || entry.AmountCents == 0 || entry.Currency != "CNY" || entry.State != "SUCCESS" {
			return false
		}
		if entry.Kind == EntryPayment && (entry.OutTradeNo == "" || entry.OutRefundNo != "") {
			return false
		}
		if entry.Kind == EntryRefund && (entry.OutRefundNo == "" || entry.OutTradeNo != "") {
			return false
		}
		if _, ok := seen[key]; ok {
			return false
		}
		seen[key] = struct{}{}
	}
	return true
}

func ownershipReceiptIdentity(e ownershipEvidence) ([32]byte, error) {
	raw, err := json.Marshal(struct {
		RuleVersion    string `json:"rule_version"`
		Environment    string `json:"environment"`
		MerchantID     string `json:"merchant_id"`
		Date           string `json:"date"`
		BillDigest     string `json:"bill_digest"`
		EvidenceDigest string `json:"evidence_digest"`
	}{e.RuleVersion, e.Environment, e.MerchantID, e.Date, e.BillDigest, e.EvidenceDigest})
	if err != nil {
		return [32]byte{}, ErrUnavailable
	}
	return sha256.Sum256(raw), nil
}
func (service *Service) persistOwnershipReconcile(ctx context.Context, result ReconcileResult, evidence ownershipEvidence, unknown bool) error {
	intrinsic, err := ownershipReceiptIdentity(evidence)
	if err != nil {
		return err
	}
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SET LOCAL lock_timeout='5s'"); err != nil {
		return ErrUnavailable
	}
	var ignored any
	var acquired bool
	if err = tx.QueryRowContext(ctx, "SELECT pg_advisory_xact_lock($1),TRUE", int64(binary.BigEndian.Uint64(intrinsic[:8]))).Scan(&ignored, &acquired); err != nil || !acquired {
		return ErrUnavailable
	}
	// Recheck every consumed identity in this transaction before deduplication/projection.
	cutoff, ok := canonicalOwnershipUint(evidence.EvidenceCutoff)
	if !ok {
		return ErrUnavailable
	}
	if len(evidence.Details) == 0 {
		_, fixed, e := service.nativeOwnershipFactsWith(ctx, tx, nil, cutoff)
		if e != nil || fixed != cutoff {
			return ErrUnavailable
		}
	}
	for start := 0; start < len(evidence.Details); start += 100 {
		end := start + 100
		if end > len(evidence.Details) {
			end = len(evidence.Details)
		}
		entries := make([]ownershipEntry, 0, end-start)
		expected := make([]ownershipFact, 0, end-start)
		for _, d := range evidence.Details[start:end] {
			entries = append(entries, d.Proof.Entry)
			expected = append(expected, d.Proof)
		}
		actual, fixed, e := service.nativeOwnershipFactsWith(ctx, tx, entries, cutoff)
		if e != nil || fixed != cutoff || !sameOwnershipFacts(expected, actual) {
			return ErrUnavailable
		}
	}
	actual, e := service.nativeSystemFactsWith(ctx, tx, result.SystemOnly, cutoff)
	if e != nil || !sameOwnershipFacts(evidence.SystemFacts, actual) {
		return ErrUnavailable
	}
	entries := append([]BillEntry(nil), result.ProviderOnly...)
	for _, entry := range result.SystemOnly {
		for _, fact := range evidence.SystemFacts {
			if systemFactMatches(entry, fact) && systemFactOwned(service.config, fact) {
				entries = append(entries, entry)
				break
			}
		}
	}
	var exists uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND target_key_hash=$1 ORDER BY id LIMIT 1`, intrinsic[:]).Scan(&exists)
	if err == nil {
		if tx.Commit() != nil {
			return ErrUnavailable
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ErrUnavailable
	}
	if err = projectPendingFacts(ctx, tx, entries); err != nil {
		return err
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return ErrUnavailable
	}
	scope := sha256.Sum256([]byte("PROVIDER:BILLING"))
	reason, auditResult := "BILL_MATCHED", "SUCCEEDED"
	if len(result.ProviderOnly) > 0 || len(result.SystemOnly) > 0 {
		reason, auditResult = "BILL_MISMATCH", "REJECTED"
	}
	if unknown {
		reason, auditResult = "BILL_OWNERSHIP_UNKNOWN", "REJECTED"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,target_key_hash,operation_key_hash,result,reason_code,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',$1,NULL,NULL,'billing.reconcile','BILL',NULL,$2,NULL,$3,$4,$5,$6)`, scope[:], intrinsic[:], auditResult, reason, raw, service.now().UTC())
	if err != nil {
		return ErrUnavailable
	}
	if tx.Commit() != nil {
		return ErrUnavailable
	}
	return nil
}
