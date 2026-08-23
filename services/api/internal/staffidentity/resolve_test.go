package staffidentity_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/staffidentity"
)

func TestResolveEnabledPrimaryIsStaffIgnoringName(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 41,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: "Unrelated Record Name", Enabled: true}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 41})
}

func TestResolveNoMatchIsVersionedVisitor(t *testing.T) {
	input := staffidentity.Input{PrimaryPhone: "+1234567890", WhitelistVersion: 42}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 42})
}

func TestResolveDisabledPrimaryIsVisitor(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 43,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: "Record Name", Enabled: false}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 43})
}

func TestResolveDistinctExtraPhoneCanMatchAfterDisabledPrimary(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Claim Name"},
		WhitelistVersion: 44,
		CandidateEntries: []staffidentity.Entry{
			{Phone: "+1234567890", Name: "Primary Name", Enabled: false},
			{Phone: "+1987654321", Name: "Claim Name", Enabled: true},
		},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 44})
}

func TestResolveExtraNameMismatchIsVisitor(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Claim Name"},
		WhitelistVersion: 45,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "Different Name", Enabled: true}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 45})
}

func TestResolveDisabledExtraIsVisitor(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Claim Name"},
		WhitelistVersion: 46,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "Claim Name", Enabled: false}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 46})
}

func TestResolveFoldsNameWidth(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "ＡＢＣ"},
		WhitelistVersion: 47,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "ABC", Enabled: true}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 47})
}

func TestResolveDeletesPostFoldASCIISpaces(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Ａ　B C"},
		WhitelistVersion: 48,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "AＢC", Enabled: true}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 48})
}

func TestResolveComposesNamesWithNFC(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Cafe\u0301ﾊﾞ"},
		WhitelistVersion: 49,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "Caféバ", Enabled: true}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 49})
}

func TestResolveDoesNotApplyNFKC(t *testing.T) {
	compatibilityPairs := [][2]string{{"①", "1"}, {"ﬃ", "ffi"}}
	for _, pair := range compatibilityPairs {
		input := staffidentity.Input{
			PrimaryPhone:     "+1234567890",
			Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: pair[0]},
			WhitelistVersion: 50,
			CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: pair[1], Enabled: true}},
		}
		requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 50})
	}
}

func TestResolveRejectsInvalidPrimaryPhone(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "123456",
		WhitelistVersion: 51,
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_PRIMARY_PHONE")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidPrimaryPhone {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidPrimaryPhone)
	}
	if strings.Contains(err.Error(), input.PrimaryPhone) {
		t.Fatalf("Resolve() error disclosed invalid primary phone: %q", err)
	}
}

func TestResolveRejectsInvalidExtraPhone(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "1987654321", Name: "Valid Extra Name"},
		WhitelistVersion: 52,
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_EXTRA_CLAIM")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidExtraClaim {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidExtraClaim)
	}
	if strings.Contains(err.Error(), input.Extra.Phone) {
		t.Fatalf("Resolve() error disclosed invalid extra phone: %q", err)
	}
	if strings.Contains(err.Error(), input.Extra.Name) {
		t.Fatalf("Resolve() error disclosed extra name: %q", err)
	}
}

func TestResolveRejectsExtraNameEmptyAfterNormalization(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "\u3000"},
		WhitelistVersion: 53,
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_EXTRA_CLAIM")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidExtraClaim {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidExtraClaim)
	}
	if strings.Contains(err.Error(), input.Extra.Name) {
		t.Fatalf("Resolve() error disclosed extra name: %q", err)
	}
	if strings.Contains(err.Error(), input.Extra.Phone) {
		t.Fatalf("Resolve() error disclosed extra phone: %q", err)
	}
}

func TestResolveRejectsInvalidUTF8ExtraName(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: string([]byte{0xff})},
		WhitelistVersion: 54,
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_EXTRA_CLAIM")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidExtraClaim {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidExtraClaim)
	}
	if strings.Contains(err.Error(), input.PrimaryPhone) {
		t.Fatalf("Resolve() error disclosed primary phone: %q", err)
	}
	if strings.Contains(err.Error(), input.Extra.Phone) {
		t.Fatalf("Resolve() error disclosed extra phone: %q", err)
	}
	if strings.Contains(err.Error(), input.Extra.Name) {
		t.Fatalf("Resolve() error disclosed invalid extra name: %q", err)
	}
}

func TestResolveRejectsZeroWhitelistVersion(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 0,
		CandidateEntries: nil,
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if err.Error() != "staffidentity: INVALID_WHITELIST_SNAPSHOT" {
		t.Fatalf("Resolve() error text = %q, want stable redacted category", err)
	}
}

func TestResolveRejectsWhitelistEntryWithInvalidPhone(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 55,
		CandidateEntries: []staffidentity.Entry{{Phone: "1987654321", Name: "Private Entry Name", Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Phone) {
		t.Fatalf("Resolve() error disclosed entry phone: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Name) {
		t.Fatalf("Resolve() error disclosed entry name: %q", err)
	}
}

func TestResolveRejectsWhitelistEntryWithEmptyNormalizedName(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 56,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: "\u3000", Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Phone) {
		t.Fatalf("Resolve() error disclosed entry phone: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Name) {
		t.Fatalf("Resolve() error disclosed entry name: %q", err)
	}
}

func TestResolveRejectsWhitelistEntryWithInvalidUTF8Name(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 57,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: string([]byte{0xff}), Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Phone) {
		t.Fatalf("Resolve() error disclosed entry phone: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Name) {
		t.Fatalf("Resolve() error disclosed invalid entry name: %q", err)
	}
}

func TestResolveRejectsUnrelatedWhitelistEntry(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 58,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1987654321", Name: "Unrelated Private Name", Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Phone) {
		t.Fatalf("Resolve() error disclosed unrelated entry phone: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Name) {
		t.Fatalf("Resolve() error disclosed unrelated entry name: %q", err)
	}
}

func TestResolveRejectsDuplicateWhitelistPhone(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		WhitelistVersion: 59,
		CandidateEntries: []staffidentity.Entry{
			{Phone: "+1234567890", Name: "First Private Name", Enabled: true},
			{Phone: "+1234567890", Name: "Second Private Name", Enabled: false},
		},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_WHITELIST_SNAPSHOT")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidWhitelistSnapshot {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidWhitelistSnapshot)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Phone) {
		t.Fatalf("Resolve() error disclosed duplicate entry phone: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[0].Name) {
		t.Fatalf("Resolve() error disclosed first entry name: %q", err)
	}
	if strings.Contains(err.Error(), input.CandidateEntries[1].Name) {
		t.Fatalf("Resolve() error disclosed second entry name: %q", err)
	}
}

func TestResolveRejectsInvalidPrimaryBeforeLowerPriorityEvidence(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "123456",
		Extra:            &staffidentity.ExtraClaim{Phone: "invalid-extra", Name: "\u3000"},
		WhitelistVersion: 0,
		CandidateEntries: []staffidentity.Entry{{Phone: "invalid-entry", Name: "", Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_PRIMARY_PHONE")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidPrimaryPhone {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidPrimaryPhone)
	}
	if err.Error() != "staffidentity: INVALID_PRIMARY_PHONE" {
		t.Fatalf("Resolve() error text = %q, want stable redacted category", err)
	}
}

func TestResolveRejectsInvalidExtraBeforeWhitelistEvidence(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "invalid-extra", Name: "Valid Extra Name"},
		WhitelistVersion: 0,
		CandidateEntries: []staffidentity.Entry{{Phone: "invalid-entry", Name: "", Enabled: true}},
	}

	got, err := staffidentity.Resolve(input)
	if got != (staffidentity.Snapshot{}) {
		t.Fatalf("Resolve() snapshot = %#v, want exact zero Snapshot", got)
	}
	if err == nil {
		t.Fatal("Resolve() error = nil, want INVALID_EXTRA_CLAIM")
	}
	typedErr, ok := err.(*staffidentity.Error)
	if !ok {
		t.Fatalf("Resolve() error type = %T, want *staffidentity.Error", err)
	}
	if typedErr.Kind() != staffidentity.ErrorInvalidExtraClaim {
		t.Fatalf("Resolve() error kind = %q, want %q", typedErr.Kind(), staffidentity.ErrorInvalidExtraClaim)
	}
	if err.Error() != "staffidentity: INVALID_EXTRA_CLAIM" {
		t.Fatalf("Resolve() error text = %q, want stable redacted category", err)
	}
}

func TestResolveDoesNotMutateInputAndIsDeterministic(t *testing.T) {
	extra := &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Ａ　B C"}
	entries := make([]staffidentity.Entry, 0, 4)
	entries = append(entries,
		staffidentity.Entry{Phone: "+1234567890", Name: "Primary Private Name", Enabled: false},
		staffidentity.Entry{Phone: "+1987654321", Name: "AＢC", Enabled: true},
	)
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            extra,
		WhitelistVersion: 60,
		CandidateEntries: entries,
	}
	originalExtraPointer := input.Extra
	originalExtra := *input.Extra
	originalEntries := slices.Clone(input.CandidateEntries)
	originalEntriesPointer := &input.CandidateEntries[0]
	originalEntriesLength := len(input.CandidateEntries)
	originalEntriesCapacity := cap(input.CandidateEntries)
	want := staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 60}

	for attempt := 0; attempt < 32; attempt++ {
		got, err := staffidentity.Resolve(input)
		if err != nil {
			t.Fatalf("Resolve() attempt %d error = %v, want nil", attempt, err)
		}
		if got != want {
			t.Fatalf("Resolve() attempt %d = %#v, want %#v", attempt, got, want)
		}
		if input.Extra != originalExtraPointer || *input.Extra != originalExtra {
			t.Fatalf("Resolve() attempt %d mutated Extra: %#v", attempt, input.Extra)
		}
		if len(input.CandidateEntries) != originalEntriesLength || cap(input.CandidateEntries) != originalEntriesCapacity {
			t.Fatalf("Resolve() attempt %d changed CandidateEntries header", attempt)
		}
		if &input.CandidateEntries[0] != originalEntriesPointer {
			t.Fatalf("Resolve() attempt %d replaced CandidateEntries backing array", attempt)
		}
		if !slices.Equal(input.CandidateEntries, originalEntries) {
			t.Fatalf("Resolve() attempt %d mutated CandidateEntries: %#v", attempt, input.CandidateEntries)
		}
	}
}

func TestResolveSamePrimaryExtraDoesNotCreateFallback(t *testing.T) {
	input := staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1234567890", Name: "Matching Private Name"},
		WhitelistVersion: 61,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: "Matching Private Name", Enabled: false}},
	}
	requireSnapshot(t, input, staffidentity.Snapshot{Kind: staffidentity.KindVisitor, WhitelistVersion: 61})
}

func requireSnapshot(t *testing.T, input staffidentity.Input, want staffidentity.Snapshot) {
	t.Helper()
	got, err := staffidentity.Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != want {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}
