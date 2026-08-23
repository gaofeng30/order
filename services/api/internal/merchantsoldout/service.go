package merchantsoldout

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const soldOutAction = string(merchantidentity.ActionProductSoldOutWrite)

var shanghaiLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// Commander is the merchant-only product sold-out writer. It shares the
// product_sold_out_dates facts read by catalog, menu, quote, and payment.
type Commander struct {
	db         *sql.DB
	authorizer merchantidentity.Authorizer
	clock      func() time.Time
}

var _ fulfillment.SoldOutCommander = (*Commander)(nil)

// New constructs the v44 MySQL sold-out command adapter.
func New(db *sql.DB, authorizer merchantidentity.Authorizer, clock func() time.Time) *Commander {
	return &Commander{db: db, authorizer: authorizer, clock: clock}
}

type productLocator struct {
	categoryID uint64
	found      bool
}

// SetSoldOut changes only the requested service-date fact and records the
// first non-PII response under the merchant command idempotency key.
func (commander *Commander) SetSoldOut(ctx context.Context, meta fulfillment.WriteMeta, command fulfillment.SoldOutCommand) error {
	if ctx == nil || !validWriteMeta(meta) || !validCommandShape(command) {
		return fulfillment.ErrInvalidInput
	}
	if commander == nil || commander.db == nil || commander.authorizer == nil {
		return fulfillment.ErrUnavailable
	}
	request := receiptRequest{
		ProductID: command.ProductID, ServiceDate: command.ServiceDate, SoldOut: *command.SoldOut,
	}
	retriedTransaction := false
	for {
		found, err := commander.replayAuthorized(ctx, meta, request)
		if err == nil {
			if found {
				return nil
			}
			break
		}
		if retryableTransaction(err) && !retriedTransaction {
			retriedTransaction = true
			continue
		}
		return publicError(err)
	}
	if commander.clock == nil {
		return fulfillment.ErrUnavailable
	}
	observedAt := commander.clock()
	for {
		if observedAt.IsZero() {
			return fulfillment.ErrUnavailable
		}
		if !allowedServiceDate(command.ServiceDate, observedAt) {
			return fulfillment.ErrInvalidInput
		}
		locator, err := commander.locateProduct(ctx, command.ProductID)
		if err == nil {
			err = commander.executeOnce(ctx, meta, request, locator)
		}
		if err == nil {
			return nil
		}
		if retryableTransaction(err) && !retriedTransaction {
			retriedTransaction = true
			observedAt = commander.clock()
			continue
		}
		originalErr := err
		for {
			found, replayErr := commander.replayAuthorized(ctx, meta, request)
			if replayErr == nil {
				if found {
					return nil
				}
				return publicError(originalErr)
			}
			if retryableTransaction(replayErr) && !retriedTransaction {
				retriedTransaction = true
				continue
			}
			return publicError(replayErr)
		}
	}
}

func (commander *Commander) locateProduct(ctx context.Context, productID uint64) (productLocator, error) {
	var categoryID uint64
	err := commander.db.QueryRowContext(ctx, `SELECT category_id FROM products WHERE id=?`, productID).Scan(&categoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return productLocator{}, nil
	}
	if err != nil {
		return productLocator{}, err
	}
	if categoryID == 0 {
		return productLocator{}, fulfillment.ErrUnavailable
	}
	return productLocator{categoryID: categoryID, found: true}, nil
}

func (commander *Commander) executeOnce(ctx context.Context, meta fulfillment.WriteMeta, request receiptRequest, locator productLocator) error {
	transaction, err := commander.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	authorization, err := commander.authorizer.AuthorizeInTx(
		ctx,
		transaction,
		meta.ActorUserID,
		merchantidentity.ActionProductSoldOutWrite,
		merchantidentity.Target{Type: "PRODUCT", ID: request.ProductID},
	)
	if err != nil {
		return err
	}
	role, ok := persistedRole(authorization.Actor)
	if !ok || authorization.MerchantAccountID == 0 || authorization.RecordVersion == 0 || authorization.AuthVersion == 0 {
		return fulfillment.ErrUnavailable
	}
	if err := lockServiceDate(ctx, transaction, request.ServiceDate); err != nil {
		return err
	}
	if !locator.found {
		return fulfillment.ErrNotFound
	}
	if err := lockProductAggregate(ctx, transaction, request.ProductID, locator.categoryID); err != nil {
		return err
	}
	currentlySoldOut, err := lockSoldOutFact(ctx, transaction, request.ServiceDate, request.ProductID)
	if err != nil {
		return err
	}
	changed := currentlySoldOut != request.SoldOut
	if changed && request.SoldOut {
		result, err := transaction.ExecContext(ctx, `INSERT INTO product_sold_out_dates(service_date,product_id) VALUES(?,?)`, request.ServiceDate, request.ProductID)
		if err != nil {
			return err
		}
		if !exactlyOne(result) {
			return fulfillment.ErrUnavailable
		}
	}
	if changed && !request.SoldOut {
		result, err := transaction.ExecContext(ctx, `DELETE FROM product_sold_out_dates WHERE service_date=? AND product_id=?`, request.ServiceDate, request.ProductID)
		if err != nil {
			return err
		}
		if !exactlyOne(result) {
			return fulfillment.ErrUnavailable
		}
	}
	response := receiptResponse{
		ProductID: request.ProductID, ServiceDate: request.ServiceDate, SoldOut: request.SoldOut,
	}
	if err := commander.appendReceiptInTx(ctx, transaction, meta, authorization, role, request, response, currentlySoldOut, changed); err != nil {
		return err
	}
	return transaction.Commit()
}

func lockServiceDate(ctx context.Context, transaction *sql.Tx, serviceDate string) error {
	var recordVersion uint64
	err := transaction.QueryRowContext(ctx, `SELECT record_version FROM service_dates WHERE service_date=? FOR UPDATE`, serviceDate).Scan(&recordVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return fulfillment.ErrNotFound
	}
	if err != nil {
		return err
	}
	if recordVersion == 0 {
		return fulfillment.ErrUnavailable
	}
	return nil
}

func lockProductAggregate(ctx context.Context, transaction *sql.Tx, productID, categoryID uint64) error {
	var categoryVersion uint64
	err := transaction.QueryRowContext(ctx, `SELECT record_version FROM categories WHERE id=? FOR UPDATE`, categoryID).Scan(&categoryVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return fulfillment.ErrNotFound
	}
	if err != nil {
		return err
	}
	if categoryVersion == 0 {
		return fulfillment.ErrUnavailable
	}
	var lockedCategoryID, productVersion uint64
	err = transaction.QueryRowContext(ctx, `SELECT category_id,record_version FROM products WHERE id=? FOR UPDATE`, productID).Scan(&lockedCategoryID, &productVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return fulfillment.ErrNotFound
	}
	if err != nil {
		return err
	}
	if lockedCategoryID != categoryID || productVersion == 0 {
		return fulfillment.ErrUnavailable
	}
	return nil
}

func lockSoldOutFact(ctx context.Context, transaction *sql.Tx, serviceDate string, productID uint64) (bool, error) {
	var lockedProductID uint64
	err := transaction.QueryRowContext(ctx, `SELECT product_id FROM product_sold_out_dates WHERE service_date=? AND product_id=? FOR UPDATE`, serviceDate, productID).Scan(&lockedProductID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if lockedProductID != productID {
		return false, fulfillment.ErrUnavailable
	}
	return true, nil
}

func validWriteMeta(meta fulfillment.WriteMeta) bool {
	return meta.ActorUserID > 0 && printable(meta.IdempotencyKey, 128) && printable(meta.RequestID, 64)
}

func validCommandShape(command fulfillment.SoldOutCommand) bool {
	if command.ProductID == 0 || command.SoldOut == nil {
		return false
	}
	parsed, err := time.Parse("2006-01-02", command.ServiceDate)
	return err == nil && parsed.Format("2006-01-02") == command.ServiceDate
}

func allowedServiceDate(serviceDate string, now time.Time) bool {
	if now.IsZero() {
		return false
	}
	local := now.In(shanghaiLocation)
	return serviceDate == local.Format("2006-01-02") || serviceDate == local.AddDate(0, 0, 1).Format("2006-01-02")
}

func printable(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func exactlyOne(result sql.Result) bool {
	if result == nil {
		return false
	}
	rows, err := result.RowsAffected()
	return err == nil && rows == 1
}

func retryableTransaction(err error) bool {
	if errors.Is(err, merchantidentity.ErrUnavailable) {
		return true
	}
	var mysqlError *mysqlDriver.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1205 || mysqlError.Number == 1213)
}

func publicError(err error) error {
	switch {
	case errors.Is(err, fulfillment.ErrInvalidInput), errors.Is(err, fulfillment.ErrIdempotencyConflict), errors.Is(err, fulfillment.ErrForbidden), errors.Is(err, fulfillment.ErrNotFound):
		return err
	case errors.Is(err, audit.ErrIdempotencyConflict):
		return fulfillment.ErrIdempotencyConflict
	case errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable), errors.Is(err, merchantidentity.ErrForbidden):
		return fulfillment.ErrForbidden
	default:
		return fulfillment.ErrUnavailable
	}
}
