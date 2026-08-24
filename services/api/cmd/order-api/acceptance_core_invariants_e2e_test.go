package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/merchantsoldout"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/orderadvance"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/migrations"
	"github.com/gin-gonic/gin"
)

const (
	coreVisitorPhone    = "+8613800000005"
	coreExtraStaffPhone = "+8613800000004"
	coreCoverObjectKey  = "products/core-acceptance-cover.png"
)

// TestAcceptanceCoreContractRejectsAlternateFacts is the L1 half of INV-01,
// INV-16 and AC-19-LOCAL. Alternate fulfillment, client payment truth and
// out-of-scope persistence vocabulary are rejected before an application call.
func TestAcceptanceCoreContractRejectsAlternateFacts(t *testing.T) {
	application := &coreQuoteSpy{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	quote.NewHandler(coreQuoteAuthenticator{}, application).RegisterRoutes(router)

	for name, body := range map[string]string{
		"delivery":       `{"contact_name":"验收用户","pickup_date":"2026-08-23","pickup_time":"11:30","fulfillment_method":"delivery","items":[{"product_id":"1","quantity":1}]}`,
		"instant pickup": `{"contact_name":"验收用户","pickup_date":"2026-08-23","pickup_time":"11:30","pickup_mode":"instant","items":[{"product_id":"1","quantity":1}]}`,
		"second point":   `{"contact_name":"验收用户","pickup_date":"2026-08-23","pickup_time":"11:30","pickup_point":"第二取餐点","items":[{"product_id":"1","quantity":1}]}`,
		"client price":   `{"contact_name":"验收用户","pickup_date":"2026-08-23","pickup_time":"11:30","items":[{"product_id":"1","quantity":1,"price_cents":1}]}`,
		"client success": `{"contact_name":"验收用户","pickup_date":"2026-08-23","pickup_time":"11:30","payment_success":true,"items":[{"product_id":"1","quantity":1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer local")
			request.Header.Set("Idempotency-Key", "core-l1-"+strings.ReplaceAll(name, " ", "-"))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || coreErrorCodeFromBytes(t, response.Body.Bytes()) != "INVALID_REQUEST" {
				t.Fatalf("alternate fact status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	if application.createCalls != 0 {
		t.Fatalf("alternate client facts reached Quote application %d times", application.createCalls)
	}

	set, err := migrate.Load(migrations.FS)
	if err != nil || len(set) != 44 || set[43].Version != 44 {
		t.Fatalf("load frozen v1-v44 ledger: count=%d err=%v", len(set), err)
	}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal("read migration ledger")
	}
	forbidden := []string{"delivery", "instant", "inventory", "coupon", "member", "external_gate", "client_success", "mock_payment"}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		raw, readErr := fs.ReadFile(migrations.FS, entry.Name())
		if readErr != nil {
			t.Fatal("read migration SQL")
		}
		lower := strings.ToLower(string(raw))
		for _, word := range forbidden {
			if strings.Contains(lower, word) {
				t.Fatalf("migration %s contains out-of-scope fact %q", entry.Name(), word)
			}
		}
	}
}

// TestAcceptanceCoreIdentityPricingRBACAndGovernanceAreServerFacts closes the
// L2 requirements of PAGE-U05/PAGE-U09, AC-05/AC-16, INV-01/INV-11/INV-16
// and AC-19-LOCAL. All mutable business fixtures cross root HTTP; direct SQL is
// limited to exact schema/failure assertions.
func TestAcceptanceCoreIdentityPricingRBACAndGovernanceAreServerFacts(t *testing.T) {
	h := newCoreAcceptanceHarness(t, true)

	store := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/storefront/settings", "", "", nil, http.StatusOK)
	pickups := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/menu/pickup-options", "", "", nil, http.StatusOK)
	menuBeforeIdentity := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/menu?date=2026-08-23&time=11:30", "", "", nil, http.StatusOK)
	coreAssertNoAlternateFulfillment(t, store, pickups, menuBeforeIdentity)
	if users, sessions := coreUserSessionCounts(t, h.db); users != 0 || sessions != 0 {
		t.Fatalf("anonymous browse created identity facts users=%d sessions=%d", users, sessions)
	}
	if _, err := h.db.ExecContext(t.Context(), `INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json,record_version) VALUES(2,'第二门店','地址','第二取餐点','','open',JSON_ARRAY(),1)`); err == nil {
		t.Fatal("singleton storefront accepted a second store/pickup point")
	}

	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", "", "core-no-session", coreQuoteBody("1", "anonymous"), http.StatusUnauthorized), "UNAUTHENTICATED")
	visitor := acceptanceMiniSession(t, h.client, h.origin, "core-visitor-login")
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", visitor, "core-unbound", coreQuoteBody("1", "unbound"), http.StatusConflict), "PRIMARY_PHONE_REQUIRED")
	if coreCount(t, h.db, `SELECT COUNT(*) FROM quotes`) != 0 {
		t.Fatal("unbound user created a quote")
	}

	owner := acceptanceMiniSession(t, h.client, h.origin, "core-owner-login")
	subaccount := acceptanceMiniSession(t, h.client, h.origin, "core-sub-login")
	acceptanceBindPhone(t, h.client, h.origin, owner, "core-owner-phone")
	acceptanceBindPhone(t, h.client, h.origin, subaccount, "core-sub-phone")
	acceptanceBindPhone(t, h.client, h.origin, visitor, "core-visitor-phone")
	acceptanceMerchantLogin(t, h.client, h.origin, owner, "OWNER")
	acceptanceMerchantLogin(t, h.client, h.origin, subaccount, "SUBACCOUNT")
	pcToken := pcClosureOwnerSession(t, h.client, h.origin, owner)

	acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/admin/staff-whitelist", pcToken, "core-extra-staff", map[string]any{
		"name": "附加员工", "phone": "13800000004",
	}, http.StatusCreated)
	productView := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/admin/products", pcToken, "core-cover-product", map[string]any{
		"name": "封面工作餐", "price_cents": 1250, "category_id": "1", "meal_period": "lunch", "description": "验收封面快照",
		"images": []map[string]any{{"object_key": coreCoverObjectKey}},
	}, http.StatusCreated)
	productID := acceptanceString(t, acceptanceObject(t, productView, "product"), "id")

	mismatch := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/me/extra-phone", visitor, "core-extra-mismatch", map[string]any{
		"phone": "13800000004", "name": "错误姓名",
	}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, mismatch, "pricing_identity"), "kind") != "VISITOR" {
		t.Fatal("phone-only extra match granted staff identity")
	}
	matched := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/me/extra-phone", visitor, "core-extra-match", map[string]any{
		"phone": "13800000004", "name": "附加员工",
	}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, matched, "pricing_identity"), "kind") != "STAFF" || acceptanceInt(t, acceptanceObject(t, matched, "pricing_identity"), "rate_percent") != 80 {
		t.Fatal("extra phone plus exact name did not derive the server staff rate")
	}
	identityView := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/me/identity", visitor, "", nil, http.StatusOK)
	identityProjection := acceptanceObject(t, identityView, "identity")
	if acceptanceString(t, acceptanceObject(t, identityProjection, "pricing_identity"), "kind") != "STAFF" || acceptanceBool(t, acceptanceObject(t, identityProjection, "merchant"), "bound") {
		t.Fatal("PAGE-U09 identity projection mixed staff and merchant identities")
	}
	ordersView := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/orders", visitor, "", nil, http.StatusOK)
	if raw, ok := ordersView["orders"].([]any); !ok || len(raw) != 0 {
		t.Fatalf("new profile exposed non-owned orders: %#v", ordersView["orders"])
	}
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/me/merchant-login", visitor, "", map[string]any{"code": "core-visitor-merchant"}, http.StatusForbidden), "MERCHANT_ACCOUNT_NOT_AVAILABLE")

	menuView := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/menu?date=2026-08-23&time=11:30", visitor, "", nil, http.StatusOK)
	menuProduct := coreFindProduct(t, menuView, productID)
	detailView := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/catalog/products/"+productID+"?date=2026-08-23&time=11:30", visitor, "", nil, http.StatusOK)
	detailProduct := acceptanceObject(t, detailView, "product")
	if acceptanceInt(t, menuProduct, "original_unit_price_cents") != 1250 || acceptanceInt(t, menuProduct, "staff_unit_price_cents") != 1000 ||
		acceptanceInt(t, detailProduct, "staff_unit_price_cents") != 1000 {
		t.Fatal("menu/detail did not share the per-item half-up staff price")
	}
	quoteView := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", visitor, "core-priced-quote", coreQuoteBody(productID, "priced"), http.StatusCreated)
	quoteProjection := acceptanceObject(t, quoteView, "quote")
	quoteID := acceptanceString(t, quoteProjection, "id")
	quoteItem := acceptanceArray(t, quoteProjection, "items")[0]
	if acceptanceInt(t, quoteItem, "discounted_unit_price_cents") != 1000 || acceptanceInt(t, quoteProjection, "payable_cents") != 2000 ||
		acceptanceString(t, quoteItem, "image_object_key") != coreCoverObjectKey || acceptanceString(t, quoteProjection, "order_note") != "整单备注-priced" {
		t.Fatal("Quote did not freeze menu/detail price, cover key, quantity and notes")
	}
	coreAssertQuoteFacts(t, h.db, quoteID)

	quotesBeforeClientMoney := coreCount(t, h.db, `SELECT COUNT(*) FROM quotes`)
	clientMoney := coreQuoteBody(productID, "client-money")
	clientMoney["items"] = []map[string]any{{"product_id": productID, "quantity": 2, "price_cents": 1}}
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", visitor, "core-client-money", clientMoney, http.StatusBadRequest), "INVALID_REQUEST")
	if coreCount(t, h.db, `SELECT COUNT(*) FROM quotes`) != quotesBeforeClientMoney {
		t.Fatal("client-owned price changed the server Quote facts")
	}

	cutoffQuote := acceptanceString(t, acceptanceObject(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", visitor, "core-cutoff-quote", coreQuoteBody(productID, "cutoff"), http.StatusCreated), "quote"), "id")
	h.clock = time.Date(2026, 8, 23, 11, 30, 0, 0, h.shanghai)
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/prepay", visitor, "core-cutoff-prepay", map[string]any{"quote_id": cutoffQuote}, http.StatusConflict), "QUOTE_UNAVAILABLE")
	h.clock = time.Date(2026, 8, 23, 8, 0, 0, 0, h.shanghai)
	driftQuote := acceptanceString(t, acceptanceObject(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", visitor, "core-drift-quote", coreQuoteBody(productID, "drift"), http.StatusCreated), "quote"), "id")
	acceptanceHTTP(t, h.client, http.MethodPut, h.origin+"/api/v1/admin/products/"+productID, pcToken, "core-product-drift", map[string]any{
		"name": "封面工作餐", "price_cents": 1300, "category_id": "1", "meal_period": "lunch", "description": "验收封面快照",
		"images": []map[string]any{{"object_key": coreCoverObjectKey}},
	}, http.StatusOK)
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/prepay", visitor, "core-drift-prepay", map[string]any{"quote_id": driftQuote}, http.StatusConflict), "QUOTE_UNAVAILABLE")
	if coreCount(t, h.db, `SELECT COUNT(*) FROM prepayments`) != 0 {
		t.Fatal("cutoff or drift created a prepayment/provider call")
	}

	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPut, h.origin+"/api/v1/merchant/products/"+productID+"/soldout", visitor, "core-user-soldout", map[string]any{"service_date": "2026-08-23", "sold_out": true}, http.StatusForbidden), "FORBIDDEN")
	acceptanceHTTP(t, h.client, http.MethodPut, h.origin+"/api/v1/merchant/products/"+productID+"/soldout", subaccount, "core-sub-soldout", map[string]any{"service_date": "2026-08-23", "sold_out": true}, http.StatusOK)
	acceptanceHTTP(t, h.client, http.MethodPut, h.origin+"/api/v1/merchant/products/"+productID+"/soldout", owner, "core-owner-soldout-reset", map[string]any{"service_date": "2026-08-23", "sold_out": false}, http.StatusOK)

	subPC := acceptanceQRLogin(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/admin/auth/qrcode", "", "", map[string]any{}, http.StatusCreated))
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/me/admin-login/approve", subaccount, "", map[string]any{
		"login_id": subPC.loginID, "approval_secret": subPC.approvalSecret, "code": "core-sub-phone",
	}, http.StatusForbidden), "FORBIDDEN")
	for name, request := range map[string]struct {
		method string
		body   map[string]any
	}{
		"disable":   {http.MethodPut, map[string]any{"enabled": false}},
		"downgrade": {http.MethodPut, map[string]any{"name": "验收主账号", "phone": "13800000003", "role": "SUBACCOUNT"}},
		"delete":    {http.MethodDelete, nil},
	} {
		coreExpectError(t, acceptanceHTTP(t, h.client, request.method, h.origin+"/api/v1/admin/merchant-accounts/1", pcToken, "core-last-owner-"+name, request.body, http.StatusConflict), "LAST_OWNER")
	}
	var role string
	var enabled bool
	if err := h.db.QueryRowContext(t.Context(), `SELECT role,enabled FROM merchant_accounts WHERE id=1`).Scan(&role, &enabled); err != nil || role != "OWNER" || !enabled {
		t.Fatalf("last OWNER protection changed durable account role=%s enabled=%v err=%v", role, enabled, err)
	}

	var cosmeticColumns int
	if err := h.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='miniprogram_users' AND column_name IN ('avatar','avatar_url','nickname','display_name')`).Scan(&cosmeticColumns); err != nil || cosmeticColumns != 0 {
		t.Fatalf("cosmetic profile became identity persistence columns=%d err=%v", cosmeticColumns, err)
	}
	coreAssertGovernanceFacts(t, h.db, []string{coreVisitorPhone, coreExtraStaffPhone, acceptanceOwnerPhone, acceptanceMerchantPhone, visitor, owner, subaccount, pcToken})
}

// TestAcceptanceCorePaymentProductionRefundSubscriptionAndSequenceAreClosed
// closes L2 for AC-09, BE-10, BE-17 and INV-14/INV-15. It uses one fresh
// schema for concurrent/cross-date numbers, InitialState+Advance, refund token
// shielding and notification failure isolation.
func TestAcceptanceCorePaymentProductionRefundSubscriptionAndSequenceAreClosed(t *testing.T) {
	h := newCoreAcceptanceHarness(t, true)
	customer := acceptanceMiniSession(t, h.client, h.origin, "core-customer-login")
	owner := acceptanceMiniSession(t, h.client, h.origin, "core-owner-login")
	acceptanceBindPhone(t, h.client, h.origin, customer, "core-customer-phone")
	acceptanceBindPhone(t, h.client, h.origin, owner, "core-owner-phone")
	acceptanceMerchantLogin(t, h.client, h.origin, owner, "OWNER")
	pcToken := pcClosureOwnerSession(t, h.client, h.origin, owner)

	todayA := coreCreatePrepayment(t, h, customer, "today-a", "2026-08-23", "11:30")
	todayB := coreCreatePrepayment(t, h, customer, "today-b", "2026-08-23", "11:30")
	tomorrow := coreCreatePrepayment(t, h, customer, "tomorrow", "2026-08-24", "11:30")

	type concurrentResult struct {
		status int
		body   map[string]any
		err    error
	}
	start := make(chan struct{})
	results := make(chan concurrentResult, 2)
	var wait sync.WaitGroup
	for _, prepared := range []corePreparedPayment{todayA, todayB} {
		prepared := prepared
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			status, body, err := coreRawJSON(h.client, http.MethodPost, h.origin+"/api/v1/orders/confirm", customer, "core-confirm-"+prepared.key, map[string]any{"prepayment_id": prepared.prepaymentID})
			results <- concurrentResult{status: status, body: body, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	todayOrderIDs := make([]string, 0, 2)
	for result := range results {
		if result.err != nil || result.status != http.StatusOK || acceptanceString(t, result.body, "state") != "ORDER_CREATED" {
			t.Fatalf("concurrent payment materialization status=%d body=%#v err=%v", result.status, result.body, result.err)
		}
		todayOrderIDs = append(todayOrderIDs, acceptanceString(t, result.body, "order_id"))
	}
	if len(todayOrderIDs) != 2 || todayOrderIDs[0] == todayOrderIDs[1] {
		t.Fatalf("concurrent prepayments did not create two unique orders: %#v", todayOrderIDs)
	}
	tomorrowConfirm := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/confirm", customer, "core-confirm-tomorrow", map[string]any{"prepayment_id": tomorrow.prepaymentID}, http.StatusOK)
	tomorrowOrderID := acceptanceString(t, tomorrowConfirm, "order_id")
	replay := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/confirm", customer, "core-confirm-today-a-replay", map[string]any{"prepayment_id": todayA.prepaymentID}, http.StatusOK)
	if replayID := acceptanceString(t, replay, "order_id"); replayID != coreOrderForPrepayment(t, h.db, todayA.prepaymentID) {
		t.Fatalf("same prepayment replay changed order %s", replayID)
	}
	coreAssertSequenceFacts(t, h.db, 2, 1)

	production := orderadvance.New(h.db)
	h.clock = time.Date(2026, 8, 23, 10, 59, 59, 0, h.shanghai)
	if run, err := production.RunProductionDue(t.Context(), h.clock, 10); err != nil || run.Advanced != 0 {
		t.Fatalf("production advanced before 30m boundary: %#v/%v", run, err)
	}
	h.clock = time.Date(2026, 8, 23, 11, 0, 0, 0, h.shanghai)
	if run, err := production.RunProductionDue(t.Context(), h.clock, 10); err != nil || run.Advanced != 2 {
		t.Fatalf("production did not compensate both due orders: %#v/%v", run, err)
	}
	if run, err := production.RunProductionDue(t.Context(), h.clock.Add(time.Minute), 10); err != nil || run.Advanced != 0 {
		t.Fatalf("production replay was not monotonic: %#v/%v", run, err)
	}

	h.clock = time.Date(2026, 8, 23, 11, 1, 0, 0, h.shanghai)
	near := coreCreatePrepayment(t, h, customer, "near", "2026-08-23", "11:30")
	nearConfirm := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/confirm", customer, "core-confirm-near", map[string]any{"prepayment_id": near.prepaymentID}, http.StatusOK)
	nearOrderID := acceptanceString(t, nearConfirm, "order_id")
	nearView := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/orders/"+nearOrderID, customer, "", nil, http.StatusOK)
	acceptanceAssertOrderState(t, nearView, "PREPARING", false)
	if actions, ok := acceptanceObject(t, nearView, "order")["available_actions"].([]any); !ok || len(actions) != 0 {
		t.Fatalf("near-time new order exposed self-cancel actions: %#v", actions)
	}

	targetOrderID := coreOrderForPrepayment(t, h.db, todayA.prepaymentID)
	otherOrderID := coreOrderForPrepayment(t, h.db, todayB.prepaymentID)
	acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/"+targetOrderID+"/subscriptions", customer, "core-ready-accepted", map[string]any{"kind": "READY", "decision": "ACCEPTED"}, http.StatusOK)
	acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/"+targetOrderID+"/subscriptions", customer, "core-refund-rejected", map[string]any{"kind": "REFUND_RESULT", "decision": "REJECTED"}, http.StatusOK)
	acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/"+otherOrderID+"/subscriptions", customer, "core-refund-accepted", map[string]any{"kind": "REFUND_RESULT", "decision": "ACCEPTED"}, http.StatusOK)
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/"+targetOrderID+"/subscriptions", customer, "core-other-subscription", map[string]any{"kind": "OTHER", "decision": "ACCEPTED"}, http.StatusBadRequest), "INVALID_REQUEST")

	h.notificationProvider.Queue(subscription.SendResult{}, &subscription.SendError{Code: "RATE_LIMITED", Permanent: false})
	ready := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/merchant/orders/"+targetOrderID+"/ready", owner, "core-ready-target", map[string]any{}, http.StatusOK)
	acceptanceAssertOrderState(t, ready, "READY_FOR_PICKUP", false)
	userReady := acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/orders/"+targetOrderID, customer, "", nil, http.StatusOK)
	token := acceptanceString(t, acceptanceObject(t, userReady, "order"), "redemption_token")
	notificationAt := time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond)
	if run, err := h.notifications.RunDue(t.Context(), notificationAt, 10); err != nil || run.Claimed != 1 || run.TemporaryFailed != 1 || run.Sent != 0 {
		t.Fatalf("notification failure classification = %#v/%v", run, err)
	}
	acceptanceAssertOrderState(t, acceptanceHTTP(t, h.client, http.MethodGet, h.origin+"/api/v1/orders/"+targetOrderID, customer, "", nil, http.StatusOK), "READY_FOR_PICKUP", true)

	refundView := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/admin/orders/"+targetOrderID+"/refund", pcToken, "core-refund-ready", map[string]any{"reason": "已备好退款核销保护"}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, refundView, "order"), "state") != "退款中" {
		t.Fatal("ready-order refund did not stop at REFUNDING")
	}
	h.clock = time.Now().UTC().Add(2 * time.Minute).Truncate(time.Microsecond)
	if run, err := h.refunds.RunDue(t.Context(), h.clock, 10); err != nil || run.Applied != 1 {
		t.Fatalf("refund worker did not apply provider finality: %#v/%v", run, err)
	}
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/verify/scan", owner, "core-refunded-scan", map[string]any{"token": token}, http.StatusConflict), "TRANSITION_NOT_ALLOWED")
	var state string
	var cipherBytes, hashBytes int
	if err := h.db.QueryRowContext(t.Context(), `SELECT state,COALESCE(OCTET_LENGTH(redemption_token_ciphertext),0),COALESCE(OCTET_LENGTH(redemption_token_hash),0) FROM orders WHERE id=?`, targetOrderID).Scan(&state, &cipherBytes, &hashBytes); err != nil || state != "REFUNDED" || cipherBytes != 0 || hashBytes != 32 {
		t.Fatalf("refunded token shield state=%s cipher=%d hash=%d err=%v", state, cipherBytes, hashBytes, err)
	}
	coreAssertSubscriptionFacts(t, h.db, targetOrderID)

	if _, err := h.db.ExecContext(t.Context(), `UPDATE pickup_sequences SET last_number=9999 WHERE service_date='2026-08-23'`); err != nil {
		t.Fatal("inject exact 9999 sequence boundary")
	}
	h.clock = time.Date(2026, 8, 23, 11, 1, 0, 0, h.shanghai)
	overflow := coreCreatePrepayment(t, h, customer, "overflow", "2026-08-23", "11:30")
	ordersBefore := coreCount(t, h.db, `SELECT COUNT(*) FROM orders`)
	coreExpectError(t, acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/confirm", customer, "core-confirm-overflow", map[string]any{"prepayment_id": overflow.prepaymentID}, http.StatusServiceUnavailable), "UNAVAILABLE")
	if coreCount(t, h.db, `SELECT COUNT(*) FROM orders`) != ordersBefore || coreCount(t, h.db, `SELECT last_number FROM pickup_sequences WHERE service_date='2026-08-23'`) != 9999 {
		t.Fatal("pickup number 10000 created an order or burned/overflowed the sequence")
	}
	if tomorrowState := coreOrderState(t, h.db, tomorrowOrderID); tomorrowState != "RESERVED" {
		t.Fatalf("today operations changed tomorrow same-number order to %s", tomorrowState)
	}
	coreAssertGovernanceFacts(t, h.db, []string{acceptanceCustomerPhone, acceptanceOwnerPhone, customer, owner, pcToken})
}

type coreAcceptanceHarness struct {
	db                   *sql.DB
	client               *http.Client
	origin               string
	clock                time.Time
	shanghai             *time.Location
	payment              *localPaymentProvider
	refunds              *refund.Service
	notifications        *subscription.Service
	notificationProvider *subscription.FakeProvider
}

func newCoreAcceptanceHarness(t *testing.T, autoPay bool) *coreAcceptanceHarness {
	t.Helper()
	h := &coreAcceptanceHarness{db: acceptanceFreshMySQL(t), shanghai: time.FixedZone("Asia/Shanghai", 8*60*60)}
	h.clock = time.Date(2026, 8, 23, 8, 0, 0, 0, h.shanghai)
	acceptanceSeedSharedFacts(t, h.db)
	if _, err := h.db.ExecContext(t.Context(), `UPDATE service_dates SET is_open=TRUE WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("open explicit tomorrow service date")
	}
	now := func() time.Time { return h.clock }
	identityRepository := identity.NewRepository(h.db)
	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{
		"core-customer-login": "core-customer-openid", "core-visitor-login": "core-visitor-openid",
		"core-owner-login": "core-owner-openid", "core-sub-login": "core-sub-openid",
	}}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"core-customer-openid": acceptanceCustomerPhone, "core-visitor-openid": coreVisitorPhone,
		"core-owner-openid": acceptanceOwnerPhone, "core-sub-openid": acceptanceMerchantPhone,
	}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phones := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	pricing := staffdiscount.NewMySQLPricing(h.db)
	urls := corePublicURLs{}
	quoteApplication := quote.NewProvider(h.db, audit.NewQuoteReceiptStore(h.db), now)
	merchantRepository := merchantidentity.NewRepository(h.db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(h.db, merchantService)
	h.payment = newLocalPaymentProvider(now)
	h.payment.autoPay = autoPay
	paymentApplication := paymentorder.NewMySQLApplication(h.db, quoteApplication, h.payment, paymentorder.Config{
		AppID: "wx-core-acceptance", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	h.notificationProvider = subscription.NewFakeProvider()
	h.notifications = subscription.New(h.db, h.notificationProvider)
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatal("compose core redemption cipher")
	}
	orders := orderquery.NewRepository(h.db, merchantService, cipher, now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(h.db, merchantRepository, cipher, h.notifications)
	refundProvider := newLocalRefundProvider("order-local-mch", now)
	h.refunds = refund.New(h.db, refundProvider, "http://127.0.0.1/api/v1/refunds/wechat/notify").
		WithNotificationEnqueuer(newRefundSubscriptionAdapter(h.notifications))
	adminReader := adminreport.NewMySQLApplication(h.db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, h.refunds, adminReader)
	adminReports := adminreport.NewMySQLApplication(h.db, adminCommands)
	catalogAdmin := catalog.NewMySQLAdminApplication(h.db, urls)
	staffAdmin := staffdiscount.NewMySQLApplication(h.db)
	soldOut := merchantsoldout.New(h.db, merchantRepository, now)

	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		storefront.NewHandler(storefront.NewRepository(h.db)),
		catalog.NewHandler(catalog.NewRepository(h.db), catalog.WithAuthenticator(miniAuth), catalog.WithPricing(pricing), catalog.WithPublicURLs(urls), catalog.WithClock(now)),
		menu.NewHandler(menu.NewRepository(h.db), now, menu.WithAuthenticator(miniAuth), menu.WithPricing(pricing), menu.WithPublicURLs(urls)),
		identity.NewHandler(sessions), identity.NewPhoneHandler(sessions, phones), merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication), newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, h.payment)),
		httpapi.NewSubscriptionHandler(sessions, h.notifications, 1), orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders, fulfillment.WithSoldOut(soldOut)),
		newRefundRoutes(refund.NewHandler(sessions, h.refunds, orders, refundProvider)),
		newAdminRoutes(sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin), []adminGroupRegistrar{
			catalog.NewAdminHandler(catalogAdmin), staffdiscount.NewHandler(staffAdmin), adminreport.NewHandler(adminReports),
		}, nil, nil),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	h.client, h.origin = server.Client(), server.URL
	return h
}

type corePublicURLs struct{}

func (corePublicURLs) PublicURL(_ context.Context, key string) (string, error) {
	if key == "" || strings.Contains(key, "..") {
		return "", errors.New("invalid object key")
	}
	return "https://objects.invalid/" + key, nil
}

type coreQuoteAuthenticator struct{}

func (coreQuoteAuthenticator) Authenticate(context.Context, string) (uint64, error) { return 1, nil }

type coreQuoteSpy struct{ createCalls int }

func (spy *coreQuoteSpy) Create(context.Context, quote.WriteMeta, quote.CreateInput) (quote.CreateResult, error) {
	spy.createCalls++
	return quote.CreateResult{}, errors.New("unexpected application call")
}
func (*coreQuoteSpy) Read(context.Context, uint64, uint64) (quote.Quote, error) {
	return quote.Quote{}, quote.ErrNotFound
}

type corePreparedPayment struct{ key, quoteID, prepaymentID string }

func coreCreatePrepayment(t *testing.T, h *coreAcceptanceHarness, token, key, date, pickupTime string) corePreparedPayment {
	t.Helper()
	quoteView := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/quotes", token, "core-quote-"+key, map[string]any{
		"contact_name": "核心验收用户", "pickup_date": date, "pickup_time": pickupTime, "order_note": key,
		"items": []map[string]any{{"product_id": "1", "quantity": 1, "flavors": []string{"少饭"}, "note": key}},
	}, http.StatusCreated)
	quoteID := acceptanceString(t, acceptanceObject(t, quoteView, "quote"), "id")
	prepayView := acceptanceHTTP(t, h.client, http.MethodPost, h.origin+"/api/v1/orders/prepay", token, "core-prepay-"+key, map[string]any{"quote_id": quoteID}, http.StatusCreated)
	return corePreparedPayment{key: key, quoteID: quoteID, prepaymentID: acceptanceString(t, acceptanceObject(t, prepayView, "prepayment"), "id")}
}

func coreQuoteBody(productID, suffix string) map[string]any {
	return map[string]any{
		"contact_name": "核心验收用户", "pickup_date": "2026-08-23", "pickup_time": "11:30", "order_note": "整单备注-" + suffix,
		"items": []map[string]any{{"product_id": productID, "quantity": 2, "flavors": []string{"少饭"}, "note": "菜品备注-" + suffix}},
	}
}

func coreFindProduct(t *testing.T, response map[string]any, productID string) map[string]any {
	t.Helper()
	for _, category := range acceptanceArray(t, response, "categories") {
		for _, product := range acceptanceArray(t, category, "products") {
			if acceptanceString(t, product, "id") == productID {
				return product
			}
		}
	}
	t.Fatalf("product %s absent from menu", productID)
	return nil
}

func coreAssertQuoteFacts(t *testing.T, db *sql.DB, quoteID string) {
	t.Helper()
	var phone, image, flavors, note, orderNote string
	var original, discounted, quantity, payable, digestBytes int
	err := db.QueryRowContext(t.Context(), `SELECT CONVERT(q.contact_phone_snapshot USING ascii),CONVERT(qi.image_object_key_snapshot USING utf8mb4),CAST(qi.flavors_json AS CHAR),qi.line_note,q.order_note,qi.original_unit_price_cents,qi.discounted_unit_price_cents,qi.quantity,q.payable_cents,OCTET_LENGTH(q.snapshot_digest) FROM quotes q JOIN quote_items qi ON qi.quote_id=q.id WHERE q.id=?`, quoteID).
		Scan(&phone, &image, &flavors, &note, &orderNote, &original, &discounted, &quantity, &payable, &digestBytes)
	if err != nil || phone != coreVisitorPhone || image != coreCoverObjectKey || flavors != `["少饭"]` || note != "菜品备注-priced" || orderNote != "整单备注-priced" || original != 1250 || discounted != 1000 || quantity != 2 || payable != 2000 || digestBytes != 32 {
		t.Fatalf("immutable Quote facts phone=%s image=%s flavors=%s note=%s order_note=%s money=%d/%d/%d/%d digest=%d err=%v", phone, image, flavors, note, orderNote, original, discounted, quantity, payable, digestBytes, err)
	}
}

func coreAssertNoAlternateFulfillment(t *testing.T, values ...map[string]any) {
	t.Helper()
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal("encode public fulfillment response")
		}
		lower := strings.ToLower(string(raw))
		for _, forbidden := range []string{"delivery", "instant", "pickup_points", "storefronts"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("public contract exposed alternate fulfillment %q: %s", forbidden, raw)
			}
		}
	}
}

func coreUserSessionCounts(t *testing.T, db *sql.DB) (int, int) {
	t.Helper()
	var users, sessions int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM miniprogram_users),(SELECT COUNT(*) FROM miniprogram_sessions)`).Scan(&users, &sessions); err != nil {
		t.Fatal("count user/session facts")
	}
	return users, sessions
}

func coreAssertGovernanceFacts(t *testing.T, db *sql.DB, sensitive []string) {
	t.Helper()
	var gateTables int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND (table_name LIKE '%external_gate%' OR table_name LIKE '%mock%' OR table_name LIKE '%client_success%')`).Scan(&gateTables); err != nil || gateTables != 0 {
		t.Fatalf("local schema persisted fake external completion facts=%d err=%v", gateTables, err)
	}
	for _, value := range sensitive {
		if value == "" {
			continue
		}
		var leaked int
		if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM action_audits WHERE CONCAT_WS('',COALESCE(CONVERT(before_state_json USING utf8mb4),''),COALESCE(CONVERT(after_state_json USING utf8mb4),''),COALESCE(CONVERT(response_json USING utf8mb4),''),COALESCE(reason_code,'')) LIKE ?`, "%"+value+"%").Scan(&leaked); err != nil || leaked != 0 {
			t.Fatalf("sanitized audit leaked sensitive runtime value count=%d err=%v", leaked, err)
		}
	}
}

func coreAssertSequenceFacts(t *testing.T, db *sql.DB, today, tomorrow int) {
	t.Helper()
	var todayLast, tomorrowLast, uniqueNumbers int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT last_number FROM pickup_sequences WHERE service_date='2026-08-23'),(SELECT last_number FROM pickup_sequences WHERE service_date='2026-08-24'),(SELECT COUNT(DISTINCT pickup_number) FROM orders WHERE pickup_date='2026-08-23')`).Scan(&todayLast, &tomorrowLast, &uniqueNumbers); err != nil || todayLast != today || tomorrowLast != tomorrow || uniqueNumbers != today {
		t.Fatalf("date sequence facts today=%d tomorrow=%d unique_today=%d err=%v", todayLast, tomorrowLast, uniqueNumbers, err)
	}
}

func coreAssertSubscriptionFacts(t *testing.T, db *sql.DB, rejectedOrderID string) {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `SELECT DISTINCT kind FROM notification_consents ORDER BY kind`)
	if err != nil {
		t.Fatal("read subscription kind facts")
	}
	defer rows.Close()
	var kinds []string
	for rows.Next() {
		var kind string
		if rows.Scan(&kind) != nil {
			t.Fatal("scan subscription kind")
		}
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	if len(kinds) != 2 || kinds[0] != "READY" || kinds[1] != "REFUND_RESULT" {
		t.Fatalf("subscription kinds=%#v", kinds)
	}
	var refundOutbox, rejectedAvailable, readyPendingRetry int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM notification_outbox WHERE order_id=? AND kind='REFUND_RESULT'),(SELECT COUNT(*) FROM notification_consents WHERE order_id=? AND kind='REFUND_RESULT' AND decision='REJECTED' AND consumed_at IS NULL),(SELECT COUNT(*) FROM notification_outbox WHERE order_id=? AND kind='READY' AND state='PENDING' AND last_error_code='RATE_LIMITED' AND next_attempt_at IS NOT NULL)`, rejectedOrderID, rejectedOrderID, rejectedOrderID).Scan(&refundOutbox, &rejectedAvailable, &readyPendingRetry); err != nil || refundOutbox != 0 || rejectedAvailable != 1 || readyPendingRetry != 1 {
		t.Fatalf("subscription failure/rejection facts refund_outbox=%d rejected=%d pending_retry=%d err=%v", refundOutbox, rejectedAvailable, readyPendingRetry, err)
	}
}

func coreOrderForPrepayment(t *testing.T, db *sql.DB, prepaymentID string) string {
	t.Helper()
	var orderID string
	if err := db.QueryRowContext(t.Context(), `SELECT CAST(id AS CHAR) FROM orders WHERE prepayment_id=?`, prepaymentID).Scan(&orderID); err != nil || orderID == "" {
		t.Fatalf("read order for prepayment %s: %v", prepaymentID, err)
	}
	return orderID
}

func coreOrderState(t *testing.T, db *sql.DB, orderID string) string {
	t.Helper()
	var state string
	if err := db.QueryRowContext(t.Context(), `SELECT state FROM orders WHERE id=?`, orderID).Scan(&state); err != nil {
		t.Fatalf("read order %s state: %v", orderID, err)
	}
	return state
}

func coreCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(t.Context(), query, args...).Scan(&count); err != nil {
		t.Fatalf("read acceptance count: %v", err)
	}
	return count
}

func coreExpectError(t *testing.T, response map[string]any, want string) {
	t.Helper()
	if code := acceptanceString(t, acceptanceObject(t, response, "error"), "code"); code != want {
		t.Fatalf("error code=%s want=%s", code, want)
	}
}

func coreErrorCodeFromBytes(t *testing.T, raw []byte) string {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var response map[string]any
	if decoder.Decode(&response) != nil {
		t.Fatal("decode error response")
	}
	return acceptanceString(t, acceptanceObject(t, response, "error"), "code")
}

func coreRawJSON(client *http.Client, method, target, bearer, key string, body any) (int, map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequest(method, target, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("Idempotency-Key", key)
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if err != nil {
		return response.StatusCode, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		return response.StatusCode, nil, err
	}
	return response.StatusCode, decoded, nil
}
