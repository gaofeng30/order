package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/subscription"
)

type subscriptionEnqueueProbe struct {
	intent subscription.NotificationIntent
}

func (probe *subscriptionEnqueueProbe) EnqueueInTx(_ context.Context, _ *sql.Tx, intent subscription.NotificationIntent) error {
	probe.intent = intent
	return nil
}

func TestRefundSubscriptionAdapterBuildsImmutableResultIntent(t *testing.T) {
	probe := &subscriptionEnqueueProbe{}
	adapter := newRefundSubscriptionAdapter(probe)
	at := time.Date(2026, 8, 25, 3, 0, 0, 0, time.UTC)
	if err := adapter.EnqueueRefundResultInTx(context.Background(), nil, 42, 7, "ORDER-42", at); err != nil {
		t.Fatal(err)
	}
	want := subscription.NotificationIntent{
		OrderID: 42, RecipientUserID: 7, Kind: subscription.KindRefundResult,
		Message: subscription.Message{OrderNumber: "ORDER-42", RefundResult: "REFUNDED"}, AvailableAt: at,
	}
	if probe.intent != want {
		t.Fatalf("intent = %#v", probe.intent)
	}
}
