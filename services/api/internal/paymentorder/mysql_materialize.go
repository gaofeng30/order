package paymentorder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
	"github.com/gaofeng30/order/services/api/internal/quote"
)

type observationRecord struct {
	id                  uint64
	prepaymentID        uint64
	outTradeNo          string
	transactionID       string
	providerState       string
	validation          string
	amountCents         int64
	currency            string
	successTime         time.Time
	materializationMode MaterializationMode
	applyState          string
	applyReason         sql.NullString
	recordVersion       uint64
}

func (service *Service) applyReady(ctx context.Context, prepaymentID uint64, manual bool, manualOwnerUserID uint64) (ConfirmResult, bool, error) {
	for attempt := 0; attempt < 2; attempt++ {
		result, applied, err := service.applyReadyOnce(ctx, prepaymentID, manual, manualOwnerUserID)
		if err == nil {
			return result, applied, nil
		}
		if (!isRetryableMySQL(err) && !errors.Is(err, quote.ErrUnavailable)) || attempt == 1 {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrInvalidInput) {
				return ConfirmResult{}, false, err
			}
			return ConfirmResult{}, false, ErrUnavailable
		}
	}
	return ConfirmResult{}, false, ErrUnavailable
}

func (service *Service) applyReadyOnce(ctx context.Context, prepaymentID uint64, manual bool, manualOwnerUserID uint64) (ConfirmResult, bool, error) {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return ConfirmResult{}, false, err
	}
	defer transaction.Rollback()

	var observationID uint64
	query := `SELECT id FROM payment_observations WHERE prepayment_id=? AND apply_state='NEW' AND materialization_mode='AUTO' ORDER BY id LIMIT 1`
	if manual {
		query = `SELECT id FROM payment_observations WHERE prepayment_id=? AND provider_state='PAID' AND validation='MATCH' AND apply_state IN ('NEW','DEFERRED') ORDER BY id LIMIT 1`
	}
	err = transaction.QueryRowContext(ctx, query, prepaymentID).Scan(&observationID)
	if errors.Is(err, sql.ErrNoRows) {
		var orderID uint64
		orderErr := transaction.QueryRowContext(ctx, `SELECT id FROM orders WHERE prepayment_id=?`, prepaymentID).Scan(&orderID)
		if orderErr == nil {
			return ConfirmResult{State: ConfirmOrderCreated, OrderID: orderID}, true, transaction.Commit()
		}
		if !errors.Is(orderErr, sql.ErrNoRows) {
			return ConfirmResult{}, false, orderErr
		}
		return ConfirmResult{State: ConfirmPending}, false, transaction.Commit()
	}
	if err != nil {
		return ConfirmResult{}, false, err
	}
	if manual {
		var accountID uint64
		err := transaction.QueryRowContext(ctx, `
			SELECT id FROM merchant_accounts
			WHERE bound_user_id=? AND role='OWNER' AND enabled=1 AND deleted_at IS NULL
			ORDER BY id LIMIT 1 FOR UPDATE
		`, manualOwnerUserID).Scan(&accountID)
		if errors.Is(err, sql.ErrNoRows) {
			return ConfirmResult{}, false, ErrForbidden
		}
		if err != nil || accountID == 0 {
			return ConfirmResult{}, false, ErrUnavailable
		}
	}

	prepayment, err := scanPrepayment(transaction.QueryRowContext(ctx, prepaymentSelect+` WHERE id=? FOR UPDATE`, prepaymentID))
	if errors.Is(err, sql.ErrNoRows) {
		return ConfirmResult{}, false, ErrNotFound
	}
	if err != nil || !prepayment.validStored() {
		return ConfirmResult{}, false, ErrUnavailable
	}
	observation, err := scanObservation(transaction.QueryRowContext(ctx, `
		SELECT id,prepayment_id,out_trade_no,transaction_id,provider_state,validation,
		       amount_cents,currency,success_time,materialization_mode,apply_state,apply_reason_code,record_version
		FROM payment_observations WHERE id=? FOR UPDATE
	`, observationID))
	if err != nil {
		return ConfirmResult{}, false, err
	}
	if observation.prepaymentID != prepayment.id || observation.providerState != "PAID" || observation.validation != "MATCH" ||
		observation.transactionID == "" || observation.amountCents <= 0 || observation.currency != "CNY" || observation.successTime.IsZero() {
		return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "PAYMENT_FACT_INVALID")
	}
	if !manual && (observation.materializationMode != MaterializationAuto || observation.applyState != "NEW") {
		return ConfirmResult{State: ConfirmPending}, false, transaction.Commit()
	}

	snapshot, snapshotErr := service.quotes.LoadSnapshotInTx(ctx, transaction, prepayment.quoteID)
	if errors.Is(snapshotErr, quote.ErrUnavailable) {
		return ConfirmResult{}, false, quote.ErrUnavailable
	}
	if snapshotErr != nil {
		if errors.Is(snapshotErr, quote.ErrSnapshotInvalid) || errors.Is(snapshotErr, quote.ErrNotFound) {
			return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "QUOTE_SNAPSHOT_INVALID")
		}
		return ConfirmResult{}, false, quote.ErrUnavailable
	}
	pickupAt, err := pickupInstant(snapshot.Pickup.Date, snapshot.Pickup.Time)
	if err != nil {
		return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "QUOTE_SNAPSHOT_INVALID")
	}
	if err := validateMaterializationFacts(prepayment, observation, snapshot, pickupAt, manual); err != nil {
		return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "PAYMENT_FACT_INVALID")
	}

	materializedAt := service.now().UTC().Truncate(time.Microsecond)
	if materializedAt.Before(observation.successTime) {
		materializedAt = observation.successTime
	}
	initial, err := orderproduction.InitialState(observation.successTime, pickupAt)
	if err != nil {
		return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "PAYMENT_TIME_INVALID")
	}
	decision, err := orderproduction.Advance(initial, materializedAt, pickupAt)
	if err != nil {
		return ConfirmResult{}, false, service.deferUnsafeObservation(ctx, transaction, prepayment, observation, "PAYMENT_TIME_INVALID")
	}
	state := decision.State
	var preparingAt any
	if state == orderproduction.StatePreparing {
		preparingAt = materializedAt
	}

	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO pickup_sequences(service_date,last_number,updated_at)
		VALUES (?,LAST_INSERT_ID(1),?)
		ON DUPLICATE KEY UPDATE last_number=LAST_INSERT_ID(last_number+1),updated_at=VALUES(updated_at)
	`, snapshot.Pickup.Date, materializedAt); err != nil {
		return ConfirmResult{}, false, err
	}
	var pickupNumber uint64
	if err := transaction.QueryRowContext(ctx, `SELECT LAST_INSERT_ID()`).Scan(&pickupNumber); err != nil || pickupNumber == 0 || pickupNumber > 9999 {
		return ConfirmResult{}, false, ErrUnavailable
	}
	orderNo := fmt.Sprintf("%s-%04d", withoutDashes(snapshot.Pickup.Date), pickupNumber)
	insert, err := transaction.ExecContext(ctx, `
		INSERT INTO orders(
			order_no,user_id,quote_id,prepayment_id,payment_observation_id,
			contact_name_snapshot,contact_phone_snapshot,identity_kind,identity_source_version,
			discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,
			pickup_date,pickup_time,pickup_at,meal_period,order_note,item_count,
			original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,transaction_id,paid_at,materialized_at,
			pickup_number,state,preparing_at,ready_at,completed_at,refunding_at,refunded_at,
			redemption_token_ciphertext,redemption_token_hash,redemption_key_version,redemption_issued_at,
			redeemed_by_account_id,redeemed_at,record_version,created_at,updated_at
		) VALUES (?,?,?,?,? ,?,?,?,?,? ,?,?,?,?,? ,?,?,?,?,? ,?,?,?,?,? ,?,?,? ,?,?,NULL,NULL,NULL,NULL ,NULL,NULL,NULL,NULL,NULL,NULL,1,?,?)
	`, []byte(orderNo), snapshot.UserID, snapshot.ID, prepayment.id, observation.id,
		snapshot.Contact.Name, []byte(snapshot.Contact.Phone), string(snapshot.Identity.Kind), snapshot.Identity.SourceVersion,
		snapshot.Discount.RatePercent, snapshot.Discount.Version, snapshot.Store.Name, snapshot.Store.Address, snapshot.Pickup.Point,
		snapshot.Pickup.Date, snapshot.Pickup.Time+":00", pickupAt, snapshot.Pickup.Meal, snapshot.OrderNote, len(snapshot.Items),
		snapshot.OriginalSubtotalCents, snapshot.DiscountCents, snapshot.PayableCents, snapshot.SnapshotDigest[:], []byte(observation.transactionID), observation.successTime, materializedAt,
		pickupNumber, string(state), preparingAt, materializedAt, materializedAt)
	if err != nil {
		return ConfirmResult{}, false, err
	}
	orderID, err := insert.LastInsertId()
	if err != nil || orderID <= 0 {
		return ConfirmResult{}, false, ErrUnavailable
	}
	for _, item := range snapshot.Items {
		flavors, err := json.Marshal(item.Flavors)
		if err != nil {
			return ConfirmResult{}, false, ErrUnavailable
		}
		var imageObjectKey any
		if item.ImageObjectKey != "" {
			imageObjectKey = []byte(item.ImageObjectKey)
		}
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO order_items(
				order_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,
				original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,
				payable_subtotal_cents,flavors_json,line_note
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		`, uint64(orderID), item.LineNumber, item.ProductID, item.ProductName, item.ProductSourceVersion[:], imageObjectKey,
			item.OriginalUnitPriceCents, item.DiscountedUnitPriceCents, item.Quantity, item.OriginalSubtotalCents,
			item.PayableSubtotalCents, flavors, item.Note); err != nil {
			return ConfirmResult{}, false, err
		}
	}
	prepaymentUpdate, err := transaction.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state='PAID',materialization_state='APPLIED',pending_reason_code=NULL,materialized_at=?,
		    lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,
		    next_reconcile_at=NULL,updated_at=?
		WHERE id=? AND record_version=? AND materialization_state IN ('READY','PENDING_MANUAL')
	`, materializedAt, materializedAt, prepayment.id, prepayment.recordVersion)
	if err != nil {
		return ConfirmResult{}, false, err
	}
	rows, err := prepaymentUpdate.RowsAffected()
	if err != nil || rows != 1 {
		return ConfirmResult{}, false, ErrUnavailable
	}
	observationUpdate, err := transaction.ExecContext(ctx, `
		UPDATE payment_observations
		SET apply_state='APPLIED',apply_reason_code=NULL,applied_at=?,record_version=record_version+1
		WHERE id=? AND record_version=? AND apply_state IN ('NEW','DEFERRED')
	`, materializedAt, observation.id, observation.recordVersion)
	if err != nil {
		return ConfirmResult{}, false, err
	}
	rows, err = observationUpdate.RowsAffected()
	if err != nil || rows != 1 {
		return ConfirmResult{}, false, ErrUnavailable
	}
	if err := transaction.Commit(); err != nil {
		return ConfirmResult{}, false, err
	}
	return ConfirmResult{State: ConfirmOrderCreated, OrderID: uint64(orderID)}, true, nil
}

func scanObservation(row rowScanner) (observationRecord, error) {
	var record observationRecord
	var mode string
	err := row.Scan(&record.id, &record.prepaymentID, &record.outTradeNo, &record.transactionID,
		&record.providerState, &record.validation, &record.amountCents, &record.currency,
		&record.successTime, &mode, &record.applyState, &record.applyReason, &record.recordVersion)
	record.materializationMode = MaterializationMode(mode)
	return record, err
}

func validateMaterializationFacts(prepayment prepaymentRecord, observation observationRecord, snapshot quote.Quote, pickupAt time.Time, manual bool) error {
	if snapshot.ID != prepayment.quoteID || snapshot.UserID != prepayment.userID || snapshot.PayableCents != prepayment.expectedAmountCents ||
		observation.prepaymentID != prepayment.id || observation.outTradeNo != prepayment.outTradeNo ||
		observation.amountCents != prepayment.expectedAmountCents || observation.currency != prepayment.currency ||
		!observation.successTime.Before(pickupAt) {
		return ErrUnavailable
	}
	if !manual && !observation.successTime.Before(prepayment.effectiveDeadline) {
		return ErrUnavailable
	}
	request, canonicalRequest, err := canonicalProviderCreateRequest(prepayment.createRequestJSON)
	if err != nil {
		return ErrUnavailable
	}
	digest := sha256.Sum256(canonicalRequest)
	if digest != prepayment.createRequestDigest || request.AppID != prepayment.expectedAppID ||
		request.MerchantID != prepayment.expectedMerchantID ||
		request.OutTradeNo != prepayment.outTradeNo || request.AmountCents != prepayment.expectedAmountCents || request.Currency != prepayment.currency ||
		request.QuoteID != decimalID(snapshot.ID) || request.QuoteDigest != hex.EncodeToString(snapshot.SnapshotDigest[:]) {
		return ErrUnavailable
	}
	return nil
}

func (service *Service) deferUnsafeObservation(ctx context.Context, transaction *sql.Tx, prepayment prepaymentRecord, observation observationRecord, reason string) error {
	if observation.applyState != "DEFERRED" || !observation.applyReason.Valid || observation.applyReason.String != reason {
		if _, err := transaction.ExecContext(ctx, `
			UPDATE payment_observations
			SET apply_state='DEFERRED',apply_reason_code=?,applied_at=NULL,record_version=record_version+1
			WHERE id=? AND record_version=? AND apply_state IN ('NEW','DEFERRED')
		`, reason, observation.id, observation.recordVersion); err != nil {
			return err
		}
	}
	if prepayment.materialState != MaterializationApplied {
		if _, err := transaction.ExecContext(ctx, `
			UPDATE prepayments
			SET materialization_state='PENDING_MANUAL',pending_reason_code=?,materialized_at=NULL,
			    lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,
			    next_reconcile_at=NULL,updated_at=?
			WHERE id=? AND record_version=?
		`, reason, service.now().UTC().Truncate(time.Microsecond), prepayment.id, prepayment.recordVersion); err != nil {
			return err
		}
	}
	return transaction.Commit()
}

func withoutDashes(value string) string {
	if len(value) != 10 {
		return value
	}
	return value[0:4] + value[5:7] + value[8:10]
}

func (service *Service) listPending(ctx context.Context, filter PendingFilter, page PageQuery) ([]PendingPayment, error) {
	query := `SELECT id,pending_reason_code,created_at FROM prepayments WHERE materialization_state='PENDING_MANUAL' AND id>?`
	args := []any{page.AfterID}
	if filter.Reason != "" {
		query += ` AND pending_reason_code=?`
		args = append(args, filter.Reason)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, page.Limit)
	rows, err := service.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	result := make([]PendingPayment, 0, page.Limit)
	for rows.Next() {
		var payment PendingPayment
		if err := rows.Scan(&payment.PrepaymentID, &payment.Reason, &payment.CreatedAt); err != nil {
			return nil, ErrUnavailable
		}
		result = append(result, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}
	return result, nil
}
