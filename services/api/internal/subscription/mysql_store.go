package subscription

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

type mysqlStore struct{ db *sql.DB }

func newMySQLStore(db *sql.DB) *mysqlStore { return &mysqlStore{db: db} }

func (store *mysqlStore) recordConsent(ctx context.Context, meta WriteMeta, input ConsentInput, now time.Time) (Subscription, error) {
	if store == nil || store.db == nil {
		return Subscription{}, ErrUnavailable
	}
	keyHash := sha256.Sum256([]byte(meta.IdempotencyKey))
	if replay, found, err := readConsent(ctx, store.db, meta.ActorUserID, keyHash); err != nil {
		return Subscription{}, ErrUnavailable
	} else if found {
		return matchReplay(replay, input)
	}

	for attempt := 0; attempt < 2; attempt++ {
		consent, err := store.recordConsentOnce(ctx, meta, input, keyHash, now)
		if err == nil {
			return consent, nil
		}
		if isDuplicate(err) {
			replay, found, readErr := readConsent(ctx, store.db, meta.ActorUserID, keyHash)
			if readErr != nil || !found {
				return Subscription{}, ErrUnavailable
			}
			return matchReplay(replay, input)
		}
		if !isRetryableMySQL(err) || attempt == 1 {
			return Subscription{}, mapSQLError(err)
		}
	}
	return Subscription{}, ErrUnavailable
}

func (store *mysqlStore) recordConsentOnce(ctx context.Context, meta WriteMeta, input ConsentInput, keyHash [32]byte, now time.Time) (Subscription, error) {
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Subscription{}, err
	}
	defer transaction.Rollback()

	var ownerUserID uint64
	if err := transaction.QueryRowContext(ctx, `
		SELECT user_id FROM orders WHERE id=? FOR UPDATE
	`, input.OrderID).Scan(&ownerUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, ErrNotFound
		}
		return Subscription{}, err
	}
	if ownerUserID != meta.ActorUserID {
		return Subscription{}, ErrNotFound
	}

	var latestSequence uint64
	err = transaction.QueryRowContext(ctx, `
		SELECT grant_sequence
		FROM notification_consents
		WHERE order_id=? AND kind=?
		ORDER BY grant_sequence DESC LIMIT 1 FOR UPDATE
	`, input.OrderID, input.Kind).Scan(&latestSequence)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, err
	}
	if latestSequence == math.MaxUint64 {
		return Subscription{}, ErrUnavailable
	}
	consent := Subscription{
		OrderID: input.OrderID, Kind: input.Kind, Decision: input.Decision,
		Available: input.Decision == DecisionAccepted, GrantSequence: latestSequence + 1,
		TemplateConfigVersion: input.TemplateConfigVersion, DecidedAt: now,
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO notification_consents(
			order_id,user_id,kind,grant_sequence,decision,template_config_version,
			idempotency_key_hash,decided_at,consumed_at
		) VALUES (?,?,?,?,?,?,?,?,NULL)
	`, consent.OrderID, meta.ActorUserID, consent.Kind, consent.GrantSequence, consent.Decision,
		consent.TemplateConfigVersion, keyHash[:], consent.DecidedAt); err != nil {
		return Subscription{}, err
	}
	if err := insertConsentReceipt(ctx, transaction, meta, consent, keyHash, now); err != nil {
		return Subscription{}, err
	}
	if err := transaction.Commit(); err != nil {
		return Subscription{}, err
	}
	return consent, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readConsent(ctx context.Context, queryer queryRower, userID uint64, keyHash [32]byte) (Subscription, bool, error) {
	var consent Subscription
	var kind string
	var decision string
	err := queryer.QueryRowContext(ctx, `
		SELECT order_id,kind,decision,grant_sequence,template_config_version,decided_at
		FROM notification_consents
		WHERE user_id=? AND idempotency_key_hash=?
	`, userID, keyHash[:]).Scan(&consent.OrderID, &kind, &decision, &consent.GrantSequence,
		&consent.TemplateConfigVersion, &consent.DecidedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, false, nil
	}
	if err != nil {
		return Subscription{}, false, err
	}
	consent.Kind = Kind(kind)
	consent.Decision = Decision(decision)
	consent.Available = consent.Decision == DecisionAccepted
	return consent, true, nil
}

func matchReplay(consent Subscription, input ConsentInput) (Subscription, error) {
	if consent.OrderID != input.OrderID || consent.Kind != input.Kind || consent.Decision != input.Decision || consent.TemplateConfigVersion != input.TemplateConfigVersion {
		return Subscription{}, ErrIdempotencyConflict
	}
	return consent, nil
}

func insertConsentReceipt(ctx context.Context, transaction *sql.Tx, meta WriteMeta, consent Subscription, keyHash [32]byte, now time.Time) error {
	response, err := json.Marshal(struct {
		Subscription struct {
			Kind      Kind     `json:"kind"`
			Decision  Decision `json:"decision"`
			Available bool     `json:"available"`
		} `json:"subscription"`
	}{Subscription: struct {
		Kind      Kind     `json:"kind"`
		Decision  Decision `json:"decision"`
		Available bool     `json:"available"`
	}{Kind: consent.Kind, Decision: consent.Decision, Available: consent.Available}})
	if err != nil {
		return err
	}
	requestHash := sha256.Sum256([]byte(meta.RequestID))
	scopeHash := userScopeHash(meta.ActorUserID)
	_, err = transaction.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,
			operation_key_hash,request_id_hash,result,reason_code,after_state_json,response_json,occurred_at
		) VALUES ('COMMAND_RECEIPT','USER',?,?,?,'ORDER',?,?,?,'SUCCEEDED','CONSENT_RECORDED',?,?,?)
	`, scopeHash[:], meta.ActorUserID, "subscription.record_consent", consent.OrderID,
		keyHash[:], requestHash[:], response, response, now)
	return err
}

func userScopeHash(userID uint64) [32]byte {
	var material [13]byte
	copy(material[:5], "USER\x00")
	binary.BigEndian.PutUint64(material[5:], userID)
	return sha256.Sum256(material[:])
}

func isDuplicate(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func isRetryableMySQL(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1205 || mysqlError.Number == 1213)
}

func mapSQLError(err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrIdempotencyConflict) || errors.Is(err, ErrUnavailable) {
		return err
	}
	return ErrUnavailable
}

func (*mysqlStore) enqueueInTx(ctx context.Context, transaction *sql.Tx, intent NotificationIntent, now time.Time) error {
	var ownerUserID uint64
	if err := transaction.QueryRowContext(ctx, `
		SELECT user_id FROM orders WHERE id=? FOR UPDATE
	`, intent.OrderID).Scan(&ownerUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return mapSQLError(err)
	}
	if ownerUserID != intent.RecipientUserID {
		return ErrForbidden
	}

	var consentID, consentUserID, templateVersion uint64
	var consumedAt sql.NullTime
	if err := transaction.QueryRowContext(ctx, `
		SELECT id,user_id,template_config_version,consumed_at
		FROM notification_consents
		WHERE order_id=? AND kind=? AND decision='ACCEPTED'
		ORDER BY grant_sequence
		LIMIT 1 FOR UPDATE
	`, intent.OrderID, intent.Kind).Scan(&consentID, &consentUserID, &templateVersion, &consumedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrForbidden
		}
		return mapSQLError(err)
	}
	if consentUserID != intent.RecipientUserID {
		return ErrForbidden
	}

	messageJSON, err := json.Marshal(intent.Message)
	if err != nil {
		return ErrInvalidInput
	}
	var existingConsentID, existingRecipientID, existingTemplateVersion uint64
	var existingMessage []byte
	err = transaction.QueryRowContext(ctx, `
		SELECT consent_id,recipient_user_id,template_config_version,immutable_message_json
		FROM notification_outbox
		WHERE order_id=? AND kind=? FOR UPDATE
	`, intent.OrderID, intent.Kind).Scan(&existingConsentID, &existingRecipientID, &existingTemplateVersion, &existingMessage)
	if err == nil {
		if existingConsentID == consentID && existingRecipientID == intent.RecipientUserID && existingTemplateVersion == templateVersion && jsonEqual(existingMessage, messageJSON) {
			return nil
		}
		return ErrIdempotencyConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return mapSQLError(err)
	}
	if consumedAt.Valid {
		return ErrUnavailable
	}

	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO notification_outbox(
			order_id,consent_id,kind,recipient_user_id,immutable_message_json,template_config_version,
			state,attempt_count,next_attempt_at,lease_owner,lease_expires_at,record_version,
			provider_message_id,last_error_code,created_at,sent_at
		) VALUES (?,?,?,?,?,?,'PENDING',0,?,NULL,NULL,1,NULL,NULL,?,NULL)
	`, intent.OrderID, consentID, intent.Kind, intent.RecipientUserID, messageJSON, templateVersion,
		intent.AvailableAt.UTC(), now); err != nil {
		return mapSQLError(err)
	}
	result, err := transaction.ExecContext(ctx, `
		UPDATE notification_consents SET consumed_at=?
		WHERE id=? AND consumed_at IS NULL
	`, now, consentID)
	if err != nil {
		return mapSQLError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return ErrUnavailable
	}
	return nil
}

func jsonEqual(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	leftCanonical, leftErr := json.Marshal(leftValue)
	rightCanonical, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && string(leftCanonical) == string(rightCanonical)
}

func (store *mysqlStore) claimDue(ctx context.Context, now time.Time, limit uint16, owner [16]byte, lease time.Duration) ([]claimedDelivery, error) {
	if store == nil || store.db == nil {
		return nil, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		claimed, err := store.claimDueOnce(ctx, now, limit, owner, lease)
		if err == nil {
			return claimed, nil
		}
		if !isRetryableMySQL(err) || attempt == 1 {
			return nil, mapSQLError(err)
		}
	}
	return nil, ErrUnavailable
}

func (store *mysqlStore) claimDueOnce(ctx context.Context, now time.Time, limit uint16, owner [16]byte, lease time.Duration) ([]claimedDelivery, error) {
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback()

	rows, err := transaction.QueryContext(ctx, `
		SELECT id,order_id,recipient_user_id,kind,immutable_message_json,
		       template_config_version,attempt_count,record_version
		FROM notification_outbox
		WHERE (state='PENDING' AND next_attempt_at<=?)
		   OR (state='IN_FLIGHT' AND lease_expires_at<=?)
		ORDER BY id
		LIMIT ? FOR UPDATE SKIP LOCKED
	`, now, now, limit)
	if err != nil {
		return nil, err
	}
	var claimed []claimedDelivery
	for rows.Next() {
		var delivery claimedDelivery
		var kind string
		var messageJSON []byte
		var previousAttempts uint16
		if err := rows.Scan(&delivery.OutboxID, &delivery.OrderID, &delivery.RecipientUserID, &kind,
			&messageJSON, &delivery.TemplateConfigVersion, &previousAttempts, &delivery.recordVersion); err != nil {
			_ = rows.Close()
			return nil, err
		}
		delivery.Kind = Kind(kind)
		if previousAttempts == math.MaxUint16 || json.Unmarshal(messageJSON, &delivery.Message) != nil || !validMessage(delivery.Kind, delivery.Message) {
			_ = rows.Close()
			return nil, ErrUnavailable
		}
		delivery.AttemptCount = previousAttempts + 1
		delivery.leaseOwner = owner
		claimed = append(claimed, delivery)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	leaseExpiresAt := now.Add(lease)
	for index := range claimed {
		delivery := &claimed[index]
		result, err := transaction.ExecContext(ctx, `
			UPDATE notification_outbox
			SET state='IN_FLIGHT',attempt_count=attempt_count+1,next_attempt_at=NULL,
			    lease_owner=?,lease_expires_at=?,record_version=record_version+1,last_error_code=NULL
			WHERE id=? AND record_version=?
			  AND ((state='PENDING' AND next_attempt_at<=?) OR (state='IN_FLIGHT' AND lease_expires_at<=?))
		`, owner[:], leaseExpiresAt, delivery.OutboxID, delivery.recordVersion, now, now)
		if err != nil {
			return nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected != 1 {
			return nil, ErrUnavailable
		}
		delivery.recordVersion++
	}
	if err := transaction.Commit(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (store *mysqlStore) markSent(ctx context.Context, delivery claimedDelivery, result SendResult, now time.Time) error {
	if !utf8.ValidString(result.ProviderMessageID) || len(result.ProviderMessageID) > 128 {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='SENT',next_attempt_at=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=?,last_error_code=NULL,sent_at=?
		WHERE id=? AND state='IN_FLIGHT' AND lease_owner=? AND record_version=?
	`, []byte(result.ProviderMessageID), now, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion)
}

func (store *mysqlStore) markTemporaryFailure(ctx context.Context, delivery claimedDelivery, code string, nextAttemptAt time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='PENDING',next_attempt_at=?,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=NULL,last_error_code=?,sent_at=NULL
		WHERE id=? AND state='IN_FLIGHT' AND lease_owner=? AND record_version=?
	`, nextAttemptAt, code, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion)
}

func (store *mysqlStore) markPermanentFailure(ctx context.Context, delivery claimedDelivery, code string, _ time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='FAILED_PERMANENT',next_attempt_at=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=NULL,last_error_code=?,sent_at=NULL
		WHERE id=? AND state='IN_FLIGHT' AND lease_owner=? AND record_version=?
	`, code, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion)
}

func (store *mysqlStore) execCAS(ctx context.Context, statement string, arguments ...any) error {
	if store == nil || store.db == nil {
		return ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := store.db.ExecContext(ctx, statement, arguments...)
		if err == nil {
			rowsAffected, rowsErr := result.RowsAffected()
			if rowsErr != nil || rowsAffected != 1 {
				return ErrUnavailable
			}
			return nil
		}
		if !isRetryableMySQL(err) || attempt == 1 {
			return mapSQLError(err)
		}
	}
	return ErrUnavailable
}
