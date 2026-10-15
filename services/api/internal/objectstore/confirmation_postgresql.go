package objectstore

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/audit"
)

const imageConfirmationAction = "object.image.confirm"
const imageConfirmationTarget = "IMAGE_OBJECT"

type PostgreSQLConfirmationStore struct {
	db       *sql.DB
	receipts *audit.ReceiptStore
}

func NewPostgreSQLConfirmationStore(db *sql.DB) *PostgreSQLConfirmationStore {
	return &PostgreSQLConfirmationStore{db: db, receipts: audit.NewReceiptStore(db)}
}

type imageConfirmationRequest struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	ObjectKey   string `json:"object_key"`
}

type imageConfirmationResponse struct {
	ObjectKey string `json:"object_key"`
}

func (store *PostgreSQLConfirmationStore) Record(ctx context.Context, meta WriteMeta, specification UploadSpecification, key string) error {
	if store == nil || store.db == nil || store.receipts == nil {
		return ErrUnavailable
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	accountID, role, authVersion, err := audit.LockOwner(ctx, tx, meta.ActorUserID)
	if err != nil {
		tx.Rollback()
		return ErrUnavailable
	}
	request := imageConfirmationRequest{specification.ContentType, specification.SizeBytes, specification.SHA256, key}
	response := imageConfirmationResponse{ObjectKey: key}
	receiptMeta := audit.CommandMeta{ActorUserID: meta.ActorUserID, ActorAccountID: accountID, ActorRole: role, ActorAuthVersion: authVersion, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
	err = store.receipts.AppendKeyInTx(ctx, tx, receiptMeta, imageConfirmationAction, imageConfirmationTarget, key, request, response)
	if errors.Is(err, audit.ErrDuplicateReceipt) {
		tx.Rollback()
		var replay imageConfirmationResponse
		ok, replayErr := store.receipts.Replay(ctx, meta.ActorUserID, accountID, imageConfirmationAction, meta.IdempotencyKey, request, &replay)
		if errors.Is(replayErr, audit.ErrIdempotencyConflict) {
			return ErrIdempotencyConflict
		}
		if replayErr != nil || !ok || replay.ObjectKey != key {
			return ErrUnavailable
		}
		return nil
	}
	if err != nil || tx.Commit() != nil {
		return ErrUnavailable
	}
	return nil
}

func (store *PostgreSQLConfirmationStore) Confirmed(ctx context.Context, keys []string) (map[string]bool, error) {
	if store == nil || store.db == nil || len(keys) == 0 || len(keys) > 3 {
		return nil, ErrUnavailable
	}
	confirmed := make(map[string]bool, len(keys))
	keyByHash := make(map[[32]byte]string, len(keys))
	placeholders := make([]string, 0, len(keys))
	arguments := []any{imageConfirmationAction, imageConfirmationTarget}
	for _, key := range keys {
		if !ValidImageObjectKey(key) {
			return nil, ErrUnavailable
		}
		hash := audit.TargetKeyHash(key)
		keyByHash[hash] = key
		placeholders = append(placeholders, "$"+strconv.Itoa(len(arguments)+1))
		arguments = append(arguments, hash[:])
	}
	query := `SELECT DISTINCT target_key_hash FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action=$1 AND target_type=$2 AND target_key_hash IN (` + strings.Join(placeholders, ",") + `) LIMIT 3`
	rows, err := store.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if rows.Scan(&raw) != nil || len(raw) != 32 {
			return nil, ErrUnavailable
		}
		var hash [32]byte
		copy(hash[:], raw)
		if key, ok := keyByHash[hash]; ok {
			confirmed[key] = true
		}
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return confirmed, nil
}
