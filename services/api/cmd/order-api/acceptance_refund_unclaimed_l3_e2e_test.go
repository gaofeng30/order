package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
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
	"github.com/gaofeng30/order/services/api/internal/wechatpay"
	"github.com/gin-gonic/gin"
)

// TestRefundUnclaimedL3Server is a process-scoped private server for the
// Chrome-rendered refund/unclaimed selector. It owns one random fresh-v44
// schema. Its controls change only the test clock and fake-provider outcomes.
func TestRefundUnclaimedL3Server(t *testing.T) {
	if os.Getenv("ORDER_REFUND_UNCLAIMED_L3_SERVE") != "YES" {
		t.Skip("refund/unclaimed L3 server not requested")
	}
	infoFile := os.Getenv("ORDER_REFUND_UNCLAIMED_L3_INFO_FILE")
	stopFile := os.Getenv("ORDER_REFUND_UNCLAIMED_L3_STOP_FILE")
	if !strings.HasPrefix(infoFile, "/private/tmp/order-refund-unclaimed-l3-") ||
		!strings.HasPrefix(stopFile, "/private/tmp/order-refund-unclaimed-l3-") {
		t.Fatal("refund/unclaimed harness paths must be exact private/tmp files")
	}

	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	pastDate := "2026-08-23"
	currentDate := "2026-08-25"
	for _, day := range []string{pastDate, currentDate} {
		if _, err := db.ExecContext(t.Context(), `INSERT INTO service_dates(service_date,is_open,record_version,updated_by_account_id,updated_at) SELECT ?,TRUE,1,id,NOW(6) FROM merchant_accounts WHERE role='OWNER' ORDER BY id LIMIT 1 ON DUPLICATE KEY UPDATE is_open=TRUE,record_version=service_dates.record_version+1,updated_at=VALUES(updated_at)`, day); err != nil {
			t.Fatalf("open refund/unclaimed service date %s: %v", day, err)
		}
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE meal_periods SET cutoff_time='11:30:00',pickup_start_time='11:30:00',pickup_end_time='11:30:00',interval_minutes=30 WHERE code='lunch'`); err != nil {
		t.Fatal("configure refund/unclaimed pickup")
	}
	clock := &refundUnclaimedClock{value: time.Date(2026, 8, 23, 8, 0, 0, 0, shanghai)}

	identityRepository := identity.NewRepository(db)
	loginProvider := refundUnclaimedLoginProvider{}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"refund-unclaimed-user-openid":  acceptanceCustomerPhone,
		"refund-unclaimed-owner-openid": acceptanceOwnerPhone,
	}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	pricing := staffdiscount.NewMySQLPricing(db)
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), clock.Now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	paymentProvider := newLocalPaymentProvider(clock.Now)
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-refund-unclaimed-l3", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1/api/v1/payments/wechat/notify",
	}, paymentorder.WithClock(clock.Now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	notifications := subscription.New(db, subscription.NewFakeProvider())
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatal("compose refund/unclaimed redemption cipher")
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, clock.Now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)
	refundProvider := newRefundUnclaimedProvider()
	refundApplication := refund.New(db, refundProvider, "http://127.0.0.1/api/v1/refunds/wechat/notify")
	adminReader := adminreport.NewMySQLApplication(db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, refundApplication, adminReader)
	adminReports := adminreport.NewMySQLApplication(db, adminCommands)
	control := &refundUnclaimedControl{
		db: db, clock: clock, production: orderadvance.New(db), refunds: refundApplication,
		provider: refundProvider, reports: adminReports, pastDate: pastDate,
	}

	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		control,
		storefront.NewHandler(storefront.NewRepository(db)),
		menu.NewHandler(menu.NewRepository(db), clock.Now, menu.WithAuthenticator(miniAuth), menu.WithPricing(pricing)),
		identity.NewHandler(sessions), identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		quote.NewHandler(sessions, quoteApplication),
		newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, paymentProvider)),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newRefundRoutes(refund.NewHandler(sessions, refundApplication, orders, refundProvider)),
		newAdminRoutes(
			sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin),
			[]adminGroupRegistrar{
				storefront.NewAdminHandler(storefront.NewMySQLAdminApplication(db)),
				adminreport.NewHandler(adminReports),
			}, nil, nil,
		),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	var schema string
	if err := db.QueryRowContext(t.Context(), `SELECT DATABASE()`).Scan(&schema); err != nil || !strings.HasPrefix(schema, "order_acceptance_") {
		t.Fatal("read refund/unclaimed private schema")
	}
	info := fmt.Sprintf("{\"origin\":%q,\"schema\":%q,\"past_date\":%q,\"current_date\":%q,\"pickup_time\":\"11:30\",\"past_clock\":%q,\"current_clock\":%q,\"user_login_code\":\"refund-user-login\",\"user_phone_code\":\"refund-user-phone\",\"owner_login_code\":\"refund-owner-login\",\"owner_phone_code\":\"refund-owner-phone\"}\n",
		server.URL, schema, pastDate, currentDate,
		time.Date(2026, 8, 23, 8, 0, 0, 0, shanghai).UTC().Format(time.RFC3339),
		time.Date(2026, 8, 24, 8, 0, 0, 0, shanghai).UTC().Format(time.RFC3339))
	if err := os.WriteFile(infoFile, []byte(info), 0o600); err != nil {
		t.Fatal("publish refund/unclaimed harness info")
	}
	t.Cleanup(func() { _ = os.Remove(infoFile) })

	deadline := time.NewTimer(10 * time.Minute)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("refund/unclaimed L3 harness timed out")
		case <-ticker.C:
			if _, err := os.Stat(stopFile); err == nil {
				_ = os.Remove(stopFile)
				return
			}
		}
	}
}

type refundUnclaimedLoginProvider struct{}

func (refundUnclaimedLoginProvider) Exchange(_ context.Context, code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("empty refund/unclaimed login code")
	}
	if strings.Contains(code, "owner") {
		return "refund-unclaimed-owner-openid", nil
	}
	return "refund-unclaimed-user-openid", nil
}

type refundUnclaimedClock struct {
	mu    sync.RWMutex
	value time.Time
}

func (clock *refundUnclaimedClock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.value
}

func (clock *refundUnclaimedClock) Set(value time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.value = value
}

type refundUnclaimedProvider struct {
	fake    *refund.FakeProvider
	mu      sync.Mutex
	mode    string
	creates map[string]uint64
	queries map[string]uint64
}

func newRefundUnclaimedProvider() *refundUnclaimedProvider {
	return &refundUnclaimedProvider{
		fake: refund.NewFakeProvider("order-local-mch"), mode: "PROCESSING",
		creates: make(map[string]uint64), queries: make(map[string]uint64),
	}
}

func (provider *refundUnclaimedProvider) CreateRefund(ctx context.Context, request refund.ProviderCreateRequest) (refund.ProviderRefund, error) {
	provider.mu.Lock()
	provider.creates[request.OutRefundNo]++
	provider.mu.Unlock()
	return provider.fake.CreateRefund(ctx, request)
}

func (provider *refundUnclaimedProvider) QueryRefund(ctx context.Context, outRefundNo string) (refund.ProviderRefund, error) {
	provider.mu.Lock()
	provider.queries[outRefundNo]++
	mode := provider.mode
	provider.mu.Unlock()
	if mode == "UNKNOWN" {
		return refund.ProviderRefund{}, refund.ErrUnavailable
	}
	if mode == "SUCCESS" {
		if err := provider.fake.MarkSuccess(outRefundNo, time.Now().UTC()); err != nil {
			return refund.ProviderRefund{}, err
		}
	}
	return provider.fake.QueryRefund(ctx, outRefundNo)
}

func (provider *refundUnclaimedProvider) ParseRefundNotification(body []byte, headers wechatpay.SignatureHeaders) (refund.VerifiedRefund, error) {
	return provider.fake.ParseRefundNotification(body, headers)
}

func (provider *refundUnclaimedProvider) SetMode(mode string) bool {
	if mode != "UNKNOWN" && mode != "PROCESSING" && mode != "SUCCESS" {
		return false
	}
	provider.mu.Lock()
	provider.mode = mode
	provider.mu.Unlock()
	return true
}

func (provider *refundUnclaimedProvider) Counts(outRefundNo string) (uint64, uint64) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.creates[outRefundNo], provider.queries[outRefundNo]
}

type refundUnclaimedControl struct {
	db         *sql.DB
	clock      *refundUnclaimedClock
	production *orderadvance.Service
	refunds    *refund.Service
	provider   *refundUnclaimedProvider
	reports    *adminreport.MySQLApplication
	pastDate   string
	mu         sync.Mutex
	workerTick uint32
}

func (control *refundUnclaimedControl) RegisterRoutes(engine *gin.Engine) {
	group := engine.Group("/api/v1/__acceptance/refund-unclaimed")
	group.PUT("/clock", control.setClock)
	group.POST("/production-worker", control.runProductionWorker)
	group.PUT("/provider-mode", control.setProviderMode)
	group.POST("/refund-worker", control.runRefundWorker)
	group.GET("/facts", control.facts)
}

func (control *refundUnclaimedControl) setClock(ctx *gin.Context) {
	var input struct {
		Now string `json:"now"`
	}
	if !refundUnclaimedDecode(ctx, &input) {
		return
	}
	value, err := time.Parse(time.RFC3339, input.Now)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_CLOCK"}})
		return
	}
	control.clock.Set(value)
	ctx.JSON(http.StatusOK, gin.H{"now": control.clock.Now().UTC().Format(time.RFC3339)})
}

func (control *refundUnclaimedControl) runProductionWorker(ctx *gin.Context) {
	result, err := control.production.RunProductionDue(ctx.Request.Context(), control.clock.Now().UTC(), 20)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "PRODUCTION_WORKER_UNAVAILABLE"}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"scanned": result.Scanned, "advanced": result.Advanced})
}

func (control *refundUnclaimedControl) setProviderMode(ctx *gin.Context) {
	var input struct {
		Mode string `json:"mode"`
	}
	if !refundUnclaimedDecode(ctx, &input) {
		return
	}
	if !control.provider.SetMode(input.Mode) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PROVIDER_MODE"}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"mode": input.Mode})
}

func (control *refundUnclaimedControl) runRefundWorker(ctx *gin.Context) {
	control.mu.Lock()
	control.workerTick++
	tick := control.workerTick
	control.mu.Unlock()
	workerAt := time.Now().UTC().Add(24*time.Hour + time.Duration(tick)*2*time.Minute)
	result, err := control.refunds.RunDue(ctx.Request.Context(), workerAt, 20)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "REFUND_WORKER_UNAVAILABLE"}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"claimed": result.Claimed, "observed": result.Observed, "pending": result.Pending, "applied": result.Applied})
}

func (control *refundUnclaimedControl) facts(ctx *gin.Context) {
	view := gin.H{}
	if raw := ctx.Query("order_id"); raw != "" {
		orderID, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || orderID == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ORDER_ID"}})
			return
		}
		var state string
		var amount uint64
		var cipherPresent, soldOut bool
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT o.state,o.payable_cents,o.redemption_token_ciphertext IS NOT NULL,EXISTS(SELECT 1 FROM product_sold_out_dates s WHERE s.service_date=o.pickup_date AND s.product_id=oi.product_id) FROM orders o JOIN order_items oi ON oi.order_id=o.id WHERE o.id=? LIMIT 1`, orderID).Scan(&state, &amount, &cipherPresent, &soldOut); err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "ORDER_NOT_FOUND"}})
			return
		}
		order := gin.H{"id": raw, "state": state, "payable_cents": amount, "redemption_cipher_present": cipherPresent, "product_sold_out": soldOut}
		var refundID uint64
		var outRefundNo, providerState, materializationState string
		err = control.db.QueryRowContext(ctx.Request.Context(), `SELECT id,CONVERT(out_refund_no USING utf8mb4),provider_state,materialization_state FROM refunds WHERE order_id=?`, orderID).Scan(&refundID, &outRefundNo, &providerState, &materializationState)
		if err == nil {
			var observations, applied int
			if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(apply_state='APPLIED'),0) FROM refund_observations WHERE refund_id=?`, refundID).Scan(&observations, &applied); err != nil {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
				return
			}
			creates, queries := control.provider.Counts(outRefundNo)
			order["refund"] = gin.H{
				"id": strconv.FormatUint(refundID, 10), "out_refund_no": outRefundNo,
				"provider_state": providerState, "materialization_state": materializationState,
				"provider_create_count": creates, "provider_query_count": queries,
				"observation_count": observations, "applied_observation_count": applied,
			}
		} else if err != sql.ErrNoRows {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
			return
		}
		view["order"] = order
	}
	var ownerUserID uint64
	if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT bound_user_id FROM merchant_accounts WHERE role='OWNER' AND enabled=TRUE AND deleted_at IS NULL`).Scan(&ownerUserID); err == nil && ownerUserID > 0 {
		at, _ := time.ParseInLocation("2006-01-02 15:04", control.pastDate+" 16:00", time.FixedZone("Asia/Shanghai", 8*60*60))
		if stats, err := control.reports.Stats(ctx.Request.Context(), ownerUserID, at); err == nil {
			productSales := uint32(0)
			for _, item := range stats.ProductSales {
				productSales += item.Quantity
			}
			view["historical_stats"] = gin.H{
				"today_revenue_cents": stats.TodayRevenueCents, "today_orders": stats.TodayOrders,
				"month_revenue_cents": stats.MonthRevenueCents, "month_orders": stats.MonthOrders,
				"product_sales": productSales, "refund_cents": stats.RefundCents,
			}
		}
	}
	var inventoryTables int
	_ = control.db.QueryRowContext(ctx.Request.Context(), `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND (table_name LIKE '%inventory%' OR table_name LIKE '%stock%')`).Scan(&inventoryTables)
	view["inventory_table_count"] = inventoryTables
	ctx.JSON(http.StatusOK, view)
}

func refundUnclaimedDecode(ctx *gin.Context, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 4097))
	decoder.DisallowUnknownFields()
	if ctx.GetHeader("Content-Type") != "application/json" || decoder.Decode(target) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_CONTROL_REQUEST"}})
		ctx.Abort()
		return false
	}
	return true
}

var _ refund.Provider = (*refundUnclaimedProvider)(nil)
var _ refund.NotificationParser = (*refundUnclaimedProvider)(nil)
