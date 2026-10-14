package fulfillment

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/orderquery"
)

const sharedPGCommandsProofRPC = "order_fulfillment_proof_v1"

type sharedPGCommand struct {
	kind        CommandKind
	orderID     uint64
	request     string
	date        any
	tokenHash   any
	locatorLive bool
	proof       sharedPGCommandProofResult
}

type sharedPGCommandProofResult struct {
	Proof         json.RawMessage `json:"proof"`
	ReceiptExists bool            `json:"receipt_exists"`
	LocatorLive   bool            `json:"locator_live"`
}

type sharedPGCommandProof struct {
	sharedPGProof
	UserID         string `json:"user_id"`
	OrderNumberHex string `json:"order_number_hex"`
	PickupTime     string `json:"pickup_time"`
	PickupPoint    string `json:"pickup_point"`
}

func (a *SharedPGApplication) resolveSharedPGCommand(ctx context.Context, meta WriteMeta, command Command) (*sharedPGCommand, error) {
	raw, err := json.Marshal(receiptRequestFor(command))
	if err != nil {
		return nil, ErrUnavailable
	}
	resolved := &sharedPGCommand{kind: command.Kind, request: string(raw)}
	var id, live any
	switch command.Kind {
	case CommandMarkReady:
		id, live = strconv.FormatUint(command.OrderID, 10), true
	case CommandRedeemToken:
		digest := sha256.Sum256([]byte(command.Token))
		resolved.tokenHash = hex.EncodeToString(digest[:])
	case CommandRedeemCurrentDateCode:
		now := a.now()
		if now.IsZero() {
			return nil, ErrUnavailable
		}
		resolved.date = now.In(fulfillmentLocation).Format("2006-01-02")
	default:
		return nil, ErrInvalidInput
	}
	if err := a.readCommandProof(ctx, meta, resolved, id, live); err != nil {
		return nil, err
	}
	if command.Kind == CommandMarkReady && resolved.orderID != command.OrderID {
		return nil, ErrUnavailable
	}
	return resolved, nil
}

func (a *SharedPGApplication) readCommandProof(ctx context.Context, meta WriteMeta, command *sharedPGCommand, orderID, live any) error {
	var value sharedPGCommandProofResult
	if err := a.client.CallRPC(ctx, sharedPGCommandsProofRPC, map[string]any{
		"p_actor": strconv.FormatUint(meta.ActorUserID, 10), "p_kind": string(command.kind),
		"p_order": orderID, "p_key": meta.IdempotencyKey, "p_request": command.request,
		"p_locator_date": command.date, "p_locator_live": live, "p_token_hash": command.tokenHash,
	}, &value); err != nil {
		return err
	}
	var proof sharedPGCommandProof
	if json.Unmarshal(value.Proof, &proof) != nil {
		return ErrUnavailable
	}
	order, err := proof.order()
	if err != nil || (command.orderID != 0 && order.id != command.orderID) || (live != nil && value.LocatorLive != live) {
		return ErrUnavailable
	}
	command.orderID, command.locatorLive, command.proof = order.id, value.LocatorLive, value
	return nil
}

func (a *SharedPGApplication) fulfillOnce(ctx context.Context, meta WriteMeta, command *sharedPGCommand, refresh bool) (Result, error) {
	if refresh {
		if err := a.readCommandProof(ctx, meta, command, strconv.FormatUint(command.orderID, 10), command.locatorLive); err != nil {
			return Result{}, err
		}
	}
	var proof sharedPGCommandProof
	if json.Unmarshal(command.proof.Proof, &proof) != nil {
		return Result{}, ErrUnavailable
	}
	order, err := proof.order()
	if err != nil {
		return Result{}, ErrUnavailable
	}
	var now time.Time
	if !command.proof.ReceiptExists {
		now = a.now().UTC().Truncate(time.Microsecond)
		if now.IsZero() {
			return Result{}, ErrUnavailable
		}
	}
	params := map[string]any{
		"p_actor": strconv.FormatUint(meta.ActorUserID, 10), "p_order": strconv.FormatUint(command.orderID, 10),
		"p_key": meta.IdempotencyKey, "p_request_id": meta.RequestID,
		"p_now": nil, "p_request": command.request, "p_proof": nil,
	}
	if !now.IsZero() {
		params["p_now"] = now.Format(time.RFC3339Nano)
	}
	var rpc string
	switch command.kind {
	case CommandMarkReady:
		rpc = "order_mark_ready_v1"
		params["p_token_material"] = nil
		if !command.proof.ReceiptExists {
			plain := make([]byte, 32)
			if _, err := io.ReadFull(a.random, plain); err != nil {
				return Result{}, ErrUnavailable
			}
			token := base64.RawURLEncoding.EncodeToString(plain)
			digest := sha256.Sum256([]byte(token))
			version, ciphertext, err := a.cipher.Seal(ctx, token)
			if err != nil || version == 0 || len(ciphertext) < 1 || len(ciphertext) > 192 {
				return Result{}, ErrUnavailable
			}
			params["p_token_material"] = map[string]any{"hash": hex.EncodeToString(digest[:]), "ciphertext": hex.EncodeToString(ciphertext), "key_version": version}
			params["p_proof"] = command.proof.Proof
			if (order.state == orderquery.StatePreparing || order.state == orderquery.StateReserved) && !validSharedPGReadyMessage(proof) {
				return Result{}, ErrUnavailable
			}
		}
		params["p_enqueue_at"] = nil
		if !command.proof.ReceiptExists {
			enqueueAt := a.enqueueNow().UTC().Truncate(time.Microsecond)
			if !enqueueAt.IsZero() {
				params["p_enqueue_at"] = enqueueAt.Format(time.RFC3339Nano)
			}
		}
	case CommandRedeemToken, CommandRedeemCurrentDateCode:
		params["p_locator_live"], params["p_locator_date"] = command.locatorLive, command.date
		rpc = "order_redeem_current_date_code_v1"
		if command.kind == CommandRedeemToken {
			rpc = "order_redeem_token_v1"
			params["p_token_hash"] = command.tokenHash
		}
		if !command.proof.ReceiptExists && command.locatorLive && order.state == orderquery.StateReadyForPickup {
			validator := PostgreSQLApplication{cipher: a.cipher}
			if err := validator.validateCurrentRedemption(ctx, order, now); err != nil {
				return Result{}, err
			}
			params["p_proof"] = command.proof.Proof
		}
	default:
		return Result{}, ErrInvalidInput
	}
	var value sharedPGResult
	params["p_receipt_at"] = nil
	if !command.proof.ReceiptExists {
		receiptAt := a.receiptNow().UTC().Truncate(time.Microsecond)
		if receiptAt.IsZero() {
			return Result{}, ErrUnavailable
		}
		params["p_receipt_at"] = receiptAt.Format(time.RFC3339Nano)
	}
	if err := a.client.CallRPC(ctx, rpc, params, &value); err != nil {
		return Result{}, err
	}
	id, err := strconv.ParseUint(value.OrderID, 10, 64)
	if err != nil || id != command.orderID || !value.Changed || value.State != expectedReceiptState(command.kind) {
		return Result{}, ErrUnavailable
	}
	return Result{OrderID: id, State: value.State, Changed: value.Changed, Replay: value.Replay}, nil
}

func validSharedPGReadyMessage(proof sharedPGCommandProof) bool {
	userID, err := strconv.ParseUint(proof.UserID, 10, 64)
	orderNumber, decodeErr := hex.DecodeString(proof.OrderNumberHex)
	pickupTime, clockErr := time.Parse("15:04", proof.PickupTime)
	bounded := func(value string, maximum int) bool {
		return utf8.ValidString(value) && value == strings.TrimSpace(value) && len(value) >= 1 && len(value) <= maximum
	}
	return err == nil && userID > 0 && decodeErr == nil && bounded(string(orderNumber), 64) &&
		validDate(proof.PickupDate) && clockErr == nil && pickupTime.Format("15:04") == proof.PickupTime && bounded(proof.PickupPoint, 256)
}
