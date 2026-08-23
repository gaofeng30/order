package refund

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
)

type mysqlStore struct{ db *sql.DB }

func newMySQLStore(db *sql.DB) *mysqlStore { return &mysqlStore{db: db} }

type orderLocator struct {
	orderID, prepaymentID, ownerUserID uint64
	pickupAt                           time.Time
}

type lockedPayment struct {
	outTradeNo, merchantID, currency string
	amount                           uint64
}

type lockedOrder struct {
	id, userID                    uint64
	state, transactionID, orderNo string
	pickupAt, refundingAt         time.Time
	amount                        uint64
}

func (store *mysqlStore) requestOrder(ctx context.Context, meta WriteMeta, orderID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	if store == nil || store.db == nil {
		return Refund{}, false, ErrUnavailable
	}
	var locator orderLocator
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,user_id,pickup_at FROM orders WHERE id=?`, orderID).Scan(&locator.orderID, &locator.prepaymentID, &locator.ownerUserID, &locator.pickupAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Refund{}, false, ErrNotFound
	}
	if err != nil {
		return Refund{}, false, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, created, err := store.requestOrderOnce(ctx, meta, locator, reason, now, notifyURL)
		if err == nil {
			return result, created, nil
		}
		if !retryableMySQL(err) || attempt == 1 {
			return Refund{}, false, mapStoreError(err)
		}
	}
	return Refund{}, false, ErrUnavailable
}

func (store *mysqlStore) requestOrderOnce(ctx context.Context, meta WriteMeta, locator orderLocator, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Refund{}, false, err
	}
	defer tx.Rollback()

	merchant := meta.ActorKind == ActorMerchant
	var accountID, authVersion uint64
	var role string
	if merchant {
		err = tx.QueryRowContext(ctx, `SELECT id,role,auth_version FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE`, meta.ActorUserID).Scan(&accountID, &role, &authVersion)
		if errors.Is(err, sql.ErrNoRows) {
			return Refund{}, false, ErrForbidden
		}
		if err != nil {
			return Refund{}, false, err
		}
	}
	payment, err := lockPayment(ctx, tx, locator.prepaymentID)
	if err != nil {
		return Refund{}, false, err
	}
	order, err := lockOrder(ctx, tx, locator.orderID)
	if err != nil {
		return Refund{}, false, err
	}
	if order.userID != locator.ownerUserID || order.amount != payment.amount {
		return Refund{}, false, ErrUnavailable
	}

	keyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	if existing, found, err := readRefundByPrepayment(ctx, tx, locator.prepaymentID); err != nil {
		return Refund{}, false, err
	} else if found {
		if !bytes.Equal(existing.keyHash, keyHash[:]) || existing.reason != reason {
			return Refund{}, false, ErrIdempotencyConflict
		}
		if (merchant && existing.accountID != accountID) || (!merchant && existing.userID != meta.ActorUserID) {
			return Refund{}, false, ErrIdempotencyConflict
		}
		replayed, err := readRequestReceipt(ctx, tx, meta, merchant, accountID)
		if err != nil || replayed.ID != existing.viewValue.ID || replayed.PrepaymentID != existing.viewValue.PrepaymentID {
			return Refund{}, false, ErrUnavailable
		}
		return replayed, false, tx.Commit()
	}
	if merchant {
		if order.state != "RESERVED" && order.state != "PREPARING" && order.state != "READY_FOR_PICKUP" && order.state != "COMPLETED" {
			return Refund{}, false, ErrTransitionNotAllowed
		}
	} else if order.state != "RESERVED" || !order.pickupAt.After(now.Add(30*time.Minute)) {
		return Refund{}, false, ErrTransitionNotAllowed
	}
	outRefundNo := "ORDER_REFUND_" + strconv.FormatUint(locator.prepaymentID, 10)
	request := ProviderCreateRequest{OutTradeNo: payment.outTradeNo, TransactionID: order.transactionID, OutRefundNo: outRefundNo, Reason: reason, NotifyURL: notifyURL, AmountCents: order.amount, TotalCents: payment.amount, Currency: payment.currency}
	requestJSON, digest, err := encodeProviderRequest(request)
	if err != nil {
		return Refund{}, false, ErrInvalidInput
	}
	var requestedUser, requestedAccount any
	if merchant {
		requestedAccount = accountID
	} else {
		requestedUser = meta.ActorUserID
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO refunds(prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,requested_by_account_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,provider_refund_id,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_query_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,'READY',NULL,NULL,'AWAITING_PROVIDER',NULL,NULL,NULL,NULL,NULL,1,NULL,?,?)`, locator.prepaymentID, locator.orderID, outRefundNo, keyHash[:], order.amount, payment.currency, reason, requestedUser, requestedAccount, requestJSON, digest[:], now, now)
	if err != nil {
		return Refund{}, false, err
	}
	refundID, err := result.LastInsertId()
	if err != nil || refundID <= 0 {
		return Refund{}, false, ErrUnavailable
	}
	updated, err := tx.ExecContext(ctx, `UPDATE orders SET state='REFUNDING',refunding_at=?,redemption_token_ciphertext=NULL,redemption_key_version=NULL,record_version=record_version+1,updated_at=? WHERE id=? AND record_version>0`, now, now, order.id)
	if err != nil {
		return Refund{}, false, err
	}
	if rows, _ := updated.RowsAffected(); rows != 1 {
		return Refund{}, false, ErrUnavailable
	}
	view := Refund{ID: uint64(refundID), PrepaymentID: locator.prepaymentID, OrderID: locator.orderID, State: ProviderReady, MaterializationState: MaterializationAwaitingProvider, AmountCents: order.amount, Currency: payment.currency, RequestedAt: now}
	if err := insertRequestReceipt(ctx, tx, meta, merchant, accountID, role, authVersion, reason, order.state, view); err != nil {
		return Refund{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Refund{}, false, err
	}
	return view, true, nil
}

func (store *mysqlStore) requestPaidPrepayment(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	if store == nil || store.db == nil {
		return Refund{}, false, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		view, created, err := store.requestPaidPrepaymentOnce(ctx, meta, prepaymentID, reason, now, notifyURL)
		if err == nil {
			return view, created, nil
		}
		if !retryableMySQL(err) || attempt == 1 {
			return Refund{}, false, mapStoreError(err)
		}
	}
	return Refund{}, false, ErrUnavailable
}

func (store *mysqlStore) requestPaidPrepaymentOnce(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Refund{}, false, err
	}
	defer tx.Rollback()
	var accountID, authVersion uint64
	var role string
	err = tx.QueryRowContext(ctx, `SELECT id,role,auth_version FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE`, meta.ActorUserID).Scan(&accountID, &role, &authVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return Refund{}, false, ErrForbidden
	}
	if err != nil {
		return Refund{}, false, err
	}
	payment, err := lockPayment(ctx, tx, prepaymentID)
	if err != nil {
		return Refund{}, false, err
	}
	var providerState, materializationState string
	err = tx.QueryRowContext(ctx, `SELECT provider_state,materialization_state FROM prepayments WHERE id=?`, prepaymentID).Scan(&providerState, &materializationState)
	if err != nil {
		return Refund{}, false, err
	}
	if providerState != "PAID" || materializationState != "PENDING_MANUAL" {
		return Refund{}, false, ErrTransitionNotAllowed
	}
	keyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	if existing, found, err := readRefundByPrepayment(ctx, tx, prepaymentID); err != nil {
		return Refund{}, false, err
	} else if found {
		if !bytes.Equal(existing.keyHash, keyHash[:]) || existing.accountID != accountID || existing.reason != reason {
			return Refund{}, false, ErrIdempotencyConflict
		}
		replayed, err := readRequestReceipt(ctx, tx, meta, true, accountID)
		if err != nil || replayed.ID != existing.viewValue.ID || replayed.PrepaymentID != existing.viewValue.PrepaymentID {
			return Refund{}, false, ErrUnavailable
		}
		return replayed, false, tx.Commit()
	}
	outRefundNo := "ORDER_REFUND_" + strconv.FormatUint(prepaymentID, 10)
	request := ProviderCreateRequest{OutTradeNo: payment.outTradeNo, OutRefundNo: outRefundNo, Reason: reason, NotifyURL: notifyURL, AmountCents: payment.amount, TotalCents: payment.amount, Currency: payment.currency}
	requestJSON, digest, err := encodeProviderRequest(request)
	if err != nil {
		return Refund{}, false, ErrInvalidInput
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO refunds(prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,requested_by_account_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,provider_refund_id,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_query_at,created_at,updated_at) VALUES(?,NULL,?,?,?,?,?,NULL,?,?,?,'READY',NULL,NULL,'AWAITING_PROVIDER',NULL,NULL,NULL,NULL,NULL,1,NULL,?,?)`, prepaymentID, outRefundNo, keyHash[:], payment.amount, payment.currency, reason, accountID, requestJSON, digest[:], now, now)
	if err != nil {
		return Refund{}, false, err
	}
	refundID, err := result.LastInsertId()
	if err != nil || refundID <= 0 {
		return Refund{}, false, ErrUnavailable
	}
	view := Refund{ID: uint64(refundID), PrepaymentID: prepaymentID, State: ProviderReady, MaterializationState: MaterializationAwaitingProvider, AmountCents: payment.amount, Currency: payment.currency, RequestedAt: now}
	if err := insertRequestReceipt(ctx, tx, meta, true, accountID, role, authVersion, reason, "PENDING_MANUAL", view); err != nil {
		return Refund{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Refund{}, false, err
	}
	return view, true, nil
}

func lockPayment(ctx context.Context, tx *sql.Tx, id uint64) (lockedPayment, error) {
	var value lockedPayment
	err := tx.QueryRowContext(ctx, `SELECT CONVERT(out_trade_no USING utf8mb4),CONVERT(expected_mchid USING utf8mb4),expected_amount_cents,currency FROM prepayments WHERE id=? FOR UPDATE`, id).Scan(&value.outTradeNo, &value.merchantID, &value.amount, &value.currency)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}

func lockOrder(ctx context.Context, tx *sql.Tx, id uint64) (lockedOrder, error) {
	var value lockedOrder
	var refunding sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT id,user_id,state,CONVERT(transaction_id USING utf8mb4),CONVERT(order_no USING utf8mb4),pickup_at,payable_cents,refunding_at FROM orders WHERE id=? FOR UPDATE`, id).Scan(&value.id, &value.userID, &value.state, &value.transactionID, &value.orderNo, &value.pickupAt, &value.amount, &refunding)
	if refunding.Valid {
		value.refundingAt = refunding.Time
	}
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}

type refundRow struct {
	viewValue         Refund
	keyHash           []byte
	userID, accountID uint64
	requestJSON       []byte
	reason            string
}

func (row refundRow) view() Refund { return row.viewValue }
func readRefundByPrepayment(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id uint64) (refundRow, bool, error) {
	var row refundRow
	var orderID sql.NullInt64
	var userID, accountID sql.NullInt64
	var providerID, pending sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code,idempotency_key_hash,requested_by_user_id,requested_by_account_id,provider_request_json,reason_code FROM refunds WHERE prepayment_id=?`, id).Scan(&row.viewValue.ID, &row.viewValue.PrepaymentID, &orderID, &row.viewValue.State, &row.viewValue.MaterializationState, &row.viewValue.AmountCents, &row.viewValue.Currency, &row.viewValue.RequestedAt, &providerID, &pending, &row.keyHash, &userID, &accountID, &row.requestJSON, &row.reason)
	if errors.Is(err, sql.ErrNoRows) {
		return refundRow{}, false, nil
	}
	if err != nil {
		return refundRow{}, false, err
	}
	if orderID.Valid {
		row.viewValue.OrderID = uint64(orderID.Int64)
	}
	if providerID.Valid {
		row.viewValue.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		row.viewValue.PendingReason = pending.String
	}
	if userID.Valid {
		row.userID = uint64(userID.Int64)
	}
	if accountID.Valid {
		row.accountID = uint64(accountID.Int64)
	}
	return row, true, nil
}

func encodeProviderRequest(request ProviderCreateRequest) ([]byte, [32]byte, error) {
	raw, err := json.Marshal(request)
	return raw, sha256.Sum256(raw), err
}

func (store *mysqlStore) claimCreate(ctx context.Context, refundID uint64, owner [16]byte, now time.Time, duration time.Duration) (createClaim, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	defer tx.Rollback()
	var state string
	var version uint64
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT provider_state,record_version,provider_request_json FROM refunds WHERE id=? FOR UPDATE`, refundID).Scan(&state, &version, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return createClaim{}, false, ErrNotFound
	}
	if err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	if state != string(ProviderReady) {
		return createClaim{}, false, nil
	}
	var request ProviderCreateRequest
	if json.Unmarshal(raw, &request) != nil || !validProviderCreateRequest(request) {
		return createClaim{}, false, ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_CLAIMED',create_attempted_at=?,lease_kind='CREATE',lease_owner=?,lease_expires_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND record_version=? AND provider_state='READY'`, now, owner[:], now.Add(duration), now, refundID, version)
	if err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return createClaim{}, false, nil
	}
	if err = tx.Commit(); err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	return createClaim{refundID: refundID, request: request, owner: owner, version: version + 1}, true, nil
}

func (store *mysqlStore) finishCreate(ctx context.Context, claim createClaim, observed ProviderRefund, providerErr error, now time.Time) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var state, leaseKind string
	var leaseOwner []byte
	var version uint64
	err = tx.QueryRowContext(ctx, `SELECT provider_state,lease_kind,lease_owner,record_version FROM refunds WHERE id=? FOR UPDATE`, claim.refundID).Scan(&state, &leaseKind, &leaseOwner, &version)
	if err != nil {
		return ErrUnavailable
	}
	if state != string(ProviderCreateClaimed) || leaseKind != "CREATE" || !bytes.Equal(leaseOwner, claim.owner[:]) || version != claim.version {
		return nil
	}
	if providerErr != nil {
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_UNKNOWN',materialization_state='PENDING_MANUAL',pending_reason_code='CREATE_RESULT_UNKNOWN',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, now, now, claim.refundID)
	} else if mismatch := createResultMismatch(claim.request, observed); mismatch != "" {
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_UNKNOWN',materialization_state='PENDING_MANUAL',pending_reason_code=?,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, mismatch, now, now, claim.refundID)
	} else {
		state := providerStateAfterCreate(observed.State)
		if state != ProviderProcessing && state != ProviderSuccess && state != ProviderClosed {
			state = ProviderCreateUnknown
		}
		if state == ProviderClosed {
			_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CLOSED',provider_refund_id=?,materialization_state='PENDING_MANUAL',pending_reason_code='PROVIDER_CLOSED',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=NULL,record_version=record_version+1,updated_at=? WHERE id=?`, observed.RefundID, now, claim.refundID)
		} else {
			_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=?,provider_refund_id=?,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, state, observed.RefundID, now, now, claim.refundID)
		}
	}
	if err != nil {
		return ErrUnavailable
	}
	if err := refreshRequestReceipt(ctx, tx, claim.refundID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return ErrUnavailable
	}
	return nil
}

func createResultMismatch(request ProviderCreateRequest, observed ProviderRefund) string {
	return ValidateProviderRefund(ExpectedRefund{MerchantID: observed.MerchantID, OutTradeNo: request.OutTradeNo, TransactionID: request.TransactionID, OutRefundNo: request.OutRefundNo, AmountCents: request.AmountCents, Currency: request.Currency}, observed)
}

func (store *mysqlStore) get(ctx context.Context, id uint64) (Refund, error) {
	var v Refund
	var orderID sql.NullInt64
	var providerID, pending sql.NullString
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code FROM refunds WHERE id=?`, id).Scan(&v.ID, &v.PrepaymentID, &orderID, &v.State, &v.MaterializationState, &v.AmountCents, &v.Currency, &v.RequestedAt, &providerID, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return Refund{}, ErrNotFound
	}
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	if orderID.Valid {
		v.OrderID = uint64(orderID.Int64)
	}
	if providerID.Valid {
		v.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		v.PendingReason = pending.String
	}
	return v, nil
}

func (store *mysqlStore) ingest(ctx context.Context, verified VerifiedRefund, now time.Time, notifier NotificationEnqueuer) (bool, error) {
	var refundID, prepaymentID uint64
	var orderID sql.NullInt64
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id FROM refunds WHERE out_refund_no=?`, verified.Refund.OutRefundNo).Scan(&refundID, &prepaymentID, &orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		applied, err := store.ingestOnce(ctx, verified, refundID, prepaymentID, orderID, now, notifier)
		if err == nil {
			return applied, nil
		}
		if !retryableMySQL(err) || attempt == 1 {
			return false, mapStoreError(err)
		}
	}
	return false, ErrUnavailable
}

func (store *mysqlStore) ingestOnce(ctx context.Context, verified VerifiedRefund, refundID, prepaymentID uint64, orderID sql.NullInt64, now time.Time, notifier NotificationEnqueuer) (bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	payment, err := lockPayment(ctx, tx, prepaymentID)
	if err != nil {
		return false, err
	}
	var order lockedOrder
	if orderID.Valid {
		order, err = lockOrder(ctx, tx, uint64(orderID.Int64))
		if err != nil {
			return false, err
		}
	}
	var row refundRow
	var storedProviderID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code,idempotency_key_hash,requested_by_user_id,requested_by_account_id,provider_request_json,reason_code FROM refunds WHERE id=? FOR UPDATE`, refundID).Scan(&row.viewValue.ID, &row.viewValue.PrepaymentID, &orderID, &row.viewValue.State, &row.viewValue.MaterializationState, &row.viewValue.AmountCents, &row.viewValue.Currency, &row.viewValue.RequestedAt, &storedProviderID, new(sql.NullString), &row.keyHash, new(sql.NullInt64), new(sql.NullInt64), &row.requestJSON, &row.reason)
	if err != nil {
		return false, err
	}
	transactionID := ""
	refundingAt := time.Time{}
	orderNo := ""
	userID := uint64(0)
	if orderID.Valid {
		transactionID = order.transactionID
		refundingAt = order.refundingAt
		orderNo = order.orderNo
		userID = order.userID
	}
	expected := ExpectedRefund{MerchantID: payment.merchantID, OutTradeNo: payment.outTradeNo, TransactionID: transactionID, OutRefundNo: verified.Refund.OutRefundNo, AmountCents: row.viewValue.AmountCents, Currency: row.viewValue.Currency}
	mismatch := ValidateProviderRefund(expected, verified.Refund)
	if mismatch == "" && storedProviderID.Valid && storedProviderID.String != verified.Refund.RefundID {
		mismatch = "REFUND_ID_MISMATCH"
	}
	if mismatch == "" && orderID.Valid && !validSuccessTime(refundingAt, verified.Refund) {
		mismatch = "SUCCESS_TIME_INVALID"
	}
	dedupe := refundObservationDedupe(verified)
	applyState := "DEFERRED"
	reason := mismatch
	if reason == "" {
		reason = "PROVIDER_" + string(verified.Refund.State)
	}
	var eventID any
	if verified.ProviderEventID != "" {
		eventID = verified.ProviderEventID
	}
	var successTime any
	if !verified.Refund.SuccessTime.IsZero() {
		successTime = verified.Refund.SuccessTime.UTC()
	}
	validation := "MATCH"
	var mismatchValue any
	if mismatch != "" {
		validation = "MISMATCH"
		mismatchValue = mismatch
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO refund_observations(refund_id,dedupe_key,source,provider_event_id,out_refund_no,provider_refund_id,provider_state,validation,mismatch_code,success_time,received_at,apply_state,apply_reason_code,applied_at,record_version) VALUES(?,?,?,?,?,?,?,?,?,?,?,'NEW',NULL,NULL,1)`, refundID, dedupe[:], verified.Source, eventID, verified.Refund.OutRefundNo, verified.Refund.RefundID, verified.Refund.State, validation, mismatchValue, successTime, now)
	if err != nil {
		var mysqlError *mysql.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return row.viewValue.MaterializationState == MaterializationApplied, tx.Commit()
		}
		return false, err
	}
	inserted, _ := result.RowsAffected()
	if inserted == 0 {
		return row.viewValue.MaterializationState == MaterializationApplied, tx.Commit()
	}
	var observationID uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM refund_observations WHERE dedupe_key=?`, dedupe[:]).Scan(&observationID)
	if err != nil {
		return false, err
	}
	newProvider := AdvanceProviderState(row.viewValue.State, verified.Refund.State)
	applied := mismatch == "" && verified.Refund.State == ProviderSuccess && (!orderID.Valid || order.state == "REFUNDING" || order.state == "REFUNDED")
	if applied {
		if orderID.Valid && order.state != "REFUNDED" {
			_, err = tx.ExecContext(ctx, `UPDATE orders SET state='REFUNDED',refunded_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND state='REFUNDING'`, verified.Refund.SuccessTime.UTC(), now, order.id)
			if err != nil {
				return false, err
			}
		}
		if !orderID.Valid {
			_, err = tx.ExecContext(ctx, `UPDATE prepayments SET materialization_state='REFUNDED_WITHOUT_ORDER',pending_reason_code=NULL,materialized_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, now, now, prepaymentID)
			if err != nil {
				return false, err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=?,provider_refund_id=?,materialization_state='APPLIED',pending_reason_code=NULL,materialized_at=?,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=NULL,record_version=record_version+1,updated_at=? WHERE id=?`, newProvider, verified.Refund.RefundID, now, now, refundID)
		if err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE refund_observations SET apply_state='APPLIED',apply_reason_code=NULL,applied_at=?,record_version=record_version+1 WHERE id=?`, now, observationID)
		if err != nil {
			return false, err
		}
		if notifier != nil && orderID.Valid {
			if notifyErr := notifier.EnqueueRefundResultInTx(ctx, tx, order.id, userID, orderNo, now); notifyErr != nil && !errors.Is(notifyErr, ErrNoConsent) {
				return false, notifyErr
			}
		}
		applyState = "APPLIED"
		reason = "REFUND_CONFIRMED"
	} else {
		pendingReason := reason
		if verified.Refund.State == ProviderProcessing && mismatch == "" {
			pendingReason = "PROVIDER_PROCESSING"
		}
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=?,provider_refund_id=?,materialization_state=IF(materialization_state='APPLIED','APPLIED',IF(?='PROVIDER_PROCESSING',materialization_state,'PENDING_MANUAL')),pending_reason_code=IF(materialization_state='APPLIED',NULL,IF(?='PROVIDER_PROCESSING',pending_reason_code,?)),lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=IF(materialization_state='APPLIED',NULL,?),record_version=record_version+1,updated_at=? WHERE id=?`, newProvider, verified.Refund.RefundID, pendingReason, pendingReason, pendingReason, now.Add(refundQueryDelay), now, refundID)
		if err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE refund_observations SET apply_state='DEFERRED',apply_reason_code=?,record_version=record_version+1 WHERE id=?`, pendingReason, observationID)
		if err != nil {
			return false, err
		}
	}
	if err := insertProviderEvidence(ctx, tx, refundID, orderID, applyState, reason, verified, now); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return applied, nil
}

func refundObservationDedupe(v VerifiedRefund) [32]byte {
	raw, _ := json.Marshal(struct {
		Source Source         `json:"source"`
		Event  string         `json:"event,omitempty"`
		Refund ProviderRefund `json:"refund"`
	}{v.Source, v.ProviderEventID, v.Refund})
	return sha256.Sum256(raw)
}

func (store *mysqlStore) claimDue(ctx context.Context, now time.Time, limit uint16, owner [16]byte, duration time.Duration) ([]queryClaim, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id FROM refunds WHERE ((provider_state='CREATE_CLAIMED' AND lease_expires_at<=?) OR (provider_state IN ('CREATE_UNKNOWN','PROCESSING') AND (lease_expires_at IS NULL OR lease_expires_at<=?) AND (next_query_at IS NULL OR next_query_at<=?))) ORDER BY id LIMIT ?`, now, now, now, limit)
	if err != nil {
		return nil, ErrUnavailable
	}
	var ids []uint64
	for rows.Next() {
		var id uint64
		if rows.Scan(&id) != nil {
			rows.Close()
			return nil, ErrUnavailable
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		rows.Close()
		return nil, ErrUnavailable
	}
	rows.Close()
	claims := make([]queryClaim, 0, len(ids))
	for _, id := range ids {
		claim, ok, err := store.claimOneQuery(ctx, id, owner, now, duration)
		if err != nil {
			return nil, err
		}
		if ok {
			claims = append(claims, claim)
		}
	}
	return claims, nil
}

func (store *mysqlStore) claimOneQuery(ctx context.Context, id uint64, owner [16]byte, now time.Time, duration time.Duration) (queryClaim, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	defer tx.Rollback()
	var state, out string
	var version uint64
	var leaseExpiry sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT provider_state,CONVERT(out_refund_no USING utf8mb4),record_version,lease_expires_at FROM refunds WHERE id=? FOR UPDATE`, id).Scan(&state, &out, &version, &leaseExpiry)
	if err != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	eligible := (state == "CREATE_CLAIMED" && leaseExpiry.Valid && !leaseExpiry.Time.After(now)) || ((state == "CREATE_UNKNOWN" || state == "PROCESSING") && (!leaseExpiry.Valid || !leaseExpiry.Time.After(now)))
	if !eligible {
		return queryClaim{}, false, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE refunds SET provider_state=IF(provider_state='CREATE_CLAIMED','CREATE_UNKNOWN',provider_state),lease_kind='QUERY',lease_owner=?,lease_expires_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND record_version=?`, owner[:], now.Add(duration), now, id, version)
	if err != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return queryClaim{}, false, nil
	}
	if tx.Commit() != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	return queryClaim{refundID: id, outRefundNo: out, owner: owner, version: version + 1}, true, nil
}

func (store *mysqlStore) finishQuery(ctx context.Context, claim queryClaim, observed ProviderRefund, providerErr error, now time.Time) (bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return false, ErrUnavailable
	}
	defer tx.Rollback()
	var leaseKind string
	var leaseOwner []byte
	var version uint64
	err = tx.QueryRowContext(ctx, `SELECT lease_kind,lease_owner,record_version FROM refunds WHERE id=? FOR UPDATE`, claim.refundID).Scan(&leaseKind, &leaseOwner, &version)
	if err != nil {
		return false, ErrUnavailable
	}
	if leaseKind != "QUERY" || !bytes.Equal(leaseOwner, claim.owner[:]) || version != claim.version {
		return false, nil
	}
	if providerErr != nil {
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, now.Add(refundQueryDelay), now, claim.refundID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=?,record_version=record_version+1,updated_at=? WHERE id=?`, now.Add(refundQueryDelay), now, claim.refundID)
	}
	if err != nil || tx.Commit() != nil {
		return false, ErrUnavailable
	}
	return providerErr == nil, nil
}

func (store *mysqlStore) listPending(ctx context.Context, userID uint64, filter PendingFilter) ([]Refund, error) {
	var ownerID uint64
	err := store.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	rows, err := store.db.QueryContext(ctx, `SELECT id,prepayment_id,COALESCE(order_id,0),provider_state,materialization_state,amount_cents,currency,created_at,COALESCE(CONVERT(provider_refund_id USING utf8mb4),''),COALESCE(pending_reason_code,'') FROM refunds WHERE id>? AND (materialization_state='PENDING_MANUAL' OR provider_state IN ('CREATE_UNKNOWN','PROCESSING','CLOSED')) ORDER BY id LIMIT ?`, filter.AfterID, filter.Limit)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	var out []Refund
	for rows.Next() {
		var v Refund
		if rows.Scan(&v.ID, &v.PrepaymentID, &v.OrderID, &v.State, &v.MaterializationState, &v.AmountCents, &v.Currency, &v.RequestedAt, &v.ProviderRefundID, &v.PendingReason) != nil {
			return nil, ErrUnavailable
		}
		out = append(out, v)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return out, nil
}

func insertRequestReceipt(ctx context.Context, tx *sql.Tx, meta WriteMeta, merchant bool, accountID uint64, role string, authVersion uint64, reason, before string, view Refund) error {
	response, _ := json.Marshal(view)
	evidenceDigest := sha256.Sum256([]byte(strconv.FormatUint(view.OrderID, 10) + "\x00" + reason))
	evidence := hex.EncodeToString(evidenceDigest[:])
	key := sha256.Sum256([]byte(meta.IdempotencyKey))
	requestID := sha256.Sum256([]byte(meta.RequestID))
	actorKind := "USER"
	scope := requestScope(meta, merchant, accountID)
	var account, snapshotID, snapshotRole, snapshotVersion any
	if merchant {
		actorKind = "MERCHANT"
		account = accountID
		snapshotID = accountID
		snapshotRole = role
		snapshotVersion = authVersion
	}
	beforeJSON, _ := json.Marshal(map[string]string{"state": before, "request_digest": evidence})
	afterJSON, _ := json.Marshal(map[string]string{"state": "REFUNDING"})
	_, err := tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT',?,?,?,?,?,?,?,'refund.request','REFUND',?,?,?,'SUCCEEDED','REFUND_REQUESTED',?,?,?,?)`, actorKind, scope[:], meta.ActorUserID, account, snapshotID, snapshotRole, snapshotVersion, view.ID, key[:], requestID[:], beforeJSON, afterJSON, response, view.RequestedAt)
	if err != nil {
		return err
	}
	return nil
}

func requestScope(meta WriteMeta, merchant bool, accountID uint64) [32]byte {
	if merchant {
		return merchantScope(meta.ActorUserID, accountID)
	}
	return userScope(meta.ActorUserID)
}

func readRequestReceipt(ctx context.Context, tx *sql.Tx, meta WriteMeta, merchant bool, accountID uint64) (Refund, error) {
	scope := requestScope(meta, merchant, accountID)
	key := sha256.Sum256([]byte(meta.IdempotencyKey))
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=? AND action='refund.request' AND operation_key_hash=?`, scope[:], key[:]).Scan(&raw)
	if err != nil {
		return Refund{}, err
	}
	var view Refund
	if json.Unmarshal(raw, &view) != nil || view.ID == 0 || view.PrepaymentID == 0 || view.AmountCents == 0 || view.Currency != "CNY" {
		return Refund{}, ErrUnavailable
	}
	return view, nil
}

func refreshRequestReceipt(ctx context.Context, tx *sql.Tx, refundID uint64) error {
	var view Refund
	var orderID sql.NullInt64
	var providerID, pending sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code FROM refunds WHERE id=?`, refundID).Scan(&view.ID, &view.PrepaymentID, &orderID, &view.State, &view.MaterializationState, &view.AmountCents, &view.Currency, &view.RequestedAt, &providerID, &pending)
	if err != nil {
		return ErrUnavailable
	}
	if orderID.Valid {
		view.OrderID = uint64(orderID.Int64)
	}
	if providerID.Valid {
		view.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		view.PendingReason = pending.String
	}
	response, err := json.Marshal(view)
	if err != nil {
		return ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE action_audits SET response_json=? WHERE entry_kind='COMMAND_RECEIPT' AND action='refund.request' AND target_type='REFUND' AND target_id=?`, response, refundID)
	if err != nil {
		return ErrUnavailable
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrUnavailable
	}
	return nil
}

func insertProviderEvidence(ctx context.Context, tx *sql.Tx, refundID uint64, orderID sql.NullInt64, applyState, reason string, verified VerifiedRefund, now time.Time) error {
	scope := sha256.Sum256([]byte("PROVIDER:REFUND"))
	before, _ := json.Marshal(map[string]string{"observation": string(verified.Refund.State)})
	after, _ := json.Marshal(map[string]string{"apply_state": applyState})
	result := "SUCCEEDED"
	if applyState != "APPLIED" {
		result = "REJECTED"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,operation_key_hash,result,reason_code,before_state_json,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',?,NULL,NULL,'refund.observe','REFUND',?,NULL,?,?,?,?,?)`, scope[:], refundID, result, reason, before, after, now)
	_ = orderID
	return err
}

func userScope(id uint64) [32]byte {
	var b [13]byte
	copy(b[:5], "USER\x00")
	binary.BigEndian.PutUint64(b[5:], id)
	return sha256.Sum256(b[:])
}
func merchantScope(user, account uint64) [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("merchant:%d:%d", user, account)))
}
func retryableMySQL(err error) bool {
	var e *mysql.MySQLError
	return errors.As(err, &e) && (e.Number == 1205 || e.Number == 1213)
}
