package billing

import (
	"context"
	"database/sql"
	"net/url"
	"time"
)

type Config struct {
	Environment      string
	AppID            string
	MerchantID       string
	PaymentNotifyURL string
	RefundNotifyURL  string
}

type Service struct {
	config   Config
	db       *sql.DB
	provider BillProvider
	now      func() time.Time
}

func New(db *sql.DB, provider BillProvider, config Config) *Service {
	return &Service{db: db, provider: provider, config: config, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) Summary(ctx context.Context, ownerUserID uint64, period BillingRange) (Summary, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) {
		return Summary{}, ErrInvalidInput
	}
	if err := authorizeOwner(ctx, service.db, ownerUserID); err != nil {
		return Summary{}, err
	}
	var summary Summary
	err := service.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(payable_cents) FILTER (WHERE state='COMPLETED'),0),COUNT(*) FILTER (WHERE state='COMPLETED') FROM orders WHERE paid_at>=$1 AND paid_at<$2`, period.From.UTC(), period.To.UTC()).Scan(&summary.EffectiveRevenueCents, &summary.EffectiveOrders)
	if err != nil {
		return Summary{}, ErrUnavailable
	}
	err = service.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_cents) FILTER (WHERE materialization_state='APPLIED'),0),COUNT(*) FILTER (WHERE materialization_state='APPLIED'),COUNT(*) FILTER (WHERE materialization_state<>'APPLIED') FROM refunds WHERE created_at>=$1 AND created_at<$2`, period.From.UTC(), period.To.UTC()).Scan(&summary.RefundCents, &summary.RefundCount, &summary.PendingRefunds)
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
	rows, err := service.db.QueryContext(ctx, `SELECT o.id,convert_from(o.order_no,'UTF8'),convert_from(p.out_trade_no,'UTF8'),convert_from(o.transaction_id,'UTF8'),o.state,o.payable_cents,o.paid_at FROM orders o JOIN prepayments p ON p.id=o.prepayment_id WHERE o.id>$1 AND o.paid_at>=$2 AND o.paid_at<$3 ORDER BY o.id LIMIT $4`, page.AfterID, period.From.UTC(), period.To.UTC(), page.Limit)
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
	rows, err := service.db.QueryContext(ctx, `SELECT id,COALESCE(order_id,0),convert_from(out_refund_no,'UTF8'),COALESCE(convert_from(provider_refund_id,'UTF8'),''),provider_state,amount_cents,created_at FROM refunds WHERE id>$1 AND created_at>=$2 AND created_at<$3 ORDER BY id LIMIT $4`, page.AfterID, period.From.UTC(), period.To.UTC(), page.Limit)
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

func (service *Service) valid() bool {
	return service != nil && service.db != nil && validBillingConfig(service.config)
}

func validRange(period BillingRange) bool {
	return !period.From.IsZero() && !period.To.IsZero() && period.From.Before(period.To) && period.To.Sub(period.From) <= 366*24*time.Hour
}

func validPage(page PageQuery) bool { return page.Limit > 0 && page.Limit <= 100 }

func authorizeOwner(ctx context.Context, db *sql.DB, userID uint64) error {
	var accountID uint64
	err := db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&accountID)
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

func validBillingConfig(config Config) bool {
	if config.Environment == "development" || config.Environment == "test" {
		return true
	}
	if (config.Environment != "uat" && config.Environment != "production") || config.AppID == "" || config.MerchantID == "" {
		return false
	}
	for _, raw := range []string{config.PaymentNotifyURL, config.RefundNotifyURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" {
			return false
		}
	}
	return true
}
func cloudBilling(config Config) bool {
	return config.Environment == "uat" || config.Environment == "production"
}
