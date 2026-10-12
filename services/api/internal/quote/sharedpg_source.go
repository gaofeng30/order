package quote

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

// The fixed RPC exposes stored facts, not a ready-made public Quote. Header
// conversion, aggregate validity, pricing and its canonical digest remain
// independently checked using the original provider's rules.
type sharedPGQuoteResponse struct {
	Outcome             string          `json:"outcome"`
	Quote               json.RawMessage `json:"quote"`
	ReceiptFound        bool            `json:"receipt_found"`
	ReceiptEvidenceText *string         `json:"receipt_evidence_text"`
	ReceiptResponseText *string         `json:"receipt_response_text"`
}

type sharedPGQuoteHeaderFacts struct {
	ID               *string         `json:"id"`
	UserID           *string         `json:"user_id"`
	ContactName      *string         `json:"contact_name"`
	ContactPhoneHex  *string         `json:"contact_phone_hex"`
	RequestDigestHex *string         `json:"request_digest_hex"`
	IdentityKind     *string         `json:"identity_kind"`
	IdentityVersion  *string         `json:"identity_source_version"`
	Rate             *int64          `json:"discount_rate_percent"`
	DiscountVersion  *string         `json:"discount_version"`
	StoreName        *string         `json:"store_name"`
	StoreAddress     *string         `json:"store_address"`
	PickupPoint      *string         `json:"pickup_point"`
	PickupDate       *string         `json:"pickup_date"`
	PickupTime       *string         `json:"pickup_time"`
	MealPeriod       *string         `json:"meal_period"`
	OrderNote        *string         `json:"order_note"`
	ItemCount        *uint16         `json:"item_count"`
	OriginalSubtotal *int64          `json:"original_subtotal_cents"`
	DiscountCents    *int64          `json:"discount_cents"`
	PayableCents     *int64          `json:"payable_cents"`
	SnapshotHex      *string         `json:"snapshot_digest_hex"`
	CreatedAt        *string         `json:"created_at"`
	ExpiresAt        *string         `json:"expires_at"`
	Items            json.RawMessage `json:"items"`
}

type sharedPGQuoteItemFacts struct {
	Line             *uint16         `json:"line_number"`
	ProductID        *string         `json:"product_id"`
	ProductName      *string         `json:"product_name"`
	ProductSourceHex *string         `json:"product_source_version_hex"`
	ImageObjectHex   *string         `json:"image_object_key_hex"`
	OriginalUnit     *int64          `json:"original_unit_price_cents"`
	DiscountedUnit   *int64          `json:"discounted_unit_price_cents"`
	Quantity         *int64          `json:"quantity"`
	OriginalSubtotal *int64          `json:"original_subtotal_cents"`
	PayableSubtotal  *int64          `json:"payable_subtotal_cents"`
	Flavors          json.RawMessage `json:"flavors_json"`
	Note             *string         `json:"line_note"`
}

func decodeSharedPGQuote(raw json.RawMessage) (storedQuoteHeader, error) {
	var h sharedPGQuoteHeaderFacts
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &h) != nil ||
		h.ID == nil || h.UserID == nil || h.ContactName == nil || h.ContactPhoneHex == nil ||
		h.IdentityKind == nil || h.IdentityVersion == nil || h.Rate == nil || h.DiscountVersion == nil ||
		h.StoreName == nil || h.StoreAddress == nil || h.PickupPoint == nil || h.MealPeriod == nil ||
		h.OrderNote == nil || h.ItemCount == nil || h.OriginalSubtotal == nil || h.DiscountCents == nil ||
		h.PayableCents == nil || h.CreatedAt == nil || h.ExpiresAt == nil {
		return storedQuoteHeader{}, ErrUnavailable
	}
	id, idOK := sharedPGQuoteUint(*h.ID)
	user, userOK := sharedPGQuoteUint(*h.UserID)
	identityVersion, identityOK := sharedPGQuoteUint(*h.IdentityVersion)
	discountVersion, discountOK := sharedPGQuoteUint(*h.DiscountVersion)
	phone, phoneErr := hex.DecodeString(*h.ContactPhoneHex)
	createdAt, createdErr := time.Parse(time.RFC3339Nano, *h.CreatedAt)
	expiresAt, expiresErr := time.Parse(time.RFC3339Nano, *h.ExpiresAt)
	if !idOK || !userOK || !identityOK || !discountOK || phoneErr != nil || createdErr != nil || expiresErr != nil {
		return storedQuoteHeader{}, ErrUnavailable
	}
	if h.RequestDigestHex == nil || h.SnapshotHex == nil || h.PickupDate == nil || h.PickupTime == nil {
		return storedQuoteHeader{}, ErrSnapshotInvalid
	}
	requestDigest, requestOK := sharedPGQuoteDigest(*h.RequestDigestHex)
	snapshotDigest, snapshotOK := sharedPGQuoteDigest(*h.SnapshotHex)
	date, dateOK := postgresDateString(*h.PickupDate)
	timeOfDay, timeOK := postgresPickupTimeString(*h.PickupTime)
	if !requestOK || !snapshotOK || !dateOK || !timeOK {
		return storedQuoteHeader{}, ErrSnapshotInvalid
	}
	q := Quote{
		ID: id, UserID: user,
		Contact:   ContactSnapshot{Name: *h.ContactName, Phone: string(phone)},
		Identity:  IdentitySnapshot{Kind: IdentityKind(*h.IdentityKind), SourceVersion: identityVersion},
		Discount:  DiscountSnapshot{RatePercent: *h.Rate, Version: discountVersion},
		Store:     StoreSnapshot{Name: *h.StoreName, Address: *h.StoreAddress},
		Pickup:    PickupSnapshot{Date: date, Time: timeOfDay, Meal: *h.MealPeriod, Point: *h.PickupPoint},
		OrderNote: *h.OrderNote, OriginalSubtotalCents: *h.OriginalSubtotal,
		DiscountCents: *h.DiscountCents, PayableCents: *h.PayableCents,
		SnapshotDigest: snapshotDigest, CreatedAt: createdAt.UTC().Truncate(time.Microsecond),
		ExpiresAt: expiresAt.UTC().Truncate(time.Microsecond),
	}
	var facts []sharedPGQuoteItemFacts
	if len(h.Items) == 0 || json.Unmarshal(h.Items, &facts) != nil {
		return storedQuoteHeader{}, ErrSnapshotInvalid
	}
	q.Items = make([]ItemSnapshot, 0, len(facts))
	for _, f := range facts {
		item, ok := decodeSharedPGQuoteItem(f)
		if !ok {
			return storedQuoteHeader{}, ErrSnapshotInvalid
		}
		q.Items = append(q.Items, item)
	}
	if !validStoredQuote(q, *h.ItemCount) || hashQuoteSnapshot(q) != q.SnapshotDigest {
		return storedQuoteHeader{}, ErrSnapshotInvalid
	}
	return storedQuoteHeader{quote: q, requestDigest: requestDigest, itemCount: *h.ItemCount}, nil
}

func decodeSharedPGQuoteItem(f sharedPGQuoteItemFacts) (ItemSnapshot, bool) {
	if f.Line == nil || f.ProductID == nil || f.ProductName == nil || f.ProductSourceHex == nil ||
		f.OriginalUnit == nil || f.DiscountedUnit == nil || f.Quantity == nil || f.OriginalSubtotal == nil ||
		f.PayableSubtotal == nil || f.Note == nil {
		return ItemSnapshot{}, false
	}
	product, productOK := sharedPGQuoteUint(*f.ProductID)
	source, sourceOK := sharedPGQuoteDigest(*f.ProductSourceHex)
	if !productOK || !sourceOK {
		return ItemSnapshot{}, false
	}
	item := ItemSnapshot{
		LineNumber: *f.Line, ProductID: product, ProductName: *f.ProductName, ProductSourceVersion: source,
		OriginalUnitPriceCents: *f.OriginalUnit, DiscountedUnitPriceCents: *f.DiscountedUnit,
		Quantity: *f.Quantity, OriginalSubtotalCents: *f.OriginalSubtotal, PayableSubtotalCents: *f.PayableSubtotal,
		Note: *f.Note,
	}
	if f.ImageObjectHex != nil {
		key, err := hex.DecodeString(*f.ImageObjectHex)
		if err != nil || !validImageObjectKey(string(key)) {
			return ItemSnapshot{}, false
		}
		item.ImageObjectKey = string(key)
	}
	if json.Unmarshal(f.Flavors, &item.Flavors) != nil {
		return ItemSnapshot{}, false
	}
	return item, true
}

func sharedPGQuoteUint(raw string) (uint64, bool) {
	value, err := strconv.ParseUint(raw, 10, 64)
	return value, err == nil && strconv.FormatUint(value, 10) == raw
}

func sharedPGQuoteDigest(raw string) ([32]byte, bool) {
	bytes, err := hex.DecodeString(raw)
	if err != nil || len(bytes) != 32 || hex.EncodeToString(bytes) != raw {
		return [32]byte{}, false
	}
	var value [32]byte
	copy(value[:], bytes)
	return value, true
}

func validateSharedPGQuoteReceipt(response sharedPGQuoteResponse, quoteID uint64, requestDigest [32]byte) error {
	if !response.ReceiptFound || response.ReceiptEvidenceText == nil || response.ReceiptResponseText == nil {
		return ErrSnapshotInvalid
	}
	evidence, body := []byte(*response.ReceiptEvidenceText), []byte(*response.ReceiptResponseText)
	if len(evidence) == 0 || len(evidence) > 256 || !json.Valid(evidence) || len(body) == 0 || len(body) > 4096 || !json.Valid(body) {
		return ErrSnapshotInvalid
	}
	var proof struct {
		RequestDigest string `json:"request_digest"`
	}
	if !decodeReceiptResponse(evidence, &proof) {
		return ErrSnapshotInvalid
	}
	digest, ok := sharedPGQuoteDigest(proof.RequestDigest)
	if !ok {
		return ErrSnapshotInvalid
	}
	if digest != requestDigest {
		return ErrIdempotencyConflict
	}
	var replay quoteCreateReceiptResponse
	if !decodeReceiptResponse(body, &replay) {
		return ErrSnapshotInvalid
	}
	actualID, validID := parseReceiptUint(replay.QuoteID)
	if !validID || actualID != quoteID {
		return ErrSnapshotInvalid
	}
	return nil
}

func sharedPGQuoteError(outcome string) error {
	switch outcome {
	case "PRIMARY_PHONE_REQUIRED":
		return ErrPrimaryPhoneRequired
	case "SELECTION_UNAVAILABLE":
		return ErrSelectionUnavailable
	case "PAYMENT_AMOUNT_TOO_SMALL":
		return ErrPaymentAmountTooSmall
	case "IDEMPOTENCY_CONFLICT":
		return ErrIdempotencyConflict
	case "SNAPSHOT_INVALID":
		return ErrSnapshotInvalid
	default:
		return ErrUnavailable
	}
}
