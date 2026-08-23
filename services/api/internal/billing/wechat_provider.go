package billing

import (
	"context"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type WeChatTransactionBillClient interface {
	DownloadTransactionBill(context.Context, time.Time) (wechatpay.TransactionBill, error)
}

// WeChatBillProvider projects verified official bill facts into reconciliation entries.
type WeChatBillProvider struct {
	client WeChatTransactionBillClient
}

func NewWeChatBillProvider(client WeChatTransactionBillClient) (*WeChatBillProvider, error) {
	if client == nil {
		return nil, ErrInvalidInput
	}
	return &WeChatBillProvider{client: client}, nil
}

func (provider *WeChatBillProvider) DownloadTransactionBill(ctx context.Context, date time.Time) (TransactionBill, error) {
	if provider == nil || provider.client == nil || date.IsZero() {
		return TransactionBill{}, ErrBillUnavailable
	}
	bill, err := provider.client.DownloadTransactionBill(ctx, date)
	if err != nil || bill.Date.IsZero() || bill.Date != normalizedBillDate(date) {
		return TransactionBill{}, ErrBillUnavailable
	}
	entries := make([]BillEntry, 0, len(bill.Entries))
	for _, entry := range bill.Entries {
		if entry.State != "SUCCESS" {
			continue
		}
		mapped := BillEntry{
			OutTradeNo: entry.OutTradeNo, OutRefundNo: entry.OutRefundNo, ProviderID: entry.ProviderID,
			AmountCents: entry.AmountCents, Currency: entry.Currency, State: entry.State, OccurredAt: entry.OccurredAt,
		}
		switch entry.Kind {
		case wechatpay.BillEntryPayment:
			mapped.Kind = EntryPayment
			if mapped.OutTradeNo == "" || mapped.OutRefundNo != "" {
				return TransactionBill{}, ErrBillUnavailable
			}
		case wechatpay.BillEntryRefund:
			mapped.Kind = EntryRefund
			if entry.OutTradeNo == "" || mapped.OutRefundNo == "" {
				return TransactionBill{}, ErrBillUnavailable
			}
			mapped.OutTradeNo = ""
		default:
			return TransactionBill{}, ErrBillUnavailable
		}
		if mapped.ProviderID == "" || mapped.AmountCents == 0 || mapped.Currency != "CNY" || mapped.OccurredAt.IsZero() {
			return TransactionBill{}, ErrBillUnavailable
		}
		entries = append(entries, mapped)
	}
	sortEntries(entries)
	return TransactionBill{Date: bill.Date, Digest: bill.Digest, Entries: entries}, nil
}

var _ BillProvider = (*WeChatBillProvider)(nil)
var _ WeChatTransactionBillClient = (*wechatpay.Client)(nil)
