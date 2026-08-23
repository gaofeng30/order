package fulfillment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
)

func TestReceiptEvidenceDistinguishesConflictFromCorruption(t *testing.T) {
	digest := sha256.Sum256([]byte("original"))
	valid, err := json.Marshal(struct {
		RequestDigest string `json:"request_digest"`
	}{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	if err := matchReceiptEvidence(valid, digest); err != nil {
		t.Fatalf("matching evidence error = %v", err)
	}
	other := sha256.Sum256([]byte("other"))
	if err := matchReceiptEvidence(valid, other); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different request error = %v, want conflict", err)
	}
	for _, corrupt := range [][]byte{nil, []byte(`{}`), []byte(`{"request_digest":"no"}`), []byte(`{"request_digest":"` + hex.EncodeToString(digest[:]) + `","extra":true}`)} {
		if err := matchReceiptEvidence(corrupt, digest); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("corrupt evidence %q error = %v, want unavailable", corrupt, err)
		}
	}
}

func TestLockedOrderFactsFailClosedBeforeTransition(t *testing.T) {
	now := time.Date(2026, 8, 25, 8, 2, 0, 0, time.UTC)
	preparing := lockedOrder{
		id: 401, state: orderquery.StatePreparing, pickupDate: "2026-08-25", pickupNumber: 13,
		preparingAt: sql.NullTime{Time: now.Add(-time.Minute), Valid: true}, recordVersion: 1,
	}
	if !validPreparingOrder(preparing, now) {
		t.Fatal("valid PREPARING facts rejected")
	}
	preparing.redemptionHash = make([]byte, sha256.Size)
	if validPreparingOrder(preparing, now) {
		t.Fatal("PREPARING with token hash accepted")
	}

	ready := lockedOrder{
		id: 401, state: orderquery.StateReadyForPickup, pickupDate: "2026-08-25", pickupNumber: 13,
		preparingAt: sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true}, readyAt: sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
		redemptionHash: make([]byte, sha256.Size), ciphertext: []byte("ciphertext"),
		keyVersion: sql.NullInt64{Int64: 1, Valid: true}, issuedAt: sql.NullTime{Time: now.Add(-time.Minute), Valid: true}, recordVersion: 2,
	}
	if !validReadyOrder(ready, now) {
		t.Fatal("valid READY facts rejected")
	}
	ready.ciphertext = nil
	if validReadyOrder(ready, now) {
		t.Fatal("READY without ciphertext accepted")
	}
}

func TestReadyTokenCiphertextMustMatchLookupHash(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	cipher, err := NewAESGCMTokenCipher(map[uint16][]byte{1: key}, 1, bytes.NewReader(make([]byte, 128)))
	if err != nil {
		t.Fatal(err)
	}
	token := "opaque-ready-token"
	version, ciphertext, err := cipher.Seal(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 8, 2, 0, 0, time.UTC)
	hash := sha256.Sum256([]byte(token))
	order := lockedOrder{
		id: 401, state: orderquery.StateReadyForPickup, pickupDate: "2026-08-25", pickupNumber: 13,
		preparingAt: sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true}, readyAt: sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
		redemptionHash: hash[:], ciphertext: ciphertext, keyVersion: sql.NullInt64{Int64: int64(version), Valid: true},
		issuedAt: sql.NullTime{Time: now.Add(-time.Minute), Valid: true}, recordVersion: 2,
	}
	application := &MySQLApplication{cipher: cipher}
	if err := application.validateCurrentRedemption(context.Background(), order, now); err != nil {
		t.Fatalf("matching token facts error = %v", err)
	}
	wrong := sha256.Sum256([]byte("different"))
	order.redemptionHash = wrong[:]
	if err := application.validateCurrentRedemption(context.Background(), order, now); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("mismatched token facts error = %v, want unavailable", err)
	}
}

func TestReceiptRoleUsesPersistedRoleVocabulary(t *testing.T) {
	tests := []struct {
		actor merchantidentity.Actor
		want  string
		ok    bool
	}{
		{merchantidentity.ActorMerchantOwner, "OWNER", true},
		{merchantidentity.ActorMerchantSubaccount, "SUBACCOUNT", true},
		{"unknown", "", false},
	}
	for _, test := range tests {
		got, ok := receiptRole(test.actor)
		if got != test.want || ok != test.ok {
			t.Fatalf("receiptRole(%q) = %q,%v, want %q,%v", test.actor, got, ok, test.want, test.ok)
		}
	}
}
