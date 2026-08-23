package quote

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFrozenQuoteCreateInterfaceUsesWriteMeta(t *testing.T) {
	var _ interface {
		Create(context.Context, WriteMeta, CreateInput) (CreateResult, error)
	} = (*Provider)(nil)
	provider := reflect.TypeOf((*Provider)(nil))
	want := []string{"Create", "FinalizeForPrepayInTx", "LoadSnapshotInTx", "Read"}
	if provider.NumMethod() != len(want) {
		t.Fatalf("Provider exported method count = %d, want exact %d", provider.NumMethod(), len(want))
	}
	for index, name := range want {
		if provider.Method(index).Name != name {
			t.Fatalf("Provider method %d = %s, want %s", index, provider.Method(index).Name, name)
		}
	}
}

func TestFrozenQuotePackageDoesNotOwnStaffDiscountMutators(t *testing.T) {
	forbidden := map[string]struct{}{
		"AdminProvider": {}, "AdminTarget": {}, "OwnerAuthorizer": {},
		"StaffEntryInput": {}, "StaffEntrySnapshot": {}, "NewAdminProvider": {},
		"SaveDiscountRate": {}, "SaveStaffEntry": {},
		"ReceiptActionDiscountSave": {}, "ReceiptActionStaffEntrySave": {},
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(files, entry.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			if _, blocked := forbidden[identifier.Name]; blocked {
				t.Errorf("quote package owns frozen StaffDiscount symbol %s in %s", identifier.Name, entry.Name())
			}
			return true
		})
	}
}

func TestFrozenSnapshotDigestIncludesExpiryAndCoverObjectKey(t *testing.T) {
	stored := storedQuoteRecordForTest(42, "frozen-digest", quoteInputForPrepayTest())
	baseline := hashQuoteSnapshot(stored.quote)

	changedExpiry := stored.quote
	changedExpiry.ExpiresAt = changedExpiry.ExpiresAt.Add(time.Microsecond)
	if hashQuoteSnapshot(changedExpiry) == baseline {
		t.Fatal("effective expiry was omitted from snapshot digest")
	}

	changedCover := stored.quote
	changedCover.Items = append([]ItemSnapshot(nil), stored.quote.Items...)
	changedCover.Items[0].ImageObjectKey = "products/8/cover.webp"
	if hashQuoteSnapshot(changedCover) == baseline {
		t.Fatal("cover object key was omitted from snapshot digest")
	}
}

func TestLoadSnapshotRejectsTamperedCoverObjectKey(t *testing.T) {
	stored := storedQuoteRecordForTest(42, "cover-tamper", quoteInputForPrepayTest())
	stored.quote.Items[0].ImageObjectKey = "products/8/cover.webp"
	provider := newTestProvider(openQuoteDriverDB(t, &quoteDriverState{stored: &stored}), time.Now)
	transaction := beginQuoteTransaction(t, provider.db)
	defer func() { _ = transaction.Rollback() }()
	if _, err := provider.LoadSnapshotInTx(context.Background(), transaction, 91); !errors.Is(err, ErrSnapshotInvalid) {
		t.Fatalf("LoadSnapshotInTx(tampered cover) error = %v", err)
	}
}
