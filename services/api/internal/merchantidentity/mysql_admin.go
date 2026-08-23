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
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type MySQLAdminApplication struct {
	db       *sql.DB
	mini     Application
	receipts *audit.ReceiptStore
	now      func() time.Time
}

func NewMySQLAdminApplication(db *sql.DB, mini Application) *MySQLAdminApplication {
	return &MySQLAdminApplication{db: db, mini: mini, receipts: audit.NewReceiptStore(db), now: time.Now}
}
func (a *MySQLAdminApplication) CurrentAccount(ctx context.Context, userID uint64) (Account, error) {
	return a.readAccount(ctx, `SELECT id,name,CONVERT(phone USING ascii),role,enabled,(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID)
}

type merchantScanner interface{ Scan(...any) error }

func (a *MySQLAdminApplication) readAccount(ctx context.Context, query string, args ...any) (Account, error) {
	return a.scanAccount(a.db.QueryRowContext(ctx, query, args...))
}
func (a *MySQLAdminApplication) scanAccount(row merchantScanner) (Account, error) {
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
func (a *MySQLAdminApplication) ListAccounts(ctx context.Context, userID uint64, q string) ([]Account, error) {
	var owner uint64
	if a.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&owner) != nil {
		return nil, ErrAdminUnavailable
	}
	rows, err := a.db.QueryContext(ctx, `SELECT id,name,CONVERT(phone USING ascii),role,enabled,(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE deleted_at IS NULL AND (?='' OR name LIKE CONCAT('%',?,'%') OR CONVERT(phone USING ascii) LIKE CONCAT('%',?,'%')) ORDER BY id`, q, q, q)
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
func (a *MySQLAdminApplication) ExecuteAccount(ctx context.Context, meta AdminWriteMeta, cmd AccountCommand) (*Account, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrAdminUnavailable
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,bound_user_id,role,enabled,auth_version,name,CONVERT(phone USING ascii),(bound_user_id IS NOT NULL) FROM merchant_accounts WHERE deleted_at IS NULL ORDER BY id FOR UPDATE`)
	if err != nil {
		tx.Rollback()
		return nil, ErrAdminUnavailable
	}
	type locked struct {
		id      uint64
		user    sql.NullInt64
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
		if item.user.Valid && uint64(item.user.Int64) == meta.ActorUserID {
			actor = item
		}
	}
	rows.Close()
	if actor.id == 0 || actor.role != RoleOwner || !actor.enabled {
		tx.Rollback()
		return nil, ErrAdminUnavailable
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
		res, e := tx.ExecContext(ctx, `INSERT INTO merchant_accounts(phone,name,role,enabled,record_version,auth_version,bound_user_id,bound_at,created_at,updated_at,created_by,updated_by,deleted_at,deleted_by_account_id) VALUES(?,?,?,TRUE,1,1,NULL,NULL,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?,?,NULL,NULL)`, phone, strings.TrimSpace(cmd.Name), cmd.Role, actor.id, actor.id)
		if e != nil {
			tx.Rollback()
			return nil, merchantSQLError(e)
		}
		id, _ := res.LastInsertId()
		result = &Account{ID: uint64(id), Name: strings.TrimSpace(cmd.Name), MaskedPhone: maskMerchantPhone(phone), Role: cmd.Role, Enabled: true}
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
		if cmd.Phone == "" {
			cmd.Phone = target.phone
		}
		phone, ok := canonicalAdminPhone(cmd.Phone)
		if !ok || strings.TrimSpace(cmd.Name) == "" || (cmd.Role != RoleOwner && cmd.Role != RoleSubaccount) {
			tx.Rollback()
			return nil, ErrAdminInvalidInput
		}
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET phone=?,name=?,role=?,record_version=record_version+1,auth_version=auth_version+1,updated_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=? AND deleted_at IS NULL`, phone, strings.TrimSpace(cmd.Name), cmd.Role, actor.id, cmd.AccountID)
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
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET enabled=?,record_version=record_version+1,auth_version=auth_version+1,updated_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=? AND deleted_at IS NULL`, *cmd.Enabled, actor.id, cmd.AccountID)
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
		res, e := tx.ExecContext(ctx, `UPDATE merchant_accounts SET enabled=FALSE,bound_user_id=NULL,bound_at=NULL,deleted_at=UTC_TIMESTAMP(6),deleted_by_account_id=?,record_version=record_version+1,auth_version=auth_version+1,updated_at=UTC_TIMESTAMP(6),updated_by=? WHERE id=? AND deleted_at IS NULL`, actor.id, actor.id, cmd.AccountID)
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

func (a *MySQLAdminApplication) BeginPCLogin(ctx context.Context) (PCLogin, error) {
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
	res, err := a.db.ExecContext(ctx, `INSERT INTO merchant_pc_sessions(approval_secret_hash,poll_secret_hash,state,login_expires_at,created_at,updated_at) VALUES(?,?,'WAITING',?,?,?)`, approvalHash[:], pollHash[:], expires, now, now)
	if err != nil {
		return PCLogin{}, ErrAdminUnavailable
	}
	id, _ := res.LastInsertId()
	loginID := fmt.Sprint(id)
	payload := "order-admin-login://approve?login_id=" + loginID + "&approval_secret=" + approval
	return PCLogin{LoginID: loginID, PollSecret: poll, QRPayload: payload, ExpiresAt: expires}, nil
}
func (a *MySQLAdminApplication) ApprovePCLogin(ctx context.Context, userID uint64, loginID, approvalSecret, phoneCode string) error {
	id, err := parseAdminID(loginID)
	if err != nil {
		return ErrAdminInvalidInput
	}
	hash := sha256.Sum256([]byte(approvalSecret))
	var state string
	var expires time.Time
	if a.db.QueryRowContext(ctx, `SELECT state,login_expires_at FROM merchant_pc_sessions WHERE id=? AND approval_secret_hash=?`, id, hash[:]).Scan(&state, &expires) != nil {
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
	if tx.QueryRowContext(ctx, `SELECT id,auth_version FROM merchant_accounts WHERE bound_user_id=? AND role='OWNER' AND enabled=TRUE AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&accountID, &authVersion) != nil {
		tx.Rollback()
		return ErrForbidden
	}
	now := a.now().UTC().Truncate(time.Microsecond)
	res, err := tx.ExecContext(ctx, `UPDATE merchant_pc_sessions SET state='APPROVED',approved_account_id=?,approved_user_id=?,approved_auth_version=?,approved_at=?,updated_at=? WHERE id=? AND approval_secret_hash=? AND state='WAITING' AND login_expires_at>=?`, accountID, userID, authVersion, now, now, id, hash[:], now)
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
func (a *MySQLAdminApplication) PollPCLogin(ctx context.Context, loginID, pollSecret string) (PCSession, error) {
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
	if tx.QueryRowContext(ctx, `SELECT state,login_expires_at FROM merchant_pc_sessions WHERE id=? AND poll_secret_hash=? FOR UPDATE`, id, hash[:]).Scan(&state, &expires) != nil {
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
	res, err := tx.ExecContext(ctx, `UPDATE merchant_pc_sessions SET state='CONSUMED',consumed_at=?,access_token_hash=?,access_issued_at=?,access_expires_at=?,updated_at=? WHERE id=? AND state='APPROVED'`, now, tokenHash[:], now, accessExpires, now, id)
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
func (a *MySQLAdminApplication) AuthenticatePC(ctx context.Context, token string) (uint64, error) {
	hash := sha256.Sum256([]byte(token))
	var userID uint64
	err := a.db.QueryRowContext(ctx, `SELECT s.approved_user_id FROM merchant_pc_sessions s JOIN merchant_accounts a ON a.id=s.approved_account_id AND a.bound_user_id=s.approved_user_id AND a.auth_version=s.approved_auth_version WHERE s.access_token_hash=? AND s.state='CONSUMED' AND s.access_expires_at>UTC_TIMESTAMP(6) AND a.enabled=TRUE AND a.deleted_at IS NULL AND a.role='OWNER'`, hash[:]).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrPCSessionExpired
	}
	if err != nil || userID == 0 {
		return 0, ErrAdminUnavailable
	}
	return userID, nil
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
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) && (me.Number == 1062 || me.Number == 1451 || me.Number == 1452 || me.Number == 3819) {
		return ErrAdminConflict
	}
	return ErrAdminUnavailable
}
