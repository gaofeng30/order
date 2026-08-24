package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/billing"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/orderadvance"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gin-gonic/gin"
)

// TestPCClosureUI1Server is a process-scoped harness for the Chrome selector.
// It exists only in this _test.go file, owns a random fresh-v44 schema, and is
// stopped by the selector before go test returns and drops that schema.
func TestPCClosureUI1Server(t *testing.T) {
	if os.Getenv("ORDER_PC_CLOSURE_SERVE") != "YES" {
		t.Skip("PC closure UI1 server not requested")
	}
	infoFile := os.Getenv("ORDER_PC_CLOSURE_INFO_FILE")
	stopFile := os.Getenv("ORDER_PC_CLOSURE_STOP_FILE")
	if !strings.HasPrefix(infoFile, "/private/tmp/order-pc-closure-") || !strings.HasPrefix(stopFile, "/private/tmp/order-pc-closure-") {
		t.Fatal("PC closure harness paths must be exact private/tmp files")
	}
	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)
	if _, err := db.ExecContext(t.Context(), `UPDATE service_dates SET is_open=TRUE WHERE service_date='2026-08-24'`); err != nil {
		t.Fatal("open second explicit fixture service date")
	}
	clock := &pcClosureMutableClock{value: time.Date(2026, 8, 23, 11, 10, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))}
	now := clock.Now
	identityRepository := identity.NewRepository(db)
	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{"pc-ui1-login": "pc-ui1-owner-openid"}}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{"pc-ui1-owner-openid": acceptanceOwnerPhone}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	paymentProvider := newLocalPaymentProvider(now)
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-pc-closure-ui1", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	notifications := subscription.New(db, subscription.NewFakeProvider())
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatal("compose UI1 redemption cipher")
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)
	refundProvider := newLocalRefundProvider("order-local-mch", time.Now)
	refundApplication := refund.New(db, refundProvider, "http://127.0.0.1/api/v1/refunds/wechat/notify")
	adminReader := adminreport.NewMySQLApplication(db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, refundApplication, adminReader)
	adminReports := adminreport.NewMySQLApplication(db, adminCommands)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpapi.NewRouter(
		logger,
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		pcClosureClockRoutes{clock: clock},
		storefront.NewHandler(storefront.NewRepository(db)),
		menu.NewHandler(menu.NewRepository(db), now, menu.WithAuthenticator(miniAuth), menu.WithPricing(staffdiscount.NewMySQLPricing(db))),
		identity.NewHandler(sessions), identity.NewPhoneHandler(sessions, phoneService), merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication), newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, paymentProvider)),
		orderquery.NewHandler(sessions, orders), fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newRefundRoutes(refund.NewHandler(sessions, refundApplication, orders, refundProvider)),
		newAdminRoutes(sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin), []adminGroupRegistrar{
			storefront.NewAdminHandler(storefront.NewMySQLAdminApplication(db)),
			adminreport.NewHandler(adminReports),
		}, nil, nil),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	workerContext, cancelWorkers := context.WithCancel(t.Context())
	t.Cleanup(cancelWorkers)
	go runRefundWorker(workerContext, refundApplication, logger)
	var schema string
	if err := db.QueryRowContext(t.Context(), `SELECT DATABASE()`).Scan(&schema); err != nil || !strings.HasPrefix(schema, "order_acceptance_") {
		t.Fatal("read UI1 private schema")
	}
	info := fmt.Sprintf("{\"origin\":%q,\"schema\":%q,\"login_code\":\"pc-ui1-login\",\"phone_code\":\"pc-ui1-phone\"}\n", server.URL, schema)
	if err := os.WriteFile(infoFile, []byte(info), 0o600); err != nil {
		t.Fatal("publish UI1 harness info")
	}
	t.Cleanup(func() { _ = os.Remove(infoFile) })
	deadline := time.NewTimer(10 * time.Minute)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("PC closure UI1 harness timed out")
		case <-ticker.C:
			if _, err := os.Stat(stopFile); err == nil {
				_ = os.Remove(stopFile)
				return
			}
		}
	}
}

type pcClosureMutableClock struct {
	mu    sync.RWMutex
	value time.Time
}

func (clock *pcClosureMutableClock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.value
}

func (clock *pcClosureMutableClock) Set(value time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.value = value
}

type pcClosureClockRoutes struct{ clock *pcClosureMutableClock }

func (routes pcClosureClockRoutes) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/api/v1/__acceptance/clock", func(c *gin.Context) {
		var input struct {
			Date string `json:"date"`
		}
		if c.ShouldBindJSON(&input) != nil || (input.Date != "2026-08-23" && input.Date != "2026-08-24") {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST"}})
			return
		}
		value, _ := time.ParseInLocation("2006-01-02 15:04", input.Date+" 11:10", time.FixedZone("Asia/Shanghai", 8*60*60))
		routes.clock.Set(value)
		c.JSON(http.StatusOK, gin.H{"date": input.Date})
	})
}

// TestAcceptancePCPagesCloseWithDerivedFactsAndFailureShields is the fresh-v44
// L2 selector behind PAGE-PC01--PC04. Normal orders are created only through
// root-composed HTTP. The single direct mutation is an explicitly named fault
// injection that corrupts one immutable quote digest for the PC04 shield.
func TestAcceptancePCPagesCloseWithDerivedFactsAndFailureShields(t *testing.T) {
	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	runtimeClock := time.Date(2026, 8, 23, 8, 0, 0, 0, shanghai)
	now := func() time.Time { return runtimeClock }
	identityRepository := identity.NewRepository(db)
	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{
		"pc-closure-customer": "pc-closure-customer-openid",
		"pc-closure-owner":    "pc-closure-owner-openid",
	}}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"pc-closure-customer-openid": acceptanceCustomerPhone,
		"pc-closure-owner-openid":    acceptanceOwnerPhone,
	}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	paymentProvider := newLocalPaymentProvider(now)
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-pc-closure", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	notifications := subscription.New(db, subscription.NewFakeProvider())
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatal("compose redemption cipher")
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)
	refundProvider := newLocalRefundProvider("order-local-mch", now)
	refundApplication := refund.New(db, refundProvider, "http://127.0.0.1/api/v1/refunds/wechat/notify")
	adminReader := adminreport.NewMySQLApplication(db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, refundApplication, adminReader)
	adminReports := adminreport.NewMySQLApplication(db, adminCommands)

	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		storefront.NewHandler(storefront.NewRepository(db)),
		menu.NewHandler(menu.NewRepository(db), now, menu.WithAuthenticator(miniAuth), menu.WithPricing(staffdiscount.NewMySQLPricing(db))),
		identity.NewHandler(sessions),
		identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication),
		newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, paymentProvider)),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newRefundRoutes(refund.NewHandler(sessions, refundApplication, orders, refundProvider)),
		newAdminRoutes(
			sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin),
			[]adminGroupRegistrar{adminreport.NewHandler(adminReports)}, nil, nil,
		),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()

	customerToken := acceptanceMiniSession(t, client, server.URL, "pc-closure-customer")
	ownerToken := acceptanceMiniSession(t, client, server.URL, "pc-closure-owner")
	acceptanceBindPhone(t, client, server.URL, customerToken, "pc-closure-customer-phone")
	acceptanceBindPhone(t, client, server.URL, ownerToken, "pc-closure-owner-phone")
	acceptanceMerchantLogin(t, client, server.URL, ownerToken, "OWNER")
	pcToken := pcClosureOwnerSession(t, client, server.URL, ownerToken)

	completed := pcClosureCreatePaidOrder(t, client, server.URL, customerToken, "completed", 1)
	unclaimed := pcClosureCreatePaidOrder(t, client, server.URL, customerToken, "unclaimed", 2)
	production := orderadvance.New(db)
	runtimeClock = time.Date(2026, 8, 23, 11, 0, 0, 0, shanghai)
	if run, runErr := production.RunProductionDue(t.Context(), runtimeClock, 10); runErr != nil || run.Advanced != 2 {
		t.Fatalf("advance two paid orders to PREPARING: %#v/%v", run, runErr)
	}
	for _, order := range []pcClosureOrder{completed, unclaimed} {
		acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/merchant/orders/"+order.id+"/ready", ownerToken, "pc-ready-"+order.key, map[string]any{}, http.StatusOK)
	}
	ready := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/orders/"+completed.id, customerToken, "", nil, http.StatusOK)
	redemptionToken := acceptanceString(t, acceptanceObject(t, ready, "order"), "redemption_token")
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/verify/scan", ownerToken, "pc-redeem-completed", map[string]any{"token": redemptionToken}, http.StatusOK)

	// PC01: only COMPLETED is effective revenue/order/sales. The past READY
	// order is queryable as unclaimed but cannot alter any dashboard aggregate.
	beforeReads := pcClosureReadFacts(t, db)
	stats := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/stats", pcToken, "", nil, http.StatusOK)
	if acceptanceInt(t, stats, "month_revenue_cents") != 1000 || acceptanceInt(t, stats, "month_orders") != 1 || acceptanceInt(t, stats, "pending_production") != 0 {
		t.Fatalf("PC01 effective stats included non-completed facts: %#v", stats)
	}
	var ownerUserID uint64
	if err := db.QueryRowContext(t.Context(), `SELECT bound_user_id FROM merchant_accounts WHERE role='OWNER'`).Scan(&ownerUserID); err != nil || ownerUserID == 0 {
		t.Fatal("read bound PC owner")
	}
	dayStats, err := adminReports.Stats(t.Context(), ownerUserID, time.Date(2026, 8, 23, 16, 0, 0, 0, shanghai))
	if err != nil || dayStats.TodayRevenueCents != 1000 || dayStats.TodayOrders != 1 || len(dayStats.ProductSales) != 1 || dayStats.ProductSales[0].Quantity != 1 {
		t.Fatalf("PC01 sales ranking included unclaimed items: %#v/%v", dayStats, err)
	}

	// PC02: the public query combines date with order number, pickup number,
	// or trusted phone. Returned monetary facts remain immutable snapshots.
	for label, query := range map[string]string{
		"order number":  completed.orderNo,
		"pickup number": completed.pickupNumber,
		"phone":         acceptanceCustomerPhone,
	} {
		view := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/orders?date=2026-08-23&q="+url.QueryEscape(query), pcToken, "", nil, http.StatusOK)
		items := acceptanceArray(t, view, "orders")
		if len(items) == 0 {
			t.Fatalf("PC02 %s search returned no orders", label)
		}
		if label != "phone" && acceptanceString(t, items[0], "id") != completed.id {
			t.Fatalf("PC02 %s search returned wrong order: %#v", label, items)
		}
		if label != "phone" && acceptanceInt(t, items[0], "payable_cents") != 1000 {
			t.Fatalf("PC02 %s search changed immutable payment amount", label)
		}
	}
	unclaimedView := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/orders?state="+url.QueryEscape("待取餐")+"&unclaimed=true", pcToken, "", nil, http.StatusOK)
	if list := acceptanceArray(t, unclaimedView, "orders"); len(list) != 1 || acceptanceString(t, list[0], "id") != unclaimed.id {
		t.Fatalf("PC02 unclaimed query is not the past READY order: %#v", list)
	}

	financeRange := "from=2026-08-01&to=2026-08-31"
	page1 := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/finance/payments?"+financeRange+"&limit=1", pcToken, "", nil, http.StatusOK)
	next := acceptanceString(t, page1, "next_after_id")
	page2 := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/finance/payments?"+financeRange+"&limit=1&after_id="+next, pcToken, "", nil, http.StatusOK)
	if len(acceptanceArray(t, page1, "payments")) != 1 || len(acceptanceArray(t, page2, "payments")) != 1 {
		t.Fatal("PC03 payments cursor did not expose both server orders")
	}
	acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/finance/summary?"+financeRange, pcToken, "", nil, http.StatusOK)
	pcClosureAssertCSV(t, client, server.URL+"/api/v1/admin/finance/export?"+financeRange, pcToken, completed.orderNo)
	afterReads := pcClosureReadFacts(t, db)
	if beforeReads != afterReads {
		t.Fatalf("PC01--03 read-only queries mutated transaction facts: before=%#v after=%#v", beforeReads, afterReads)
	}

	// PC03 L2: the fake provider produces one matched payment plus explicit
	// provider-only and system-only facts. Mismatch is durable; unavailability
	// produces no false reconciliation audit and changes no transaction state.
	billDate := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	matching := pcClosurePaymentBillEntry(t, db, completed.prepaymentID)
	providerOnly := billing.BillEntry{Kind: billing.EntryPayment, OutTradeNo: "PROVIDER-ONLY-PC03", ProviderID: "WX-PROVIDER-ONLY-PC03", AmountCents: 777, Currency: "CNY", State: "SUCCESS", OccurredAt: billDate.Add(time.Hour)}
	billProvider := billing.NewFakeBillProvider()
	billProvider.SetBill(billDate, []billing.BillEntry{matching, providerOnly})
	billingApplication := billing.New(db, billProvider)
	result, billErr := billingApplication.RunReconcile(t.Context(), billDate, 100)
	if !errors.Is(billErr, billing.ErrBillMismatch) || result.Matched != 1 || len(result.ProviderOnly) != 1 || len(result.SystemOnly) != 1 || result.ProviderOnly[0].OutTradeNo != providerOnly.OutTradeNo {
		t.Fatalf("PC03 one-sided reconciliation = %#v/%v", result, billErr)
	}
	var reconcileAudits int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM action_audits WHERE action='billing.reconcile' AND reason_code='BILL_MISMATCH'`).Scan(&reconcileAudits); err != nil || reconcileAudits != 1 {
		t.Fatalf("PC03 mismatch audit = %d/%v", reconcileAudits, err)
	}
	beforeUnavailable := pcClosureReadFacts(t, db)
	billProvider.SetUnavailable(true)
	if _, err := billingApplication.RunReconcile(t.Context(), billDate.AddDate(0, 0, 1), 100); !errors.Is(err, billing.ErrBillUnavailable) {
		t.Fatalf("PC03 unavailable bill was not fail-closed: %v", err)
	}
	if after := pcClosureReadFacts(t, db); beforeUnavailable != after {
		t.Fatalf("PC03 unavailable bill mutated facts: before=%#v after=%#v", beforeUnavailable, after)
	}

	// PC02 full refund keeps the original amount and only provider finality
	// advances REFUNDING to REFUNDED.
	refundView := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/orders/"+completed.id+"/refund", pcToken, "pc02-refund", map[string]any{"reason": "PC02验收全额退款"}, http.StatusOK)
	if acceptanceInt(t, acceptanceObject(t, refundView, "refund"), "amount_cents") != 1000 || acceptanceString(t, acceptanceObject(t, refundView, "order"), "state") != "退款中" {
		t.Fatal("PC02 refund did not preserve the immutable full amount")
	}
	runtimeClock = time.Now().UTC().Truncate(time.Microsecond)
	if run, runErr := refundApplication.RunDue(t.Context(), runtimeClock, 10); runErr != nil || run.Applied != 1 {
		t.Fatalf("PC02 refund worker finality = %#v/%v", run, runErr)
	}
	refundsView := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/finance/refunds?"+financeRange, pcToken, "", nil, http.StatusOK)
	if items := acceptanceArray(t, refundsView, "refunds"); len(items) != 1 || acceptanceString(t, items[0], "state") != "已退款" || acceptanceInt(t, items[0], "amount_cents") != 1000 {
		t.Fatalf("PC03 refund ledger is not final and immutable: %#v", items)
	}

	// PC04 fault fixture: corrupt only the stored digest after a normal HTTP
	// prepay. Confirm must defer, manual materialization must reject without an
	// order or burned number, while the full paid-prepayment refund converges.
	runtimeClock = time.Date(2026, 8, 23, 8, 0, 0, 0, shanghai)
	corrupt := pcClosureCreatePrepayment(t, client, server.URL, customerToken, "corrupt", 1)
	if _, err := db.ExecContext(t.Context(), `UPDATE quotes SET snapshot_digest=? WHERE id=?`, make([]byte, 32), corrupt.quoteID); err != nil {
		t.Fatal("PC04 fault fixture could not corrupt quote digest")
	}
	confirm := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/orders/confirm", customerToken, "pc-confirm-corrupt", map[string]any{"prepayment_id": corrupt.prepaymentID}, http.StatusAccepted)
	if acceptanceString(t, confirm, "state") != "PENDING" {
		t.Fatalf("PC04 corrupt snapshot confirm = %#v", confirm)
	}
	pending := acceptanceHTTP(t, client, http.MethodGet, server.URL+"/api/v1/admin/pending-payments", pcToken, "", nil, http.StatusOK)
	pendingItems := acceptanceArray(t, pending, "prepayments")
	if len(pendingItems) != 1 || acceptanceString(t, pendingItems[0], "id") != corrupt.prepaymentID || acceptanceString(t, pendingItems[0], "blocking_reason") != "QUOTE_SNAPSHOT_INVALID" {
		t.Fatalf("PC04 corrupt snapshot is not visible as one pending payment: %#v", pendingItems)
	}
	ordersBefore, sequenceBefore := pcClosureOrderAndSequenceCounts(t, db)
	acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/pending-payments/"+corrupt.prepaymentID, pcToken, "pc04-materialize-corrupt", map[string]any{"action": "MATERIALIZE", "reason": ""}, http.StatusConflict)
	ordersAfter, sequenceAfter := pcClosureOrderAndSequenceCounts(t, db)
	if ordersAfter != ordersBefore || sequenceAfter != sequenceBefore {
		t.Fatalf("PC04 corrupt MATERIALIZE created order/burned number: orders %d->%d sequence %d->%d", ordersBefore, ordersAfter, sequenceBefore, sequenceAfter)
	}
	voided := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/admin/pending-payments/"+corrupt.prepaymentID, pcToken, "pc04-refund-corrupt", map[string]any{"action": "REFUND", "reason": "快照损坏无法安全补单"}, http.StatusOK)
	if acceptanceInt(t, acceptanceObject(t, voided, "refund"), "amount_cents") != 1000 {
		t.Fatal("PC04 pending refund changed the paid amount")
	}
	runtimeClock = time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond)
	if run, runErr := refundApplication.RunDue(t.Context(), runtimeClock, 10); runErr != nil || run.Applied != 1 {
		t.Fatalf("PC04 pending refund worker finality = %#v/%v", run, runErr)
	}
	ordersFinal, sequenceFinal := pcClosureOrderAndSequenceCounts(t, db)
	if ordersFinal != ordersBefore || sequenceFinal != sequenceBefore {
		t.Fatal("PC04 refund path created an order or burned a pickup number")
	}
	var refundState, refundMaterialization string
	if err := db.QueryRowContext(t.Context(), `SELECT provider_state,materialization_state FROM refunds WHERE prepayment_id=?`, corrupt.prepaymentID).Scan(&refundState, &refundMaterialization); err != nil || refundState != "SUCCESS" || refundMaterialization != "APPLIED" {
		t.Fatalf("PC04 pending refund did not converge: %s/%s/%v", refundState, refundMaterialization, err)
	}
}

type pcClosureOrder struct {
	key, id, orderNo, pickupNumber, quoteID, prepaymentID string
}

func pcClosureCreatePaidOrder(t *testing.T, client *http.Client, origin, token, key string, quantity int) pcClosureOrder {
	t.Helper()
	prepared := pcClosureCreatePrepayment(t, client, origin, token, key, quantity)
	confirmed := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/orders/confirm", token, "pc-confirm-"+key, map[string]any{"prepayment_id": prepared.prepaymentID}, http.StatusOK)
	prepared.id = acceptanceString(t, confirmed, "order_id")
	view := acceptanceHTTP(t, client, http.MethodGet, origin+"/api/v1/orders/"+prepared.id, token, "", nil, http.StatusOK)
	order := acceptanceObject(t, view, "order")
	prepared.orderNo = acceptanceString(t, order, "order_no")
	prepared.pickupNumber = acceptanceString(t, order, "pickup_number")
	return prepared
}

func pcClosureCreatePrepayment(t *testing.T, client *http.Client, origin, token, key string, quantity int) pcClosureOrder {
	t.Helper()
	quoteView := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/quotes", token, "pc-quote-"+key, map[string]any{
		"contact_name": "PC验收顾客", "pickup_date": "2026-08-23", "pickup_time": "11:30", "order_note": key,
		"items": []map[string]any{{"product_id": "1", "quantity": quantity, "flavors": []string{"少饭"}, "note": key}},
	}, http.StatusCreated)
	quoteID := acceptanceString(t, acceptanceObject(t, quoteView, "quote"), "id")
	prepayView := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/orders/prepay", token, "pc-prepay-"+key, map[string]any{"quote_id": quoteID}, http.StatusCreated)
	return pcClosureOrder{key: key, quoteID: quoteID, prepaymentID: acceptanceString(t, acceptanceObject(t, prepayView, "prepayment"), "id")}
}

func pcClosureOwnerSession(t *testing.T, client *http.Client, origin, ownerMiniToken string) string {
	t.Helper()
	login := acceptanceQRLogin(t, acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/auth/qrcode", "", "", map[string]any{}, http.StatusCreated))
	acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/me/admin-login/approve", ownerMiniToken, "", map[string]any{
		"login_id": login.loginID, "approval_secret": login.approvalSecret, "code": "pc-closure-owner-phone",
	}, http.StatusOK)
	poll := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/auth/poll", "", "", map[string]any{
		"login_id": login.loginID, "poll_secret": login.pollSecret,
	}, http.StatusOK)
	return acceptanceString(t, acceptanceObject(t, poll, "session"), "token")
}

type pcClosureFacts struct{ Orders, Items, Prepayments, Observations, Refunds, Sequences, Audits, Versions int64 }

func pcClosureReadFacts(t *testing.T, db *sql.DB) pcClosureFacts {
	t.Helper()
	var facts pcClosureFacts
	err := db.QueryRowContext(t.Context(), `SELECT
		(SELECT COUNT(*) FROM orders),(SELECT COUNT(*) FROM order_items),(SELECT COUNT(*) FROM prepayments),
		(SELECT COUNT(*) FROM payment_observations),(SELECT COUNT(*) FROM refunds),(SELECT COUNT(*) FROM pickup_sequences),
		(SELECT COUNT(*) FROM action_audits),
		(SELECT COALESCE(SUM(record_version),0) FROM orders)+(SELECT COALESCE(SUM(record_version),0) FROM prepayments)+(SELECT COALESCE(SUM(record_version),0) FROM payment_observations)+(SELECT COALESCE(SUM(record_version),0) FROM refunds)
	`).Scan(&facts.Orders, &facts.Items, &facts.Prepayments, &facts.Observations, &facts.Refunds, &facts.Sequences, &facts.Audits, &facts.Versions)
	if err != nil {
		t.Fatal("read PC closure transaction facts")
	}
	return facts
}

func pcClosureAssertCSV(t *testing.T, client *http.Client, target, token, orderNo string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal("build finance export request")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal("request finance export")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK || !strings.Contains(string(body), "订单号") || !strings.Contains(string(body), orderNo) {
		t.Fatalf("PC03 finance CSV invalid: status=%d err=%v body=%q", response.StatusCode, err, string(body))
	}
}

func pcClosurePaymentBillEntry(t *testing.T, db *sql.DB, prepaymentID string) billing.BillEntry {
	t.Helper()
	var entry billing.BillEntry
	entry.Kind, entry.Currency, entry.State = billing.EntryPayment, "CNY", "SUCCESS"
	if err := db.QueryRowContext(t.Context(), `SELECT CONVERT(p.out_trade_no USING utf8mb4),CONVERT(o.transaction_id USING utf8mb4),o.payable_cents,o.paid_at FROM prepayments p JOIN orders o ON o.prepayment_id=p.id WHERE p.id=?`, prepaymentID).
		Scan(&entry.OutTradeNo, &entry.ProviderID, &entry.AmountCents, &entry.OccurredAt); err != nil {
		t.Fatal("read PC03 system payment bill entry")
	}
	return entry
}

func pcClosureOrderAndSequenceCounts(t *testing.T, db *sql.DB) (int, int) {
	t.Helper()
	var orders, sequence int
	if err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM orders),(SELECT COALESCE(SUM(last_number),0) FROM pickup_sequences)`).Scan(&orders, &sequence); err != nil {
		t.Fatal(fmt.Sprintf("read PC04 order/sequence counts: %v", err))
	}
	return orders, sequence
}
