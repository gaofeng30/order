/* global App, Behavior, Component, ORDER_MERCHANT_FAILURE_FIXTURE, ORDER_MERCHANT_FAILURE_ORIGIN, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const adminOrderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.wxml');
const adminOrdersTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.wxml');
const adminProductsTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-products/admin-products.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const fixture = ORDER_MERCHANT_FAILURE_FIXTURE;
const pages = {};
const observations = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let currentRoute = 'pages/admin-orders/admin-orders';
let failureMode = '';
let lastNavigation = '';

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages[registeringPage] = definition; };
globalThis.wx = {
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  request: options => {
    const requestURL = new URL(options.url);
    const headers = Object.assign({}, options.header || {});
    if (failureMode) headers['X-Merchant-Failure-Mode'] = failureMode;
    fetch(requestURL.toString(), {
      method: options.method || 'GET', headers,
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let data = {};
      if (raw) { try { data = JSON.parse(raw); } catch {} }
      observations.push({
        method: options.method || 'GET', path: requestURL.pathname, query: requestURL.search,
        status: response.status, failure_mode: failureMode || undefined,
      });
      options.success({ statusCode: response.status, data });
    }).catch(error => options.fail(error));
  },
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
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/app.js');
for (const name of ['admin-orders', 'admin-order-detail', 'admin-products']) {
  registeringPage = name;
  require(`../../../../apps/wechat-miniprogram/pages/${name}/${name}.js`);
}

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
app.globalData.apiBaseUrl = ORDER_MERCHANT_FAILURE_ORIGIN;
app.globalData.runtimeEndpoint = { state: 'ready', envVersion: 'develop', origin: ORDER_MERCHANT_FAILURE_ORIGIN, errorCode: '' };
app.globalData.session = { state: 'ready', accessToken: fixture.mini_token, expiresAt: fixture.expires_at };
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: currentRoute }];

describe('PAGE-M02/M03/M05 merchant failure closure', () => {
  it('renders five live lanes/search and makes failed store close visibly fail without false success', async () => {
    failureMode = '';
    const page = render('admin-orders', 'm02', { lane: '已预约' });
    await readyList(page, 'M02 initial');
    const lanes = ['已预约', '制作中', '待取餐', '已完成', '已退款'];
    if (JSON.stringify(page.instance.data.lanes) !== JSON.stringify(lanes)) {
      throw new Error(`M02 lanes ${JSON.stringify(page.instance.data.lanes)}`);
    }
    for (const lane of lanes) {
      await pages['admin-orders'].switchLane.call(page.instance, { currentTarget: { dataset: { l: lane } } });
      if (!page.instance.data.list.some(order => order.id === fixture.lanes[lane])) {
        throw new Error(`M02 ${lane} omitted ${fixture.lanes[lane]}`);
      }
    }
    await pages['admin-orders'].onKw.call(page.instance, { detail: { value: fixture.search_order_no } });
    if (page.instance.data.list.length !== 1 || page.instance.data.list[0].id !== fixture.search_order_id
      || !page.dom.textContent.includes(fixture.search_pickup_number)) {
      throw new Error(`M02 search mismatch ${JSON.stringify(page.instance.data.list)}`);
    }

    const closed = page.querySelectorAll('.biz-seg').find(node => node.dom.dataset.b === 'closed');
    failureMode = 'store-status-503';
    await tap(closed);
    await waitFor(() => page.instance.data.statusWriteState === 'error', () => `M02 failed close ${page.instance.data.statusWriteState}`);
    if (page.instance.data.storeStatus !== 'open' || !page.dom.textContent.includes('营业状态保存失败')) {
      throw new Error('M02 failed close projected local success');
    }
    failureMode = '';
    await tap(page.querySelectorAll('.biz-seg').find(node => node.dom.dataset.b === 'closed'));
    await waitFor(() => page.instance.data.statusWriteState === 'ready', () => `M02 close retry ${page.instance.data.statusWriteState}`);
    if (page.instance.data.storeStatus !== 'closed') throw new Error('M02 successful close was not rendered');
    lastNavigation = '';
    pages['admin-orders'].reset.call(page.instance);
    if (lastNavigation !== '/pages/launch/launch') throw new Error(`M02 identity reset ${lastNavigation}`);
  });

  it('keeps illegal/503/enqueue-failed READY attempts unchanged and renders one legal READY response', async () => {
    failureMode = '';
    const illegal = render('admin-order-detail', 'm03-illegal', { id: fixture.ready.illegal_order_id });
    await readyDetail(illegal, 'M03 illegal');
    const beforeIllegal = observations.length;
    const illegalResult = await pages['admin-order-detail'].markReady.call(illegal.instance);
    if (illegalResult !== false || observations.length !== beforeIllegal || illegal.instance.data.o.state !== 'RESERVED'
      || illegal.instance.data.meta.label !== '已预约') {
      throw new Error('M03 illegal state issued READY or projected a new state');
    }

    const unavailable = render('admin-order-detail', 'm03-503', { id: fixture.ready.http_503_order_id });
    await readyDetail(unavailable, 'M03 503');
    failureMode = 'ready-503';
    await tap(unavailable.querySelector('.foot-main'));
    await waitFor(() => unavailable.instance.data.actionState === 'error', () => `M03 503 ${unavailable.instance.data.actionState}`);
    if (unavailable.instance.data.o.state !== 'PREPARING' || !unavailable.dom.textContent.includes('操作失败')) {
      throw new Error('M03 HTTP 503 projected READY');
    }

    const enqueue = render('admin-order-detail', 'm03-enqueue', { id: fixture.ready.enqueue_fail_order_id });
    await readyDetail(enqueue, 'M03 enqueue');
    failureMode = 'ready-enqueue-fail';
    await tap(enqueue.querySelector('.foot-main'));
    await waitFor(() => enqueue.instance.data.actionState === 'error', () => `M03 enqueue ${enqueue.instance.data.actionState}`);
    if (enqueue.instance.data.o.state !== 'PREPARING' || !enqueue.dom.textContent.includes('操作失败')) {
      throw new Error('M03 enqueue failure projected READY');
    }

    const success = render('admin-order-detail', 'm03-success', { id: fixture.ready.success_order_id });
    await readyDetail(success, 'M03 success');
    failureMode = '';
    await tap(success.querySelector('.foot-main'));
    await waitFor(() => success.instance.data.o && success.instance.data.o.state === 'READY_FOR_PICKUP',
      () => `M03 success ${success.instance.data.actionState}/${success.instance.data.o && success.instance.data.o.state}`);
    if (success.instance.data.actionState !== 'ready' || success.instance.data.meta.label !== '核销') {
      throw new Error('M03 legal READY response was not rendered');
    }
  });

  it('fails closed on sold-out HTTP/drift, then renders only the successful today fact', async () => {
    failureMode = '';
    const page = render('admin-products', 'm05');
    await readyList(page, 'M05 initial');
    let product = targetProduct(page);
    if (product.soldOut || !page.dom.textContent.includes('可售')) throw new Error('M05 did not start from today available');

    failureMode = 'soldout-503';
    await tap(soldoutControl(page));
    await waitFor(() => page.instance.data.actionState === 'error', () => `M05 503 ${page.instance.data.actionState}`);
    product = targetProduct(page);
    if (product.soldOut || !page.dom.textContent.includes('售罄状态保存失败')) {
      throw new Error('M05 HTTP failure projected sold-out');
    }

    page.instance.setData({ actionState: 'idle' });
    failureMode = 'soldout-drift';
    await tap(soldoutControl(page));
    await waitFor(() => page.instance.data.actionState === 'error', () => `M05 drift ${page.instance.data.actionState}`);
    if (targetProduct(page).soldOut) throw new Error('M05 mismatched server fact projected sold-out');

    page.instance.setData({ actionState: 'idle' });
    failureMode = '';
    await tap(soldoutControl(page));
    await waitFor(() => page.instance.data.actionState === 'ready', () => `M05 success ${page.instance.data.actionState}`);
    if (!targetProduct(page).soldOut || !page.dom.textContent.includes('恢复售卖')) {
      throw new Error('M05 successful today sold-out was not rendered');
    }
  });
});

function components(suffix, productPage) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const using = {
    icon,
    money: registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' }),
    pill: registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' }),
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
  };
  if (productPage) using.imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
  return using;
}
function render(name, suffix, loadOptions = {}) {
  document.body.innerHTML = '';
  currentRoute = `pages/${name}/${name}`;
  return renderPage({
    definition: pages[name],
    template: name === 'admin-orders' ? adminOrdersTemplate : name === 'admin-order-detail' ? adminOrderDetailTemplate : adminProductsTemplate,
    id: `${name}-${suffix}`, usingComponents: components(`${name}-${suffix}`, name === 'admin-products'), loadOptions,
  });
}
async function tap(node) {
  if (!node) throw new Error('rendered control missing');
  await node.dispatchEvent('touchstart');
  await node.dispatchEvent('touchend');
}
async function readyList(page, label) {
  await waitFor(() => page.instance.data.listState !== 'loading', () => `${label} stayed loading`);
  if (page.instance.data.listState !== 'ready') throw new Error(`${label} ended ${page.instance.data.listState}`);
}
async function readyDetail(page, label) {
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `${label} stayed loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`${label} ended ${page.instance.data.detailState}`);
}
function targetProduct(page) {
  const product = page.instance.data.list.find(item => item.id === fixture.product.id);
  if (!product) throw new Error(`M05 omitted product ${fixture.product.id}`);
  return product;
}
function soldoutControl(page) {
  return page.querySelectorAll('.pa-soldout').find(node => node.dom.dataset.id === fixture.product.id);
}
async function waitFor(predicate, message, timeout = 10000) {
  const deadline = Date.now() + timeout;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(20);
  }
}
