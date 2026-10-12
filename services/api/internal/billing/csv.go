package billing

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"strconv"
)

func (service *Service) ExportCSV(ctx context.Context, ownerUserID uint64, period BillingRange) (io.ReadCloser, error) {
	if !service.valid() || ownerUserID == 0 || !validRange(period) {
		return nil, ErrInvalidInput
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"order_id", "order_no", "out_trade_no", "transaction_id", "state", "amount_cents", "paid_at"}); err != nil {
		return nil, ErrUnavailable
	}
	cursor := uint64(0)
	for {
		items, next, err := service.ListPayments(ctx, ownerUserID, period, PageQuery{AfterID: cursor, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if err := writer.Write([]string{strconv.FormatUint(item.OrderID, 10), item.OrderNo, item.OutTradeNo, item.TransactionID, item.State, strconv.FormatUint(item.AmountCents, 10), item.PaidAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}); err != nil {
				return nil, ErrUnavailable
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	writer.Flush()
	if writer.Error() != nil {
		return nil, ErrUnavailable
	}
	return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}
