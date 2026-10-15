package paymentorder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"strings"
	"testing"
	"time"
)

type failedQueryProvider struct {
	PaymentProvider
	closeErr error
	closes   int
}

func (p *failedQueryProvider) QueryTransaction(context.Context, string) (wechatpay.Transaction, error) {
	return wechatpay.Transaction{}, errors.New("query timeout")
}
func (p *failedQueryProvider) CloseTransaction(context.Context, string) error {
	p.closes++
	return p.closeErr
}

type queryContractRPC struct {
	t             *testing.T
	now, deadline time.Time
	finishes      []map[string]any
}

func (r *queryContractRPC) CallRPC(_ context.Context, name string, args map[string]any, out any) error {
	var value any
	switch name {
	case "order_payment_claim_query_v1":
		h := func(s string) string { return hex.EncodeToString([]byte(s)) }
		req, _ := json.Marshal(ProviderCreateRequest{AppID: "app", MerchantID: "merchant", Description: "order", OutTradeNo: "trade", TimeExpire: r.deadline.Format(time.RFC3339), NotifyURL: "https://example.invalid/notify", AmountCents: 1, Currency: "CNY", PayerOpenID: "payer", QuoteID: "1", QuoteDigest: strings.Repeat("0", 64)})
		digest := sha256.Sum256(req)
		record := map[string]any{"id": "1", "user_id": "1", "quote_id": "1", "record_version": "2", "idempotency_key_hash": strings.Repeat("0", 64), "out_trade_no": h("trade"), "expected_appid": h("app"), "expected_mchid": h("merchant"), "expected_amount_cents": "1", "currency": "CNY", "provider_create_request_json": string(req), "provider_create_request_digest": hex.EncodeToString(digest[:]), "effective_deadline": r.deadline, "provider_state": "CREATE_UNKNOWN", "create_attempted_at": r.now, "wx_request_payment_json": nil, "provider_prepay_id": nil, "last_queried_at": nil, "materialization_state": "AWAITING_PAYMENT", "pending_reason_code": nil, "materialized_at": nil, "lease_kind": "QUERY", "lease_owner": args["p_owner_hex"], "lease_expires_at": r.now.Add(30 * time.Second), "next_reconcile_at": r.now, "created_at": r.now, "updated_at": r.now}
		value = map[string]any{"outcome": "CLAIMED", "record": record, "owner_hex": args["p_owner_hex"], "version": "2", "expires_at": r.now.Add(30 * time.Second)}
	case "order_payment_finish_create_v1":
		r.finishes = append(r.finishes, args)
		value = map[string]any{"outcome": "FINISHED", "rows": "1"}
	case "order_payment_finish_query_v1":
		r.finishes = append(r.finishes, args)
		value = map[string]any{"outcome": "FINISHED"}
	default:
		r.t.Fatalf("unexpected RPC: %s", name)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func TestPaymentQueryFailureRetryContract(t *testing.T) {
	now := time.Date(2026, 10, 15, 4, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		expired   bool
		closeErr  error
		wantClose bool
	}{
		{"query failure before deadline", false, nil, false},
		{"expired close failure", true, errors.New("close timeout"), false},
		{"expired close success stops retry", true, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deadline := now.Add(time.Minute)
			if tc.expired {
				deadline = now.Add(-time.Second)
			}
			rpc := &queryContractRPC{t: t, now: now, deadline: deadline}
			provider := &failedQueryProvider{closeErr: tc.closeErr}
			service := NewSharedPGApplication(rpc, provider, Config{Environment: "test"})
			service.now = func() time.Time { return now }
			service.leaseOwner = func() ([16]byte, error) { return [16]byte{1}, nil }
			_, err := service.queryOne(context.Background(), 1, now, false)
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("expected query unavailable, got %v", err)
			}
			if len(rpc.finishes) != 1 {
				t.Fatalf("expected one completion, got %d", len(rpc.finishes))
			}
			finish := rpc.finishes[0]
			if finish["p_close"] != tc.wantClose {
				t.Fatalf("wrong completion: %v", finish)
			}
			if !tc.wantClose && !finish["p_next"].(time.Time).Equal(now.Add(30*time.Minute)) {
				t.Fatalf("retry at %v, want failure +30 minutes", finish["p_next"])
			}
			expectedCloses := 0
			if tc.expired {
				expectedCloses = 1
			}
			if provider.closes != expectedCloses {
				t.Fatalf("closes=%d", provider.closes)
			}
		})
	}
}

func TestUnknownCreateSchedulesThirtyMinuteCompensation(t *testing.T) {
	now := time.Date(2026, 10, 15, 4, 0, 0, 0, time.UTC)
	rpc := &queryContractRPC{t: t}
	service := NewSharedPGApplication(rpc, &failedQueryProvider{}, Config{Environment: "test"})
	if err := service.finishCreateUnknown(context.Background(), claimedCreate{id: 1, owner: [16]byte{1}, version: 2}, now); err != nil {
		t.Fatal(err)
	}
	if len(rpc.finishes) != 1 {
		t.Fatalf("completion count %d", len(rpc.finishes))
	}
	if got := rpc.finishes[0]["p_next"].(time.Time); !got.Equal(now.Add(30 * time.Minute)) {
		t.Fatalf("unknown create retry at %s, want 04:30 UTC", got)
	}
}

func TestSuccessfulCreateDoesNotRegisterCompensation(t *testing.T) {
	now := time.Date(2026, 10, 15, 4, 0, 0, 0, time.UTC)
	rpc := &queryContractRPC{t: t}
	service := NewSharedPGApplication(rpc, &failedQueryProvider{}, Config{Environment: "test"})
	if err := service.finishCreateSuccess(context.Background(), claimedCreate{id: 1, owner: [16]byte{1}, version: 2}, ProviderCreateResult{}, now); err != nil {
		t.Fatal(err)
	}
	if len(rpc.finishes) != 1 {
		t.Fatalf("completion count %d", len(rpc.finishes))
	}
	if got := rpc.finishes[0]["p_next"]; got != nil {
		t.Fatalf("normal waiting payment registered compensation: %v", got)
	}
}
