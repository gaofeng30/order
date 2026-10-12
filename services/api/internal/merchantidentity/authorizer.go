package merchantidentity

import (
	"context"
	"database/sql"
	"strings"
)

// Actor is the request-time merchant authorization projection.
type Actor string

const (
	ActorMerchantOwner      Actor = "merchant_owner"
	ActorMerchantSubaccount Actor = "merchant_subaccount"
)

// Action is the complete T1 merchant authorization vocabulary.
type Action string

const (
	ActionOrderRead             Action = "merchant.order.read"
	ActionOrderMarkReady        Action = "merchant.order.mark_ready"
	ActionOrderRedeem           Action = "merchant.order.redeem"
	ActionProductSoldOutWrite   Action = "merchant.product.sold_out.write"
	ActionStoreStatusWrite      Action = "merchant.store.operating_status.write"
	ActionMerchantAccountManage Action = "merchant.account.manage"
)

// Target is an internal resource reference produced by the owning lane.
type Target struct {
	Type string
	ID   uint64
}

// Authorization contains only internal transaction-time authorization facts.
type Authorization struct {
	MerchantAccountID uint64
	Actor             Actor
	RecordVersion     uint64
	AuthVersion       uint64
}

// Authorizer is the transaction-bound seam consumed by merchant business lanes.
type Authorizer interface {
	AuthorizeInTx(context.Context, *sql.Tx, uint64, Action, Target) (Authorization, error)
}

var _ Authorizer = (*Repository)(nil)

// AuthorizeInTx reads and locks the current bound account inside the caller transaction.
func (repository *Repository) AuthorizeInTx(ctx context.Context, transaction *sql.Tx, userID uint64, action Action, target Target) (Authorization, error) {
	if transaction == nil || userID == 0 || !validTarget(target) || !supportedAction(action) {
		return Authorization{}, ErrUnavailable
	}
	account, found, err := readBoundAccount(ctx, transaction, userID, "FOR SHARE")
	if err != nil {
		return Authorization{}, ErrUnavailable
	}
	if !found || !account.Enabled {
		return Authorization{}, ErrMerchantAccountNotAvailable
	}
	if !validAccount(account) {
		return Authorization{}, ErrUnavailable
	}
	actor := ActorMerchantSubaccount
	if account.Role == RoleOwner {
		actor = ActorMerchantOwner
	}
	if action == ActionMerchantAccountManage && account.Role != RoleOwner {
		return Authorization{}, ErrForbidden
	}
	return Authorization{
		MerchantAccountID: account.ID,
		Actor:             actor,
		RecordVersion:     account.RecordVersion,
		AuthVersion:       account.AuthVersion,
	}, nil
}

func validTarget(target Target) bool {
	return target.ID > 0 && target.Type != "" && len(target.Type) <= 64 && strings.TrimSpace(target.Type) == target.Type
}

func supportedAction(action Action) bool {
	switch action {
	case ActionOrderRead, ActionOrderMarkReady, ActionOrderRedeem, ActionProductSoldOutWrite, ActionStoreStatusWrite, ActionMerchantAccountManage:
		return true
	default:
		return false
	}
}
