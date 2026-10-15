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

	"github.com/jackc/pgx/v5/pgconn"
)

type postgresStore struct{ db *sql.DB }

func newPostgreSQLStore(db *sql.DB) *postgresStore { return &postgresStore{db: db} }

func (store *postgresStore) recordConsent(ctx context.Context, meta WriteMeta, input ConsentInput, now time.Time) (Subscription, error) {
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
		if !isRetryablePostgreSQL(err) || attempt == 1 {
			return Subscription{}, mapSQLError(err)
		}
	}
	return Subscription{}, ErrUnavailable
}

func (store *postgresStore) recordConsentOnce(ctx context.Context, meta WriteMeta, input ConsentInput, keyHash [32]byte, now time.Time) (Subscription, error) {
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return Subscription{}, err
	}
	defer transaction.Rollback()

	var ownerUserID uint64
	if err := transaction.QueryRowContext(ctx, `
		SELECT user_id FROM orders WHERE id=$1 FOR UPDATE
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
		WHERE order_id=$1 AND kind=$2
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
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULL)
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
		WHERE user_id=$1 AND idempotency_key_hash=$2
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
		) VALUES ('COMMAND_RECEIPT','USER',$1,$2,$3,'ORDER',$4,$5,$6,'SUCCEEDED','CONSENT_RECORDED',$7,$8,$9)
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
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505" &&
		(pgError.ConstraintName == "uq_notification_consents_user_idempotency" || pgError.ConstraintName == "uq_action_audits_command_receipt")
}

func isRetryablePostgreSQL(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && (pgError.Code == "40P01" || pgError.Code == "40001" || pgError.Code == "55P03")
}

func mapSQLError(err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrIdempotencyConflict) || errors.Is(err, ErrUnavailable) {
		return err
	}
	return ErrUnavailable
}

func (*postgresStore) enqueueInTx(ctx context.Context, transaction *sql.Tx, intent NotificationIntent, now time.Time) error {
	var ownerUserID uint64
	if err := transaction.QueryRowContext(ctx, `
		SELECT user_id FROM orders WHERE id=$1 FOR UPDATE
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
		WHERE order_id=$1 AND kind=$2 AND decision='ACCEPTED'
		ORDER BY grant_sequence
		LIMIT 1 FOR UPDATE
	`, intent.OrderID, intent.Kind).Scan(&consentID, &consentUserID, &templateVersion, &consumedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
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
		WHERE order_id=$1 AND kind=$2 FOR UPDATE
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
		) VALUES ($1,$2,$3,$4,$5,$6,'PENDING',0,$7,NULL,NULL,1,NULL,NULL,$8,NULL)
	`, intent.OrderID, consentID, intent.Kind, intent.RecipientUserID, messageJSON, templateVersion,
		intent.AvailableAt.UTC(), now); err != nil {
		return mapSQLError(err)
	}
	result, err := transaction.ExecContext(ctx, `
		UPDATE notification_consents SET consumed_at=$1
		WHERE id=$2 AND consumed_at IS NULL
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

func (store *postgresStore) claimDue(ctx context.Context, dueBefore, leaseNow time.Time, limit uint16, owner [16]byte, lease time.Duration) ([]claimedDelivery, error) {
	if store == nil || store.db == nil {
		return nil, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		claimed, err := store.claimDueOnce(ctx, dueBefore, leaseNow, limit, owner, lease)
		if err == nil {
			return claimed, nil
		}
		if !isRetryablePostgreSQL(err) || attempt == 1 {
			return nil, mapSQLError(err)
		}
	}
	return nil, ErrUnavailable
}

func (store *postgresStore) claimDueOnce(ctx context.Context, dueBefore, leaseNow time.Time, limit uint16, owner [16]byte, lease time.Duration) ([]claimedDelivery, error) {
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback()

	rows, err := transaction.QueryContext(ctx, `
		SELECT id,order_id,recipient_user_id,kind,immutable_message_json,
		       template_config_version,attempt_count,record_version
		FROM notification_outbox
		WHERE (state='PENDING' AND next_attempt_at<=$1)
		   OR (state='IN_FLIGHT' AND lease_expires_at<=$2)
		ORDER BY id
		LIMIT $3 FOR UPDATE SKIP LOCKED
	`, dueBefore, leaseNow, limit)
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

	leaseExpiresAt := leaseNow.Add(lease)
	for index := range claimed {
		delivery := &claimed[index]
		result, err := transaction.ExecContext(ctx, `
			UPDATE notification_outbox
			SET state='IN_FLIGHT',attempt_count=attempt_count+1,next_attempt_at=NULL,
			    lease_owner=$1,lease_expires_at=$2,record_version=record_version+1,last_error_code=NULL
			WHERE id=$3 AND record_version=$4
			  AND ((state='PENDING' AND next_attempt_at<=$5) OR (state='IN_FLIGHT' AND lease_expires_at<=$6))
		`, owner[:], leaseExpiresAt, delivery.OutboxID, delivery.recordVersion, dueBefore, leaseNow)
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

func (store *postgresStore) markSent(ctx context.Context, delivery claimedDelivery, result SendResult, now time.Time) error {
	if !utf8.ValidString(result.ProviderMessageID) || len(result.ProviderMessageID) > 128 {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='SENT',next_attempt_at=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=$1,last_error_code=NULL,sent_at=$2
		WHERE id=$3 AND state='IN_FLIGHT' AND lease_owner=$4 AND record_version=$5 AND lease_expires_at>$6
	`, []byte(result.ProviderMessageID), now, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion, now)
}

func (store *postgresStore) markTemporaryFailure(ctx context.Context, delivery claimedDelivery, code string, completedAt, nextAttemptAt time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='PENDING',next_attempt_at=$1,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=NULL,last_error_code=$2,sent_at=NULL
		WHERE id=$3 AND state='IN_FLIGHT' AND lease_owner=$4 AND record_version=$5 AND lease_expires_at>$6
	`, nextAttemptAt, code, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion, completedAt)
}

func (store *postgresStore) markPermanentFailure(ctx context.Context, delivery claimedDelivery, code string, completedAt time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return store.execCAS(ctx, `
		UPDATE notification_outbox
		SET state='FAILED_PERMANENT',next_attempt_at=NULL,lease_owner=NULL,lease_expires_at=NULL,
		    record_version=record_version+1,provider_message_id=NULL,last_error_code=$1,sent_at=NULL
		WHERE id=$2 AND state='IN_FLIGHT' AND lease_owner=$3 AND record_version=$4 AND lease_expires_at>$5
	`, code, delivery.OutboxID, delivery.leaseOwner[:], delivery.recordVersion, completedAt)
}

func (store *postgresStore) execCAS(ctx context.Context, statement string, arguments ...any) error {
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
		if !isRetryablePostgreSQL(err) || attempt == 1 {
			return mapSQLError(err)
		}
	}
	return ErrUnavailable
}
