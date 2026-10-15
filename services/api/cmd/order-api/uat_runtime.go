package main

import (
	"context"
	"errors"
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
	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/importbatch"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/merchantsoldout"
	"github.com/gaofeng30/order/services/api/internal/objectstore"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/paymentorder"
	"github.com/gaofeng30/order/services/api/internal/quote"
	"github.com/gaofeng30/order/services/api/internal/refund"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"github.com/gaofeng30/order/services/api/internal/storestatus"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	"github.com/gaofeng30/order/services/api/sharedpg"
)

type cloudRunRuntime struct {
	handler http.Handler
	workers []func(context.Context)
}

func runCloudRun(logger *slog.Logger) int {
	cfg, err := config.LoadCloudRunRuntime()
	if err != nil {
		logger.Error("uat runtime configuration error", "reason", config.Reason(err))
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtime, err := composeCloudRun(ctx, cfg, logger)
	if err != nil {
		logger.Error("uat runtime composition error", "reason", err.Error())
		return 1
	}
	if err := runWithWorkers(ctx, cfg.HTTP, runtime.handler, logger, net.Listen, runtime.workers...); err != nil {
		logger.Error("order-api stopped with error", "error", err)
		return 1
	}
	return 0
}

func composeCloudRun(ctx context.Context, cfg config.CloudRunRuntimeConfig, logger *slog.Logger) (cloudRunRuntime, error) {
	client, err := cloudbasepg.New(cfg.EnvID, cfg.SharedPG)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_shared_pg_configuration_invalid")
	}
	loginProvider, phoneProvider, phoneClient, err := composeWeChatProviders(cfg.HTTP.Environment, cfg.HTTP.MiniProgram)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_wechat_configuration_invalid")
	}
	material, err := config.LoadCloudWeChatPayMaterial(ctx, cfg.HTTP.Environment, cfg.COS.Region())
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_payment_configuration_invalid")
	}
	payments, err := composeProductionWeChatPayRuntime(cfg.HTTP.MiniProgram.AppID, material)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_payment_provider_invalid")
	}
	payments.config.Environment = string(cfg.HTTP.Environment)
	subscriptionConfig, err := loadWeChatSubscriptionProviderConfig(os.LookupEnv)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_subscription_configuration_invalid")
	}
	if validateCloudRunBinding(cfg, payments.config.PaymentNotifyURL, subscriptionConfig.miniProgramState) != nil {
		return cloudRunRuntime{}, errors.New("cloudrun_callback_or_notification_environment_invalid")
	}
	tokenCipher, err := composeRedemptionTokenCipher(cfg.HTTP.Environment, cfg.COS.Region())
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_redemption_configuration_invalid")
	}
	cosProvider, err := config.NewCloudRunCOSCredentialProvider(cfg.COS, cfg.COSSourceSecretID, cfg.COSSourceSecretKey)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_cos_configuration_invalid")
	}
	credentialContext, cancel := context.WithTimeout(ctx, cosCredentialLoadTimeout)
	credential, err := loadCOSCredential(credentialContext, cosProvider)
	cancel()
	if err != nil || credential == nil {
		return cloudRunRuntime{}, errors.New("uat_cos_credentials_unavailable")
	}
	secretID, secretKey, token := credential.GetCredential()
	if secretID == "" || secretKey == "" || token == "" {
		return cloudRunRuntime{}, errors.New("uat_cos_credentials_unavailable")
	}
	cosAdapter, err := objectstore.NewCOSAdapter(objectstore.COSConfig{Bucket: cfg.COS.Bucket(), Region: cfg.COS.Region()}, credential)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_cos_provider_invalid")
	}
	objects := objectstore.NewSharedPGVersionedService(cosAdapter, client)
	if cfg.HTTP.Environment == config.Production {
		objects = objectstore.NewSharedPGImmutableService(cosAdapter, client)
	}
	identityRepository := identity.NewSharedPGRepository(client)
	sessions := identity.NewService(loginProvider, identityRepository)
	authenticator := miniRequestAuthenticator{sessions: sessions}
	merchantStore := merchantidentity.NewSharedPGStore(client)
	merchantService := merchantidentity.NewService(merchantStore, phoneProvider)
	merchantAdmin := merchantidentity.NewSharedPGAdminApplication(client, merchantService)
	pricing := staffdiscount.NewSharedPGPricing(client)
	quoteApplication := quote.NewSharedPGApplication(client, time.Now)
	paymentApplication := paymentorder.NewSharedPGApplication(client, payments.payment, payments.config)
	providerHTTPClient := &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	notificationProvider := subscription.NewWeChatProvider(
		providerHTTPClient, phoneClient, identityRecipientResolver{repository: identityRepository},
		staticWeChatSubscriptionTemplateResolver{config: subscriptionConfig},
		subscription.WeChatProviderConfig{MiniProgramState: subscriptionConfig.miniProgramState, Language: subscriptionConfig.language},
	)
	notifications := subscription.NewSharedPG(client, notificationProvider)
	orders := orderquery.NewSharedPGRepository(client, tokenCipher, time.Now)
	fulfillmentApplication := fulfillment.NewSharedPGApplication(client, tokenCipher)
	storeStatus := storestatus.NewSharedPG(client, func() time.Time { return time.Now().UTC() })
	soldOut := merchantsoldout.NewSharedPGCommander(client, time.Now)
	refundProvider, err := refund.NewWeChatProvider(payments.client, payments.config.MerchantID)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_refund_provider_invalid")
	}
	refundParser, err := refund.NewWeChatNotificationParser(payments.client)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_refund_parser_invalid")
	}
	refundNotifyURL, err := productionRefundNotifyURL(payments.config.PaymentNotifyURL)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_refund_notify_url_invalid")
	}
	refundConfig := refund.Config{
		Environment:      payments.config.Environment,
		AppID:            payments.config.AppID,
		MerchantID:       payments.config.MerchantID,
		PaymentNotifyURL: payments.config.PaymentNotifyURL,
		RefundNotifyURL:  refundNotifyURL,
	}
	refunds := refund.NewSharedPG(client, refundProvider, refundConfig, notifications)
	billProvider, err := billing.NewWeChatBillProvider(payments.client)
	if err != nil {
		return cloudRunRuntime{}, errors.New("uat_billing_provider_invalid")
	}
	billingConfig := billing.Config{
		Environment:      payments.config.Environment,
		AppID:            payments.config.AppID,
		MerchantID:       payments.config.MerchantID,
		PaymentNotifyURL: payments.config.PaymentNotifyURL,
		RefundNotifyURL:  refundNotifyURL,
	}
	bills := billing.NewSharedPG(client, billProvider, billingConfig)
	adminReader := adminreport.NewSharedPGApplication(client, nil)
	commands := newAdminCommandAdapter(paymentApplication, refunds, adminReader)
	imports := importbatch.NewHandler(importbatch.NewSharedPGApplication(client, objects))
	adminRoutes := []adminGroupRegistrar{
		catalog.NewAdminHandler(catalog.NewSharedPGAdminApplication(client, objects, objects)),
		storefront.NewAdminHandler(storefront.NewSharedPGAdminApplication(client, objects)),
		staffdiscount.NewHandler(staffdiscount.NewSharedPGApplication(client)), imports,
		adminreport.NewHandler(adminreport.NewSharedPGApplication(client, commands)),
		fulfillment.NewAdminHandler(fulfillmentApplication, orders),
		audit.NewHandler(audit.NewSharedPGSearcher(client)),
	}
	registrars := []httpapi.RouteRegistrar{
		storefront.NewHandler(storefront.NewSharedPGRepository(client), objects),
		catalog.NewHandler(catalog.NewSharedPGRepository(client), catalog.WithAuthenticator(authenticator), catalog.WithPricing(pricing), catalog.WithPublicURLs(objects)),
		menu.NewHandler(menu.NewSharedPGRepository(client), time.Now, menu.WithAuthenticator(authenticator), menu.WithPricing(pricing), menu.WithPublicURLs(objects)),
		identity.NewHandler(sessions), identity.NewPhoneHandler(sessions, identity.NewPhoneService(phoneProvider, identityRepository)),
		merchantidentity.NewHandler(sessions, merchantService), quote.NewHandler(sessions, quoteApplication),
		newPaymentRoutes(paymentorder.NewHandler(sessions, paymentApplication, payments.payment)),
		httpapi.NewSubscriptionHandler(sessions, notifications, subscriptionConfig.templateConfigVersion),
		orderquery.NewHandler(sessions, orders),
		fulfillment.NewHandler(sessions, fulfillmentApplication, orders, fulfillment.WithStoreStatus(storeStatus), fulfillment.WithSoldOut(soldOut)),
		newRefundRoutes(refund.NewHandler(sessions, refunds, orders, refundParser)),
		newAdminRoutes(sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin), adminRoutes,
			[]adminGroupRegistrar{objectstore.NewHandler(objects)}, imports),
	}
	if cfg.HTTP.Environment == config.Production {
		registrars = append(registrars, newCompensationRoutes(client, paymentApplication, refunds, notifications))
	}
	ready := func(ctx context.Context) httpapi.ReadinessResult {
		checkContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if sharedpg.Check(checkContext, client, cfg.SharedPG.ServiceUID, payments.config.Environment, payments.config.PaymentNotifyURL, refundNotifyURL) != nil {
			return httpapi.ReadinessResult{Reason: "database_incompatible"}
		}
		return httpapi.ReadinessResult{Ready: true}
	}
	workers := []func(context.Context){
		func(ctx context.Context) { runBillingWorker(ctx, bills, logger) },
	}
	if cfg.HTTP.Environment == config.Production {
		workers = append(workers, client.MaintainAuthentication)
	}
	if cfg.HTTP.Environment == config.UAT {
		workers = append(workers,
			func(ctx context.Context) { runRefundWorker(ctx, refunds, logger) },
			func(ctx context.Context) { runSubscriptionWorker(ctx, notifications, logger) },
			func(ctx context.Context) { runPaymentWorker(ctx, paymentApplication, logger) })
	}
	return cloudRunRuntime{handler: httpapi.NewRouter(logger, ready, registrars...), workers: workers}, nil
}

func validateCloudRunBinding(cfg config.CloudRunRuntimeConfig, paymentURL, notificationState string) error {
	expectedState := "trial"
	if cfg.HTTP.Environment == config.Production {
		expectedState = "formal"
	} else if cfg.HTTP.Environment != config.UAT {
		return errors.New("cloudrun_environment_invalid")
	}
	if cfg.PublicOrigin == "" || paymentURL != cfg.PublicOrigin+"/api/v1/payments/wechat/notify" || notificationState != expectedState {
		return errors.New("cloudrun_binding_invalid")
	}
	return nil
}
