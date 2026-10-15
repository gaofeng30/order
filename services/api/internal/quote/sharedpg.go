package quote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

type sharedPGQuoteCaller interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGApplication implements the public quote port with the fixed atomic
// functions. It does not implement the payment caller's transaction port.
type SharedPGApplication struct {
	client     sharedPGQuoteCaller
	now        func() time.Time
	receiptNow func() time.Time
}

var _ Application = (*SharedPGApplication)(nil)

func NewSharedPGApplication(client sharedPGQuoteCaller, now func() time.Time) *SharedPGApplication {
	return &SharedPGApplication{client: client, now: now, receiptNow: time.Now}
}

func (a *SharedPGApplication) Create(ctx context.Context, meta WriteMeta, rawInput CreateInput) (CreateResult, error) {
	if a == nil || a.client == nil || a.now == nil || a.receiptNow == nil || ctx == nil {
		return CreateResult{}, ErrUnavailable
	}
	if !validWriteMeta(meta) {
		return CreateResult{}, ErrInvalidInput
	}
	capturedAt := a.now()
	input, _, err := normalizeCreateInput(rawInput, capturedAt)
	if err != nil {
		return CreateResult{}, err
	}
	keyHash, requestDigest := hashIdempotencyKey(meta.ActorUserID, meta.IdempotencyKey), hashCreateInput(input)
	operationHash, requestHash := sha256.Sum256([]byte(meta.IdempotencyKey)), sha256.Sum256([]byte(meta.RequestID))
	params := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_key_hash": hex.EncodeToString(keyHash[:]), "p_request_digest": hex.EncodeToString(requestDigest[:]), "p_operation_hash": hex.EncodeToString(operationHash[:]), "p_request_id_hash": hex.EncodeToString(requestHash[:]), "p_input": sharedPGQuoteInput(input), "p_captured_at": capturedAt.UTC().Truncate(time.Microsecond), "p_receipt_at": nil}
	var response sharedPGQuoteResponse
	if err := a.client.CallRPC(ctx, "order_quote_create_v1", params, &response); err != nil {
		return CreateResult{}, ErrUnavailable
	}
	if response.Outcome == "NEED_RECEIPT" {
		params["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		response = sharedPGQuoteResponse{}
		if err := a.client.CallRPC(ctx, "order_quote_create_v1", params, &response); err != nil {
			return CreateResult{}, ErrUnavailable
		}
	}
	if response.Outcome != "CREATED" && response.Outcome != "REPLAY" {
		return CreateResult{}, sharedPGQuoteError(response.Outcome)
	}
	stored, err := decodeSharedPGQuote(response.Quote)
	if err != nil {
		return CreateResult{}, err
	}
	if stored.quote.UserID != meta.ActorUserID {
		return CreateResult{}, ErrSnapshotInvalid
	}
	if stored.requestDigest != requestDigest {
		return CreateResult{}, ErrIdempotencyConflict
	}
	if response.Outcome == "REPLAY" {
		if err := validateSharedPGQuoteReceipt(response, stored.quote.ID, requestDigest); err != nil {
			return CreateResult{}, err
		}
	}
	return CreateResult{Quote: stored.quote, Created: response.Outcome == "CREATED"}, nil
}

func (a *SharedPGApplication) Read(ctx context.Context, userID, quoteID uint64) (Quote, error) {
	if a == nil || a.client == nil || ctx == nil {
		return Quote{}, ErrUnavailable
	}
	if userID == 0 || quoteID == 0 {
		return Quote{}, ErrNotFound
	}
	var response sharedPGQuoteResponse
	if err := a.client.CallRPC(ctx, "order_quote_read_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_quote": strconv.FormatUint(quoteID, 10)}, &response); err != nil {
		return Quote{}, ErrUnavailable
	}
	if response.Outcome != "FOUND" {
		if response.Outcome == "NOT_FOUND" {
			return Quote{}, ErrNotFound
		}
		return Quote{}, sharedPGQuoteError(response.Outcome)
	}
	stored, err := decodeSharedPGQuote(response.Quote)
	if err != nil {
		return Quote{}, err
	}
	if stored.quote.ID != quoteID || stored.quote.UserID != userID {
		return Quote{}, ErrSnapshotInvalid
	}
	return stored.quote, nil
}

func sharedPGQuoteInput(input CreateInput) map[string]any {
	items := make([]map[string]any, 0, len(input.Items))
	for _, item := range input.Items {
		flavors := make([]string, 0, len(item.Flavors))
		for _, flavor := range item.Flavors {
			flavors = append(flavors, hex.EncodeToString([]byte(flavor)))
		}
		items = append(items, map[string]any{"product_id": strconv.FormatUint(item.ProductID, 10), "quantity": item.Quantity, "flavors_hex": flavors, "note_hex": hex.EncodeToString([]byte(item.Note))})
	}
	return map[string]any{"contact_name_hex": hex.EncodeToString([]byte(input.ContactName)), "pickup_date": input.PickupDate, "pickup_time": input.PickupTime, "order_note_hex": hex.EncodeToString([]byte(input.OrderNote)), "items": items}
}
