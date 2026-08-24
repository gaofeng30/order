/* global App, Behavior, Component, ORDER_TRANSACTION_L3_FIXTURE, ORDER_TRANSACTION_L3_PROXY_ORIGIN, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const orderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.wxml');
const resultTemplate = require('../../../../apps/wechat-miniprogram/pages/result/result.wxml');
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const qrcodeTemplate = require('../../../../apps/wechat-miniprogram/components/qrcode/qrcode.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const fixture = ORDER_TRANSACTION_L3_FIXTURE;
const proxyOrigin = ORDER_TRANSACTION_L3_PROXY_ORIGIN;
const pageDefinitions = {};
const paymentCalls = [];
const paymentDecisions = [];
const subscribeDecisions = [];
const requestObservations = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let lastNavigation = null;

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: 'transaction-user-login' })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  setClipboardData: options => queueMicrotask(() => options.success && options.success()),
  request: options => {
    const requestURL = new URL(options.url);
    fetch(requestURL.toString(), {
      method: options.method || 'GET',
      headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let data = {};
      if (raw) try { data = JSON.parse(raw); } catch {}
      requestObservations.push({ method: options.method || 'GET', path: requestURL.pathname, status: response.status });
      options.success({ statusCode: response.status, data });
    }).catch(error => options.fail(error));
  },
  requestPayment: options => {
    paymentCalls.push({
      timeStamp: options.timeStamp, nonceStr: options.nonceStr, package: options.package,
      signType: options.signType, paySign: options.paySign,
    });
    const decision = paymentDecisions.shift();
    queueMicrotask(() => {
      if (decision === 'cancel') options.fail({ errMsg: 'requestPayment:fail cancel' });
      else if (decision === 'fail') options.fail({ errMsg: 'requestPayment:fail system error' });
      else options.success({ errMsg: 'requestPayment:ok' });
    });
  },
  requestSubscribeMessage: options => {
    const decision = subscribeDecisions.shift();
    queueMicrotask(() => {
      if (!decision || decision === 'fail') options.fail({ errMsg: 'requestSubscribeMessage:fail' });
      else options.success({ [options.tmplIds[0]]: decision });
    });
  },
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
};

require('../../../../apps/wechat-miniprogram/components/icon/icon.js');
const iconDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/imageph/imageph.js');
const imagephDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/money/money.js');
const moneyDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/navbar/navbar.js');
const navbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/pill/pill.js');
const pillDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/qrcode/qrcode.js');
const qrcodeDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/stepper/stepper.js');
const stepperDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/customize/customize.js');
const customizeDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/app.js');
registeringPage = 'confirm';
require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.js');
registeringPage = 'result';
require('../../../../apps/wechat-miniprogram/pages/result/result.js');
registeringPage = 'home';
require('../../../../apps/wechat-miniprogram/pages/home/home.js');
registeringPage = 'order-detail';
require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/home/home' }];

describe('transaction and order L3 against private root HTTP and fresh MySQL', () => {
  it('closes payment retry, Query recovery, cancellation boundary and supplemental consent', async () => {
    app.onLaunch();
    await waitFor(() => app.globalData.session.state !== 'loading', () => `session stayed ${app.globalData.session.state}`);
    if (app.globalData.session.state !== 'ready') throw new Error(`session ended ${app.globalData.session.state}`);
    app.globalData.subscriptionTemplateIds = { READY: 'transaction-ready-template', REFUND_RESULT: 'transaction-refund-template' };

    paymentDecisions.push('cancel', 'success');
    const retryConfirm = await renderConfirm('cancel-retry');
    const firstPay = retryConfirm.querySelector('.pay-btn');
    firstPay.dispatchEvent('touchstart');
    firstPay.dispatchEvent('touchend');
    await waitFor(
      () => retryConfirm.instance.data.paymentState === 'error' || !!lastNavigation,
      () => `cancelled payment stayed ${retryConfirm.instance.data.paymentState}`,
    );
    if (lastNavigation || retryConfirm.instance.data.paymentState !== 'error'
      || Object.keys(app.globalData.cart).length !== 1) {
      throw new Error(`requestPayment:fail cancel became success ${lastNavigation || retryConfirm.instance.data.paymentState}`);
    }
    const retryPrepaymentID = exactID(retryConfirm.instance._prepayment?.id, 'retained retry prepayment');
    const afterCancel = await facts({ prepayment_id: retryPrepaymentID });
    assertPaymentFacts(afterCancel.payment, { creates: 1, queries: 0, observations: 0, orders: 0 }, 'cancel');
    if (requestObservations.some(item => item.path === '/api/v1/orders/confirm')) {
      throw new Error('cancelled native payment reached server confirmation');
    }

    retryConfirm.querySelector('.pay-btn').dispatchEvent('touchstart');
    retryConfirm.querySelector('.pay-btn').dispatchEvent('touchend');
    await waitFor(() => !!lastNavigation, () => `payment retry stayed ${retryConfirm.instance.data.paymentState}`);
    if (!lastNavigation.startsWith('/pages/result/result?id=') || Object.keys(app.globalData.cart).length !== 0) {
      throw new Error(`payment retry did not create one real order: ${lastNavigation || 'no navigation'}`);
    }
    const retryOrderID = exactID(new URL(`http://mini.local${lastNavigation}`).searchParams.get('id'), 'retry order id');
    const afterRetry = await facts({ prepayment_id: retryPrepaymentID });
    assertPaymentFacts(afterRetry.payment, { creates: 1, queries: 1, observations: 1, orders: 1 }, 'retry');
    if (afterRetry.payment.query_observation_count !== 1 || afterRetry.payment.callback_observation_count !== 0
      || paymentCalls.length !== 2 || paymentCalls[0].package !== paymentCalls[1].package) {
      throw new Error('payment retry reuses one provider Create and lost callback is recovered only by Query');
    }

    await control('PUT', '/apply-sql-failure', { enabled: true });
    paymentDecisions.push('success');
    const failedApply = await renderConfirm('apply-sql-failure');
    failedApply.querySelector('.pay-btn').dispatchEvent('touchstart');
    failedApply.querySelector('.pay-btn').dispatchEvent('touchend');
    await waitFor(() => failedApply.instance.data.paymentState === 'error', () => `apply failure stayed ${failedApply.instance.data.paymentState}`);
    const failedPrepaymentID = exactID(failedApply.instance._prepayment?.id, 'failed apply prepayment');
    const failedFacts = await facts({ prepayment_id: failedPrepaymentID });
    assertPaymentFacts(failedFacts.payment, { creates: 1, queries: 1, observations: 1, orders: 0 }, 'apply SQL failure');
    if (failedFacts.payment.new_observation_count !== 1 || failedFacts.payment.materialization_state !== 'READY'
      || lastNavigation || Object.keys(app.globalData.cart).length !== 1) {
      throw new Error('apply_sql_failure did not remain durable pending with zero fake success');
    }
    await control('PUT', '/apply-sql-failure', { enabled: false });

    const exactDetail = await renderDetail('exact-boundary', fixture.exact.id);
    const nearDetail = await renderDetail('near-boundary', fixture.near.id);
    if (fixture.exact.cancel_status !== 409 || fixture.near.cancel_status !== 409
      || exactDetail.instance.data.o.state !== 'PREPARING' || nearDetail.instance.data.o.state !== 'PREPARING'
      || exactDetail.instance.data.canCancel || nearDetail.instance.data.canCancel
      || exactDetail.querySelector('.cancel-btn.on') || nearDetail.querySelector('.cancel-btn.on')) {
      throw new Error('exact 30-minute cancellation stays unavailable and near-time cancellation stays fail closed');
    }

    subscribeDecisions.push('reject');
    const rejectedResult = renderResult('ready-rejected', retryOrderID);
    await waitFor(() => rejectedResult.instance.data.state !== 'loading', () => `rejected result stayed ${rejectedResult.instance.data.state}`);
    await waitForFacts(retryOrderID, view => view.ready_rejected_count === 1 && view.ready_accepted_count === 0);
    await merchantReady(retryOrderID, 'retry-ready');

    const home = renderPage({
      definition: pageDefinitions.home, template: homeTemplate, id: 'transaction-home',
      usingComponents: components('transaction-home'),
    });
    await waitFor(() => home.instance.data.ordersState !== 'loading', () => `home orders stayed ${home.instance.data.ordersState}`);
    if (!home.instance.data.ongoing?.ready || home.instance.data.ongoing.orderId !== retryOrderID
      || !home.querySelector('.supplement-ready')) {
      throw new Error('rejected READY consent has a supplemental entry on the rendered home page');
    }
    const supplementalDetail = await renderDetail('ready-supplement', retryOrderID);
    if (!supplementalDetail.instance.data.canSubscribeReady || !supplementalDetail.querySelector('.supplement-ready')) {
      throw new Error('rejected READY consent has a supplemental entry on the rendered order detail');
    }
    subscribeDecisions.push('accept');
    supplementalDetail.querySelector('.supplement-ready').dispatchEvent('touchstart');
    supplementalDetail.querySelector('.supplement-ready').dispatchEvent('touchend');
    const supplemented = await waitForFacts(retryOrderID, view => view.ready_rejected_count === 1 && view.ready_accepted_count === 1);
    if (supplemented.ready_outbox_count !== 0) throw new Error('post-READY supplemental consent invented an already-missed send');
    const supplementalWrites = requestObservations.filter(item => item.path === `/api/v1/orders/${retryOrderID}/subscriptions` && item.status === 200);
    if (supplementalWrites.length !== 2) throw new Error(`supplemental READY consent wrote ${supplementalWrites.length} times`);

    subscribeDecisions.push('accept');
    const providerFailureResult = renderResult('provider-failure', fixture.notification.id);
    await waitFor(() => providerFailureResult.instance.data.state !== 'loading', () => `provider failure result stayed ${providerFailureResult.instance.data.state}`);
    await waitForFacts(fixture.notification.id, view => view.ready_accepted_count === 1);
    await control('POST', '/notification-provider-failure');
    await merchantReady(fixture.notification.id, 'notification-ready');
    const worker = await control('POST', '/notification-worker');
    if (worker.claimed !== 1 || worker.temporary_failed !== 1 || worker.sent !== 0) {
      throw new Error(`notification_provider_failure worker diverged ${JSON.stringify(worker)}`);
    }
    const providerFailureFacts = (await facts({ order_id: fixture.notification.id })).order;
    if (providerFailureFacts.state !== 'READY_FOR_PICKUP' || providerFailureFacts.ready_outbox_count !== 1
      || providerFailureFacts.ready_outbox_state !== 'PENDING' || providerFailureFacts.ready_outbox_last_error !== 'RATE_LIMITED') {
      throw new Error(`notification provider failure changed the order: ${JSON.stringify(providerFailureFacts)}`);
    }
    const providerFailureDetail = await renderDetail('provider-failure-ready', fixture.notification.id);
    if (providerFailureDetail.instance.data.o.state !== 'READY_FOR_PICKUP' || !providerFailureDetail.instance.data.showQr) {
      throw new Error('notification provider failure removed the rendered READY order');
    }
  });
});

async function renderConfirm(suffix) {
  document.body.innerHTML = '';
  lastNavigation = null;
  app.globalData.pickup = { date: fixture.pickup_date, mealPeriod: 'lunch', time: fixture.pickup_time };
  app.globalData.cart = {
    [fixture.product.id]: { product: JSON.parse(JSON.stringify(fixture.product)), qty: 1, flavors: ['少饭'], note: suffix },
  };
  const page = renderPage({
    definition: pageDefinitions.confirm, template: confirmTemplate, id: `transaction-confirm-${suffix}`,
    usingComponents: components(`transaction-confirm-${suffix}`, true, false),
  });
  await waitFor(() => page.instance.data.phoneState !== 'loading', () => `confirm phone stayed ${page.instance.data.phoneState}`);
  if (page.instance.data.phoneState !== 'bound') throw new Error(`confirm phone ended ${page.instance.data.phoneState}`);
  pageDefinitions.confirm.onInput.call(page.instance, {
    currentTarget: { dataset: { k: 'contact' } }, detail: { value: '交易闭环用户' },
  });
  return page;
}

function renderResult(suffix, orderID) {
  document.body.innerHTML = '';
  return renderPage({
    definition: pageDefinitions.result, template: resultTemplate, id: `transaction-result-${suffix}`,
    usingComponents: components(`transaction-result-${suffix}`), loadOptions: { id: orderID },
  });
}

async function renderDetail(suffix, orderID) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pageDefinitions['order-detail'], template: orderDetailTemplate, id: `transaction-detail-${suffix}`,
    usingComponents: components(`transaction-detail-${suffix}`, false, true), loadOptions: { id: orderID },
  });
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `detail ${orderID} stayed loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`detail ${orderID} ended ${page.instance.data.detailState}`);
  return page;
}

function components(suffix, includeCustomize = false, includeOrder = false) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const result = {
    icon,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
  };
  if (includeCustomize || includeOrder) {
    result.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  }
  if (includeCustomize) {
    const imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
    const stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
    result.customize = registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, imageph, money: result.money, stepper },
    });
  }
  if (includeOrder) {
    result.pill = registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' });
    result.qrcode = registerComponent({ definition: qrcodeDefinition, template: qrcodeTemplate, id: `qrcode-${suffix}`, tagName: 'qrcode' });
  }
  return result;
}

async function merchantReady(orderID, suffix) {
  const view = await direct('/api/v1/merchant/orders/' + orderID + '/ready', {
    method: 'POST', token: fixture.owner_token, key: `transaction-${suffix}-${crypto.randomUUID()}`, body: {}, expected: 200,
  });
  if (view.order?.state !== 'READY_FOR_PICKUP') throw new Error(`merchant ready ${orderID} ended ${view.order?.state}`);
}

async function facts(query) {
  const params = new URLSearchParams(query);
  return direct(`/api/v1/__acceptance/transaction-order/facts?${params.toString()}`, { expected: 200 });
}

async function waitForFacts(orderID, predicate) {
  const deadline = Date.now() + 3000;
  let view = {};
  while (Date.now() < deadline) {
    view = (await facts({ order_id: orderID })).order;
    if (predicate(view)) return view;
    await simulate.sleep(20);
  }
  throw new Error(`order facts did not converge ${JSON.stringify(view)}`);
}

function control(method, suffix, body) {
  return direct(`/api/v1/__acceptance/transaction-order${suffix}`, { method, body, expected: 200 });
}

async function direct(pathname, options = {}) {
  const headers = {};
  if (options.token) headers.authorization = `Bearer ${options.token}`;
  if (options.key) headers['idempotency-key'] = options.key;
  if (options.body !== undefined) headers['content-type'] = 'application/json';
  const response = await fetch(`${proxyOrigin}${pathname}`, {
    method: options.method || 'GET', headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  const raw = await response.text();
  let body = {};
  if (raw) try { body = JSON.parse(raw); } catch {}
  const expected = options.expected || 200;
  if (response.status !== expected) throw new Error(`${options.method || 'GET'} ${pathname} returned ${response.status}/${body.error?.code || 'UNKNOWN'}`);
  return body;
}

function assertPaymentFacts(actual, expected, label) {
  if (!actual || actual.provider_create_count !== expected.creates || actual.provider_query_count !== expected.queries
    || actual.observation_count !== expected.observations || actual.order_count !== expected.orders) {
    throw new Error(`${label} payment facts diverged ${JSON.stringify(actual)}`);
  }
}

function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw new Error(`${label} was ${value || 'missing'}`);
  return value;
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 5000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}
