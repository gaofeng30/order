package paymentobservation

import (
	"time"

	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

// Source identifies the trusted provider ingress that produced a transaction.
type Source string

const (
	SourceCallback    Source = "CALLBACK"
	SourceActiveQuery Source = "ACTIVE_QUERY"
)

// Expectation is the immutable local payment fact a provider transaction must match.
type Expectation struct {
	AppID       string
	MerchantID  string
	OutTradeNo  string
	TotalAmount int64
	Currency    string
}

// Input is a trusted, typed provider transaction and its ingress source.
type Input struct {
	Source      Source
	Transaction wechatpay.Transaction
}

// Validation reports whether provider facts matched the local expectation.
type Validation string

const (
	ValidationAccepted         Validation = "ACCEPTED"
	ValidationRejectedMismatch Validation = "REJECTED_MISMATCH"
)

// Mismatch is the first stable, non-sensitive business mismatch category.
type Mismatch string

const (
	MismatchNone        Mismatch = "NONE"
	MismatchAppID       Mismatch = "APP_ID"
	MismatchMerchantID  Mismatch = "MERCHANT_ID"
	MismatchOutTradeNo  Mismatch = "OUT_TRADE_NO"
	MismatchTotalAmount Mismatch = "TOTAL_AMOUNT"
	MismatchCurrency    Mismatch = "CURRENCY"
)

// State is the supported domain interpretation of a provider trade state.
type State string

const (
	StatePaid    State = "PAID"
	StateNotPaid State = "NOT_PAID"
	StateClosed  State = "CLOSED"
)

// Observation is the minimal durable payment fact produced by Normalize.
type Observation struct {
	DedupeKey     string
	Validation    Validation
	Mismatch      Mismatch
	State         State
	OutTradeNo    string
	TransactionID string
	SuccessTime   time.Time
	TotalAmount   int64
	Currency      string
}

// ErrorKind is a stable normalization failure category.
type ErrorKind string

const (
	ErrorMalformedExpectation   ErrorKind = "MALFORMED_EXPECTATION"
	ErrorMalformedInput         ErrorKind = "MALFORMED_INPUT"
	ErrorUnsupportedSource      ErrorKind = "UNSUPPORTED_SOURCE"
	ErrorUnsupportedTradeState  ErrorKind = "UNSUPPORTED_TRADE_STATE"
	ErrorUnsupportedSourceState ErrorKind = "UNSUPPORTED_SOURCE_STATE"
)

// Error contains no provider or caller values.
type Error struct {
	kind ErrorKind
}

func (err *Error) Error() string { return "paymentobservation: " + string(err.kind) }

// Kind returns the stable failure category.
func (err *Error) Kind() ErrorKind { return err.kind }
