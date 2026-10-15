package identity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"
)

var errPersistence = errors.New("identity persistence failed")

// Repository stores Mini Program users and hash-only sessions in PostgreSQL.
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

	var userID uint64
	err = transaction.QueryRowContext(ctx, `
		INSERT INTO miniprogram_users(openid,created_at,last_login_at)
		VALUES ($1,$2,$2)
		ON CONFLICT (openid) DO UPDATE SET last_login_at=EXCLUDED.last_login_at
		RETURNING id
	`, []byte(params.OpenID), params.IssuedAt).Scan(&userID)
	if err != nil || userID == 0 {
		return errPersistence
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO miniprogram_sessions(token_hash,user_id,issued_at,expires_at)
		VALUES ($1,$2,$3,$4)
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
		WHERE token_hash=$1 AND issued_at<=$2 AND expires_at>$2
	`, tokenHash[:], at).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrSessionNotFound
	}
	if err != nil || userID == 0 {
		return 0, errPersistence
	}
	return userID, nil
}

// RevokeSession removes only the presented hash. Zero affected rows is a successful replay.
func (repository *Repository) RevokeSession(ctx context.Context, tokenHash [sha256.Size]byte) error {
	if _, err := repository.db.ExecContext(ctx, `DELETE FROM miniprogram_sessions WHERE token_hash=$1`, tokenHash[:]); err != nil {
		return errPersistence
	}
	return nil
}
