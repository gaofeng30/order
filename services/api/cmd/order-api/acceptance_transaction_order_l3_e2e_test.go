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

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
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

// TestTransactionOrderL3Server is a process-scoped private server for the
// Chrome rendered transaction/order selector. Test-only routes control only
// external provider outcomes, the clock and one explicit SQL failure trigger.
func TestTransactionOrderL3Server(t *testing.T) {
	if os.Getenv("ORDER_TRANSACTION_L3_SERVE") != "YES" {
		t.Skip("transaction/order L3 server not requested")
	}
	infoFile := os.Getenv("ORDER_TRANSACTION_L3_INFO_FILE")
	stopFile := os.Getenv("ORDER_TRANSACTION_L3_STOP_FILE")
	if !strings.HasPrefix(infoFile, "/private/tmp/order-transaction-l3-") ||
		!strings.HasPrefix(stopFile, "/private/tmp/order-transaction-l3-") {
		t.Fatal("transaction/order harness paths must be exact private/tmp files")
	}

	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	wallNow := time.Now().In(shanghai).Truncate(time.Minute)
	pickupAt := wallNow.Add(20 * time.Minute)
	if pickupAt.Format("2006-01-02") != wallNow.Format("2006-01-02") {
		t.Fatal("transaction/order harness requires a same-day pickup")
	}
	exactNow := pickupAt.Add(-30 * time.Minute)
	nearNow := pickupAt.Add(-29 * time.Minute)
	if _, err := db.ExecContext(t.Context(), `UPDATE service_dates SET is_open=(service_date=?)`, pickupAt.Format("2006-01-02")); err != nil {
		t.Fatal("open exact transaction/order service date")
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE meal_periods SET cutoff_time=?,pickup_start_time=?,pickup_end_time=?,interval_minutes=30 WHERE code='lunch'`,
		pickupAt.Format("15:04:00"), pickupAt.Format("15:04:00"), pickupAt.Format("15:04:00")); err != nil {
		t.Fatal("configure exact transaction/order pickup boundary")
	}
	clock := &transactionOrderL3Clock{value: nearNow}

	identityRepository := identity.NewRepository(db)
	loginProvider := transactionOrderL3LoginProvider{}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"transaction-order-user-openid":  acceptanceCustomerPhone,
		"transaction-order-owner-openid": acceptanceOwnerPhone,
	}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	miniAuth := miniRequestAuthenticator{sessions: sessions}
	pricing := staffdiscount.NewMySQLPricing(db)
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), clock.Now)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	paymentProvider := newTransactionOrderL3PaymentProvider(clock.Now)
	paymentApplication := paymentorder.NewMySQLApplication(db, quoteApplication, paymentProvider, paymentorder.Config{
		AppID: "wx-transaction-order-l3", MerchantID: "order-local-mch", Description: "预约点餐",
		PaymentNotifyURL: "http://127.0.0.1/api/v1/payments/wechat/notify",
		LeaseDuration:    time.Second, ReconcileInterval: time.Second,
	}, paymentorder.WithClock(clock.Now), paymentorder.WithLeaseOwnerSource(acceptanceLeaseOwner))
	notificationProvider := subscription.NewFakeProvider()
	notifications := subscription.New(db, notificationProvider)
	cipher, err := composeRedemptionTokenCipher("development", "")
	if err != nil {
		t.Fatal("compose transaction/order redemption cipher")
	}
	orders := orderquery.NewRepository(db, merchantService, cipher, clock.Now)
	fulfillmentApplication := fulfillment.NewMySQLApplication(db, merchantRepository, cipher, notifications)
	refundProvider := newLocalRefundProvider("order-local-mch", clock.Now)
	refundApplication := refund.New(db, refundProvider, "http://127.0.0.1/api/v1/refunds/wechat/notify").
		WithNotificationEnqueuer(newRefundSubscriptionAdapter(notifications))
	control := &transactionOrderL3Control{
		db: db, clock: clock, payment: paymentApplication, paymentProvider: paymentProvider,
		notifications: notifications, notificationProvider: notificationProvider,
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
		httpapi.NewSubscriptionHandler(sessions, notifications, 1),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders),
		newRefundRoutes(refund.NewHandler(sessions, refundApplication, orders, refundProvider)),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	var schema string
	if err := db.QueryRowContext(t.Context(), `SELECT DATABASE()`).Scan(&schema); err != nil || !strings.HasPrefix(schema, "order_acceptance_") {
		t.Fatal("read transaction/order private schema")
	}
	info := fmt.Sprintf("{\"origin\":%q,\"schema\":%q,\"pickup_date\":%q,\"pickup_time\":%q,\"exact_now\":%q,\"near_now\":%q,\"user_login_code\":\"transaction-user-login\",\"user_phone_code\":\"transaction-user-phone\",\"owner_login_code\":\"transaction-owner-login\",\"owner_phone_code\":\"transaction-owner-phone\"}\n",
		server.URL, schema, pickupAt.Format("2006-01-02"), pickupAt.Format("15:04"), exactNow.UTC().Format(time.RFC3339), nearNow.UTC().Format(time.RFC3339))
	if err := os.WriteFile(infoFile, []byte(info), 0o600); err != nil {
		t.Fatal("publish transaction/order harness info")
	}
	t.Cleanup(func() { _ = os.Remove(infoFile) })

	deadline := time.NewTimer(10 * time.Minute)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("transaction/order L3 harness timed out")
		case <-ticker.C:
			if _, err := os.Stat(stopFile); err == nil {
				_ = os.Remove(stopFile)
				return
			}
		}
	}
}

type transactionOrderL3LoginProvider struct{}

func (transactionOrderL3LoginProvider) Exchange(_ context.Context, code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("empty transaction/order login code")
	}
	if strings.Contains(code, "owner") {
		return "transaction-order-owner-openid", nil
	}
	return "transaction-order-user-openid", nil
}

type transactionOrderL3Clock struct {
	mu    sync.RWMutex
	value time.Time
}

func (clock *transactionOrderL3Clock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.value
}

func (clock *transactionOrderL3Clock) Set(value time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.value = value
}

type transactionOrderL3Control struct {
	db                   *sql.DB
	clock                *transactionOrderL3Clock
	payment              *paymentorder.Service
	paymentProvider      *transactionOrderL3PaymentProvider
	notifications        *subscription.Service
	notificationProvider *subscription.FakeProvider
}

func (control *transactionOrderL3Control) RegisterRoutes(engine *gin.Engine) {
	group := engine.Group("/api/v1/__acceptance/transaction-order")
	group.PUT("/clock", control.setClock)
	group.PUT("/apply-sql-failure", control.setApplySQLFailure)
	group.POST("/payment-worker", control.runPaymentWorker)
	group.POST("/notification-provider-failure", control.queueNotificationFailure)
	group.POST("/notification-worker", control.runNotificationWorker)
	group.GET("/facts", control.facts)
}

func (control *transactionOrderL3Control) setClock(ctx *gin.Context) {
	var input struct {
		Now string `json:"now"`
	}
	if !transactionOrderL3Decode(ctx, &input) {
		return
	}
	value, err := time.Parse(time.RFC3339, input.Now)
	if err != nil || value.Location() == time.Local {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_CLOCK"}})
		return
	}
	control.clock.Set(value)
	ctx.JSON(http.StatusOK, gin.H{"now": control.clock.Now().Format(time.RFC3339)})
}

func (control *transactionOrderL3Control) setApplySQLFailure(ctx *gin.Context) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if !transactionOrderL3Decode(ctx, &input) || input.Enabled == nil {
		if !ctx.IsAborted() {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_FAILURE_CONTROL"}})
		}
		return
	}
	if _, err := control.db.ExecContext(ctx.Request.Context(), `DROP TRIGGER IF EXISTS acceptance_fail_order_insert`); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FAILURE_CONTROL_UNAVAILABLE"}})
		return
	}
	if *input.Enabled {
		if _, err := control.db.ExecContext(ctx.Request.Context(), `CREATE TRIGGER acceptance_fail_order_insert BEFORE INSERT ON orders FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='acceptance apply failure'`); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FAILURE_CONTROL_UNAVAILABLE"}})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"apply_sql_failure": *input.Enabled})
}

func (control *transactionOrderL3Control) runPaymentWorker(ctx *gin.Context) {
	result, err := control.payment.RunDue(ctx.Request.Context(), control.clock.Now().UTC(), 20)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "PAYMENT_WORKER_UNAVAILABLE"}})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (control *transactionOrderL3Control) queueNotificationFailure(ctx *gin.Context) {
	control.notificationProvider.Queue(subscription.SendResult{}, &subscription.SendError{Code: "RATE_LIMITED", Permanent: false})
	ctx.JSON(http.StatusOK, gin.H{"notification_provider_failure": "RATE_LIMITED"})
}

func (control *transactionOrderL3Control) runNotificationWorker(ctx *gin.Context) {
	result, err := control.notifications.RunDue(ctx.Request.Context(), time.Now().UTC().Add(time.Minute), 20)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "NOTIFICATION_WORKER_UNAVAILABLE"}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"claimed": result.Claimed, "sent": result.Sent,
		"temporary_failed": result.TemporaryFailed, "permanent_failed": result.PermanentFailed,
	})
}

func (control *transactionOrderL3Control) facts(ctx *gin.Context) {
	view := gin.H{}
	if raw := ctx.Query("prepayment_id"); raw != "" {
		prepaymentID, ok := transactionOrderL3ID(raw)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_PREPAYMENT_ID"}})
			return
		}
		var outTradeNo, providerState, materializationState string
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT CONVERT(out_trade_no USING ascii),provider_state,materialization_state FROM prepayments WHERE id=?`, prepaymentID).
			Scan(&outTradeNo, &providerState, &materializationState); err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "PREPAYMENT_NOT_FOUND"}})
			return
		}
		var observations, queryObservations, callbackObservations, newObservations, orders int
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(source='ACTIVE_QUERY'),0),COALESCE(SUM(source='CALLBACK'),0),COALESCE(SUM(apply_state='NEW'),0) FROM payment_observations WHERE prepayment_id=?`, prepaymentID).
			Scan(&observations, &queryObservations, &callbackObservations, &newObservations); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
			return
		}
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT COUNT(*) FROM orders WHERE prepayment_id=?`, prepaymentID).Scan(&orders); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
			return
		}
		view["payment"] = gin.H{
			"prepayment_id": strconv.FormatUint(prepaymentID, 10), "provider_state": providerState,
			"materialization_state": materializationState,
			"provider_create_count": control.paymentProvider.CreateCount(outTradeNo),
			"provider_query_count":  control.paymentProvider.QueryCount(outTradeNo),
			"observation_count":     observations, "query_observation_count": queryObservations,
			"callback_observation_count": callbackObservations, "new_observation_count": newObservations,
			"order_count": orders,
		}
	}
	if raw := ctx.Query("order_id"); raw != "" {
		orderID, ok := transactionOrderL3ID(raw)
		if !ok {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ORDER_ID"}})
			return
		}
		var state string
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT state FROM orders WHERE id=?`, orderID).Scan(&state); err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "ORDER_NOT_FOUND"}})
			return
		}
		var accepted, rejected, outbox int
		if err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT COALESCE(SUM(decision='ACCEPTED'),0),COALESCE(SUM(decision='REJECTED'),0) FROM notification_consents WHERE order_id=? AND kind='READY'`, orderID).
			Scan(&accepted, &rejected); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
			return
		}
		var outboxState, lastError string
		err := control.db.QueryRowContext(ctx.Request.Context(), `SELECT state,COALESCE(last_error_code,'') FROM notification_outbox WHERE order_id=? AND kind='READY'`, orderID).
			Scan(&outboxState, &lastError)
		if err == nil {
			outbox = 1
		} else if err != sql.ErrNoRows {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "FACTS_UNAVAILABLE"}})
			return
		}
		view["order"] = gin.H{
			"order_id": strconv.FormatUint(orderID, 10), "state": state,
			"ready_accepted_count": accepted, "ready_rejected_count": rejected,
			"ready_outbox_count": outbox, "ready_outbox_state": outboxState, "ready_outbox_last_error": lastError,
		}
	}
	ctx.JSON(http.StatusOK, view)
}

type transactionOrderL3PaymentProvider struct {
	local   *localPaymentProvider
	mu      sync.Mutex
	queries map[string]int
}

func newTransactionOrderL3PaymentProvider(now func() time.Time) *transactionOrderL3PaymentProvider {
	return &transactionOrderL3PaymentProvider{
		local: newLocalPaymentProvider(now), queries: make(map[string]int),
	}
}

func (provider *transactionOrderL3PaymentProvider) CreateJSAPI(ctx context.Context, request paymentorder.ProviderCreateRequest) (paymentorder.ProviderCreateResult, error) {
	return provider.local.CreateJSAPI(ctx, request)
}

func (provider *transactionOrderL3PaymentProvider) QueryTransaction(ctx context.Context, outTradeNo string) (wechatpay.Transaction, error) {
	provider.mu.Lock()
	provider.queries[outTradeNo]++
	provider.mu.Unlock()
	return provider.local.QueryTransaction(ctx, outTradeNo)
}

func (provider *transactionOrderL3PaymentProvider) CloseTransaction(ctx context.Context, outTradeNo string) error {
	return provider.local.CloseTransaction(ctx, outTradeNo)
}

func (provider *transactionOrderL3PaymentProvider) ParsePaymentNotification(body []byte, headers wechatpay.SignatureHeaders) (paymentorder.VerifiedPayment, error) {
	return provider.local.ParsePaymentNotification(body, headers)
}

func (provider *transactionOrderL3PaymentProvider) CreateCount(outTradeNo string) uint64 {
	return provider.local.fake.CreateCount(outTradeNo)
}

func (provider *transactionOrderL3PaymentProvider) QueryCount(outTradeNo string) int {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.queries[outTradeNo]
}

func transactionOrderL3Decode(ctx *gin.Context, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(ctx.Request.Body, 4097))
	decoder.DisallowUnknownFields()
	if ctx.GetHeader("Content-Type") != "application/json" || decoder.Decode(target) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_CONTROL_REQUEST"}})
		ctx.Abort()
		return false
	}
	return true
}

func transactionOrderL3ID(raw string) (uint64, bool) {
	value, err := strconv.ParseUint(raw, 10, 64)
	return value, err == nil && value > 0 && strconv.FormatUint(value, 10) == raw
}
