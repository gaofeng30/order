package subscription

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestRecordConsentPersistsAcceptedDecision(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newService(store, NewFakeProvider())
	now := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	got, err := service.RecordConsent(context.Background(), WriteMeta{
		ActorUserID:    7,
		IdempotencyKey: "accept-ready-order-42",
		RequestID:      "request-accept-ready-order-42",
	}, ConsentInput{
		OrderID:               42,
		Kind:                  KindReady,
		Decision:              DecisionAccepted,
		TemplateConfigVersion: 3,
	})
	if err != nil {
		t.Fatalf("RecordConsent() error = %v", err)
	}
	if got.OrderID != 42 || got.Kind != KindReady || got.Decision != DecisionAccepted || !got.Available || got.GrantSequence != 1 || !got.DecidedAt.Equal(now) {
		t.Fatalf("RecordConsent() = %#v", got)
	}
	if store.consentCount() != 1 {
		t.Fatalf("persisted consents = %d, want 1", store.consentCount())
	}
}

func TestRunDueSendsClaimedIntentAndMarksItSent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 18, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	store.claims = []claimedDelivery{{
		Delivery: Delivery{
			OutboxID: 11, OrderID: 42, RecipientUserID: 7, Kind: KindReady,
			Message:               Message{OrderNumber: "ORDER-42", PickupDate: "2026-08-25", PickupTime: "12:00", PickupPoint: "North gate"},
			TemplateConfigVersion: 3, AttemptCount: 1,
		},
		recordVersion: 2,
	}}
	provider := NewFakeProvider()
	service := newService(store, provider)
	service.owner = func() ([16]byte, error) { return [16]byte{1}, nil }

	got, err := service.RunDue(context.Background(), now, 5)
	if err != nil {
		t.Fatalf("RunDue() error = %v", err)
	}
	if got != (RunResult{Claimed: 1, Sent: 1}) {
		t.Fatalf("RunDue() = %#v", got)
	}
	if deliveries := provider.Deliveries(); len(deliveries) != 1 || deliveries[0].OutboxID != 11 {
		t.Fatalf("provider deliveries = %#v", deliveries)
	}
	if store.sentCount() != 1 {
		t.Fatalf("sent marks = %d, want 1", store.sentCount())
	}
}

func TestRunDueClassifiesProviderFailuresWithoutPersistingRawError(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 18, 15, 0, 0, time.UTC)
	store := newMemoryStore()
	base := Delivery{OrderID: 42, RecipientUserID: 7, Kind: KindRefundResult, Message: Message{OrderNumber: "ORDER-42", RefundResult: "REFUNDED"}, TemplateConfigVersion: 4, AttemptCount: 1}
	store.claims = []claimedDelivery{
		{Delivery: base, recordVersion: 2},
		{Delivery: base, recordVersion: 2},
	}
	store.claims[0].OutboxID = 12
	store.claims[1].OutboxID = 13
	provider := NewFakeProvider()
	provider.Queue(SendResult{}, &SendError{Code: "RATE_LIMITED", Permanent: false})
	provider.Queue(SendResult{}, &SendError{Code: "TEMPLATE_REJECTED", Permanent: true})
	service := newService(store, provider)
	service.owner = func() ([16]byte, error) { return [16]byte{2}, nil }

	got, err := service.RunDue(context.Background(), now, 2)
	if err != nil {
		t.Fatalf("RunDue() error = %v", err)
	}
	if got != (RunResult{Claimed: 2, TemporaryFailed: 1, PermanentFailed: 1}) {
		t.Fatalf("RunDue() = %#v", got)
	}
	temporary, permanent := store.failureCodes()
	if len(temporary) != 1 || temporary[0] != "RATE_LIMITED" || len(permanent) != 1 || permanent[0] != "TEMPLATE_REJECTED" {
		t.Fatalf("failure codes = temporary %#v, permanent %#v", temporary, permanent)
	}
}

func TestEnqueueRejectsMalformedTemplatePayloadBeforePersistence(t *testing.T) {
	t.Parallel()

	db, script := openScriptDB(t, beginStep(), rollbackStep())
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	service := newService(newMemoryStore(), NewFakeProvider())
	err = service.EnqueueInTx(context.Background(), transaction, NotificationIntent{
		OrderID: 42, RecipientUserID: 7, Kind: KindReady, AvailableAt: time.Now().UTC(),
		Message: Message{OrderNumber: "ORDER-42", PickupDate: "2026/08/25", PickupTime: "12-00", PickupPoint: "North gate"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("EnqueueInTx() error = %v, want ErrInvalidInput", err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

type memoryStore struct {
	mu        sync.Mutex
	consents  []Subscription
	claims    []claimedDelivery
	sent      []claimedDelivery
	temporary []string
	permanent []string
}

func newMemoryStore() *memoryStore { return &memoryStore{} }

func (store *memoryStore) recordConsent(_ context.Context, _ WriteMeta, input ConsentInput, now time.Time) (Subscription, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	consent := Subscription{
		OrderID: input.OrderID, Kind: input.Kind, Decision: input.Decision,
		Available: input.Decision == DecisionAccepted, GrantSequence: uint64(len(store.consents) + 1),
		TemplateConfigVersion: input.TemplateConfigVersion, DecidedAt: now,
	}
	store.consents = append(store.consents, consent)
	return consent, nil
}

func (store *memoryStore) consentCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.consents)
}

func (store *memoryStore) sentCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.sent)
}

func (store *memoryStore) failureCodes() ([]string, []string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]string(nil), store.temporary...), append([]string(nil), store.permanent...)
}

func (*memoryStore) enqueueInTx(context.Context, *sql.Tx, NotificationIntent, time.Time) error {
	return nil
}
func (store *memoryStore) claimDue(_ context.Context, _ time.Time, limit uint16, owner [16]byte, _ time.Duration) ([]claimedDelivery, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	count := int(limit)
	if count > len(store.claims) {
		count = len(store.claims)
	}
	claimed := append([]claimedDelivery(nil), store.claims[:count]...)
	for index := range claimed {
		claimed[index].leaseOwner = owner
	}
	store.claims = store.claims[count:]
	return claimed, nil
}
func (store *memoryStore) markSent(_ context.Context, delivery claimedDelivery, _ SendResult, _ time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.sent = append(store.sent, delivery)
	return nil
}
func (store *memoryStore) markTemporaryFailure(_ context.Context, _ claimedDelivery, code string, _ time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.temporary = append(store.temporary, code)
	return nil
}
func (store *memoryStore) markPermanentFailure(_ context.Context, _ claimedDelivery, code string, _ time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.permanent = append(store.permanent, code)
	return nil
}
