package quote

import (
	"math"
	"testing"
	"time"
)

func TestStoredQuoteValidationRecalculatesHalfUpPricing(t *testing.T) {
	value := validQuoteForValidation()
	value.Discount = DiscountSnapshot{RatePercent: 50, Version: 1}
	value.Items[0].OriginalUnitPriceCents = 1
	value.Items[0].DiscountedUnitPriceCents = 0
	value.Items[0].Quantity = 1
	value.Items[0].OriginalSubtotalCents = 1
	value.Items[0].PayableSubtotalCents = 0
	value.OriginalSubtotalCents = 1
	value.DiscountCents = 1
	value.PayableCents = 0

	if validStoredQuote(value, 1) {
		t.Fatal("validStoredQuote() accepted a line that skipped half-up rounding")
	}
}

func TestStoredQuoteValidationRejectsArithmeticOverflow(t *testing.T) {
	value := validQuoteForValidation()
	value.Items[0].OriginalUnitPriceCents = math.MaxInt64
	value.Items[0].DiscountedUnitPriceCents = math.MaxInt64
	value.Items[0].Quantity = 2
	value.Items[0].OriginalSubtotalCents = -2
	value.Items[0].PayableSubtotalCents = -2
	value.OriginalSubtotalCents = -2
	value.DiscountCents = 0
	value.PayableCents = -2

	if validStoredQuote(value, 1) {
		t.Fatal("validStoredQuote() accepted overflowed persisted arithmetic")
	}
}

func TestMySQLDateAndTimeSnapshotValuesAreNormalizedStrictly(t *testing.T) {
	date, ok := mysqlDateString(time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC))
	if !ok || date != "2026-08-24" {
		t.Fatalf("mysqlDateString(time.Time) = %q/%t", date, ok)
	}
	date, ok = mysqlDateString([]byte("2026-08-24"))
	if !ok || date != "2026-08-24" {
		t.Fatalf("mysqlDateString([]byte) = %q/%t", date, ok)
	}
	if _, ok := mysqlDateString("2026-8-24"); ok {
		t.Fatal("mysqlDateString() accepted a non-canonical date")
	}

	pickup, ok := mysqlPickupTimeString([]byte("12:00:00"))
	if !ok || pickup != "12:00" {
		t.Fatalf("mysqlPickupTimeString([]byte) = %q/%t", pickup, ok)
	}
	if _, ok := mysqlPickupTimeString("12:00:01"); ok {
		t.Fatal("mysqlPickupTimeString() accepted nonzero seconds")
	}
}

func validQuoteForValidation() Quote {
	var sourceVersion [32]byte
	sourceVersion[0] = 1
	return Quote{
		ID: 1, UserID: 2,
		Contact:  ContactSnapshot{Name: "张三", Phone: "+1234567890"},
		Identity: IdentitySnapshot{Kind: IdentityVisitor, SourceVersion: 1},
		Discount: DiscountSnapshot{RatePercent: 100, Version: 1},
		Store:    StoreSnapshot{Name: "门店", Address: "地址"},
		Pickup:   PickupSnapshot{Date: "2026-08-24", Time: "12:00", Meal: "lunch", Point: "取餐点"},
		Items: []ItemSnapshot{{
			LineNumber: 1, ProductID: 3, ProductName: "商品", ProductSourceVersion: sourceVersion,
			OriginalUnitPriceCents: 1, DiscountedUnitPriceCents: 1, Quantity: 1,
			OriginalSubtotalCents: 1, PayableSubtotalCents: 1, Flavors: []string{},
		}},
		OriginalSubtotalCents: 1, DiscountCents: 0, PayableCents: 1,
		CreatedAt: time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 8, 23, 0, 10, 0, 0, time.UTC),
	}
}
