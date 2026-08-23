package fulfillment

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestAESGCMTokenCipherRoundTripAndTamperShield(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	random := bytes.NewReader(bytes.Repeat([]byte{0x17}, 64))
	cipher, err := NewAESGCMTokenCipher(map[uint16][]byte{3: key}, 3, random)
	if err != nil {
		t.Fatal(err)
	}
	version, sealed, err := cipher.Seal(context.Background(), "opaque-token")
	if err != nil || version != 3 || len(sealed) == 0 {
		t.Fatalf("Seal() = %d/%x/%v", version, sealed, err)
	}
	opened, err := cipher.Open(context.Background(), version, sealed)
	if err != nil || opened != "opaque-token" {
		t.Fatalf("Open() = %q/%v", opened, err)
	}
	sealed[len(sealed)-1] ^= 0xff
	if opened, err := cipher.Open(context.Background(), version, sealed); !errors.Is(err, ErrTokenInvalid) || opened != "" {
		t.Fatalf("tampered Open() = %q/%v", opened, err)
	}
	if opened, err := cipher.Open(context.Background(), 9, []byte("ciphertext")); !errors.Is(err, ErrTokenInvalid) || opened != "" {
		t.Fatalf("unknown version Open() = %q/%v", opened, err)
	}
}

func TestAESGCMTokenCipherRejectsInvalidKeyring(t *testing.T) {
	for _, keys := range []map[uint16][]byte{
		nil,
		{1: bytes.Repeat([]byte{1}, 31)},
		{0: bytes.Repeat([]byte{1}, 32)},
	} {
		if cipher, err := NewAESGCMTokenCipher(keys, 1, bytes.NewReader(make([]byte, 32))); err == nil || cipher != nil {
			t.Fatalf("NewAESGCMTokenCipher(%v) = %#v/%v", keys, cipher, err)
		}
	}
}
