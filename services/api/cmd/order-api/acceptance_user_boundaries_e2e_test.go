package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/orderadvance"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/internal/wechat"
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
)

const (
	boundaryVisitorPhone      = "+8613810000001"
	boundaryExtraPrimaryPhone = "+8613810000002"
	boundaryUnboundPhone      = "+8613810000003"
	boundaryExtraStaffPhone   = "+8613920000002"
)

// TestAcceptanceUserBoundariesAreFailClosed crosses the root-composed HTTP
// seam and a freshly migrated v1-v44 MySQL schema. It is the L2 companion to
// the rendered Mini Program boundary cases BE-01--06 and BE-22--26.
func TestAcceptanceUserBoundariesAreFailClosed(t *testing.T) {
	db := acceptanceFreshMySQL(t)
	boundarySeedBootstrap(t, db)

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	today := time.Now().In(shanghai).Format("2006-01-02")
	tomorrow := time.Now().In(shanghai).AddDate(0, 0, 1).Format("2006-01-02")
	runtimeClock := boundaryAt(t, today, "04:00", shanghai)
	now := func() time.Time { return runtimeClock }

	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{
		"boundary-visitor-login": "boundary-visitor-openid",
		"boundary-extra-login":   "boundary-extra-openid",
		"boundary-unbound-login": "boundary-unbound-openid",
		"boundary-owner-login":   "boundary-owner-openid",
	}}
	phoneProvider := boundaryPhoneProvider{phones: map[string]string{
		"boundary-visitor-openid": boundaryVisitorPhone,
		"boundary-extra-openid":   boundaryExtraPrimaryPhone,
		"boundary-unbound-openid": boundaryUnboundPhone,
		"boundary-owner-openid":   acceptanceOwnerPhone,
	}}
	identityRepository := identity.NewRepository(db)
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	pricing := staffdiscount.NewMySQLPricing(db)
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	merchantAdminHandler := merchantidentity.NewAdminHandler(merchantAdmin)

	paymentProvider := &boundaryPaymentProvider{delegate: newLocalPaymentProvider(now)}
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-boundary-local", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	notifications := subscription.New(db, subscription.NewFakeProvider())
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatalf("compose boundary redemption cipher: %v", err)
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)

	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		storefront.NewHandler(storefront.NewRepository(db)),
		catalog.NewHandler(catalog.NewRepository(db), catalog.WithAuthenticator(miniAuth), catalog.WithPricing(pricing), catalog.WithClock(now)),
		menu.NewHandler(menu.NewRepository(db), now, menu.WithAuthenticator(miniAuth), menu.WithPricing(pricing)),
		identity.NewHandler(sessions),
		identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication),
		newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, paymentProvider)),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newAdminRoutes(sessions, merchantAdmin, merchantAdminHandler, []adminGroupRegistrar{
			catalog.NewAdminHandler(catalog.NewMySQLAdminApplication(db, nil)),
			storefront.NewAdminHandler(storefront.NewMySQLAdminApplication(db)),
			staffdiscount.NewHandler(staffdiscount.NewMySQLApplication(db)),
		}, nil, nil),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()

	visitorToken := acceptanceMiniSession(t, client, server.URL, "boundary-visitor-login")
	extraToken := acceptanceMiniSession(t, client, server.URL, "boundary-extra-login")
	unboundToken := acceptanceMiniSession(t, client, server.URL, "boundary-unbound-login")
	ownerToken := acceptanceMiniSession(t, client, server.URL, "boundary-owner-login")
	acceptanceBindPhone(t, client, server.URL, visitorToken, "visitor-phone")
	acceptanceBindPhone(t, client, server.URL, extraToken, "extra-phone")
	acceptanceBindPhone(t, client, server.URL, ownerToken, "owner-phone")
	acceptanceMerchantLogin(t, client, server.URL, ownerToken, "OWNER")
	pcToken := acceptanceImportPCSession(t, client, server.URL, ownerToken)

	boundaryConfigure(t, client, server.URL, pcToken, "boundary-config-open", "open", today, tomorrow)
	categoryID := boundaryCreateCategory(t, client, server.URL, pcToken)
	lunchProductID := boundaryCreateProduct(t, client, server.URL, pcToken, categoryID, "边界午餐", 1250, "lunch", "boundary-product-lunch")
	dinnerProductID := boundaryCreateProduct(t, client, server.URL, pcToken, categoryID, "边界晚餐", 1500, "dinner", "boundary-product-dinner")
	hiddenProductID := boundaryCreateProduct(t, client, server.URL, pcToken, categoryID, "边界下架餐", 990, "lunch", "boundary-product-hidden")
	boundaryProductStatus(t, client, server.URL, pcToken, hiddenProductID, "OFF", "boundary-product-hidden-off")
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/staff-whitelist", pcToken, "boundary-staff-create", map[string]any{
		"name": "验收员工", "phone": strings.TrimPrefix(boundaryExtraStaffPhone, "+86"),
	}, http.StatusCreated)
	acceptanceHTTP(t, client, http.MethodPut, server.URL+"/api/v1/admin/discount-rate", pcToken, "boundary-rate-80", map[string]any{
		"rate_percent": 80,
	}, http.StatusOK)

	// BE-01: a closed store remains browseable, but no Quote, prepayment or
	// order may be created from the same server-owned facts.
	baseline := boundaryTransactionCounts(t, db)
	boundaryConfigure(t, client, server.URL, pcToken, "boundary-config-closed", "closed", today, tomorrow)
	storeView := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/storefront/settings", "", "", nil, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, storeView, "storefront"), "business_status") != "closed" {
		t.Fatal("BE-01 storefront did not expose the closed business fact")
	}
	closedMenu := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu?date="+today+"&time=05:00", "", "", nil, http.StatusOK)
	closedStatus := acceptanceObject(t, closedMenu, "store_status")
	if acceptanceString(t, closedStatus, "business_status") != "closed" || acceptanceBool(t, closedStatus, "meal_available") {
		t.Fatal("BE-01 closed menu was not browse-only")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", visitorToken, "boundary-closed-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1), http.StatusConflict), "QUOTE_SELECTION_UNAVAILABLE")
	boundaryAssertTransactionCounts(t, db, baseline, "BE-01 closed store")
	boundaryConfigure(t, client, server.URL, pcToken, "boundary-config-reopen", "open", today, tomorrow)

	// BE-02: at the dinner cutoff, both of today's meal groups are unavailable
	// while tomorrow remains a real service date. The menu still exposes facts.
	runtimeClock = boundaryAt(t, today, "17:00", shanghai)
	pickups := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu/pickup-options", "", "", nil, http.StatusOK)
	dates := boundaryObjects(t, pickups, "dates")
	if len(dates) != 2 || acceptanceString(t, dates[0], "date") != today || acceptanceBool(t, dates[0], "available") || !acceptanceBool(t, dates[1], "available") {
		t.Fatalf("BE-02 pickup date availability = %#v", dates)
	}
	for _, mealPeriod := range boundaryObjects(t, dates[0], "meal_periods") {
		if acceptanceBool(t, mealPeriod, "available") {
			t.Fatal("BE-02 returned a false available meal period after all cutoffs")
		}
	}
	cutoffMenu := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu?date="+today+"&time=17:00", "", "", nil, http.StatusOK)
	if !acceptanceBool(t, acceptanceObject(t, cutoffMenu, "store_status"), "cutoff_passed") {
		t.Fatal("BE-02 cutoff menu omitted the server cutoff fact")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", visitorToken, "boundary-cutoff-quote", boundaryQuoteBody(today, "17:00", dinnerProductID, 1), http.StatusConflict), "QUOTE_SELECTION_UNAVAILABLE")
	boundaryAssertTransactionCounts(t, db, baseline, "BE-02 cutoff")
	runtimeClock = boundaryAt(t, today, "04:00", shanghai)

	// BE-03: date-scoped sold-out stays visible and disabled; an unlisted
	// product disappears; Quote revalidates both conditions server-side.
	boundaryProductSoldOut(t, client, server.URL, pcToken, lunchProductID, today, true, "boundary-soldout-on")
	soldOutMenu := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu?date="+today+"&time=05:00", visitorToken, "", nil, http.StatusOK)
	menuProducts := boundaryMenuProducts(t, soldOutMenu)
	if !acceptanceBool(t, boundaryProduct(t, menuProducts, lunchProductID), "sold_out") || boundaryHasProduct(menuProducts, hiddenProductID) {
		t.Fatal("BE-03 menu did not distinguish sold-out from unlisted")
	}
	detail := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/catalog/products/"+lunchProductID+"?date="+today+"&time=05:00", visitorToken, "", nil, http.StatusOK)
	if !acceptanceBool(t, acceptanceObject(t, detail, "product"), "sold_out") {
		t.Fatal("BE-03 detail omitted the sold-out fact")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", visitorToken, "boundary-soldout-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1), http.StatusConflict), "QUOTE_SELECTION_UNAVAILABLE")
	boundaryProductSoldOut(t, client, server.URL, pcToken, lunchProductID, today, false, "boundary-soldout-off")
	boundaryAssertTransactionCounts(t, db, baseline, "BE-03 sold-out/unlisted")

	// BE-04: a dinner-only item is absent from lunch public reads and rejected
	// by Quote, while the authenticated OWNER still sees the same catalog fact.
	lunchMenu := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu?date="+today+"&time=05:00", visitorToken, "", nil, http.StatusOK)
	if boundaryHasProduct(boundaryMenuProducts(t, lunchMenu), dinnerProductID) {
		t.Fatal("BE-04 lunch menu exposed a dinner-only item")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/catalog/products/"+dinnerProductID+"?date="+today+"&time=05:00", visitorToken, "", nil, http.StatusNotFound), "PRODUCT_NOT_FOUND")
	adminDinner := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/products/"+dinnerProductID+"?service_date="+today, pcToken, "", nil, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, adminDinner, "product"), "meal_period") != "dinner" {
		t.Fatal("BE-04 OWNER catalog did not retain the dinner product")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", visitorToken, "boundary-meal-mismatch", boundaryQuoteBody(today, "05:00", dinnerProductID, 1), http.StatusConflict), "QUOTE_SELECTION_UNAVAILABLE")
	boundaryAssertTransactionCounts(t, db, baseline, "BE-04 meal mismatch")

	// BE-05: Quote just before cutoff is not authority to prepay at or after
	// cutoff. The provider seam remains untouched because no prepayment exists.
	runtimeClock = boundaryAt(t, today, "04:29", shanghai).Add(59 * time.Second)
	cutoffQuote := boundaryCreateQuote(t, client, server.URL, visitorToken, "boundary-before-cutoff", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	runtimeClock = boundaryAt(t, today, "04:30", shanghai)
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/prepay", visitorToken, "boundary-at-cutoff-prepay", map[string]any{"quote_id": cutoffQuote}, http.StatusConflict), "QUOTE_UNAVAILABLE")
	if boundaryTransactionCounts(t, db).prepayments != 0 {
		t.Fatal("BE-05 created a prepayment at cutoff")
	}
	if paymentProvider.creates.Load() != 0 {
		t.Fatal("BE-05 called the payment provider at cutoff")
	}

	// BE-06: price and sold-out drift after Quote both require a new Quote.
	// Only the Quote rows exist; no provider request can be durable.
	runtimeClock = boundaryAt(t, today, "04:00", shanghai)
	priceQuote := boundaryCreateQuote(t, client, server.URL, visitorToken, "boundary-price-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	boundaryUpdateProduct(t, client, server.URL, pcToken, categoryID, lunchProductID, "边界午餐", 1300, "lunch", "boundary-price-change")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/prepay", visitorToken, "boundary-price-drift-prepay", map[string]any{"quote_id": priceQuote}, http.StatusConflict), "QUOTE_UNAVAILABLE")
	boundaryUpdateProduct(t, client, server.URL, pcToken, categoryID, lunchProductID, "边界午餐", 1250, "lunch", "boundary-price-restore")
	soldOutQuote := boundaryCreateQuote(t, client, server.URL, visitorToken, "boundary-pre-soldout-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	boundaryProductSoldOut(t, client, server.URL, pcToken, lunchProductID, today, true, "boundary-after-quote-soldout")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/prepay", visitorToken, "boundary-soldout-drift-prepay", map[string]any{"quote_id": soldOutQuote}, http.StatusConflict), "QUOTE_UNAVAILABLE")
	boundaryProductSoldOut(t, client, server.URL, pcToken, lunchProductID, today, false, "boundary-after-quote-restock")
	if boundaryTransactionCounts(t, db).prepayments != 0 {
		t.Fatal("BE-06 current-fact drift reached the payment provider boundary")
	}
	if paymentProvider.creates.Load() != 0 {
		t.Fatal("BE-06 current-fact drift called the payment provider")
	}

	// BE-22: an unbound user gets no Quote. A rejected trusted phone exchange
	// stays unbound; a later accepted exchange can continue the same checkout.
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", unboundToken, "boundary-unbound-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1), http.StatusConflict), "PRIMARY_PHONE_REQUIRED")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/me/bind-phone", unboundToken, "", map[string]any{"code": "rejected"}, http.StatusUnprocessableEntity), "PHONE_CODE_REJECTED")
	unboundStatus := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/me/primary-phone", unboundToken, "", nil, http.StatusOK)
	if acceptanceBool(t, unboundStatus, "primary_phone_bound") {
		t.Fatal("BE-22 rejected phone exchange produced a false bound state")
	}
	acceptanceBindPhone(t, client, server.URL, unboundToken, "unbound-phone-accepted")
	unboundQuote := boundaryCreateQuote(t, client, server.URL, unboundToken, "boundary-bound-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	if unboundQuote == "" {
		t.Fatal("BE-22 did not resume checkout after accepted phone binding")
	}

	// BE-23 and BE-24: phone-only whitelist match remains VISITOR. The same
	// extra phone with the normalized exact name becomes STAFF; visitor Quotes
	// retain original price and never present a fake discount.
	mismatch := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/me/extra-phone", extraToken, "boundary-extra-mismatch", map[string]any{
		"phone": strings.TrimPrefix(boundaryExtraStaffPhone, "+86"), "name": "错误姓名",
	}, http.StatusOK)
	boundaryAssertPricingIdentity(t, acceptanceObject(t, mismatch, "pricing_identity"), "VISITOR", 100, "BE-23 mismatch")
	visitorQuote := boundaryCreateQuoteView(t, client, server.URL, extraToken, "boundary-extra-visitor-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	boundaryAssertQuotePricing(t, visitorQuote, "VISITOR", 100, 1250, "BE-24 visitor")
	matched := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/me/extra-phone", extraToken, "boundary-extra-match", map[string]any{
		"phone": strings.TrimPrefix(boundaryExtraStaffPhone, "+86"), "name": "　验收员工　",
	}, http.StatusOK)
	boundaryAssertPricingIdentity(t, acceptanceObject(t, matched, "pricing_identity"), "STAFF", 80, "BE-23 match")
	staffQuote := boundaryCreateQuoteView(t, client, server.URL, extraToken, "boundary-extra-staff-quote", boundaryQuoteBody(today, "05:00", lunchProductID, 1))
	boundaryAssertQuotePricing(t, staffQuote, "STAFF", 80, 1000, "BE-23 staff")

	// BE-25 server shield: even if a client sends what should have been an
	// impossible empty/invalid cart request, quantities fail closed with zero
	// Quote, prepayment and order rows.
	beforeInvalid := boundaryTransactionCounts(t, db)
	invalidBodies := []struct {
		key    string
		body   map[string]any
		status int
		code   string
	}{
		{"boundary-empty-items", map[string]any{"contact_name": "边界用户", "pickup_date": today, "pickup_time": "05:00", "items": []any{}}, http.StatusBadRequest, "INVALID_REQUEST"},
		{"boundary-zero-quantity", boundaryQuoteBody(today, "05:00", lunchProductID, 0), http.StatusBadRequest, "INVALID_REQUEST"},
		{"boundary-negative-quantity", boundaryQuoteBody(today, "05:00", lunchProductID, -1), http.StatusBadRequest, "INVALID_REQUEST"},
		{"boundary-overflow-quantity", boundaryQuoteBody(today, "05:00", lunchProductID, math.MaxInt64), http.StatusServiceUnavailable, "QUOTE_UNAVAILABLE"},
	}
	for _, invalid := range invalidBodies {
		boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", visitorToken, invalid.key, invalid.body, invalid.status), invalid.code)
	}
	boundaryAssertTransactionCounts(t, db, beforeInvalid, "BE-25 invalid carts")

	// BE-26: a RESERVED order has no redemption token. Wrong/current-date
	// code attempts cannot advance it; after READY, scan replay is monotonic.
	runtimeClock = boundaryAt(t, today, "04:00", shanghai)
	todayOrderID := boundaryMaterializeOrder(t, client, server.URL, visitorToken, "boundary-today-order", today, lunchProductID)
	reserved := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+todayOrderID, visitorToken, "", nil, http.StatusOK)
	boundaryAssertOrderNoToken(t, reserved, "RESERVED", "BE-26 reserved")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/code", ownerToken, "boundary-wrong-code", map[string]any{"pickup_number": "9999"}, http.StatusNotFound), "ORDER_NOT_FOUND")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/code", ownerToken, "boundary-not-ready-code", map[string]any{"pickup_number": "0001"}, http.StatusConflict), "TRANSITION_NOT_ALLOWED")
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders", "", "", nil, http.StatusUnauthorized), "UNAUTHENTICATED")

	runtimeClock = boundaryAt(t, today, "04:30", shanghai)
	production, err := orderadvance.New(db).RunProductionDue(t.Context(), runtimeClock, 10)
	if err != nil || production.Advanced != 1 {
		t.Fatalf("BE-26 production boundary = %#v/%v", production, err)
	}
	ready := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/merchant/orders/"+todayOrderID+"/ready", ownerToken, "boundary-ready", map[string]any{}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, ready, "order"), "state") != "READY_FOR_PICKUP" {
		t.Fatal("BE-26 merchant ready did not expose READY_FOR_PICKUP")
	}
	readyForUser := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+todayOrderID, visitorToken, "", nil, http.StatusOK)
	redemptionToken := acceptanceString(t, acceptanceObject(t, readyForUser, "order"), "redemption_token")
	firstRedeem := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/scan", ownerToken, "boundary-redeem-replay", map[string]any{"token": redemptionToken}, http.StatusOK)
	replayedRedeem := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/scan", ownerToken, "boundary-redeem-replay", map[string]any{"token": redemptionToken}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, firstRedeem, "order"), "state") != "COMPLETED" || acceptanceString(t, acceptanceObject(t, replayedRedeem, "order"), "state") != "COMPLETED" {
		t.Fatal("BE-26 redemption replay was not monotonic")
	}

	// The next service date starts again at 0001. Manual code redemption is
	// still scoped to the real current business date and cannot touch it.
	runtimeClock = boundaryAt(t, tomorrow, "04:00", shanghai)
	tomorrowOrderID := boundaryMaterializeOrder(t, client, server.URL, unboundToken, "boundary-tomorrow-order", tomorrow, lunchProductID)
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/code", ownerToken, "boundary-cross-date-code", map[string]any{"pickup_number": "0001"}, http.StatusConflict), "TRANSITION_NOT_ALLOWED")
	tomorrowOrder := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+tomorrowOrderID, unboundToken, "", nil, http.StatusOK)
	boundaryAssertOrderNoToken(t, tomorrowOrder, "RESERVED", "BE-26 cross-date")

	// Last, make the shared database dependency unavailable. Authentication
	// fails closed as 503 and cannot fabricate an order/token response.
	if err := db.Close(); err != nil {
		t.Fatal("close boundary MySQL for unavailable shield")
	}
	boundaryExpectError(t, acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders", visitorToken, "", nil, http.StatusServiceUnavailable), "ORDERS_UNAVAILABLE")
}

type boundaryPhoneProvider struct{ phones map[string]string }

func (provider boundaryPhoneProvider) Exchange(_ context.Context, code, openID string) (string, error) {
	if code == "rejected" || strings.TrimSpace(code) == "" || provider.phones[openID] == "" {
		return "", wechat.ErrPhoneCodeRejected
	}
	return provider.phones[openID], nil
}

type boundaryPaymentProvider struct {
	delegate *localPaymentProvider
	creates  atomic.Uint64
}

func (provider *boundaryPaymentProvider) CreateJSAPI(ctx context.Context, request paymentorder.ProviderCreateRequest) (paymentorder.ProviderCreateResult, error) {
	provider.creates.Add(1)
	return provider.delegate.CreateJSAPI(ctx, request)
}

func (provider *boundaryPaymentProvider) QueryTransaction(ctx context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	return provider.delegate.QueryTransaction(ctx, outTradeNo)
}

func (provider *boundaryPaymentProvider) CloseTransaction(ctx context.Context, outTradeNo string) error {
	return provider.delegate.CloseTransaction(ctx, outTradeNo)
}

func (provider *boundaryPaymentProvider) ParsePaymentNotification(body []byte, headers wechatpay.SignatureHeaders) (paymentorder.VerifiedPayment, error) {
	return provider.delegate.ParsePaymentNotification(body, headers)
}

type boundaryCounts struct{ quotes, prepayments, orders int }

func boundarySeedBootstrap(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json,record_version) VALUES(1,'边界验收食堂','边界验收园区','北门取餐点','边界验收','open',JSON_ARRAY('少饭'),1)`, nil},
		{`INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,created_at,updated_at) VALUES(1,?,'边界验收主账号','OWNER',TRUE,1,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, []any{acceptanceOwnerPhone}},
		{`INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES(1,100,1,1,UTC_TIMESTAMP(6))`, nil},
	}
	for index, statement := range statements {
		if _, err := db.ExecContext(t.Context(), statement.query, statement.args...); err != nil {
			t.Fatalf("seed boundary bootstrap %d: %v", index, err)
		}
	}
}

func boundaryConfigure(t *testing.T, client *http.Client, origin, pcToken, key, status, today, tomorrow string) {
	t.Helper()
	acceptanceHTTP(t, client, http.MethodPut, origin+"/api/v1/admin/settings", pcToken, key, map[string]any{
		"store_status": status, "pickup_point": "北门取餐点", "notice": "边界验收", "pickup_step_min": 30,
		"meal_periods": []map[string]any{
			{"code": "lunch", "name": "午餐", "cutoff_time": "04:30", "pickup_from": "05:00", "pickup_to": "06:00"},
			{"code": "dinner", "name": "晚餐", "cutoff_time": "17:00", "pickup_from": "17:00", "pickup_to": "19:00"},
		},
		"service_dates": []map[string]any{{"date": today, "status": "open"}, {"date": tomorrow, "status": "open"}},
	}, http.StatusOK)
}

func boundaryCreateCategory(t *testing.T, client *http.Client, origin, pcToken string) string {
	t.Helper()
	view := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/categories", pcToken, "boundary-category-create", map[string]any{"name": "边界套餐"}, http.StatusCreated)
	return acceptanceString(t, acceptanceObject(t, view, "category"), "id")
}

func boundaryCreateProduct(t *testing.T, client *http.Client, origin, pcToken, categoryID, name string, price int, meal, key string) string {
	t.Helper()
	view := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/products", pcToken, key, map[string]any{
		"name": name, "price_cents": price, "category_id": categoryID, "meal_period": meal, "description": "边界验收", "images": []any{},
	}, http.StatusCreated)
	return acceptanceString(t, acceptanceObject(t, view, "product"), "id")
}

func boundaryUpdateProduct(t *testing.T, client *http.Client, origin, pcToken, categoryID, productID, name string, price int, meal, key string) {
	t.Helper()
	acceptanceHTTP(t, client, http.MethodPut, origin+"/api/v1/admin/products/"+productID, pcToken, key, map[string]any{
		"name": name, "price_cents": price, "category_id": categoryID, "meal_period": meal, "description": "边界验收", "images": []any{},
	}, http.StatusOK)
}

func boundaryProductStatus(t *testing.T, client *http.Client, origin, pcToken, productID, status, key string) {
	t.Helper()
	acceptanceHTTP(t, client, http.MethodPut, origin+"/api/v1/admin/products/"+productID+"/status", pcToken, key, map[string]any{"status": status}, http.StatusOK)
}

func boundaryProductSoldOut(t *testing.T, client *http.Client, origin, pcToken, productID, date string, soldOut bool, key string) {
	t.Helper()
	acceptanceHTTP(t, client, http.MethodPut, origin+"/api/v1/admin/products/"+productID+"/soldout", pcToken, key, map[string]any{"service_date": date, "sold_out": soldOut}, http.StatusOK)
}

func boundaryQuoteBody(date, pickup, productID string, quantity int64) map[string]any {
	return map[string]any{
		"contact_name": "边界用户", "pickup_date": date, "pickup_time": pickup, "order_note": "",
		"items": []map[string]any{{"product_id": productID, "quantity": quantity, "flavors": []string{}, "note": ""}},
	}
}

func boundaryCreateQuote(t *testing.T, client *http.Client, origin, token, key string, body map[string]any) string {
	t.Helper()
	return acceptanceString(t, boundaryCreateQuoteView(t, client, origin, token, key, body), "id")
}

func boundaryCreateQuoteView(t *testing.T, client *http.Client, origin, token, key string, body map[string]any) map[string]any {
	t.Helper()
	view := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/quotes", token, key, body, http.StatusCreated)
	return acceptanceObject(t, view, "quote")
}

func boundaryMaterializeOrder(t *testing.T, client *http.Client, origin, token, prefix, date, productID string) string {
	t.Helper()
	quoteID := boundaryCreateQuote(t, client, origin, token, prefix+"-quote", boundaryQuoteBody(date, "05:00", productID, 1))
	prepay := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/orders/prepay", token, prefix+"-prepay", map[string]any{"quote_id": quoteID}, http.StatusCreated)
	prepaymentID := acceptanceString(t, acceptanceObject(t, prepay, "prepayment"), "id")
	confirm := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/orders/confirm", token, prefix+"-confirm", map[string]any{"prepayment_id": prepaymentID}, http.StatusOK)
	if acceptanceString(t, confirm, "state") != "ORDER_CREATED" {
		t.Fatal("boundary payment confirmation did not materialize an order")
	}
	return acceptanceString(t, confirm, "order_id")
}

func boundaryTransactionCounts(t *testing.T, db *sql.DB) boundaryCounts {
	t.Helper()
	var result boundaryCounts
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM quotes),(SELECT COUNT(*) FROM prepayments),(SELECT COUNT(*) FROM orders)`).Scan(&result.quotes, &result.prepayments, &result.orders); err != nil {
		t.Fatal("read boundary transaction counts")
	}
	return result
}

func boundaryAssertTransactionCounts(t *testing.T, db *sql.DB, want boundaryCounts, scenario string) {
	t.Helper()
	if got := boundaryTransactionCounts(t, db); got != want {
		t.Fatalf("%s transaction counts = %#v, want %#v", scenario, got, want)
	}
}

func boundaryExpectError(t *testing.T, response map[string]any, code string) {
	t.Helper()
	if got := acceptanceString(t, acceptanceObject(t, response, "error"), "code"); got != code {
		t.Fatalf("boundary error code = %q, want %q", got, code)
	}
}

func boundaryObjects(t *testing.T, object map[string]any, key string) []map[string]any {
	t.Helper()
	raw, ok := object[key].([]any)
	if !ok {
		t.Fatalf("boundary response field %s is not an array", key)
	}
	result := make([]map[string]any, 0, len(raw))
	for index, value := range raw {
		item, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("boundary response field %s[%d] is not an object", key, index)
		}
		result = append(result, item)
	}
	return result
}

func boundaryMenuProducts(t *testing.T, menuView map[string]any) []map[string]any {
	t.Helper()
	products := make([]map[string]any, 0)
	for _, category := range boundaryObjects(t, menuView, "categories") {
		products = append(products, boundaryObjects(t, category, "products")...)
	}
	return products
}

func boundaryHasProduct(products []map[string]any, productID string) bool {
	for _, product := range products {
		if value, _ := product["id"].(string); value == productID {
			return true
		}
	}
	return false
}

func boundaryProduct(t *testing.T, products []map[string]any, productID string) map[string]any {
	t.Helper()
	for _, product := range products {
		if value, _ := product["id"].(string); value == productID {
			return product
		}
	}
	t.Fatalf("boundary product %s is absent", productID)
	return nil
}

func boundaryAssertPricingIdentity(t *testing.T, pricing map[string]any, kind string, rate int64, scenario string) {
	t.Helper()
	if acceptanceString(t, pricing, "kind") != kind || acceptanceInt(t, pricing, "rate_percent") != rate {
		t.Fatalf("%s pricing identity = %#v", scenario, pricing)
	}
}

func boundaryAssertQuotePricing(t *testing.T, quoteView map[string]any, kind string, rate, payable int64, scenario string) {
	t.Helper()
	if acceptanceString(t, acceptanceObject(t, quoteView, "identity"), "kind") != kind || acceptanceInt(t, acceptanceObject(t, quoteView, "discount"), "rate_percent") != rate || acceptanceInt(t, quoteView, "payable_cents") != payable {
		t.Fatalf("%s quote pricing = %#v", scenario, quoteView)
	}
}

func boundaryAssertOrderNoToken(t *testing.T, view map[string]any, state, scenario string) {
	t.Helper()
	order := acceptanceObject(t, view, "order")
	if acceptanceString(t, order, "state") != state {
		t.Fatalf("%s order state = %#v", scenario, order)
	}
	if _, exists := order["redemption_token"]; exists {
		t.Fatalf("%s exposed a redemption token before READY", scenario)
	}
}

func boundaryAt(t *testing.T, date, minute string, location *time.Location) time.Time {
	t.Helper()
	value, err := time.ParseInLocation("2006-01-02 15:04", date+" "+minute, location)
	if err != nil {
		t.Fatalf("parse boundary clock %s %s: %v", date, minute, err)
	}
	return value
}
