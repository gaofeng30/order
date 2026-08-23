package wechatpay

import "time"

// RefundNotification is a verified strict envelope and decrypted domestic-refund resource.
type RefundNotification struct {
	ID           string
	CreateTime   time.Time
	ResourceType string
	EventType    string
	Summary      string
	MerchantID   string
	Refund       Refund
}

type refundNotificationResource struct {
	MerchantID          string                           `json:"mchid"`
	OutTradeNo          string                           `json:"out_trade_no"`
	TransactionID       string                           `json:"transaction_id"`
	OutRefundNo         string                           `json:"out_refund_no"`
	RefundID            string                           `json:"refund_id"`
	RefundStatus        string                           `json:"refund_status"`
	SuccessTime         string                           `json:"success_time"`
	UserReceivedAccount string                           `json:"user_received_account"`
	Amount              refundNotificationResourceAmount `json:"amount"`
}

type refundNotificationResourceAmount struct {
	Total       int64 `json:"total"`
	Refund      int64 `json:"refund"`
	PayerTotal  int64 `json:"payer_total"`
	PayerRefund int64 `json:"payer_refund"`
}

// ParseRefundNotification verifies the original callback bytes before strict decoding and decryption.
func (client *Client) ParseRefundNotification(body []byte, headers SignatureHeaders) (RefundNotification, error) {
	if err := client.verify(body, headers); err != nil {
		return RefundNotification{}, err
	}

	var envelope encryptedNotification
	if err := decodeStrictJSON(body, &envelope); err != nil ||
		envelope.ID == "" || envelope.ResourceType != "encrypt-resource" || envelope.Summary == "" ||
		envelope.Resource.OriginalType != "refund" || envelope.Resource.Algorithm != "AEAD_AES_256_GCM" ||
		envelope.Resource.Ciphertext == "" || envelope.Resource.Nonce == "" || !validRefundEvent(envelope.EventType) {
		return RefundNotification{}, &Error{kind: ErrorProtocol}
	}
	createTime, err := time.Parse(time.RFC3339, envelope.CreateTime)
	if err != nil {
		return RefundNotification{}, &Error{kind: ErrorProtocol}
	}
	if envelope.Resource.AssociatedData == nil {
		empty := ""
		envelope.Resource.AssociatedData = &empty
	}
	plaintext, err := client.decrypt(envelope.Resource)
	if err != nil {
		return RefundNotification{}, err
	}

	var resource refundNotificationResource
	if err := decodeStrictJSON(plaintext, &resource); err != nil || !validRefundNotificationResource(envelope.EventType, resource) {
		return RefundNotification{}, &Error{kind: ErrorProtocol}
	}
	var successTime time.Time
	if resource.SuccessTime != "" {
		successTime, err = time.Parse(time.RFC3339, resource.SuccessTime)
		if err != nil {
			return RefundNotification{}, &Error{kind: ErrorProtocol}
		}
	}

	return RefundNotification{
		ID: envelope.ID, CreateTime: createTime, ResourceType: envelope.ResourceType,
		EventType: envelope.EventType, Summary: envelope.Summary, MerchantID: resource.MerchantID,
		Refund: Refund{
			RefundID: resource.RefundID, OutRefundNo: resource.OutRefundNo,
			TransactionID: resource.TransactionID, OutTradeNo: resource.OutTradeNo,
			UserReceivedAccount: resource.UserReceivedAccount, SuccessTime: successTime, Status: resource.RefundStatus,
			Amount: RefundAmount{
				Total: resource.Amount.Total, Refund: resource.Amount.Refund,
				PayerTotal: resource.Amount.PayerTotal, PayerRefund: resource.Amount.PayerRefund, Currency: "CNY",
			},
		},
	}, nil
}

func validRefundEvent(eventType string) bool {
	switch eventType {
	case "REFUND.SUCCESS", "REFUND.CLOSED", "REFUND.ABNORMAL":
		return true
	default:
		return false
	}
}

func validRefundNotificationResource(eventType string, resource refundNotificationResource) bool {
	if resource.MerchantID == "" || resource.OutTradeNo == "" || resource.TransactionID == "" ||
		resource.OutRefundNo == "" || resource.RefundID == "" || resource.UserReceivedAccount == "" ||
		resource.Amount.Total <= 0 || resource.Amount.Refund <= 0 || resource.Amount.Refund > resource.Amount.Total ||
		resource.Amount.PayerTotal < 0 || resource.Amount.PayerRefund < 0 || resource.Amount.PayerRefund > resource.Amount.PayerTotal {
		return false
	}
	wantState := map[string]string{
		"REFUND.SUCCESS": "SUCCESS", "REFUND.CLOSED": "CLOSED", "REFUND.ABNORMAL": "ABNORMAL",
	}[eventType]
	return resource.RefundStatus == wantState && (resource.RefundStatus == "SUCCESS") == (resource.SuccessTime != "")
}
