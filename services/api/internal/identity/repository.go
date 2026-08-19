package identity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"
)

var errPersistence = errors.New("identity persistence failed")

// Repository stores Mini Program users and hash-only sessions in MySQL.
type Repository struct {
	db           *sql.DB
	beforeCommit func(*sql.Tx)
}

// NewRepository constructs the runtime identity repository.
func NewRepository(db *sql.DB) *Repository {
	return newRepository(db, nil)
}

func newRepository(db *sql.DB, beforeCommit func(*sql.Tx)) *Repository {
	return &Repository{db: db, beforeCommit: beforeCommit}
}

// CreateSession atomically resolves the user and inserts one independent session.
func (repository *Repository) CreateSession(ctx context.Context, params CreateSessionParams) (result error) {
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return errPersistence
	}
	defer func() {
		if result != nil {
			_ = transaction.Rollback()
		}
	}()

	userResult, err := transaction.ExecContext(ctx, `
		INSERT INTO miniprogram_users(openid,created_at,last_login_at)
		VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id),last_login_at=?
	`, params.OpenID, params.IssuedAt, params.IssuedAt, params.IssuedAt)
	if err != nil {
		return errPersistence
	}
	userID, err := userResult.LastInsertId()
	if err != nil || userID <= 0 {
		return errPersistence
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO miniprogram_sessions(token_hash,user_id,issued_at,expires_at)
		VALUES (?,?,?,?)
	`, params.TokenHash[:], userID, params.IssuedAt, params.ExpiresAt); err != nil {
		return errPersistence
	}
	if repository.beforeCommit != nil {
		repository.beforeCommit(transaction)
	}
	if err := transaction.Commit(); err != nil {
		return errPersistence
	}
	return nil
}

// FindActiveUser resolves a session only inside its inclusive-issued/exclusive-expiry interval.
func (repository *Repository) FindActiveUser(ctx context.Context, tokenHash [sha256.Size]byte, at time.Time) (uint64, error) {
	var userID uint64
	err := repository.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM miniprogram_sessions
		WHERE token_hash=? AND issued_at<=? AND expires_at>?
	`, tokenHash[:], at, at).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrSessionNotFound
	}
	if err != nil || userID == 0 {
		return 0, errPersistence
	}
	return userID, nil
}
