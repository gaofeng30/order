package orderquery

import (
	"context"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/orderproduction"
)

var (
	ErrInvalidInput    = errors.New("invalid order query")
	ErrUnauthenticated = errors.New("order query unauthenticated")
	ErrForbidden       = errors.New("order query forbidden")
	ErrNotFound        = errors.New("order not found")
	ErrUnavailable     = errors.New("order query unavailable")
)

type State = orderproduction.State

const (
	StateReserved       = orderproduction.StateReserved
	StatePreparing      = orderproduction.StatePreparing
	StateReadyForPickup = orderproduction.StateReadyForPickup
	StateCompleted      = orderproduction.StateCompleted
	StateRefunding      = orderproduction.StateRefunding
	StateRefunded       = orderproduction.StateRefunded
)

type Action string

const (
	ActionCancel Action = "CANCEL"
	ActionReady  Action = "READY"
	ActionRedeem Action = "REDEEM"
)

type Summary struct {
	ID               uint64
	OrderNo          string
	State            State
	PickupDate       string
	PickupTime       string
	PickupPoint      string
	PickupNumber     uint16
	PayableCents     uint64
	MaterializedAt   time.Time
	AvailableActions []Action
}

type Contact struct {
	Name        string
	MaskedPhone string
}

type Identity struct{ Kind string }
type Discount struct{ RatePercent uint8 }

type Item struct {
	ProductID      uint64
	Name           string
	Quantity       uint64
	UnitPriceCents uint64
	LineTotalCents uint64
	Flavors        []string
	Note           string
}

type TransitionTimes struct {
	PreparingAt time.Time
	ReadyAt     time.Time
	CompletedAt time.Time
	RefundingAt time.Time
	RefundedAt  time.Time
}

type Detail struct {
	Summary
	Contact             Contact
	Identity            Identity
	Discount            Discount
	Items               []Item
	TransactionID       string
	PaidAt              time.Time
	RedemptionToken     string
	TransitionTimes     TransitionTimes
	NotificationOptions []string
	OrderNote           string
}

type UserQuery struct {
	State   State
	Active  bool
	AfterID uint64
	Limit   uint16
}

type MerchantQuery struct {
	State   State
	Date    string
	Search  string
	AfterID uint64
	Limit   uint16
}

type Page struct {
	Orders      []Summary
	NextAfterID uint64
}

type Application interface {
	ListUser(context.Context, uint64, UserQuery) (Page, error)
	GetUser(context.Context, uint64, uint64) (Detail, error)
	SearchMerchant(context.Context, uint64, MerchantQuery) (Page, error)
	GetMerchant(context.Context, uint64, uint64) (Detail, error)
	GetMerchantAtState(context.Context, uint64, uint64, State) (Detail, error)
}
