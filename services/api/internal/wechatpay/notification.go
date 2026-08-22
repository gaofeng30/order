package wechatpay

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"time"
)

// TransactionAmount is the verified payment amount from a transaction callback.
type TransactionAmount struct {
	Total         int64
	PayerTotal    int64
	Currency      string
	PayerCurrency string
}

// Transaction is the verified, decrypted provider payment resource.
type Transaction struct {
	AppID                 string
	MerchantID            string
	OutTradeNo            string
	TransactionID         string
	TradeType             string
	TradeState            string
	TradeStateDescription string
	BankType              string
	Attach                string
	SuccessTime           time.Time
	Payer                 Payer
	Amount                TransactionAmount
}

// TransactionNotification is a verified strict envelope and decrypted transaction resource.
type TransactionNotification struct {
	ID           string
	CreateTime   time.Time
	ResourceType string
	EventType    string
	Summary      string
	Transaction  Transaction
}

type encryptedNotification struct {
	ID           string            `json:"id"`
	CreateTime   string            `json:"create_time"`
	ResourceType string            `json:"resource_type"`
	EventType    string            `json:"event_type"`
	Summary      string            `json:"summary"`
	Resource     encryptedResource `json:"resource"`
}

type encryptedResource struct {
	OriginalType   string `json:"original_type"`
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	AssociatedData string `json:"associated_data"`
	Nonce          string `json:"nonce"`
}

type transactionResource struct {
	AppID                 string                    `json:"appid"`
	MerchantID            string                    `json:"mchid"`
	OutTradeNo            string                    `json:"out_trade_no"`
	TransactionID         string                    `json:"transaction_id"`
	TradeType             string                    `json:"trade_type"`
	TradeState            string                    `json:"trade_state"`
	TradeStateDescription string                    `json:"trade_state_desc"`
	BankType              string                    `json:"bank_type"`
	Attach                string                    `json:"attach"`
	SuccessTime           string                    `json:"success_time"`
	Payer                 Payer                     `json:"payer"`
	Amount                transactionResourceAmount `json:"amount"`
}

type transactionResourceAmount struct {
	Total         int64   `json:"total"`
	PayerTotal    *int64  `json:"payer_total"`
	Currency      string  `json:"currency"`
	PayerCurrency *string `json:"payer_currency"`
}

// ParseTransactionNotification verifies the original body before strict decoding and decryption.
func (client *Client) ParseTransactionNotification(body []byte, headers SignatureHeaders) (TransactionNotification, error) {
	if err := client.verify(body, headers); err != nil {
		return TransactionNotification{}, err
	}

	var envelope encryptedNotification
	if err := decodeStrictJSON(body, &envelope); err != nil ||
		envelope.ID == "" || envelope.ResourceType != "encrypt-resource" ||
		envelope.EventType != "TRANSACTION.SUCCESS" || envelope.Summary == "" ||
		envelope.Resource.OriginalType != "transaction" || envelope.Resource.Algorithm != "AEAD_AES_256_GCM" ||
		envelope.Resource.Ciphertext == "" || envelope.Resource.AssociatedData == "" || envelope.Resource.Nonce == "" {
		return TransactionNotification{}, &Error{kind: ErrorProtocol}
	}
	createTime, err := time.Parse(time.RFC3339, envelope.CreateTime)
	if err != nil {
		return TransactionNotification{}, &Error{kind: ErrorProtocol}
	}
	plaintext, err := client.decrypt(envelope.Resource)
	if err != nil {
		return TransactionNotification{}, err
	}

	var resource transactionResource
	if err := decodeStrictJSON(plaintext, &resource); err != nil ||
		resource.AppID == "" || resource.MerchantID == "" || resource.OutTradeNo == "" ||
		resource.TransactionID == "" || resource.TradeState != "SUCCESS" || resource.SuccessTime == "" ||
		resource.Amount.Total <= 0 || resource.Amount.Currency == "" ||
		resource.Amount.PayerTotal == nil || resource.Amount.PayerCurrency == nil || *resource.Amount.PayerCurrency == "" {
		return TransactionNotification{}, &Error{kind: ErrorProtocol}
	}
	transaction, err := transactionFromResource(resource)
	if err != nil {
		return TransactionNotification{}, err
	}
	return TransactionNotification{
		ID: envelope.ID, CreateTime: createTime, ResourceType: envelope.ResourceType,
		EventType: envelope.EventType, Summary: envelope.Summary,
		Transaction: transaction,
	}, nil
}

func transactionFromResource(resource transactionResource) (Transaction, error) {
	var successTime time.Time
	if resource.SuccessTime != "" {
		var err error
		successTime, err = time.Parse(time.RFC3339, resource.SuccessTime)
		if err != nil {
			return Transaction{}, &Error{kind: ErrorProtocol}
		}
	}
	var payerTotal int64
	if resource.Amount.PayerTotal != nil {
		payerTotal = *resource.Amount.PayerTotal
	}
	var payerCurrency string
	if resource.Amount.PayerCurrency != nil {
		payerCurrency = *resource.Amount.PayerCurrency
	}
	return Transaction{
		AppID: resource.AppID, MerchantID: resource.MerchantID, OutTradeNo: resource.OutTradeNo,
		TransactionID: resource.TransactionID, TradeType: resource.TradeType, TradeState: resource.TradeState,
		TradeStateDescription: resource.TradeStateDescription, BankType: resource.BankType,
		Attach: resource.Attach, SuccessTime: successTime, Payer: resource.Payer,
		Amount: TransactionAmount{
			Total: resource.Amount.Total, PayerTotal: payerTotal,
			Currency: resource.Amount.Currency, PayerCurrency: payerCurrency,
		},
	}, nil
}

func (client *Client) decrypt(resource encryptedResource) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(resource.Ciphertext)
	if err != nil {
		return nil, &Error{kind: ErrorDecryption}
	}
	block, err := aes.NewCipher(client.apiV3Key)
	if err != nil {
		return nil, &Error{kind: ErrorDecryption}
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(resource.Nonce) != aead.NonceSize() {
		return nil, &Error{kind: ErrorDecryption}
	}
	plaintext, err := aead.Open(nil, []byte(resource.Nonce), ciphertext, []byte(resource.AssociatedData))
	if err != nil {
		return nil, &Error{kind: ErrorDecryption}
	}
	return plaintext, nil
}
