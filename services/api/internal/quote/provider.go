package quote

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/quotepricing"
	"github.com/gaofeng30/order/services/api/internal/staffidentity"
	"golang.org/x/text/unicode/norm"
)

const (
	quoteLockSeconds = 5
	quoteTimezone    = "Asia/Shanghai"
)

var quoteLocation = time.FixedZone(quoteTimezone, 8*60*60)

// Provider is the MySQL implementation behind the deep quote interface.
type Provider struct {
	db           *sql.DB
	receipts     OperationReceiptStore
	now          func() time.Time
	beforeCommit func(*sql.Tx) error
}

var _ Application = (*Provider)(nil)

// NewProvider constructs an immutable quote provider over the shared pool.
func NewProvider(db *sql.DB, receipts OperationReceiptStore, now func() time.Time) *Provider {
	return &Provider{db: db, receipts: receipts, now: now}
}

type sourceSettings struct {
	RatePercent      int64
	DiscountVersion  uint64
	WhitelistVersion uint64
}

type resolvedIdentitySnapshot struct {
	Snapshot     IdentitySnapshot
	PrimaryPhone string
}

type userIdentityRecord struct {
	PrimaryPhone string
	Extra        *staffidentity.ExtraClaim
}

type productRecord struct {
	ID             uint64
	CategoryID     uint64
	Name           string
	PriceCents     int64
	MealPeriod     string
	Listed         bool
	CategoryActive bool
	SoldOut        bool
	ImageObjectKey string
}

type storefrontFacts struct {
	Snapshot       StoreSnapshot
	PickupPoint    string
	BusinessStatus string
	FlavorOptions  map[string]struct{}
}

// Create freezes one complete server-owned snapshot or returns an exact replay.
func (provider *Provider) Create(ctx context.Context, meta WriteMeta, rawInput CreateInput) (result CreateResult, returnErr error) {
	if provider == nil || provider.db == nil || provider.receipts == nil || provider.now == nil {
		return CreateResult{}, ErrUnavailable
	}
	if !validWriteMeta(meta) {
		return CreateResult{}, ErrInvalidInput
	}
	userID := meta.ActorUserID
	key := meta.IdempotencyKey
	now := provider.now()
	input, serviceDate, err := normalizeCreateInput(rawInput, now)
	if err != nil {
		return CreateResult{}, err
	}
	keyHash := hashIdempotencyKey(userID, key)
	requestDigest := hashCreateInput(input)
	lockName := quoteLockName(keyHash)

	connection, err := provider.db.Conn(ctx)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	defer connection.Close()
	locked, err := acquireQuoteLock(ctx, connection, lockName)
	if err != nil || !locked {
		return CreateResult{}, ErrUnavailable
	}
	lockHeld := true
	releaseLock := func() error {
		if !lockHeld {
			return nil
		}
		releaseContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := releaseQuoteLock(releaseContext, connection, lockName); err != nil {
			return err
		}
		lockHeld = false
		return nil
	}
	defer func() {
		if err := releaseLock(); err != nil && returnErr == nil {
			result = CreateResult{}
			returnErr = ErrUnavailable
		}
	}()

	transaction, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	// The connection-scoped named lock already serializes this exact owner/key.
	// Avoid a missing-row next-key lock here: different new keys for one owner
	// must be able to insert concurrently without forming an InnoDB gap-lock
	// deadlock.
	existing, found, err := readQuoteHeader(ctx, transaction, "WHERE user_id=? AND idempotency_key_hash=?", userID, keyHash[:])
	if err != nil {
		return CreateResult{}, normalizeSnapshotReadError(err)
	}
	if found {
		if existing.requestDigest != requestDigest {
			return CreateResult{}, ErrIdempotencyConflict
		}
		if err := transaction.Rollback(); err != nil {
			return CreateResult{}, ErrUnavailable
		}
		if err := releaseLock(); err != nil {
			return CreateResult{}, ErrUnavailable
		}
		return provider.replayCreatedQuote(ctx, meta, keyHash, requestDigest)
	}

	userIdentity, err := readUserIdentity(ctx, transaction, userID)
	if errors.Is(err, ErrPrimaryPhoneRequired) {
		return CreateResult{}, err
	}
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	settings, err := readSourceSettings(ctx, transaction)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	identitySnapshot, err := resolveIdentitySnapshot(ctx, transaction, userIdentity, settings.WhitelistVersion)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	storefront, err := readStoreSnapshot(ctx, transaction)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	if storefront.BusinessStatus != "open" {
		return CreateResult{}, ErrSelectionUnavailable
	}
	if !selectedFlavorsAvailable(input.Items, storefront.FlavorOptions) {
		return CreateResult{}, ErrSelectionUnavailable
	}
	serviceDateOpen, err := readServiceDate(ctx, transaction, input.PickupDate)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	if !serviceDateOpen {
		return CreateResult{}, ErrSelectionUnavailable
	}
	mealSelection, err := readMealSelection(ctx, transaction, serviceDate, input.PickupTime, now)
	if errors.Is(err, menu.ErrInvalidSelection) || (err == nil && !mealSelection.Orderable) {
		return CreateResult{}, ErrSelectionUnavailable
	}
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}

	recordsByID := make(map[uint64]productRecord, len(input.Items))
	orderedItems := append([]ItemInput(nil), input.Items...)
	sort.Slice(orderedItems, func(left, right int) bool { return orderedItems[left].ProductID < orderedItems[right].ProductID })
	for _, item := range orderedItems {
		record, err := readProduct(ctx, transaction, item.ProductID, input.PickupDate)
		if errors.Is(err, sql.ErrNoRows) {
			return CreateResult{}, ErrSelectionUnavailable
		}
		if err != nil {
			return CreateResult{}, ErrUnavailable
		}
		if !record.Listed || !record.CategoryActive || record.SoldOut || (record.MealPeriod != "all" && record.MealPeriod != string(mealSelection.Code)) {
			return CreateResult{}, ErrSelectionUnavailable
		}
		recordsByID[item.ProductID] = record
	}
	records := make([]productRecord, 0, len(input.Items))
	pricingLines := make([]quotepricing.Line, 0, len(input.Items))
	for _, item := range input.Items {
		record := recordsByID[item.ProductID]
		records = append(records, record)
		pricingLines = append(pricingLines, quotepricing.Line{UnitPriceCents: record.PriceCents, Quantity: item.Quantity})
	}
	ratePercent := int64(100)
	if identitySnapshot.Snapshot.Kind == IdentityStaff {
		ratePercent = settings.RatePercent
	}
	pricing, err := quotepricing.Calculate(quotepricing.Input{RatePercent: ratePercent, Lines: pricingLines})
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	if pricing.PayableCents < 1 {
		return CreateResult{}, ErrPaymentAmountTooSmall
	}
	createdAt := now.UTC().Truncate(time.Microsecond)
	quote := Quote{
		UserID:                userID,
		Contact:               ContactSnapshot{Name: input.ContactName, Phone: identitySnapshot.PrimaryPhone},
		Identity:              identitySnapshot.Snapshot,
		Discount:              DiscountSnapshot{RatePercent: ratePercent, Version: settings.DiscountVersion},
		Store:                 storefront.Snapshot,
		Pickup:                PickupSnapshot{Date: input.PickupDate, Time: input.PickupTime, Meal: string(mealSelection.Code), Point: storefront.PickupPoint},
		OrderNote:             input.OrderNote,
		Items:                 make([]ItemSnapshot, 0, len(input.Items)),
		OriginalSubtotalCents: pricing.OriginalSubtotalCents,
		DiscountCents:         pricing.DiscountCents,
		PayableCents:          pricing.PayableCents,
		CreatedAt:             createdAt,
	}
	for index, item := range input.Items {
		line := pricing.Lines[index]
		quote.Items = append(quote.Items, ItemSnapshot{
			LineNumber: uint16(index + 1), ProductID: item.ProductID, ProductName: records[index].Name,
			ProductSourceVersion:   hashProductSource(records[index], input.PickupDate),
			ImageObjectKey:         records[index].ImageObjectKey,
			OriginalUnitPriceCents: line.OriginalUnitPriceCents, DiscountedUnitPriceCents: line.DiscountedUnitPriceCents,
			Quantity: line.Quantity, OriginalSubtotalCents: line.OriginalSubtotalCents, PayableSubtotalCents: line.PayableSubtotalCents,
			Flavors: append(make([]string, 0, len(item.Flavors)), item.Flavors...), Note: item.Note,
		})
	}
	expiresAt, ok := deriveQuoteExpiresAt(quote.CreatedAt, quote.Pickup.Date, quote.Pickup.Time)
	if !ok {
		return CreateResult{}, ErrUnavailable
	}
	quote.ExpiresAt = expiresAt
	quote.SnapshotDigest = hashQuoteSnapshot(quote)

	quoteID, err := insertQuote(ctx, transaction, quote, keyHash, requestDigest)
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	quote.ID = quoteID
	for _, item := range quote.Items {
		if err := insertQuoteItem(ctx, transaction, quoteID, item); err != nil {
			return CreateResult{}, ErrUnavailable
		}
	}
	if err := appendReceipt(ctx, transaction, provider.receipts, meta, ReceiptActionQuoteCreate, requestDigest, quoteCreateReceiptResponse{QuoteID: strconv.FormatUint(quoteID, 10)}); errors.Is(err, ErrOperationReceiptExists) {
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			return CreateResult{}, ErrUnavailable
		}
		if releaseErr := releaseLock(); releaseErr != nil {
			return CreateResult{}, ErrUnavailable
		}
		return provider.replayCreatedQuote(ctx, meta, keyHash, requestDigest)
	} else if err != nil {
		return CreateResult{}, err
	}
	if provider.beforeCommit != nil {
		if err := provider.beforeCommit(transaction); err != nil {
			return CreateResult{}, ErrUnavailable
		}
	}
	if err := transaction.Commit(); err != nil {
		return CreateResult{}, ErrUnavailable
	}
	committed = true
	return CreateResult{Quote: quote, Created: true}, nil
}

func (provider *Provider) replayCreatedQuote(ctx context.Context, meta WriteMeta, keyHash [32]byte, requestDigest [32]byte) (CreateResult, error) {
	transaction, err := provider.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return CreateResult{}, ErrUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()
	header, found, err := readQuoteHeader(ctx, transaction, "WHERE user_id=? AND idempotency_key_hash=?", meta.ActorUserID, keyHash[:])
	if err != nil {
		return CreateResult{}, normalizeSnapshotReadError(err)
	}
	if !found || header.requestDigest != requestDigest {
		if found {
			return CreateResult{}, ErrIdempotencyConflict
		}
		return CreateResult{}, ErrSnapshotInvalid
	}
	quote, err := completeStoredQuote(ctx, transaction, header)
	if err != nil {
		return CreateResult{}, normalizeSnapshotReadError(err)
	}
	var replay quoteCreateReceiptResponse
	replayed, err := replayReceipt(ctx, transaction, provider.receipts, meta, ReceiptActionQuoteCreate, requestDigest, &replay)
	quoteID, validQuoteID := parseReceiptUint(replay.QuoteID)
	if err != nil {
		return CreateResult{}, err
	}
	if !replayed || !validQuoteID || quoteID != quote.ID {
		return CreateResult{}, ErrSnapshotInvalid
	}
	if err := transaction.Commit(); err != nil {
		return CreateResult{}, ErrUnavailable
	}
	committed = true
	return CreateResult{Quote: quote, Created: false}, nil
}

// Read returns an immutable quote only when the supplied user owns it.
func (provider *Provider) Read(ctx context.Context, userID, quoteID uint64) (Quote, error) {
	if provider == nil || provider.db == nil {
		return Quote{}, ErrUnavailable
	}
	if userID == 0 || quoteID == 0 {
		return Quote{}, ErrNotFound
	}
	transaction, err := provider.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()
	header, found, err := readQuoteHeader(ctx, transaction, "WHERE id=? AND user_id=?", quoteID, userID)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if !found {
		return Quote{}, ErrNotFound
	}
	result, err := completeStoredQuote(ctx, transaction, header)
	if err != nil {
		return Quote{}, normalizeSnapshotReadError(err)
	}
	if err := transaction.Commit(); err != nil {
		return Quote{}, ErrUnavailable
	}
	committed = true
	return result, nil
}

func validIdempotencyKey(key string) bool {
	_, ok := exactIdempotencyKey([]string{key})
	return ok
}

func normalizeCreateInput(input CreateInput, now time.Time) (CreateInput, time.Time, error) {
	if !validContactName(input.ContactName) || !strictDate(input.PickupDate) || !strictPickupTime(input.PickupTime) || !utf8.ValidString(input.OrderNote) || len(input.Items) == 0 || len(input.Items) > math.MaxUint16 {
		return CreateInput{}, time.Time{}, ErrInvalidInput
	}
	serviceDate, err := time.ParseInLocation("2006-01-02", input.PickupDate, quoteLocation)
	if err != nil {
		return CreateInput{}, time.Time{}, ErrInvalidInput
	}
	localNow := now.In(quoteLocation)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, quoteLocation)
	if !serviceDate.Equal(today) && !serviceDate.Equal(today.AddDate(0, 0, 1)) {
		return CreateInput{}, time.Time{}, ErrInvalidInput
	}
	normalized := CreateInput{ContactName: input.ContactName, PickupDate: input.PickupDate, PickupTime: input.PickupTime, OrderNote: input.OrderNote, Items: make([]ItemInput, 0, len(input.Items))}
	seen := make(map[uint64]struct{}, len(input.Items))
	for _, item := range input.Items {
		if item.ProductID == 0 || item.Quantity <= 0 || !utf8.ValidString(item.Note) {
			return CreateInput{}, time.Time{}, ErrInvalidInput
		}
		if _, exists := seen[item.ProductID]; exists {
			return CreateInput{}, time.Time{}, ErrInvalidInput
		}
		seen[item.ProductID] = struct{}{}
		flavors := make([]string, 0, len(item.Flavors))
		seenFlavors := make(map[string]struct{}, len(item.Flavors))
		for _, flavor := range item.Flavors {
			if !validRequiredText(flavor) {
				return CreateInput{}, time.Time{}, ErrInvalidInput
			}
			if _, duplicate := seenFlavors[flavor]; duplicate {
				return CreateInput{}, time.Time{}, ErrInvalidInput
			}
			seenFlavors[flavor] = struct{}{}
			flavors = append(flavors, flavor)
		}
		normalized.Items = append(normalized.Items, ItemInput{ProductID: item.ProductID, Quantity: item.Quantity, Flavors: flavors, Note: item.Note})
	}
	return normalized, serviceDate, nil
}

func strictDate(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	for index := range value {
		if index == 4 || index == 7 {
			continue
		}
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func strictPickupTime(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	for _, index := range []int{0, 1, 3, 4} {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	return hour <= 23 && minute <= 59
}

func validContactName(value string) bool {
	return validRequiredText(value) && len(value) <= 64
}

func validPrimaryPhone(value string) bool {
	if len(value) < 2 || len(value) > 16 || value[0] != '+' || value[1] < '1' || value[1] > '9' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func deriveQuoteExpiresAt(createdAt time.Time, pickupDate, pickupTime string) (time.Time, bool) {
	if createdAt.IsZero() || !strictDate(pickupDate) || !strictPickupTime(pickupTime) {
		return time.Time{}, false
	}
	pickupAt, err := time.ParseInLocation("2006-01-02 15:04", pickupDate+" "+pickupTime, quoteLocation)
	if err != nil {
		return time.Time{}, false
	}
	createdUTC := createdAt.UTC().Truncate(time.Microsecond)
	pickupUTC := pickupAt.UTC()
	if !pickupUTC.After(createdUTC) {
		return time.Time{}, false
	}
	expiresAt := createdUTC.Add(quotePrepayWindow)
	if pickupUTC.Before(expiresAt) {
		expiresAt = pickupUTC
	}
	return expiresAt, true
}

func quoteLockName(keyHash [32]byte) string {
	return "order_quote_" + hex.EncodeToString(keyHash[:26])
}

func acquireQuoteLock(ctx context.Context, connection *sql.Conn, name string) (bool, error) {
	var value sql.NullInt64
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", name, quoteLockSeconds).Scan(&value); err != nil || !value.Valid {
		return false, err
	}
	return value.Int64 == 1, nil
}

func releaseQuoteLock(ctx context.Context, connection *sql.Conn, name string) error {
	var value sql.NullInt64
	if err := connection.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", name).Scan(&value); err != nil || !value.Valid || value.Int64 != 1 {
		return ErrUnavailable
	}
	return nil
}

func readSourceSettings(ctx context.Context, transaction *sql.Tx) (sourceSettings, error) {
	var result sourceSettings
	err := transaction.QueryRowContext(ctx, `SELECT rate_percent,discount_version,whitelist_version FROM discount_settings WHERE id=1 FOR SHARE`).Scan(
		&result.RatePercent, &result.DiscountVersion, &result.WhitelistVersion,
	)
	if err != nil || result.RatePercent < 1 || result.RatePercent > 100 || result.DiscountVersion == 0 || result.WhitelistVersion == 0 {
		return sourceSettings{}, ErrUnavailable
	}
	return result, nil
}

func readUserIdentity(ctx context.Context, transaction *sql.Tx, userID uint64) (userIdentityRecord, error) {
	var primaryPhone, extraPhone, extraName sql.NullString
	var primaryBoundAt, extraPhoneSetAt sql.NullTime
	var extraNameKey []byte
	var recordVersion uint64
	if err := transaction.QueryRowContext(ctx, `SELECT
  primary_phone,primary_phone_bound_at,extra_phone,extra_name,extra_name_key,extra_phone_set_at,record_version
FROM miniprogram_users
WHERE id=?
FOR SHARE`, userID).Scan(
		&primaryPhone, &primaryBoundAt, &extraPhone, &extraName, &extraNameKey, &extraPhoneSetAt, &recordVersion,
	); err != nil {
		return userIdentityRecord{}, ErrUnavailable
	}
	if primaryPhone.Valid != primaryBoundAt.Valid || recordVersion == 0 {
		return userIdentityRecord{}, ErrUnavailable
	}
	if !primaryPhone.Valid {
		return userIdentityRecord{}, ErrPrimaryPhoneRequired
	}
	if !validPrimaryPhone(primaryPhone.String) {
		return userIdentityRecord{}, ErrUnavailable
	}
	extraFields := []bool{extraPhone.Valid, extraName.Valid, extraNameKey != nil, extraPhoneSetAt.Valid}
	extraCount := 0
	for _, present := range extraFields {
		if present {
			extraCount++
		}
	}
	result := userIdentityRecord{PrimaryPhone: primaryPhone.String}
	if extraCount == 0 {
		return result, nil
	}
	if extraCount != len(extraFields) || !validPrimaryPhone(extraPhone.String) {
		return userIdentityRecord{}, ErrUnavailable
	}
	nameKey, validNameKey := canonicalStaffNameKey(extraName.String)
	if !validNameKey || nameKey != string(extraNameKey) {
		return userIdentityRecord{}, ErrUnavailable
	}
	result.Extra = &staffidentity.ExtraClaim{Phone: extraPhone.String, Name: extraName.String}
	return result, nil
}

func resolveIdentitySnapshot(ctx context.Context, transaction *sql.Tx, user userIdentityRecord, whitelistVersion uint64) (resolvedIdentitySnapshot, error) {
	if !validPrimaryPhone(user.PrimaryPhone) || whitelistVersion == 0 {
		return resolvedIdentitySnapshot{}, ErrUnavailable
	}
	phones := []any{user.PrimaryPhone}
	query := `SELECT phone,name,name_key,enabled FROM staff_whitelist WHERE phone=? ORDER BY id FOR SHARE`
	if user.Extra != nil && user.Extra.Phone != user.PrimaryPhone {
		phones = append(phones, user.Extra.Phone)
		query = `SELECT phone,name,name_key,enabled FROM staff_whitelist WHERE phone IN (?,?) ORDER BY id FOR SHARE`
	}
	rows, err := transaction.QueryContext(ctx, query, phones...)
	if err != nil {
		return resolvedIdentitySnapshot{}, ErrUnavailable
	}
	defer rows.Close()
	entries := make([]staffidentity.Entry, 0, len(phones))
	for rows.Next() {
		var phone, name string
		var nameKey []byte
		var enabled bool
		if err := rows.Scan(&phone, &name, &nameKey, &enabled); err != nil {
			return resolvedIdentitySnapshot{}, ErrUnavailable
		}
		canonicalNameKey, validNameKey := canonicalStaffNameKey(name)
		if !validNameKey || canonicalNameKey != string(nameKey) {
			return resolvedIdentitySnapshot{}, ErrUnavailable
		}
		entries = append(entries, staffidentity.Entry{Phone: phone, Name: name, Enabled: enabled})
	}
	if rows.Err() != nil {
		return resolvedIdentitySnapshot{}, ErrUnavailable
	}
	resolved, err := staffidentity.Resolve(staffidentity.Input{
		PrimaryPhone: user.PrimaryPhone, Extra: user.Extra, WhitelistVersion: whitelistVersion, CandidateEntries: entries,
	})
	if err != nil {
		return resolvedIdentitySnapshot{}, ErrUnavailable
	}
	kind := IdentityVisitor
	if resolved.Kind == staffidentity.KindStaff {
		kind = IdentityStaff
	} else if resolved.Kind != staffidentity.KindVisitor {
		return resolvedIdentitySnapshot{}, ErrUnavailable
	}
	return resolvedIdentitySnapshot{
		Snapshot:     IdentitySnapshot{Kind: kind, SourceVersion: resolved.WhitelistVersion},
		PrimaryPhone: user.PrimaryPhone,
	}, nil
}

func canonicalStaffNameKey(name string) (string, bool) {
	if !validRequiredText(name) {
		return "", false
	}
	var key strings.Builder
	for _, character := range norm.NFKC.String(name) {
		if !unicode.IsSpace(character) {
			key.WriteRune(character)
		}
	}
	result := key.String()
	return result, result != "" && len(result) <= 400
}

func readStoreSnapshot(ctx context.Context, transaction *sql.Tx) (storefrontFacts, error) {
	var result storefrontFacts
	var flavorOptionsJSON []byte
	var recordVersion uint64
	err := transaction.QueryRowContext(ctx, `SELECT
  store_name,store_address,pickup_point,business_status,flavor_options_json,record_version
FROM storefront_settings
WHERE id=1
FOR SHARE`).Scan(
		&result.Snapshot.Name, &result.Snapshot.Address, &result.PickupPoint, &result.BusinessStatus, &flavorOptionsJSON, &recordVersion,
	)
	if err != nil || !validRequiredText(result.Snapshot.Name) || !validRequiredText(result.Snapshot.Address) || !validRequiredText(result.PickupPoint) || recordVersion == 0 {
		return storefrontFacts{}, ErrUnavailable
	}
	if result.BusinessStatus != "open" && result.BusinessStatus != "closed" && result.BusinessStatus != "cutoff" {
		return storefrontFacts{}, ErrUnavailable
	}
	options, ok := parseFlavorOptions(flavorOptionsJSON)
	if !ok {
		return storefrontFacts{}, ErrUnavailable
	}
	result.FlavorOptions = options
	return result, nil
}

func parseFlavorOptions(encoded []byte) (map[string]struct{}, bool) {
	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) < 2 || trimmed[0] != '[' {
		return nil, false
	}
	var values []string
	if json.Unmarshal(trimmed, &values) != nil || values == nil {
		return nil, false
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validRequiredText(value) {
			return nil, false
		}
		if _, duplicate := result[value]; duplicate {
			return nil, false
		}
		result[value] = struct{}{}
	}
	return result, true
}

func selectedFlavorsAvailable(items []ItemInput, options map[string]struct{}) bool {
	for _, item := range items {
		for _, flavor := range item.Flavors {
			if _, available := options[flavor]; !available {
				return false
			}
		}
	}
	return true
}

func snapshotFlavorsAvailable(items []ItemSnapshot, options map[string]struct{}) bool {
	for _, item := range items {
		for _, flavor := range item.Flavors {
			if _, available := options[flavor]; !available {
				return false
			}
		}
	}
	return true
}

func readServiceDate(ctx context.Context, transaction *sql.Tx, serviceDate string) (bool, error) {
	if !strictDate(serviceDate) {
		return false, ErrUnavailable
	}
	var isOpen bool
	var recordVersion uint64
	err := transaction.QueryRowContext(ctx, `SELECT is_open,record_version FROM service_dates WHERE service_date=? FOR SHARE`, serviceDate).Scan(&isOpen, &recordVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil || recordVersion == 0 {
		return false, ErrUnavailable
	}
	return isOpen, nil
}

func validRequiredText(value string) bool {
	return utf8.ValidString(value) && value != "" && strings.TrimSpace(value) == value
}

func validImageObjectKey(value string) bool {
	return value != "" && len(value) <= 1024 && utf8.ValidString(value) && strings.TrimSpace(value) == value
}

func nullableImageObjectKey(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func readMealSelection(ctx context.Context, transaction *sql.Tx, serviceDate time.Time, pickup string, now time.Time) (menu.MealSelection, error) {
	rows, err := transaction.QueryContext(ctx, `SELECT code,cutoff_time,pickup_start_time,pickup_end_time,interval_minutes FROM meal_periods ORDER BY code FOR SHARE`)
	if err != nil {
		return menu.MealSelection{}, ErrUnavailable
	}
	defer rows.Close()
	records := make([]menu.MealPeriodRecord, 0, 2)
	for rows.Next() {
		var record menu.MealPeriodRecord
		if err := rows.Scan(&record.Code, &record.CutoffTime, &record.PickupStartTime, &record.PickupEndTime, &record.IntervalMinutes); err != nil {
			return menu.MealSelection{}, ErrUnavailable
		}
		records = append(records, record)
	}
	if rows.Err() != nil {
		return menu.MealSelection{}, ErrUnavailable
	}
	return menu.ResolveMeal(records, serviceDate, pickup, now.In(quoteLocation))
}

func readProduct(ctx context.Context, transaction *sql.Tx, productID uint64, serviceDate string) (productRecord, error) {
	var result productRecord
	var price uint64
	var imageObjectKey sql.NullString
	err := transaction.QueryRowContext(ctx, `SELECT
  p.id,p.category_id,p.name,p.price_cents,p.meal_period,p.is_listed,c.is_active,
  (s.product_id IS NOT NULL) AS sold_out,
  JSON_UNQUOTE(JSON_EXTRACT(p.images_json,'$[0].object_key')) AS image_object_key
FROM products AS p
INNER JOIN categories AS c ON c.id=p.category_id
LEFT JOIN product_sold_out_dates AS s ON s.product_id=p.id AND s.service_date=?
WHERE p.id=?
LIMIT 1
FOR SHARE`, serviceDate, productID).Scan(
		&result.ID, &result.CategoryID, &result.Name, &price, &result.MealPeriod, &result.Listed, &result.CategoryActive, &result.SoldOut, &imageObjectKey,
	)
	if err != nil {
		return productRecord{}, err
	}
	if result.ID != productID || result.CategoryID == 0 || !validRequiredText(result.Name) || price > math.MaxInt64 ||
		(result.MealPeriod != "all" && result.MealPeriod != "lunch" && result.MealPeriod != "dinner") {
		return productRecord{}, ErrUnavailable
	}
	result.PriceCents = int64(price)
	if imageObjectKey.Valid {
		if !validImageObjectKey(imageObjectKey.String) {
			return productRecord{}, ErrUnavailable
		}
		result.ImageObjectKey = imageObjectKey.String
	}
	return result, nil
}

func insertQuote(ctx context.Context, transaction *sql.Tx, quote Quote, keyHash, requestDigest [32]byte) (uint64, error) {
	result, err := transaction.ExecContext(ctx, `INSERT INTO quotes(
	  user_id,contact_name_snapshot,contact_phone_snapshot,idempotency_key_hash,request_digest,identity_kind,identity_source_version,
	  discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,
	  pickup_date,pickup_time,meal_period,order_note,item_count,
	  original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		quote.UserID, quote.Contact.Name, quote.Contact.Phone, keyHash[:], requestDigest[:], string(quote.Identity.Kind), quote.Identity.SourceVersion,
		quote.Discount.RatePercent, quote.Discount.Version, quote.Store.Name, quote.Store.Address, quote.Pickup.Point,
		quote.Pickup.Date, quote.Pickup.Time+":00", quote.Pickup.Meal, quote.OrderNote, len(quote.Items),
		quote.OriginalSubtotalCents, quote.DiscountCents, quote.PayableCents, quote.SnapshotDigest[:], quote.CreatedAt, quote.ExpiresAt,
	)
	if err != nil {
		return 0, err
	}
	inserted, err := result.LastInsertId()
	if err != nil || inserted <= 0 {
		return 0, ErrUnavailable
	}
	return uint64(inserted), nil
}

func insertQuoteItem(ctx context.Context, transaction *sql.Tx, quoteID uint64, item ItemSnapshot) error {
	flavors, err := json.Marshal(item.Flavors)
	if err != nil {
		return ErrUnavailable
	}
	result, err := transaction.ExecContext(ctx, `INSERT INTO quote_items(
  quote_id,line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,
  original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,
  flavors_json,line_note
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		quoteID, item.LineNumber, item.ProductID, item.ProductName, item.ProductSourceVersion[:], nullableImageObjectKey(item.ImageObjectKey),
		item.OriginalUnitPriceCents, item.DiscountedUnitPriceCents, item.Quantity, item.OriginalSubtotalCents, item.PayableSubtotalCents,
		flavors, item.Note,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrUnavailable
	}
	return nil
}

type storedQuoteHeader struct {
	quote         Quote
	requestDigest [32]byte
	itemCount     uint16
}

func readQuoteHeader(ctx context.Context, transaction *sql.Tx, predicate string, args ...any) (storedQuoteHeader, bool, error) {
	query := `SELECT
	  id,user_id,contact_name_snapshot,contact_phone_snapshot,request_digest,identity_kind,identity_source_version,
  discount_rate_percent,discount_version,store_name_snapshot,store_address_snapshot,pickup_point_snapshot,
  pickup_date,pickup_time,meal_period,order_note,item_count,
	  original_subtotal_cents,discount_cents,payable_cents,snapshot_digest,created_at,expires_at
FROM quotes ` + predicate
	var result storedQuoteHeader
	var requestDigest, snapshotDigest []byte
	var identityKind string
	var pickupDate, pickupTime any
	err := transaction.QueryRowContext(ctx, query, args...).Scan(
		&result.quote.ID, &result.quote.UserID, &result.quote.Contact.Name, &result.quote.Contact.Phone, &requestDigest, &identityKind, &result.quote.Identity.SourceVersion,
		&result.quote.Discount.RatePercent, &result.quote.Discount.Version, &result.quote.Store.Name, &result.quote.Store.Address, &result.quote.Pickup.Point,
		&pickupDate, &pickupTime, &result.quote.Pickup.Meal, &result.quote.OrderNote, &result.itemCount,
		&result.quote.OriginalSubtotalCents, &result.quote.DiscountCents, &result.quote.PayableCents, &snapshotDigest, &result.quote.CreatedAt, &result.quote.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return storedQuoteHeader{}, false, nil
	}
	if err != nil {
		return storedQuoteHeader{}, false, ErrUnavailable
	}
	date, validDate := mysqlDateString(pickupDate)
	timeOfDay, validTime := mysqlPickupTimeString(pickupTime)
	if len(requestDigest) != 32 || len(snapshotDigest) != 32 || !validDate || !validTime {
		return storedQuoteHeader{}, false, ErrSnapshotInvalid
	}
	copy(result.requestDigest[:], requestDigest)
	copy(result.quote.SnapshotDigest[:], snapshotDigest)
	result.quote.Identity.Kind = IdentityKind(identityKind)
	result.quote.Pickup.Date = date
	result.quote.Pickup.Time = timeOfDay
	result.quote.CreatedAt = result.quote.CreatedAt.UTC().Truncate(time.Microsecond)
	result.quote.ExpiresAt = result.quote.ExpiresAt.UTC().Truncate(time.Microsecond)
	return result, true, nil
}

func mysqlDateString(value any) (string, bool) {
	var result string
	switch typed := value.(type) {
	case time.Time:
		if typed.Hour() != 0 || typed.Minute() != 0 || typed.Second() != 0 || typed.Nanosecond() != 0 {
			return "", false
		}
		result = typed.Format("2006-01-02")
	case string:
		result = typed
	case []byte:
		result = string(typed)
	default:
		return "", false
	}
	return result, strictDate(result)
}

func mysqlPickupTimeString(value any) (string, bool) {
	var result string
	switch typed := value.(type) {
	case string:
		result = typed
	case []byte:
		result = string(typed)
	default:
		return "", false
	}
	if len(result) != 8 || result[5:] != ":00" || !strictPickupTime(result[:5]) {
		return "", false
	}
	return result[:5], true
}

func completeStoredQuote(ctx context.Context, transaction *sql.Tx, header storedQuoteHeader) (Quote, error) {
	return completeStoredQuoteWithLock(ctx, transaction, header, false)
}

func completeStoredQuoteWithLock(ctx context.Context, transaction *sql.Tx, header storedQuoteHeader, lock bool) (Quote, error) {
	query := `SELECT
	  line_number,product_id,product_name_snapshot,product_source_version,image_object_key_snapshot,
  original_unit_price_cents,discounted_unit_price_cents,quantity,original_subtotal_cents,payable_subtotal_cents,
  flavors_json,line_note
FROM quote_items
WHERE quote_id=?
ORDER BY line_number`
	if lock {
		query += " FOR UPDATE"
	}
	rows, err := transaction.QueryContext(ctx, query, header.quote.ID)
	if err != nil {
		return Quote{}, ErrUnavailable
	}
	defer rows.Close()
	items := make([]ItemSnapshot, 0, header.itemCount)
	for rows.Next() {
		var item ItemSnapshot
		var sourceVersion, flavorsJSON []byte
		var imageObjectKey sql.NullString
		if err := rows.Scan(
			&item.LineNumber, &item.ProductID, &item.ProductName, &sourceVersion, &imageObjectKey,
			&item.OriginalUnitPriceCents, &item.DiscountedUnitPriceCents, &item.Quantity, &item.OriginalSubtotalCents, &item.PayableSubtotalCents,
			&flavorsJSON, &item.Note,
		); err != nil {
			return Quote{}, ErrSnapshotInvalid
		}
		if len(sourceVersion) != 32 || json.Unmarshal(flavorsJSON, &item.Flavors) != nil {
			return Quote{}, ErrSnapshotInvalid
		}
		copy(item.ProductSourceVersion[:], sourceVersion)
		if imageObjectKey.Valid {
			if !validImageObjectKey(imageObjectKey.String) {
				return Quote{}, ErrSnapshotInvalid
			}
			item.ImageObjectKey = imageObjectKey.String
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return Quote{}, ErrUnavailable
	}
	header.quote.Items = items
	if !validStoredQuote(header.quote, header.itemCount) || hashQuoteSnapshot(header.quote) != header.quote.SnapshotDigest {
		return Quote{}, ErrSnapshotInvalid
	}
	return header.quote, nil
}

func normalizeSnapshotReadError(err error) error {
	if errors.Is(err, ErrSnapshotInvalid) {
		return ErrSnapshotInvalid
	}
	return ErrUnavailable
}

func validStoredQuote(value Quote, itemCount uint16) bool {
	expiresAt, validDeadline := deriveQuoteExpiresAt(value.CreatedAt, value.Pickup.Date, value.Pickup.Time)
	if value.ID == 0 || value.UserID == 0 || uint16(len(value.Items)) != itemCount || itemCount == 0 ||
		!validContactName(value.Contact.Name) || !validPrimaryPhone(value.Contact.Phone) ||
		(value.Identity.Kind != IdentityStaff && value.Identity.Kind != IdentityVisitor) || value.Identity.SourceVersion == 0 ||
		value.Discount.RatePercent < 1 || value.Discount.RatePercent > 100 || value.Discount.Version == 0 ||
		!validRequiredText(value.Store.Name) || !validRequiredText(value.Store.Address) || !validRequiredText(value.Pickup.Point) ||
		!strictDate(value.Pickup.Date) || !strictPickupTime(value.Pickup.Time) ||
		(value.Pickup.Meal != "lunch" && value.Pickup.Meal != "dinner") || !utf8.ValidString(value.OrderNote) || !validDeadline || !value.ExpiresAt.Equal(expiresAt) {
		return false
	}
	lines := make([]quotepricing.Line, 0, len(value.Items))
	for index, item := range value.Items {
		if item.LineNumber != uint16(index+1) || item.ProductID == 0 || !validRequiredText(item.ProductName) ||
			item.ProductSourceVersion == ([32]byte{}) || item.OriginalUnitPriceCents < 0 || item.DiscountedUnitPriceCents < 0 || item.Quantity <= 0 ||
			(item.ImageObjectKey != "" && !validImageObjectKey(item.ImageObjectKey)) || !utf8.ValidString(item.Note) || item.Flavors == nil {
			return false
		}
		seenFlavors := make(map[string]struct{}, len(item.Flavors))
		for _, flavor := range item.Flavors {
			if !validRequiredText(flavor) {
				return false
			}
			if _, duplicate := seenFlavors[flavor]; duplicate {
				return false
			}
			seenFlavors[flavor] = struct{}{}
		}
		lines = append(lines, quotepricing.Line{UnitPriceCents: item.OriginalUnitPriceCents, Quantity: item.Quantity})
	}
	pricing, err := quotepricing.Calculate(quotepricing.Input{RatePercent: value.Discount.RatePercent, Lines: lines})
	if err != nil || pricing.OriginalSubtotalCents != value.OriginalSubtotalCents || pricing.DiscountCents != value.DiscountCents || pricing.PayableCents != value.PayableCents {
		return false
	}
	for index, line := range pricing.Lines {
		item := value.Items[index]
		if item.OriginalUnitPriceCents != line.OriginalUnitPriceCents || item.DiscountedUnitPriceCents != line.DiscountedUnitPriceCents ||
			item.Quantity != line.Quantity || item.OriginalSubtotalCents != line.OriginalSubtotalCents || item.PayableSubtotalCents != line.PayableSubtotalCents {
			return false
		}
	}
	return true
}
