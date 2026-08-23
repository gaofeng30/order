package subscription

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrInvalidInput        = errors.New("invalid_input")
	ErrUnauthenticated     = errors.New("unauthenticated")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not_found")
	ErrIdempotencyConflict = errors.New("idempotency_conflict")
	ErrUnavailable         = errors.New("unavailable")
)

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type Kind string

const (
	KindReady        Kind = "READY"
	KindRefundResult Kind = "REFUND_RESULT"
)

type Decision string

const (
	DecisionAccepted Decision = "ACCEPTED"
	DecisionRejected Decision = "REJECTED"
)

type ConsentInput struct {
	OrderID               uint64
	Kind                  Kind
	Decision              Decision
	TemplateConfigVersion uint64
}

type Subscription struct {
	OrderID               uint64
	Kind                  Kind
	Decision              Decision
	Available             bool
	GrantSequence         uint64
	TemplateConfigVersion uint64
	DecidedAt             time.Time
}

type Message struct {
	OrderNumber  string `json:"order_number"`
	PickupDate   string `json:"pickup_date,omitempty"`
	PickupTime   string `json:"pickup_time,omitempty"`
	PickupPoint  string `json:"pickup_point,omitempty"`
	RefundResult string `json:"refund_result,omitempty"`
}

type NotificationIntent struct {
	OrderID         uint64
	RecipientUserID uint64
	Kind            Kind
	Message         Message
	AvailableAt     time.Time
}

type Delivery struct {
	OutboxID              uint64
	OrderID               uint64
	RecipientUserID       uint64
	Kind                  Kind
	Message               Message
	TemplateConfigVersion uint64
	AttemptCount          uint16
}

type SendResult struct {
	ProviderMessageID string
}

type Provider interface {
	SendSubscription(context.Context, Delivery) (SendResult, error)
}

type RunResult struct {
	Claimed         uint16
	Sent            uint16
	TemporaryFailed uint16
	PermanentFailed uint16
}

type store interface {
	recordConsent(context.Context, WriteMeta, ConsentInput, time.Time) (Subscription, error)
	enqueueInTx(context.Context, *sql.Tx, NotificationIntent, time.Time) error
	claimDue(context.Context, time.Time, uint16, [16]byte, time.Duration) ([]claimedDelivery, error)
	markSent(context.Context, claimedDelivery, SendResult, time.Time) error
	markTemporaryFailure(context.Context, claimedDelivery, string, time.Time) error
	markPermanentFailure(context.Context, claimedDelivery, string, time.Time) error
}

type claimedDelivery struct {
	Delivery
	leaseOwner    [16]byte
	recordVersion uint64
}
