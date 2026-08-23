package merchantsoldout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
)

func TestNewImplementsFulfillmentSoldOutCommander(t *testing.T) {
	var commander fulfillment.SoldOutCommander = New(nil, nil, func() time.Time {
		return time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)
	})
	if commander == nil {
		t.Fatal("New() returned nil")
	}
}

func TestSetSoldOutRejectsInvalidFactsBeforeDatabaseAccess(t *testing.T) {
	now := time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC) // 09:00 Asia/Shanghai.
	service := New(nil, nil, func() time.Time { return now })
	truth := true
	validMeta := fulfillment.WriteMeta{ActorUserID: 7, IdempotencyKey: "soldout-7", RequestID: "request-7"}
	validCommand := fulfillment.SoldOutCommand{ProductID: 7, ServiceDate: "2026-08-25", SoldOut: &truth}

	tests := []struct {
		name    string
		ctx     context.Context
		meta    fulfillment.WriteMeta
		command fulfillment.SoldOutCommand
	}{
		{name: "nil context", ctx: nil, meta: validMeta, command: validCommand},
		{name: "missing actor", ctx: context.Background(), meta: fulfillment.WriteMeta{IdempotencyKey: "soldout-7", RequestID: "request-7"}, command: validCommand},
		{name: "bad key", ctx: context.Background(), meta: fulfillment.WriteMeta{ActorUserID: 7, IdempotencyKey: "bad key", RequestID: "request-7"}, command: validCommand},
		{name: "bad request id", ctx: context.Background(), meta: fulfillment.WriteMeta{ActorUserID: 7, IdempotencyKey: "soldout-7", RequestID: ""}, command: validCommand},
		{name: "missing product", ctx: context.Background(), meta: validMeta, command: fulfillment.SoldOutCommand{ServiceDate: "2026-08-25", SoldOut: &truth}},
		{name: "missing boolean", ctx: context.Background(), meta: validMeta, command: fulfillment.SoldOutCommand{ProductID: 7, ServiceDate: "2026-08-25"}},
		{name: "invalid calendar date", ctx: context.Background(), meta: validMeta, command: fulfillment.SoldOutCommand{ProductID: 7, ServiceDate: "2026-02-30", SoldOut: &truth}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := service.SetSoldOut(test.ctx, test.meta, test.command); !errors.Is(err, fulfillment.ErrInvalidInput) {
				t.Fatalf("SetSoldOut() error = %v, want invalid input", err)
			}
		})
	}
	if err := service.SetSoldOut(context.Background(), validMeta, validCommand); !errors.Is(err, fulfillment.ErrUnavailable) {
		t.Fatalf("valid command without dependencies error = %v, want unavailable", err)
	}
	if err := New(nil, nil, func() time.Time { return time.Time{} }).SetSoldOut(context.Background(), validMeta, validCommand); !errors.Is(err, fulfillment.ErrUnavailable) {
		t.Fatalf("zero clock error = %v, want unavailable", err)
	}
}

func TestAllowedServiceDateIsShanghaiTodayOrTomorrowOnly(t *testing.T) {
	now := time.Date(2026, 8, 25, 16, 30, 0, 0, time.UTC) // 2026-08-26 00:30 Asia/Shanghai.
	for _, test := range []struct {
		date string
		want bool
	}{
		{date: "2026-08-25", want: false},
		{date: "2026-08-26", want: true},
		{date: "2026-08-27", want: true},
		{date: "2026-08-28", want: false},
	} {
		if got := allowedServiceDate(test.date, now); got != test.want {
			t.Fatalf("allowedServiceDate(%q) = %v, want %v", test.date, got, test.want)
		}
	}
}

func TestReceiptEvidenceDistinguishesConflictFromCorruption(t *testing.T) {
	request := receiptRequest{ProductID: 7, ServiceDate: "2026-08-25", SoldOut: true}
	digest, err := requestDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	valid := []byte(`{"request_digest":"` + hex.EncodeToString(digest[:]) + `","sold_out":false}`)
	if err := matchReceiptEvidence(valid, request); err != nil {
		t.Fatalf("matching evidence error = %v", err)
	}
	if err := matchReceiptEvidence(valid, receiptRequest{ProductID: 7, ServiceDate: "2026-08-25", SoldOut: false}); !errors.Is(err, fulfillment.ErrIdempotencyConflict) {
		t.Fatalf("different request error = %v, want conflict", err)
	}
	other := sha256.Sum256([]byte("other"))
	for _, corrupt := range [][]byte{
		nil,
		[]byte(`{}`),
		[]byte(`{"request_digest":"not-hex"}`),
		[]byte(`{"request_digest":"` + hex.EncodeToString(other[:]) + `","extra":true}`),
	} {
		if err := matchReceiptEvidence(corrupt, request); !errors.Is(err, fulfillment.ErrUnavailable) {
			t.Fatalf("corrupt evidence %q error = %v, want unavailable", corrupt, err)
		}
	}
}

func TestPersistedRoleVocabulary(t *testing.T) {
	for _, test := range []struct {
		actor merchantidentity.Actor
		want  merchantidentity.Role
		ok    bool
	}{
		{actor: merchantidentity.ActorMerchantOwner, want: merchantidentity.RoleOwner, ok: true},
		{actor: merchantidentity.ActorMerchantSubaccount, want: merchantidentity.RoleSubaccount, ok: true},
		{actor: "unknown"},
	} {
		got, ok := persistedRole(test.actor)
		if got != test.want || ok != test.ok {
			t.Fatalf("persistedRole(%q) = %q,%v, want %q,%v", test.actor, got, ok, test.want, test.ok)
		}
	}
}
