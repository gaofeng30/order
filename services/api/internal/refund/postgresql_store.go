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
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresStore struct {
	db             *sql.DB
	config         Config
	merchantNumber func(string, string) (string, error)
}

func newPostgreSQLStore(db *sql.DB, config Config) *postgresStore {
	return &postgresStore{db: db, config: config, merchantNumber: wechatpay.NewMerchantNumber}
}

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

func (store *postgresStore) requestOrder(ctx context.Context, meta WriteMeta, orderID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	if store == nil || store.db == nil {
		return Refund{}, false, ErrUnavailable
	}
	var locator orderLocator
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,user_id,pickup_at FROM orders WHERE id=$1`, orderID).Scan(&locator.orderID, &locator.prepaymentID, &locator.ownerUserID, &locator.pickupAt)
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
		if attempt == 0 && (store.config.Environment == "uat" || store.config.Environment == "production") && refundNumberCollision(err) {
			continue
		}
		if !retryablePostgreSQL(err) || attempt == 1 {
			return Refund{}, false, mapStoreError(err)
		}
	}
	return Refund{}, false, ErrUnavailable
}

func (store *postgresStore) requestOrderOnce(ctx context.Context, meta WriteMeta, locator orderLocator, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Refund{}, false, err
	}
	defer tx.Rollback()

	merchant := meta.ActorKind == ActorMerchant
	var accountID, authVersion uint64
	var role string
	if merchant {
		err = tx.QueryRowContext(ctx, `SELECT id,role,auth_version FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE`, meta.ActorUserID).Scan(&accountID, &role, &authVersion)
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
	if !merchant && order.userID != meta.ActorUserID {
		return Refund{}, false, ErrNotFound
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
	if !store.parentInEnvironment(ctx, tx, locator.prepaymentID) {
		return Refund{}, false, ErrUnavailable
	}
	outRefundNo, err := store.newOutRefundNo(locator.prepaymentID)
	if err != nil {
		return Refund{}, false, ErrUnavailable
	}
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
	var refundID uint64
	err = tx.QueryRowContext(ctx, `INSERT INTO refunds(prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,requested_by_account_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,provider_refund_id,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_query_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'READY',NULL,NULL,'AWAITING_PROVIDER',NULL,NULL,NULL,NULL,NULL,1,NULL,$12,$13) RETURNING id`, locator.prepaymentID, locator.orderID, []byte(outRefundNo), keyHash[:], order.amount, payment.currency, reason, requestedUser, requestedAccount, requestJSON, digest[:], now, now).Scan(&refundID)
	if err != nil {
		return Refund{}, false, err
	}
	if refundID == 0 {
		return Refund{}, false, ErrUnavailable
	}
	updated, err := tx.ExecContext(ctx, `UPDATE orders SET state='REFUNDING',refunding_at=$1,redemption_token_ciphertext=NULL,redemption_key_version=NULL,record_version=record_version+1,updated_at=$2 WHERE id=$3 AND record_version>0`, now, now, order.id)
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

func (store *postgresStore) requestPaidPrepayment(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	if store == nil || store.db == nil {
		return Refund{}, false, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		view, created, err := store.requestPaidPrepaymentOnce(ctx, meta, prepaymentID, reason, now, notifyURL)
		if err == nil {
			return view, created, nil
		}
		if attempt == 0 && (store.config.Environment == "uat" || store.config.Environment == "production") && refundNumberCollision(err) {
			continue
		}
		if !retryablePostgreSQL(err) || attempt == 1 {
			return Refund{}, false, mapStoreError(err)
		}
	}
	return Refund{}, false, ErrUnavailable
}

func (store *postgresStore) requestPaidPrepaymentOnce(ctx context.Context, meta WriteMeta, prepaymentID uint64, reason string, now time.Time, notifyURL string) (Refund, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Refund{}, false, err
	}
	defer tx.Rollback()
	var accountID, authVersion uint64
	var role string
	err = tx.QueryRowContext(ctx, `SELECT id,role,auth_version FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE`, meta.ActorUserID).Scan(&accountID, &role, &authVersion)
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
	err = tx.QueryRowContext(ctx, `SELECT provider_state,materialization_state FROM prepayments WHERE id=$1`, prepaymentID).Scan(&providerState, &materializationState)
	if err != nil {
		return Refund{}, false, err
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
	if providerState != "PAID" || materializationState != "PENDING_MANUAL" {
		return Refund{}, false, ErrTransitionNotAllowed
	}
	if !store.parentInEnvironment(ctx, tx, prepaymentID) {
		return Refund{}, false, ErrUnavailable
	}
	outRefundNo, err := store.newOutRefundNo(prepaymentID)
	if err != nil {
		return Refund{}, false, ErrUnavailable
	}
	request := ProviderCreateRequest{OutTradeNo: payment.outTradeNo, OutRefundNo: outRefundNo, Reason: reason, NotifyURL: notifyURL, AmountCents: payment.amount, TotalCents: payment.amount, Currency: payment.currency}
	requestJSON, digest, err := encodeProviderRequest(request)
	if err != nil {
		return Refund{}, false, ErrInvalidInput
	}
	var refundID uint64
	err = tx.QueryRowContext(ctx, `INSERT INTO refunds(prepayment_id,order_id,out_refund_no,idempotency_key_hash,amount_cents,currency,reason_code,requested_by_user_id,requested_by_account_id,provider_request_json,provider_request_digest,provider_state,create_attempted_at,provider_refund_id,materialization_state,pending_reason_code,materialized_at,lease_kind,lease_owner,lease_expires_at,record_version,next_query_at,created_at,updated_at) VALUES($1,NULL,$2,$3,$4,$5,$6,NULL,$7,$8,$9,'READY',NULL,NULL,'AWAITING_PROVIDER',NULL,NULL,NULL,NULL,NULL,1,NULL,$10,$11) RETURNING id`, prepaymentID, []byte(outRefundNo), keyHash[:], payment.amount, payment.currency, reason, accountID, requestJSON, digest[:], now, now).Scan(&refundID)
	if err != nil {
		return Refund{}, false, err
	}
	if refundID == 0 {
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
	err := tx.QueryRowContext(ctx, `SELECT out_trade_no,expected_mchid,expected_amount_cents,currency FROM prepayments WHERE id=$1 FOR UPDATE`, id).Scan(&value.outTradeNo, &value.merchantID, &value.amount, &value.currency)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	return value, err
}

func lockOrder(ctx context.Context, tx *sql.Tx, id uint64) (lockedOrder, error) {
	var value lockedOrder
	var refunding sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT id,user_id,state,transaction_id,order_no,pickup_at,payable_cents,refunding_at FROM orders WHERE id=$1 FOR UPDATE`, id).Scan(&value.id, &value.userID, &value.state, &value.transactionID, &value.orderNo, &value.pickupAt, &value.amount, &refunding)
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
	var orderID sql.Null[uint64]
	var userID, accountID sql.Null[uint64]
	var providerID, pending sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code,idempotency_key_hash,requested_by_user_id,requested_by_account_id,provider_request_json,reason_code FROM refunds WHERE prepayment_id=$1`, id).Scan(&row.viewValue.ID, &row.viewValue.PrepaymentID, &orderID, &row.viewValue.State, &row.viewValue.MaterializationState, &row.viewValue.AmountCents, &row.viewValue.Currency, &row.viewValue.RequestedAt, &providerID, &pending, &row.keyHash, &userID, &accountID, &row.requestJSON, &row.reason)
	if errors.Is(err, sql.ErrNoRows) {
		return refundRow{}, false, nil
	}
	if err != nil {
		return refundRow{}, false, err
	}
	if orderID.Valid {
		row.viewValue.OrderID = uint64(orderID.V)
	}
	if providerID.Valid {
		row.viewValue.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		row.viewValue.PendingReason = pending.String
	}
	if userID.Valid {
		row.userID = uint64(userID.V)
	}
	if accountID.Valid {
		row.accountID = uint64(accountID.V)
	}
	return row, true, nil
}

func encodeProviderRequest(request ProviderCreateRequest) ([]byte, [32]byte, error) {
	raw, err := json.Marshal(request)
	return raw, sha256.Sum256(raw), err
}

func (store *postgresStore) claimCreate(ctx context.Context, refundID uint64, owner [16]byte, now time.Time, duration time.Duration) (createClaim, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	defer tx.Rollback()
	var state string
	var version uint64
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT provider_state,record_version,provider_request_json FROM refunds WHERE id=$1 FOR UPDATE`, refundID).Scan(&state, &version, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return createClaim{}, false, ErrNotFound
	}
	if err != nil {
		return createClaim{}, false, ErrUnavailable
	}
	if state != string(ProviderReady) {
		return createClaim{}, false, nil
	}
	if !store.refundInEnvironment(ctx, tx, refundID) {
		return createClaim{}, false, ErrUnavailable
	}
	var request ProviderCreateRequest
	if json.Unmarshal(raw, &request) != nil || !validProviderCreateRequest(request) {
		return createClaim{}, false, ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_CLAIMED',create_attempted_at=$1,lease_kind='CREATE',lease_owner=$2,lease_expires_at=$3,record_version=record_version+1,updated_at=$4 WHERE id=$5 AND record_version=$6 AND provider_state='READY'`, now, owner[:], now.Add(duration), now, refundID, version)
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

func (store *postgresStore) finishCreate(ctx context.Context, claim createClaim, observed ProviderRefund, providerErr error, now time.Time) (Refund, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	defer tx.Rollback()
	var state string
	var leaseKind sql.NullString
	var leaseExpiry sql.NullTime
	var leaseOwner []byte
	var version uint64
	err = tx.QueryRowContext(ctx, `SELECT provider_state,lease_kind,lease_owner,record_version,lease_expires_at FROM refunds WHERE id=$1 FOR UPDATE`, claim.refundID).Scan(&state, &leaseKind, &leaseOwner, &version, &leaseExpiry)
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	// Only the live owner publishes provider facts. The request response and its
	// receipt are frozen from the same locked durable row even if a callback won.
	if state == string(ProviderCreateClaimed) && leaseKind.Valid && leaseKind.String == "CREATE" && bytes.Equal(leaseOwner, claim.owner[:]) && version == claim.version && leaseExpiry.Valid && now.Before(leaseExpiry.Time) {
		if providerErr != nil {
			_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_UNKNOWN',materialization_state='PENDING_MANUAL',pending_reason_code='CREATE_RESULT_UNKNOWN',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=$1,record_version=record_version+1,updated_at=$2 WHERE id=$3`, now.Add(refundFailureDelay), now, claim.refundID)
		} else if mismatch := createResultMismatch(claim.request, observed); mismatch != "" {
			_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CREATE_UNKNOWN',materialization_state='PENDING_MANUAL',pending_reason_code=$1,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=$2,record_version=record_version+1,updated_at=$3 WHERE id=$4`, mismatch, nil, now, claim.refundID)
		} else {
			state := providerStateAfterCreate(observed.State)
			if state != ProviderProcessing && state != ProviderSuccess && state != ProviderClosed {
				state = ProviderCreateUnknown
			}
			if state == ProviderClosed {
				_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state='CLOSED',provider_refund_id=$1,materialization_state='PENDING_MANUAL',pending_reason_code='PROVIDER_CLOSED',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=NULL,record_version=record_version+1,updated_at=$2 WHERE id=$3`, []byte(observed.RefundID), now, claim.refundID)
			} else {
				var nextQuery any
				if observed.State == ProviderSuccess {
					nextQuery = now
				}
				_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=$1,provider_refund_id=$2,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=$3,record_version=record_version+1,updated_at=$4 WHERE id=$5`, state, []byte(observed.RefundID), nextQuery, now, claim.refundID)
			}
		}
		if err != nil {
			return Refund{}, ErrUnavailable
		}
	}
	view, err := refreshRequestReceipt(ctx, tx, claim.refundID)
	if err != nil {
		return Refund{}, err
	}
	if err = tx.Commit(); err != nil {
		return Refund{}, ErrUnavailable
	}
	return view, nil
}

func createResultMismatch(request ProviderCreateRequest, observed ProviderRefund) string {
	return ValidateProviderRefund(ExpectedRefund{MerchantID: observed.MerchantID, OutTradeNo: request.OutTradeNo, TransactionID: request.TransactionID, OutRefundNo: request.OutRefundNo, AmountCents: request.AmountCents, Currency: request.Currency}, observed)
}

func (store *postgresStore) get(ctx context.Context, id uint64) (Refund, error) {
	var v Refund
	var orderID sql.Null[uint64]
	var providerID, pending sql.NullString
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code FROM refunds WHERE id=$1`, id).Scan(&v.ID, &v.PrepaymentID, &orderID, &v.State, &v.MaterializationState, &v.AmountCents, &v.Currency, &v.RequestedAt, &providerID, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return Refund{}, ErrNotFound
	}
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	if orderID.Valid {
		v.OrderID = uint64(orderID.V)
	}
	if providerID.Valid {
		v.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		v.PendingReason = pending.String
	}
	return v, nil
}

var errStaleQueryLease = errors.New("refund query lease stale")

func (store *postgresStore) ingest(ctx context.Context, verified VerifiedRefund, now time.Time, notifier NotificationEnqueuer) (bool, error) {
	return store.ingestWithClaim(ctx, verified, now, notifier, nil)
}

func (store *postgresStore) ingestWithClaim(ctx context.Context, verified VerifiedRefund, now time.Time, notifier NotificationEnqueuer, claim *queryClaim) (bool, error) {
	var refundID, prepaymentID uint64
	var orderID sql.Null[uint64]
	err := store.db.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id FROM refunds WHERE out_refund_no=$1`, []byte(verified.Refund.OutRefundNo)).Scan(&refundID, &prepaymentID, &orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, ErrUnavailable
	}
	if claim != nil && claim.refundID != refundID {
		return false, errStaleQueryLease
	}
	for attempt := 0; attempt < 2; attempt++ {
		applied, err := store.ingestOnce(ctx, verified, refundID, prepaymentID, orderID, now, notifier, claim)
		if err == nil {
			return applied, nil
		}
		if errors.Is(err, errStaleQueryLease) {
			return false, err
		}
		if attempt == 0 && (store.config.Environment == "uat" || store.config.Environment == "production") && refundNumberCollision(err) {
			continue
		}
		if !retryablePostgreSQL(err) || attempt == 1 {
			return false, mapStoreError(err)
		}
	}
	return false, ErrUnavailable
}

func (store *postgresStore) ingestOnce(ctx context.Context, verified VerifiedRefund, refundID, prepaymentID uint64, orderID sql.Null[uint64], now time.Time, notifier NotificationEnqueuer, claim *queryClaim) (bool, error) {
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
		order, err = lockOrder(ctx, tx, uint64(orderID.V))
		if err != nil {
			return false, err
		}
	}
	var row refundRow
	var storedProviderID, leaseKind sql.NullString
	var leaseOwner []byte
	var leaseVersion uint64
	var leaseExpiry sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code,idempotency_key_hash,requested_by_user_id,requested_by_account_id,provider_request_json,reason_code,lease_kind,lease_owner,record_version,lease_expires_at FROM refunds WHERE id=$1 FOR UPDATE`, refundID).Scan(&row.viewValue.ID, &row.viewValue.PrepaymentID, &orderID, &row.viewValue.State, &row.viewValue.MaterializationState, &row.viewValue.AmountCents, &row.viewValue.Currency, &row.viewValue.RequestedAt, &storedProviderID, new(sql.NullString), &row.keyHash, new(sql.Null[uint64]), new(sql.Null[uint64]), &row.requestJSON, &row.reason, &leaseKind, &leaseOwner, &leaseVersion, &leaseExpiry)
	if err != nil {
		return false, err
	}
	if !store.refundCallbackInEnvironment(ctx, tx, refundID, verified.Refund) {
		return false, ErrUnavailable
	}
	if claim != nil && (!leaseKind.Valid || leaseKind.String != "QUERY" || !bytes.Equal(leaseOwner, claim.owner[:]) || leaseVersion != claim.version || !leaseExpiry.Valid || !now.Before(leaseExpiry.Time)) {
		return false, errStaleQueryLease
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
		eventID = []byte(verified.ProviderEventID)
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
	result, err := tx.ExecContext(ctx, `INSERT INTO refund_observations(refund_id,dedupe_key,source,provider_event_id,out_refund_no,provider_refund_id,provider_state,validation,mismatch_code,success_time,received_at,apply_state,apply_reason_code,applied_at,record_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'NEW',NULL,NULL,1) ON CONFLICT ON CONSTRAINT uq_refund_observations_dedupe DO NOTHING`, refundID, dedupe[:], verified.Source, eventID, []byte(verified.Refund.OutRefundNo), []byte(verified.Refund.RefundID), verified.Refund.State, validation, mismatchValue, successTime, now)
	if err != nil {
		return false, err
	}
	inserted, _ := result.RowsAffected()
	if inserted == 0 {
		if claim != nil {
			_, err := tx.ExecContext(ctx, `UPDATE refunds SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=CASE WHEN materialization_state='APPLIED' THEN NULL ELSE $1::timestamptz END,record_version=record_version+1,updated_at=$2 WHERE id=$3`, nil, now, refundID)
			if err != nil {
				return false, err
			}
		}
		return row.viewValue.MaterializationState == MaterializationApplied, tx.Commit()
	}
	var observationID uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM refund_observations WHERE dedupe_key=$1`, dedupe[:]).Scan(&observationID)
	if err != nil {
		return false, err
	}
	newProvider := AdvanceProviderState(row.viewValue.State, verified.Refund.State)
	applied := mismatch == "" && verified.Refund.State == ProviderSuccess && (!orderID.Valid || order.state == "REFUNDING" || order.state == "REFUNDED")
	if applied {
		if orderID.Valid && order.state != "REFUNDED" {
			_, err = tx.ExecContext(ctx, `UPDATE orders SET state='REFUNDED',refunded_at=$1,record_version=record_version+1,updated_at=$2 WHERE id=$3 AND state='REFUNDING'`, verified.Refund.SuccessTime.UTC(), now, order.id)
			if err != nil {
				return false, err
			}
		}
		if !orderID.Valid {
			_, err = tx.ExecContext(ctx, `UPDATE prepayments SET materialization_state='REFUNDED_WITHOUT_ORDER',pending_reason_code=NULL,materialized_at=$1,record_version=record_version+1,updated_at=$2 WHERE id=$3`, now, now, prepaymentID)
			if err != nil {
				return false, err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=$1,provider_refund_id=$2,materialization_state='APPLIED',pending_reason_code=NULL,materialized_at=$3,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=NULL,record_version=record_version+1,updated_at=$4 WHERE id=$5`, newProvider, []byte(verified.Refund.RefundID), now, now, refundID)
		if err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE refund_observations SET apply_state='APPLIED',apply_reason_code=NULL,applied_at=$1,record_version=record_version+1 WHERE id=$2`, now, observationID)
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
		_, err = tx.ExecContext(ctx, `UPDATE refunds SET provider_state=$1,provider_refund_id=$2,materialization_state=CASE WHEN materialization_state='APPLIED' THEN 'APPLIED' WHEN $3='PROVIDER_PROCESSING' THEN materialization_state ELSE 'PENDING_MANUAL' END,pending_reason_code=CASE WHEN materialization_state='APPLIED' THEN NULL WHEN $4='PROVIDER_PROCESSING' THEN pending_reason_code ELSE $5 END,lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=CASE WHEN materialization_state='APPLIED' THEN NULL ELSE $6::timestamptz END,record_version=record_version+1,updated_at=$7 WHERE id=$8`, newProvider, []byte(verified.Refund.RefundID), pendingReason, pendingReason, pendingReason, nil, now, refundID)
		if err != nil {
			return false, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE refund_observations SET apply_state='DEFERRED',apply_reason_code=$1,record_version=record_version+1 WHERE id=$2`, pendingReason, observationID)
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

func (store *postgresStore) listDueIDs(ctx context.Context, dueBefore, leaseNow time.Time, limit uint16) ([]uint64, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id FROM refunds WHERE provider_state='READY' OR ((provider_state='CREATE_CLAIMED' AND lease_expires_at<=$1) OR (provider_state IN ('CREATE_UNKNOWN','PROCESSING') AND (lease_expires_at IS NULL OR lease_expires_at<=$2) AND (lease_expires_at IS NOT NULL OR (next_query_at IS NOT NULL AND next_query_at<=$3)))) ORDER BY id LIMIT $4`, leaseNow, leaseNow, dueBefore, limit)
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
	return ids, nil
}

func (store *postgresStore) claimOneQuery(ctx context.Context, id uint64, owner [16]byte, dueBefore, now time.Time, duration time.Duration) (queryClaim, bool, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	defer tx.Rollback()
	var state, out string
	var version uint64
	var leaseExpiry, nextQuery sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT provider_state,out_refund_no,record_version,lease_expires_at,next_query_at FROM refunds WHERE id=$1 FOR UPDATE`, id).Scan(&state, &out, &version, &leaseExpiry, &nextQuery)
	if err != nil {
		return queryClaim{}, false, ErrUnavailable
	}
	eligible := (state == "CREATE_CLAIMED" && leaseExpiry.Valid && !leaseExpiry.Time.After(now)) || ((state == "CREATE_UNKNOWN" || state == "PROCESSING") && (!leaseExpiry.Valid || !leaseExpiry.Time.After(now)) && (leaseExpiry.Valid || (nextQuery.Valid && !nextQuery.Time.After(dueBefore))))
	if !eligible {
		return queryClaim{}, false, nil
	}
	if !store.refundInEnvironment(ctx, tx, id) {
		return queryClaim{}, false, ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE refunds SET provider_state=CASE WHEN provider_state='CREATE_CLAIMED' THEN 'CREATE_UNKNOWN' ELSE provider_state END,lease_kind='QUERY',lease_owner=$1,lease_expires_at=$2,record_version=record_version+1,updated_at=$3 WHERE id=$4 AND record_version=$5`, owner[:], now.Add(duration), now, id, version)
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

func (store *postgresStore) finishQuery(ctx context.Context, claim queryClaim, observed ProviderRefund, providerErr error, now time.Time, notifier NotificationEnqueuer) (bool, bool, error) {
	if providerErr == nil {
		applied, err := store.ingestWithClaim(ctx, VerifiedRefund{Source: SourceActiveQuery, Refund: observed}, now, notifier, &claim)
		if errors.Is(err, errStaleQueryLease) {
			return false, false, nil
		}
		return err == nil, applied, err
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return false, false, ErrUnavailable
	}
	defer tx.Rollback()
	var leaseKind sql.NullString
	var leaseExpiry sql.NullTime
	var leaseOwner []byte
	var version uint64
	err = tx.QueryRowContext(ctx, `SELECT lease_kind,lease_owner,record_version,lease_expires_at FROM refunds WHERE id=$1 FOR UPDATE`, claim.refundID).Scan(&leaseKind, &leaseOwner, &version, &leaseExpiry)
	if err != nil {
		return false, false, ErrUnavailable
	}
	if !leaseKind.Valid || leaseKind.String != "QUERY" || !bytes.Equal(leaseOwner, claim.owner[:]) || version != claim.version || !leaseExpiry.Valid || !now.Before(leaseExpiry.Time) {
		return false, false, nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE refunds SET lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,next_query_at=$1,record_version=record_version+1,updated_at=$2 WHERE id=$3`, now.Add(refundFailureDelay), now, claim.refundID)
	if err != nil || tx.Commit() != nil {
		return false, false, ErrUnavailable
	}
	return false, false, nil
}

func (store *postgresStore) listPending(ctx context.Context, userID uint64, filter PendingFilter) ([]Refund, error) {
	var ownerID uint64
	err := store.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	rows, err := store.db.QueryContext(ctx, `SELECT id,prepayment_id,COALESCE(order_id,0),provider_state,materialization_state,amount_cents,currency,created_at,COALESCE(provider_refund_id,''),COALESCE(pending_reason_code,'') FROM refunds WHERE id>$1 AND (materialization_state='PENDING_MANUAL' OR provider_state IN ('CREATE_UNKNOWN','PROCESSING','CLOSED')) ORDER BY id LIMIT $2`, filter.AfterID, filter.Limit)
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
	_, err := tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT',$1,$2,$3,$4,$5,$6,$7,'refund.request','REFUND',$8,$9,$10,'SUCCEEDED','REFUND_REQUESTED',$11,$12,$13,$14)`, actorKind, scope[:], meta.ActorUserID, account, snapshotID, snapshotRole, snapshotVersion, view.ID, key[:], requestID[:], beforeJSON, afterJSON, response, view.RequestedAt)
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
	err := tx.QueryRowContext(ctx, `SELECT response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=$1 AND action='refund.request' AND operation_key_hash=$2`, scope[:], key[:]).Scan(&raw)
	if err != nil {
		return Refund{}, err
	}
	var view Refund
	if json.Unmarshal(raw, &view) != nil || view.ID == 0 || view.PrepaymentID == 0 || view.AmountCents == 0 || view.Currency != "CNY" {
		return Refund{}, ErrUnavailable
	}
	return view, nil
}

func refreshRequestReceipt(ctx context.Context, tx *sql.Tx, refundID uint64) (Refund, error) {
	var view Refund
	var orderID sql.Null[uint64]
	var providerID, pending sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT id,prepayment_id,order_id,provider_state,materialization_state,amount_cents,currency,created_at,provider_refund_id,pending_reason_code FROM refunds WHERE id=$1`, refundID).Scan(&view.ID, &view.PrepaymentID, &orderID, &view.State, &view.MaterializationState, &view.AmountCents, &view.Currency, &view.RequestedAt, &providerID, &pending)
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	if orderID.Valid {
		view.OrderID = uint64(orderID.V)
	}
	if providerID.Valid {
		view.ProviderRefundID = providerID.String
	}
	if pending.Valid {
		view.PendingReason = pending.String
	}
	response, err := json.Marshal(view)
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE action_audits SET response_json=$1 WHERE entry_kind='COMMAND_RECEIPT' AND action='refund.request' AND target_type='REFUND' AND target_id=$2`, response, refundID)
	if err != nil {
		return Refund{}, ErrUnavailable
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return Refund{}, ErrUnavailable
	}
	return view, nil
}

func insertProviderEvidence(ctx context.Context, tx *sql.Tx, refundID uint64, orderID sql.Null[uint64], applyState, reason string, verified VerifiedRefund, now time.Time) error {
	scope := sha256.Sum256([]byte("PROVIDER:REFUND"))
	before, _ := json.Marshal(map[string]string{"observation": string(verified.Refund.State)})
	after, _ := json.Marshal(map[string]string{"apply_state": applyState})
	result := "SUCCEEDED"
	if applyState != "APPLIED" {
		result = "REJECTED"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,action,target_type,target_id,operation_key_hash,result,reason_code,before_state_json,after_state_json,occurred_at) VALUES('SYSTEM_EVIDENCE','PROVIDER',$1,NULL,NULL,'refund.observe','REFUND',$2,NULL,$3,$4,$5,$6,$7)`, scope[:], refundID, result, reason, before, after, now)
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
func retryablePostgreSQL(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && (e.Code == "40P01" || e.Code == "40001" || e.Code == "55P03")
}

func (store *postgresStore) newOutRefundNo(prepaymentID uint64) (string, error) {
	if store.config.Environment == "uat" || store.config.Environment == "production" {
		if store.merchantNumber == nil {
			return "", ErrUnavailable
		}
		return store.merchantNumber(store.config.Environment, "refund")
	}
	return "ORDER_REFUND_" + strconv.FormatUint(prepaymentID, 10), nil
}

func refundNumberCollision(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "uq_refunds_out_refund_no"
}

// This exact field order mirrors the original payment request Go digest, without
// exposing raw request or customer fields through an ownership result.
type refundParentRequest struct {
	AppID       string `json:"appid"`
	MerchantID  string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	TimeExpire  string `json:"time_expire"`
	NotifyURL   string `json:"notify_url"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	PayerOpenID string `json:"payer_openid"`
	QuoteID     string `json:"quote_id"`
	QuoteDigest string `json:"quote_digest"`
}

func (store *postgresStore) parentInEnvironment(ctx context.Context, tx *sql.Tx, id uint64) bool {
	if store.config.Environment == "development" || store.config.Environment == "test" {
		return true
	}
	request, ok := store.parentProof(ctx, tx, id)
	if !ok {
		return false
	}
	environments := map[string]bool{}
	ownProof := false
	if env, kind, ok := wechatpay.ParseMerchantNumber(request.OutTradeNo); ok {
		if kind != "payment" {
			return false
		}
		environments[env] = true
		ownProof = true
	}
	if request.NotifyURL == store.config.PaymentNotifyURL {
		environments[store.config.Environment] = true
		ownProof = true
	}
	var count uint64
	var provider sql.NullString
	if tx.QueryRowContext(ctx, `SELECT count(DISTINCT transaction_id),min(convert_from(transaction_id,'UTF8')) FROM payment_observations WHERE prepayment_id=$1 AND validation='MATCH' AND transaction_id IS NOT NULL`, id).Scan(&count, &provider) != nil || count > 1 {
		return false
	}
	h := func(v string) string { return hex.EncodeToString([]byte(v)) }
	identity, _ := json.Marshal(map[string]string{"kind": "PAYMENT", "appid_hex": h(request.AppID), "mchid_hex": h(request.MerchantID), "out_trade_no_hex": h(request.OutTradeNo), "out_refund_no_hex": "", "provider_id_hex": h(provider.String), "payment_provider_id_hex": h(provider.String)})
	return store.refundConfirmedEnvironment(ctx, tx, identity, environments, ownProof)
}

func (store *postgresStore) parentProof(ctx context.Context, tx *sql.Tx, id uint64) (refundParentRequest, bool) {
	var raw, digest, appid, mchid, number []byte
	var amount int64
	var currency string
	if tx.QueryRowContext(ctx, `SELECT provider_create_request_json,provider_create_request_digest,expected_appid,expected_mchid,out_trade_no,expected_amount_cents,currency FROM prepayments WHERE id=$1`, id).Scan(&raw, &digest, &appid, &mchid, &number, &amount, &currency) != nil {
		return refundParentRequest{}, false
	}
	var request refundParentRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		return refundParentRequest{}, false
	}
	canonical, err := json.Marshal(request)
	if err != nil {
		return refundParentRequest{}, false
	}
	sum := sha256.Sum256(canonical)
	if !bytes.Equal(sum[:], digest) || request.AppID != string(appid) || request.MerchantID != string(mchid) || request.AppID != store.config.AppID || request.MerchantID != store.config.MerchantID || request.OutTradeNo != string(number) || request.AmountCents != amount || request.Currency != currency {
		return refundParentRequest{}, false
	}
	return request, true
}
func (store *postgresStore) refundInEnvironment(ctx context.Context, tx *sql.Tx, id uint64) bool {
	if store.config.Environment == "development" || store.config.Environment == "test" {
		return true
	}
	request, _, ok := store.refundProof(ctx, tx, id)
	if !ok {
		return false
	}
	var parentID uint64
	if tx.QueryRowContext(ctx, `SELECT prepayment_id FROM refunds WHERE id=$1`, id).Scan(&parentID) != nil || !store.parentInEnvironment(ctx, tx, parentID) {
		return false
	}
	if env, kind, ok := wechatpay.ParseMerchantNumber(request.OutRefundNo); ok {
		return env == store.config.Environment && kind == "refund"
	}
	return request.NotifyURL == store.config.RefundNotifyURL
}
func (store *postgresStore) refundProof(ctx context.Context, tx *sql.Tx, id uint64) (ProviderCreateRequest, refundParentRequest, bool) {
	var raw, digest, number, parent []byte
	var parentID, amount uint64
	var currency string
	if tx.QueryRowContext(ctx, `SELECT r.provider_request_json,r.provider_request_digest,r.out_refund_no,r.prepayment_id,p.out_trade_no,r.amount_cents,r.currency FROM refunds r JOIN prepayments p ON p.id=r.prepayment_id WHERE r.id=$1`, id).Scan(&raw, &digest, &number, &parentID, &parent, &amount, &currency) != nil {
		return ProviderCreateRequest{}, refundParentRequest{}, false
	}
	var request ProviderCreateRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || !validProviderCreateRequest(request) {
		return ProviderCreateRequest{}, refundParentRequest{}, false
	}
	canonical, err := json.Marshal(request)
	if err != nil {
		return ProviderCreateRequest{}, refundParentRequest{}, false
	}
	sum := sha256.Sum256(canonical)
	if !bytes.Equal(sum[:], digest) || request.OutRefundNo != string(number) || request.OutTradeNo != string(parent) || request.AmountCents != amount || request.TotalCents != amount || request.Currency != currency {
		return ProviderCreateRequest{}, refundParentRequest{}, false
	}
	parentRequest, ok := store.parentProof(ctx, tx, parentID)
	if !ok {
		return ProviderCreateRequest{}, refundParentRequest{}, false
	}
	return request, parentRequest, true
}

func (store *postgresStore) refundCallbackInEnvironment(ctx context.Context, tx *sql.Tx, id uint64, observed ProviderRefund) bool {
	config := store.config
	if config.Environment == "development" || config.Environment == "test" {
		return true
	}
	request, parent, ok := store.refundProof(ctx, tx, id)
	if !ok || observed.MerchantID != config.MerchantID || observed.OutTradeNo != parent.OutTradeNo || observed.OutRefundNo != request.OutRefundNo || (request.TransactionID != "" && observed.TransactionID != request.TransactionID) {
		return false
	}
	environments := map[string]bool{}
	ownProof := false
	if env, kind, ok := wechatpay.ParseMerchantNumber(request.OutRefundNo); ok {
		if kind != "refund" {
			return false
		}
		environments[env] = true
		ownProof = true
	}
	if env, kind, ok := wechatpay.ParseMerchantNumber(parent.OutTradeNo); ok {
		if kind != "payment" {
			return false
		}
		environments[env] = true
	}
	if parent.NotifyURL == config.PaymentNotifyURL {
		environments[config.Environment] = true
		if request.NotifyURL == config.RefundNotifyURL {
			ownProof = true
		}
	}
	if request.NotifyURL == config.RefundNotifyURL {
		environments[config.Environment] = true
	}
	h := func(v string) string { return hex.EncodeToString([]byte(v)) }
	identity, _ := json.Marshal(map[string]string{"kind": "REFUND", "appid_hex": h(parent.AppID), "mchid_hex": h(parent.MerchantID), "out_trade_no_hex": h(observed.OutTradeNo), "out_refund_no_hex": h(observed.OutRefundNo), "provider_id_hex": h(observed.RefundID), "payment_provider_id_hex": h(observed.TransactionID)})
	return store.refundConfirmedEnvironment(ctx, tx, identity, environments, ownProof)
}
func (store *postgresStore) refundConfirmedEnvironment(ctx context.Context, tx *sql.Tx, identity []byte, environments map[string]bool, ownProof bool) bool {
	config := store.config
	rows, err := tx.QueryContext(ctx, `SELECT id,after_state_json FROM action_audits WHERE entry_kind='SYSTEM_EVIDENCE' AND actor_kind='SYSTEM' AND action='billing.ownership_confirmed' AND after_state_json->'entry'=$1::jsonb ORDER BY id`, identity)
	if err != nil {
		return false
	}
	defer rows.Close()
	leaves := map[uint64]string{}
	seen := map[uint64]bool{}
	for rows.Next() {
		var id uint64
		var raw []byte
		if rows.Scan(&id, &raw) != nil {
			return false
		}
		value, err := sharedRefundObject(raw, "rule_version", "entry", "environment", "source_ref_hex", "evidence_sha256", "supersedes")
		if err != nil {
			return false
		}
		fields := map[string]string{}
		for _, key := range []string{"rule_version", "environment", "source_ref_hex", "evidence_sha256", "supersedes"} {
			v, e := sharedRefundString(value[key])
			if e != nil {
				return false
			}
			fields[key] = v
		}
		parent, e := strconv.ParseUint(fields["supersedes"], 10, 64)
		source, e1 := hex.DecodeString(fields["source_ref_hex"])
		digest, e2 := hex.DecodeString(fields["evidence_sha256"])
		if fields["rule_version"] != "1" || (fields["environment"] != "uat" && fields["environment"] != "production") || e != nil || strconv.FormatUint(parent, 10) != fields["supersedes"] || parent >= id || seen[id] || e1 != nil || e2 != nil || len(source) == 0 || !utf8.Valid(source) || bytes.ContainsRune(source, 0) || hex.EncodeToString(source) != fields["source_ref_hex"] || len(digest) != 32 || hex.EncodeToString(digest) != fields["evidence_sha256"] {
			return false
		}
		if parent != 0 {
			if _, ok := leaves[parent]; !ok {
				return false
			}
			delete(leaves, parent)
		}
		seen[id] = true
		leaves[id] = fields["environment"]
	}
	if rows.Err() != nil {
		return false
	}
	for _, env := range leaves {
		ownProof = true
		environments[env] = true
	}
	return ownProof && len(environments) == 1 && environments[config.Environment]
}
