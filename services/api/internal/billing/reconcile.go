package billing

import (
	"context"
	"crypto/sha256"
	"database/sql"
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
	end := date.Add(24 * time.Hour)
	rows, err := service.db.QueryContext(ctx, `SELECT CONVERT(p.out_trade_no USING utf8mb4),CONVERT(o.transaction_id USING utf8mb4),o.payable_cents,p.currency,o.paid_at FROM orders o JOIN prepayments p ON p.id=o.prepayment_id WHERE o.paid_at>=? AND o.paid_at<? ORDER BY p.out_trade_no`, date, end)
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
	rows, err = service.db.QueryContext(ctx, `SELECT CONVERT(out_refund_no USING utf8mb4),CONVERT(provider_refund_id USING utf8mb4),amount_cents,currency,materialized_at FROM refunds WHERE materialization_state='APPLIED' AND materialized_at>=? AND materialized_at<? ORDER BY out_refund_no`, date, end)
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
	lockName := "order:bill:" + hex.EncodeToString(intrinsic[:12])
	connection, err := service.db.Conn(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer connection.Close()
	var acquired int
	if err := connection.QueryRowContext(ctx, `SELECT GET_LOCK(?,5)`, lockName).Scan(&acquired); err != nil || acquired != 1 {
		return ErrUnavailable
	}
	defer func() {
		var released sql.NullInt64
		_ = connection.QueryRowContext(context.Background(), `SELECT RELEASE_LOCK(?)`, lockName).Scan(&released)
	}()
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var exists uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='PROVIDER' AND action='billing.reconcile' AND target_key_hash=? ORDER BY id LIMIT 1`, intrinsic[:]).Scan(&exists)
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
	_, err = tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,target_key_hash,operation_key_hash,result,reason_code,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',?,NULL,NULL,'billing.reconcile','BILL',NULL,?,NULL,?,?,?,?)`, scope[:], intrinsic[:], auditResult, reason, evidence, service.now().UTC())
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
			err = tx.QueryRowContext(ctx, `SELECT id FROM prepayments WHERE out_trade_no=?`, entry.OutTradeNo).Scan(&id)
		} else {
			err = tx.QueryRowContext(ctx, `SELECT id FROM refunds WHERE out_refund_no=?`, entry.OutRefundNo).Scan(&id)
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
		if _, err := tx.ExecContext(ctx, `UPDATE prepayments SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND materialization_state IN ('AWAITING_PAYMENT','READY','PENDING_MANUAL')`, id); err != nil {
			return ErrUnavailable
		}
	}
	for _, id := range orderedRefunds {
		if _, err := tx.ExecContext(ctx, `UPDATE refunds SET materialization_state='PENDING_MANUAL',pending_reason_code='BILL_MISMATCH',materialized_at=NULL,record_version=record_version+1,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND materialization_state<>'APPLIED'`, id); err != nil {
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
