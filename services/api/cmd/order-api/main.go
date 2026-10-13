package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/billing"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/importbatch"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/merchantsoldout"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/internal/objectstore"
	"github.com/gaofeng30/order/services/api/internal/orderadvance"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/storestatus"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/migrations/postgresql"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if os.Getenv("ORDER_ENV") == "uat" {
		return runUAT(logger)
	}
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "reason", config.Reason(err))
		return 1
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		logger.Error("database configuration error", "reason", database.Reason(err))
		return 1
	}
	defer db.Close()
	migrationSet, err := migrate.Load(postgresql.FS)
	if err != nil {
		logger.Error("migration assets invalid", "reason", migrate.Reason(err))
		return 1
	}
	readiness := func(ctx context.Context) httpapi.ReadinessResult {
		checkContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		state := migrate.Check(checkContext, db, migrationSet)
		return httpapi.ReadinessResult{Ready: state.Ready, Reason: state.Reason}
	}
	identityRepository := identity.NewRepository(db)
	loginProvider, phoneProvider, productionPhoneProvider, err := composeWeChatProviders(cfg.Environment, cfg.MiniProgram)
	if err != nil {
		logger.Error("wechat provider composition error")
		return 1
	}
	sessionService := identity.NewService(
		loginProvider,
		identityRepository,
	)
	identityHandler := identity.NewHandler(sessionService)
	phoneHandler := identity.NewPhoneHandler(
		sessionService,
		identity.NewPhoneService(phoneProvider, identityRepository),
	)
	requestAuthenticator := miniRequestAuthenticator{sessions: sessionService}
	pricingApplication := staffdiscount.NewPostgreSQLPricing(db)
	confirmationStore := objectstore.NewPostgreSQLConfirmationStore(db)
	var objectService *objectstore.Service
	var localObjects httpapi.RouteRegistrar
	if cfg.Environment.UsesCloudProviders() {
		material, materialErr := config.ParseCOSMaterial(
			os.Getenv("ORDER_COS_BUCKET"), os.Getenv("ORDER_TENCENT_REGION"),
		)
		if materialErr != nil {
			logger.Error("cos object store configuration error", "reason", config.Reason(materialErr))
			return 1
		}
		objectService, err = composeCloudObjectStore(context.Background(), cfg.Environment, material, confirmationStore)
		if err != nil {
			logger.Error("cos object store provider configuration error")
			return 1
		}
	} else {
		const localObjectRoot = "/private/tmp/order-local-objects"
		fileAdapter, fileAdapterErr := objectstore.NewFileAdapter(localObjectRoot, "/api/v1/objects")
		if fileAdapterErr != nil {
			logger.Error("local object store configuration error")
			return 1
		}
		localObjectRoutes, localObjectsErr := newLocalObjectRoutes(localObjectRoot)
		if localObjectsErr != nil {
			logger.Error("local object serving configuration error")
			return 1
		}
		objectService = objectstore.NewService(fileAdapter, confirmationStore)
		localObjects = localObjectRoutes
	}
	catalogOptions := []catalog.HandlerOption{catalog.WithAuthenticator(requestAuthenticator), catalog.WithPricing(pricingApplication)}
	menuOptions := []menu.HandlerOption{menu.WithAuthenticator(requestAuthenticator), menu.WithPricing(pricingApplication)}
	storefrontHandler := storefront.NewHandler(storefront.NewRepository(db))
	if objectService != nil {
		catalogOptions = append(catalogOptions, catalog.WithPublicURLs(objectService))
		menuOptions = append(menuOptions, menu.WithPublicURLs(objectService))
		storefrontHandler = storefront.NewHandler(storefront.NewRepository(db), objectService)
	}
	catalogHandler := catalog.NewHandler(catalog.NewRepository(db), catalogOptions...)
	menuHandler := menu.NewHandler(menu.NewRepository(db), time.Now, menuOptions...)
	quoteApplication := quote.NewProvider(db, audit.NewQuoteReceiptStore(db), time.Now)
	quoteHandler := quote.NewHandler(sessionService, quoteApplication)
	merchantIdentityRepository := merchantidentity.NewRepository(db)
	merchantIdentityService := merchantidentity.NewService(merchantIdentityRepository, phoneProvider)
	merchantIdentityHandler := merchantidentity.NewHandler(
		sessionService,
		merchantIdentityService,
	)
	registrars := []httpapi.RouteRegistrar{storefrontHandler, catalogHandler, menuHandler, identityHandler, phoneHandler, merchantIdentityHandler, quoteHandler}
	var paymentProvider paymentorder.PaymentProvider
	var paymentParser paymentorder.NotificationParser
	var paymentConfig paymentorder.Config
	var productionPaymentRuntime productionWeChatPayRuntime
	if cfg.Environment.UsesCloudProviders() {
		material, materialErr := config.LoadCloudWeChatPayMaterial(context.Background(), cfg.Environment, os.Getenv("ORDER_TENCENT_REGION"))
		if materialErr != nil {
			logger.Error("wechat payment configuration error", "reason", config.Reason(materialErr))
			return 1
		}
		runtime, providerErr := composeProductionWeChatPayRuntime(cfg.MiniProgram.AppID, material)
		if providerErr != nil {
			logger.Error("wechat payment provider configuration error")
			return 1
		}
		productionPaymentRuntime = runtime
		paymentProvider, paymentParser, paymentConfig = runtime.payment, runtime.payment, runtime.config
	} else {
		provider, providerErr := newConfiguredLocalPaymentProvider(os.LookupEnv, time.Now)
		if providerErr != nil {
			logger.Error("local payment provider configuration error")
			return 1
		}
		paymentProvider, paymentParser = provider, provider
		paymentConfig = paymentorder.Config{
			AppID: cfg.MiniProgram.AppID, MerchantID: "order-local-mch", Description: "预约点餐",
			PaymentNotifyURL: "http://127.0.0.1:8080/api/v1/payments/wechat/notify",
		}
	}
	paymentConfig.Environment = string(cfg.Environment)
	paymentApplication := paymentorder.NewPostgreSQLApplication(db, quoteApplication, paymentProvider, paymentConfig)
	registrars = append(registrars, newPaymentRoutes(paymentorder.NewHandler(sessionService, paymentApplication, paymentParser)))
	var notificationService *subscription.Service
	var subscriptionTemplateVersion uint64
	if !cfg.Environment.UsesCloudProviders() {
		notificationService = subscription.New(db, subscription.NewFakeProvider())
		subscriptionTemplateVersion = 1
	} else {
		providerConfig, providerConfigErr := loadWeChatSubscriptionProviderConfig(os.LookupEnv)
		if providerConfigErr != nil {
			logger.Error("wechat subscription configuration error")
			return 1
		}
		providerHTTPClient := &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		provider, providerErr := newProductionWeChatSubscriptionProvider(providerHTTPClient, productionPhoneProvider, identityRepository, providerConfig)
		if providerErr != nil {
			logger.Error("wechat subscription provider configuration error")
			return 1
		}
		notificationService = subscription.New(db, provider)
		subscriptionTemplateVersion = providerConfig.templateConfigVersion
	}
	registrars = append(registrars, httpapi.NewSubscriptionHandler(sessionService, notificationService, subscriptionTemplateVersion))
	tokenCipher, err := composeRedemptionTokenCipher(cfg.Environment, os.Getenv("ORDER_TENCENT_REGION"))
	if err != nil {
		logger.Error("redemption token configuration error")
		return 1
	}
	orderRepository := orderquery.NewRepository(db, merchantIdentityService, tokenCipher, time.Now)
	fulfillmentApplication := fulfillment.NewPostgreSQLApplication(db, merchantIdentityRepository, tokenCipher, notificationService)
	storeStatusCore := storestatus.New(db, merchantIdentityRepository, func() time.Time { return time.Now().UTC() })
	soldOutCommander := merchantsoldout.New(db, merchantIdentityRepository, time.Now)
	registrars = append(registrars,
		orderquery.NewHandler(sessionService, orderRepository),
		fulfillment.NewHandler(sessionService, fulfillmentApplication, orderRepository,
			fulfillment.WithStoreStatus(storeStatusCore), fulfillment.WithSoldOut(soldOutCommander)),
	)
	var refundProvider refund.Provider
	var refundParser refund.NotificationParser
	var refundNotifyURL string
	if cfg.Environment.UsesCloudProviders() {
		refundProvider, err = refund.NewWeChatProvider(productionPaymentRuntime.client, paymentConfig.MerchantID)
		if err == nil {
			refundParser, err = refund.NewWeChatNotificationParser(productionPaymentRuntime.client)
		}
		if err == nil {
			refundNotifyURL, err = productionRefundNotifyURL(paymentConfig.PaymentNotifyURL)
		}
		if err != nil {
			logger.Error("wechat refund provider configuration error")
			return 1
		}
	} else {
		provider := newLocalRefundProvider(paymentConfig.MerchantID, time.Now)
		refundProvider, refundParser = provider, provider
		refundNotifyURL = "http://127.0.0.1:8080/api/v1/refunds/wechat/notify"
	}
	refundConfig := refund.Config{
		Environment:      paymentConfig.Environment,
		AppID:            paymentConfig.AppID,
		MerchantID:       paymentConfig.MerchantID,
		PaymentNotifyURL: paymentConfig.PaymentNotifyURL,
		RefundNotifyURL:  refundNotifyURL,
	}
	refundService := refund.New(db, refundProvider, refundConfig).WithNotificationEnqueuer(newRefundSubscriptionAdapter(notificationService))
	registrars = append(registrars, newRefundRoutes(refund.NewHandler(sessionService, refundService, orderRepository, refundParser)))
	var billProvider billing.BillProvider
	if cfg.Environment.UsesCloudProviders() {
		billProvider, err = billing.NewWeChatBillProvider(productionPaymentRuntime.client)
		if err != nil {
			logger.Error("wechat billing provider configuration error")
			return 1
		}
	} else {
		billProvider = billing.NewFakeBillProvider()
	}
	billingConfig := billing.Config{
		Environment:      paymentConfig.Environment,
		AppID:            paymentConfig.AppID,
		MerchantID:       paymentConfig.MerchantID,
		PaymentNotifyURL: paymentConfig.PaymentNotifyURL,
		RefundNotifyURL:  refundNotifyURL,
	}
	billingService := billing.New(db, billProvider, billingConfig)
	merchantAdminApplication := merchantidentity.NewPostgreSQLAdminApplication(db, merchantIdentityService)
	merchantAdminHandler := merchantidentity.NewAdminHandler(merchantAdminApplication)
	importHandler := importbatch.NewHandler(importbatch.NewPostgreSQLApplication(db, objectService))
	adminOrderReader := adminreport.NewPostgreSQLApplication(db, nil)
	adminCommands := newAdminCommandAdapter(paymentApplication, refundService, adminOrderReader)
	adminFeatureRoutes := []adminGroupRegistrar{
		catalog.NewAdminHandler(catalog.NewPostgreSQLAdminApplication(db, objectService)),
		storefront.NewAdminHandler(storefront.NewPostgreSQLAdminApplication(db, objectService)),
		staffdiscount.NewHandler(staffdiscount.NewPostgreSQLApplication(db)),
		importHandler,
		adminreport.NewHandler(adminreport.NewPostgreSQLApplication(db, adminCommands)),
		audit.NewHandler(audit.NewPostgreSQLSearcher(db)),
	}
	ownerFeatureRoutes := []adminGroupRegistrar{objectstore.NewHandler(objectService)}
	registrars = append(registrars,
		newAdminRoutes(sessionService, merchantAdminApplication, merchantAdminHandler, adminFeatureRoutes, ownerFeatureRoutes, importHandler),
		localObjects,
	)
	orderProductionService := orderadvance.New(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workers := []func(context.Context){
		func(ctx context.Context) { runRefundWorker(ctx, refundService, logger) },
		func(ctx context.Context) { runOrderProductionWorker(ctx, orderProductionService, logger) },
	}
	if notificationService != nil {
		workers = append(workers, func(ctx context.Context) { runSubscriptionWorker(ctx, notificationService, logger) })
	}
	if paymentApplication != nil {
		workers = append(workers, func(ctx context.Context) { runPaymentWorker(ctx, paymentApplication, logger) })
	}
	if cfg.Environment.UsesCloudProviders() {
		workers = append(workers, func(ctx context.Context) { runBillingWorker(ctx, billingService, logger) })
	}

	if err := runWithWorkers(ctx, cfg, httpapi.NewRouter(logger, readiness, registrars...), logger, net.Listen, workers...); err != nil {
		logger.Error("order-api stopped with error", "error", err)
		return 1
	}
	return 0
}
