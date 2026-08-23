package audit

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/quote"
)

func TestQuoteReceiptEvidenceRoundTripContainsNoRequestPII(t *testing.T) {
	digest := sha256.Sum256([]byte("contact-name phone order-note flavors"))
	evidence, err := encodeQuoteReceiptEvidence(digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte("contact-name"), []byte("phone"), []byte("order-note"), []byte("flavors")} {
		if bytes.Contains(evidence, forbidden) {
			t.Fatalf("evidence leaked %q: %s", forbidden, evidence)
		}
	}
	decoded, err := decodeQuoteReceiptEvidence(evidence)
	if err != nil || decoded != digest {
		t.Fatalf("decoded = %x, %v", decoded, err)
	}
	for _, corrupted := range [][]byte{nil, []byte(`{}`), []byte(`{"request_digest":"bad"}`), append(evidence, []byte(`{}`)...)} {
		if _, err := decodeQuoteReceiptEvidence(corrupted); !errors.Is(err, quote.ErrSnapshotInvalid) {
			t.Fatalf("corruption %q error = %v", corrupted, err)
		}
	}
}

func TestQuoteReceiptHashesUseFrozenUserScopeAndOpaqueKeys(t *testing.T) {
	var material [13]byte
	copy(material[:5], "USER\x00")
	binary.BigEndian.PutUint64(material[5:], 7)
	wantScope := sha256.Sum256(material[:])
	if got := quoteUserScopeHash(7); got != wantScope {
		t.Fatalf("scope hash = %x, want %x", got, wantScope)
	}
	if quoteOperationKeyHash("quote-key") != sha256.Sum256([]byte("quote-key")) {
		t.Fatal("operation key hash drifted")
	}
	if quoteRequestIDHash("request-1") != sha256.Sum256([]byte("request-1")) {
		t.Fatal("request id hash drifted")
	}
}
