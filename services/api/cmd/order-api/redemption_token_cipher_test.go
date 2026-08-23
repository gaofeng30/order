package main

import (
	"context"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/config"
)

func TestComposeRedemptionTokenCipherProvidesLocalRoundTrip(t *testing.T) {
	cipher, err := composeRedemptionTokenCipher(config.Development, "")
	if err != nil || cipher == nil {
		t.Fatalf("compose local cipher = %T/%v", cipher, err)
	}
	version, envelope, err := cipher.Seal(context.Background(), "local-redemption-token")
	if err != nil || version != 1 || string(envelope) == "local-redemption-token" {
		t.Fatalf("Seal() = version=%d bytes=%d err=%v", version, len(envelope), err)
	}
	plaintext, err := cipher.Open(context.Background(), version, envelope)
	if err != nil || plaintext != "local-redemption-token" {
		t.Fatalf("Open() = %q/%v", plaintext, err)
	}
}
