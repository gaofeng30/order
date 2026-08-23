package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/subscription"
)

type subscriptionTransactionEnqueuer interface {
	EnqueueInTx(context.Context, *sql.Tx, subscription.NotificationIntent) error
}

type refundSubscriptionAdapter struct {
	subscriptions subscriptionTransactionEnqueuer
}

func newRefundSubscriptionAdapter(subscriptions subscriptionTransactionEnqueuer) refund.NotificationEnqueuer {
	return &refundSubscriptionAdapter{subscriptions: subscriptions}
}

func (adapter *refundSubscriptionAdapter) EnqueueRefundResultInTx(ctx context.Context, transaction *sql.Tx, orderID, userID uint64, orderNo string, availableAt time.Time) error {
	if adapter == nil || adapter.subscriptions == nil {
		return refund.ErrUnavailable
	}
	return adapter.subscriptions.EnqueueInTx(ctx, transaction, subscription.NotificationIntent{
		OrderID: orderID, RecipientUserID: userID, Kind: subscription.KindRefundResult,
		Message: subscription.Message{OrderNumber: orderNo, RefundResult: "REFUNDED"}, AvailableAt: availableAt,
	})
}
