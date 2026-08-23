const go = (packagePath, test) => ({ kind: 'go-test', package: packagePath, test });
const node = file => ({ kind: 'node-test', file, name_pattern: '.*' });
const command = (...argv) => ({ kind: 'command', argv });

export const profiles = Object.freeze({
  mini_entry_ui0: evidence('UI0', 'Mini entry/home Node harness covers client fail-closed behavior only; it is not rendered UI1.',
    node('apps/wechat-miniprogram/tests/overnight-entry-home-ui0.test.js')),
  mini_menu_ui0: evidence('UI0', 'Mini menu/detail Node harness covers client presentation and zero-side-effect shields only.',
    node('apps/wechat-miniprogram/tests/overnight-menu-detail-ui0.test.js')),
  mini_checkout_ui0: evidence('UI0', 'Mini checkout Node harness covers client Quote/prepay/confirm sequencing only.',
    node('apps/wechat-miniprogram/tests/overnight-checkout-ui0.test.js')),
  mini_orders_ui0: evidence('UI0', 'Mini order/profile Node harness covers client DTO rendering and guards only.',
    node('apps/wechat-miniprogram/tests/overnight-user-orders-profile-ui0.test.js')),
  mini_merchant_ui0: evidence('UI0', 'Merchant Mini Node harness covers four page adapters only.',
    node('apps/wechat-miniprogram/tests/overnight-merchant-ui0.test.js')),
  mini_public_ui0: evidence('UI0', 'Public read Node harness covers menu pricing and fail-closed client consumption only.',
    node('apps/wechat-miniprogram/tests/public-read-menu-ui0.test.js')),
  mini_ui1_fixture: evidence('UI1', 'Chromium simulator renders three anonymous/menu interactions against a loopback fixture; it is not MySQL/fake-payment E2E.',
    command('npm', 'run', 'ui1', '--prefix', 'tools/miniprogram-ui')),
  mini_composed_ui1_l3: evidence('L3', 'Four rendered Mini Chromium scenarios use the root-composed HTTP API and MySQL for anonymous launch/menu/cart, trusted phone checkout, local payment/order list/detail/refund, a storefront HTTP shield, and a confirm HTTP shield that preserves the cart and suppresses success navigation. This is supporting evidence only: it does not cover phone refusal, subscription consent, payment cancel/unknown, cutoff/sold-out boundaries, or real WeChat UI3.',
    command('npm', '--prefix', 'tools/miniprogram-ui', 'run', 'ui1:composed')),
  mini_composed_pending_ui1_l3: evidence('L3', 'One rendered Mini pending-payment scenario uses the root-composed API in explicit local NOTPAY mode: two Confirm calls return 202, the UI stays pending, preserves the cart, suppresses result navigation, avoids a second requestPayment call and rotates the Confirm idempotency key. This is supporting evidence only: it does not directly assert the durable observation/prepayment rows, absence of an order row, Query reconciliation, or real WeChat behavior.',
    command('env', 'ORDER_COMPOSED_PAYMENT_EXPECTATION=pending', 'npm', '--prefix', 'tools/miniprogram-ui', 'run', 'ui1:composed')),
  mini_composed_merchant_ui1_l3: evidence('L3', 'One rendered Merchant Mini Chrome scenario uses the root-composed HTTP API and MySQL to create a near-time PREPARING order, authenticate an OWNER, write and restore store status, advance PREPARING to READY, expose the user token only at READY, atomically scan it COMPLETED with an idempotency key, and toggle the selected date sold-out true then false. The integrated Red receipt records the former unkeyed scan returning a real 400; runner cleanup fails unless saved Admin settings and sold-out baseline are restored. This remains supporting evidence only: it does not cover all five lanes/search, manual and cross-date code redemption, replay/refunded rejection, notification failure, tomorrow isolation, or real WeChat payment/camera UI3.',
    command('env', 'ORDER_COMPOSED_FLOW=merchant', 'npm', '--prefix', 'tools/miniprogram-ui', 'run', 'ui1:composed')),
  mini_composed_user_boundaries_ui1_l3: evidence('L3', 'One locked-Chrome rendered Mini selector drives the real WXML controls against the root-composed HTTP API and MySQL for BE-01--06 and BE-22--26. It closes closed/cutoff browsing, sold-out and off-shelf cart revalidation, meal mismatch, cutoff and current-fact drift, byte-exact staff identity, visitor pricing, and empty-cart shields. BE-22 and BE-26 remain supporting projections because their receipt explicitly substitutes an unbound primary-phone response and filters unrelated READY rows inside the browser seam.',
    command('node', 'tools/miniprogram-ui/run-ui1-composed-boundaries.mjs')),
  web_contract_ui0: evidence('UI0', 'PC Admin contract suite proves twelve-page wiring and no browser truth, not rendered full-flow E2E.',
    node('apps/web-admin/tests/http-contract.test.js')),
  web_chrome_ui1_fixture: evidence('UI1', 'PC Chrome smoke renders login/nav against a local fixture, not the composed MySQL API.',
    command('node', 'apps/web-admin/tests/ui1-chrome-runner.js')),
  web_composed_reads_ui1_l3: evidence('L3', 'Twenty-seven PC Chrome checks acquire a root-composed OWNER QR session, render dashboard facts, visit eleven routes and require every observed server GET, including all three finance endpoints, to return 200. This is supporting read evidence only: successful reads do not prove mutations or each matrix row failure shield.',
    command('node', 'apps/web-admin/tests/composed-ui1-chrome-runner.mjs')),
  web_composed_writes_ui1_l3: evidence('L3', 'Twenty-one PC Chrome checks drive category, product, staff, discount, settings and merchant-account UI writes, then read the composed server facts back; category deletion also asserts a redacted audit receipt. This is supporting CRUD evidence only: ordering, upload/image, invalid-input, FK, role/session, cross-client and other row-specific failure shields are not all asserted.',
    command('node', 'apps/web-admin/tests/composed-ui1-writes-chrome-runner.mjs')),
  web_composed_imports_ui1_l3: evidence('L3', 'Eleven PC Chrome checks drive non-xlsx rejection, product preview/commit with one generated category, staff duplicate-phone isolation, preview-token replay/conflict, visible server readback, masked PII and durable import audits. This is supporting import evidence only: template download, header and size/row limits, existing-name/disabled-row behavior and every batch failure shield are not all asserted.',
    command('node', 'apps/web-admin/tests/composed-ui1-imports-chrome-runner.mjs')),
  web_composed_catalog_image_ui1_l3: evidence('L3', 'Twenty-six PC Chrome checks use a root-composed OWNER session and server readback for category create/rename/enable/order/delete, normalized duplicate and FK 409 shields, product create/delete/order/shelf/date-sold-out, zero/one/three ordered images with public object reads, and launch PNG upload/geometry/enable/new-context projection/clear/audit/cleanup. This closes PAGE-PC06 and BE-20 exactly. It is supporting-only for PAGE-PC05, PAGE-PC08 and AC-17 because product-edit/upload-failure, unreadable-object rendering, and rendered Mini cross-client coverage remain absent.',
    command('node', 'apps/web-admin/tests/composed-ui1-catalog-image-chrome-runner.mjs')),

  composed_order_refund_l2: evidence('L2', 'One root-composed HTTP and worker selector uses a fresh v1-v44 MySQL schema plus deterministic local WeChat/payment/refund providers to prove trusted phone and staff Quote pricing, confirmed-payment order materialization, exact 30-minute production, READY token and scan redemption, SUBACCOUNT versus OWNER PC QR authorization, durable full refund, audit receipts and two sent notification outbox intents. It is not UI1 and does not claim unexercised boundary variants.',
    go('./services/api/cmd/order-api', 'TestAcceptanceLocalThreeRoleOrderToRefund')),
  composed_import_boundaries_l2: evidence('L2', 'One root-composed OWNER PC multipart selector uses a fresh v1-v44 MySQL schema to prove non-xlsx and missing-header zero writes, 10 MiB and 500/5000 row limits, existing-product isolation, one ordered enabled category for two products, duplicate-phone first-row ownership, exact replay versus idempotency conflict, three committed batches and six unique import audit receipts.',
    go('./services/api/cmd/order-api', 'TestAcceptanceImportBoundariesAreDurable')),
  composed_user_boundaries_l2: evidence('L2', 'One root-composed HTTP selector uses a fresh v1-v44 MySQL schema and deterministic local providers to prove BE-01--06 and BE-22--26 server facts and failure shields. It exactly closes the required L2 rows for meal mismatch, Quote current-fact drift, and byte-exact extra-phone plus name staff identity; the remaining rows use it as supporting server evidence only.',
    go('./services/api/cmd/order-api', 'TestAcceptanceUserBoundariesAreFailClosed')),

  schema_no_inventory_l1: evidence('L1', 'Frozen migration ledger rejects PRD-out-of-scope inventory/member/coupon/summary tables.',
    go('./services/api/migrations', 'TestFrozenV18ToV44LedgerContracts')),
  menu_rules_l1: multi('L1', 'Pure menu rules enforce Shanghai discrete points, fixed cutoff and invalid-schedule fail closed.', [
    go('./services/api/internal/menu', 'TestMealConfigurationSelectsConfiguredDiscretePointsAndCutoff'),
    go('./services/api/internal/menu', 'TestMenuExactOrLateCutoffIsUnavailable'),
  ]),
  menu_mysql_l2: evidence('L2', 'Fresh MySQL plus HTTP exercises today/tomorrow pickup options, cutoff boundaries, meal filtering and date-scoped sold-out facts.',
    go('./services/api/internal/menu', 'TestMenuMySQLIntegration')),
  catalog_mysql_l2: evidence('L2', 'Fresh MySQL plus HTTP exercises visible catalog/detail selection and fail-closed database recovery.',
    go('./services/api/internal/catalog', 'TestCatalogRepositoryAndHTTPIntegration')),
  storefront_mysql_l2_legacy: evidence('L2', 'Fresh MySQL plus HTTP covers storefront singleton/read failure, but its launch-layer assertions are legacy v11 and cannot close R2 launch acceptance.',
    go('./services/api/internal/storefront', 'TestStorefrontMySQL8Integration')),
  identity_mysql_l2: multi('L2', 'Fresh MySQL covers session, trusted primary-phone binding/status and HTTP projections.', [
    go('./services/api/internal/identity', 'TestMiniprogramSessionMySQLIntegration'),
    go('./services/api/internal/identity', 'TestMiniprogramPhoneMySQLIntegration'),
    go('./services/api/internal/identity', 'TestMiniprogramPhoneStatusMySQLIntegration'),
  ]),
  merchant_mysql_l2: evidence('L2', 'Fresh MySQL covers live merchant binding, OWNER/SUBACCOUNT authorization, receipts and last-owner protection.',
    go('./services/api/internal/merchantidentity', 'TestMerchantIdentityMySQL8Integration')),
  store_status_mysql_l2: multi('L2', 'Fresh MySQL covers OWNER/SUBACCOUNT store-status commands, replay, audit and concurrency.', [
    go('./services/api/internal/storestatus', 'TestApplyOwnerChangesOnlyBusinessStatus'),
    go('./services/api/internal/storestatus', 'TestApplySubaccountChangesOnlyBusinessStatus'),
    go('./services/api/internal/storestatus', 'TestApplyConcurrentSameKeyConvergesToOneResultAndAudit'),
  ]),
  soldout_mysql_l2: evidence('L2', 'Fresh MySQL covers OWNER/SUBACCOUNT date-scoped sold-out toggle, tomorrow isolation, replay and authorization.',
    go('./services/api/internal/merchantsoldout', 'TestMySQL8MerchantSoldOutVerticalSlice')),
  quote_pricing_l1: multi('L1', 'Pure pricing rules enforce per-unit half-up, integer cents, 1..100 rate, payable floor and overflow shields.', [
    go('./services/api/internal/quotepricing', 'TestCalculateRoundsUnitHalfUpBeforeQuantity'),
    go('./services/api/internal/quotepricing', 'TestCalculateRoundsEachUnitBeforeSumming'),
    go('./services/api/internal/quote', 'TestCreateRejectsSubCentPaymentAmountWithoutWritingQuote'),
    go('./services/api/internal/quote', 'TestStoredQuoteValidationRejectsArithmeticOverflow'),
  ]),
  quote_mysql_l2: evidence('L2', 'Fresh MySQL covers primary-phone snapshot, extra-phone/name identity, immutable pricing, current-fact drift, cutoff, digest and atomic replay.',
    go('./services/api/internal/quote', 'TestQuoteMySQL8Integration')),
  payment_rules_l1: multi('L1', 'Pure payment rules cover effective deadline, one-minute Create threshold, trusted success-time boundary and monotonic observations.', [
    go('./services/api/internal/paymentorder', 'TestEffectiveDeadlineUsesEarlierQuoteWindowOrPickup'),
    go('./services/api/internal/paymentorder', 'TestCreateEligibilityHasExactOneMinuteBoundary'),
    go('./services/api/internal/paymentorder', 'TestMaterializationModeUsesTrustedSuccessTimeOnly'),
    go('./services/api/internal/paymentorder', 'TestProviderStateNeverRegressesAcrossOutOfOrderFacts'),
  ]),
  payment_mysql_l2: multi('L2', 'Fresh MySQL covers one durable prepayment, confirmed-payment materialization, callback-first durability, replay, deadline/manual shield and corrupt snapshot shield.', [
    go('./services/api/internal/paymentorder', 'TestPaymentOrderMySQL8PaidMaterializationIsAtomicAndConcurrentSafe'),
    go('./services/api/internal/paymentorder', 'TestPaymentOrderMySQL8CallbackIsDurableBeforeWorkerMaterialization'),
    go('./services/api/internal/paymentorder', 'TestPaymentOrderMySQL8ExactDeadlinePaymentStaysManualWithoutNumber'),
    go('./services/api/internal/paymentorder', 'TestPaymentOrderMySQL8CorruptSnapshotDefersDurablePayment'),
  ]),
  production_rules_l1: multi('L1', 'Pure six-state production policy covers exact 30-minute boundary, missed ticks and no successor regression.', [
    go('./services/api/internal/orderproduction', 'TestStateVocabulary'),
    go('./services/api/internal/orderproduction', 'TestInitialStateLessThanThirtyMinutesBeforePickup'),
    go('./services/api/internal/orderproduction', 'TestAdvanceAtThresholdStartsPreparing'),
    go('./services/api/internal/orderproduction', 'TestAdvanceAfterMissedThresholdStartsPreparing'),
    go('./services/api/internal/orderproduction', 'TestAdvanceDoesNotMoveSuccessorStates'),
  ]),
  production_mysql_l2: evidence('L2', 'Fresh MySQL worker advances exact-boundary and missed RESERVED orders once and writes system evidence.',
    go('./services/api/internal/orderadvance', 'TestRunProductionDueMySQL8AdvancesBoundaryAndLateOrdersOnce')),
  fulfillment_mysql_l2: evidence('L2', 'Fresh MySQL covers READY token issuance, encrypted token query, scan/code/direct redemption, cross-date shield, replay and audit.',
    go('./services/api/internal/fulfillment', 'TestFulfillmentMySQLVerticalSlice')),
  refund_rules_l1: multi('L1', 'Pure refund rules enforce full amount, provider-fact validation, monotonicity and accepted-not-final semantics.', [
    go('./services/api/internal/refund', 'TestFakeProviderIsDeterministicAndFullAmountOnly'),
    go('./services/api/internal/refund', 'TestRefundProviderStateNeverRegressesAcrossOutOfOrderFacts'),
    go('./services/api/internal/refund', 'TestCreateSuccessRemainsQueryableUntilDurableObservation'),
  ]),
  refund_mysql_l2: multi('L2', 'Fresh MySQL covers full refund request/replay, REFUNDING to confirmed REFUNDED, durable callback and exact self-cancel boundary.', [
    go('./services/api/internal/refund', 'TestRefundMySQL8FullAmountRequestQueryAndObservation'),
    go('./services/api/internal/refund', 'TestRefundMySQL8SelfCancellationBoundaryAndOwnerPaidPrepayment'),
  ]),
  subscription_l1: multi('L1', 'Deterministic store/provider tests cover accepted/rejected consent, one outbox intent, provider failure classification and order-independent delivery.', [
    go('./services/api/internal/subscription', 'TestEnqueueInTxWithoutAcceptedConsentIsNoOp'),
    go('./services/api/internal/subscription', 'TestRunDueCommitsLeaseBeforeProviderAndMarksSentWithCAS'),
    go('./services/api/internal/subscription', 'TestRunDueClassifiesProviderFailuresWithoutPersistingRawError'),
  ]),
  billing_l1: multi('L1', 'Deterministic bill comparison covers stable digest/order and explicit provider/system single sides.', [
    go('./services/api/internal/billing', 'TestFakeBillProviderIsStableAndSupportsExplicitSingleSides'),
    go('./services/api/internal/billing', 'TestCompareBillReportsProviderAndSystemSingleSidesWithoutInventingRows'),
  ]),
  import_l1: multi('L1', 'Server-side XLSX parser covers first-sheet ownership and configured row-limit fail closed.', [
    go('./services/api/internal/importbatch', 'TestParseRowsReadsFirstWorksheetWithoutClientXLSXTruth'),
    go('./services/api/internal/importbatch', 'TestParseRowsFailsClosedOverLimit'),
  ]),
  order_query_l1: multi('L1', 'Order query adapter covers owner hiding, fixed filters, phone masking and server-derived actions.', [
    go('./services/api/internal/orderquery', 'TestUserDetailHidesOwnerMismatchAsNotFound'),
    go('./services/api/internal/orderquery', 'TestMaskPhoneNeverReturnsUnmaskedPII'),
    go('./services/api/internal/orderquery', 'TestActionsAreDerivedWithoutAdvancingState'),
  ]),
  admin_pending_l1: multi('L1', 'PC pending mutation adapter covers typed materialize/refund projections and idempotency shield.', [
    go('./services/api/internal/adminreport', 'TestPendingMutationProjectsTypedNestedResponse'),
    go('./services/api/cmd/order-api', 'TestAdminCommandAdapterRefundsOrderAndPaidPrepaymentWithoutIDConfusion'),
  ]),
  object_store_l1: multi('L1', 'Object-store adapters cover official COS request mapping and invalid key/provider fail closed.', [
    go('./services/api/internal/objectstore', 'TestCOSAdapterPutUsesOfficialCOSRequest'),
    go('./services/api/internal/objectstore', 'TestCOSAdapterFailsClosedForInvalidKeyAndProviderError'),
  ]),
});

const caseProfiles = new Map();

assign(['PAGE-U01'], ['mini_entry_ui0', 'mini_ui1_fixture']);
assign(['PAGE-U02'], ['mini_entry_ui0']);
assign(['PAGE-U03'], ['mini_menu_ui0', 'mini_public_ui0', 'menu_mysql_l2']);
assign(['PAGE-U04'], ['mini_menu_ui0', 'catalog_mysql_l2']);
assign(['PAGE-U05'], ['mini_checkout_ui0', 'identity_mysql_l2', 'quote_mysql_l2']);
assign(['PAGE-U06'], ['mini_checkout_ui0', 'payment_mysql_l2', 'subscription_l1']);
assign(['PAGE-U07'], ['mini_orders_ui0', 'fulfillment_mysql_l2', 'refund_mysql_l2', 'subscription_l1']);
assign(['PAGE-U08'], ['mini_orders_ui0', 'order_query_l1']);
assign(['PAGE-U09'], ['mini_orders_ui0', 'identity_mysql_l2', 'merchant_mysql_l2']);
assign(['PAGE-M02'], ['mini_merchant_ui0', 'order_query_l1', 'store_status_mysql_l2']);
assign(['PAGE-M03'], ['mini_merchant_ui0', 'fulfillment_mysql_l2', 'subscription_l1']);
assign(['PAGE-M04'], ['mini_merchant_ui0', 'fulfillment_mysql_l2']);
assign(['PAGE-M05'], ['mini_merchant_ui0', 'soldout_mysql_l2']);
assign(pagePCRange(), ['web_contract_ui0', 'web_chrome_ui1_fixture']);
assign(['PAGE-PC01'], ['billing_l1', 'order_query_l1']);
assign(['PAGE-PC02'], ['refund_mysql_l2', 'order_query_l1']);
assign(['PAGE-PC03'], ['billing_l1']);
assign(['PAGE-PC04'], ['admin_pending_l1', 'payment_mysql_l2']);
assign(['PAGE-PC05'], ['catalog_mysql_l2', 'object_store_l1']);
assign(['PAGE-PC06'], ['catalog_mysql_l2']);
assign(['PAGE-PC07'], ['storefront_mysql_l2_legacy', 'menu_mysql_l2']);
assign(['PAGE-PC08'], ['storefront_mysql_l2_legacy', 'object_store_l1']);
assign(['PAGE-PC09'], ['quote_mysql_l2']);
assign(['PAGE-PC10'], ['merchant_mysql_l2']);
assign(['PAGE-PC11', 'PAGE-PC12'], ['import_l1']);

// These root-composed rendered selectors provide materially stronger UI
// evidence than UI0/fixture smoke. They remain supporting-only because no
// selector below asserts every behavior and failure shield in its matrix row.
assign(['PAGE-U01', 'PAGE-U03', 'PAGE-U05', 'PAGE-U06', 'PAGE-U07'], ['mini_composed_ui1_l3']);
assign(['AC-01', 'AC-06', 'AC-14'], ['mini_composed_ui1_l3']);
assign(['BE-12', 'BE-22'], ['mini_composed_ui1_l3']);
assign(['INV-05', 'INV-09', 'INV-10', 'INV-11', 'INV-16'], ['mini_composed_ui1_l3']);

assign(['PAGE-U06'], ['mini_composed_pending_ui1_l3']);
assign(['AC-06'], ['mini_composed_pending_ui1_l3']);
assign(['BE-07', 'BE-08'], ['mini_composed_pending_ui1_l3']);
assign(['INV-05', 'INV-16'], ['mini_composed_pending_ui1_l3']);

assign(['PAGE-U07', 'PAGE-M02', 'PAGE-M03', 'PAGE-M04', 'PAGE-M05'], ['mini_composed_merchant_ui1_l3']);
assign(['AC-09', 'AC-10', 'AC-11', 'AC-13', 'AC-16'], ['mini_composed_merchant_ui1_l3']);
assign(['BE-10', 'BE-15', 'BE-20'], ['mini_composed_merchant_ui1_l3']);
assign(['INV-05', 'INV-07', 'INV-08', 'INV-13', 'INV-16'], ['mini_composed_merchant_ui1_l3']);

assign(['BE-01', 'BE-02', 'BE-03', 'BE-04', 'BE-05', 'BE-06', 'BE-23', 'BE-24', 'BE-25'], [['mini_composed_user_boundaries_ui1_l3', true]]);
assign(['BE-22', 'BE-26'], ['mini_composed_user_boundaries_ui1_l3']);

assign(['PAGE-PC01', 'PAGE-PC02', 'PAGE-PC03', 'PAGE-PC05', 'PAGE-PC06', 'PAGE-PC07', 'PAGE-PC08', 'PAGE-PC09'], ['web_composed_reads_ui1_l3']);
assign(['PAGE-PC05', 'PAGE-PC06', 'PAGE-PC07', 'PAGE-PC09'], ['web_composed_writes_ui1_l3']);
assign(['PAGE-PC11', 'PAGE-PC12'], ['web_composed_imports_ui1_l3']);
assign(['BE-27', 'BE-31', 'BE-32', 'BE-33'], ['web_composed_imports_ui1_l3']);
assign(['PAGE-PC05', 'PAGE-PC08', 'AC-13', 'AC-17'], ['web_composed_catalog_image_ui1_l3']);
assign(['PAGE-PC06', 'BE-20'], [['web_composed_catalog_image_ui1_l3', true]]);

// This first composed release selector is useful supporting evidence for the
// broader rows below, but only the five exact L2 scenarios explicitly marked
// true are closed by what the selector actually asserts.
assign(['PAGE-U05', 'PAGE-U06', 'PAGE-U07', 'PAGE-M03', 'PAGE-M04', 'PAGE-PC02', 'PAGE-PC10'], ['composed_order_refund_l2']);

assign(['AC-01'], ['mini_entry_ui0', 'mini_ui1_fixture', 'identity_mysql_l2']);
assign(['AC-02'], [['quote_mysql_l2', true], 'mini_orders_ui0']);
assign(['AC-03'], [['menu_mysql_l2', true], 'mini_menu_ui0']);
assign(['AC-04'], ['menu_mysql_l2', 'mini_menu_ui0', 'mini_public_ui0']);
assign(['AC-05'], ['catalog_mysql_l2', 'quote_mysql_l2', 'mini_public_ui0']);
assign(['AC-06'], ['payment_mysql_l2', 'mini_checkout_ui0']);
assign(['AC-07'], ['payment_mysql_l2', 'admin_pending_l1']);
assign(['AC-08'], ['quote_mysql_l2', 'mini_checkout_ui0']);
assign(['AC-09'], ['production_rules_l1', 'production_mysql_l2']);
assign(['AC-10'], [['production_rules_l1', false], ['payment_mysql_l2', true], ['production_mysql_l2', true], ['fulfillment_mysql_l2', true], ['refund_mysql_l2', true]]);
assign(['AC-11'], ['fulfillment_mysql_l2', 'mini_orders_ui0']);
assign(['AC-12'], [['fulfillment_mysql_l2', true], 'mini_merchant_ui0']);
assign(['AC-13'], ['soldout_mysql_l2', 'mini_merchant_ui0']);
assign(['AC-14'], ['refund_rules_l1', 'refund_mysql_l2']);
assign(['AC-15'], ['subscription_l1', 'mini_orders_ui0']);
assign(['AC-16'], ['merchant_mysql_l2', 'web_contract_ui0']);
assign(['AC-17'], ['catalog_mysql_l2', 'storefront_mysql_l2_legacy', 'web_contract_ui0']);
assign(['AC-18'], ['order_query_l1', 'billing_l1']);
assign(['AC-19'], ['schema_no_inventory_l1']);
assign(['AC-05', 'AC-06', 'AC-09', 'AC-10', 'AC-11', 'AC-14', 'AC-15', 'AC-16'], ['composed_order_refund_l2']);

assign(['BE-01', 'BE-02'], ['menu_mysql_l2', 'mini_menu_ui0']);
assign(['BE-03'], ['catalog_mysql_l2', 'quote_mysql_l2', 'mini_menu_ui0']);
assign(['BE-04'], ['catalog_mysql_l2', 'quote_mysql_l2', ['composed_user_boundaries_l2', true]]);
assign(['BE-05'], ['quote_mysql_l2', 'mini_checkout_ui0']);
assign(['BE-06'], [['quote_mysql_l2', true], ['composed_user_boundaries_l2', true]]);
assign(['BE-07', 'BE-08', 'BE-09'], ['payment_mysql_l2', 'mini_checkout_ui0']);
assign(['BE-10'], ['payment_rules_l1', 'payment_mysql_l2']);
assign(['BE-11'], [['production_mysql_l2', true], ['composed_order_refund_l2', true]]);
assign(['BE-12', 'BE-13', 'BE-14'], ['refund_rules_l1', 'refund_mysql_l2']);
assign(['BE-15'], ['order_query_l1', 'mini_orders_ui0', ['composed_order_refund_l2', true]]);
assign(['BE-16'], [['fulfillment_mysql_l2', true], 'mini_merchant_ui0']);
assign(['BE-17'], ['fulfillment_mysql_l2']);
assign(['BE-18'], [['fulfillment_mysql_l2', true], 'mini_merchant_ui0']);
assign(['BE-19'], ['order_query_l1', 'refund_mysql_l2']);
assign(['BE-20'], ['soldout_mysql_l2', 'mini_merchant_ui0']);
assign(['BE-21'], ['subscription_l1', 'mini_orders_ui0']);
assign(['BE-22'], ['identity_mysql_l2', 'mini_checkout_ui0']);
assign(['BE-23'], ['quote_mysql_l2', ['composed_user_boundaries_l2', true]]);
assign(['BE-24'], ['quote_mysql_l2', 'mini_checkout_ui0']);
assign(['BE-25'], ['mini_checkout_ui0']);
assign(['BE-26'], ['order_query_l1', 'mini_orders_ui0']);
assign(['BE-01', 'BE-02', 'BE-03', 'BE-05', 'BE-22', 'BE-24', 'BE-25', 'BE-26'], ['composed_user_boundaries_l2']);
assign(['BE-27', 'BE-28', 'BE-29', 'BE-30', 'BE-31', 'BE-32', 'BE-33'], ['import_l1', 'web_contract_ui0']);
assign(['BE-27', 'BE-28', 'BE-29', 'BE-30', 'BE-31', 'BE-32', 'BE-33'], [['composed_import_boundaries_l2', true]]);
assign(['BE-34', 'BE-35'], ['catalog_mysql_l2', 'mini_menu_ui0']);

assign(['INV-01'], ['storefront_mysql_l2_legacy', 'menu_rules_l1']);
assign(['INV-02'], [['menu_rules_l1', true], ['menu_mysql_l2', true]]);
assign(['INV-03'], [['schema_no_inventory_l1', true], ['soldout_mysql_l2', true]]);
assign(['INV-04'], [['menu_rules_l1', true], ['menu_mysql_l2', true]]);
assign(['INV-05'], [['payment_mysql_l2', true], 'payment_rules_l1']);
assign(['INV-06'], [['payment_mysql_l2', true], 'admin_pending_l1']);
assign(['INV-07'], [['production_rules_l1', true], ['payment_mysql_l2', true], ['production_mysql_l2', true], ['fulfillment_mysql_l2', true], ['refund_mysql_l2', true]]);
assign(['INV-08'], [['production_rules_l1', true], ['production_mysql_l2', true]]);
assign(['INV-09'], [['production_rules_l1', true], ['refund_mysql_l2', true]]);
assign(['INV-10'], [['refund_rules_l1', true], 'refund_mysql_l2']);
assign(['INV-11'], ['identity_mysql_l2', 'quote_mysql_l2']);
assign(['INV-12'], [['quote_pricing_l1', true], ['quote_mysql_l2', true]]);
assign(['INV-13'], ['merchant_mysql_l2', 'web_contract_ui0']);
assign(['INV-14'], ['payment_mysql_l2', 'fulfillment_mysql_l2']);
assign(['INV-15'], ['subscription_l1']);
assign(['INV-16'], ['schema_no_inventory_l1', 'payment_mysql_l2', 'mini_ui1_fixture']);
assign(['INV-05', 'INV-07', 'INV-11', 'INV-15', 'INV-16'], ['composed_order_refund_l2']);
assign(['INV-08', 'INV-10', 'INV-13'], [['composed_order_refund_l2', true]]);

export function coverageFor(caseID) {
  return (caseProfiles.get(caseID) || []).map(([profileID, satisfies]) => ({
    profile_id: profileID,
    satisfies,
    ...structuredClone(profiles[profileID]),
  }));
}

function assign(caseIDs, requestedProfiles) {
  for (const caseID of caseIDs) {
    const existing = caseProfiles.get(caseID) || [];
    for (const requested of requestedProfiles) {
      const normalized = Array.isArray(requested) ? requested : [requested, false];
      if (!profiles[normalized[0]]) throw new Error(`unknown coverage profile ${normalized[0]}`);
      if (!existing.some(([profileID]) => profileID === normalized[0])) existing.push(normalized);
    }
    caseProfiles.set(caseID, existing);
  }
}

function evidence(level, claim, selector) {
  return { level, status: 'AVAILABLE', claim, selector };
}

function multi(level, claim, selectors) {
  return { level, status: 'AVAILABLE', claim, selectors };
}

function pagePCRange() {
  return Array.from({ length: 12 }, (_, index) => `PAGE-PC${String(index + 1).padStart(2, '0')}`);
}
