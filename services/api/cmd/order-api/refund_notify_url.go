package main

import (
	"errors"
	"net/url"
)

var errRefundNotifyURLUnavailable = errors.New("refund notify url unavailable")

func productionRefundNotifyURL(paymentNotifyURL string) (string, error) {
	parsed, err := url.ParseRequestURI(paymentNotifyURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "/api/v1/payments/wechat/notify" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errRefundNotifyURLUnavailable
	}
	parsed.Path = "/api/v1/refunds/wechat/notify"
	return parsed.String(), nil
}
