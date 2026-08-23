package storestatus

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
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/go-sql-driver/mysql"
)

var (
	ErrInvalidCommand      = errors.New("invalid store status command")
	ErrIdempotencyConflict = errors.New("store status idempotency conflict")
	ErrUnavailable         = errors.New("store status unavailable")
)

const storeStatusReceiptJSONLimit = 1024

// Command requests one authenticated singleton operating-status transition.
type Command struct {
	UserID         uint64
	DesiredStatus  storefront.BusinessStatus
	IdempotencyKey string
	RequestID      string
}

// Result is the first committed result for one actor and idempotency key.
type Result struct {
	Before  storefront.BusinessStatus
	After   storefront.BusinessStatus
	Changed bool
}

// Core is the sole storefront operating-status mutation module.
type Core struct {
	db         *sql.DB
	authorizer merchantidentity.Authorizer
	clock      func() time.Time
	commit     func(*sql.Tx) error
}

// New constructs the operating-status command core over shared dependencies.
func New(db *sql.DB, authorizer merchantidentity.Authorizer, clock func() time.Time) *Core {
	return newCore(db, authorizer, clock, func(transaction *sql.Tx) error { return transaction.Commit() })
}

func newCore(db *sql.DB, authorizer merchantidentity.Authorizer, clock func() time.Time, commit func(*sql.Tx) error) *Core {
	return &Core{db: db, authorizer: authorizer, clock: clock, commit: commit}
}

// Apply validates and applies one authorized operating-status command.
func (core *Core) Apply(ctx context.Context, command Command) (Result, error) {
	if !validCommand(command) {
		return Result{}, ErrInvalidCommand
	}
	if core == nil || core.db == nil || core.authorizer == nil || core.clock == nil || core.commit == nil {
		return Result{}, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := core.applyOnce(ctx, command)
		if err == nil {
			return result, nil
		}
		if retryableTransaction(err) && attempt == 0 {
			continue
		}
		if stableApplyError(err) {
			return Result{}, err
		}
		return Result{}, ErrUnavailable
	}
	return Result{}, ErrUnavailable
}

func (core *Core) applyOnce(ctx context.Context, command Command) (result Result, resultErr error) {
	at := core.clock().UTC().Truncate(time.Microsecond)
	if at.IsZero() {
		return Result{}, ErrUnavailable
	}
	transaction, err := core.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if resultErr != nil {
			_ = transaction.Rollback()
		}
	}()
	authorization, err := core.authorizer.AuthorizeInTx(
		ctx,
		transaction,
		command.UserID,
		merchantidentity.ActionStoreStatusWrite,
		merchantidentity.Target{Type: "storefront_settings", ID: 1},
	)
	if err != nil {
		return Result{}, err
	}
	role, roleOK := authorizationRole(authorization)
	if !roleOK {
		return Result{}, ErrUnavailable
	}

	var current string
	if err := transaction.QueryRowContext(ctx, `
		SELECT business_status FROM storefront_settings WHERE id=1 FOR UPDATE
	`).Scan(&current); err != nil {
		return Result{}, err
	}
	before := storefront.BusinessStatus(current)
	if !validStatus(before) {
		return Result{}, ErrUnavailable
	}
	replayed, found, err := readReplay(ctx, transaction, command, authorization)
	if err != nil {
		return Result{}, err
	}
	if found {
		if err := core.commit(transaction); err != nil {
			return Result{}, err
		}
		return replayed, nil
	}
	changed := before != command.DesiredStatus
	reason := "OPERATING_STATUS_UNCHANGED"
	if changed {
		reason = "OPERATING_STATUS_CHANGED"
		update, err := transaction.ExecContext(ctx, `
			UPDATE storefront_settings SET business_status=? WHERE id=1
		`, command.DesiredStatus)
		if err != nil {
			return Result{}, err
		}
		rows, err := update.RowsAffected()
		if err != nil {
			return Result{}, err
		}
		if rows != 1 {
			return Result{}, ErrUnavailable
		}
	}
	requestEvidence, responseJSON, afterStateJSON, err := encodeReceipt(command.DesiredStatus, Result{
		Before: before, After: command.DesiredStatus, Changed: changed,
	})
	if err != nil {
		return Result{}, ErrUnavailable
	}
	actorScopeHash := merchantActorScopeHash(command.UserID, authorization.MerchantAccountID)
	operationKeyHash := commandOperationKeyHash(command.IdempotencyKey)
	requestIDHash := commandRequestIDHash(command.RequestID)
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,
			actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,
			action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,
			result,reason_code,before_state_json,after_state_json,response_json,occurred_at
		) VALUES (
			'COMMAND_RECEIPT','MERCHANT',?,?,?,?,?,?,?,
			'storefront_settings',1,NULL,?,?,'SUCCEEDED',?,?,?,?,?
		)
	`,
		actorScopeHash[:], command.UserID, authorization.MerchantAccountID, authorization.MerchantAccountID,
		role, authorization.AuthVersion, merchantidentity.ActionStoreStatusWrite,
		operationKeyHash[:], requestIDHash[:], reason, requestEvidence, afterStateJSON, responseJSON, at,
	); err != nil {
		return Result{}, err
	}
	if err := core.commit(transaction); err != nil {
		return Result{}, err
	}
	return Result{Before: before, After: command.DesiredStatus, Changed: changed}, nil
}

func readReplay(ctx context.Context, transaction *sql.Tx, command Command, authorization merchantidentity.Authorization) (Result, bool, error) {
	actorScopeHash := merchantActorScopeHash(command.UserID, authorization.MerchantAccountID)
	operationKeyHash := commandOperationKeyHash(command.IdempotencyKey)
	var actorUserID, actorAccountID, snapshotAccountID, snapshotAuthVersion, targetID uint64
	var role merchantidentity.Role
	var targetType, resultValue, reason string
	var requestEvidence, afterStateJSON, responseJSON []byte
	err := transaction.QueryRowContext(ctx, `
		SELECT actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,
		       actor_auth_version_snapshot,target_type,target_id,result,reason_code,
		       before_state_json,after_state_json,response_json
		FROM action_audits
		WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT'
		  AND actor_scope_hash=? AND action=? AND operation_key_hash=?
		LIMIT 1
	`, actorScopeHash[:], merchantidentity.ActionStoreStatusWrite, operationKeyHash[:]).Scan(
		&actorUserID, &actorAccountID, &snapshotAccountID, &role, &snapshotAuthVersion,
		&targetType, &targetID, &resultValue, &reason, &requestEvidence, &afterStateJSON, &responseJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, err
	}
	if actorUserID != command.UserID || actorAccountID != authorization.MerchantAccountID ||
		snapshotAccountID != actorAccountID || snapshotAuthVersion == 0 ||
		(role != merchantidentity.RoleOwner && role != merchantidentity.RoleSubaccount) ||
		targetType != "storefront_settings" || targetID != 1 || resultValue != "SUCCEEDED" {
		return Result{}, false, ErrUnavailable
	}
	if err := matchRequestEvidence(requestEvidence, command.DesiredStatus); err != nil {
		return Result{}, false, err
	}
	var replay Result
	if decodeReceiptJSON(responseJSON, &replay) != nil || !validStatus(replay.Before) || !validStatus(replay.After) || replay.Changed != (replay.Before != replay.After) {
		return Result{}, false, ErrUnavailable
	}
	var afterState receiptAfterState
	if decodeReceiptJSON(afterStateJSON, &afterState) != nil || afterState.BusinessStatus != replay.After {
		return Result{}, false, ErrUnavailable
	}
	wantReason := "OPERATING_STATUS_UNCHANGED"
	if replay.Changed {
		wantReason = "OPERATING_STATUS_CHANGED"
	}
	if reason != wantReason {
		return Result{}, false, ErrUnavailable
	}
	return replay, true, nil
}

type receiptRequest struct {
	DesiredStatus storefront.BusinessStatus `json:"desired_status"`
}

type receiptRequestEvidence struct {
	RequestDigest string `json:"request_digest"`
}

type receiptResponse struct {
	Before  storefront.BusinessStatus `json:"before"`
	After   storefront.BusinessStatus `json:"after"`
	Changed bool                      `json:"changed"`
}

type receiptAfterState struct {
	BusinessStatus storefront.BusinessStatus `json:"business_status"`
}

func encodeReceipt(desired storefront.BusinessStatus, result Result) ([]byte, []byte, []byte, error) {
	digest, err := receiptRequestDigest(desired)
	if err != nil {
		return nil, nil, nil, err
	}
	evidence, err := json.Marshal(receiptRequestEvidence{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		return nil, nil, nil, err
	}
	response, err := json.Marshal(receiptResponse{Before: result.Before, After: result.After, Changed: result.Changed})
	if err != nil {
		return nil, nil, nil, err
	}
	afterState, err := json.Marshal(receiptAfterState{BusinessStatus: result.After})
	if err != nil {
		return nil, nil, nil, err
	}
	return evidence, response, afterState, nil
}

func matchRequestEvidence(raw []byte, desired storefront.BusinessStatus) error {
	var evidence receiptRequestEvidence
	if decodeReceiptJSON(raw, &evidence) != nil {
		return ErrUnavailable
	}
	stored, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(stored) != sha256.Size || hex.EncodeToString(stored) != evidence.RequestDigest {
		return ErrUnavailable
	}
	want, err := receiptRequestDigest(desired)
	if err != nil {
		return ErrUnavailable
	}
	if subtle.ConstantTimeCompare(stored, want[:]) != 1 {
		return ErrIdempotencyConflict
	}
	return nil
}

func receiptRequestDigest(desired storefront.BusinessStatus) ([sha256.Size]byte, error) {
	raw, err := json.Marshal(receiptRequest{DesiredStatus: desired})
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func decodeReceiptJSON(raw []byte, target any) error {
	if len(raw) == 0 || len(raw) > storeStatusReceiptJSONLimit {
		return ErrUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrUnavailable
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrUnavailable
	}
	return nil
}

func merchantActorScopeHash(userID, accountID uint64) [sha256.Size]byte {
	return sha256.Sum256([]byte("merchant:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(accountID, 10)))
}

func commandOperationKeyHash(key string) [sha256.Size]byte {
	return sha256.Sum256([]byte("operation:" + key))
}

func commandRequestIDHash(requestID string) [sha256.Size]byte {
	return sha256.Sum256([]byte("request:" + requestID))
}

func retryableTransaction(err error) bool {
	if errors.Is(err, merchantidentity.ErrUnavailable) {
		return true
	}
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1213 || mysqlError.Number == 1205)
}

func stableApplyError(err error) bool {
	return errors.Is(err, ErrIdempotencyConflict) ||
		errors.Is(err, ErrUnavailable) ||
		errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable) ||
		errors.Is(err, merchantidentity.ErrForbidden) ||
		errors.Is(err, merchantidentity.ErrUnavailable)
}

func authorizationRole(authorization merchantidentity.Authorization) (merchantidentity.Role, bool) {
	if authorization.MerchantAccountID == 0 || authorization.RecordVersion == 0 || authorization.AuthVersion == 0 {
		return "", false
	}
	switch authorization.Actor {
	case merchantidentity.ActorMerchantOwner:
		return merchantidentity.RoleOwner, true
	case merchantidentity.ActorMerchantSubaccount:
		return merchantidentity.RoleSubaccount, true
	default:
		return "", false
	}
}

func validCommand(command Command) bool {
	return command.UserID > 0 &&
		validStatus(command.DesiredStatus) &&
		validText(command.IdempotencyKey, 0) &&
		validText(command.RequestID, 64)
}

func validText(value string, maxBytes int) bool {
	return value != "" &&
		(maxBytes == 0 || len(value) <= maxBytes) &&
		utf8.ValidString(value) &&
		strings.TrimSpace(value) == value
}

func validStatus(status storefront.BusinessStatus) bool {
	switch status {
	case storefront.BusinessOpen, storefront.BusinessClosed, storefront.BusinessCutoff:
		return true
	default:
		return false
	}
}
