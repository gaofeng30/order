package orderquery

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type openerStub struct {
	token string
	err   error
}

func (stub openerStub) Open(context.Context, uint16, []byte) (string, error) {
	return stub.token, stub.err
}

func TestMaskPhoneNeverReturnsUnmaskedPII(t *testing.T) {
	if got, ok := maskPhone("+8613812345678"); !ok || got != "+*********5678" {
		t.Fatalf("maskPhone() = %q/%v", got, ok)
	}
	for _, invalid := range []string{"", "13812345678", "+0861234", "+123-456", "+1234"} {
		if got, ok := maskPhone(invalid); ok || got != "" {
			t.Fatalf("maskPhone(%q) = %q/%v", invalid, got, ok)
		}
	}
}

func TestRedemptionTokenRequiresDecryptAndHashMatch(t *testing.T) {
	token := "opaque-redemption-token"
	hash := sha256.Sum256([]byte(token))
	got, err := openRedemption(context.Background(), openerStub{token: token}, 2, []byte("ciphertext"), hash[:])
	if err != nil || got != token {
		t.Fatalf("openRedemption() = %q/%v", got, err)
	}
	badHash := sha256.Sum256([]byte("different"))
	for _, test := range []struct {
		name       string
		opener     TokenOpener
		version    uint16
		ciphertext []byte
		hash       []byte
	}{
		{name: "missing opener", version: 2, ciphertext: []byte("ciphertext"), hash: hash[:]},
		{name: "bad version", opener: openerStub{token: token}, ciphertext: []byte("ciphertext"), hash: hash[:]},
		{name: "decrypt error", opener: openerStub{err: errors.New("decrypt")}, version: 2, ciphertext: []byte("ciphertext"), hash: hash[:]},
		{name: "hash mismatch", opener: openerStub{token: token}, version: 2, ciphertext: []byte("ciphertext"), hash: badHash[:]},
	} {
		t.Run(test.name, func(t *testing.T) {
			if value, err := openRedemption(context.Background(), test.opener, test.version, test.ciphertext, test.hash); !errors.Is(err, ErrUnavailable) || value != "" {
				t.Fatalf("openRedemption() = %q/%v", value, err)
			}
		})
	}
}

func TestActionsAreDerivedWithoutAdvancingState(t *testing.T) {
	pickup := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	if got := userActions(StateReserved, pickup.Add(-31*time.Minute), pickup); len(got) != 1 || got[0] != ActionCancel {
		t.Fatalf("userActions before cutoff = %#v", got)
	}
	if got := userActions(StateReserved, pickup.Add(-30*time.Minute), pickup); len(got) != 0 {
		t.Fatalf("userActions equality = %#v", got)
	}
	if got := merchantActions(StatePreparing); len(got) != 1 || got[0] != ActionReady {
		t.Fatalf("merchant preparing actions = %#v", got)
	}
	if got := merchantActions(StateReadyForPickup); len(got) != 1 || got[0] != ActionRedeem {
		t.Fatalf("merchant ready actions = %#v", got)
	}
}

func TestTransitionHistoryFailsClosedOnImpossibleChronology(t *testing.T) {
	materialized := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	record := orderRecord{
		summary:     Summary{MaterializedAt: materialized},
		preparingAt: sql.NullTime{Time: materialized.Add(time.Minute), Valid: true},
		readyAt:     sql.NullTime{Time: materialized.Add(2 * time.Minute), Valid: true},
	}
	if !validTransitionHistory(record) {
		t.Fatal("valid transition history rejected")
	}
	record.readyAt.Time = materialized
	if validTransitionHistory(record) {
		t.Fatal("READY before PREPARING accepted")
	}
	record.readyAt.Time = materialized.Add(2 * time.Minute)
	record.refundedAt = sql.NullTime{Time: materialized.Add(3 * time.Minute), Valid: true}
	if validTransitionHistory(record) {
		t.Fatal("REFUNDED without REFUNDING accepted")
	}
}
