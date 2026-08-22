package wechatpay

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RefundRequestAmount is the provider refund request amount in integer minor units.
type RefundRequestAmount struct {
	Refund   int64  `json:"refund"`
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

// RefundCreateRequest is the typed provider request for a domestic refund.
type RefundCreateRequest struct {
	TransactionID string
	OutTradeNo    string
	OutRefundNo   string
	Reason        string
	NotifyURL     string
	FundsAccount  string
	Amount        RefundRequestAmount
}

// RefundAmount is the verified provider refund amount.
type RefundAmount struct {
	Total            int64
	Refund           int64
	PayerTotal       int64
	PayerRefund      int64
	SettlementRefund int64
	SettlementTotal  int64
	DiscountRefund   int64
	Currency         string
}

// Refund is a verified provider refund response; acceptance is not final success.
type Refund struct {
	RefundID            string
	OutRefundNo         string
	TransactionID       string
	OutTradeNo          string
	Channel             string
	UserReceivedAccount string
	CreateTime          time.Time
	SuccessTime         time.Time
	Status              string
	FundsAccount        string
	Amount              RefundAmount
}

type refundResponse struct {
	RefundID            string               `json:"refund_id"`
	OutRefundNo         string               `json:"out_refund_no"`
	TransactionID       string               `json:"transaction_id"`
	OutTradeNo          string               `json:"out_trade_no"`
	Channel             string               `json:"channel"`
	UserReceivedAccount string               `json:"user_received_account"`
	CreateTime          string               `json:"create_time"`
	SuccessTime         string               `json:"success_time"`
	Status              string               `json:"status"`
	FundsAccount        string               `json:"funds_account"`
	Amount              refundResponseAmount `json:"amount"`
}

type refundResponseAmount struct {
	Total            int64  `json:"total"`
	Refund           int64  `json:"refund"`
	PayerTotal       int64  `json:"payer_total"`
	PayerRefund      int64  `json:"payer_refund"`
	SettlementRefund int64  `json:"settlement_refund"`
	SettlementTotal  int64  `json:"settlement_total"`
	DiscountRefund   int64  `json:"discount_refund"`
	Currency         string `json:"currency"`
}

// QueryTransactionByOutTradeNo returns one verified transaction by merchant order number.
func (client *Client) QueryTransactionByOutTradeNo(ctx context.Context, outTradeNo string) (Transaction, error) {
	requestTarget, err := client.transactionQueryTarget("/v3/pay/transactions/out-trade-no/", outTradeNo)
	if err != nil {
		return Transaction{}, err
	}
	body, err := client.do(ctx, http.MethodGet, requestTarget, nil)
	if err != nil {
		return Transaction{}, err
	}
	return decodeTransactionResponse(body)
}

// QueryTransactionByID returns one verified transaction by WeChat Pay transaction identifier.
func (client *Client) QueryTransactionByID(ctx context.Context, transactionID string) (Transaction, error) {
	requestTarget, err := client.transactionQueryTarget("/v3/pay/transactions/id/", transactionID)
	if err != nil {
		return Transaction{}, err
	}
	body, err := client.do(ctx, http.MethodGet, requestTarget, nil)
	if err != nil {
		return Transaction{}, err
	}
	return decodeTransactionResponse(body)
}

// CloseTransaction closes one transaction by merchant order number after provider verification.
func (client *Client) CloseTransaction(ctx context.Context, outTradeNo string) error {
	if outTradeNo == "" {
		return &Error{kind: ErrorProtocol}
	}
	body, err := json.Marshal(struct {
		MerchantID string `json:"mchid"`
	}{MerchantID: client.merchantID})
	if err != nil {
		return &Error{kind: ErrorProtocol}
	}
	requestTarget := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(outTradeNo) + "/close"
	_, err = client.do(ctx, http.MethodPost, requestTarget, body)
	return err
}

// CreateRefund submits one typed domestic refund and returns only its verified acceptance state.
func (client *Client) CreateRefund(ctx context.Context, input RefundCreateRequest) (Refund, error) {
	if !singleIdentifier(input.TransactionID, input.OutTradeNo) {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	body, err := json.Marshal(struct {
		TransactionID string              `json:"transaction_id,omitempty"`
		OutTradeNo    string              `json:"out_trade_no,omitempty"`
		OutRefundNo   string              `json:"out_refund_no"`
		Reason        string              `json:"reason,omitempty"`
		NotifyURL     string              `json:"notify_url,omitempty"`
		FundsAccount  string              `json:"funds_account,omitempty"`
		Amount        RefundRequestAmount `json:"amount"`
	}{
		TransactionID: input.TransactionID, OutTradeNo: input.OutTradeNo,
		OutRefundNo: input.OutRefundNo, Reason: input.Reason, NotifyURL: input.NotifyURL,
		FundsAccount: input.FundsAccount, Amount: input.Amount,
	})
	if err != nil {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	responseBody, err := client.do(ctx, http.MethodPost, "/v3/refund/domestic/refunds", body)
	if err != nil {
		return Refund{}, err
	}
	return decodeRefundResponse(responseBody)
}

// QueryRefund returns one verified provider refund by merchant refund number.
func (client *Client) QueryRefund(ctx context.Context, outRefundNo string) (Refund, error) {
	if outRefundNo == "" {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	requestTarget := "/v3/refund/domestic/refunds/" + url.PathEscape(outRefundNo)
	body, err := client.do(ctx, http.MethodGet, requestTarget, nil)
	if err != nil {
		return Refund{}, err
	}
	return decodeRefundResponse(body)
}

func (client *Client) transactionQueryTarget(prefix, identifier string) (string, error) {
	if identifier == "" {
		return "", &Error{kind: ErrorProtocol}
	}
	query := url.Values{"mchid": []string{client.merchantID}}
	return prefix + url.PathEscape(identifier) + "?" + query.Encode(), nil
}

func decodeTransactionResponse(body []byte) (Transaction, error) {
	if err := rejectDuplicateJSONKeys(body); err != nil {
		return Transaction{}, &Error{kind: ErrorProtocol}
	}
	var resource transactionResource
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&resource); err != nil || requireJSONEnd(decoder) != nil ||
		resource.AppID == "" || resource.MerchantID == "" || resource.OutTradeNo == "" || resource.TradeState == "" {
		return Transaction{}, &Error{kind: ErrorProtocol}
	}
	var successTime time.Time
	if resource.SuccessTime != "" {
		var err error
		successTime, err = time.Parse(time.RFC3339, resource.SuccessTime)
		if err != nil {
			return Transaction{}, &Error{kind: ErrorProtocol}
		}
	}
	if resource.TradeState == "SUCCESS" && (resource.TransactionID == "" || successTime.IsZero() || resource.Amount.Total <= 0 || resource.Amount.Currency == "") {
		return Transaction{}, &Error{kind: ErrorProtocol}
	}
	return Transaction{
		AppID: resource.AppID, MerchantID: resource.MerchantID, OutTradeNo: resource.OutTradeNo,
		TransactionID: resource.TransactionID, TradeType: resource.TradeType, TradeState: resource.TradeState,
		TradeStateDescription: resource.TradeStateDescription, BankType: resource.BankType,
		Attach: resource.Attach, SuccessTime: successTime, Payer: resource.Payer,
		Amount: TransactionAmount{
			Total: resource.Amount.Total, PayerTotal: resource.Amount.PayerTotal,
			Currency: resource.Amount.Currency, PayerCurrency: resource.Amount.PayerCurrency,
		},
	}, nil
}

func decodeRefundResponse(body []byte) (Refund, error) {
	if err := rejectDuplicateJSONKeys(body); err != nil {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	var response refundResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil || requireJSONEnd(decoder) != nil ||
		response.RefundID == "" || response.OutRefundNo == "" || response.Status == "" ||
		response.Amount.Refund <= 0 || response.Amount.Total <= 0 || response.Amount.Currency == "" {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	createdAt, err := time.Parse(time.RFC3339, response.CreateTime)
	if err != nil {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	var successTime time.Time
	if response.SuccessTime != "" {
		successTime, err = time.Parse(time.RFC3339, response.SuccessTime)
		if err != nil {
			return Refund{}, &Error{kind: ErrorProtocol}
		}
	}
	if response.Status == "SUCCESS" && successTime.IsZero() {
		return Refund{}, &Error{kind: ErrorProtocol}
	}
	return Refund{
		RefundID: response.RefundID, OutRefundNo: response.OutRefundNo,
		TransactionID: response.TransactionID, OutTradeNo: response.OutTradeNo,
		Channel: response.Channel, UserReceivedAccount: response.UserReceivedAccount,
		CreateTime: createdAt, SuccessTime: successTime, Status: response.Status, FundsAccount: response.FundsAccount,
		Amount: RefundAmount{
			Total: response.Amount.Total, Refund: response.Amount.Refund,
			PayerTotal: response.Amount.PayerTotal, PayerRefund: response.Amount.PayerRefund,
			SettlementRefund: response.Amount.SettlementRefund, SettlementTotal: response.Amount.SettlementTotal,
			DiscountRefund: response.Amount.DiscountRefund, Currency: response.Amount.Currency,
		},
	}, nil
}

func singleIdentifier(transactionID, outTradeNo string) bool {
	return (strings.TrimSpace(transactionID) == "") != (strings.TrimSpace(outTradeNo) == "")
}
