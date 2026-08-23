package paymentorder

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"
)

const (
	receiptActionPrepare = "payment.prepare"
	receiptActionConfirm = "payment.confirm"
	receiptActionManual  = "payment.materialize_pending"
	receiptTargetQuote   = "QUOTE"
	receiptTargetPrepay  = "PREPAYMENT"
	maxReceiptJSONBytes  = 1024
)

type commandReceiptResponse struct {
	State        string `json:"state"`
	PrepaymentID string `json:"prepayment_id,omitempty"`
	OrderID      string `json:"order_id,omitempty"`
}

type commandReceiptEvidence struct {
	RequestDigest string `json:"request_digest"`
}

func commandRequestDigest(action string, targetID uint64) [32]byte {
	return sha256.Sum256([]byte(action + "\x00" + decimalID(targetID)))
}

func (service *Service) replayUserCommand(ctx context.Context, meta WriteMeta, action string, targetID uint64) (commandReceiptResponse, bool, error) {
	scope := userScopeHash(meta.ActorUserID)
	operation := hashOperationKey(meta.IdempotencyKey)
	var evidenceJSON, responseJSON []byte
	err := service.db.QueryRowContext(ctx, `
		SELECT before_state_json,response_json
		FROM action_audits
		WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='USER'
		  AND actor_scope_hash=? AND action=? AND operation_key_hash=?
	`, scope[:], action, operation[:]).Scan(&evidenceJSON, &responseJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return commandReceiptResponse{}, false, nil
	}
	if err != nil {
		return commandReceiptResponse{}, false, ErrUnavailable
	}
	want := commandRequestDigest(action, targetID)
	stored, err := decodeCommandEvidence(evidenceJSON)
	if err != nil {
		return commandReceiptResponse{}, false, ErrUnavailable
	}
	if subtle.ConstantTimeCompare(stored[:], want[:]) != 1 {
		return commandReceiptResponse{}, false, ErrIdempotencyConflict
	}
	var response commandReceiptResponse
	decoder := json.NewDecoder(bytes.NewReader(responseJSON))
	decoder.DisallowUnknownFields()
	if len(responseJSON) == 0 || len(responseJSON) > maxReceiptJSONBytes || decoder.Decode(&response) != nil || requireJSONEnd(decoder) != nil || response.State == "" {
		return commandReceiptResponse{}, false, ErrUnavailable
	}
	return response, true, nil
}

func (service *Service) appendUserCommand(ctx context.Context, meta WriteMeta, action, targetType string, targetID uint64, response commandReceiptResponse, reason string) error {
	digest := commandRequestDigest(action, targetID)
	evidenceJSON, err := json.Marshal(commandReceiptEvidence{RequestDigest: hex.EncodeToString(digest[:])})
	if err != nil {
		return ErrUnavailable
	}
	responseJSON, err := json.Marshal(response)
	if err != nil || len(responseJSON) > maxReceiptJSONBytes {
		return ErrUnavailable
	}
	scope := userScopeHash(meta.ActorUserID)
	operation := hashOperationKey(meta.IdempotencyKey)
	requestID := sha256.Sum256([]byte(meta.RequestID))
	_, err = service.db.ExecContext(ctx, `
		INSERT INTO action_audits(
			entry_kind,actor_kind,actor_scope_hash,actor_user_id,action,target_type,target_id,
			operation_key_hash,request_id_hash,result,reason_code,before_state_json,response_json,occurred_at
		) VALUES ('COMMAND_RECEIPT','USER',?,?,?,?,?,?,?,'SUCCEEDED',?,?,?,?)
	`, scope[:], meta.ActorUserID, action, targetType, targetID, operation[:], requestID[:],
		reason, evidenceJSON, responseJSON, service.now().UTC().Truncate(time.Microsecond))
	if isDuplicate(err) {
		stored, found, replayErr := service.replayUserCommand(ctx, meta, action, targetID)
		if replayErr != nil {
			return replayErr
		}
		if !found || stored != response {
			return ErrIdempotencyConflict
		}
		return nil
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

func decodeCommandEvidence(value []byte) ([32]byte, error) {
	if len(value) == 0 || len(value) > 256 {
		return [32]byte{}, ErrUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	var evidence commandReceiptEvidence
	if decoder.Decode(&evidence) != nil {
		return [32]byte{}, ErrUnavailable
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return [32]byte{}, ErrUnavailable
	}
	decoded, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != evidence.RequestDigest {
		return [32]byte{}, ErrUnavailable
	}
	var digest [32]byte
	copy(digest[:], decoded)
	return digest, nil
}

func prepareReceipt(record prepaymentRecord) commandReceiptResponse {
	return commandReceiptResponse{State: string(record.providerState), PrepaymentID: decimalID(record.id)}
}

func confirmReceipt(result ConfirmResult) commandReceiptResponse {
	response := commandReceiptResponse{State: string(result.State)}
	if result.OrderID > 0 {
		response.OrderID = decimalID(result.OrderID)
	}
	return response
}

func receiptConfirmResult(response commandReceiptResponse) (ConfirmResult, error) {
	result := ConfirmResult{State: ConfirmState(response.State)}
	if response.OrderID != "" {
		value, err := parseDecimalID(response.OrderID)
		if err != nil {
			return ConfirmResult{}, ErrUnavailable
		}
		result.OrderID = value
	}
	if result.State == ConfirmOrderCreated && result.OrderID > 0 {
		return result, nil
	}
	if result.State == ConfirmPending && result.OrderID == 0 {
		return result, nil
	}
	return ConfirmResult{}, ErrUnavailable
}

func parseDecimalID(value string) (uint64, error) {
	var result uint64
	if value == "" || value[0] == '0' {
		return 0, ErrUnavailable
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' || result > (^uint64(0)-uint64(value[index]-'0'))/10 {
			return 0, ErrUnavailable
		}
		result = result*10 + uint64(value[index]-'0')
	}
	if result == 0 || decimalID(result) != value {
		return 0, ErrUnavailable
	}
	return result, nil
}
