package storestatus

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
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
	keyHash := sha256.Sum256([]byte(command.IdempotencyKey))
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
	replayed, found, err := readReplay(ctx, transaction, command, keyHash)
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
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO merchant_action_audits(
			merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
			actor_user_id,action,result,reason,target_type,target_id,request_id,idempotency_key_hash,
			state_before,state_after,occurred_at
		) VALUES (?,?,?,?,?,?,'SUCCEEDED',?,'storefront_settings',1,?,?,?,?,?)
	`,
		authorization.MerchantAccountID, authorization.MerchantAccountID, role, authorization.AuthVersion,
		command.UserID, merchantidentity.ActionStoreStatusWrite, reason, []byte(command.RequestID), keyHash[:], before, command.DesiredStatus, at,
	); err != nil {
		return Result{}, err
	}
	if err := core.commit(transaction); err != nil {
		return Result{}, err
	}
	return Result{Before: before, After: command.DesiredStatus, Changed: changed}, nil
}

func readReplay(ctx context.Context, transaction *sql.Tx, command Command, keyHash [32]byte) (Result, bool, error) {
	var targetType, resultValue, reason, beforeValue, afterValue string
	var targetID uint64
	err := transaction.QueryRowContext(ctx, `
		SELECT target_type,target_id,result,reason,state_before,state_after
		FROM merchant_action_audits
		WHERE actor_user_id=? AND action=? AND idempotency_key_hash=?
		ORDER BY id ASC
		LIMIT 1
	`, command.UserID, merchantidentity.ActionStoreStatusWrite, keyHash[:]).Scan(
		&targetType, &targetID, &resultValue, &reason, &beforeValue, &afterValue,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, err
	}
	before := storefront.BusinessStatus(beforeValue)
	after := storefront.BusinessStatus(afterValue)
	if !validStatus(before) || !validStatus(after) {
		return Result{}, false, ErrUnavailable
	}
	changed := before != after
	wantReason := "OPERATING_STATUS_UNCHANGED"
	if changed {
		wantReason = "OPERATING_STATUS_CHANGED"
	}
	if resultValue != "SUCCEEDED" || reason != wantReason {
		return Result{}, false, ErrUnavailable
	}
	if targetType != "storefront_settings" || targetID != 1 || after != command.DesiredStatus {
		return Result{}, false, ErrIdempotencyConflict
	}
	return Result{Before: before, After: after, Changed: changed}, true, nil
}

func retryableTransaction(err error) bool {
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
