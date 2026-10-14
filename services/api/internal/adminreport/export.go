package adminreport

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

type exportPayment struct {
	ID                    uint64
	OrderNo, PickupNumber string
	PaidAt                time.Time
	TransactionID         string
	PayableCents          uint64
}

// Retain the complete CSV, but never hydrate or retain unused order items.
func exportCSV(ctx context.Context, read func(context.Context, uint64) ([]exportPayment, error)) (io.ReadCloser, error) {
	var b strings.Builder
	b.WriteString("\ufeff")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"订单号", "取餐号", "支付时间", "微信交易号", "实付金额"})
	cursor := uint64(0)
	for {
		items, err := read(ctx, cursor)
		if err != nil {
			return nil, err
		}
		for _, p := range items {
			if p.ID <= cursor {
				return nil, ErrUnavailable
			}
			cursor = p.ID
			_ = w.Write([]string{p.OrderNo, p.PickupNumber, p.PaidAt.Format(time.RFC3339), p.TransactionID, fmt.Sprintf("%d.%02d", p.PayableCents/100, p.PayableCents%100)})
		}
		if len(items) < 100 {
			break
		}
	}
	w.Flush()
	if w.Error() != nil {
		return nil, ErrUnavailable
	}
	return io.NopCloser(strings.NewReader(b.String())), nil
}
