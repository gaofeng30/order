package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// FindPhoneUser reads only the provider identity and current immutable binding.
func (repository *Repository) FindPhoneUser(ctx context.Context, userID uint64) (PhoneUser, error) {
	var openID string
	var phone sql.NullString
	err := repository.db.QueryRowContext(ctx, `
		SELECT openid,primary_phone
		FROM miniprogram_users
		WHERE id=$1
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
		WHERE id=$1
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
			SET primary_phone=$1,primary_phone_bound_at=$2
			WHERE id=$3 AND primary_phone IS NULL AND primary_phone_bound_at IS NULL
		`, []byte(phone), boundAt, userID)
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
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "uq_miniprogram_users_primary_phone"
}
