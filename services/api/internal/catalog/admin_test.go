package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeAdminCatalog struct {
	command CatalogCommand
	meta    WriteMeta
}

func (*fakeAdminCatalog) ListCategories(context.Context) ([]AdminCategory, error) {
	return []AdminCategory{{ID: 1, Name: "主食", Enabled: true, ProductCount: 2}}, nil
}
func (*fakeAdminCatalog) ListProducts(context.Context, AdminQuery) ([]AdminProduct, error) {
	return nil, nil
}
func (*fakeAdminCatalog) GetProduct(context.Context, uint64, AdminQuery) (AdminProduct, error) {
	return AdminProduct{}, ErrAdminNotFound
}
func (f *fakeAdminCatalog) Execute(_ context.Context, m WriteMeta, c CatalogCommand) (CatalogResult, error) {
	f.meta = m
	f.command = c
	return CatalogResult{Category: &AdminCategory{ID: 2, Name: c.Name, Enabled: true}}, nil
}
func adminCatalogEngine(app AdminApplication) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	g := e.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Set("actor_user_id", uint64(9)); c.Next() })
	NewAdminHandler(app).RegisterRoutes(g)
	return e
}
func TestAdminCategoryCommandRequiresReceiptKeyAndMapsCommand(t *testing.T) {
	fake := &fakeAdminCatalog{}
	e := adminCatalogEngine(fake)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/categories", strings.NewReader(`{"name":"主食"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "op-1")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fake.meta.ActorUserID != 9 || fake.meta.IdempotencyKey != "op-1" || fake.command.Kind != CommandCreateCategory {
		t.Fatalf("unexpected command: %#v %#v", fake.meta, fake.command)
	}
}
func TestAdminCatalogRejectsUnknownWriteField(t *testing.T) {
	e := adminCatalogEngine(&fakeAdminCatalog{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/categories", strings.NewReader(`{"name":"主食","specification":"禁止写"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "op-1")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
