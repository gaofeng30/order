package main

import (
	"errors"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

var errProductionWeChatPaymentUnavailable = errors.New("production wechat payment unavailable")

type productionWeChatPayRuntime struct {
	client  *wechatpay.Client
	payment *paymentorder.WeChatPayAdapter
	config  paymentorder.Config
}

func composeProductionWeChatPayment(appID string, material config.WeChatPayMaterial) (*paymentorder.WeChatPayAdapter, paymentorder.Config, error) {
	runtime, err := composeProductionWeChatPayRuntime(appID, material)
	if err != nil {
		return nil, paymentorder.Config{}, err
	}
	return runtime.payment, runtime.config, nil
}

func composeProductionWeChatPayRuntime(appID string, material config.WeChatPayMaterial) (productionWeChatPayRuntime, error) {
	client, err := material.NewClient(appID)
	if err != nil {
		return productionWeChatPayRuntime{}, errProductionWeChatPaymentUnavailable
	}
	provider, err := paymentorder.NewWeChatPayAdapter(client, appID, material.MerchantID())
	if err != nil {
		return productionWeChatPayRuntime{}, errProductionWeChatPaymentUnavailable
	}
	return productionWeChatPayRuntime{client: client, payment: provider, config: paymentorder.Config{
		AppID: appID, MerchantID: material.MerchantID(), Description: "预约点餐",
		PaymentNotifyURL: material.NotifyURL(),
	}}, nil
}
