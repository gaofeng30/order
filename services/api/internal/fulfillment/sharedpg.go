package fulfillment

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGApplication executes the existing fulfillment commands through fixed
// atomic RPCs, keeping token encryption and the Order identity in Go.
type SharedPGApplication struct {
	client     sharedPGExecutor
	cipher     TokenCipher
	random     io.Reader
	now        func() time.Time
	enqueueNow func() time.Time
	receiptNow func() time.Time
}

func NewSharedPGApplication(client sharedPGExecutor, cipher TokenCipher) *SharedPGApplication {
	return &SharedPGApplication{client: client, cipher: cipher, random: rand.Reader, now: time.Now, enqueueNow: time.Now, receiptNow: time.Now}
}

var _ Application = (*SharedPGApplication)(nil)

const sharedPGProofRPC = "order_redeem_order_proof_v1"
const sharedPGRedeemRPC = "order_redeem_order_v1"

type sharedPGProof struct {
	ID           string           `json:"id"`
	State        orderquery.State `json:"state"`
	Version      string           `json:"version"`
	PickupDate   string           `json:"pickup_date"`
	PickupNumber uint16           `json:"pickup_number"`
	PreparingAt  *time.Time       `json:"preparing_at"`
	ReadyAt      *time.Time       `json:"ready_at"`
	CompletedAt  *time.Time       `json:"completed_at"`
	Hash         *string          `json:"hash"`
	Ciphertext   *string          `json:"ciphertext"`
	KeyVersion   *uint16          `json:"key_version"`
	IssuedAt     *time.Time       `json:"issued_at"`
	RedeemedBy   *string          `json:"redeemed_by"`
	RedeemedAt   *time.Time       `json:"redeemed_at"`
}
type sharedPGProofResult struct {
	Proof         json.RawMessage `json:"proof"`
	ReceiptExists bool            `json:"receipt_exists"`
}
type sharedPGResult struct {
	OrderID string           `json:"order_id"`
	State   orderquery.State `json:"state"`
	Changed bool             `json:"changed"`
	Replay  bool             `json:"replay"`
}

func (a *SharedPGApplication) Execute(ctx context.Context, meta WriteMeta, command Command) (Result, error) {
	if ctx == nil || !validWrite(meta) || !validCommand(command) {
		return Result{}, ErrInvalidInput
	}
	if a == nil || a.client == nil || a.cipher == nil || a.now == nil || a.random == nil || a.enqueueNow == nil || a.receiptNow == nil {
		return Result{}, ErrUnavailable
	}
	var resolved *sharedPGCommand
	if command.Kind != CommandRedeemOrder {
		var err error
		resolved, err = a.resolveSharedPGCommand(ctx, meta, command)
		if err != nil {
			return Result{}, mapSharedPGError(err)
		}
	}
	// Two attempts preserve the existing bounded contention retry. An unknown
	// transport result may have committed; any retry keeps the exact same key.
	for attempt := 0; attempt < 2; attempt++ {
		var result Result
		var err error
		if command.Kind == CommandRedeemOrder {
			result, err = a.redeemOnce(ctx, meta, command)
		} else {
			result, err = a.fulfillOnce(ctx, meta, resolved, attempt > 0)
		}
		if err == nil {
			return result, nil
		}
		var pg *cloudbasepg.Error
		retry := errors.Is(err, cloudbasepg.ErrUnavailable)
		if errors.As(err, &pg) {
			retry = pg.SQLState == "40P01" || pg.SQLState == "40001" || pg.SQLState == "55P03" || pg.SQLState == "P1005" || pg.SQLState == "P1008"
		}
		if !retry || attempt == 1 || ctx.Err() != nil {
			return Result{}, mapSharedPGError(err)
		}
	}
	return Result{}, ErrUnavailable
}

func (a *SharedPGApplication) redeemOnce(ctx context.Context, meta WriteMeta, command Command) (Result, error) {
	actor := strconv.FormatUint(meta.ActorUserID, 10)
	id := strconv.FormatUint(command.OrderID, 10)
	var proofResult sharedPGProofResult
	if err := a.client.CallRPC(ctx, sharedPGProofRPC, map[string]any{"p_actor": actor, "p_order": id, "p_key": meta.IdempotencyKey}, &proofResult); err != nil {
		return Result{}, err
	}
	var proof sharedPGProof
	if json.Unmarshal(proofResult.Proof, &proof) != nil {
		return Result{}, ErrUnavailable
	}
	order, err := proof.order()
	if err != nil || order.id != command.OrderID {
		return Result{}, ErrUnavailable
	}
	now := a.now().UTC().Truncate(time.Microsecond)
	if now.IsZero() {
		return Result{}, ErrUnavailable
	}
	var validatedProof any // SQL NULL: no decryption has been performed.
	if !proofResult.ReceiptExists && order.state == orderquery.StateReadyForPickup {
		validator := PostgreSQLApplication{cipher: a.cipher}
		if err := validator.validateCurrentRedemption(ctx, order, now); err != nil {
			return Result{}, err
		}
		validatedProof = proofResult.Proof
	}
	var value sharedPGResult
	if err := a.client.CallRPC(ctx, sharedPGRedeemRPC, map[string]any{"p_actor": actor, "p_order": id, "p_key": meta.IdempotencyKey, "p_request_id": meta.RequestID, "p_now": now.Format(time.RFC3339Nano), "p_proof": validatedProof}, &value); err != nil {
		return Result{}, err
	}
	resultID, err := strconv.ParseUint(value.OrderID, 10, 64)
	if err != nil || resultID != command.OrderID || value.State != orderquery.StateCompleted || !value.Changed {
		return Result{}, ErrUnavailable
	}
	return Result{OrderID: resultID, State: value.State, Changed: value.Changed, Replay: value.Replay}, nil
}

func (p sharedPGProof) order() (lockedOrder, error) {
	id, e1 := strconv.ParseUint(p.ID, 10, 64)
	version, e2 := strconv.ParseUint(p.Version, 10, 64)
	if e1 != nil || e2 != nil || id == 0 || version == 0 || !validDate(p.PickupDate) || p.PickupNumber < 1 || p.PickupNumber > 9999 {
		return lockedOrder{}, ErrUnavailable
	}
	o := lockedOrder{id: id, recordVersion: version, state: p.State, pickupDate: p.PickupDate, pickupNumber: p.PickupNumber, preparingAt: nullableTime(p.PreparingAt), readyAt: nullableTime(p.ReadyAt), completedAt: nullableTime(p.CompletedAt), issuedAt: nullableTime(p.IssuedAt), redeemedAt: nullableTime(p.RedeemedAt)}
	var err error
	if p.Hash != nil {
		o.redemptionHash, err = hex.DecodeString(*p.Hash)
		if err != nil {
			return lockedOrder{}, ErrUnavailable
		}
	}
	if p.Ciphertext != nil {
		o.ciphertext, err = hex.DecodeString(*p.Ciphertext)
		if err != nil {
			return lockedOrder{}, ErrUnavailable
		}
	}
	if p.KeyVersion != nil {
		o.keyVersion = sql.NullInt64{Valid: true, Int64: int64(*p.KeyVersion)}
	}
	if p.RedeemedBy != nil {
		v, e := strconv.ParseUint(*p.RedeemedBy, 10, 64)
		if e != nil {
			return lockedOrder{}, ErrUnavailable
		}
		o.redeemedBy = sql.Null[uint64]{V: v, Valid: true}
	}
	return o, nil
}
func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func mapSharedPGError(err error) error {
	var pg *cloudbasepg.Error
	if errors.As(err, &pg) {
		switch pg.SQLState {
		case "P1001":
			return ErrInvalidInput
		case "P1002":
			return ErrForbidden
		case "P1003":
			return ErrNotFound
		case "P1004":
			return ErrTransitionNotAllowed
		case "P1006":
			return ErrIdempotencyConflict
		case "P1009":
			return ErrRedemptionInvalid
		}
	}
	switch {
	case errors.Is(err, ErrInvalidInput):
		return ErrInvalidInput
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case errors.Is(err, ErrForbidden):
		return ErrForbidden
	case errors.Is(err, ErrIdempotencyConflict):
		return ErrIdempotencyConflict
	case errors.Is(err, ErrTransitionNotAllowed):
		return ErrTransitionNotAllowed
	case errors.Is(err, ErrRedemptionInvalid):
		return ErrRedemptionInvalid
	default:
		return ErrUnavailable
	}
}
