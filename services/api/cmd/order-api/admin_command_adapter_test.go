package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/refund"
)

type adminPaymentProbe struct {
	result paymentorder.ConfirmResult
	err    error
}

func (probe *adminPaymentProbe) MaterializePending(context.Context, paymentorder.WriteMeta, uint64) (paymentorder.ConfirmResult, error) {
	return probe.result, probe.err
}

type adminRefundProbe struct {
	result              refund.Refund
	err                 error
	orderID, prepayment uint64
	orderMeta, paidMeta refund.WriteMeta
}

func (probe *adminRefundProbe) RequestOrder(_ context.Context, meta refund.WriteMeta, orderID uint64, _ string) (refund.Refund, error) {
	probe.orderID = orderID
	probe.orderMeta = meta
	return probe.result, probe.err
}
func (probe *adminRefundProbe) RequestPaidPrepayment(_ context.Context, meta refund.WriteMeta, prepaymentID uint64, _ string) (refund.Refund, error) {
	probe.prepayment = prepaymentID
	probe.paidMeta = meta
	return probe.result, probe.err
}

type adminOrderProbe struct {
	order adminreport.Order
	err   error
	id    uint64
}

func (probe *adminOrderProbe) GetOrder(_ context.Context, _ uint64, id uint64) (adminreport.Order, error) {
	probe.id = id
	return probe.order, probe.err
}

func TestAdminCommandAdapterMaterializesOnlyCreatedOrder(t *testing.T) {
	reader := &adminOrderProbe{order: adminreport.Order{ID: 81, OrderNo: "ORDER-81", State: "制作中"}}
	payments := &adminPaymentProbe{result: paymentorder.ConfirmResult{State: paymentorder.ConfirmOrderCreated, OrderID: 81}}
	adapter := newAdminCommandAdapter(payments, &adminRefundProbe{}, reader)
	got, err := adapter.ProcessPending(context.Background(), adminreport.WriteMeta{ActorUserID: 7, IdempotencyKey: "materialize-1", RequestID: "request-1"}, 31, adminreport.Materialize, "")
	if err != nil || got.Order == nil || got.Order.ID != 81 || got.Refund != nil || reader.id != 81 {
		t.Fatalf("ProcessPending() = %#v, %v reader=%d", got, err, reader.id)
	}

	payments.result = paymentorder.ConfirmResult{State: paymentorder.ConfirmPending}
	if _, err := adapter.ProcessPending(context.Background(), adminreport.WriteMeta{ActorUserID: 7, IdempotencyKey: "materialize-2", RequestID: "request-2"}, 32, adminreport.Materialize, ""); !errors.Is(err, adminreport.ErrConflict) {
		t.Fatalf("pending materialization error = %v, want conflict", err)
	}
}

func TestAdminCommandAdapterRefundsOrderAndPaidPrepaymentWithoutIDConfusion(t *testing.T) {
	requestedAt := time.Date(2026, 8, 25, 1, 2, 3, 0, time.UTC)
	domainRefund := refund.Refund{ID: 901, OrderID: 81, State: refund.ProviderReady, AmountCents: 2100, RequestedAt: requestedAt}
	refunds := &adminRefundProbe{result: domainRefund}
	reader := &adminOrderProbe{order: adminreport.Order{ID: 81, OrderNo: "ORDER-81", State: "退款中", TransactionID: "TX-81"}}
	adapter := newAdminCommandAdapter(&adminPaymentProbe{}, refunds, reader)
	meta := adminreport.WriteMeta{ActorUserID: 7, IdempotencyKey: "refund-1", RequestID: "request-refund-1"}

	order, projected, err := adapter.RequestRefund(context.Background(), meta, 81, "客户取消")
	if err != nil || order.ID != 81 || projected.ID != 901 || projected.OrderID != 81 || projected.OrderNo != "ORDER-81" || refunds.orderID != 81 || reader.id != 81 || refunds.orderMeta.ActorKind != refund.ActorMerchant || refunds.orderMeta.ActorUserID != 7 {
		t.Fatalf("RequestRefund() = %#v/%#v, %v probes=%d/%d", order, projected, err, refunds.orderID, reader.id)
	}

	refunds.result = refund.Refund{ID: 902, OrderID: 0, State: refund.ProviderReady, AmountCents: 2100, RequestedAt: requestedAt}
	result, err := adapter.ProcessPending(context.Background(), meta, 51, adminreport.RefundPaid, "无法补建")
	if err != nil || result.Order != nil || result.Refund == nil || result.Refund.ID != 902 || result.Refund.OrderID != 0 || refunds.prepayment != 51 || refunds.paidMeta.ActorKind != refund.ActorMerchant || refunds.paidMeta.ActorUserID != 7 {
		t.Fatalf("paid refund result = %#v, %v prepayment=%d", result, err, refunds.prepayment)
	}
}

func TestAdminCommandAdapterMapsStableDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"invalid", paymentorder.ErrInvalidInput, adminreport.ErrInvalidInput},
		{"forbidden", refund.ErrForbidden, adminreport.ErrForbidden},
		{"not found", refund.ErrNotFound, adminreport.ErrNotFound},
		{"idempotency", refund.ErrIdempotencyConflict, adminreport.ErrConflict},
		{"transition", refund.ErrTransitionNotAllowed, adminreport.ErrConflict},
		{"unavailable", refund.ErrUnavailable, adminreport.ErrUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := newAdminCommandAdapter(&adminPaymentProbe{err: test.err}, &adminRefundProbe{}, &adminOrderProbe{})
			_, err := adapter.ProcessPending(context.Background(), adminreport.WriteMeta{ActorUserID: 7, IdempotencyKey: "key", RequestID: "request"}, 1, adminreport.Materialize, "")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}
