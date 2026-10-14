package catalog

import (
	"context"
	"errors"
	"testing"
)

type imageBatch struct {
	calls int
	fail  bool
}

func (b *imageBatch) PublicURLs(_ context.Context, keys []string) (map[string]string, error) {
	b.calls++
	if b.fail {
		return nil, errors.New("pin store unavailable")
	}
	return map[string]string{keys[0]: "https://example.invalid/confirmed"}, nil
}
func TestImagesCannotBlockCatalogFacts(t *testing.T) {
	b := &imageBatch{}
	h := NewHandler(nil, WithPublicURLs(b))
	products := []Product{{ID: 1, CategoryID: 1, Name: "dish", MealPeriod: "all", Listed: true, OriginalUnitPriceCents: 100, ImageObjectKeys: []string{"a", "missing"}}}
	for _, fail := range []bool{false, true} {
		b.fail = fail
		out, ok := h.projectProducts(context.Background(), products, 0)
		if !ok || len(out) != 1 || out[0].OriginalUnitPriceCents != 100 {
			t.Fatal("image failure lost menu facts")
		}
		want := 1
		if fail {
			want = 0
		}
		if len(out[0].Images) != want {
			t.Fatal("unconfirmed image exposed")
		}
	}
	if b.calls != 2 {
		t.Fatal("must batch each catalog projection")
	}
	if _, ok := h.projectProducts(context.Background(), products, 123); ok {
		t.Fatal("pricing service failure must remain fail closed")
	}
	products[0].OriginalUnitPriceCents = 0
	if _, ok := h.projectProducts(context.Background(), products, 0); ok {
		t.Fatal("invalid product facts accepted")
	}
}
