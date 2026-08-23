package paymentobservation_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/paymentobservation"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

func TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "wx-app-fixture", MerchantID: "mch-fixture", OutTradeNo: "out-trade-fixture",
		TotalAmount: 3200, Currency: "CNY",
	}
	transaction := wechatpay.Transaction{
		AppID: "wx-app-fixture", MerchantID: "mch-fixture", OutTradeNo: "out-trade-fixture",
		TransactionID: "transaction-fixture", TradeState: "SUCCESS",
		SuccessTime: time.Date(2026, 8, 23, 16, 1, 2, 123456000, time.FixedZone("fixture", 8*60*60)),
		Amount:      wechatpay.TransactionAmount{Total: 3200, Currency: "CNY"},
	}

	callback, callbackErr := paymentobservation.Normalize(expected, paymentobservation.Input{
		Source: paymentobservation.SourceCallback, Transaction: transaction,
	})
	query, queryErr := paymentobservation.Normalize(expected, paymentobservation.Input{
		Source: paymentobservation.SourceActiveQuery, Transaction: transaction,
	})
	want := paymentobservation.Observation{
		DedupeKey:     "df90a7d0e1163be9db83b70456e962b380f0ccff9ec4053bc26abc49069927c8",
		Validation:    paymentobservation.ValidationAccepted,
		Mismatch:      paymentobservation.MismatchNone,
		State:         paymentobservation.StatePaid,
		OutTradeNo:    "out-trade-fixture",
		TransactionID: "transaction-fixture",
		SuccessTime:   time.Date(2026, 8, 23, 8, 1, 2, 123456000, time.UTC),
		TotalAmount:   3200,
		Currency:      "CNY",
	}

	if callbackErr != nil || queryErr != nil || !reflect.DeepEqual(callback, want) || !reflect.DeepEqual(query, want) {
		t.Fatalf("same successful provider transaction normalized differently: callback_err=%v query_err=%v", callbackErr, queryErr)
	}
}

func TestNormalizeRejectsMalformedExpectation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected paymentobservation.Expectation
	}{
		{name: "missing app id", expected: paymentobservation.Expectation{MerchantID: "mch", OutTradeNo: "out", TotalAmount: 1, Currency: "CNY"}},
		{name: "non-positive total", expected: paymentobservation.Expectation{AppID: "app", MerchantID: "mch", OutTradeNo: "out", Currency: "CNY"}},
		{name: "non-CNY expectation", expected: paymentobservation.Expectation{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 1, Currency: "USD"}},
		{name: "NUL-bearing identifier", expected: paymentobservation.Expectation{AppID: "app\x00suffix", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 1, Currency: "CNY"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			observation, err := paymentobservation.Normalize(test.expected, paymentobservation.Input{})
			var normalizationError *paymentobservation.Error
			if observation != (paymentobservation.Observation{}) || !errors.As(err, &normalizationError) ||
				normalizationError.Kind() != paymentobservation.ErrorMalformedExpectation ||
				err.Error() != "paymentobservation: MALFORMED_EXPECTATION" {
				t.Fatalf("malformed expectation error contract mismatch: observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestNormalizeReturnsStableTypedErrorsForUnusableProviderInput(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 1, Currency: "CNY",
	}
	validSuccess := wechatpay.Transaction{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TransactionID: "transaction",
		TradeState: "SUCCESS", SuccessTime: time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC),
		Amount: wechatpay.TransactionAmount{Total: 1, Currency: "CNY"},
	}
	tests := []struct {
		name  string
		input paymentobservation.Input
		kind  paymentobservation.ErrorKind
	}{
		{name: "unsupported source", input: paymentobservation.Input{Source: "OTHER", Transaction: validSuccess}, kind: paymentobservation.ErrorUnsupportedSource},
		{name: "malformed base transaction", input: paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: wechatpay.Transaction{TradeState: "NOTPAY"}}, kind: paymentobservation.ErrorMalformedInput},
		{name: "malformed successful transaction", input: paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: wechatpay.Transaction{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: "SUCCESS", SuccessTime: validSuccess.SuccessTime, Amount: validSuccess.Amount}}, kind: paymentobservation.ErrorMalformedInput},
		{name: "partial non-payment amount", input: paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: wechatpay.Transaction{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: "NOTPAY", Amount: wechatpay.TransactionAmount{Total: 1}}}, kind: paymentobservation.ErrorMalformedInput},
		{name: "malformed transaction precedes callback state", input: paymentobservation.Input{Source: paymentobservation.SourceCallback, Transaction: wechatpay.Transaction{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: "NOTPAY", Amount: wechatpay.TransactionAmount{Total: 1}}}, kind: paymentobservation.ErrorMalformedInput},
		{name: "unsupported trade state", input: paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: wechatpay.Transaction{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: "USERPAYING"}}, kind: paymentobservation.ErrorUnsupportedTradeState},
		{name: "unsupported callback state", input: paymentobservation.Input{Source: paymentobservation.SourceCallback, Transaction: wechatpay.Transaction{AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: "NOTPAY"}}, kind: paymentobservation.ErrorUnsupportedSourceState},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			observation, err := paymentobservation.Normalize(expected, test.input)
			var normalizationError *paymentobservation.Error
			if observation != (paymentobservation.Observation{}) || !errors.As(err, &normalizationError) ||
				normalizationError.Kind() != test.kind || err.Error() != "paymentobservation: "+string(test.kind) {
				t.Fatalf("provider input error contract mismatch: observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestNormalizeMapsSupportedNonPaymentQueryStates(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 1, Currency: "CNY",
	}
	tests := []struct {
		providerState string
		want          paymentobservation.Observation
	}{
		{providerState: "NOTPAY", want: paymentobservation.Observation{
			DedupeKey:  "3fbb7ecd53260879d8e309cd9774c906958ff7ff6a479aca0b620116a11f74d0",
			Validation: paymentobservation.ValidationAccepted, Mismatch: paymentobservation.MismatchNone,
			State: paymentobservation.StateNotPaid, OutTradeNo: "out",
		}},
		{providerState: "CLOSED", want: paymentobservation.Observation{
			DedupeKey:  "7510e7702ed6d445dc7eb38f1e8fa5cc1d312c66c11133dc63cf25486af4832a",
			Validation: paymentobservation.ValidationAccepted, Mismatch: paymentobservation.MismatchNone,
			State: paymentobservation.StateClosed, OutTradeNo: "out",
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.providerState, func(t *testing.T) {
			t.Parallel()
			observation, err := paymentobservation.Normalize(expected, paymentobservation.Input{
				Source: paymentobservation.SourceActiveQuery,
				Transaction: wechatpay.Transaction{
					AppID: "app", MerchantID: "mch", OutTradeNo: "out", TradeState: test.providerState,
				},
			})
			if err != nil || !reflect.DeepEqual(observation, test.want) {
				t.Fatalf("supported non-payment state mapping mismatch: err=%v observation=%+v", err, observation)
			}
		})
	}
}

func TestNormalizeDedupeSeparatesEveryCriticalFact(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 10, Currency: "CNY",
	}
	base := wechatpay.Transaction{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TransactionID: "transaction",
		TradeState: "SUCCESS", SuccessTime: time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC),
		Amount: wechatpay.TransactionAmount{Total: 10, Currency: "CNY"},
	}
	mutations := []wechatpay.Transaction{base, base, base, base, base, base, base}
	mutations[0].AppID = "other-app"
	mutations[1].MerchantID = "other-mch"
	mutations[2].OutTradeNo = "other-out"
	mutations[3].TransactionID = "other-transaction"
	mutations[4].SuccessTime = mutations[4].SuccessTime.Add(time.Nanosecond)
	mutations[5].Amount.Total++
	mutations[6].Amount.Currency = "USD"

	keys := map[string]struct{}{}
	baseObservation, err := paymentobservation.Normalize(expected, paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: base})
	if err != nil {
		t.Fatal("base observation failed")
	}
	keys[baseObservation.DedupeKey] = struct{}{}
	for _, mutation := range mutations {
		observation, err := paymentobservation.Normalize(expected, paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: mutation})
		if err != nil {
			t.Fatal("critical-fact mutation failed to normalize")
		}
		keys[observation.DedupeKey] = struct{}{}
	}
	if len(keys) != len(mutations)+1 {
		t.Fatalf("critical provider facts collided: unique_keys=%d want=%d", len(keys), len(mutations)+1)
	}
}

func TestNormalizeUsesFrozenMismatchPrecedence(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 10, Currency: "CNY",
	}
	base := wechatpay.Transaction{
		AppID: "other-app", MerchantID: "other-mch", OutTradeNo: "other-out",
		TransactionID: "transaction", TradeState: "SUCCESS",
		SuccessTime: time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC),
		Amount:      wechatpay.TransactionAmount{Total: 11, Currency: "USD"},
	}
	merchantMismatch := base
	merchantMismatch.AppID = expected.AppID
	orderMismatch := merchantMismatch
	orderMismatch.MerchantID = expected.MerchantID
	amountMismatch := orderMismatch
	amountMismatch.OutTradeNo = expected.OutTradeNo
	currencyMismatch := amountMismatch
	currencyMismatch.Amount.Total = expected.TotalAmount
	tests := []struct {
		name        string
		transaction wechatpay.Transaction
		want        paymentobservation.Mismatch
	}{
		{name: "app id precedes all", transaction: base, want: paymentobservation.MismatchAppID},
		{name: "merchant id precedes order and money", transaction: merchantMismatch, want: paymentobservation.MismatchMerchantID},
		{name: "order number precedes money", transaction: orderMismatch, want: paymentobservation.MismatchOutTradeNo},
		{name: "amount precedes currency", transaction: amountMismatch, want: paymentobservation.MismatchTotalAmount},
		{name: "currency is last", transaction: currencyMismatch, want: paymentobservation.MismatchCurrency},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			observation, err := paymentobservation.Normalize(expected, paymentobservation.Input{Source: paymentobservation.SourceActiveQuery, Transaction: test.transaction})
			if err != nil || observation.Validation != paymentobservation.ValidationRejectedMismatch || observation.Mismatch != test.want {
				t.Fatalf("mismatch precedence changed: err=%v validation=%s mismatch=%s", err, observation.Validation, observation.Mismatch)
			}
		})
	}
}

func TestNormalizeIsStableAcrossRepeatedAndConcurrentCalls(t *testing.T) {
	t.Parallel()

	expected := paymentobservation.Expectation{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TotalAmount: 10, Currency: "CNY",
	}
	input := paymentobservation.Input{Source: paymentobservation.SourceCallback, Transaction: wechatpay.Transaction{
		AppID: "app", MerchantID: "mch", OutTradeNo: "out", TransactionID: "transaction",
		TradeState: "SUCCESS", SuccessTime: time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC),
		Amount: wechatpay.TransactionAmount{Total: 10, Currency: "CNY"},
	}}
	want, err := paymentobservation.Normalize(expected, input)
	if err != nil {
		t.Fatal("baseline normalization failed")
	}

	results := make(chan paymentobservation.Observation, 32)
	for range cap(results) {
		go func() {
			observation, normalizeErr := paymentobservation.Normalize(expected, input)
			if normalizeErr != nil {
				results <- paymentobservation.Observation{}
				return
			}
			results <- observation
		}()
	}
	for range cap(results) {
		if observation := <-results; !reflect.DeepEqual(observation, want) {
			t.Fatalf("repeated normalization changed: got=%+v want=%+v", observation, want)
		}
	}
}

func TestObservationInterfaceContainsOnlyDurableProjectionFacts(t *testing.T) {
	t.Parallel()

	typeOfObservation := reflect.TypeOf(paymentobservation.Observation{})
	fields := make([]string, 0, typeOfObservation.NumField())
	for index := range typeOfObservation.NumField() {
		fields = append(fields, typeOfObservation.Field(index).Name)
	}
	want := []string{"DedupeKey", "Validation", "Mismatch", "State", "OutTradeNo", "TransactionID", "SuccessTime", "TotalAmount", "Currency"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("observation interface leaked or lost fields: got=%v want=%v", fields, want)
	}
}

func TestNormalizePersistsOnlyTheFirstSafeMismatch(t *testing.T) {
	t.Parallel()

	observation, err := paymentobservation.Normalize(paymentobservation.Expectation{
		AppID: "wx-app-fixture", MerchantID: "mch-fixture", OutTradeNo: "out-trade-fixture",
		TotalAmount: 3200, Currency: "CNY",
	}, paymentobservation.Input{
		Source: paymentobservation.SourceActiveQuery,
		Transaction: wechatpay.Transaction{
			AppID: "wx-other", MerchantID: "mch-other", OutTradeNo: "out-other",
			TransactionID: "transaction-other", TradeState: "SUCCESS",
			SuccessTime: time.Date(2026, 8, 23, 8, 1, 2, 0, time.UTC),
			Amount:      wechatpay.TransactionAmount{Total: 3199, Currency: "USD"},
		},
	})
	want := paymentobservation.Observation{
		DedupeKey:  "574da389bfb231820f63aacb413b167b796bd192656df763e10ecb4d8e92511f",
		Validation: paymentobservation.ValidationRejectedMismatch,
		Mismatch:   paymentobservation.MismatchAppID,
		State:      paymentobservation.StatePaid,
		OutTradeNo: "out-trade-fixture",
	}

	if err != nil || !reflect.DeepEqual(observation, want) {
		t.Fatalf("mismatched provider transaction was not minimized: err=%v observation=%+v", err, observation)
	}
}
