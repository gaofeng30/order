package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/gaofeng30/order/services/api/internal/quote"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const maxQuoteReceiptResponseBytes = 4096

type QuoteReceiptStore struct {
	db  *sql.DB
	now func() time.Time
}

var _ quote.OperationReceiptStore = (*QuoteReceiptStore)(nil)

func NewQuoteReceiptStore(db *sql.DB) *QuoteReceiptStore {
	return &QuoteReceiptStore{db: db, now: time.Now}
}

func (store *QuoteReceiptStore) ReplayInTx(ctx context.Context, transaction *sql.Tx, meta quote.WriteMeta, action quote.ReceiptAction) (quote.OperationReceipt, bool, error) {
	if store == nil || store.db == nil || transaction == nil || ctx == nil || !validQuoteReceiptIdentity(meta, action) {
		return quote.OperationReceipt{}, false, quote.ErrUnavailable
	}
	scopeHash := quoteUserScopeHash(meta.ActorUserID)
	operationHash := quoteOperationKeyHash(meta.IdempotencyKey)
	var evidence, response []byte
	err := transaction.QueryRowContext(ctx, `SELECT before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER' AND actor_scope_hash=? AND action=? AND operation_key_hash=? LIMIT 1`, scopeHash[:], string(action), operationHash[:]).Scan(&evidence, &response)
	if errors.Is(err, sql.ErrNoRows) {
		return quote.OperationReceipt{}, false, nil
	}
	if err != nil {
		return quote.OperationReceipt{}, false, err
	}
	requestDigest, err := decodeQuoteReceiptEvidence(evidence)
	if err != nil || len(response) == 0 || len(response) > maxQuoteReceiptResponseBytes || !json.Valid(response) {
		return quote.OperationReceipt{}, false, quote.ErrSnapshotInvalid
	}
	return quote.OperationReceipt{RequestDigest: requestDigest, ResponseJSON: append([]byte(nil), response...)}, true, nil
}

func (store *QuoteReceiptStore) AppendInTx(ctx context.Context, transaction *sql.Tx, meta quote.WriteMeta, action quote.ReceiptAction, receipt quote.OperationReceipt) error {
	if store == nil || store.db == nil || store.now == nil || transaction == nil || ctx == nil || !validQuoteReceiptIdentity(meta, action) || len(receipt.ResponseJSON) == 0 || len(receipt.ResponseJSON) > maxQuoteReceiptResponseBytes || !json.Valid(receipt.ResponseJSON) {
		return quote.ErrUnavailable
	}
	evidence, err := encodeQuoteReceiptEvidence(receipt.RequestDigest)
	if err != nil {
		return quote.ErrUnavailable
	}
	scopeHash := quoteUserScopeHash(meta.ActorUserID)
	operationHash := quoteOperationKeyHash(meta.IdempotencyKey)
	requestIDHash := quoteRequestIDHash(meta.RequestID)
	_, err = transaction.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','USER',?,?,?,?,?,'SUCCEEDED','QUOTE_CREATED',?,?,?)`, scopeHash[:], meta.ActorUserID, string(action), operationHash[:], requestIDHash[:], evidence, receipt.ResponseJSON, store.now().UTC().Truncate(time.Microsecond))
	var mysqlError *mysqlDriver.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return quote.ErrOperationReceiptExists
	}
	return err
}

type quoteReceiptEvidence struct {
	RequestDigest string `json:"request_digest"`
}

func encodeQuoteReceiptEvidence(digest [sha256.Size]byte) ([]byte, error) {
	return json.Marshal(quoteReceiptEvidence{RequestDigest: hex.EncodeToString(digest[:])})
}

func decodeQuoteReceiptEvidence(encoded []byte) ([sha256.Size]byte, error) {
	if len(encoded) == 0 || len(encoded) > 256 || !json.Valid(encoded) {
		return [sha256.Size]byte{}, quote.ErrSnapshotInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var evidence quoteReceiptEvidence
	if err := decoder.Decode(&evidence); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return [sha256.Size]byte{}, quote.ErrSnapshotInvalid
	}
	decoded, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != evidence.RequestDigest {
		return [sha256.Size]byte{}, quote.ErrSnapshotInvalid
	}
	var digest [sha256.Size]byte
	copy(digest[:], decoded)
	return digest, nil
}

func quoteUserScopeHash(userID uint64) [sha256.Size]byte {
	var material [13]byte
	copy(material[:5], "USER\x00")
	binary.BigEndian.PutUint64(material[5:], userID)
	return sha256.Sum256(material[:])
}

func quoteOperationKeyHash(value string) [sha256.Size]byte { return sha256.Sum256([]byte(value)) }
func quoteRequestIDHash(value string) [sha256.Size]byte    { return sha256.Sum256([]byte(value)) }

func validQuoteReceiptIdentity(meta quote.WriteMeta, action quote.ReceiptAction) bool {
	return meta.ActorUserID > 0 && meta.IdempotencyKey != "" && len(meta.IdempotencyKey) <= 128 && meta.RequestID != "" && len(meta.RequestID) <= 64 && action == quote.ReceiptActionQuoteCreate
}
