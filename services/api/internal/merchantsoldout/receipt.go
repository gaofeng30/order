package merchantsoldout

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type receiptRequest struct {
	ProductID   uint64 `json:"product_id"`
	ServiceDate string `json:"service_date"`
	SoldOut     bool   `json:"sold_out"`
}

type receiptResponse struct {
	ProductID   uint64 `json:"product_id"`
	ServiceDate string `json:"service_date"`
	SoldOut     bool   `json:"sold_out"`
}

func (commander *Commander) replayAuthorized(ctx context.Context, meta fulfillment.WriteMeta, request receiptRequest) (bool, error) {
	transaction, err := commander.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer transaction.Rollback()
	authorization, err := commander.authorizer.AuthorizeInTx(
		ctx,
		transaction,
		meta.ActorUserID,
		merchantidentity.ActionProductSoldOutWrite,
		merchantidentity.Target{Type: "PRODUCT", ID: request.ProductID},
	)
	if err != nil {
		return false, err
	}
	if _, ok := persistedRole(authorization.Actor); !ok || authorization.MerchantAccountID == 0 || authorization.RecordVersion == 0 || authorization.AuthVersion == 0 {
		return false, fulfillment.ErrUnavailable
	}
	found, err := readReceiptInTx(ctx, transaction, meta, authorization, request)
	if err != nil {
		return false, err
	}
	if err := transaction.Commit(); err != nil {
		return false, err
	}
	return found, nil
}

func readReceiptInTx(ctx context.Context, transaction *sql.Tx, meta fulfillment.WriteMeta, authorization merchantidentity.Authorization, request receiptRequest) (bool, error) {
	scope := merchantScopeHash(meta.ActorUserID, authorization.MerchantAccountID)
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	var targetType string
	var targetID uint64
	var beforeRaw, afterRaw, responseRaw []byte
	err := transaction.QueryRowContext(ctx, `SELECT target_type,target_id,before_state_json,after_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_scope_hash=? AND action=? AND operation_key_hash=? LIMIT 1 FOR SHARE`, scope[:], soldOutAction, operation[:]).Scan(&targetType, &targetID, &beforeRaw, &afterRaw, &responseRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := matchReceiptEvidence(beforeRaw, request); err != nil {
		return false, err
	}
	afterSoldOut, ok := decodeSoldOutState(afterRaw)
	if !ok || afterSoldOut != request.SoldOut {
		return false, fulfillment.ErrUnavailable
	}
	response, ok := decodeReceiptResponse(responseRaw)
	if !ok || targetType != "PRODUCT" || targetID != request.ProductID || response.ProductID != request.ProductID || response.ServiceDate != request.ServiceDate || response.SoldOut != request.SoldOut {
		return false, fulfillment.ErrUnavailable
	}
	return true, nil
}

func (commander *Commander) appendReceiptInTx(
	ctx context.Context,
	transaction *sql.Tx,
	meta fulfillment.WriteMeta,
	authorization merchantidentity.Authorization,
	role merchantidentity.Role,
	request receiptRequest,
	response receiptResponse,
	beforeSoldOut bool,
	changed bool,
) error {
	digest, err := requestDigest(request)
	if err != nil {
		return fulfillment.ErrUnavailable
	}
	beforeRaw, err := json.Marshal(struct {
		RequestDigest string `json:"request_digest"`
		SoldOut       bool   `json:"sold_out"`
	}{RequestDigest: hex.EncodeToString(digest[:]), SoldOut: beforeSoldOut})
	if err != nil {
		return fulfillment.ErrUnavailable
	}
	afterRaw, err := json.Marshal(struct {
		SoldOut bool `json:"sold_out"`
	}{SoldOut: request.SoldOut})
	if err != nil {
		return fulfillment.ErrUnavailable
	}
	responseRaw, err := json.Marshal(response)
	if err != nil {
		return fulfillment.ErrUnavailable
	}
	scope := merchantScopeHash(meta.ActorUserID, authorization.MerchantAccountID)
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	requestID := sha256.Sum256([]byte("request:" + meta.RequestID))
	reason := "SOLD_OUT_UNCHANGED"
	if changed {
		reason = "SOLD_OUT_CHANGED"
	}
	occurredAt := commander.clock().UTC().Truncate(time.Microsecond)
	if occurredAt.IsZero() {
		return fulfillment.ErrUnavailable
	}
	_, err = transaction.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','MERCHANT',?,?,?,?,?,?,?,'PRODUCT',?,NULL,?,?,'SUCCEEDED',?,?,?,?,?)`, scope[:], meta.ActorUserID, authorization.MerchantAccountID, authorization.MerchantAccountID, string(role), authorization.AuthVersion, soldOutAction, request.ProductID, operation[:], requestID[:], reason, beforeRaw, afterRaw, responseRaw, occurredAt)
	if err != nil {
		var mysqlError *mysqlDriver.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return audit.ErrDuplicateReceipt
		}
		return err
	}
	return nil
}

func requestDigest(request receiptRequest) ([sha256.Size]byte, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func matchReceiptEvidence(raw []byte, request receiptRequest) error {
	if len(raw) == 0 || len(raw) > 256 || !json.Valid(raw) {
		return fulfillment.ErrUnavailable
	}
	var evidence struct {
		RequestDigest string `json:"request_digest"`
		SoldOut       *bool  `json:"sold_out"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&evidence) != nil || decoder.Decode(&struct{}{}) != io.EOF || evidence.SoldOut == nil {
		return fulfillment.ErrUnavailable
	}
	stored, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(stored) != sha256.Size {
		return fulfillment.ErrUnavailable
	}
	digest, err := requestDigest(request)
	if err != nil {
		return fulfillment.ErrUnavailable
	}
	if subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
		return fulfillment.ErrIdempotencyConflict
	}
	return nil
}

func decodeSoldOutState(raw []byte) (bool, bool) {
	if len(raw) == 0 || len(raw) > 128 || !json.Valid(raw) {
		return false, false
	}
	var stored struct {
		SoldOut *bool `json:"sold_out"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&stored) != nil || decoder.Decode(&struct{}{}) != io.EOF || stored.SoldOut == nil {
		return false, false
	}
	return *stored.SoldOut, true
}

func decodeReceiptResponse(raw []byte) (receiptResponse, bool) {
	if len(raw) == 0 || len(raw) > 1024 || !json.Valid(raw) {
		return receiptResponse{}, false
	}
	var stored struct {
		ProductID   *uint64 `json:"product_id"`
		ServiceDate *string `json:"service_date"`
		SoldOut     *bool   `json:"sold_out"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&stored) != nil || decoder.Decode(&struct{}{}) != io.EOF || stored.ProductID == nil || stored.ServiceDate == nil || stored.SoldOut == nil || *stored.ProductID == 0 {
		return receiptResponse{}, false
	}
	return receiptResponse{ProductID: *stored.ProductID, ServiceDate: *stored.ServiceDate, SoldOut: *stored.SoldOut}, true
}

func merchantScopeHash(userID, accountID uint64) [sha256.Size]byte {
	return sha256.Sum256([]byte("merchant:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(accountID, 10)))
}

func persistedRole(actor merchantidentity.Actor) (merchantidentity.Role, bool) {
	switch actor {
	case merchantidentity.ActorMerchantOwner:
		return merchantidentity.RoleOwner, true
	case merchantidentity.ActorMerchantSubaccount:
		return merchantidentity.RoleSubaccount, true
	default:
		return "", false
	}
}
