package quote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthenticatedUserCreatesAndReadsOwnImmutableQuote(t *testing.T) {
	var productVersion [32]byte
	for index := range productVersion {
		productVersion[index] = byte(255 - index)
	}
	createdAt := time.Date(2026, 8, 23, 1, 2, 3, 456789000, time.UTC)
	wantQuote := Quote{
		ID: 91, UserID: 42,
		Contact:   ContactSnapshot{Name: "张三", Phone: "+1234567890"},
		Identity:  IdentitySnapshot{Kind: IdentityStaff, SourceVersion: 7},
		Discount:  DiscountSnapshot{RatePercent: 80, Version: 11},
		Store:     StoreSnapshot{Name: "绥安食品", Address: "党政办公中心后院老食堂"},
		Pickup:    PickupSnapshot{Date: "2026-08-24", Time: "12:00", Meal: "lunch", Point: "党政办公中心后院老食堂北门"},
		OrderNote: "整单少盐",
		Items: []ItemSnapshot{{
			LineNumber: 1, ProductID: 8, ProductName: "套餐", ProductSourceVersion: productVersion, ImageObjectKey: "products/8/cover.webp",
			OriginalUnitPriceCents: 101, DiscountedUnitPriceCents: 81, Quantity: 2,
			OriginalSubtotalCents: 202, PayableSubtotalCents: 162,
			Flavors: []string{"少饭"}, Note: "不要葱",
		}},
		OriginalSubtotalCents: 202, DiscountCents: 40, PayableCents: 162,
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(10 * time.Minute),
	}
	wantQuote.SnapshotDigest = hashQuoteSnapshot(wantQuote)
	authenticator := &sessionAuthenticatorStub{userID: 42}
	application := &quoteApplicationStub{createResult: CreateResult{Quote: wantQuote, Created: true}, readQuote: wantQuote}
	router := quoteTestRouter(NewHandler(authenticator, application))

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(`{"contact_name":"张三","pickup_date":"2026-08-24","pickup_time":"12:00","order_note":"整单少盐","items":[{"product_id":"8","quantity":2,"flavors":["少饭"],"note":"不要葱"}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer opaque-session")
	request.Header.Set("Idempotency-Key", "quote-attempt-1")
	router.ServeHTTP(create, request)

	wantBody := `{"quote":{"id":"91","contact":{"name":"张三","masked_phone":"+******7890"},"identity":{"kind":"STAFF"},"discount":{"rate_percent":80},"store":{"name":"绥安食品","address":"党政办公中心后院老食堂"},"pickup":{"date":"2026-08-24","time":"12:00","meal_period":"lunch","point":"党政办公中心后院老食堂北门"},"order_note":"整单少盐","items":[{"line_number":1,"product_id":"8","name":"套餐","image_object_key":"products/8/cover.webp","original_unit_price_cents":101,"discounted_unit_price_cents":81,"quantity":2,"original_subtotal_cents":202,"payable_subtotal_cents":162,"flavors":["少饭"],"note":"不要葱"}],"original_subtotal_cents":202,"discount_cents":40,"payable_cents":162,"created_at":"2026-08-23T01:02:03.456789Z","expires_at":"2026-08-23T01:12:03.456789Z"}}`
	assertQuoteResponse(t, create, http.StatusCreated, wantBody)
	if authenticator.calls != 1 || application.createCalls != 1 || application.createMeta != testWriteMeta(42, "quote-attempt-1") {
		t.Fatalf("create dependencies = auth:%d app:%d meta:%#v", authenticator.calls, application.createCalls, application.createMeta)
	}
	if application.createInput.ContactName != "张三" || application.createInput.Items[0].ProductID != 8 || application.createInput.Items[0].Quantity != 2 {
		t.Fatalf("create input = %#v", application.createInput)
	}

	read := httptest.NewRecorder()
	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/91", nil)
	readRequest.Header.Set("Authorization", "Bearer opaque-session")
	router.ServeHTTP(read, readRequest)
	assertQuoteResponse(t, read, http.StatusOK, wantBody)
	if application.readCalls != 1 || application.readUserID != 42 || application.readID != 91 {
		t.Fatalf("read dependencies = calls:%d user:%d id:%d", application.readCalls, application.readUserID, application.readID)
	}
}

type sessionAuthenticatorStub struct {
	userID uint64
	err    error
	calls  int
}

func (stub *sessionAuthenticatorStub) Authenticate(context.Context, string) (uint64, error) {
	stub.calls++
	return stub.userID, stub.err
}

type quoteApplicationStub struct {
	createResult CreateResult
	createErr    error
	readQuote    Quote
	readErr      error
	createCalls  int
	createMeta   WriteMeta
	createInput  CreateInput
	readCalls    int
	readUserID   uint64
	readID       uint64
}

func (stub *quoteApplicationStub) Create(_ context.Context, meta WriteMeta, input CreateInput) (CreateResult, error) {
	stub.createCalls++
	stub.createMeta = meta
	stub.createInput = input
	return stub.createResult, stub.createErr
}

func (stub *quoteApplicationStub) Read(_ context.Context, userID, quoteID uint64) (Quote, error) {
	stub.readCalls++
	stub.readUserID = userID
	stub.readID = quoteID
	return stub.readQuote, stub.readErr
}

func quoteTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(func(ctx *gin.Context) {
		ctx.Set("request_id", "request-quote-attempt-1")
		ctx.Next()
	})
	handler.RegisterRoutes(router)
	return router
}

func assertQuoteResponse(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status || strings.TrimSpace(response.Body.String()) != body {
		t.Fatalf("response = %d/%q, want %d/%q", response.Code, strings.TrimSpace(response.Body.String()), status, body)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response headers = content-type:%q cache:%q", response.Header().Get("Content-Type"), response.Header().Get("Cache-Control"))
	}
}
