package main

import (
	"context"
	"errors"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/refund"
)

type adminPendingMaterializer interface {
	MaterializePending(context.Context, paymentorder.WriteMeta, uint64) (paymentorder.ConfirmResult, error)
}

type adminRefundRequester interface {
	RequestOrder(context.Context, refund.WriteMeta, uint64, string) (refund.Refund, error)
	RequestPaidPrepayment(context.Context, refund.WriteMeta, uint64, string) (refund.Refund, error)
}

type adminOrderProjector interface {
	GetOrder(context.Context, uint64, uint64) (adminreport.Order, error)
}

type adminCommandAdapter struct {
	payments adminPendingMaterializer
	refunds  adminRefundRequester
	orders   adminOrderProjector
}

func newAdminCommandAdapter(payments adminPendingMaterializer, refunds adminRefundRequester, orders adminOrderProjector) adminreport.CommandAdapter {
	return &adminCommandAdapter{payments: payments, refunds: refunds, orders: orders}
}

func (adapter *adminCommandAdapter) ProcessPending(ctx context.Context, meta adminreport.WriteMeta, prepaymentID uint64, action adminreport.PendingAction, reason string) (adminreport.PendingResult, error) {
	if adapter == nil || adapter.payments == nil || adapter.refunds == nil || adapter.orders == nil || prepaymentID == 0 {
		return adminreport.PendingResult{}, adminreport.ErrUnavailable
	}
	switch action {
	case adminreport.Materialize:
		if reason != "" {
			return adminreport.PendingResult{}, adminreport.ErrInvalidInput
		}
		result, err := adapter.payments.MaterializePending(ctx, paymentMeta(meta), prepaymentID)
		if err != nil {
			return adminreport.PendingResult{}, mapAdminCommandError(err)
		}
		if result.State != paymentorder.ConfirmOrderCreated || result.OrderID == 0 {
			return adminreport.PendingResult{}, adminreport.ErrConflict
		}
		order, err := adapter.orders.GetOrder(ctx, meta.ActorUserID, result.OrderID)
		if err != nil {
			return adminreport.PendingResult{}, mapAdminCommandError(err)
		}
		if order.ID != result.OrderID {
			return adminreport.PendingResult{}, adminreport.ErrUnavailable
		}
		return adminreport.PendingResult{Order: &order}, nil
	case adminreport.RefundPaid:
		if reason == "" {
			return adminreport.PendingResult{}, adminreport.ErrInvalidInput
		}
		created, err := adapter.refunds.RequestPaidPrepayment(ctx, refundMeta(meta), prepaymentID, reason)
		if err != nil {
			return adminreport.PendingResult{}, mapAdminCommandError(err)
		}
		if !validAdminRefund(created, 0) {
			return adminreport.PendingResult{}, adminreport.ErrUnavailable
		}
		projected := projectAdminRefund(created, nil, reason)
		return adminreport.PendingResult{Refund: &projected}, nil
	default:
		return adminreport.PendingResult{}, adminreport.ErrInvalidInput
	}
}

func (adapter *adminCommandAdapter) RequestRefund(ctx context.Context, meta adminreport.WriteMeta, orderID uint64, reason string) (adminreport.Order, adminreport.Refund, error) {
	if adapter == nil || adapter.refunds == nil || adapter.orders == nil || orderID == 0 || reason == "" {
		return adminreport.Order{}, adminreport.Refund{}, adminreport.ErrInvalidInput
	}
	created, err := adapter.refunds.RequestOrder(ctx, refundMeta(meta), orderID, reason)
	if err != nil {
		return adminreport.Order{}, adminreport.Refund{}, mapAdminCommandError(err)
	}
	if !validAdminRefund(created, orderID) {
		return adminreport.Order{}, adminreport.Refund{}, adminreport.ErrUnavailable
	}
	order, err := adapter.orders.GetOrder(ctx, meta.ActorUserID, orderID)
	if err != nil {
		return adminreport.Order{}, adminreport.Refund{}, mapAdminCommandError(err)
	}
	if order.ID != orderID {
		return adminreport.Order{}, adminreport.Refund{}, adminreport.ErrUnavailable
	}
	return order, projectAdminRefund(created, &order, reason), nil
}

func paymentMeta(meta adminreport.WriteMeta) paymentorder.WriteMeta {
	return paymentorder.WriteMeta{ActorUserID: meta.ActorUserID, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
}

func refundMeta(meta adminreport.WriteMeta) refund.WriteMeta {
	return refund.WriteMeta{ActorKind: refund.ActorMerchant, ActorUserID: meta.ActorUserID, IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID}
}

func projectAdminRefund(value refund.Refund, order *adminreport.Order, reason string) adminreport.Refund {
	projected := adminreport.Refund{
		ID: value.ID, OrderID: value.OrderID, State: adminRefundState(value.State),
		ProviderRefundID: value.ProviderRefundID, Reason: reason, AmountCents: value.AmountCents, RequestedAt: value.RequestedAt,
	}
	if order != nil {
		projected.OrderNo = order.OrderNo
		projected.TransactionID = order.TransactionID
		projected.PaidAt = order.PaidAt
	}
	return projected
}

func adminRefundState(state refund.ProviderState) string {
	switch state {
	case refund.ProviderReady, refund.ProviderCreateClaimed, refund.ProviderCreateUnknown, refund.ProviderProcessing:
		return "退款中"
	case refund.ProviderSuccess:
		return "已退款"
	case refund.ProviderClosed:
		return "退款失败"
	default:
		return ""
	}
}

func validAdminRefund(value refund.Refund, expectedOrderID uint64) bool {
	return value.ID > 0 && value.OrderID == expectedOrderID && value.AmountCents > 0 && !value.RequestedAt.IsZero() && adminRefundState(value.State) != ""
}

func mapAdminCommandError(err error) error {
	switch {
	case errors.Is(err, adminreport.ErrInvalidInput), errors.Is(err, paymentorder.ErrInvalidInput), errors.Is(err, refund.ErrInvalidInput):
		return adminreport.ErrInvalidInput
	case errors.Is(err, adminreport.ErrForbidden), errors.Is(err, paymentorder.ErrUnauthenticated), errors.Is(err, paymentorder.ErrForbidden), errors.Is(err, refund.ErrUnauthenticated), errors.Is(err, refund.ErrForbidden):
		return adminreport.ErrForbidden
	case errors.Is(err, adminreport.ErrNotFound), errors.Is(err, paymentorder.ErrNotFound), errors.Is(err, refund.ErrNotFound):
		return adminreport.ErrNotFound
	case errors.Is(err, adminreport.ErrConflict), errors.Is(err, paymentorder.ErrIdempotencyConflict), errors.Is(err, refund.ErrIdempotencyConflict), errors.Is(err, refund.ErrTransitionNotAllowed):
		return adminreport.ErrConflict
	default:
		return adminreport.ErrUnavailable
	}
}
