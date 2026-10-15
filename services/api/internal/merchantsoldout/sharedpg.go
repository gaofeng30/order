package merchantsoldout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type SharedPGCommander struct {
	client sharedPGExecutor
	clock  func() time.Time
}

func NewSharedPGCommander(client sharedPGExecutor, clock func() time.Time) *SharedPGCommander {
	return &SharedPGCommander{client: client, clock: clock}
}

func (c *SharedPGCommander) SetSoldOut(ctx context.Context, meta fulfillment.WriteMeta, command fulfillment.SoldOutCommand) error {
	if ctx == nil || !validWriteMeta(meta) || !validCommandShape(command) {
		return fulfillment.ErrInvalidInput
	}
	if c == nil || c.client == nil {
		return fulfillment.ErrUnavailable
	}
	request := receiptRequest{ProductID: command.ProductID, ServiceDate: command.ServiceDate, SoldOut: *command.SoldOut}
	retried := false
	for {
		found, err := c.replay(ctx, meta, request)
		if err == nil {
			if found {
				return nil
			}
			break
		}
		if sharedSoldOutRetryable(err) && !retried {
			retried = true
			continue
		}
		return publicError(err)
	}
	if c.clock == nil {
		return fulfillment.ErrUnavailable
	}
	observedAt := c.clock()
	for {
		if observedAt.IsZero() {
			return fulfillment.ErrUnavailable
		}
		if !allowedServiceDate(request.ServiceDate, observedAt) {
			return fulfillment.ErrInvalidInput
		}
		locator, err := c.locate(ctx, request)
		if err == nil {
			err = c.execute(ctx, meta, request, locator, observedAt)
		}
		if err == nil {
			return nil
		}
		if sharedSoldOutRetryable(err) && !retried {
			retried = true
			observedAt = c.clock()
			continue
		}
		originalErr := err
		for {
			found, replayErr := c.replay(ctx, meta, request)
			if replayErr == nil {
				if found {
					return nil
				}
				return publicError(originalErr)
			}
			if sharedSoldOutRetryable(replayErr) && !retried {
				retried = true
				continue
			}
			return publicError(replayErr)
		}
	}
}

var _ fulfillment.SoldOutCommander = (*SharedPGCommander)(nil)

type sharedSoldOutResult struct {
	Outcome string `json:"outcome"`
	Receipt *struct {
		TargetType string  `json:"target_type"`
		TargetID   string  `json:"target_id"`
		Before     *string `json:"before_text"`
		After      *string `json:"after_text"`
		Response   *string `json:"response_text"`
	} `json:"receipt"`
}

func sharedSoldOutParameters(meta fulfillment.WriteMeta, request receiptRequest) map[string]any {
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	requestID := sha256.Sum256([]byte("request:" + meta.RequestID))
	digest, _ := requestDigest(request)
	return map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_product": strconv.FormatUint(request.ProductID, 10), "p_service_date": request.ServiceDate, "p_sold_out": request.SoldOut, "p_operation_hash": hex.EncodeToString(operation[:]), "p_request_id_hash": hex.EncodeToString(requestID[:]), "p_request_digest": hex.EncodeToString(digest[:])}
}
func sharedSoldOutOutcome(outcome string) error {
	switch outcome {
	case "SUCCEEDED":
		return nil
	case "FORBIDDEN":
		return fulfillment.ErrForbidden
	case "NOT_FOUND":
		return fulfillment.ErrNotFound
	case "AUTH_UNAVAILABLE":
		return merchantidentity.ErrUnavailable
	case "DUPLICATE":
		return audit.ErrDuplicateReceipt
	default:
		return fulfillment.ErrUnavailable
	}
}
func (c *SharedPGCommander) replay(ctx context.Context, meta fulfillment.WriteMeta, request receiptRequest) (bool, error) {
	var f sharedSoldOutResult
	if err := c.client.CallRPC(ctx, "order_merchant_soldout_replay_v1", sharedSoldOutParameters(meta, request), &f); err != nil {
		return false, err
	}
	if err := sharedSoldOutOutcome(f.Outcome); err != nil {
		return false, err
	}
	if f.Receipt == nil {
		return false, nil
	}
	r := f.Receipt
	if r.Before == nil || r.After == nil || r.Response == nil {
		return false, fulfillment.ErrUnavailable
	}
	if err := matchReceiptEvidence([]byte(*r.Before), request); err != nil {
		return false, err
	}
	sold, ok := decodeSoldOutState([]byte(*r.After))
	if !ok || sold != request.SoldOut {
		return false, fulfillment.ErrUnavailable
	}
	response, ok := decodeReceiptResponse([]byte(*r.Response))
	id, err := strconv.ParseUint(r.TargetID, 10, 64)
	if !ok || err != nil || strconv.FormatUint(id, 10) != r.TargetID || r.TargetType != "PRODUCT" || id != request.ProductID || response.ProductID != request.ProductID || response.ServiceDate != request.ServiceDate || response.SoldOut != request.SoldOut {
		return false, fulfillment.ErrUnavailable
	}
	return true, nil
}
func (c *SharedPGCommander) locate(ctx context.Context, request receiptRequest) (productLocator, error) {
	var f struct {
		Outcome  string  `json:"outcome"`
		Found    *bool   `json:"found"`
		Category *string `json:"category_id"`
	}
	if err := c.client.CallRPC(ctx, "order_catalog_product_v1", map[string]any{"p_product": strconv.FormatUint(request.ProductID, 10), "p_service_date": request.ServiceDate}, &f); err != nil {
		return productLocator{}, err
	}
	if f.Outcome != "SUCCEEDED" || f.Found == nil {
		return productLocator{}, fulfillment.ErrUnavailable
	}
	if !*f.Found {
		if f.Category != nil {
			return productLocator{}, fulfillment.ErrUnavailable
		}
		return productLocator{}, nil
	}
	if f.Category == nil {
		return productLocator{}, fulfillment.ErrUnavailable
	}
	id, err := strconv.ParseUint(*f.Category, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != *f.Category {
		return productLocator{}, fulfillment.ErrUnavailable
	}
	return productLocator{categoryID: id, found: true}, nil
}
func (c *SharedPGCommander) execute(ctx context.Context, meta fulfillment.WriteMeta, request receiptRequest, locator productLocator, now time.Time) error {
	p := sharedSoldOutParameters(meta, request)
	p["p_now"] = now.UTC().Truncate(time.Microsecond)
	p["p_locator_found"] = locator.found
	p["p_category"] = strconv.FormatUint(locator.categoryID, 10)
	p["p_receipt_at"] = nil
	var f sharedSoldOutResult
	if err := c.client.CallRPC(ctx, "order_merchant_soldout_command_v1", p, &f); err != nil {
		return err
	}
	if f.Outcome == "NEED_MATERIAL" {
		at := c.clock().UTC().Truncate(time.Microsecond)
		if at.IsZero() {
			return fulfillment.ErrUnavailable
		}
		p["p_receipt_at"] = at
		if err := c.client.CallRPC(ctx, "order_merchant_soldout_command_v1", p, &f); err != nil {
			return err
		}
	}
	return sharedSoldOutOutcome(f.Outcome)
}
func sharedSoldOutRetryable(err error) bool {
	if retryableTransaction(err) {
		return true
	}
	var e *cloudbasepg.Error
	return errors.As(err, &e) && (e.SQLState == "40P01" || e.SQLState == "40001" || e.SQLState == "55P03")
}
