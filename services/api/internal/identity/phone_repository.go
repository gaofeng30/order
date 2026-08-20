package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
)

// FindPhoneUser reads only the provider identity and current immutable binding.
func (repository *Repository) FindPhoneUser(ctx context.Context, userID uint64) (PhoneUser, error) {
	var openID string
	var phone sql.NullString
	err := repository.db.QueryRowContext(ctx, `
		SELECT openid,primary_phone
		FROM miniprogram_users
		WHERE id=?
	`, userID).Scan(&openID, &phone)
	if err != nil || userID == 0 || openID == "" {
		return PhoneUser{}, errPersistence
	}
	return PhoneUser{OpenID: openID, PrimaryPhoneBound: phone.Valid, PrimaryPhone: phone.String}, nil
}

// BindPrimaryPhone locks one user and writes only its first canonical phone.
func (repository *Repository) BindPrimaryPhone(ctx context.Context, userID uint64, phone string, boundAt time.Time) (boundPhone string, result error) {
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return "", errPersistence
	}
	defer func() {
		if result != nil {
			_ = transaction.Rollback()
		}
	}()

	var current sql.NullString
	if err := transaction.QueryRowContext(ctx, `
		SELECT primary_phone
		FROM miniprogram_users
		WHERE id=?
		FOR UPDATE
	`, userID).Scan(&current); err != nil {
		return "", errPersistence
	}
	if current.Valid {
		if _, err := maskedBinding(current.String); err != nil {
			return "", errPersistence
		}
		if current.String != phone {
			return "", ErrPrimaryPhoneAlreadyBound
		}
	}
	if !current.Valid {
		update, err := transaction.ExecContext(ctx, `
			UPDATE miniprogram_users
			SET primary_phone=?,primary_phone_bound_at=?
			WHERE id=? AND primary_phone IS NULL AND primary_phone_bound_at IS NULL
		`, phone, boundAt, userID)
		if isDuplicatePhone(err) {
			return "", ErrPhoneInUse
		}
		if err != nil {
			return "", errPersistence
		}
		rows, err := update.RowsAffected()
		if err != nil || rows != 1 {
			return "", errPersistence
		}
	}
	if repository.beforeCommit != nil {
		repository.beforeCommit(transaction)
	}
	if err := transaction.Commit(); err != nil {
		return "", errPersistence
	}
	return phone, nil
}

func isDuplicatePhone(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}
