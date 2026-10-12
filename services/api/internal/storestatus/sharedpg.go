package storestatus

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/jackc/pgx/v5/pgconn"
)

type sharedStatusExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGCore uses the fixed atomic store-status command port.
type SharedPGCore struct {
	client sharedStatusExecutor
	clock  func() time.Time
}

func NewSharedPG(client sharedStatusExecutor, clock func() time.Time) *SharedPGCore {
	return &SharedPGCore{client: client, clock: clock}
}

type sharedStatusReply struct {
	Outcome                                string `json:"outcome"`
	SQLState                               string `json:"sqlstate"`
	Result                                 Result `json:"result"`
	Actor, Account, Snapshot, Auth, Target string
	Role, TargetType, ResultValue, Reason  string
	BeforeHex, AfterHex, ResponseHex       string
}

func (core *SharedPGCore) Apply(ctx context.Context, command Command) (Result, error) {
	if !validCommand(command) {
		return Result{}, ErrInvalidCommand
	}
	if core == nil || core.client == nil || core.clock == nil {
		return Result{}, ErrUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := core.applyOnce(ctx, command)
		if err == nil {
			return result, nil
		}
		if retryableTransaction(err) && attempt == 0 {
			continue
		}
		if stableApplyError(err) {
			return Result{}, err
		}
		return Result{}, ErrUnavailable
	}
	return Result{}, ErrUnavailable
}

func (core *SharedPGCore) applyOnce(ctx context.Context, command Command) (Result, error) {
	at := core.clock().UTC().Truncate(time.Microsecond)
	if at.IsZero() {
		return Result{}, ErrUnavailable
	}
	operation := commandOperationKeyHash(command.IdempotencyKey)
	request := commandRequestIDHash(command.RequestID)
	digest, err := receiptRequestDigest(command.DesiredStatus)
	if err != nil {
		return Result{}, ErrUnavailable
	}
	var r sharedStatusReply
	err = core.client.CallRPC(ctx, "order_store_status_apply_v1", map[string]any{
		"p_actor": strconv.FormatUint(command.UserID, 10), "p_desired": string(command.DesiredStatus),
		"p_operation_hash": hex.EncodeToString(operation[:]), "p_request_hash": hex.EncodeToString(request[:]), "p_request_digest": hex.EncodeToString(digest[:]),
		"p_now": map[string]any{"epoch_seconds": strconv.FormatInt(at.Unix(), 10), "nanosecond": at.Nanosecond(), "zero": at.IsZero(), "canonical_utc": at.Format(time.RFC3339Nano)},
	}, &r)
	if err != nil {
		var pg *cloudbasepg.Error
		if errors.As(err, &pg) && (pg.SQLState == "40P01" || pg.SQLState == "40001" || pg.SQLState == "55P03") {
			return Result{}, &pgconn.PgError{Code: pg.SQLState}
		}
		return Result{}, ErrUnavailable
	}
	switch r.Outcome {
	case "APPLIED":
		if !validStatus(r.Result.Before) || !validStatus(r.Result.After) || r.Result.After != command.DesiredStatus || r.Result.Changed != (r.Result.Before != r.Result.After) {
			return Result{}, ErrUnavailable
		}
		return r.Result, nil
	case "REPLAY":
		return sharedStatusReplay(command, r)
	case "ACCOUNT_NOT_AVAILABLE":
		return Result{}, merchantidentity.ErrMerchantAccountNotAvailable
	case "FORBIDDEN":
		return Result{}, merchantidentity.ErrForbidden
	case "AUTHORIZATION_UNAVAILABLE":
		return Result{}, merchantidentity.ErrUnavailable
	case "SQL_FAILURE":
		if r.SQLState == "40P01" || r.SQLState == "40001" || r.SQLState == "55P03" {
			return Result{}, &pgconn.PgError{Code: r.SQLState}
		}
		return Result{}, ErrUnavailable
	default:
		return Result{}, ErrUnavailable
	}
}

func sharedStatusReplay(command Command, r sharedStatusReply) (Result, error) {
	parse := func(v string) (uint64, error) { return strconv.ParseUint(v, 10, 64) }
	actor, e1 := parse(r.Actor)
	account, e2 := parse(r.Account)
	snapshot, e3 := parse(r.Snapshot)
	auth, e4 := parse(r.Auth)
	target, e5 := parse(r.Target)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || actor != command.UserID || account != snapshot || auth == 0 || target != 1 || r.TargetType != "storefront_settings" || r.ResultValue != "SUCCEEDED" || (r.Role != "OWNER" && r.Role != "SUBACCOUNT") {
		return Result{}, ErrUnavailable
	}
	before, e1 := hex.DecodeString(r.BeforeHex)
	after, e2 := hex.DecodeString(r.AfterHex)
	response, e3 := hex.DecodeString(r.ResponseHex)
	if e1 != nil || e2 != nil || e3 != nil {
		return Result{}, ErrUnavailable
	}
	if err := matchRequestEvidence(before, command.DesiredStatus); err != nil {
		return Result{}, err
	}
	var result Result
	if decodeReceiptJSON(response, &result) != nil || !validStatus(result.Before) || !validStatus(result.After) || result.Changed != (result.Before != result.After) {
		return Result{}, ErrUnavailable
	}
	var state receiptAfterState
	if decodeReceiptJSON(after, &state) != nil || state.BusinessStatus != result.After {
		return Result{}, ErrUnavailable
	}
	reason := "OPERATING_STATUS_UNCHANGED"
	if result.Changed {
		reason = "OPERATING_STATUS_CHANGED"
	}
	if r.Reason != reason {
		return Result{}, ErrUnavailable
	}
	return result, nil
}
