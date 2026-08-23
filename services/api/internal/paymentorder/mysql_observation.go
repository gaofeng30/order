package paymentorder

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
)

func (service *Service) IngestPayment(ctx context.Context, verified VerifiedPayment) error {
	return service.ingestPayment(ctx, verified, 0, nil)
}

func (service *Service) ingestPayment(ctx context.Context, verified VerifiedPayment, expectedPrepaymentID uint64, claim *queryClaim) error {
	if !service.ready() {
		return ErrUnavailable
	}
	if verified.Source != paymentobservation.SourceCallback && verified.Source != paymentobservation.SourceActiveQuery {
		return ErrInvalidInput
	}
	now := service.now().UTC().Truncate(time.Microsecond)
	for attempt := 0; attempt < 2; attempt++ {
		err := service.ingestPaymentOnce(ctx, verified, expectedPrepaymentID, claim, now)
		if err == nil {
			return nil
		}
		if !isRetryableMySQL(err) || attempt == 1 {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidInput) {
				return err
			}
			return ErrUnavailable
		}
	}
	return ErrUnavailable
}

func (service *Service) ingestPaymentOnce(ctx context.Context, verified VerifiedPayment, expectedPrepaymentID uint64, claim *queryClaim, receivedAt time.Time) error {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	var locator any = verified.Transaction.OutTradeNo
	query := prepaymentSelect + ` WHERE out_trade_no=? FOR UPDATE`
	if expectedPrepaymentID > 0 {
		locator = expectedPrepaymentID
		query = prepaymentSelect + ` WHERE id=? FOR UPDATE`
	}
	record, err := scanPrepayment(transaction.QueryRowContext(ctx, query, locator))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil || !record.validStored() {
		return ErrUnavailable
	}
	normalized, err := paymentobservation.Normalize(paymentobservation.Expectation{
		AppID: record.expectedAppID, MerchantID: record.expectedMerchantID,
		OutTradeNo: record.outTradeNo, TotalAmount: record.expectedAmountCents, Currency: record.currency,
	}, paymentobservation.Input{Source: verified.Source, Transaction: verified.Transaction})
	if err != nil {
		return ErrInvalidInput
	}
	dedupeKey, err := hex.DecodeString(normalized.DedupeKey)
	if err != nil || len(dedupeKey) != 32 {
		return ErrUnavailable
	}
	mode := DecideMaterializationMode(normalized, record.effectiveDeadline)
	validation := "MATCH"
	var mismatchCode any
	if normalized.Validation != paymentobservation.ValidationAccepted {
		validation = "MISMATCH"
		mismatchCode = string(normalized.Mismatch)
	}
	providerState := string(normalized.State)
	var transactionID, amountCents, currency, successTime any
	if normalized.State == paymentobservation.StatePaid {
		transactionID = []byte(verified.Transaction.TransactionID)
		amountCents = verified.Transaction.Amount.Total
		currency = verified.Transaction.Amount.Currency
		successTime = verified.Transaction.SuccessTime.UTC().Truncate(time.Microsecond)
	}
	applyState := "NEW"
	var applyReason any
	if mode == MaterializationDelayedManual {
		applyState = "DEFERRED"
		applyReason = delayedReason(normalized)
	}
	var eventID any
	if verified.ProviderEventID != "" {
		eventID = []byte(verified.ProviderEventID)
	}
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO payment_observations(
			prepayment_id,dedupe_key,source,provider_event_id,out_trade_no,transaction_id,
			provider_state,validation,mismatch_code,amount_cents,currency,success_time,received_at,
			materialization_mode,apply_state,apply_reason_code,applied_at,record_version
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NULL,1)
	`, record.id, dedupeKey, string(verified.Source), eventID, []byte(record.outTradeNo), transactionID,
		providerState, validation, mismatchCode, amountCents, currency, successTime, receivedAt,
		string(mode), applyState, applyReason)
	if isDuplicate(err) {
		var existingPrepaymentID uint64
		if readErr := transaction.QueryRowContext(ctx, `SELECT prepayment_id FROM payment_observations WHERE dedupe_key=?`, dedupeKey).Scan(&existingPrepaymentID); readErr == nil && existingPrepaymentID == record.id {
			err = nil
		} else {
			return ErrUnavailable
		}
	}
	if err != nil {
		return err
	}

	nextProviderState := advanceProviderState(record.providerState, normalized.State)
	nextMaterialState := record.materialState
	var pendingReason any
	if record.pendingReason.Valid {
		pendingReason = record.pendingReason.String
	}
	if record.materialState != MaterializationApplied {
		switch {
		case mode == MaterializationAuto:
			nextMaterialState = MaterializationReady
			pendingReason = nil
		case normalized.State == paymentobservation.StatePaid || normalized.Validation != paymentobservation.ValidationAccepted:
			nextMaterialState = MaterializationPendingManual
			pendingReason = delayedReason(normalized)
		}
	}
	var nextReconcile any
	if nextMaterialState == MaterializationReady {
		nextReconcile = receivedAt
	} else if nextMaterialState == MaterializationAwaitingPayment && nextProviderState != ProviderClosed {
		nextReconcile = receivedAt.Add(service.config.ReconcileInterval)
	}
	var lastQueried any
	if record.lastQueriedAt.Valid {
		lastQueried = record.lastQueriedAt.Time
	}
	if verified.Source == paymentobservation.SourceActiveQuery {
		lastQueried = receivedAt
	}
	var leaseKind, leaseOwner, leaseExpiresAt any
	preserveLease := false
	if verified.Source == paymentobservation.SourceActiveQuery {
		if record.leaseKind.Valid && record.leaseKind.String != "QUERY" {
			return ErrUnavailable
		}
		claimMatches := queryClaimOwns(record, claim)
		preserveLease = !claimMatches && record.leaseKind.Valid
	}
	if preserveLease {
		leaseKind = record.leaseKind.String
		leaseOwner = record.leaseOwner
		if record.leaseExpiresAt.Valid {
			leaseExpiresAt = record.leaseExpiresAt.Time
		}
		if record.nextReconcileAt.Valid {
			nextReconcile = record.nextReconcileAt.Time
		}
	}
	update, err := transaction.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state=?,last_queried_at=?,materialization_state=?,pending_reason_code=?,
		    lease_kind=?,lease_owner=?,lease_expires_at=?,record_version=record_version+1,
		    next_reconcile_at=?,updated_at=?
		WHERE id=? AND record_version=?
	`, string(nextProviderState), lastQueried, string(nextMaterialState), pendingReason,
		leaseKind, leaseOwner, leaseExpiresAt, nextReconcile, receivedAt,
		record.id, record.recordVersion)
	if err != nil {
		return err
	}
	rows, err := update.RowsAffected()
	if err != nil || rows != 1 {
		return ErrUnavailable
	}
	return transaction.Commit()
}

func queryClaimOwns(record prepaymentRecord, claim *queryClaim) bool {
	return claim != nil && record.leaseKind.Valid && record.leaseKind.String == "QUERY" &&
		len(record.leaseOwner) == len(claim.owner) && bytes.Equal(record.leaseOwner, claim.owner[:]) &&
		record.recordVersion == claim.version
}

func delayedReason(observation paymentobservation.Observation) string {
	if observation.Validation != paymentobservation.ValidationAccepted {
		return "PAYMENT_" + string(observation.Mismatch) + "_MISMATCH"
	}
	if observation.State != paymentobservation.StatePaid {
		return "PAYMENT_NOT_SUCCESSFUL"
	}
	return "PAYMENT_AT_OR_AFTER_DEADLINE"
}

func advanceProviderState(current ProviderState, observed paymentobservation.State) ProviderState {
	if current == ProviderPaid {
		return current
	}
	if observed == paymentobservation.StatePaid {
		return ProviderPaid
	}
	if current == ProviderClosed {
		return current
	}
	if observed == paymentobservation.StateClosed {
		return ProviderClosed
	}
	return ProviderNotPaid
}

type queryClaim struct {
	record  prepaymentRecord
	owner   [16]byte
	version uint64
}

func (service *Service) queryOne(ctx context.Context, prepaymentID uint64, now time.Time, force bool) (bool, error) {
	owner, err := service.leaseOwner()
	if err != nil || owner == ([16]byte{}) {
		return false, ErrUnavailable
	}
	var claim queryClaim
	var claimed bool
	for attempt := 0; attempt < 2; attempt++ {
		claim, claimed, err = service.claimQuery(ctx, prepaymentID, now, force, owner)
		if err == nil {
			break
		}
		if !isRetryableMySQL(err) || attempt == 1 {
			return false, ErrUnavailable
		}
	}
	if !claimed {
		return false, nil
	}
	providerTransaction, err := service.provider.QueryTransaction(ctx, claim.record.outTradeNo)
	if err != nil {
		if !now.Before(claim.record.effectiveDeadline) {
			_ = service.closeExpiredClaim(ctx, claim, now)
		}
		_ = service.releaseQuery(ctx, claim, now.Add(service.config.ReconcileInterval), now)
		return false, ErrUnavailable
	}
	if providerTransaction.TradeState == "NOTPAY" && !now.Before(claim.record.effectiveDeadline) {
		if err := service.provider.CloseTransaction(ctx, claim.record.outTradeNo); err == nil {
			providerTransaction.TradeState = "CLOSED"
		}
	}
	err = service.ingestPayment(ctx, VerifiedPayment{Source: paymentobservation.SourceActiveQuery, Transaction: providerTransaction}, claim.record.id, &claim)
	if err != nil {
		_ = service.releaseQuery(ctx, claim, now.Add(service.config.ReconcileInterval), now)
		return false, err
	}
	return true, nil
}

func (service *Service) closeExpiredClaim(ctx context.Context, claim queryClaim, now time.Time) error {
	if err := service.provider.CloseTransaction(ctx, claim.record.outTradeNo); err != nil && !errors.Is(err, ErrNotFound) {
		return ErrUnavailable
	}
	update, err := service.db.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state='CLOSED',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,next_reconcile_at=NULL,updated_at=?
		WHERE id=? AND lease_kind='QUERY' AND lease_owner=? AND record_version=?
	`, now, claim.record.id, claim.owner[:], claim.version)
	if err != nil {
		return ErrUnavailable
	}
	_, _ = update.RowsAffected()
	return nil
}

func (service *Service) claimQuery(ctx context.Context, prepaymentID uint64, now time.Time, force bool, owner [16]byte) (queryClaim, bool, error) {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return queryClaim{}, false, err
	}
	defer transaction.Rollback()
	record, err := scanPrepayment(transaction.QueryRowContext(ctx, prepaymentSelect+` WHERE id=? FOR UPDATE`, prepaymentID))
	if errors.Is(err, sql.ErrNoRows) {
		return queryClaim{}, false, ErrNotFound
	}
	if err != nil || !record.validStored() {
		return queryClaim{}, false, ErrUnavailable
	}
	if record.providerState == ProviderPaid || record.providerState == ProviderClosed || record.materialState == MaterializationApplied || record.materialState == MaterializationPendingManual {
		return queryClaim{}, false, transaction.Commit()
	}
	if record.providerState == ProviderReady {
		return queryClaim{}, false, ErrUnavailable
	}
	if record.leaseKind.Valid && record.leaseExpiresAt.Valid && record.leaseExpiresAt.Time.After(now) {
		return queryClaim{}, false, transaction.Commit()
	}
	if !force && record.nextReconcileAt.Valid && record.nextReconcileAt.Time.After(now) {
		return queryClaim{}, false, transaction.Commit()
	}
	nextState := record.providerState
	if record.providerState == ProviderCreateClaimed {
		nextState = ProviderCreateUnknown
	}
	leaseExpiry := now.Add(service.config.LeaseDuration)
	update, err := transaction.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state=?,lease_kind='QUERY',lease_owner=?,lease_expires_at=?,
		    record_version=record_version+1,next_reconcile_at=?,updated_at=?
		WHERE id=? AND record_version=?
	`, string(nextState), owner[:], leaseExpiry, leaseExpiry, now, record.id, record.recordVersion)
	if err != nil {
		return queryClaim{}, false, err
	}
	rows, err := update.RowsAffected()
	if err != nil || rows != 1 {
		return queryClaim{}, false, ErrUnavailable
	}
	if err := transaction.Commit(); err != nil {
		return queryClaim{}, false, err
	}
	record.providerState = nextState
	record.recordVersion++
	return queryClaim{record: record, owner: owner, version: record.recordVersion}, true, nil
}

func (service *Service) releaseQuery(ctx context.Context, claim queryClaim, next, now time.Time) error {
	update, err := service.db.ExecContext(ctx, `
		UPDATE prepayments
		SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,
		    next_reconcile_at=?,updated_at=?
		WHERE id=? AND lease_kind='QUERY' AND lease_owner=? AND record_version=?
	`, next, now, claim.record.id, claim.owner[:], claim.version)
	if err != nil {
		return ErrUnavailable
	}
	_, _ = update.RowsAffected()
	return nil
}

func (service *Service) listDuePrepaymentIDs(ctx context.Context, now time.Time, limit uint16) ([]uint64, error) {
	rows, err := service.db.QueryContext(ctx, `
		SELECT id FROM prepayments
		WHERE materialization_state='READY'
		   OR (materialization_state='AWAITING_PAYMENT' AND next_reconcile_at<=?)
		   OR (materialization_state='AWAITING_PAYMENT' AND lease_expires_at<=?)
		ORDER BY id LIMIT ?
	`, now, now, limit)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	ids := make([]uint64, 0, limit)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrUnavailable
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}
	return ids, nil
}
