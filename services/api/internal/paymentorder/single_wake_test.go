package paymentorder

import (
	"context"
	"errors"
	"testing"
	"time"
)

type singleWakeRPC struct {
	t     *testing.T
	calls int
}

func (r *singleWakeRPC) CallRPC(_ context.Context, name string, args map[string]any, _ any) error {
	r.calls++
	if name != "order_payment_read_v1" || args["p_prepayment"] != "42" {
		r.t.Fatalf("unexpected targeted RPC %s %v", name, args)
	}
	return errors.New("read failed")
}
func TestRunOneDoesNotScanOrProcessOtherPayments(t *testing.T) {
	rpc := &singleWakeRPC{t: t}
	app := NewSharedPGApplication(rpc, &failedQueryProvider{}, Config{Environment: "production", AppID: "app", MerchantID: "merchant", Description: "order", PaymentNotifyURL: "https://order.suianwuye.com/api/v1/payments/wechat/notify"})
	_, err := app.RunOne(context.Background(), time.Now(), 42)
	if !errors.Is(err, ErrUnavailable) || rpc.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, rpc.calls)
	}
}
