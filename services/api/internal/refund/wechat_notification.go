package refund

import (
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

type weChatRefundNotificationClient interface {
	ParseRefundNotification([]byte, wechatpay.SignatureHeaders) (wechatpay.RefundNotification, error)
}

// WeChatNotificationParser maps only facts already verified and decrypted by the APIv3 client.
type WeChatNotificationParser struct {
	client weChatRefundNotificationClient
}

func NewWeChatNotificationParser(client weChatRefundNotificationClient) (*WeChatNotificationParser, error) {
	if client == nil {
		return nil, ErrInvalidInput
	}
	return &WeChatNotificationParser{client: client}, nil
}

func (parser *WeChatNotificationParser) ParseRefundNotification(body []byte, headers wechatpay.SignatureHeaders) (VerifiedRefund, error) {
	if parser == nil || parser.client == nil {
		return VerifiedRefund{}, ErrUnavailable
	}
	notification, err := parser.client.ParseRefundNotification(body, headers)
	if err != nil {
		return VerifiedRefund{}, err
	}
	if notification.Refund.Amount.Refund <= 0 || notification.Refund.Amount.Total <= 0 ||
		notification.Refund.Amount.Refund > notification.Refund.Amount.Total {
		return VerifiedRefund{}, ErrUnavailable
	}
	state := ProviderClosed
	switch notification.Refund.Status {
	case "SUCCESS":
		state = ProviderSuccess
	case "CLOSED", "ABNORMAL":
		state = ProviderClosed
	default:
		return VerifiedRefund{}, ErrUnavailable
	}
	result := VerifiedRefund{
		Source: SourceCallback, ProviderEventID: notification.ID,
		Refund: ProviderRefund{
			MerchantID: notification.MerchantID, OutTradeNo: notification.Refund.OutTradeNo,
			TransactionID: notification.Refund.TransactionID, OutRefundNo: notification.Refund.OutRefundNo,
			RefundID: notification.Refund.RefundID, State: state,
			AmountCents: uint64(notification.Refund.Amount.Refund), TotalCents: uint64(notification.Refund.Amount.Total),
			Currency: notification.Refund.Amount.Currency, SuccessTime: notification.Refund.SuccessTime.UTC(),
		},
	}
	if result.Refund.State != ProviderSuccess {
		result.Refund.SuccessTime = time.Time{}
	}
	if result.ProviderEventID == "" || !validProviderRefund(result.Refund) {
		return VerifiedRefund{}, ErrUnavailable
	}
	return result, nil
}

var _ NotificationParser = (*WeChatNotificationParser)(nil)
