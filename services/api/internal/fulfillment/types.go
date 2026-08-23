package fulfillment

import (
	"context"
	"errors"

	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/storestatus"
)

var (
	ErrInvalidInput         = errors.New("invalid fulfillment input")
	ErrTransitionNotAllowed = errors.New("fulfillment transition not allowed")
	ErrRedemptionInvalid    = errors.New("redemption invalid")
	ErrIdempotencyConflict  = errors.New("fulfillment idempotency conflict")
	ErrForbidden            = errors.New("fulfillment forbidden")
	ErrNotFound             = errors.New("fulfillment order not found")
	ErrTokenInvalid         = errors.New("redemption token invalid")
	ErrUnavailable          = errors.New("fulfillment unavailable")
)

type WriteMeta struct {
	ActorUserID    uint64
	IdempotencyKey string
	RequestID      string
}

type CommandKind string

const (
	CommandMarkReady             CommandKind = "MARK_READY"
	CommandRedeemToken           CommandKind = "REDEEM_TOKEN"
	CommandRedeemCurrentDateCode CommandKind = "REDEEM_CURRENT_DATE_CODE"
	CommandRedeemOrder           CommandKind = "REDEEM_ORDER"
)

type Command struct {
	Kind         CommandKind
	OrderID      uint64
	Token        string
	PickupNumber string
}

type Result struct {
	OrderID uint64
	State   orderquery.State
	Changed bool
	Replay  bool
}

type Application interface {
	Execute(context.Context, WriteMeta, Command) (Result, error)
}

type MerchantOrderReader interface {
	GetMerchant(context.Context, uint64, uint64) (orderquery.Detail, error)
	GetMerchantAtState(context.Context, uint64, uint64, orderquery.State) (orderquery.Detail, error)
}

type SoldOutCommand struct {
	ProductID   uint64
	ServiceDate string
	SoldOut     *bool
}

type SoldOutCommander interface {
	SetSoldOut(context.Context, WriteMeta, SoldOutCommand) error
}

type StoreStatusCommander interface {
	Apply(context.Context, storestatus.Command) (storestatus.Result, error)
}
