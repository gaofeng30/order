package merchantidentity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgreSQLAdminApplication struct {
	db       *sql.DB
	mini     Application
	receipts *audit.ReceiptStore
	now      func() time.Time
}

func NewPostgreSQLAdminApplication(db *sql.DB, mini Application) *PostgreSQLAdminApplication {
	return &PostgreSQLAdminApplication{db: db, mini: mini, receipts: audit.NewReceiptStore(db), now: time.Now}
}
func (a *PostgreSQLAdminApplication) CurrentAccount(ctx context.Context, userID uint64) (Account, error) {
	return a.readAccount(ctx, `SELECT id,name,convert_from(phone,'UTF8'),role,enabled,(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID)
}

type merchantScanner interface{ Scan(...any) error }

func (a *PostgreSQLAdminApplication) readAccount(ctx context.Context, query string, args ...any) (Account, error) {
	return a.scanAccount(a.db.QueryRowContext(ctx, query, args...))
}
func (a *PostgreSQLAdminApplication) scanAccount(row merchantScanner) (Account, error) {
	var out Account
	var phone string
	err := row.Scan(&out.ID, &out.Name, &phone, &out.Role, &out.Enabled, &out.Bound)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrAdminNotFound
	}
	if err != nil {
		return Account{}, ErrAdminUnavailable
	}
	out.MaskedPhone = maskMerchantPhone(phone)
	return out, nil
}
func (a *PostgreSQLAdminApplication) ListAccounts(ctx context.Context, userID uint64, q string) ([]Account, error) {
	var owner uint64
	if a.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&owner) != nil {
		return nil, ErrAdminUnavailable
	}
	rows, err := a.db.QueryContext(ctx, `SELECT id,name,convert_from(phone,'UTF8'),role,enabled,(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE deleted_at IS NULL AND ($1='' OR name LIKE '%'||$2||'%' OR convert_from(phone,'UTF8') LIKE '%'||$3||'%') ORDER BY id`, q, q, q)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var item Account
		var phone string
		if rows.Scan(&item.ID, &item.Name, &phone, &item.Role, &item.Enabled, &item.Bound) != nil {
			return nil, ErrAdminUnavailable
		}
		item.MaskedPhone = maskMerchantPhone(phone)
		out = append(out, item)
	}
	if rows.Err() != nil {
		return nil, ErrAdminUnavailable
	}
	return out, nil
}
func (a *PostgreSQLAdminApplication) ExecuteAccount(ctx context.Context, meta AdminWriteMeta, cmd AccountCommand) (*Account, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,bound_user_id,role,enabled,auth_version,name,convert_from(phone,'UTF8'),(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE deleted_at IS NULL ORDER BY id FOR UPDATE`)
	if err != nil {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	type locked struct {
		id      uint64
		user    sql.Null[uint64]
		role    Role
		enabled bool
		auth    uint64
		name    string
		phone   string
		bound   bool
	}
	all := []locked{}
	var actor locked
	for rows.Next() {
		var item locked
		if rows.Scan(&item.id, &item.user, &item.role, &item.enabled, &item.auth, &item.name, &item.phone, &item.bound) != nil {
			rows.Close()
			tx.Rollback()
			return nil, ErrAdminUnavailable
		}
		all = append(all, item)
		if item.user.Valid && item.user.V == meta.ActorUserID {
			actor = item
		}
	}
	if rows.Err() != nil {
		rows.Close()
		return nil, ErrAdminUnavailable
	}
	rows.Close()
	if actor.id == 0 || actor.role != RoleOwner || !actor.enabled {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	var replay struct {
		Account *Account `json:"account"`
		Deleted bool     `json:"deleted"`
	}
	found, replayErr := a.receipts.ReplayInTx(ctx, tx, meta.ActorUserID, actor.id, string(cmd.Kind), meta.IdempotencyKey, cmd, &replay)
	if errors.Is(replayErr, audit.ErrIdempotencyConflict) {
		return nil, ErrAdminIdempotencyConflict
	}
	if replayErr != nil {
		return nil, ErrAdminUnavailable
	}
	if found {
		return replay.Account, nil
	}
	enabledOwners := func(except uint64) int {
		n := 0
		for _, x := range all {
			if x.id != except && x.role == RoleOwner && x.enabled {
				n++
			}
		}
		return n
	}
	find := func(id uint64) (locked, bool) {
		for _, item := range all {
			if item.id == id {
				return item, true
			}
		}
		return locked{}, false
	}
	var result *Account
	switch cmd.Kind {
	case CreateAccount:
		phone, ok := canonicalAdminPhone(cmd.Phone)
		if !ok || strings.TrimSpace(cmd.Name) == "" || (cmd.Role != RoleOwner && cmd.Role != RoleSubaccount) {
			tx.Rollback()
			return nil, ErrAdminInvalidInput
		}
		var id uint64
		e := tx.QueryRowContext(ctx, `INSERT INTO merchant_accounts(phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at,created_by,updated_by,deleted_at,deleted_by_account_id) VALUES($1,$2,$3,TRUE,1,1,NULL,NULL,clock_timestamp(),clock_timestamp(),$4,$5,NULL,NULL) RETURNING id`, []byte(phone), strings.TrimSpace(cmd.Name), cmd.Role, actor.id, actor.id).Scan(&id)
		if e != nil {
			tx.Rollback()
			return nil, merchantSQLError(e)
		}
		result = &Account{ID: id, Name: strings.TrimSpace(cmd.Name), MaskedPhone: maskMerchantPhone(phone), Role: cmd.Role, Enabled: true}
	case UpdateAccount:
		target, found := find(cmd.AccountID)
		if !found {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
		if target.role == RoleOwner && target.enabled && cmd.Role != RoleOwner && enabledOwners(target.id) == 0 {
			tx.Rollback()
			return nil, ErrLastOwner
		}
		targetPhone := cmd.Phone
		if targetPhone == "" {
			targetPhone = target.phone
		}
		phone, ok := canonicalAdminPhone(targetPhone)
		if !ok || strings.TrimSpace(cmd.Name) == "" || (cmd.Role != RoleOwner && cmd.Role != RoleSubaccount) {
			tx.Rollback()
			return nil, ErrAdminInvalidInput
		}
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET phone=$1,name=$2,role=$3,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=$4 WHERE id=$5 AND deleted_at IS NULL`, []byte(phone), strings.TrimSpace(cmd.Name), cmd.Role, actor.id, cmd.AccountID)
		if e != nil {
			tx.Rollback()
			return nil, merchantSQLError(e)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
		result = &Account{ID: cmd.AccountID, Name: strings.TrimSpace(cmd.Name), MaskedPhone: maskMerchantPhone(phone), Role: cmd.Role, Enabled: target.enabled, Bound: target.bound}
	case SetAccountEnabled:
		if cmd.Enabled == nil {
			tx.Rollback()
			return nil, ErrAdminInvalidInput
		}
		target, found := find(cmd.AccountID)
		if !found {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
		if target.role == RoleOwner && target.enabled && !*cmd.Enabled && enabledOwners(target.id) == 0 {
			tx.Rollback()
			return nil, ErrLastOwner
		}
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET enabled=$1,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=$2 WHERE id=$3 AND deleted_at IS NULL`, *cmd.Enabled, actor.id, cmd.AccountID)
		if e != nil {
			tx.Rollback()
			return nil, merchantSQLError(e)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
		result = &Account{ID: cmd.AccountID, Name: target.name, MaskedPhone: maskMerchantPhone(target.phone), Role: target.role, Enabled: *cmd.Enabled, Bound: target.bound}
	case DeleteAccount:
		target, found := find(cmd.AccountID)
		if !found {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
		if target.role == RoleOwner && target.enabled && enabledOwners(target.id) == 0 {
			tx.Rollback()
			return nil, ErrLastOwner
		}
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET enabled=FALSE,bound_user_id=NULL,bound_at=NULL,deleted_at=clock_timestamp(),deleted_by_account_id=$1,record_version=record_version+1,auth_version=auth_version+1,updated_at=clock_timestamp(),updated_by=$2 WHERE id=$3 AND deleted_at IS NULL`, actor.id, actor.id, cmd.AccountID)
		if e != nil {
			tx.Rollback()
			return nil, merchantSQLError(e)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			tx.Rollback()
			return nil, ErrAdminNotFound
		}
	default:
		tx.Rollback()
		return nil, ErrAdminInvalidInput
	}
	receiptResult := struct {
		Account *Account `json:"account,omitempty"`
		Deleted bool     `json:"deleted,omitempty"`
	}{result, cmd.Kind == DeleteAccount}
	targetID := cmd.AccountID
	if result != nil {
		targetID = result.ID
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: actor.id, ActorRole: string(actor.role), ActorAuthVersion: actor.auth, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	action := string(cmd.Kind)
	err = a.receipts.AppendInTx(ctx, tx, rm, action, "MERCHANT_ACCOUNT", targetID, cmd, receiptResult)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay struct {
			Account *Account `json:"account"`
			Deleted bool     `json:"deleted"`
		}
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, actor.id, action, meta.IdempotencyKey, cmd, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return nil, ErrAdminIdempotencyConflict
		}
		if re != nil || !ok {
			return nil, ErrAdminUnavailable
		}
		return replay.Account, nil
	}
	if err != nil || tx.Commit() != nil {
		return nil, ErrAdminUnavailable
	}
	return result, nil
}

func (a *PostgreSQLAdminApplication) BeginPCLogin(ctx context.Context) (PCLogin, error) {
	approval, err := secret()
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	poll, err := secret()
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	approvalHash := sha256.Sum256([]byte(approval))
	pollHash := sha256.Sum256([]byte(poll))
	now := a.now().UTC().Truncate(time.Microsecond)
	expires := now.Add(2 * time.Minute)
	var id uint64
	err = a.db.QueryRowContext(ctx, `INSERT INTO merchant_pc_sessions(approval_secret_hash,poll_secret_hash,state,login_expires_at,created_at,updated_at) VALUES($1,$2,'WAITING',$3,$4,$5) RETURNING id`, approvalHash[:], pollHash[:], expires, now, now).Scan(&id)
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	loginID := fmt.Sprint(id)
	payload := "order-admin-login://approve?login_id=" + loginID + "&approval_secret=" + approval
	return PCLogin{LoginID: loginID, PollSecret: poll, QRPayload: payload, ExpiresAt: expires}, nil
}
func (a *PostgreSQLAdminApplication) ApprovePCLogin(ctx context.Context, userID uint64, loginID, approvalSecret, phoneCode string) error {
	id, err := parseAdminID(loginID)
	if err != nil {
		return ErrAdminInvalidInput
	}
	hash := sha256.Sum256([]byte(approvalSecret))
	var state string
	var expires time.Time
	if a.db.QueryRowContext(ctx, `SELECT state,login_expires_at FROM merchant_pc_sessions WHERE id=$1 AND approval_secret_hash=$2`, id, hash[:]).Scan(&state, &expires) != nil {
		return ErrPCLoginExpired
	}
	if state != "WAITING" || !a.now().Before(expires) {
		return ErrPCLoginExpired
	}
	identity, err := a.mini.Login(ctx, userID, phoneCode, "pc-approve-"+loginID)
	if err != nil {
		return err
	}
	if identity.Merchant == nil || identity.Merchant.Role != RoleOwner {
		return ErrForbidden
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrAdminUnavailable
	}
	var accountID, authVersion uint64
	if tx.QueryRowContext(ctx, `SELECT id,auth_version FROM merchant_accounts WHERE bound_user_id=$1 AND role='OWNER' AND enabled=TRUE AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID, &authVersion) != nil {
		tx.Rollback()
		return ErrForbidden
	}
	now := a.now().UTC().Truncate(time.Microsecond)
	res, err := tx.ExecContext(ctx, `UPDATE merchant_pc_sessions SET state='APPROVED',approved_account_id=$1,approved_user_id=$2,approved_auth_version=$3,approved_at=$4,updated_at=$5 WHERE id=$6 AND approval_secret_hash=$7 AND state='WAITING' AND login_expires_at>$8`, accountID, userID, authVersion, now, now, id, hash[:], now)
	if err != nil {
		tx.Rollback()
		return ErrAdminUnavailable
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		tx.Rollback()
		return ErrPCLoginExpired
	}
	if tx.Commit() != nil {
		return ErrAdminUnavailable
	}
	return nil
}
func (a *PostgreSQLAdminApplication) PollPCLogin(ctx context.Context, loginID, pollSecret string) (PCSession, error) {
	id, err := parseAdminID(loginID)
	if err != nil {
		return PCSession{}, ErrAdminInvalidInput
	}
	hash := sha256.Sum256([]byte(pollSecret))
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return PCSession{}, ErrAdminUnavailable
	}
	var state string
	var expires time.Time
	if tx.QueryRowContext(ctx, `SELECT state,login_expires_at FROM merchant_pc_sessions WHERE id=$1 AND poll_secret_hash=$2 FOR UPDATE`, id, hash[:]).Scan(&state, &expires) != nil {
		tx.Rollback()
		return PCSession{}, ErrPCLoginExpired
	}
	now := a.now().UTC().Truncate(time.Microsecond)
	if !now.Before(expires) {
		tx.Rollback()
		return PCSession{}, ErrPCLoginExpired
	}
	if state == "WAITING" {
		tx.Commit()
		return PCSession{State: "WAITING"}, nil
	}
	if state != "APPROVED" {
		tx.Rollback()
		return PCSession{}, ErrPCLoginExpired
	}
	token, err := secret()
	if err != nil {
		tx.Rollback()
		return PCSession{}, ErrAdminUnavailable
	}
	tokenHash := sha256.Sum256([]byte(token))
	accessExpires := now.Add(12 * time.Hour)
	res, err := tx.ExecContext(ctx, `UPDATE merchant_pc_sessions SET state='CONSUMED',consumed_at=$1,access_token_hash=$2,access_issued_at=$3,access_expires_at=$4,updated_at=$5 WHERE id=$6 AND state='APPROVED'`, now, tokenHash[:], now, accessExpires, now, id)
	if err != nil {
		tx.Rollback()
		return PCSession{}, ErrAdminUnavailable
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		tx.Rollback()
		return PCSession{}, ErrPCLoginExpired
	}
	if tx.Commit() != nil {
		return PCSession{}, ErrAdminUnavailable
	}
	return PCSession{State: "APPROVED", Token: token, ExpiresAt: accessExpires}, nil
}
func (a *PostgreSQLAdminApplication) AuthenticatePC(ctx context.Context, token string) (uint64, error) {
	hash := sha256.Sum256([]byte(token))
	var userID uint64
	err := a.db.QueryRowContext(ctx, `SELECT s.approved_user_id FROM merchant_pc_sessions s JOIN merchant_accounts a ON a.id=s.approved_account_id AND a.bound_user_id=s.approved_user_id AND a.auth_version=s.approved_auth_version WHERE s.access_token_hash=$1 AND s.state='CONSUMED' AND s.access_expires_at>$2 AND a.enabled=TRUE AND a.deleted_at IS NULL AND a.role='OWNER'`, hash[:], a.now().UTC()).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrPCSessionExpired
	}
	if err != nil || userID == 0 {
		return 0, ErrAdminUnavailable
	}
	return userID, nil
}

// RevokePC deletes only the presented access-token digest; repeated logout is safe.
func (a *PostgreSQLAdminApplication) RevokePC(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	if _, err := a.db.ExecContext(ctx, `DELETE FROM merchant_pc_sessions WHERE access_token_hash=$1`, hash[:]); err != nil {
		return ErrAdminUnavailable
	}
	return nil
}

func secret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func parseAdminID(v string) (uint64, error) {
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrAdminInvalidInput
	}
	return id, nil
}
func canonicalAdminPhone(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if len(v) == 11 && v[0] == '1' {
		v = "+86" + v
	}
	return v, canonicalPhone(v)
}
func maskMerchantPhone(phone string) string {
	digits := strings.TrimPrefix(phone, "+86")
	if len(digits) < 7 {
		return "***"
	}
	return digits[:3] + "****" + digits[len(digits)-4:]
}
func merchantSQLError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && (pgError.Code == "23505" || pgError.Code == "23503" || pgError.Code == "23514") {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
