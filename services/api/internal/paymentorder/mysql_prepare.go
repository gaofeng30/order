package paymentorder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/go-sql-driver/mysql"
)

type prepaymentRecord struct {
	id                   uint64
	userID               uint64
	quoteID              uint64
	idempotencyKeyHash   [32]byte
	outTradeNo           string
	expectedAppID        string
	expectedMerchantID   string
	expectedAmountCents  int64
	currency             string
	createRequestJSON    []byte
	createRequestDigest  [32]byte
	effectiveDeadline    time.Time
	providerState        ProviderState
	createAttemptedAt    sql.NullTime
	wxRequestPaymentJSON []byte
	providerPrepayID     []byte
	lastQueriedAt        sql.NullTime
	materialState        MaterializationState
	pendingReason        sql.NullString
	materializedAt       sql.NullTime
	leaseKind            sql.NullString
	leaseOwner           []byte
	leaseExpiresAt       sql.NullTime
	recordVersion        uint64
	nextReconcileAt      sql.NullTime
	createdAt            time.Time
	updatedAt            time.Time
}

func (record prepaymentRecord) public() Prepayment {
	result := Prepayment{
		ID: record.id, UserID: record.userID, QuoteID: record.quoteID,
		State: record.providerState, MaterializationState: record.materialState,
		ExpiresAt: record.effectiveDeadline.UTC(),
	}
	if len(record.wxRequestPaymentJSON) > 0 {
		var requestPayment struct {
			TimeStamp string `json:"timeStamp"`
			NonceStr  string `json:"nonceStr"`
			Package   string `json:"package"`
			SignType  string `json:"signType"`
			PaySign   string `json:"paySign"`
		}
		if json.Unmarshal(record.wxRequestPaymentJSON, &requestPayment) == nil &&
			requestPayment.TimeStamp != "" && requestPayment.NonceStr != "" && requestPayment.Package != "" && requestPayment.SignType == "RSA" && requestPayment.PaySign != "" {
			converted := structToRequestPayment(requestPayment)
			result.WxRequestPayment = &converted
		}
	}
	return result
}

func structToRequestPayment(value struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}) (result wechatpay.RequestPayment) {
	return wechatpay.RequestPayment{
		TimeStamp: value.TimeStamp, NonceStr: value.NonceStr, Package: value.Package,
		SignType: value.SignType, PaySign: value.PaySign,
	}
}

type claimedCreate struct {
	id      uint64
	owner   [16]byte
	version uint64
	request ProviderCreateRequest
}

func hashOperationKey(value string) [32]byte { return sha256.Sum256([]byte(value)) }

func userScopeHash(userID uint64) [32]byte {
	var material [13]byte
	copy(material[:5], "USER\x00")
	binary.BigEndian.PutUint64(material[5:], userID)
	return sha256.Sum256(material[:])
}

func (service *Service) prepareOnce(
	ctx context.Context,
	meta WriteMeta,
	quoteID uint64,
	keyHash [32]byte,
	openID string,
	owner [16]byte,
	now time.Time,
) (claimedCreate, error) {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return claimedCreate{}, err
	}
	defer transaction.Rollback()

	snapshot, err := service.quotes.FinalizeForPrepayInTx(ctx, transaction, meta.ActorUserID, quoteID, now)
	if err != nil {
		return claimedCreate{}, err
	}
	pickupAt, err := pickupInstant(snapshot.Pickup.Date, snapshot.Pickup.Time)
	if err != nil {
		return claimedCreate{}, ErrUnavailable
	}
	deadline := EffectiveDeadline(snapshot.CreatedAt.UTC(), pickupAt)
	if !deadline.Equal(snapshot.ExpiresAt.UTC()) {
		return claimedCreate{}, ErrUnavailable
	}
	if err := RequireCreateWindow(now, deadline); err != nil {
		return claimedCreate{}, err
	}
	outTradeNo, err := randomOutTradeNo(quoteID)
	if err != nil {
		return claimedCreate{}, ErrUnavailable
	}
	request := ProviderCreateRequest{
		AppID: service.config.AppID, MerchantID: service.config.MerchantID,
		Description: service.config.Description, OutTradeNo: outTradeNo,
		TimeExpire: deadline.UTC().Format(time.RFC3339), NotifyURL: service.config.PaymentNotifyURL,
		AmountCents: snapshot.PayableCents, Currency: "CNY", PayerOpenID: openID,
		QuoteID: decimalID(snapshot.ID), QuoteDigest: hex.EncodeToString(snapshot.SnapshotDigest[:]),
	}
	requestJSON, err := json.Marshal(request)
	if err != nil || !validProviderCreateRequest(request) {
		return claimedCreate{}, ErrUnavailable
	}
	requestDigest := sha256.Sum256(requestJSON)
	result, err := transaction.ExecContext(ctx, `
		INSERT INTO prepayments(
			user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,
			expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,
			effective_deadline,provider_state,create_attempted_at,wx_request_payment_json,provider_prepay_id,
			last_queried_at,materialization_state,pending_reason_code,materialized_at,
			lease_kind,lease_owner,lease_expires_at,record_version,next_reconcile_at,created_at,updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,? ,?,'READY',NULL,NULL,NULL,NULL,'AWAITING_PAYMENT',NULL,NULL,NULL,NULL,NULL,1,NULL,?,?)
	`, meta.ActorUserID, snapshot.ID, keyHash[:], []byte(outTradeNo), []byte(service.config.AppID), []byte(service.config.MerchantID),
		snapshot.PayableCents, "CNY", requestJSON, requestDigest[:], deadline, now, now)
	if err != nil {
		return claimedCreate{}, err
	}
	id, err := result.LastInsertId()
	if err != nil || id <= 0 {
		return claimedCreate{}, ErrUnavailable
	}
	leaseExpiresAt := now.Add(service.config.LeaseDuration)
	update, err := transaction.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state='CREATE_CLAIMED',create_attempted_at=?,lease_kind='CREATE',lease_owner=?,
		    lease_expires_at=?,record_version=record_version+1,next_reconcile_at=?,updated_at=?
		WHERE id=? AND provider_state='READY' AND record_version=1 AND lease_kind IS NULL
	`, now, owner[:], leaseExpiresAt, leaseExpiresAt, now, uint64(id))
	if err != nil {
		return claimedCreate{}, err
	}
	rows, err := update.RowsAffected()
	if err != nil || rows != 1 {
		return claimedCreate{}, ErrUnavailable
	}
	if err := transaction.Commit(); err != nil {
		return claimedCreate{}, err
	}
	return claimedCreate{id: uint64(id), owner: owner, version: 2, request: request}, nil
}

func (service *Service) finishCreateSuccess(ctx context.Context, claimed claimedCreate, result ProviderCreateResult, now time.Time) error {
	wxJSON, err := json.Marshal(result.RequestPayment)
	if err != nil {
		return ErrUnavailable
	}
	update, err := service.db.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state='PAYMENT_REQUESTED',provider_prepay_id=?,wx_request_payment_json=?,
		    lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,record_version=record_version+1,
		    next_reconcile_at=?,updated_at=?
		WHERE id=? AND provider_state='CREATE_CLAIMED' AND lease_kind='CREATE' AND lease_owner=? AND record_version=?
	`, []byte(result.PrepayID), wxJSON, now.Add(service.config.ReconcileInterval), now, claimed.id, claimed.owner[:], claimed.version)
	if err != nil {
		return ErrUnavailable
	}
	rows, err := update.RowsAffected()
	if err != nil {
		return ErrUnavailable
	}
	if rows != 1 {
		record, readErr := service.readPrepaymentByID(ctx, claimed.id)
		if readErr == nil && (record.providerState == ProviderPaid || record.providerState == ProviderClosed) {
			return nil
		}
		return ErrUnavailable
	}
	return nil
}

func (service *Service) finishCreateUnknown(ctx context.Context, claimed claimedCreate, now time.Time) error {
	update, err := service.db.ExecContext(ctx, `
		UPDATE prepayments
		SET provider_state='CREATE_UNKNOWN',lease_kind=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,next_reconcile_at=?,updated_at=?
		WHERE id=? AND provider_state='CREATE_CLAIMED' AND lease_kind='CREATE' AND lease_owner=? AND record_version=?
	`, now.Add(service.config.ReconcileInterval), now, claimed.id, claimed.owner[:], claimed.version)
	if err != nil {
		return ErrUnavailable
	}
	rows, err := update.RowsAffected()
	if err != nil || rows != 1 {
		return ErrUnavailable
	}
	return nil
}

func (service *Service) findPrepareReplay(ctx context.Context, userID, quoteID uint64, keyHash [32]byte) (prepaymentRecord, bool, error) {
	record, found, err := service.readPrepaymentWhere(ctx, `WHERE user_id=? AND idempotency_key_hash=?`, userID, keyHash[:])
	if err != nil {
		return prepaymentRecord{}, false, err
	}
	if found {
		if record.quoteID != quoteID {
			return prepaymentRecord{}, false, ErrIdempotencyConflict
		}
		return record, true, nil
	}
	record, found, err = service.readPrepaymentWhere(ctx, `WHERE quote_id=?`, quoteID)
	if err != nil {
		return prepaymentRecord{}, false, err
	}
	if !found {
		return prepaymentRecord{}, false, nil
	}
	if record.userID != userID {
		return prepaymentRecord{}, false, ErrNotFound
	}
	return record, true, nil
}

func (service *Service) readPrepaymentByID(ctx context.Context, id uint64) (prepaymentRecord, error) {
	record, found, err := service.readPrepaymentWhere(ctx, `WHERE id=?`, id)
	if err != nil {
		return prepaymentRecord{}, err
	}
	if !found {
		return prepaymentRecord{}, ErrNotFound
	}
	return record, nil
}

func (service *Service) readPrepaymentWhere(ctx context.Context, where string, args ...any) (prepaymentRecord, bool, error) {
	row := service.db.QueryRowContext(ctx, prepaymentSelect+" "+where, args...)
	record, err := scanPrepayment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return prepaymentRecord{}, false, nil
	}
	if err != nil {
		return prepaymentRecord{}, false, ErrUnavailable
	}
	if !record.validStored() {
		return prepaymentRecord{}, false, ErrUnavailable
	}
	return record, true, nil
}

const prepaymentSelect = `SELECT
	id,user_id,quote_id,idempotency_key_hash,out_trade_no,expected_appid,expected_mchid,
	expected_amount_cents,currency,provider_create_request_json,provider_create_request_digest,
	effective_deadline,provider_state,create_attempted_at,wx_request_payment_json,provider_prepay_id,
	last_queried_at,materialization_state,pending_reason_code,materialized_at,
	lease_kind,lease_owner,lease_expires_at,record_version,next_reconcile_at,created_at,updated_at
	FROM prepayments`

type rowScanner interface{ Scan(...any) error }

func scanPrepayment(row rowScanner) (prepaymentRecord, error) {
	var record prepaymentRecord
	var keyHash, requestDigest []byte
	var providerState, materialState string
	err := row.Scan(
		&record.id, &record.userID, &record.quoteID, &keyHash, &record.outTradeNo,
		&record.expectedAppID, &record.expectedMerchantID, &record.expectedAmountCents, &record.currency,
		&record.createRequestJSON, &requestDigest, &record.effectiveDeadline, &providerState,
		&record.createAttemptedAt, &record.wxRequestPaymentJSON, &record.providerPrepayID,
		&record.lastQueriedAt, &materialState, &record.pendingReason, &record.materializedAt,
		&record.leaseKind, &record.leaseOwner, &record.leaseExpiresAt, &record.recordVersion,
		&record.nextReconcileAt, &record.createdAt, &record.updatedAt,
	)
	if err != nil {
		return prepaymentRecord{}, err
	}
	if len(keyHash) != 32 || len(requestDigest) != 32 {
		return prepaymentRecord{}, ErrUnavailable
	}
	copy(record.idempotencyKeyHash[:], keyHash)
	copy(record.createRequestDigest[:], requestDigest)
	record.providerState = ProviderState(providerState)
	record.materialState = MaterializationState(materialState)
	return record, nil
}

func (record prepaymentRecord) validStored() bool {
	if record.id == 0 || record.userID == 0 || record.quoteID == 0 || record.outTradeNo == "" ||
		record.expectedAppID == "" || record.expectedMerchantID == "" || record.expectedAmountCents <= 0 ||
		record.currency != "CNY" || record.effectiveDeadline.IsZero() || record.recordVersion == 0 {
		return false
	}
	_, canonical, err := canonicalProviderCreateRequest(record.createRequestJSON)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(canonical)
	return digest == record.createRequestDigest
}

func canonicalProviderCreateRequest(raw []byte) (ProviderCreateRequest, []byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request ProviderCreateRequest
	if err := decoder.Decode(&request); err != nil || requireJSONEnd(decoder) != nil || !validProviderCreateRequest(request) {
		return ProviderCreateRequest{}, nil, ErrUnavailable
	}
	canonical, err := json.Marshal(request)
	if err != nil {
		return ProviderCreateRequest{}, nil, ErrUnavailable
	}
	return request, canonical, nil
}

func isDuplicate(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func isRetryableMySQL(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1205 || mysqlError.Number == 1213)
}

func pickupInstant(date, at string) (time.Time, error) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	value, err := time.ParseInLocation("2006-01-02 15:04", date+" "+at, location)
	if err != nil {
		return time.Time{}, err
	}
	return value.UTC(), nil
}

func decimalID(value uint64) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	index := len(buf)
	for value > 0 {
		index--
		buf[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[index:])
}
