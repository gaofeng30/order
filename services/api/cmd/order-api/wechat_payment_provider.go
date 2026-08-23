package main

import (
	"errors"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
)

var errProductionWeChatPaymentUnavailable = errors.New("production wechat payment unavailable")

func composeProductionWeChatPayment(appID string, material config.WeChatPayMaterial) (*paymentorder.WeChatPayAdapter, paymentorder.Config, error) {
	client, err := material.NewClient(appID)
	if err != nil {
		return nil, paymentorder.Config{}, errProductionWeChatPaymentUnavailable
	}
	provider, err := paymentorder.NewWeChatPayAdapter(client, appID, material.MerchantID())
	if err != nil {
		return nil, paymentorder.Config{}, errProductionWeChatPaymentUnavailable
	}
	return provider, paymentorder.Config{
		AppID: appID, MerchantID: material.MerchantID(), Description: "预约点餐",
		PaymentNotifyURL: material.NotifyURL(),
	}, nil
}
