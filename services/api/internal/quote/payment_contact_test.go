package quote

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCreateContactContractRejectsMissingAndClientPhone(t *testing.T) {
	valid, ok := decodeCreateRequest([]byte(`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":1}]}`))
	if !ok || valid.ContactName != "张三" {
		t.Fatalf("decode valid contact = %#v/%t", valid, ok)
	}
	for _, body := range []string{
		`{"pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":1}]}`,
		`{"contact_name":"张三","contact_phone":"+1234567890","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":1}]}`,
		`{"contact_name":" 张三 ","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":1}]}`,
	} {
		if _, ok := decodeCreateRequest([]byte(body)); ok {
			t.Fatalf("decodeCreateRequest() accepted missing, client-owned, or untrimmed contact: %s", body)
		}
	}
}

func TestContactNameUsesExactSixtyFourByteBoundary(t *testing.T) {
	validName := strings.Repeat("a", 64)
	validBody := `{"contact_name":"` + validName + `","pickup_date":"2026-08-24","pickup_time":"12:00","items":[{"product_id":"8","quantity":1}]}`
	if input, ok := decodeCreateRequest([]byte(validBody)); !ok || input.ContactName != validName {
		t.Fatalf("64-byte contact rejected: %#v/%t", input, ok)
	}
	invalidBody := strings.Replace(validBody, validName, validName+"b", 1)
	if _, ok := decodeCreateRequest([]byte(invalidBody)); ok {
		t.Fatal("65-byte contact accepted")
	}
}

func TestProviderFreezesServerPrimaryPhoneAndRejectsMissingContactBeforeWrite(t *testing.T) {
	state := &quoteDriverState{sourceRate: 80, sourceDiscountVersion: 11, sourceWhitelistVersion: 7}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
	})
	input := CreateInput{
		PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	}
	if _, err := provider.Create(context.Background(), testWriteMeta(42, "missing-contact"), input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Create(missing contact) error = %v", err)
	}
	state.mu.Lock()
	if state.begins != 0 || state.quoteInserts != 0 || state.itemInserts != 0 {
		state.mu.Unlock()
		t.Fatal("missing contact reached the quote transaction")
	}
	state.mu.Unlock()

	input.ContactName = "张三"
	created, err := provider.Create(context.Background(), testWriteMeta(42, "server-phone"), input)
	if err != nil || created.Quote.Contact != (ContactSnapshot{Name: "张三", Phone: "+1234567890"}) {
		t.Fatalf("Create(server phone snapshot) = %#v/%v", created, err)
	}
	response := newQuoteResponse(created.Quote)
	if response.Contact.Name != "张三" || response.Contact.MaskedPhone != "+******7890" {
		t.Fatalf("HTTP contact projection = %#v", response.Contact)
	}
	body, err := json.Marshal(response)
	if err != nil || strings.Contains(string(body), "+1234567890") {
		t.Fatalf("HTTP contact response disclosed full phone: %s/%v", body, err)
	}
}

func TestContactSnapshotIsCoveredByImmutableDigest(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "contact-digest", input)
	stored.quote.Contact.Name = "篡改姓名"
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{stored: &stored}), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("LoadSnapshotInTx(tampered contact) error = %v", err)
	}
}

func TestCreateRejectsSubCentPaymentAmountWithoutWritingQuote(t *testing.T) {
	state := &quoteDriverState{
		sourceRate: 1, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
		productMode: "one-cent",
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), func() time.Time {
		return time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC)
	})
	_, err := provider.Create(context.Background(), testWriteMeta(42, "zero-payment"), CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 1}},
	})
	if !errors.Is(err, ErrPaymentAmountTooSmall) {
		t.Fatalf("Create(zero payment) error = %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.quoteInserts != 0 || state.itemInserts != 0 || state.commits != 0 || state.rollbacks != 1 {
		t.Fatalf("zero payment transaction effects = %#v", state)
	}
}

func TestFinalizeRejectsPersistedZeroPaymentSnapshot(t *testing.T) {
	input := quoteInputForPrepayTest()
	stored := storedQuoteRecordForTest(42, "zero-finalize", input)
	stored.quote.Discount = DiscountSnapshot{RatePercent: 1, Version: 11}
	stored.quote.Items[0].OriginalUnitPriceCents = 1
	stored.quote.Items[0].DiscountedUnitPriceCents = 0
	stored.quote.Items[0].OriginalSubtotalCents = 2
	stored.quote.Items[0].PayableSubtotalCents = 0
	stored.quote.OriginalSubtotalCents = 2
	stored.quote.DiscountCents = 2
	stored.quote.PayableCents = 0
	stored.quote.Items[0].ProductSourceVersion = hashProductSource(productRecord{
		ID: 8, CategoryID: 2, Name: "套餐", PriceCents: 1,
		MealPeriod: "lunch", Listed: true, CategoryActive: true,
	}, stored.quote.Pickup.Date)
	stored.quote.SnapshotDigest = hashQuoteSnapshot(stored.quote)
	state := &quoteDriverState{
		sourceRate: 1, sourceDiscountVersion: 11, sourceWhitelistVersion: 7,
		productMode: "one-cent", stored: &stored,
	}
	provider := newTestProvider(openQuoteDriverDB(t, state), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.FinalizeForPrepayInTx(context.Background(), transaction, 42, 91, stored.quote.CreatedAt.Add(time.Minute)); !errors.Is(err, ErrPaymentAmountTooSmall) {
		t.Fatalf("FinalizeForPrepayInTx(zero payment) error = %v", err)
	}
}

func TestQuantityAboveNinetyNineRemainsValidWithoutInventoryCap(t *testing.T) {
	input, _, err := normalizeCreateInput(CreateInput{
		ContactName: "张三", PickupDate: "2026-08-24", PickupTime: "12:00",
		Items: []ItemInput{{ProductID: 8, Quantity: 100}},
	}, time.Date(2026, 8, 23, 1, 0, 0, 0, time.UTC))
	if err != nil || input.Items[0].Quantity != 100 {
		t.Fatalf("normalize quantity 100 = %#v/%v", input, err)
	}
}
