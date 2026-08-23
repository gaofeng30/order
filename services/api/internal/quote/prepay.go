package quote

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/gaofeng30/order/services/api/internal/menu"
)

const quotePrepayWindow = 10 * time.Minute

var _ PrepayTransactionSource = (*Provider)(nil)

// LoadSnapshotInTx validates and returns only the frozen quote rows in the caller's transaction.
func (provider *Provider) LoadSnapshotInTx(ctx context.Context, transaction *sql.Tx, quoteID uint64) (Quote, error) {
	if provider == nil || transaction == nil {
		return Quote{}, ErrUnavailable
	}
	if quoteID == 0 {
		return Quote{}, ErrNotFound
	}
	header, found, err := readQuoteHeader(ctx, transaction, "WHERE id=?", quoteID)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if !found {
		return Quote{}, ErrNotFound
	}
	snapshot, err := completeStoredQuote(ctx, transaction, header)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	return snapshot, nil
}

// FinalizeForPrepayInTx locks and revalidates one owned quote without repricing or writing downstream records.
func (provider *Provider) FinalizeForPrepayInTx(ctx context.Context, transaction *sql.Tx, userID, quoteID uint64, observedAt time.Time) (Quote, error) {
	if provider == nil || transaction == nil || observedAt.IsZero() {
		return Quote{}, ErrUnavailable
	}
	if userID == 0 || quoteID == 0 {
		return Quote{}, ErrNotFound
	}
	locatorHeader, found, err := readQuoteHeader(ctx, transaction, "WHERE id=? AND user_id=?", quoteID, userID)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if !found {
		return Quote{}, ErrNotFound
	}
	locator, err := completeStoredQuote(ctx, transaction, locatorHeader)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}

	userIdentity, userIdentityErr := readUserIdentity(ctx, transaction, userID)
	if userIdentityErr != nil && !errors.Is(userIdentityErr, ErrPrimaryPhoneRequired) {
		return Quote{}, ErrUnavailable
	}
	settings, err := readSourceSettings(ctx, transaction)
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	var identity resolvedIdentitySnapshot
	if userIdentityErr == nil {
		identity, err = resolveIdentitySnapshot(ctx, transaction, userIdentity, settings.WhitelistVersion)
		if err != nil {
			return Quote{}, ErrUnavailable
		}
	}

	storefront, err := readStoreSnapshot(ctx, transaction)
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	serviceDate, err := time.ParseInLocation("2006-01-02", locator.Pickup.Date, quoteLocation)
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	serviceDateOpen, err := readServiceDate(ctx, transaction, locator.Pickup.Date)
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	selection, selectionErr := readMealSelection(ctx, transaction, serviceDate, locator.Pickup.Time, observedAt)
	if selectionErr != nil && !errors.Is(selectionErr, menu.ErrInvalidSelection) {
		return Quote{}, ErrUnavailable
	}

	orderedItems := append([]ItemSnapshot(nil), locator.Items...)
	sort.Slice(orderedItems, func(left, right int) bool { return orderedItems[left].ProductID < orderedItems[right].ProductID })
	currentProducts := make(map[uint64]productRecord, len(orderedItems))
	missingProducts := make(map[uint64]bool)
	for _, item := range orderedItems {
		record, err := readProduct(ctx, transaction, item.ProductID, locator.Pickup.Date)
		if errors.Is(err, sql.ErrNoRows) {
			missingProducts[item.ProductID] = true
			continue
		}
		if err != nil {
			return Quote{}, ErrUnavailable
		}
		currentProducts[item.ProductID] = record
	}

	lockedHeader, found, err := readQuoteHeader(ctx, transaction, "WHERE id=? AND user_id=? FOR UPDATE", quoteID, userID)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if !found {
		return Quote{}, ErrNotFound
	}
	snapshot, err := completeStoredQuoteWithLock(ctx, transaction, lockedHeader, true)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if snapshot.SnapshotDigest != locator.SnapshotDigest {
		return Quote{}, ErrSnapshotInvalid
	}
	if !observedAt.Before(snapshot.ExpiresAt) {
		return Quote{}, ErrExpired
	}
	if snapshot.PayableCents < 1 {
		return Quote{}, ErrPaymentAmountTooSmall
	}
	if errors.Is(userIdentityErr, ErrPrimaryPhoneRequired) {
		return Quote{}, ErrPrimaryPhoneRequired
	}
	// Discount rate/version and whitelist source-version drift do not invalidate
	// an immutable Quote. Only a change in resolved identity semantics does.
	if identity.Snapshot.Kind != snapshot.Identity.Kind || identity.PrimaryPhone != snapshot.Contact.Phone {
		return Quote{}, ErrQuoteStale
	}
	if storefront.BusinessStatus != "open" || !serviceDateOpen || errors.Is(selectionErr, menu.ErrInvalidSelection) || string(selection.Code) != snapshot.Pickup.Meal {
		return Quote{}, ErrQuoteStale
	}
	if storefront.Snapshot != snapshot.Store || storefront.PickupPoint != snapshot.Pickup.Point || !snapshotFlavorsAvailable(snapshot.Items, storefront.FlavorOptions) {
		return Quote{}, ErrQuoteStale
	}
	if !selection.Orderable {
		return Quote{}, ErrPickupCutoffPassed
	}

	for _, item := range snapshot.Items {
		if missingProducts[item.ProductID] {
			return Quote{}, ErrItemUnavailable
		}
		record := currentProducts[item.ProductID]
		if !record.Listed || !record.CategoryActive || record.SoldOut || (record.MealPeriod != "all" && record.MealPeriod != snapshot.Pickup.Meal) {
			return Quote{}, ErrItemUnavailable
		}
		if record.PriceCents != item.OriginalUnitPriceCents || hashProductSource(record, snapshot.Pickup.Date) != item.ProductSourceVersion {
			return Quote{}, ErrQuoteStale
		}
	}
	return snapshot, nil
}
