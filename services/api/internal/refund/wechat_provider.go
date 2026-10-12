package refund

import (
	"context"
	"math"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type WeChatPayClient interface {
	CreateRefund(context.Context, wechatpay.RefundCreateRequest) (wechatpay.Refund, error)
	QueryRefund(context.Context, string) (wechatpay.Refund, error)
}

// WeChatProvider is the production APIv3 Create/Query adapter. Callback
// signature/decryption remains the HTTP ingress parser seam and is never
// performed inside the refund transaction.
type WeChatProvider struct {
	client     WeChatPayClient
	merchantID string
}

func NewWeChatProvider(client WeChatPayClient, merchantID string) (*WeChatProvider, error) {
	if client == nil || merchantID == "" || len(merchantID) > 64 {
		return nil, ErrInvalidInput
	}
	return &WeChatProvider{client: client, merchantID: merchantID}, nil
}

func (provider *WeChatProvider) CreateRefund(ctx context.Context, request ProviderCreateRequest) (ProviderRefund, error) {
	if provider == nil || provider.client == nil || !validProviderCreateRequest(request) || request.AmountCents > math.MaxInt64 || request.TotalCents > math.MaxInt64 {
		return ProviderRefund{}, ErrInvalidInput
	}
	input := wechatpay.RefundCreateRequest{
		OutRefundNo: request.OutRefundNo, Reason: request.Reason, NotifyURL: request.NotifyURL,
		Amount: wechatpay.RefundRequestAmount{Refund: int64(request.AmountCents), Total: int64(request.TotalCents), Currency: request.Currency},
	}
	if request.TransactionID != "" {
		input.TransactionID = request.TransactionID
	} else {
		input.OutTradeNo = request.OutTradeNo
	}
	result, err := provider.client.CreateRefund(ctx, input)
	if err != nil {
		return ProviderRefund{}, err
	}
	return provider.normalize(result)
}

func (provider *WeChatProvider) QueryRefund(ctx context.Context, outRefundNo string) (ProviderRefund, error) {
	if provider == nil || provider.client == nil || outRefundNo == "" || len(outRefundNo) > 64 {
		return ProviderRefund{}, ErrInvalidInput
	}
	result, err := provider.client.QueryRefund(ctx, outRefundNo)
	if err != nil {
		return ProviderRefund{}, err
	}
	return provider.normalize(result)
}

func (provider *WeChatProvider) normalize(value wechatpay.Refund) (ProviderRefund, error) {
	state := ProviderState(value.Status)
	switch value.Status {
	case "PROCESSING":
		state = ProviderProcessing
	case "SUCCESS":
		state = ProviderSuccess
	case "CLOSED", "ABNORMAL":
		state = ProviderClosed
	default:
		return ProviderRefund{}, ErrUnavailable
	}
	if value.Amount.Refund <= 0 || value.Amount.Total <= 0 {
		return ProviderRefund{}, ErrUnavailable
	}
	result := ProviderRefund{
		MerchantID: provider.merchantID, OutTradeNo: value.OutTradeNo, TransactionID: value.TransactionID,
		OutRefundNo: value.OutRefundNo, RefundID: value.RefundID, State: state,
		AmountCents: uint64(value.Amount.Refund), TotalCents: uint64(value.Amount.Total), Currency: value.Amount.Currency,
		SuccessTime: value.SuccessTime.UTC(),
	}
	if result.State != ProviderSuccess {
		result.SuccessTime = time.Time{}
	}
	if !validProviderRefund(result) {
		return ProviderRefund{}, ErrUnavailable
	}
	return result, nil
}

var _ Provider = (*WeChatProvider)(nil)
