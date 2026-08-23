// Package paymentobservation normalizes trusted provider transactions into
// deterministic domain observations before any persistence or projection.
package paymentobservation

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

// Normalize returns a deterministic observation without I/O or side effects.
func Normalize(expected Expectation, input Input) (Observation, error) {
	if malformedExpectation(expected) {
		return Observation{}, &Error{kind: ErrorMalformedExpectation}
	}
	transaction := input.Transaction
	if input.Source != SourceCallback && input.Source != SourceActiveQuery {
		return Observation{}, &Error{kind: ErrorUnsupportedSource}
	}
	if malformedBaseTransaction(transaction) {
		return Observation{}, &Error{kind: ErrorMalformedInput}
	}
	state, ok := mapState(transaction.TradeState)
	if !ok {
		return Observation{}, &Error{kind: ErrorUnsupportedTradeState}
	}
	if malformedStateFacts(transaction, state) {
		return Observation{}, &Error{kind: ErrorMalformedInput}
	}
	if input.Source == SourceCallback && state != StatePaid {
		return Observation{}, &Error{kind: ErrorUnsupportedSourceState}
	}
	successTime := canonicalSuccessTime(transaction.SuccessTime)
	validation := ValidationAccepted
	mismatch := firstMismatch(expected, transaction, state)
	if mismatch != MismatchNone {
		validation = ValidationRejectedMismatch
	}
	canonical := canonicalBytes(expected, transaction, validation, mismatch, state, successTime)
	digest := sha256.Sum256(canonical)
	observation := Observation{
		DedupeKey: hex.EncodeToString(digest[:]), Validation: validation,
		Mismatch: mismatch, State: state, OutTradeNo: expected.OutTradeNo,
	}
	if validation == ValidationRejectedMismatch || state != StatePaid {
		return observation, nil
	}
	observation.TransactionID = transaction.TransactionID
	observation.SuccessTime = successTime
	observation.TotalAmount = transaction.Amount.Total
	observation.Currency = transaction.Amount.Currency
	return observation, nil
}

func malformedBaseTransaction(transaction wechatpay.Transaction) bool {
	return transaction.AppID == "" || transaction.MerchantID == "" || transaction.OutTradeNo == "" ||
		transaction.TradeState == "" ||
		containsNUL(
			transaction.AppID,
			transaction.MerchantID,
			transaction.OutTradeNo,
			transaction.TransactionID,
			transaction.TradeType,
			transaction.TradeState,
			transaction.TradeStateDescription,
			transaction.BankType,
			transaction.Attach,
			transaction.Payer.OpenID,
			transaction.Amount.Currency,
			transaction.Amount.PayerCurrency,
		)
}

func mapState(tradeState string) (State, bool) {
	switch tradeState {
	case "SUCCESS":
		return StatePaid, true
	case "NOTPAY":
		return StateNotPaid, true
	case "CLOSED":
		return StateClosed, true
	default:
		return "", false
	}
}

func malformedStateFacts(transaction wechatpay.Transaction, state State) bool {
	if state == StatePaid {
		return transaction.TransactionID == "" || transaction.SuccessTime.IsZero() ||
			transaction.Amount.Total <= 0 || transaction.Amount.Currency == ""
	}
	amountMissing := transaction.Amount.Total == 0
	currencyMissing := transaction.Amount.Currency == ""
	return transaction.Amount.Total < 0 || amountMissing != currencyMissing
}

func canonicalSuccessTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.UTC()
}

func malformedExpectation(expected Expectation) bool {
	return expected.AppID == "" || expected.MerchantID == "" || expected.OutTradeNo == "" ||
		expected.TotalAmount <= 0 || expected.Currency != "CNY" ||
		containsNUL(expected.AppID, expected.MerchantID, expected.OutTradeNo, expected.Currency)
}

func containsNUL(values ...string) bool {
	for _, value := range values {
		if strings.ContainsRune(value, '\x00') {
			return true
		}
	}
	return false
}

func firstMismatch(expected Expectation, transaction wechatpay.Transaction, state State) Mismatch {
	switch {
	case expected.AppID != transaction.AppID:
		return MismatchAppID
	case expected.MerchantID != transaction.MerchantID:
		return MismatchMerchantID
	case expected.OutTradeNo != transaction.OutTradeNo:
		return MismatchOutTradeNo
	case (state == StatePaid || transaction.Amount.Total != 0) && expected.TotalAmount != transaction.Amount.Total:
		return MismatchTotalAmount
	case (state == StatePaid || transaction.Amount.Currency != "") && expected.Currency != transaction.Amount.Currency:
		return MismatchCurrency
	default:
		return MismatchNone
	}
}

func canonicalBytes(
	expected Expectation,
	transaction wechatpay.Transaction,
	validation Validation,
	mismatch Mismatch,
	state State,
	successTime time.Time,
) []byte {
	actualAmount := ""
	if transaction.Amount.Total != 0 {
		actualAmount = strconv.FormatInt(transaction.Amount.Total, 10)
	}
	actualSuccessTime := ""
	if !successTime.IsZero() {
		actualSuccessTime = successTime.Format(time.RFC3339Nano)
	}
	return []byte(strings.Join([]string{
		"order.payment-observation.v1",
		string(validation),
		string(mismatch),
		string(state),
		expected.AppID,
		transaction.AppID,
		expected.MerchantID,
		transaction.MerchantID,
		expected.OutTradeNo,
		transaction.OutTradeNo,
		strconv.FormatInt(expected.TotalAmount, 10),
		actualAmount,
		expected.Currency,
		transaction.Amount.Currency,
		transaction.TransactionID,
		actualSuccessTime,
	}, "\x00"))
}
