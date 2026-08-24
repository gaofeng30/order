/* global App, Behavior, Component, ORDER_REFUND_UNCLAIMED_L3_FIXTURE, ORDER_REFUND_UNCLAIMED_L3_PROXY_ORIGIN, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const orderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.wxml');
const adminOrdersTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.wxml');
const adminOrderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const qrcodeTemplate = require('../../../../apps/wechat-miniprogram/components/qrcode/qrcode.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const fixture = ORDER_REFUND_UNCLAIMED_L3_FIXTURE;
const proxyOrigin = ORDER_REFUND_UNCLAIMED_L3_PROXY_ORIGIN;
const pageDefinitions = {};
const requestObservations = [];
let appDefinition;
let componentDefinition;
let registeringPage;

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: 'refund-user-login' })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  navigateTo: () => {}, reLaunch: () => {}, redirectTo: () => {}, navigateBack: () => {},
  setClipboardData: options => queueMicrotask(() => options.success && options.success()),
  requestSubscribeMessage: options => queueMicrotask(() => options.success({ [options.tmplIds[0]]: 'reject' })),
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  request: options => {
    const url = new URL(options.url);
    fetch(url.toString(), {
      method: options.method || 'GET', headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let data = {};
      if (raw) try { data = JSON.parse(raw); } catch {}
      requestObservations.push({ method: options.method || 'GET', path: url.pathname, status: response.status });
      options.success({ statusCode: response.status, data });
    }).catch(error => options.fail(error));
  },
};

require('../../../../apps/wechat-miniprogram/components/icon/icon.js');
const iconDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/money/money.js');
const moneyDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/navbar/navbar.js');
const navbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/pill/pill.js');
const pillDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/qrcode/qrcode.js');
const qrcodeDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/app.js');
registeringPage = 'order-detail';
require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.js');
registeringPage = 'admin-orders';
require('../../../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.js');
registeringPage = 'admin-order-detail';
require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/order-detail/order-detail' }];

describe('refund and unclaimed L3 against private root HTTP and fresh MySQL', () => {
  it('renders user cancellation, durable provider finality, Merchant cross-day search and legal redeem', async () => {
    app.onLaunch();
    await waitFor(() => app.globalData.session.state !== 'loading', () => `session stayed ${app.globalData.session.state}`);
    if (app.globalData.session.state !== 'ready') throw new Error(`user session ended ${app.globalData.session.state}`);
    app.globalData.subscriptionTemplateIds = { REFUND_RESULT: 'refund-result-template' };

    const user = await renderOrderDetail('user-cancel', fixture.user_order.id);
    if (!user.instance.data.canCancel || !user.querySelector('.cancel-btn.on')) {
      throw new Error('user full-refund entry is not rendered for eligible RESERVED order');
    }
    user.querySelector('.cancel-btn.on').dispatchEvent('touchstart');
    user.querySelector('.cancel-btn.on').dispatchEvent('touchend');
    await waitFor(() => user.instance.data.cancelSheet, () => 'user cancellation sheet did not render');
    user.querySelector('.cs-confirm').dispatchEvent('touchstart');
    user.querySelector('.cs-confirm').dispatchEvent('touchend');
    await waitFor(() => user.instance.data.o && user.instance.data.o.state === 'REFUNDING', () => `user cancellation ended ${user.instance.data.o && user.instance.data.o.state}`);
    let facts = await orderFacts(fixture.user_order.id);
    assertRefund(facts, { state: 'REFUNDING', provider: 'PROCESSING', materialization: 'AWAITING_PROVIDER', creates: 1, queries: 0, observations: 0 }, 'user create');
    if (facts.order.payable_cents !== fixture.user_order.amount_cents || facts.order.redemption_cipher_present) {
      throw new Error('user full refund changed amount or retained a redemption cipher');
    }

    await setProviderMode('UNKNOWN');
    let worker = await control('POST', '/refund-worker');
    if (worker.claimed !== 1 || worker.pending !== 1 || worker.applied !== 0) throw new Error(`UNKNOWN worker diverged ${JSON.stringify(worker)}`);
    facts = await orderFacts(fixture.user_order.id);
    assertRefund(facts, { state: 'REFUNDING', provider: 'PROCESSING', materialization: 'AWAITING_PROVIDER', creates: 1, queries: 1, observations: 0 }, 'user UNKNOWN');

    await setProviderMode('PROCESSING');
    worker = await control('POST', '/refund-worker');
    if (worker.claimed !== 1 || worker.pending !== 1 || worker.applied !== 0) throw new Error(`PROCESSING worker diverged ${JSON.stringify(worker)}`);
    facts = await orderFacts(fixture.user_order.id);
    assertRefund(facts, { state: 'REFUNDING', provider: 'PROCESSING', materialization: 'AWAITING_PROVIDER', creates: 1, queries: 2, observations: 1 }, 'user PROCESSING');

    await setProviderMode('SUCCESS');
    worker = await control('POST', '/refund-worker');
    if (worker.claimed !== 1 || worker.applied !== 1) throw new Error(`SUCCESS worker diverged ${JSON.stringify(worker)}`);
    facts = await orderFacts(fixture.user_order.id);
    assertRefund(facts, { state: 'REFUNDED', provider: 'SUCCESS', materialization: 'APPLIED', creates: 1, queries: 3, observations: 2 }, 'user SUCCESS');
    if (facts.order.payable_cents !== fixture.user_order.amount_cents || facts.order.redemption_cipher_present) {
      throw new Error('provider SUCCESS did not preserve full amount and clear redemption capability');
    }
    worker = await control('POST', '/refund-worker');
    if (worker.claimed !== 0) throw new Error('terminal refund was queried/materialized twice');
    const refunded = await renderOrderDetail('user-refunded', fixture.user_order.id);
    if (refunded.instance.data.o.state !== 'REFUNDED' || refunded.instance.data.canCancel || refunded.querySelector('.cancel-btn.on')) {
      throw new Error('rendered user page invented an action after REFUNDED');
    }

    const beforeRedeem = await orderFacts(fixture.past_b.id);
    if (beforeRedeem.historical_stats.today_orders !== 0 || beforeRedeem.historical_stats.today_revenue_cents !== 0 || beforeRedeem.historical_stats.product_sales !== 0) {
      throw new Error('past READY unclaimed orders counted as effective revenue/sales');
    }
    app.globalData.session = { state: 'ready', accessToken: fixture.owner_token, expiresAt: '2099-01-01T00:00:00Z' };
    const merchantList = renderPage({
      definition: pageDefinitions['admin-orders'], template: adminOrdersTemplate, id: 'refund-merchant-orders',
      usingComponents: components('merchant-orders'), loadOptions: { lane: '待取餐' },
    });
    await waitFor(() => merchantList.instance.data.listState !== 'loading', () => 'Merchant READY lane stayed loading');
    merchantList.instance.setData({ kw: fixture.past_b.order_no });
    await pageDefinitions['admin-orders'].load.call(merchantList.instance);
    if (merchantList.instance.data.list.length !== 1 || merchantList.instance.data.list[0].id !== fixture.past_b.id
      || merchantList.instance.data.list[0].pickupDate !== fixture.past_date
      || !merchantList.querySelector('.aorder')) {
      throw new Error('rendered Merchant cross-day search did not identify the exact past READY order');
    }

    const merchantDetail = renderPage({
      definition: pageDefinitions['admin-order-detail'], template: adminOrderDetailTemplate, id: 'refund-merchant-detail',
      usingComponents: components('merchant-detail'), loadOptions: { id: fixture.past_b.id },
    });
    await waitFor(() => merchantDetail.instance.data.detailState !== 'loading', () => 'Merchant order detail stayed loading');
    if (merchantDetail.instance.data.o.state !== 'READY_FOR_PICKUP' || merchantDetail.instance.data.meta.label !== '核销') {
      throw new Error('past READY order did not render its legal Merchant redeem action');
    }
    merchantDetail.querySelector('.foot-main').dispatchEvent('touchstart');
    merchantDetail.querySelector('.foot-main').dispatchEvent('touchend');
    await waitFor(() => merchantDetail.instance.data.o.state === 'COMPLETED', () => `Merchant redeem ended ${merchantDetail.instance.data.o.state}`);
    const afterRedeem = await orderFacts(fixture.past_b.id);
    if (afterRedeem.historical_stats.today_orders !== 1
      || afterRedeem.historical_stats.today_revenue_cents !== fixture.past_b.amount_cents
      || afterRedeem.historical_stats.product_sales !== fixture.past_b.quantity) {
      throw new Error(`only COMPLETED order should count as effective facts ${JSON.stringify(afterRedeem.historical_stats)}`);
    }
    if (afterRedeem.inventory_table_count !== 0 || facts.inventory_table_count !== 0) {
      throw new Error('refund flow invented an inventory/stock model outside the PRD');
    }
  });
});

async function renderOrderDetail(suffix, id) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pageDefinitions['order-detail'], template: orderDetailTemplate, id: `refund-order-detail-${suffix}`,
    usingComponents: components(`order-detail-${suffix}`, true), loadOptions: { id },
  });
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `user detail ${id} stayed loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`user detail ${id} ended ${page.instance.data.detailState}`);
  return page;
}

function components(suffix, includeQR = false) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  const pill = registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' });
  const result = {
    icon, money, pill,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
  };
  if (includeQR) result.qrcode = registerComponent({ definition: qrcodeDefinition, template: qrcodeTemplate, id: `qrcode-${suffix}`, tagName: 'qrcode' });
  return result;
}

async function setProviderMode(mode) {
  const view = await control('PUT', '/provider-mode', { mode });
  if (view.mode !== mode) throw new Error(`provider mode ended ${view.mode}`);
}

function control(method, suffix, body) {
  return direct(`/api/v1/__acceptance/refund-unclaimed${suffix}`, { method, body, expected: 200 });
}

async function orderFacts(orderID) {
  return direct(`/api/v1/__acceptance/refund-unclaimed/facts?order_id=${encodeURIComponent(orderID)}`, { expected: 200 });
}

async function direct(pathname, options = {}) {
  const headers = {};
  if (options.body !== undefined) headers['content-type'] = 'application/json';
  const response = await fetch(`${proxyOrigin}${pathname}`, {
    method: options.method || 'GET', headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  const raw = await response.text();
  let body = {};
  if (raw) try { body = JSON.parse(raw); } catch {}
  if (response.status !== (options.expected || 200)) throw new Error(`${options.method || 'GET'} ${pathname} returned ${response.status}/${body.error?.code || 'UNKNOWN'}`);
  return body;
}

function assertRefund(view, expected, label) {
  const order = view && view.order;
  const actual = order && order.refund;
  if (!actual || order.state !== expected.state || actual.provider_state !== expected.provider
    || actual.materialization_state !== expected.materialization
    || actual.provider_create_count !== expected.creates || actual.provider_query_count !== expected.queries
    || actual.observation_count !== expected.observations) {
    throw new Error(`${label} refund facts diverged ${JSON.stringify(view)}`);
  }
}

async function waitFor(predicate, describe) {
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    if (predicate()) return;
    await simulate.sleep(20);
  }
  throw new Error(describe());
}
