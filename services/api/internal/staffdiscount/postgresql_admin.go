package staffdiscount

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/unicode/norm"
)

type PostgreSQLApplication struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
}

func NewPostgreSQLApplication(db *sql.DB) *PostgreSQLApplication {
	return &PostgreSQLApplication{db: db, receipts: audit.NewReceiptStore(db)}
}
func (a *PostgreSQLApplication) List(ctx context.Context, q string) ([]Staff, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT s.id,convert_from(s.phone,'UTF8'),s.name,s.enabled,s.created_at,EXISTS(SELECT 1 FROM miniprogram_users u WHERE u.primary_phone=s.phone OR u.extra_phone=s.phone),COALESCE((SELECT SUM(o.payable_cents) FROM orders o JOIN miniprogram_users u ON u.id=o.user_id WHERE (u.primary_phone=s.phone OR u.extra_phone=s.phone) AND o.identity_kind='STAFF' AND o.state<>'REFUNDED'),0),COALESCE((SELECT COUNT(*) FROM orders o JOIN miniprogram_users u ON u.id=o.user_id WHERE (u.primary_phone=s.phone OR u.extra_phone=s.phone) AND o.identity_kind='STAFF' AND o.state<>'REFUNDED'),0) FROM staff_whitelist s WHERE ($1='' OR s.name LIKE '%'||$2||'%' OR convert_from(s.phone,'UTF8') LIKE '%'||$3||'%') ORDER BY s.id DESC LIMIT 5000`, q, q, q)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	out := []Staff{}
	for rows.Next() {
		var s Staff
		var phone string
		if rows.Scan(&s.ID, &phone, &s.Name, &s.Enabled, &s.CreatedAt, &s.Bound, &s.SpendCents, &s.OrderCount) != nil {
			return nil, ErrUnavailable
		}
		s.MaskedPhone = maskPhone(phone)
		out = append(out, s)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return out, nil
}
func (a *PostgreSQLApplication) DiscountRate(ctx context.Context) (uint8, error) {
	var rate uint8
	if a.db.QueryRowContext(ctx, `SELECT rate_percent FROM discount_settings WHERE id=1`).Scan(&rate) != nil {
		return 0, ErrUnavailable
	}
	if rate < 1 || rate > 100 {
		return 0, ErrUnavailable
	}
	return rate, nil
}
func (a *PostgreSQLApplication) Execute(ctx context.Context, meta WriteMeta, cmd Command) (Result, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, ErrUnavailable
	}
	defer tx.Rollback()
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return Result{}, ErrUnavailable
	}
	var settingsID uint8
	if tx.QueryRowContext(ctx, `SELECT id FROM discount_settings WHERE id=1 FOR UPDATE`).Scan(&settingsID) != nil {
		tx.Rollback()
		return Result{}, ErrUnavailable
	}
	var replay Result
	found, replayErr := a.receipts.ReplayInTx(ctx, tx, meta.ActorUserID, accountID, string(cmd.Kind), meta.IdempotencyKey, cmd, &replay)
	if errors.Is(replayErr, audit.ErrIdempotencyConflict) {
		return Result{}, ErrIdempotencyConflict
	}
	if replayErr != nil {
		return Result{}, ErrUnavailable
	}
	if found {
		return replay, nil
	}
	result, err := a.executeTx(ctx, tx, cmd)
	if err != nil {
		tx.Rollback()
		return Result{}, err
	}
	action := string(cmd.Kind)
	target := "DISCOUNT_SETTINGS"
	targetID := uint64(1)
	if cmd.StaffID > 0 {
		target, targetID = "STAFF", cmd.StaffID
	}
	rm := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = a.receipts.AppendInTx(ctx, tx, rm, action, target, targetID, cmd, result)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay Result
		ok, re := a.receipts.Replay(ctx, meta.ActorUserID, accountID, action, meta.IdempotencyKey, cmd, &replay)
		if errors.Is(re, audit.ErrIdempotencyConflict) {
			return Result{}, ErrIdempotencyConflict
		}
		if re != nil || !ok {
			return Result{}, ErrUnavailable
		}
		return replay, nil
	}
	if err != nil || tx.Commit() != nil {
		return Result{}, ErrUnavailable
	}
	return result, nil
}
func (a *PostgreSQLApplication) executeTx(ctx context.Context, tx *sql.Tx, cmd Command) (Result, error) {
	switch cmd.Kind {
	case SetDiscountRate:
		if cmd.RatePercent < 1 || cmd.RatePercent > 100 {
			return Result{}, ErrInvalidInput
		}
		_, err := tx.ExecContext(ctx, `UPDATE discount_settings SET rate_percent=$1,discount_version=discount_version+1,updated_at=clock_timestamp() WHERE id=1`, cmd.RatePercent)
		if err != nil {
			return Result{}, staffSQLError(err)
		}
		return Result{RatePercent: cmd.RatePercent}, nil
	case CreateStaff:
		name, key, phone, ok := staffInput(cmd.Name, cmd.Phone)
		if !ok {
			return Result{}, ErrInvalidInput
		}
		var id uint64
		err := tx.QueryRowContext(ctx, `INSERT INTO staff_whitelist(phone,name,name_key,enabled,record_version,created_at,updated_at) VALUES($1,$2,$3,TRUE,1,clock_timestamp(),clock_timestamp()) RETURNING id`, []byte(phone), name, key).Scan(&id)
		if err != nil {
			return Result{}, staffSQLError(err)
		}
		if _, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1`); err != nil {
			return Result{}, ErrUnavailable
		}
		return Result{Staff: &Staff{ID: id, Name: name, MaskedPhone: maskPhone(phone), Enabled: true}}, nil
	case UpdateStaff:
		var existing string
		var enabled bool
		if tx.QueryRowContext(ctx, `SELECT convert_from(phone,'UTF8'),enabled FROM staff_whitelist WHERE id=$1 FOR UPDATE`, cmd.StaffID).Scan(&existing, &enabled) != nil {
			return Result{}, ErrNotFound
		}
		if cmd.Phone == "" {
			cmd.Phone = existing
		}
		name, key, phone, ok := staffInput(cmd.Name, cmd.Phone)
		if !ok {
			return Result{}, ErrInvalidInput
		}
		_, err := tx.ExecContext(ctx, `UPDATE staff_whitelist SET phone=$1,name=$2,name_key=$3,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=$4`, []byte(phone), name, key, cmd.StaffID)
		if err != nil {
			return Result{}, staffSQLError(err)
		}
		if _, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1`); err != nil {
			return Result{}, ErrUnavailable
		}
		return Result{Staff: &Staff{ID: cmd.StaffID, Name: name, MaskedPhone: maskPhone(phone), Enabled: enabled}}, nil
	case SetStaffEnabled:
		if cmd.Enabled == nil {
			return Result{}, ErrInvalidInput
		}
		var name, phone string
		if tx.QueryRowContext(ctx, `SELECT name,convert_from(phone,'UTF8') FROM staff_whitelist WHERE id=$1 FOR UPDATE`, cmd.StaffID).Scan(&name, &phone) != nil {
			return Result{}, ErrNotFound
		}
		_, err := tx.ExecContext(ctx, `UPDATE staff_whitelist SET enabled=$1,record_version=record_version+1,updated_at=clock_timestamp() WHERE id=$2`, *cmd.Enabled, cmd.StaffID)
		if err != nil {
			return Result{}, staffSQLError(err)
		}
		if _, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1`); err != nil {
			return Result{}, ErrUnavailable
		}
		return Result{Staff: &Staff{ID: cmd.StaffID, Name: name, MaskedPhone: maskPhone(phone), Enabled: *cmd.Enabled}}, nil
	case DeleteStaff:
		res, err := tx.ExecContext(ctx, `DELETE FROM staff_whitelist WHERE id=$1`, cmd.StaffID)
		if err != nil {
			return Result{}, staffSQLError(err)
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return Result{}, ErrNotFound
		}
		if _, err = tx.ExecContext(ctx, `UPDATE discount_settings SET whitelist_version=whitelist_version+1,updated_at=clock_timestamp() WHERE id=1`); err != nil {
			return Result{}, ErrUnavailable
		}
		return Result{}, nil
	default:
		return Result{}, ErrInvalidInput
	}
}
func staffInput(name, phone string) (string, []byte, string, bool) {
	name = strings.TrimSpace(norm.NFKC.String(name))
	keyText := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name)
	phone = strings.TrimSpace(phone)
	if len(phone) == 11 && phone[0] == '1' {
		phone = "+86" + phone
	}
	return name, []byte(keyText), phone, name != "" && keyText != "" && len([]byte(keyText)) <= 400 && strings.HasPrefix(phone, "+") && len(phone) <= 16
}
func maskPhone(phone string) string {
	digits := strings.TrimPrefix(phone, "+86")
	if len(digits) < 7 {
		return "***"
	}
	return digits[:3] + "****" + digits[len(digits)-4:]
}
func staffSQLError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && (pgError.Code == "23505" || pgError.Code == "23503" || pgError.Code == "23514") {
		return ErrConflict
	}
	return ErrUnavailable
}
