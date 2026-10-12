package quote

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
)

type canonicalDigest struct {
	hash hash.Hash
	word [8]byte
}

func newCanonicalDigest(domain string) *canonicalDigest {
	value := &canonicalDigest{hash: sha256.New()}
	value.writeString(domain)
	return value
}

func (value *canonicalDigest) writeUint64(input uint64) {
	binary.BigEndian.PutUint64(value.word[:], input)
	_, _ = value.hash.Write(value.word[:])
}

func (value *canonicalDigest) writeInt64(input int64) {
	value.writeUint64(uint64(input))
}

func (value *canonicalDigest) writeBool(input bool) {
	if input {
		value.writeUint64(1)
		return
	}
	value.writeUint64(0)
}

func (value *canonicalDigest) writeString(input string) {
	value.writeUint64(uint64(len(input)))
	_, _ = value.hash.Write([]byte(input))
}

func (value *canonicalDigest) writeBytes(input []byte) {
	value.writeUint64(uint64(len(input)))
	_, _ = value.hash.Write(input)
}

func (value *canonicalDigest) sum() [32]byte {
	var result [32]byte
	copy(result[:], value.hash.Sum(nil))
	return result
}

func hashIdempotencyKey(userID uint64, key string) [32]byte {
	value := newCanonicalDigest("order.quote.idempotency.v1")
	value.writeUint64(userID)
	value.writeString(key)
	return value.sum()
}

func hashCreateInput(input CreateInput) [32]byte {
	value := newCanonicalDigest("order.quote.request.v1")
	value.writeString(input.ContactName)
	value.writeString(input.PickupDate)
	value.writeString(input.PickupTime)
	value.writeString(input.OrderNote)
	value.writeUint64(uint64(len(input.Items)))
	for _, item := range input.Items {
		value.writeUint64(item.ProductID)
		value.writeInt64(item.Quantity)
		value.writeUint64(uint64(len(item.Flavors)))
		for _, flavor := range item.Flavors {
			value.writeString(flavor)
		}
		value.writeString(item.Note)
	}
	return value.sum()
}

func hashProductSource(record productRecord, serviceDate string) [32]byte {
	value := newCanonicalDigest("order.quote.product-source.v1")
	value.writeUint64(record.ID)
	value.writeUint64(record.CategoryID)
	value.writeString(record.Name)
	value.writeInt64(record.PriceCents)
	value.writeString(record.MealPeriod)
	value.writeString(record.ImageObjectKey)
	value.writeBool(record.Listed)
	value.writeBool(record.CategoryActive)
	value.writeBool(record.SoldOut)
	value.writeString(serviceDate)
	return value.sum()
}

func hashQuoteSnapshot(input Quote) [32]byte {
	value := newCanonicalDigest("order.quote.snapshot.v1")
	value.writeUint64(input.UserID)
	value.writeString(input.Contact.Name)
	value.writeString(input.Contact.Phone)
	value.writeString(string(input.Identity.Kind))
	value.writeUint64(input.Identity.SourceVersion)
	value.writeInt64(input.Discount.RatePercent)
	value.writeUint64(input.Discount.Version)
	value.writeString(input.Store.Name)
	value.writeString(input.Store.Address)
	value.writeString(input.Pickup.Date)
	value.writeString(input.Pickup.Time)
	value.writeString(input.Pickup.Meal)
	value.writeString(input.Pickup.Point)
	value.writeString(input.OrderNote)
	value.writeUint64(uint64(len(input.Items)))
	for _, item := range input.Items {
		value.writeUint64(uint64(item.LineNumber))
		value.writeUint64(item.ProductID)
		value.writeString(item.ProductName)
		value.writeBytes(item.ProductSourceVersion[:])
		value.writeBool(item.ImageObjectKey != "")
		value.writeString(item.ImageObjectKey)
		value.writeInt64(item.OriginalUnitPriceCents)
		value.writeInt64(item.DiscountedUnitPriceCents)
		value.writeInt64(item.Quantity)
		value.writeInt64(item.OriginalSubtotalCents)
		value.writeInt64(item.PayableSubtotalCents)
		value.writeUint64(uint64(len(item.Flavors)))
		for _, flavor := range item.Flavors {
			value.writeString(flavor)
		}
		value.writeString(item.Note)
	}
	value.writeInt64(input.OriginalSubtotalCents)
	value.writeInt64(input.DiscountCents)
	value.writeInt64(input.PayableCents)
	value.writeInt64(input.CreatedAt.UTC().UnixMicro())
	value.writeInt64(input.ExpiresAt.UTC().UnixMicro())
	return value.sum()
}
