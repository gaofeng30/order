package billing

import (
	"context"
	"database/sql"
	"time"
)

type Service struct {
	db       *sql.DB
	provider BillProvider
	now      func() time.Time
}

func New(db *sql.DB, provider BillProvider) *Service {
	return &Service{db: db, provider: provider, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Summary(ctx context.Context, ownerUserID uint64, period BillingRange) (Summary, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) {
		return Summary{}, ErrInvalidInput
	}
	if err := authorizeOwner(ctx, service.db, ownerUserID); err != nil {
		return Summary{}, err
	}
	var summary Summary
	err := service.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(IF(state='COMPLETED',payable_cents,0)),0),COALESCE(SUM(state='COMPLETED'),0) FROM orders WHERE paid_at>=? AND paid_at<?`, period.From.UTC(), period.To.UTC()).Scan(&summary.EffectiveRevenueCents, &summary.EffectiveOrders)
	if err != nil {
		return Summary{}, ErrUnavailable
	}
	err = service.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(IF(materialization_state='APPLIED',amount_cents,0)),0),COALESCE(SUM(materialization_state='APPLIED'),0),COALESCE(SUM(materialization_state<>'APPLIED'),0) FROM refunds WHERE created_at>=? AND created_at<?`, period.From.UTC(), period.To.UTC()).Scan(&summary.RefundCents, &summary.RefundCount, &summary.PendingRefunds)
	if err != nil {
		return Summary{}, ErrUnavailable
	}
	return summary, nil
}

func (service *Service) ListPayments(ctx context.Context, ownerUserID uint64, period BillingRange, page PageQuery) ([]Payment, uint64, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) || !validPage(page) {
		return nil, 0, ErrInvalidInput
	}
	if err := authorizeOwner(ctx, service.db, ownerUserID); err != nil {
		return nil, 0, err
	}
	rows, err := service.db.QueryContext(ctx, `SELECT o.id,CONVERT(o.order_no USING utf8mb4),CONVERT(p.out_trade_no USING utf8mb4),CONVERT(o.transaction_id USING utf8mb4),o.state,o.payable_cents,o.paid_at FROM orders o JOIN prepayments p ON p.id=o.prepayment_id WHERE o.id>? AND o.paid_at>=? AND o.paid_at<? ORDER BY o.id LIMIT ?`, page.AfterID, period.From.UTC(), period.To.UTC(), page.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	items := make([]Payment, 0)
	for rows.Next() {
		var item Payment
		if rows.Scan(&item.OrderID, &item.OrderNo, &item.OutTradeNo, &item.TransactionID, &item.State, &item.AmountCents, &item.PaidAt) != nil {
			return nil, 0, ErrUnavailable
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	return items, nextPaymentCursor(items, page.Limit), nil
}

func (service *Service) ListRefunds(ctx context.Context, ownerUserID uint64, period BillingRange, page PageQuery) ([]Refund, uint64, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) || !validPage(page) {
		return nil, 0, ErrInvalidInput
	}
	if err := authorizeOwner(ctx, service.db, ownerUserID); err != nil {
		return nil, 0, err
	}
	rows, err := service.db.QueryContext(ctx, `SELECT id,COALESCE(order_id,0),CONVERT(out_refund_no USING utf8mb4),COALESCE(CONVERT(provider_refund_id USING utf8mb4),''),provider_state,amount_cents,created_at FROM refunds WHERE id>? AND created_at>=? AND created_at<? ORDER BY id LIMIT ?`, page.AfterID, period.From.UTC(), period.To.UTC(), page.Limit)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	items := make([]Refund, 0)
	for rows.Next() {
		var item Refund
		if rows.Scan(&item.ID, &item.OrderID, &item.OutRefundNo, &item.ProviderRefundID, &item.State, &item.AmountCents, &item.RequestedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	return items, nextRefundCursor(items, page.Limit), nil
}

func (service *Service) valid() bool { return service != nil && service.db != nil }

func validRange(period BillingRange) bool {
	return !period.From.IsZero() && !period.To.IsZero() && period.From.Before(period.To) && period.To.Sub(period.From) <= 366*24*time.Hour
}

func validPage(page PageQuery) bool { return page.Limit > 0 && page.Limit <= 100 }

func authorizeOwner(ctx context.Context, db *sql.DB, userID uint64) error {
	var accountID uint64
	err := db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&accountID)
	if err == sql.ErrNoRows {
		return ErrForbidden
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

func nextPaymentCursor(items []Payment, limit uint16) uint64 {
	if len(items) == int(limit) {
		return items[len(items)-1].OrderID
	}
	return 0
}

func nextRefundCursor(items []Refund, limit uint16) uint64 {
	if len(items) == int(limit) {
		return items[len(items)-1].ID
	}
	return 0
}
