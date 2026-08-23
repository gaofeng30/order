package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/orderadvance"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/internal/wechat"
	"github.com/gaofeng30/order/services/api/migrations"
)

const (
	acceptanceCustomerPhone = "+8613800000001"
	acceptanceMerchantPhone = "+8613800000002"
	acceptanceOwnerPhone    = "+8613800000003"
)

// TestAcceptanceLocalThreeRoleOrderToRefund is the first release-level L2
// selector. It crosses the root HTTP router and the production worker seams;
// only the three external providers are deterministic local adapters. Every
// business fact is persisted in one freshly migrated schema.
func TestAcceptanceLocalThreeRoleOrderToRefund(t *testing.T) {
	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	runtimeClock := time.Date(2026, 8, 23, 8, 0, 0, 0, shanghai)
	now := func() time.Time { return runtimeClock }

	identityRepository := identity.NewRepository(db)
	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{
		"customer-login": "acceptance-customer-openid",
		"merchant-login": "acceptance-merchant-openid",
		"owner-login":    "acceptance-owner-openid",
	}}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"acceptance-customer-openid": acceptanceCustomerPhone,
		"acceptance-merchant-openid": acceptanceMerchantPhone,
		"acceptance-owner-openid":    acceptanceOwnerPhone,
	}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	pricing := staffdiscount.NewMySQLPricing(db)

	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdminApplication := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	merchantAdminHandler := merchantidentity.NewAdminHandler(merchantAdminApplication)

	paymentProvider := newLocalPaymentProvider(now)
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-acceptance-local", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))

	notificationProvider := subscription.NewFakeProvider()
	notifications := subscription.New(db, notificationProvider)
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatalf("compose local redemption cipher: %v", err)
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)
	refundProvider := newLocalRefundProvider("order-local-mch", now)
	refundApplication := refund.New(db, refundProvider, "http://127.0.0.1:8080/api/v1/refunds/wechat/notify").
		WithNotificationEnqueuer(newRefundSubscriptionAdapter(notifications))
	adminOrderReader := adminreport.NewMySQLApplication(db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, refundApplication, adminOrderReader)
	adminReports := adminreport.NewMySQLApplication(db, adminCommands)

	registrars := []httpapi.RouteRegistrar{
		storefront.NewHandler(storefront.NewRepository(db)),
		menu.NewHandler(menu.NewRepository(db), now, menu.WithAuthenticator(miniAuth), menu.WithPricing(pricing)),
		identity.NewHandler(sessions),
		identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication),
		newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, paymentProvider)),
		httpapi.NewSubscriptionHandler(sessions, notifications, 1),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newRefundRoutes(refund.NewHandler(sessions, refundApplication, orders, refundProvider)),
		newAdminRoutes(
			sessions, merchantAdminApplication, merchantAdminHandler,
			[]adminGroupRegistrar{adminreport.NewHandler(adminReports)}, nil, nil,
		),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.NewRouter(logger, func(context.Context) httpapi.ReadinessResult {
		return httpapi.ReadinessResult{Ready: true}
	}, registrars...)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()

	customerToken := acceptanceMiniSession(t, client, server.URL, "customer-login")
	merchantToken := acceptanceMiniSession(t, client, server.URL, "merchant-login")
	ownerToken := acceptanceMiniSession(t, client, server.URL, "owner-login")
	acceptanceBindPhone(t, client, server.URL, customerToken, "customer-phone")
	acceptanceBindPhone(t, client, server.URL, merchantToken, "merchant-phone")
	acceptanceBindPhone(t, client, server.URL, ownerToken, "owner-phone")
	acceptanceMerchantLogin(t, client, server.URL, merchantToken, "SUBACCOUNT")
	acceptanceMerchantLogin(t, client, server.URL, ownerToken, "OWNER")

	pickups := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu/pickup-options", "", "", nil, http.StatusOK)
	dates := acceptanceArray(t, pickups, "dates")
	if len(dates) != 2 || acceptanceString(t, dates[0], "date") != "2026-08-23" || !acceptanceBool(t, dates[0], "available") {
		t.Fatal("pickup options did not expose the open service date from MySQL")
	}
	menuView := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/menu?date=2026-08-23&time=11:30", customerToken, "", nil, http.StatusOK)
	categories := acceptanceArray(t, menuView, "categories")
	products := acceptanceArray(t, categories[0], "products")
	if acceptanceInt(t, products[0], "original_unit_price_cents") != 1250 || acceptanceInt(t, products[0], "staff_unit_price_cents") != 1000 {
		t.Fatal("authenticated menu did not derive the staff price from shared MySQL facts")
	}

	quoteView := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/quotes", customerToken, "acceptance-quote", map[string]any{
		"contact_name": "验收顾客", "pickup_date": "2026-08-23", "pickup_time": "11:30", "order_note": "少盐",
		"items": []map[string]any{{"product_id": "1", "quantity": 2, "flavors": []string{"少饭"}, "note": "不要葱"}},
	}, http.StatusCreated)
	quoteBody := acceptanceObject(t, quoteView, "quote")
	quoteID := acceptanceString(t, quoteBody, "id")
	if acceptanceString(t, acceptanceObject(t, quoteBody, "identity"), "kind") != "STAFF" ||
		acceptanceInt(t, quoteBody, "payable_cents") != 2000 ||
		acceptanceString(t, acceptanceObject(t, quoteBody, "contact"), "masked_phone") == acceptanceCustomerPhone {
		t.Fatal("quote did not freeze server-owned staff pricing and a masked trusted phone")
	}

	prepayView := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/prepay", customerToken, "acceptance-prepay", map[string]any{
		"quote_id": quoteID,
	}, http.StatusCreated)
	prepayment := acceptanceObject(t, prepayView, "prepayment")
	prepaymentID := acceptanceString(t, prepayment, "id")
	acceptanceObject(t, prepayment, "wx_request_payment")
	confirmView := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/confirm", customerToken, "acceptance-confirm", map[string]any{
		"prepayment_id": prepaymentID,
	}, http.StatusOK)
	orderID := acceptanceString(t, confirmView, "order_id")
	if acceptanceString(t, confirmView, "state") != "ORDER_CREATED" {
		t.Fatal("server payment confirmation did not materialize an order")
	}
	replayed := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/confirm", customerToken, "acceptance-confirm", map[string]any{
		"prepayment_id": prepaymentID,
	}, http.StatusOK)
	if acceptanceString(t, replayed, "order_id") != orderID {
		t.Fatal("confirm replay created a divergent order")
	}

	reserved := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+orderID, customerToken, "", nil, http.StatusOK)
	acceptanceAssertOrderState(t, reserved, "RESERVED", false)
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/merchant/orders/"+orderID+"/ready", merchantToken, "acceptance-ready-too-early", map[string]any{}, http.StatusConflict)

	productionWorker := orderadvance.New(db)
	runtimeClock = time.Date(2026, 8, 23, 10, 59, 59, 0, shanghai)
	beforeDue, err := productionWorker.RunProductionDue(t.Context(), runtimeClock, 10)
	if err != nil || beforeDue.Scanned != 0 || beforeDue.Advanced != 0 {
		t.Fatalf("production worker advanced before the 30-minute boundary: %#v/%v", beforeDue, err)
	}
	runtimeClock = time.Date(2026, 8, 23, 11, 0, 0, 0, shanghai)
	due, err := productionWorker.RunProductionDue(t.Context(), runtimeClock, 10)
	if err != nil || due.Scanned != 1 || due.Advanced != 1 {
		t.Fatalf("production worker did not advance the due order exactly once: %#v/%v", due, err)
	}
	replayDue, err := productionWorker.RunProductionDue(t.Context(), runtimeClock.Add(time.Minute), 10)
	if err != nil || replayDue.Advanced != 0 {
		t.Fatalf("production worker replay was not monotonic: %#v/%v", replayDue, err)
	}
	preparing := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+orderID, customerToken, "", nil, http.StatusOK)
	acceptanceAssertOrderState(t, preparing, "PREPARING", false)

	acceptanceConsent(t, client, server.URL, customerToken, orderID, "READY", "acceptance-consent-ready")
	acceptanceConsent(t, client, server.URL, customerToken, orderID, "REFUND_RESULT", "acceptance-consent-refund")
	runtimeClock = time.Now().UTC().Truncate(time.Microsecond)
	ready := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/merchant/orders/"+orderID+"/ready", merchantToken, "acceptance-ready", map[string]any{}, http.StatusOK)
	acceptanceAssertOrderState(t, ready, "READY_FOR_PICKUP", false)
	readyForUser := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+orderID, customerToken, "", nil, http.StatusOK)
	acceptanceAssertOrderState(t, readyForUser, "READY_FOR_PICKUP", true)
	redemptionToken := acceptanceString(t, acceptanceObject(t, readyForUser, "order"), "redemption_token")
	completed := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/scan", merchantToken, "acceptance-redeem", map[string]any{
		"token": redemptionToken,
	}, http.StatusOK)
	acceptanceAssertOrderState(t, completed, "COMPLETED", false)

	rejectedLogin := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/auth/qrcode", "", "", map[string]any{}, http.StatusCreated)
	rejectedLoginBody := acceptanceQRLogin(t, rejectedLogin)
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/me/admin-login/approve", merchantToken, "", map[string]any{
		"login_id": rejectedLoginBody.loginID, "approval_secret": rejectedLoginBody.approvalSecret, "code": "merchant-pc-phone",
	}, http.StatusForbidden)

	pcLogin := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/auth/qrcode", "", "", map[string]any{}, http.StatusCreated)
	pcLoginBody := acceptanceQRLogin(t, pcLogin)
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/me/admin-login/approve", ownerToken, "", map[string]any{
		"login_id": pcLoginBody.loginID, "approval_secret": pcLoginBody.approvalSecret, "code": "owner-pc-phone",
	}, http.StatusOK)
	poll := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/auth/poll", "", "", map[string]any{
		"login_id": pcLoginBody.loginID, "poll_secret": pcLoginBody.pollSecret,
	}, http.StatusOK)
	pcToken := acceptanceString(t, acceptanceObject(t, poll, "session"), "token")
	if acceptanceString(t, poll, "state") != "APPROVED" {
		t.Fatal("owner PC QR approval did not yield an authenticated session")
	}

	refundView := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/orders/"+orderID+"/refund", pcToken, "acceptance-pc-refund", map[string]any{
		"reason": "甲方验收退款",
	}, http.StatusOK)
	if acceptanceString(t, acceptanceObject(t, refundView, "order"), "state") != "退款中" ||
		acceptanceString(t, acceptanceObject(t, refundView, "refund"), "state") != "退款中" {
		t.Fatal("PC refund did not durably stop at REFUNDING before provider confirmation")
	}

	workerAt := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Microsecond)
	runtimeClock = workerAt
	refundRun, err := refundApplication.RunDue(t.Context(), workerAt, 10)
	if err != nil || refundRun.Claimed != 1 || refundRun.Observed != 1 || refundRun.Applied != 1 {
		t.Fatalf("refund worker did not observe and apply deterministic provider success: %#v/%v", refundRun, err)
	}
	notificationRun, err := notifications.RunDue(t.Context(), workerAt, 10)
	if err != nil || notificationRun.Claimed != 2 || notificationRun.Sent != 2 {
		t.Fatalf("subscription worker did not deliver both accepted intents: %#v/%v", notificationRun, err)
	}
	refunded := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+orderID, customerToken, "", nil, http.StatusOK)
	acceptanceAssertOrderState(t, refunded, "REFUNDED", false)

	acceptanceAssertDurableFacts(t, db, paymentProvider, refundProvider, notificationProvider, orderID, quoteID, prepaymentID, pcToken, customerToken)
}

type acceptanceLoginProvider struct{ openIDs map[string]string }

func (provider acceptanceLoginProvider) Exchange(_ context.Context, code string) (string, error) {
	if openID := provider.openIDs[code]; openID != "" {
		return openID, nil
	}
	return "", wechat.ErrLoginRejected
}

type acceptancePhoneProvider struct{ phones map[string]string }

func (provider acceptancePhoneProvider) Exchange(_ context.Context, code, openID string) (string, error) {
	if strings.TrimSpace(code) == "" || provider.phones[openID] == "" {
		return "", wechat.ErrPhoneCodeRejected
	}
	return provider.phones[openID], nil
}

func acceptanceLeaseOwner() ([16]byte, error) { return [16]byte{0x95}, nil }

func acceptanceFreshMySQL(t *testing.T) *sql.DB {
	t.Helper()
	keys := []string{"ORDER_TEST_MYSQL_HOST", "ORDER_TEST_MYSQL_PORT", "ORDER_TEST_MYSQL_USER", "ORDER_TEST_MYSQL_PASSWORD", "ORDER_TEST_MYSQL_TLS_MODE", "ORDER_TEST_MYSQL_INSTANCE", "ORDER_TEST_MYSQL_ISOLATED"}
	present := 0
	for _, key := range keys {
		if os.Getenv(key) != "" {
			present++
		}
	}
	if present == 0 {
		t.Skip("isolated order-mysql-w3 is not configured")
	}
	if present != len(keys) || os.Getenv("ORDER_TEST_MYSQL_INSTANCE") != "order-mysql-w3" || os.Getenv("ORDER_TEST_MYSQL_ISOLATED") != "YES" {
		t.Fatal("acceptance MySQL environment must be complete and identify the isolated order-mysql-w3 instance")
	}
	port, err := strconv.ParseUint(os.Getenv("ORDER_TEST_MYSQL_PORT"), 10, 16)
	if err != nil || port == 0 {
		t.Fatal("acceptance MySQL port is invalid")
	}
	configuration := database.ConnectionConfig{
		Host: os.Getenv("ORDER_TEST_MYSQL_HOST"), Port: uint16(port), Database: "mysql",
		User: os.Getenv("ORDER_TEST_MYSQL_USER"), Password: os.Getenv("ORDER_TEST_MYSQL_PASSWORD"), TLSMode: os.Getenv("ORDER_TEST_MYSQL_TLS_MODE"),
	}
	admin := acceptanceOpenMySQL(t, configuration)
	var version string
	if err := admin.QueryRowContext(t.Context(), "SELECT VERSION()").Scan(&version); err != nil || !strings.HasPrefix(version, "8.0.") {
		t.Fatal("acceptance selector requires reachable MySQL 8.0")
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		t.Fatal("create isolated schema identity")
	}
	schema := "order_acceptance_" + hex.EncodeToString(random)
	if _, err := admin.ExecContext(t.Context(), "CREATE DATABASE `"+schema+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("create isolated acceptance schema")
	}
	t.Cleanup(func() {
		if strings.HasPrefix(schema, "order_acceptance_") && len(schema) == len("order_acceptance_")+32 {
			_, _ = admin.ExecContext(context.Background(), "DROP DATABASE `"+schema+"`")
		}
	})
	configuration.Database = schema
	db := acceptanceOpenMySQL(t, configuration)
	set, err := migrate.Load(migrations.FS)
	if err != nil || len(set) != 44 || set[43].Version != 44 {
		t.Fatalf("load exact v1-v44 migration ledger: count=%d err=%v", len(set), err)
	}
	applied, err := migrate.Run(t.Context(), db, set)
	if err != nil || applied.FromVersion != 0 || applied.ToVersion != 44 || applied.AppliedCount != 44 {
		t.Fatalf("migrate fresh acceptance schema: %#v/%v", applied, err)
	}
	return db
}

func acceptanceOpenMySQL(t *testing.T, configuration database.ConnectionConfig) *sql.DB {
	t.Helper()
	db, err := database.Open(configuration)
	if err != nil {
		t.Fatalf("open acceptance MySQL: %s", database.Reason(err))
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("acceptance MySQL is unreachable")
	}
	return db
}

func acceptanceSeedSharedFacts(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json,record_version) VALUES(1,'验收食堂','验收园区','北门取餐点','本地验收','open',JSON_ARRAY('少饭'),1)`, nil},
		{`INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,created_at,updated_at) VALUES(1,?,'验收主账号','OWNER',TRUE,1,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6)),(2,?,'验收子账号','SUBACCOUNT',TRUE,1,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, []any{acceptanceOwnerPhone, acceptanceMerchantPhone}},
		{`INSERT INTO service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at) VALUES('2026-08-23',TRUE,1,1,UTC_TIMESTAMP(6)),('2026-08-24',FALSE,1,1,UTC_TIMESTAMP(6))`, nil},
		{`INSERT INTO categories(id,name,name_key,sort_order,is_active,record_version) VALUES(1,'验收套餐',CONVERT('验收套餐' USING binary),1,TRUE,1)`, nil},
		{`INSERT INTO products(id,category_id,name,name_key,description,specification,images_json,price_cents,sort_order,is_listed,meal_period,record_version) VALUES(1,1,'工作餐',CONVERT('工作餐' USING binary),'两荤一素','份',JSON_ARRAY(),1250,1,TRUE,'lunch',1)`, nil},
		{`INSERT INTO staff_whitelist(id,phone,name,name_key,enabled,record_version,created_at,updated_at) VALUES(1,?,'验收员工',CONVERT('验收员工' USING binary),TRUE,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, []any{acceptanceCustomerPhone}},
		{`INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES(1,80,1,1,UTC_TIMESTAMP(6))`, nil},
	}
	for index, statement := range statements {
		if _, err := db.ExecContext(t.Context(), statement.query, statement.args...); err != nil {
			t.Fatalf("seed acceptance fact %d: %v", index, err)
		}
	}
}

func acceptanceMiniSession(t *testing.T, client *http.Client, origin, code string) string {
	t.Helper()
	response := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/auth/miniprogram/session", "", "", map[string]any{"code": code}, http.StatusCreated)
	return acceptanceString(t, response, "access_token")
}

func acceptanceBindPhone(t *testing.T, client *http.Client, origin, token, code string) {
	t.Helper()
	response := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/me/bind-phone", token, "", map[string]any{"code": code}, http.StatusOK)
	if !acceptanceBool(t, response, "primary_phone_bound") || acceptanceString(t, response, "masked_phone") == "" {
		t.Fatal("trusted phone binding did not complete")
	}
}

func acceptanceMerchantLogin(t *testing.T, client *http.Client, origin, token, role string) {
	t.Helper()
	response := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/me/merchant-login", token, "", map[string]any{"code": "merchant-phone-code"}, http.StatusOK)
	merchant := acceptanceObject(t, response, "merchant")
	if !acceptanceBool(t, merchant, "bound") || acceptanceString(t, merchant, "role") != role {
		t.Fatalf("merchant login did not bind role %s", role)
	}
}

func acceptanceConsent(t *testing.T, client *http.Client, origin, token, orderID, kind, key string) {
	t.Helper()
	response := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/orders/"+orderID+"/subscriptions", token, key, map[string]any{"kind": kind, "decision": "ACCEPTED"}, http.StatusOK)
	if !acceptanceBool(t, acceptanceObject(t, response, "subscription"), "available") {
		t.Fatal("accepted subscription was not persisted as available")
	}
}

type acceptancePCLogin struct{ loginID, pollSecret, approvalSecret string }

func acceptanceQRLogin(t *testing.T, response map[string]any) acceptancePCLogin {
	t.Helper()
	payload, err := url.Parse(acceptanceString(t, response, "qr_payload"))
	if err != nil {
		t.Fatal("parse PC QR payload")
	}
	result := acceptancePCLogin{
		loginID: acceptanceString(t, response, "login_id"), pollSecret: acceptanceString(t, response, "poll_secret"),
		approvalSecret: payload.Query().Get("approval_secret"),
	}
	if result.loginID == "" || result.pollSecret == "" || result.approvalSecret == "" {
		t.Fatal("PC QR response omitted an intrinsic login proof")
	}
	return result
}

func acceptanceAssertOrderState(t *testing.T, response map[string]any, state string, wantToken bool) {
	t.Helper()
	order := acceptanceObject(t, response, "order")
	if acceptanceString(t, order, "state") != state {
		t.Fatalf("order state = %q, want %q", acceptanceString(t, order, "state"), state)
	}
	_, hasToken := order["redemption_token"]
	if hasToken != wantToken {
		t.Fatalf("order state %s token visibility = %v, want %v", state, hasToken, wantToken)
	}
}

func acceptanceAssertDurableFacts(t *testing.T, db *sql.DB, payments *localPaymentProvider, refunds *localRefundProvider, notifications *subscription.FakeProvider, orderID, quoteID, prepaymentID, pcToken, customerToken string) {
	t.Helper()
	var users, sessions, boundPhones int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*),SUM(primary_phone IS NOT NULL AND primary_phone_bound_at IS NOT NULL) FROM miniprogram_users`).Scan(&users, &boundPhones); err != nil {
		t.Fatal("read durable user facts")
	}
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM miniprogram_sessions`).Scan(&sessions); err != nil || users != 3 || sessions != 3 || boundPhones != 3 {
		t.Fatalf("identity durability counts users=%d sessions=%d bound=%d", users, sessions, boundPhones)
	}
	var quotePhone, identityKind string
	var quoteRate, payable, quoteItems, digestBytes int
	if err := db.QueryRowContext(t.Context(), `SELECT CONVERT(contact_phone_snapshot USING ascii),identity_kind,discount_rate_percent,payable_cents,OCTET_LENGTH(snapshot_digest),(SELECT COUNT(*) FROM quote_items WHERE quote_id=q.id) FROM quotes q WHERE id=?`, quoteID).
		Scan(&quotePhone, &identityKind, &quoteRate, &payable, &digestBytes, &quoteItems); err != nil || quotePhone != acceptanceCustomerPhone || identityKind != "STAFF" || quoteRate != 80 || payable != 2000 || digestBytes != 32 || quoteItems != 1 {
		t.Fatalf("quote durable snapshot invariant failed kind=%s rate=%d payable=%d digest=%d items=%d err=%v", identityKind, quoteRate, payable, digestBytes, quoteItems, err)
	}
	var outTradeNo, paymentState, paymentMaterialization, paymentObservation string
	if err := db.QueryRowContext(t.Context(), `SELECT CONVERT(p.out_trade_no USING utf8mb4),p.provider_state,p.materialization_state,(SELECT apply_state FROM payment_observations WHERE prepayment_id=p.id AND provider_state='PAID' LIMIT 1) FROM prepayments p WHERE id=?`, prepaymentID).
		Scan(&outTradeNo, &paymentState, &paymentMaterialization, &paymentObservation); err != nil || paymentState != "PAID" || paymentMaterialization != "APPLIED" || paymentObservation != "APPLIED" {
		t.Fatalf("payment durable facts state=%s materialization=%s observation=%s err=%v", paymentState, paymentMaterialization, paymentObservation, err)
	}
	// The local wrapper performs one provider confirmation operation; its
	// deterministic fake is read once before and once after MarkPaid.
	if payments.fake.CreateCount(outTradeNo) != 1 || payments.fake.QueryCount(outTradeNo) != 2 {
		t.Fatalf("payment provider calls create=%d query=%d", payments.fake.CreateCount(outTradeNo), payments.fake.QueryCount(outTradeNo))
	}
	var orderState string
	var orderItems, pickupNumber, tokenCipherBytes, tokenHashBytes int
	var preparingAt, readyAt, completedAt, refundingAt, refundedAt sql.NullTime
	if err := db.QueryRowContext(t.Context(), `SELECT state,(SELECT COUNT(*) FROM order_items WHERE order_id=o.id),pickup_number,COALESCE(OCTET_LENGTH(redemption_token_ciphertext),0),COALESCE(OCTET_LENGTH(redemption_token_hash),0),preparing_at,ready_at,completed_at,refunding_at,refunded_at FROM orders o WHERE id=?`, orderID).
		Scan(&orderState, &orderItems, &pickupNumber, &tokenCipherBytes, &tokenHashBytes, &preparingAt, &readyAt, &completedAt, &refundingAt, &refundedAt); err != nil || orderState != "REFUNDED" || orderItems != 1 || pickupNumber != 1 || tokenCipherBytes != 0 || tokenHashBytes != 32 || !preparingAt.Valid || !readyAt.Valid || !completedAt.Valid || !refundingAt.Valid || !refundedAt.Valid {
		t.Fatalf("six-state durable history invariant failed state=%s items=%d number=%d cipher=%d hash=%d err=%v", orderState, orderItems, pickupNumber, tokenCipherBytes, tokenHashBytes, err)
	}
	var lastNumber int
	if err := db.QueryRowContext(t.Context(), `SELECT last_number FROM pickup_sequences WHERE service_date='2026-08-23'`).Scan(&lastNumber); err != nil || lastNumber != 1 {
		t.Fatalf("pickup sequence burned or diverged: %d/%v", lastNumber, err)
	}
	var outRefundNo, refundState, refundMaterialization, refundObservation string
	var refundAmount int
	if err := db.QueryRowContext(t.Context(), `SELECT CONVERT(r.out_refund_no USING utf8mb4),r.provider_state,r.materialization_state,r.amount_cents,(SELECT apply_state FROM refund_observations WHERE refund_id=r.id AND provider_state='SUCCESS' LIMIT 1) FROM refunds r WHERE order_id=?`, orderID).
		Scan(&outRefundNo, &refundState, &refundMaterialization, &refundAmount, &refundObservation); err != nil || refundState != "SUCCESS" || refundMaterialization != "APPLIED" || refundAmount != 2000 || refundObservation != "APPLIED" {
		t.Fatalf("refund durable facts state=%s materialization=%s amount=%d observation=%s err=%v", refundState, refundMaterialization, refundAmount, refundObservation, err)
	}
	// The refund wrapper has the same before/after deterministic observation
	// shape while still making one worker-level provider query operation.
	if refunds.fake.CreateCount(outRefundNo) != 1 || refunds.fake.QueryCount(outRefundNo) != 2 {
		t.Fatalf("refund provider calls create=%d query=%d", refunds.fake.CreateCount(outRefundNo), refunds.fake.QueryCount(outRefundNo))
	}
	var consents, outbox, sent int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM notification_consents WHERE order_id=?),(SELECT COUNT(*) FROM notification_outbox WHERE order_id=?),(SELECT COUNT(*) FROM notification_outbox WHERE order_id=? AND state='SENT')`, orderID, orderID, orderID).
		Scan(&consents, &outbox, &sent); err != nil || consents != 2 || outbox != 2 || sent != 2 || len(notifications.Deliveries()) != 2 {
		t.Fatalf("subscription durability consents=%d outbox=%d sent=%d deliveries=%d err=%v", consents, outbox, sent, len(notifications.Deliveries()), err)
	}
	var auditCount, requiredActions int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*),COUNT(DISTINCT CASE WHEN action IN ('quote.create','payment.prepare','payment.confirm','order.production_due','fulfillment.mark_ready','fulfillment.redeem_token','refund.request') THEN action END) FROM action_audits`).Scan(&auditCount, &requiredActions); err != nil || auditCount < 12 || requiredActions != 7 {
		t.Fatalf("audit trail incomplete count=%d required_actions=%d err=%v", auditCount, requiredActions, err)
	}
	var leaked int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM action_audits WHERE CONCAT_WS('',COALESCE(CONVERT(before_state_json USING utf8mb4),''),COALESCE(CONVERT(after_state_json USING utf8mb4),''),COALESCE(CONVERT(response_json USING utf8mb4),'')) LIKE ?`, "%"+acceptanceCustomerPhone+"%").Scan(&leaked); err != nil || leaked != 0 {
		t.Fatalf("audit evidence leaked a trusted phone: count=%d err=%v", leaked, err)
	}
	var rawMiniToken, rawPCToken int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM miniprogram_sessions WHERE HEX(token_hash)=HEX(?)),(SELECT COUNT(*) FROM merchant_pc_sessions WHERE HEX(access_token_hash)=HEX(?))`, []byte(customerToken), []byte(pcToken)).Scan(&rawMiniToken, &rawPCToken); err != nil || rawMiniToken != 0 || rawPCToken != 0 {
		t.Fatalf("raw session credential was persisted mini=%d pc=%d err=%v", rawMiniToken, rawPCToken, err)
	}
}

func acceptanceHTTP(t *testing.T, client *http.Client, method, target, bearer, idempotency string, body any, wantStatus int) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal("encode acceptance request")
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(t.Context(), method, target, reader)
	if err != nil {
		t.Fatal("build acceptance request")
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if idempotency != "" {
		request.Header.Set("Idempotency-Key", idempotency)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("acceptance HTTP %s failed: %v", method, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if err != nil {
		t.Fatal("read acceptance HTTP response")
	}
	if response.StatusCode != wantStatus {
		var envelope map[string]any
		_ = json.Unmarshal(data, &envelope)
		code := ""
		if problem, ok := envelope["error"].(map[string]any); ok {
			code, _ = problem["code"].(string)
		}
		t.Fatalf("acceptance HTTP %s %s status=%d want=%d error_code=%s", method, request.URL.Path, response.StatusCode, wantStatus, code)
	}
	if len(data) == 0 {
		return map[string]any{}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatal("decode acceptance HTTP response")
	}
	return decoded
}

func acceptanceObject(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("acceptance response field %s is not an object", key)
	}
	return value
}

func acceptanceArray(t *testing.T, object map[string]any, key string) []map[string]any {
	t.Helper()
	raw, ok := object[key].([]any)
	if !ok || len(raw) == 0 {
		t.Fatalf("acceptance response field %s is not a non-empty array", key)
	}
	result := make([]map[string]any, len(raw))
	for index, item := range raw {
		var itemOK bool
		result[index], itemOK = item.(map[string]any)
		if !itemOK {
			t.Fatalf("acceptance response field %s[%d] is not an object", key, index)
		}
	}
	return result
}

func acceptanceString(t *testing.T, object map[string]any, key string) string {
	t.Helper()
	value, ok := object[key].(string)
	if !ok || value == "" {
		t.Fatalf("acceptance response field %s is not a non-empty string", key)
	}
	return value
}

func acceptanceBool(t *testing.T, object map[string]any, key string) bool {
	t.Helper()
	value, ok := object[key].(bool)
	if !ok {
		t.Fatalf("acceptance response field %s is not boolean", key)
	}
	return value
}

func acceptanceInt(t *testing.T, object map[string]any, key string) int64 {
	t.Helper()
	value, ok := object[key].(json.Number)
	if !ok {
		t.Fatalf("acceptance response field %s is not a number", key)
	}
	parsed, err := value.Int64()
	if err != nil {
		t.Fatalf("acceptance response field %s is not an integer", key)
	}
	return parsed
}
