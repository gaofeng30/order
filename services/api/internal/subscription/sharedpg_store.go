package subscription

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

type sharedPGStore struct{ client sharedSubscriptionExecutor }

func newSharedPGStore(client sharedSubscriptionExecutor) *sharedPGStore {
	return &sharedPGStore{client: client}
}

type subscriptionTime struct {
	EpochSeconds string `json:"epoch_seconds"`
	Nanosecond   int    `json:"nanosecond"`
	Zero         bool   `json:"zero"`
	Canonical    string `json:"canonical"`
}

func subscriptionTimeWire(at time.Time) subscriptionTime {
	at = at.UTC()
	return subscriptionTime{strconv.FormatInt(at.Unix(), 10), at.Nanosecond(), at.IsZero(), at.Format(time.RFC3339Nano)}
}

type subscriptionConsentWire struct {
	Order             string   `json:"order"`
	Kind              Kind     `json:"kind"`
	Decision          Decision `json:"decision"`
	Sequence          string   `json:"sequence"`
	Template          string   `json:"template"`
	DecidedSeconds    string   `json:"decided_seconds"`
	DecidedNanosecond int      `json:"decided_nanosecond"`
}
type subscriptionReply struct {
	Outcome    string                  `json:"outcome"`
	SQLState   string                  `json:"sqlstate"`
	Constraint string                  `json:"constraint"`
	Consent    subscriptionConsentWire `json:"consent"`
	Delivery   struct {
		ID        string `json:"id"`
		Order     string `json:"order"`
		Recipient string `json:"recipient"`
		Kind      Kind   `json:"kind"`
		Message   string `json:"message"`
		Template  string `json:"template"`
		Attempts  uint16 `json:"attempts"`
		Version   string `json:"version"`
	} `json:"delivery"`
}

func subscriptionUint(value string) (uint64, error) {
	n, e := strconv.ParseUint(value, 10, 64)
	if e != nil || strconv.FormatUint(n, 10) != value {
		return 0, ErrUnavailable
	}
	return n, nil
}
func (s *sharedPGStore) call(ctx context.Context, name string, args map[string]any) (subscriptionReply, error) {
	var r subscriptionReply
	if s == nil || s.client == nil || s.client.CallRPC(ctx, name, args, &r) != nil {
		return r, ErrUnavailable
	}
	switch r.Outcome {
	case "FOUND", "NOT_FOUND", "RECORDED", "CLAIMED", "EMPTY", "FINISHED":
		return r, nil
	case "SQL_FAILURE":
		if r.SQLState == "40P01" || r.SQLState == "40001" || r.SQLState == "55P03" {
			return r, &pgconn.PgError{Code: r.SQLState}
		}
		if r.SQLState == "23505" && (r.Constraint == "uq_notification_consents_user_idempotency" || r.Constraint == "uq_action_audits_command_receipt") {
			return r, &pgconn.PgError{Code: r.SQLState, ConstraintName: r.Constraint}
		}
	}
	return r, ErrUnavailable
}
func consentFromWire(w subscriptionConsentWire) (Subscription, error) {
	order, e := subscriptionUint(w.Order)
	if e != nil {
		return Subscription{}, e
	}
	sequence, e := subscriptionUint(w.Sequence)
	if e != nil {
		return Subscription{}, e
	}
	template, e := subscriptionUint(w.Template)
	if e != nil {
		return Subscription{}, e
	}
	secs, e := strconv.ParseInt(w.DecidedSeconds, 10, 64)
	if e != nil || strconv.FormatInt(secs, 10) != w.DecidedSeconds || w.DecidedNanosecond < 0 || w.DecidedNanosecond >= 1000000000 {
		return Subscription{}, ErrUnavailable
	}
	at := time.Unix(secs, int64(w.DecidedNanosecond))
	return Subscription{OrderID: order, Kind: w.Kind, Decision: w.Decision, Available: w.Decision == DecisionAccepted, GrantSequence: sequence, TemplateConfigVersion: template, DecidedAt: at.In(time.Local)}, nil
}
func (s *sharedPGStore) readConsent(ctx context.Context, user uint64, key [32]byte) (Subscription, bool, error) {
	r, e := s.call(ctx, "order_subscription_consent_read_v1", map[string]any{"p_user": strconv.FormatUint(user, 10), "p_key_hash": hex.EncodeToString(key[:])})
	if e != nil {
		return Subscription{}, false, e
	}
	if r.Outcome == "NOT_FOUND" {
		return Subscription{}, false, nil
	}
	if r.Outcome != "FOUND" {
		return Subscription{}, false, ErrUnavailable
	}
	v, e := consentFromWire(r.Consent)
	return v, e == nil, e
}
func (s *sharedPGStore) recordConsent(ctx context.Context, meta WriteMeta, input ConsentInput, now time.Time) (Subscription, error) {
	key := sha256.Sum256([]byte(meta.IdempotencyKey))
	if v, found, e := s.readConsent(ctx, meta.ActorUserID, key); e != nil {
		return Subscription{}, ErrUnavailable
	} else if found {
		return matchReplay(v, input)
	}
	request := sha256.Sum256([]byte(meta.RequestID))
	for attempt := 0; attempt < 2; attempt++ {
		r, e := s.call(ctx, "order_subscription_consent_record_v1", map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_order": strconv.FormatUint(input.OrderID, 10), "p_kind": string(input.Kind), "p_decision": string(input.Decision), "p_template_version": strconv.FormatUint(input.TemplateConfigVersion, 10), "p_key_hash": hex.EncodeToString(key[:]), "p_request_hash": hex.EncodeToString(request[:]), "p_now": subscriptionTimeWire(now)})
		if e == nil {
			if r.Outcome == "NOT_FOUND" {
				return Subscription{}, ErrNotFound
			}
			if r.Outcome != "RECORDED" {
				return Subscription{}, ErrUnavailable
			}
			sequence, e := subscriptionUint(r.Consent.Sequence)
			if e != nil {
				return Subscription{}, e
			}
			return Subscription{OrderID: input.OrderID, Kind: input.Kind, Decision: input.Decision, Available: input.Decision == DecisionAccepted, GrantSequence: sequence, TemplateConfigVersion: input.TemplateConfigVersion, DecidedAt: now}, nil
		}
		if isDuplicate(e) {
			v, found, readErr := s.readConsent(ctx, meta.ActorUserID, key)
			if readErr != nil || !found {
				return Subscription{}, ErrUnavailable
			}
			return matchReplay(v, input)
		}
		if !isRetryablePostgreSQL(e) || attempt == 1 {
			return Subscription{}, mapSQLError(e)
		}
	}
	return Subscription{}, ErrUnavailable
}
func (s *sharedPGStore) retry(ctx context.Context, name string, args map[string]any) (subscriptionReply, error) {
	for attempt := 0; attempt < 2; attempt++ {
		r, e := s.call(ctx, name, args)
		if e == nil {
			return r, nil
		}
		if !isRetryablePostgreSQL(e) || attempt == 1 {
			return r, mapSQLError(e)
		}
	}
	return subscriptionReply{}, ErrUnavailable
}
func (s *sharedPGStore) claimDue(ctx context.Context, cutoff, leaseNow time.Time, _ uint16, owner [16]byte, _ time.Duration) ([]claimedDelivery, error) {
	r, e := s.retry(ctx, "order_subscription_claim_v1", map[string]any{"p_cutoff": subscriptionTimeWire(cutoff), "p_lease_now": subscriptionTimeWire(leaseNow), "p_owner_hex": hex.EncodeToString(owner[:])})
	if e != nil {
		return nil, e
	}
	if r.Outcome == "EMPTY" {
		return nil, nil
	}
	if r.Outcome != "CLAIMED" {
		return nil, ErrUnavailable
	}
	w := r.Delivery
	var d claimedDelivery
	if d.OutboxID, e = subscriptionUint(w.ID); e != nil {
		return nil, e
	}
	if d.OrderID, e = subscriptionUint(w.Order); e != nil {
		return nil, e
	}
	if d.RecipientUserID, e = subscriptionUint(w.Recipient); e != nil {
		return nil, e
	}
	if d.TemplateConfigVersion, e = subscriptionUint(w.Template); e != nil {
		return nil, e
	}
	if d.recordVersion, e = subscriptionUint(w.Version); e != nil {
		return nil, e
	}
	d.Kind = w.Kind
	d.AttemptCount = w.Attempts
	d.leaseOwner = owner
	if json.Unmarshal([]byte(w.Message), &d.Message) != nil || !validMessage(d.Kind, d.Message) {
		return nil, ErrUnavailable
	}
	return []claimedDelivery{d}, nil
}
func (s *sharedPGStore) finish(ctx context.Context, name string, d claimedDelivery, args map[string]any) error {
	args["p_outbox"] = strconv.FormatUint(d.OutboxID, 10)
	args["p_owner_hex"] = hex.EncodeToString(d.leaseOwner[:])
	args["p_version"] = strconv.FormatUint(d.recordVersion, 10)
	r, e := s.retry(ctx, name, args)
	if e != nil {
		return e
	}
	if r.Outcome != "FINISHED" {
		return ErrUnavailable
	}
	return nil
}
func (s *sharedPGStore) markSent(ctx context.Context, d claimedDelivery, result SendResult, now time.Time) error {
	if !utf8.ValidString(result.ProviderMessageID) || len(result.ProviderMessageID) > 128 {
		return ErrUnavailable
	}
	return s.finish(ctx, "order_subscription_sent_v1", d, map[string]any{"p_provider_message_hex": hex.EncodeToString([]byte(result.ProviderMessageID)), "p_completed_at": subscriptionTimeWire(now)})
}
func (s *sharedPGStore) markTemporaryFailure(ctx context.Context, d claimedDelivery, code string, now, next time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return s.finish(ctx, "order_subscription_temporary_failure_v1", d, map[string]any{"p_code": code, "p_completed_at": subscriptionTimeWire(now), "p_next_at": subscriptionTimeWire(next)})
}
func (s *sharedPGStore) markPermanentFailure(ctx context.Context, d claimedDelivery, code string, now time.Time) error {
	if !validErrorCode(code) {
		return ErrUnavailable
	}
	return s.finish(ctx, "order_subscription_permanent_failure_v1", d, map[string]any{"p_code": code, "p_completed_at": subscriptionTimeWire(now)})
}

var _ sharedSubscriptionStore = (*sharedPGStore)(nil)
