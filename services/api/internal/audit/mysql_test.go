package audit

import (
	"bytes"
	"errors"
	"testing"
)

func TestRequestEvidenceMatchesOnlyTheOriginalCommandWithoutPersistingPII(t *testing.T) {
	type command struct {
		Name, Phone string
		Enabled     bool
	}
	original := command{Name: "林建国", Phone: "+8613800006620", Enabled: true}
	evidence, err := encodeRequestEvidence(original)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(evidence, []byte(original.Name)) || bytes.Contains(evidence, []byte(original.Phone)) {
		t.Fatalf("request evidence leaked PII: %s", evidence)
	}
	if err := matchRequestEvidence(evidence, original); err != nil {
		t.Fatalf("same request did not replay: %v", err)
	}
	changed := original
	changed.Enabled = false
	if err := matchRequestEvidence(evidence, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed request error=%v, want idempotency conflict", err)
	}
}
