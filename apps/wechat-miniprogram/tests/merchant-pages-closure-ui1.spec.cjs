/* global App, Behavior, Component, ORDER_MERCHANT_CLOSURE_FIXTURE, ORDER_MERCHANT_CLOSURE_ORIGIN, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const adminOrderDetailTemplate = require('../pages/admin-order-detail/admin-order-detail.wxml');
const adminOrdersTemplate = require('../pages/admin-orders/admin-orders.wxml');
const adminProductsTemplate = require('../pages/admin-products/admin-products.wxml');
const adminVerifyTemplate = require('../pages/admin-verify/admin-verify.wxml');
const iconTemplate = require('../components/icon/icon.wxml');
const imagephTemplate = require('../components/imageph/imageph.wxml');
const moneyTemplate = require('../components/money/money.wxml');
const navbarTemplate = require('../components/navbar/navbar.wxml');
const pillTemplate = require('../components/pill/pill.wxml');
const tabbarTemplate = require('../components/tabbar/tabbar.wxml');
const toastTemplate = require('../components/toast/toast.wxml');
const { registerComponent, renderPage } = require('../../../tools/miniprogram-ui/test/browser/page-adapter.cjs');

const fixture = ORDER_MERCHANT_CLOSURE_FIXTURE;
let appDefinition;
let componentDefinition;
let registeringPage = '';
let lastNavigation = '';
const pageDefinitions = {};
const requests = [];
const scanTokens = [];

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  scanCode: options => queueMicrotask(() => {
    const token = scanTokens.shift();
    if (token) options.success({ result: token });
    else options.fail({ errMsg: 'scanCode:fail empty closure fixture' });
  }),
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  request: options => {
    const requestURL = new URL(options.url);
    fetch(requestURL.toString(), {
      method: options.method || 'GET', headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let data = {};
      if (raw) try { data = JSON.parse(raw); } catch (error) { data = {}; }
      requests.push({ method: options.method || 'GET', path: requestURL.pathname + requestURL.search, status: response.status });
      options.success({ statusCode: response.status, data });
    }).catch(error => options.fail(error));
  },
};

require('../components/icon/icon.js');
const iconDefinition = componentDefinition;
require('../components/imageph/imageph.js');
const imagephDefinition = componentDefinition;
require('../components/money/money.js');
const moneyDefinition = componentDefinition;
require('../components/navbar/navbar.js');
const navbarDefinition = componentDefinition;
require('../components/pill/pill.js');
const pillDefinition = componentDefinition;
require('../components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../app.js');
for (const page of ['admin-orders', 'admin-order-detail', 'admin-verify', 'admin-products']) {
  registeringPage = page;
  require(`../pages/${page}/${page}.js`);
}

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
app.globalData.apiBaseUrl = ORDER_MERCHANT_CLOSURE_ORIGIN;
app.globalData.runtimeEndpoint = { state: 'ready', envVersion: 'develop', origin: ORDER_MERCHANT_CLOSURE_ORIGIN, errorCode: '' };
app.globalData.session = { state: 'ready', accessToken: fixture.mini_token, expiresAt: fixture.expires_at };
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/admin-orders/admin-orders' }];

function components(suffix, includeOrder, includeProduct) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const using = {
    icon,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
  };
  if (includeOrder || includeProduct) {
    using.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
    using.pill = registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' });
  }
  if (includeProduct) {
    using.imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
  }
  return using;
}

async function waitFor(predicate, message, timeout = 6000) {
  const deadline = Date.now() + timeout;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(20);
  }
}

function render(name, suffix, options = {}) {
  document.body.innerHTML = '';
  return renderPage({
    definition: pageDefinitions[name],
    template: name === 'admin-orders' ? adminOrdersTemplate
      : name === 'admin-order-detail' ? adminOrderDetailTemplate
        : name === 'admin-verify' ? adminVerifyTemplate : adminProductsTemplate,
    id: `${name}-${suffix}`,
    usingComponents: components(`${name}-${suffix}`, true, name === 'admin-products'),
    loadOptions: options,
  });
}

describe('PAGE-M02..M05 rendered merchant closure against root HTTP and fresh MySQL', () => {
  it('PAGE-M02 renders exactly five server lanes, cross-lane search, store status, and identity reset', async () => {
    const page = render('admin-orders', 'm02', { lane: '已预约' });
    await waitFor(() => page.instance.data.listState !== 'loading', () => `M02 initial ${page.instance.data.listState}`);
    if (JSON.stringify(page.instance.data.lanes) !== JSON.stringify(['已预约', '制作中', '待取餐', '已完成', '已退款'])) {
      throw new Error(`M02 lanes ${JSON.stringify(page.instance.data.lanes)}`);
    }
    const expectations = fixture.lanes;
    for (const lane of page.instance.data.lanes) {
      await pageDefinitions['admin-orders'].switchLane.call(page.instance, { currentTarget: { dataset: { l: lane } } });
      if (!page.instance.data.list.some(order => order.id === expectations[lane])) {
        throw new Error(`M02 ${lane} omitted ${expectations[lane]}`);
      }
    }
    await pageDefinitions['admin-orders'].onKw.call(page.instance, { detail: { value: fixture.search_order_no } });
    if (page.instance.data.list.length !== 1 || page.instance.data.list[0].id !== fixture.search_order_id) {
      throw new Error(`M02 cross-lane search mismatch ${JSON.stringify(page.instance.data.list.map(item => item.id))}`);
    }
    const closed = page.querySelectorAll('.biz-seg').find(item => item.dom.dataset.b === 'closed');
    closed.dispatchEvent('touchstart'); closed.dispatchEvent('touchend');
    await waitFor(() => page.instance.data.storeStatus === 'closed', () => `M02 close ${page.instance.data.storeStatus}`);
    lastNavigation = '';
    pageDefinitions['admin-orders'].reset.call(page.instance);
    if (lastNavigation !== '/pages/launch/launch') throw new Error(`M02 reset ${lastNavigation}`);
  });

  it('PAGE-M03 renders PREPARING detail and commits READY from the production response', async () => {
    const page = render('admin-order-detail', 'm03', { id: fixture.ready_order_id });
    await waitFor(() => page.instance.data.detailState !== 'loading', () => `M03 load ${page.instance.data.detailState}`);
    if (page.instance.data.o.state !== 'PREPARING' || page.instance.data.meta.label !== '备好') {
      throw new Error(`M03 initial ${page.instance.data.o.state}/${page.instance.data.meta.label}`);
    }
    page.querySelector('.foot-main').dispatchEvent('touchstart'); page.querySelector('.foot-main').dispatchEvent('touchend');
    await waitFor(() => page.instance.data.o && page.instance.data.o.state === 'READY_FOR_PICKUP',
      () => `M03 ready ${page.instance.data.actionState}/${page.instance.data.o && page.instance.data.o.state}`);
    if (page.instance.data.actionState !== 'ready' || page.instance.data.meta.label !== '核销') throw new Error('M03 did not render READY response');
  });

  it('PAGE-M04 atomically scans, replays under a fresh key, manually redeems, and rejects refunded/invalid credentials', async () => {
    const page = render('admin-verify', 'm04');
    scanTokens.push(fixture.scan_token, fixture.scan_token, fixture.refunded_token);
    const scan = page.querySelector('.verify-center .btn');
    for (let attempt = 0; attempt < 2; attempt += 1) {
      scan.dispatchEvent('touchstart'); scan.dispatchEvent('touchend');
      await waitFor(() => page.instance.data.lookupState !== 'loading' && page.instance.data.lookupState !== 'idle',
        () => `M04 scan ${attempt} ${page.instance.data.lookupState}`);
      if (page.instance.data.lookupState !== 'completed' || page.instance.data.lastResult.id !== fixture.scan_order_id) {
        throw new Error(`M04 scan ${attempt} ${page.instance.data.lookupState}`);
      }
      page.instance.setData({ lookupState: 'idle', lastResult: null });
    }
    page.instance.setData({ code: fixture.manual_code });
    await pageDefinitions['admin-verify'].manual.call(page.instance);
    if (page.instance.data.lookupState !== 'completed' || page.instance.data.lastResult.id !== fixture.manual_order_id) {
      throw new Error(`M04 manual ${page.instance.data.lookupState}`);
    }
    page.instance.setData({ lookupState: 'idle', lastResult: null });
    scan.dispatchEvent('touchstart'); scan.dispatchEvent('touchend');
    await waitFor(() => page.instance.data.lookupState === 'error', () => `M04 refunded ${page.instance.data.lookupState}`);
    page.instance.setData({ code: '9999', lookupState: 'idle' });
    await pageDefinitions['admin-verify'].manual.call(page.instance);
    if (page.instance.data.lookupState !== 'error' || !page.dom.textContent.includes('核销失败')) throw new Error('M04 invalid code was not visibly rejected');
  });

  it('PAGE-M05 loads while closed, merges meal products, and writes today-only sold-out fact', async () => {
    const page = render('admin-products', 'm05');
    await waitFor(() => page.instance.data.listState !== 'loading', () => `M05 load ${page.instance.data.listState}`);
    const ids = new Set(page.instance.data.list.map(item => item.id));
    if (!ids.has(fixture.products.lunch) || !ids.has(fixture.products.dinner)) {
      throw new Error(`M05 meal products ${JSON.stringify([...ids])}`);
    }
    const control = page.querySelectorAll('.pa-soldout').find(item => item.dom.dataset.id === fixture.products.lunch);
    if (!control) throw new Error('M05 lunch sold-out control absent');
    control.dispatchEvent('touchstart'); control.dispatchEvent('touchend');
    await waitFor(() => page.instance.data.list.find(item => item.id === fixture.products.lunch).soldOut === true,
      () => `M05 soldout ${page.instance.data.actionState}`);
    if (page.instance.data.actionState !== 'ready') throw new Error('M05 did not render sold-out response');
  });

  after(() => {
    globalThis.__ORDER_MERCHANT_CLOSURE_EVIDENCE = { requests };
  });
});
