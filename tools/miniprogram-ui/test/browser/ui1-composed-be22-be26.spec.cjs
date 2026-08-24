/* global App, Behavior, Component, ORDER_BE22_BE26_PROXY_ORIGIN, ORDER_BE22_BE26_RUN_ID, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const adminOrderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.wxml');
const adminVerifyTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-verify/admin-verify.wxml');
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const orderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.wxml');
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

const proxyOrigin = ORDER_BE22_BE26_PROXY_ORIGIN;
const runID = ORDER_BE22_BE26_RUN_ID;
const pages = {};
const observations = [];
const paymentCalls = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let lastNavigation = null;
let loginCode = `${runID}-unbound-session`;
let scanToken = '';
let unauthorizedPath = '';
let unavailablePath = '';
let exactReplayPath = '';

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: loginCode })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, pixelRatio: 2, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, pixelRatio: 2, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  requestPayment: options => { paymentCalls.push(options.package); queueMicrotask(() => options.success({ errMsg: 'requestPayment:ok' })); },
  setClipboardData: options => queueMicrotask(() => options.success && options.success()),
  canvasToTempFilePath: options => queueMicrotask(() => options.success({ tempFilePath: '/private/tmp/ui1-qrcode.png' })),
  scanCode: options => queueMicrotask(() => scanToken
    ? options.success({ result: scanToken })
    : options.fail({ errMsg: 'scanCode:fail no token' })),
  request: options => {
    const url = new URL(options.url);
    const headers = Object.assign({}, options.header || {});
    if (url.pathname === unauthorizedPath) headers.Authorization = 'Bearer invalid-be22-be26-token';
    if (url.pathname === unavailablePath) headers['X-BE22-BE26-Force-Status'] = '503';
    if (url.pathname === exactReplayPath) headers['X-BE22-BE26-Network-Replay'] = 'same-request';
    fetch(url.toString(), {
      method: options.method || 'GET', headers,
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      let data = {};
      const raw = await response.text();
      if (raw) { try { data = JSON.parse(raw); } catch {} }
      observations.push({ method: options.method || 'GET', path: url.pathname, status: response.status });
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
registeringPage = 'confirm'; require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.js');
registeringPage = 'home'; require('../../../../apps/wechat-miniprogram/pages/home/home.js');
registeringPage = 'menu'; require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'order-detail'; require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.js');
registeringPage = 'admin-order-detail'; require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.js');
registeringPage = 'admin-verify'; require('../../../../apps/wechat-miniprogram/pages/admin-verify/admin-verify.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/home/home' }];

describe('BE-22 and BE-26 exact rendered Mini controls against fresh MySQL/root HTTP', () => {
  it('rejects phone consent without side effects, then proves truthful READY token and redemption shields', async () => {
    await launchFreshSession();
    const initialStatus = await api('GET', '/api/v1/me/primary-phone', undefined, app.globalData.session.accessToken);
    if (initialStatus.primary_phone_bound !== false) throw new Error(`BE-22 initial MySQL projection was ${JSON.stringify(initialStatus)}`);

    const time = runtimeTimes();
    seedVisibleCheckoutFixture(time.targetDate, time.pickupTime);
    const be22 = await openConfirm('be22-unbound', '未绑定用户', false);
    if (be22.instance.data.phoneState !== 'unbound' || !be22.dom.textContent.includes('未绑定')) {
      throw new Error(`BE-22 did not visibly start unbound: ${be22.instance.data.phoneState}`);
    }
    if (!be22.dom.textContent.includes('绑定微信手机号')) throw new Error('BE-22 rendered phone authorization control was absent');
    const refusalStart = observations.length;
    await pages.confirm.onGetPhoneNumber.call(be22.instance, { detail: { errMsg: 'getPhoneNumber:fail user deny' } });
    await simulate.sleep(30);
    await pages.confirm.onGetPhoneNumber.call(be22.instance, { detail: { errMsg: 'getPhoneNumber:fail system error' } });
    await simulate.sleep(30);
    lastNavigation = null;
    tap(be22.querySelector('.pay-btn'));
    await simulate.sleep(30);
    const refused = observations.slice(refusalStart);
    if (be22.instance.data.phoneState !== 'unbound' || Object.keys(app.globalData.cart).length !== 1
      || lastNavigation !== null || refused.some(item => ['/api/v1/me/bind-phone', '/api/v1/quotes', '/api/v1/orders/prepay'].includes(item.path))
      || !be22.dom.textContent.includes('请先绑定手机号')) {
      throw new Error(`BE-22 refusal was not zero-side-effect: ${JSON.stringify(refused)}`);
    }
    const stillUnbound = await api('GET', '/api/v1/me/primary-phone', undefined, app.globalData.session.accessToken);
    if (stillUnbound.primary_phone_bound !== false) throw new Error('BE-22 refusal bound a persisted phone');

    await pages.confirm.onGetPhoneNumber.call(be22.instance, { detail: { code: `${runID}-accepted-phone` } });
    await waitFor(() => be22.instance.data.phoneState !== 'binding', () => `BE-22 accepted phone remained ${be22.instance.data.phoneState}`);
    if (be22.instance.data.phoneState !== 'bound' || Object.keys(app.globalData.cart).length !== 1
      || !be22.dom.textContent.includes('+*********0000') || lastNavigation !== null) {
      throw new Error('BE-22 accepted retry did not bind on the same checkout with its cart');
    }
    const bound = await api('GET', '/api/v1/me/primary-phone', undefined, app.globalData.session.accessToken);
    if (bound.primary_phone_bound !== true || bound.masked_phone !== '+*********0000') throw new Error(`BE-22 bound fact was ${JSON.stringify(bound)}`);

    const pcToken = await acquirePCSession();
    await api('PUT', '/api/v1/admin/settings', settings(time), pcToken, newKey('settings'));
    const category = await api('POST', '/api/v1/admin/categories', { name: `BE22BE26-${runID}` }, pcToken, newKey('category'), 201);
    const categoryID = exactID(category.category.id, 'category');
    const created = await api('POST', '/api/v1/admin/products', {
      name: `BE26商品-${runID}`, price_cents: 1234, category_id: categoryID,
      meal_period: 'dinner', description: 'BE-26 fresh composed order', images: [],
    }, pcToken, newKey('product'), 201);
    const productID = exactID(created.product.id, 'product');

    const nonReady = await createOrderThroughRenderedUI('non-ready', productID, time.targetDate, time.pickupTime);
    if (nonReady.state !== 'PREPARING') throw new Error(`BE-26 non-READY setup was ${nonReady.state}`);
    const home = await openHome('no-ready');
    if (!home.instance.data.ongoing || home.instance.data.ongoing.ready !== false) {
      throw new Error(`BE-26 real active orders did not project non-READY: ${JSON.stringify(home.instance.data.ongoing)}`);
    }
    lastNavigation = null;
    tap(home.querySelectorAll('.grid-item').find(node => node.dom.dataset.k === 'pickup'));
    await simulate.sleep(30);
    if (lastNavigation !== null || !home.dom.textContent.includes('暂无可取餐订单')) {
      throw new Error(`BE-26 no-READY entry navigated to ${lastNavigation || 'nothing without visible toast'}`);
    }
    const nonReadyDetail = await openUserDetail('non-ready', nonReady.id);
    if (nonReadyDetail.instance.data.showQr || nonReadyDetail.instance.data.qrToken
      || nonReadyDetail.querySelector('.qr-white') || !nonReadyDetail.dom.textContent.includes('订单未备好时不会展示核销二维码')) {
      throw new Error('BE-26 PREPARING order exposed a token or take-code UI');
    }

    const verify = await openVerify('negative-shields');
    await manualVerify(verify, nonReady.code, 'not-ready');
    await manualVerify(verify, '9999', 'wrong-code');

    const ready = await createOrderThroughRenderedUI('ready', productID, time.targetDate, time.pickupTime);
    const merchantDetail = await openMerchantDetail('ready', ready.id);
    tap(merchantDetail.querySelector('.foot-main'));
    await waitFor(() => merchantDetail.instance.data.o && merchantDetail.instance.data.o.state === 'READY_FOR_PICKUP',
      () => `BE-26 mark-ready remained ${merchantDetail.instance.data.o && merchantDetail.instance.data.o.state}`);

    let crossDateCode = ready.code;
    if (time.targetDate === time.today) {
      for (let index = 1; index <= 3; index++) {
        const far = await createOrderThroughRenderedUI(`cross-date-${index}`, productID, time.tomorrow, time.pickupTime);
        crossDateCode = far.code;
        if (far.state !== 'RESERVED') throw new Error(`cross-date order ${index} was ${far.state}`);
      }
    }
    await manualVerify(verify, crossDateCode, 'cross-date');

    const readyHome = await openHome('ready');
    if (!readyHome.instance.data.ongoing || readyHome.instance.data.ongoing.ready !== true
      || readyHome.instance.data.ongoing.orderId !== ready.id) throw new Error('BE-26 READY home entry did not use server READY fact');
    lastNavigation = null;
    tap(readyHome.querySelectorAll('.grid-item').find(node => node.dom.dataset.k === 'pickup'));
    await simulate.sleep(20);
    if (lastNavigation !== `/pages/order-detail/order-detail?id=${ready.id}`) throw new Error(`READY take-code navigated ${lastNavigation}`);
    const readyDetail = await openUserDetail('ready-token', ready.id);
    scanToken = readyDetail.instance.data.qrToken;
    if (readyDetail.instance.data.o.state !== 'READY_FOR_PICKUP' || !readyDetail.instance.data.showQr || !scanToken
      || !readyDetail.querySelector('.qr-white') || !readyDetail.dom.textContent.includes('出示二维码')) {
      throw new Error('BE-26 READY did not render the server token and take-code UI');
    }

    unauthorizedPath = '/api/v1/verify/scan';
    await scanAndExpectFailure(verify, '401');
    unauthorizedPath = '';
    unavailablePath = '/api/v1/verify/scan';
    await scanAndExpectFailure(verify, '503');
    unavailablePath = '';
    const stillReady = await openUserDetail('still-ready-after-failures', ready.id);
    if (stillReady.instance.data.o.state !== 'READY_FOR_PICKUP' || !stillReady.instance.data.showQr) {
      throw new Error('401/503 produced a false terminal redemption');
    }

    exactReplayPath = '/api/v1/verify/scan';
    document.body.innerHTML = '';
    const replayVerify = await openVerify('exact-network-replay');
    tap(replayVerify.querySelector('.verify-center .btn'));
    await waitFor(() => replayVerify.instance.data.lookupState !== 'loading' && replayVerify.instance.data.lookupState !== 'idle',
      () => `exact replay remained ${replayVerify.instance.data.lookupState}`);
    exactReplayPath = '';
    if (replayVerify.instance.data.lookupState !== 'completed' || !replayVerify.instance.data.lastResult
      || replayVerify.instance.data.lastResult.id !== ready.id || replayVerify.instance.data.lastResult.state !== 'COMPLETED'
      || !replayVerify.dom.textContent.includes('核销成功')) {
      throw new Error('exact same-request replay did not render the first honest COMPLETED result');
    }
    const completed = await openUserDetail('completed', ready.id);
    if (completed.instance.data.o.state !== 'COMPLETED' || completed.instance.data.showQr || completed.instance.data.qrToken
      || completed.querySelector('.qr-white')) throw new Error('COMPLETED detail retained a stale redemption token');

    const activeAfter = await api('GET', '/api/v1/orders?active=true', undefined, app.globalData.session.accessToken);
    if (activeAfter.orders.some(order => order.id === ready.id)) throw new Error('COMPLETED order remained in active orders');
  });
});

function globalComponents(suffix, kind = 'basic') {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const components = {
    icon,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
  };
  if (kind === 'menu' || kind === 'order') components.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  if (kind === 'menu') {
    components.stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
    const imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
    components.imageph = imageph;
    components.customize = registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, imageph, money: components.money, stepper: components.stepper },
    });
  }
  if (kind === 'order') {
    components.pill = registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' });
    components.qrcode = registerComponent({ definition: qrcodeDefinition, template: qrcodeTemplate, id: `qrcode-${suffix}`, tagName: 'qrcode' });
  }
  return components;
}

async function launchFreshSession() {
  app.globalData.session = { state: 'idle', accessToken: '', expiresAt: '' };
  app.globalData.cart = {};
  app.globalData.pickup = null;
  app.onLaunch();
  await waitFor(() => app.globalData.session.state !== 'loading', () => `fresh session remained ${app.globalData.session.state}`);
  if (app.globalData.session.state !== 'ready') throw new Error(`fresh session ended ${app.globalData.session.state}`);
}

function seedVisibleCheckoutFixture(date, time) {
  app.globalData.pickup = { date, mealPeriod: 'dinner', time };
  app.globalData.cart = {
    999999: {
      product: {
        id: '999999', category_id: '999999', name: '绑定前购物车可见商品', description: '', specification: '',
        meal_period: 'dinner', images: [], listed: true, sold_out: false,
        original_unit_price_cents: 1234, isStaffPrice: false, price_cents: 1234,
      },
      qty: 1, flavors: [], note: '',
    },
  };
}

async function openConfirm(suffix, contact, requireBound = true) {
  document.body.innerHTML = '';
  lastNavigation = null;
  const page = renderPage({ definition: pages.confirm, template: confirmTemplate, id: `confirm-${suffix}`, usingComponents: globalComponents(`${suffix}-confirm`, 'menu') });
  await waitFor(() => page.instance.data.phoneState !== 'loading', () => `confirm ${suffix} remained loading`);
  if (requireBound && page.instance.data.phoneState !== 'bound') throw new Error(`confirm ${suffix} phone was ${page.instance.data.phoneState}`);
  page.querySelector('.field-in').dispatchEvent('input', { detail: { value: contact } });
  await waitFor(() => page.instance.data.form.contact === contact, () => `confirm ${suffix} contact missing`);
  return page;
}

async function createOrderThroughRenderedUI(suffix, productID, date, time) {
  app.globalData.cart = {};
  app.globalData.pickup = { date, mealPeriod: 'dinner', time };
  document.body.innerHTML = '';
  const menu = renderPage({ definition: pages.menu, template: menuTemplate, id: `menu-${suffix}`, usingComponents: globalComponents(`${suffix}-menu`, 'menu') });
  await waitFor(() => menu.instance.data.listState !== 'loading', () => `menu ${suffix} remained loading`);
  if (menu.instance.data.listState !== 'ready') throw new Error(`menu ${suffix} ended ${menu.instance.data.listState}`);
  if (menu.instance.data.pickup.date !== date || menu.instance.data.pickup.time !== time) {
    tap(menu.querySelector('.pickup-bar'));
    await waitFor(() => menu.instance.data.pickerVisible, () => `picker ${suffix} did not open`);
    if (menu.instance.data.pickerDate !== date) {
      tap(menu.querySelectorAll('.pk-date').find(node => node.dom.dataset.date === date));
      await simulate.sleep(20);
    }
    const option = menu.querySelectorAll('.pk-time').find(node => node.dom.dataset.date === date
      && node.dom.dataset.period === 'dinner' && node.dom.dataset.t === time);
    tap(option);
    await waitFor(() => menu.instance.data.listState === 'ready' && menu.instance.data.pickup.date === date,
      () => `picker ${suffix} did not select ${date}/${time}`);
  }
  const choose = menu.querySelectorAll('.act-btn').find(node => node.dom.dataset.id === productID);
  tap(choose);
  await waitFor(() => menu.instance.data.czVisible, () => `product ${productID} did not open`);
  pages.menu.onCzConfirm.call(menu.instance, { detail: { qty: 1, flavors: [], note: '' } });
  await simulate.sleep(20);
  if (menu.instance.data.count !== 1 || !menu.instance.data.canCheckout) throw new Error(`menu ${suffix} did not visibly hold one orderable item`);
  const confirm = await openConfirm(suffix, `用户-${suffix}`);
  const payCount = paymentCalls.length;
  tap(confirm.querySelector('.pay-btn'));
  await waitFor(() => lastNavigation !== null || confirm.instance.data.paymentState === 'error',
    () => `order ${suffix} remained ${confirm.instance.data.paymentState}`);
  if (!lastNavigation || !lastNavigation.startsWith('/pages/result/result?id=') || paymentCalls.length !== payCount + 1
    || Object.keys(app.globalData.cart).length !== 0) throw new Error(`order ${suffix} failed at ${lastNavigation || confirm.instance.data.paymentState}`);
  const id = new URL(`http://ui1.local${lastNavigation}`).searchParams.get('id');
  const detail = await api('GET', `/api/v1/orders/${id}`, undefined, app.globalData.session.accessToken);
  return { id, state: detail.order.state, code: detail.order.pickup_number };
}

async function openHome(suffix) {
  document.body.innerHTML = '';
  const page = renderPage({ definition: pages.home, template: homeTemplate, id: `home-${suffix}`, usingComponents: globalComponents(`${suffix}-home`) });
  await waitFor(() => page.instance.data.settingsState !== 'loading' && page.instance.data.ordersState !== 'loading',
    () => `home ${suffix} remained ${page.instance.data.settingsState}/${page.instance.data.ordersState}`);
  return page;
}

async function openUserDetail(suffix, id) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pages['order-detail'], template: orderDetailTemplate, id: `user-detail-${suffix}`,
    usingComponents: globalComponents(`${suffix}-user-detail`, 'order'), loadOptions: { id },
  });
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `user detail ${suffix} remained loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`user detail ${suffix} ended ${page.instance.data.detailState}`);
  return page;
}

async function openMerchantDetail(suffix, id) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pages['admin-order-detail'], template: adminOrderDetailTemplate, id: `merchant-detail-${suffix}`,
    usingComponents: globalComponents(`${suffix}-merchant-detail`, 'order'), loadOptions: { id },
  });
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `merchant detail ${suffix} remained loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`merchant detail ${suffix} ended ${page.instance.data.detailState}`);
  return page;
}

async function openVerify(suffix) {
  document.body.innerHTML = '';
  const page = renderPage({ definition: pages['admin-verify'], template: adminVerifyTemplate, id: `verify-${suffix}`, usingComponents: globalComponents(`${suffix}-verify`, 'order') });
  await simulate.sleep(20);
  return page;
}

async function manualVerify(page, code, label) {
  page.querySelector('.manual-in').dispatchEvent('input', { detail: { value: code } });
  await waitFor(() => page.instance.data.code === code, () => `${label} code did not render`);
  const requestCount = observations.filter(item => item.path === '/api/v1/verify/code').length;
  tap(page.querySelector('.manual-btn'));
  await waitFor(() => observations.filter(item => item.path === '/api/v1/verify/code').length === requestCount + 1
    && !['idle', 'loading'].includes(page.instance.data.lookupState), () => `${label} verify remained ${page.instance.data.lookupState}`);
  if (page.instance.data.lookupState !== 'error' || page.instance.data.lastResult !== null
    || !page.dom.textContent.includes('核销失败，请核对取餐码、日期与订单状态后重试')) {
    throw new Error(`${label} verify produced false success: state=${page.instance.data.lookupState},result=${JSON.stringify(page.instance.data.lastResult)},text=${page.dom.textContent.slice(-160)}`);
  }
}

async function scanAndExpectFailure(page, label) {
  const requestCount = observations.filter(item => item.path === '/api/v1/verify/scan').length;
  tap(page.querySelector('.verify-center .btn'));
  await waitFor(() => observations.filter(item => item.path === '/api/v1/verify/scan').length === requestCount + 1
    && !['idle', 'loading'].includes(page.instance.data.lookupState), () => `${label} scan remained ${page.instance.data.lookupState}`);
  if (page.instance.data.lookupState !== 'error' || page.instance.data.lastResult !== null
    || !page.dom.textContent.includes('核销失败，请核对取餐码、日期与订单状态后重试')) {
    throw new Error(`${label} scan produced false success: state=${page.instance.data.lookupState},result=${JSON.stringify(page.instance.data.lastResult)},text=${page.dom.textContent.slice(-160)}`);
  }
}

async function acquirePCSession() {
  const login = await api('POST', '/api/v1/admin/auth/qrcode', {}, '', '', 201);
  const qr = new URL(login.qr_payload);
  await api('POST', '/api/v1/me/admin-login/approve', {
    login_id: login.login_id,
    approval_secret: qr.searchParams.get('approval_secret'),
    code: `${runID}-owner-approval`,
  }, app.globalData.session.accessToken);
  const poll = await api('POST', '/api/v1/admin/auth/poll', { login_id: login.login_id, poll_secret: login.poll_secret });
  if (poll.state !== 'APPROVED' || !poll.session || !poll.session.token) throw new Error('PC session did not approve');
  return poll.session.token;
}

function settings(value) {
  return {
    store_status: 'open', pickup_point: 'UI1临时取餐点', notice: '', pickup_step_min: 5,
    meal_periods: [
      { code: 'lunch', name: '午餐', cutoff_time: '00:00', pickup_from: '11:30', pickup_to: '11:30' },
      { code: 'dinner', name: '晚餐', cutoff_time: value.cutoffTime, pickup_from: value.pickupTime, pickup_to: value.pickupTime },
    ],
    service_dates: [{ date: value.today, status: 'open' }, { date: value.tomorrow, status: 'open' }],
  };
}

function runtimeTimes() {
  const now = new Date();
  const pickup = new Date(now.getTime() + 15 * 60000);
  const cutoff = new Date(now.getTime() + 7 * 60000);
  const today = shanghai(now).date;
  return {
    today,
    tomorrow: shanghai(new Date(now.getTime() + 24 * 60 * 60000)).date,
    targetDate: shanghai(pickup).date,
    pickupTime: shanghai(pickup).time,
    cutoffTime: shanghai(cutoff).time,
  };
}

function shanghai(date) {
  const parts = Object.fromEntries(new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  }).formatToParts(date).filter(part => part.type !== 'literal').map(part => [part.type, part.value]));
  return { date: `${parts.year}-${parts.month}-${parts.day}`, time: `${parts.hour}:${parts.minute}` };
}

async function api(method, pathname, body, bearer = '', key = '', expected = 200) {
  const headers = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (bearer) headers.Authorization = `Bearer ${bearer}`;
  if (key) headers['Idempotency-Key'] = key;
  const response = await fetch(`${proxyOrigin}${pathname}`, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  const raw = await response.text();
  let data = {};
  if (raw) { try { data = JSON.parse(raw); } catch { throw new Error(`${method} ${pathname} invalid JSON`); } }
  if (response.status !== expected) throw new Error(`${method} ${pathname} returned ${response.status}/${data.error && data.error.code || 'UNKNOWN'}, want ${expected}`);
  return data;
}

function tap(node) { if (!node) throw new Error('required rendered control was absent'); node.dispatchEvent('touchstart'); node.dispatchEvent('touchend'); }
function exactID(value, label) { if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw new Error(`${label} id malformed`); return value; }
function newKey(scope) { return `${scope}-${crypto.randomUUID()}`; }
async function waitFor(predicate, message) {
  const deadline = Date.now() + 8000;
  while (!predicate()) { if (Date.now() >= deadline) throw new Error(message()); await simulate.sleep(10); }
}
