package paymentorder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"time"
)

func (a *SharedPGApplication) replayUserCommand(ctx context.Context, meta WriteMeta, action string, target uint64) (commandReceiptResponse, bool, error) {
	key := hashOperationKey(meta.IdempotencyKey)
	raw, e := a.sharedPaymentCall(ctx, "order_payment_receipt_read_v1", map[string]any{"p_user": decimalID(meta.ActorUserID), "p_action": action, "p_key_hash": hex.EncodeToString(key[:])})
	if e != nil {
		return commandReceiptResponse{}, false, e
	}
	var hint struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(raw, &hint)
	if hint.Outcome == "ABSENT" {
		_, e = sharedPaymentObject(raw, "outcome")
		return commandReceiptResponse{}, false, e
	}
	v, e := sharedPaymentOutcome(raw, "FOUND", "evidence_text", "response_text")
	if e != nil {
		return commandReceiptResponse{}, false, e
	}
	evidence, e := sharedPaymentString(v["evidence_text"])
	if e != nil {
		return commandReceiptResponse{}, false, e
	}
	digest, e := decodeCommandEvidence([]byte(evidence))
	if e != nil {
		return commandReceiptResponse{}, false, e
	}
	want := commandRequestDigest(action, target)
	if subtle.ConstantTimeCompare(digest[:], want[:]) != 1 {
		return commandReceiptResponse{}, false, ErrIdempotencyConflict
	}
	response, e := sharedPaymentString(v["response_text"])
	if e != nil || len(response) == 0 || len(response) > maxReceiptJSONBytes {
		return commandReceiptResponse{}, false, ErrUnavailable
	}
	d := json.NewDecoder(bytes.NewReader([]byte(response)))
	d.DisallowUnknownFields()
	var result commandReceiptResponse
	if d.Decode(&result) != nil || requireJSONEnd(d) != nil || result.State == "" {
		return result, false, ErrUnavailable
	}
	return result, true, nil
}
func (a *SharedPGApplication) appendUserCommand(ctx context.Context, meta WriteMeta, action, targetType string, target uint64, response commandReceiptResponse, reason string) error {
	// target/reason are the original fixed action projection, never arbitrary wire.
	if !((action == receiptActionPrepare && targetType == receiptTargetQuote && reason == "PAYMENT_PREPARED") || (action == receiptActionConfirm && targetType == receiptTargetPrepay && reason == "PAYMENT_CONFIRMED") || (action == receiptActionManual && targetType == receiptTargetPrepay && reason == "PAYMENT_CONFIRMED")) {
		return ErrUnavailable
	}
	encoded, e := json.Marshal(response)
	if e != nil || len(encoded) > maxReceiptJSONBytes {
		return ErrUnavailable
	}
	key := hashOperationKey(meta.IdempotencyKey)
	request := sha256.Sum256([]byte(meta.RequestID))
	raw, e := a.sharedPaymentCall(ctx, "order_payment_receipt_append_v1", map[string]any{"p_user": decimalID(meta.ActorUserID), "p_action": action, "p_target": decimalID(target), "p_key_hash": hex.EncodeToString(key[:]), "p_request_hash": hex.EncodeToString(request[:]), "p_response_json": string(encoded), "p_at": a.now().UTC().Truncate(time.Microsecond)})
	if isDuplicate(e) {
		stored, found, replayErr := a.replayUserCommand(ctx, meta, action, target)
		if replayErr != nil {
			return replayErr
		}
		if !found || stored != response {
			return ErrIdempotencyConflict
		}
		return nil
	}
	if e != nil {
		return ErrUnavailable
	}
	_, e = sharedPaymentOutcome(raw, "APPENDED")
	return e
}
