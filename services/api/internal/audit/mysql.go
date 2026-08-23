package audit

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

var (
	ErrDuplicateReceipt    = errors.New("duplicate command receipt")
	ErrIdempotencyConflict = errors.New("command receipt request conflict")
)

type CommandMeta struct {
	ActorUserID, ActorAccountID uint64
	ActorRole                   string
	ActorAuthVersion            uint64
	IdempotencyKey, RequestID   string
}
type ReceiptStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewReceiptStore(db *sql.DB) *ReceiptStore { return &ReceiptStore{db: db, now: time.Now} }
func scopeHash(userID, accountID uint64) [32]byte {
	return sha256.Sum256([]byte("merchant:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(accountID, 10)))
}
func keyHash(prefix, value string) [32]byte { return sha256.Sum256([]byte(prefix + ":" + value)) }

type requestEvidence struct {
	RequestDigest string `json:"request_digest"`
}

func digestRequest(request any) ([32]byte, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return [32]byte{}, fmt.Errorf("marshal command request: %w", err)
	}
	return sha256.Sum256(raw), nil
}
func encodeRequestEvidence(request any) ([]byte, error) {
	digest, err := digestRequest(request)
	if err != nil {
		return nil, err
	}
	evidence, err := json.Marshal(requestEvidence{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		return nil, fmt.Errorf("marshal request evidence: %w", err)
	}
	return evidence, nil
}
func matchRequestEvidence(raw []byte, request any) error {
	digest, err := digestRequest(request)
	if err != nil {
		return err
	}
	var evidence requestEvidence
	if !json.Valid(raw) || json.Unmarshal(raw, &evidence) != nil {
		return ErrUnavailable
	}
	stored, decodeErr := hex.DecodeString(evidence.RequestDigest)
	if decodeErr != nil || len(stored) != len(digest) || subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
		return ErrIdempotencyConflict
	}
	return nil
}

func (s *ReceiptStore) Replay(ctx context.Context, userID, accountID uint64, action, key string, request, out any) (bool, error) {
	scope := scopeHash(userID, accountID)
	operation := keyHash("operation", key)
	var raw, evidenceRaw []byte
	err := s.db.QueryRowContext(ctx, `SELECT before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_scope_hash=? AND action=? AND operation_key_hash=? LIMIT 1`, scope[:], action, operation[:]).Scan(&evidenceRaw, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read command receipt: %w", err)
	}
	if err := matchRequestEvidence(evidenceRaw, request); err != nil {
		return false, err
	}
	if !json.Valid(raw) || json.Unmarshal(raw, out) != nil {
		return false, ErrUnavailable
	}
	return true, nil
}
func (s *ReceiptStore) AppendInTx(ctx context.Context, tx *sql.Tx, meta CommandMeta, action, targetType string, targetID uint64, request, response any) error {
	raw, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal command response: %w", err)
	}
	scope := scopeHash(meta.ActorUserID, meta.ActorAccountID)
	operation := keyHash("operation", meta.IdempotencyKey)
	evidence, err := encodeRequestEvidence(request)
	if err != nil {
		return err
	}
	var requestIDHash any
	if meta.RequestID != "" {
		sum := keyHash("request", meta.RequestID)
		requestIDHash = sum[:]
	}
	var target, targetTypeValue any
	if targetID > 0 {
		target = targetID
		targetTypeValue = targetType
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO action_audits(entry_kind,actor_kind,actor_scope_hash,actor_user_id,actor_account_id,actor_account_id_snapshot,actor_role_snapshot,actor_auth_version_snapshot,action,target_type,target_id,target_key_hash,operation_key_hash,request_id_hash,result,reason_code,before_state_json,after_state_json,response_json,occurred_at) VALUES('COMMAND_RECEIPT','MERCHANT',?,?,?,?,?,?,?, ?,?,NULL,?,?, 'SUCCEEDED','OK',?,NULL,?,?)`, scope[:], meta.ActorUserID, meta.ActorAccountID, meta.ActorAccountID, meta.ActorRole, meta.ActorAuthVersion, action, targetTypeValue, target, operation[:], requestIDHash, evidence, raw, s.now().UTC())
	if err != nil {
		var driverErr *mysqlDriver.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == 1062 {
			return ErrDuplicateReceipt
		}
		return fmt.Errorf("append command receipt: %w", err)
	}
	return nil
}
func LockOwner(ctx context.Context, tx *sql.Tx, userID uint64) (accountID uint64, role string, authVersion uint64, err error) {
	err = tx.QueryRowContext(ctx, `SELECT id,role,auth_version FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER' FOR UPDATE`, userID).Scan(&accountID, &role, &authVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", 0, ErrForbidden
	}
	if err != nil {
		return 0, "", 0, fmt.Errorf("lock owner account: %w", err)
	}
	return accountID, role, authVersion, nil
}
