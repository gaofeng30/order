package quote

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type quoteCreateReceiptResponse struct {
	QuoteID string `json:"quote_id"`
}

func validWriteMeta(meta WriteMeta) bool {
	return meta.ActorUserID > 0 && validIdempotencyKey(meta.IdempotencyKey) &&
		meta.RequestID != "" && len(meta.RequestID) <= 64 && utf8.ValidString(meta.RequestID) && strings.TrimSpace(meta.RequestID) == meta.RequestID
}

func replayReceipt(ctx context.Context, transaction *sql.Tx, store OperationReceiptStore, meta WriteMeta, action ReceiptAction, requestDigest [32]byte, response any) (bool, error) {
	if store == nil {
		return false, ErrUnavailable
	}
	receipt, found, err := store.ReplayInTx(ctx, transaction, meta, action)
	if err != nil {
		return false, normalizeReceiptError(err)
	}
	if !found {
		return false, nil
	}
	if receipt.RequestDigest != requestDigest {
		return false, ErrIdempotencyConflict
	}
	if !decodeReceiptResponse(receipt.ResponseJSON, response) {
		return false, ErrSnapshotInvalid
	}
	return true, nil
}

func appendReceipt(ctx context.Context, transaction *sql.Tx, store OperationReceiptStore, meta WriteMeta, action ReceiptAction, requestDigest [32]byte, response any) error {
	if store == nil {
		return ErrUnavailable
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return ErrUnavailable
	}
	if err := store.AppendInTx(ctx, transaction, meta, action, OperationReceipt{RequestDigest: requestDigest, ResponseJSON: encoded}); errors.Is(err, ErrOperationReceiptExists) {
		return ErrOperationReceiptExists
	} else if err != nil {
		return normalizeReceiptError(err)
	}
	return nil
}

func decodeReceiptResponse(encoded []byte, response any) bool {
	if len(encoded) == 0 || !utf8.Valid(encoded) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(response); err != nil {
		return false
	}
	return errors.Is(decoder.Decode(&struct{}{}), io.EOF)
}

func normalizeReceiptError(err error) error {
	switch {
	case errors.Is(err, ErrIdempotencyConflict):
		return ErrIdempotencyConflict
	case errors.Is(err, ErrSnapshotInvalid):
		return ErrSnapshotInvalid
	default:
		return ErrUnavailable
	}
}

func parseReceiptUint(value string) (uint64, bool) {
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil && parsed > 0 && strconv.FormatUint(parsed, 10) == value
}
