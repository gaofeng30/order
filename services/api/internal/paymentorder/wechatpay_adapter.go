package paymentorder

import (
	"context"
	"errors"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

// WeChatPayClient is the already-verified APIv3 client surface used by this module.
type WeChatPayClient interface {
	CreateJSAPIPrepay(context.Context, wechatpay.JSAPICreateRequest) (wechatpay.JSAPIPrepay, error)
	QueryTransactionByOutTradeNo(context.Context, string) (wechatpay.Transaction, error)
	CloseTransaction(context.Context, string) error
	ParseTransactionNotification([]byte, wechatpay.SignatureHeaders) (wechatpay.TransactionNotification, error)
}

// WeChatPayAdapter delegates all signing, verification and decryption to internal/wechatpay.
type WeChatPayAdapter struct {
	client     WeChatPayClient
	appID      string
	merchantID string
}

func NewWeChatPayAdapter(client WeChatPayClient, appID, merchantID string) (*WeChatPayAdapter, error) {
	if client == nil || appID == "" || merchantID == "" {
		return nil, ErrInvalidInput
	}
	return &WeChatPayAdapter{client: client, appID: appID, merchantID: merchantID}, nil
}

func (adapter *WeChatPayAdapter) CreateJSAPI(ctx context.Context, request ProviderCreateRequest) (ProviderCreateResult, error) {
	if adapter == nil || adapter.client == nil || request.AppID != adapter.appID || request.MerchantID != adapter.merchantID || !validProviderCreateRequest(request) {
		return ProviderCreateResult{}, ErrInvalidInput
	}
	result, err := adapter.client.CreateJSAPIPrepay(ctx, wechatpay.JSAPICreateRequest{
		Description: request.Description,
		OutTradeNo:  request.OutTradeNo,
		TimeExpire:  request.TimeExpire,
		NotifyURL:   request.NotifyURL,
		Amount:      wechatpay.Amount{Total: request.AmountCents, Currency: request.Currency},
		Payer:       wechatpay.Payer{OpenID: request.PayerOpenID},
	})
	if err != nil {
		return ProviderCreateResult{}, err
	}
	return ProviderCreateResult{PrepayID: result.PrepayID, RequestPayment: result.RequestPayment}, nil
}

func (adapter *WeChatPayAdapter) QueryTransaction(ctx context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	if adapter == nil || adapter.client == nil || outTradeNo == "" {
		return wechatpay.Transaction{}, ErrInvalidInput
	}
	return adapter.client.QueryTransactionByOutTradeNo(ctx, outTradeNo)
}

func (adapter *WeChatPayAdapter) CloseTransaction(ctx context.Context, outTradeNo string) error {
	if adapter == nil || adapter.client == nil || outTradeNo == "" {
		return ErrInvalidInput
	}
	err := adapter.client.CloseTransaction(ctx, outTradeNo)
	var providerError *wechatpay.Error
	if errors.As(err, &providerError) && providerError.ProviderCode() == "ORDER_NOT_EXIST" {
		return ErrNotFound
	}
	return err
}

func (adapter *WeChatPayAdapter) ParsePaymentNotification(body []byte, headers wechatpay.SignatureHeaders) (VerifiedPayment, error) {
	if adapter == nil || adapter.client == nil {
		return VerifiedPayment{}, ErrUnavailable
	}
	notification, err := adapter.client.ParseTransactionNotification(body, headers)
	if err != nil {
		return VerifiedPayment{}, err
	}
	return VerifiedPayment{
		Source: paymentobservation.SourceCallback, ProviderEventID: notification.ID,
		Transaction: notification.Transaction,
	}, nil
}

var _ PaymentProvider = (*WeChatPayAdapter)(nil)
var _ NotificationParser = (*WeChatPayAdapter)(nil)
