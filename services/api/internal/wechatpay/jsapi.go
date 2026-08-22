package wechatpay

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Amount is the provider's integer minor-unit amount.
type Amount struct {
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

// Payer identifies the provider-side Mini Program payer.
type Payer struct {
	OpenID string `json:"openid"`
}

// JSAPICreateRequest is the typed provider request for JSAPI transaction creation.
type JSAPICreateRequest struct {
	Description string
	OutTradeNo  string
	TimeExpire  string
	NotifyURL   string
	Amount      Amount
	Payer       Payer
}

// RequestPayment is the server-signed value passed unchanged to wx.requestPayment.
type RequestPayment struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// JSAPIPrepay is a verified provider prepay identifier plus signed Mini Program parameters.
type JSAPIPrepay struct {
	PrepayID       string
	RequestPayment RequestPayment
}

// CreateJSAPIPrepay creates one provider JSAPI transaction and signs wx.requestPayment parameters.
func (client *Client) CreateJSAPIPrepay(ctx context.Context, input JSAPICreateRequest) (JSAPIPrepay, error) {
	body, err := json.Marshal(struct {
		AppID       string `json:"appid"`
		MerchantID  string `json:"mchid"`
		Description string `json:"description"`
		OutTradeNo  string `json:"out_trade_no"`
		TimeExpire  string `json:"time_expire,omitempty"`
		NotifyURL   string `json:"notify_url"`
		Amount      Amount `json:"amount"`
		Payer       Payer  `json:"payer"`
	}{
		AppID: client.appID, MerchantID: client.merchantID, Description: input.Description,
		OutTradeNo: input.OutTradeNo, TimeExpire: input.TimeExpire, NotifyURL: input.NotifyURL,
		Amount: input.Amount, Payer: input.Payer,
	})
	if err != nil {
		return JSAPIPrepay{}, &Error{kind: ErrorProtocol}
	}
	responseBody, err := client.do(ctx, http.MethodPost, "/v3/pay/transactions/jsapi", body)
	if err != nil {
		return JSAPIPrepay{}, err
	}
	var response struct {
		PrepayID string `json:"prepay_id"`
	}
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil || response.PrepayID == "" || requireJSONEnd(decoder) != nil {
		return JSAPIPrepay{}, &Error{kind: ErrorProtocol}
	}

	timestamp := strconv.FormatInt(client.now().Unix(), 10)
	nonce, err := client.nonce()
	if err != nil || !safeHeaderToken(nonce) {
		return JSAPIPrepay{}, &Error{kind: ErrorProtocol}
	}
	requestPayment := RequestPayment{
		TimeStamp: timestamp, NonceStr: nonce,
		Package: "prepay_id=" + response.PrepayID, SignType: "RSA",
	}
	message := client.appID + "\n" + requestPayment.TimeStamp + "\n" + requestPayment.NonceStr + "\n" + requestPayment.Package + "\n"
	requestPayment.PaySign, err = signSHA256RSA(client.merchantPrivateKey, []byte(message))
	if err != nil {
		return JSAPIPrepay{}, err
	}
	return JSAPIPrepay{PrepayID: response.PrepayID, RequestPayment: requestPayment}, nil
}
