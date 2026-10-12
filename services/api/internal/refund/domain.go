package refund

import "time"

func AdvanceProviderState(current, observed ProviderState) ProviderState {
	if current == ProviderSuccess || observed == ProviderSuccess {
		return ProviderSuccess
	}
	if current == ProviderClosed || observed == ProviderClosed {
		return ProviderClosed
	}
	if observed == ProviderProcessing {
		return ProviderProcessing
	}
	return current
}

func ValidateProviderRefund(expected ExpectedRefund, observed ProviderRefund) string {
	switch {
	case expected.MerchantID == "" || observed.MerchantID != expected.MerchantID:
		return "MERCHANT_MISMATCH"
	case expected.OutTradeNo == "" || observed.OutTradeNo != expected.OutTradeNo:
		return "OUT_TRADE_NO_MISMATCH"
	case expected.TransactionID != "" && observed.TransactionID != expected.TransactionID:
		return "TRANSACTION_ID_MISMATCH"
	case expected.OutRefundNo == "" || observed.OutRefundNo != expected.OutRefundNo:
		return "OUT_REFUND_NO_MISMATCH"
	case observed.RefundID == "":
		return "REFUND_ID_MISSING"
	case expected.AmountCents == 0 || observed.AmountCents != expected.AmountCents || observed.TotalCents != expected.AmountCents:
		return "AMOUNT_MISMATCH"
	case expected.Currency != "CNY" || observed.Currency != expected.Currency:
		return "CURRENCY_MISMATCH"
	case observed.State == ProviderSuccess && observed.SuccessTime.IsZero():
		return "SUCCESS_TIME_MISSING"
	case observed.State != ProviderProcessing && observed.State != ProviderSuccess && observed.State != ProviderClosed:
		return "PROVIDER_STATE_INVALID"
	default:
		return ""
	}
}

// A Create response is only an acceptance fact. Even when it says SUCCESS, a
// callback or active Query must produce the durable Observation that applies it.
func providerStateAfterCreate(observed ProviderState) ProviderState {
	if observed == ProviderSuccess {
		return ProviderProcessing
	}
	return observed
}

func validSuccessTime(refundingAt time.Time, observed ProviderRefund) bool {
	return observed.State != ProviderSuccess || (!observed.SuccessTime.IsZero() && !observed.SuccessTime.Before(refundingAt))
}
