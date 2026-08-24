/* global App, Behavior, Component, ORDER_COMPOSED_ENTRY_EXPECTATION, ORDER_COMPOSED_FLOW, ORDER_COMPOSED_MERCHANT_SETUP, ORDER_COMPOSED_PAYMENT_EXPECTATION, ORDER_COMPOSED_RUN_ID, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const adminOrderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.wxml');
const adminOrdersTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.wxml');
const adminProductsTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-products/admin-products.wxml');
const adminVerifyTemplate = require('../../../../apps/wechat-miniprogram/pages/admin-verify/admin-verify.wxml');
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const launchTemplate = require('../../../../apps/wechat-miniprogram/pages/launch/launch.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const resultTemplate = require('../../../../apps/wechat-miniprogram/pages/result/result.wxml');
const ordersTemplate = require('../../../../apps/wechat-miniprogram/pages/orders/orders.wxml');
const orderDetailTemplate = require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.wxml');
const profileTemplate = require('../../../../apps/wechat-miniprogram/pages/profile/profile.wxml');
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

let appDefinition;
let componentDefinition;
const pageDefinitions = {};
let registeringPage;
let lastNavigation = null;
const paymentCalls = [];
const confirmRequestKeys = [];
const confirmResponseStatuses = [];
const paymentExpectation = ORDER_COMPOSED_PAYMENT_EXPECTATION;
const composedFlow = ORDER_COMPOSED_FLOW;
const entryExpectation = ORDER_COMPOSED_ENTRY_EXPECTATION;
const merchantSetup = ORDER_COMPOSED_MERCHANT_SETUP;
const composedRunID = ORDER_COMPOSED_RUN_ID;
const requestObservations = [];
const subscribeCalls = [];
const subscribeDecisions = [];
let scanToken = '';

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: 'ui1-composed-login-code' })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  scanCode: options => queueMicrotask(() => {
    if (scanToken) options.success({ result: scanToken });
    else options.fail({ errMsg: 'scanCode:fail no composed token' });
  }),
  request: options => {
    const requestURL = new URL(options.url);
    const isConfirmRequest = requestURL.pathname === '/api/v1/orders/confirm';
    if (isConfirmRequest) {
      confirmRequestKeys.push((options.header || {})['Idempotency-Key'] || '');
    }
    if (globalThis.__ui1ComposedHTTPFault || requestURL.pathname === globalThis.__ui1ComposedHTTPFaultPath) {
      requestURL.pathname = `${requestURL.pathname}/__composed_ui1_http_error__`;
    }
    fetch(requestURL.toString(), {
      method: options.method || 'GET',
      headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    })
      .then(async response => {
        requestObservations.push({ method: options.method || 'GET', path: requestURL.pathname, status: response.status });
        if (isConfirmRequest) confirmResponseStatuses.push(response.status);
        const raw = await response.text();
        let data = {};
        if (raw) {
          try { data = JSON.parse(raw); } catch (error) { data = {}; }
        }
        options.success({ statusCode: response.status, data });
      })
      .catch(error => options.fail(error));
  },
  requestPayment: options => {
    paymentCalls.push({
      timeStamp: options.timeStamp,
      nonceStr: options.nonceStr,
      package: options.package,
      signType: options.signType,
      paySign: options.paySign,
    });
    queueMicrotask(() => options.success({ errMsg: 'requestPayment:ok' }));
  },
  requestSubscribeMessage: options => {
    subscribeCalls.push({ tmplIds: (options.tmplIds || []).slice() });
    const decision = subscribeDecisions.shift();
    queueMicrotask(() => {
      if (!decision || decision === 'fail') options.fail({ errMsg: 'requestSubscribeMessage:fail composed seam' });
      else options.success({ [options.tmplIds[0]]: decision });
    });
  },
  getRandomValues: bytes => {
    crypto.getRandomValues(bytes);
    return bytes;
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
registeringPage = 'launch';
require('../../../../apps/wechat-miniprogram/pages/launch/launch.js');
registeringPage = 'home';
require('../../../../apps/wechat-miniprogram/pages/home/home.js');
registeringPage = 'menu';
require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'confirm';
require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.js');
registeringPage = 'result';
require('../../../../apps/wechat-miniprogram/pages/result/result.js');
registeringPage = 'orders';
require('../../../../apps/wechat-miniprogram/pages/orders/orders.js');
registeringPage = 'order-detail';
require('../../../../apps/wechat-miniprogram/pages/order-detail/order-detail.js');
registeringPage = 'profile';
require('../../../../apps/wechat-miniprogram/pages/profile/profile.js');
registeringPage = 'admin-orders';
require('../../../../apps/wechat-miniprogram/pages/admin-orders/admin-orders.js');
registeringPage = 'admin-order-detail';
require('../../../../apps/wechat-miniprogram/pages/admin-order-detail/admin-order-detail.js');
registeringPage = 'admin-verify';
require('../../../../apps/wechat-miniprogram/pages/admin-verify/admin-verify.js');
registeringPage = 'admin-products';
require('../../../../apps/wechat-miniprogram/pages/admin-products/admin-products.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/home/home' }];

function globalComponents(suffix, includeMenuComponents = false, includeOrderComponents = false) {
  const iconID = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const components = {
    icon: iconID,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon: iconID } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon: iconID } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon: iconID } }),
  };
  if (includeMenuComponents || includeOrderComponents) {
    components.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  }
  if (includeMenuComponents) {
    components.stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
    const imagephID = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
    components.imageph = imagephID;
    components.customize = registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon: iconID, imageph: imagephID, money: components.money, stepper: components.stepper },
    });
  }
  if (includeOrderComponents) {
    components.pill = registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' });
    components.qrcode = registerComponent({ definition: qrcodeDefinition, template: qrcodeTemplate, id: `qrcode-${suffix}`, tagName: 'qrcode' });
  }
  return components;
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 3000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}

async function openCheckout(suffix, contactName, note, selectedPickupTime = '', orderNote = '') {
  document.body.innerHTML = '';
  lastNavigation = null;
  paymentCalls.length = 0;
  confirmRequestKeys.length = 0;
  confirmResponseStatuses.length = 0;
  app.globalData.cart = {};
  app.globalData.pickup = null;
  app.onLaunch();
  await waitFor(
    () => app.globalData.session.state !== 'loading',
    () => `checkout session remained ${app.globalData.session.state}`,
  );
  if (app.globalData.session.state !== 'ready') throw new Error(`checkout session ended ${app.globalData.session.state}`);
  if (composedFlow === 'consent') {
    app.globalData.subscriptionTemplateIds = { READY: 'ui1-ready-template', REFUND_RESULT: 'ui1-refund-template' };
  }

  const menu = renderPage({
    definition: pageDefinitions.menu,
    template: menuTemplate,
    id: `menu-${suffix}`,
    usingComponents: globalComponents(`${suffix}-menu`, true),
  });
  await waitFor(
    () => menu.instance.data.listState !== 'loading',
    () => `checkout menu remained ${menu.instance.data.listState}`,
  );
  if (menu.instance.data.listState !== 'ready') throw new Error(`checkout menu ended ${menu.instance.data.listState}`);
  if (selectedPickupTime && menu.instance.data.pickup.time !== selectedPickupTime) {
    const pickupBar = menu.querySelector('.pickup-bar');
    pickupBar.dispatchEvent('touchstart');
    pickupBar.dispatchEvent('touchend');
    await waitFor(
      () => menu.instance.data.pickerVisible === true,
      () => `pickup picker remained ${menu.instance.data.pickerVisible}`,
    );
    const target = menu.querySelectorAll('.pk-time').find(control => control.dom.dataset.t === selectedPickupTime);
    if (!target) throw new Error(`pickup option ${selectedPickupTime} was absent`);
    target.dispatchEvent('touchstart');
    target.dispatchEvent('touchend');
    await waitFor(
      () => menu.instance.data.listState === 'ready' && menu.instance.data.pickup.time === selectedPickupTime,
      () => `pickup selection ended ${menu.instance.data.listState}/${menu.instance.data.pickup.time}`,
    );
  }
  const firstChoice = menu.querySelector('.act-btn');
  if (!firstChoice) throw new Error('checkout menu had no orderable product control');
  firstChoice.dispatchEvent('touchstart');
  firstChoice.dispatchEvent('touchend');
  await waitFor(
    () => menu.instance.data.czVisible === true,
    () => `product choice did not open customization: ${menu.instance.data.czVisible}`,
  );
  pageDefinitions.menu.onCzConfirm.call(menu.instance, { detail: { qty: 1, flavors: [], note } });
  await simulate.sleep(10);
  if (menu.instance.data.count !== 1 || Object.keys(app.globalData.cart).length !== 1) {
    throw new Error(`rendered cart count was ${menu.instance.data.count}`);
  }
  const checkout = menu.querySelector('.cart-go');
  checkout.dispatchEvent('touchstart');
  checkout.dispatchEvent('touchend');
  await simulate.sleep(10);
  if (lastNavigation !== '/pages/confirm/confirm') throw new Error(`cart navigated to ${lastNavigation || 'nothing'}`);

  const confirm = renderPage({
    definition: pageDefinitions.confirm,
    template: confirmTemplate,
    id: `confirm-${suffix}`,
    usingComponents: globalComponents(`${suffix}-confirm`, true),
  });
  await waitFor(
    () => confirm.instance.data.phoneState !== 'loading',
    () => `phone state remained ${confirm.instance.data.phoneState}`,
  );
  const bound = await pageDefinitions.confirm.onGetPhoneNumber.call(confirm.instance, { detail: { code: 'ui1-trusted-phone-code' } });
  if (!bound || confirm.instance.data.phoneState !== 'bound' || !confirm.instance.data.maskedPhone) {
    throw new Error(`trusted phone bind ended ${confirm.instance.data.phoneState}`);
  }
  const contact = confirm.querySelector('.field-in');
  contact.dispatchEvent('input', { detail: { value: contactName } });
  await waitFor(
    () => confirm.instance.data.form.contact === contactName,
    () => `contact input was ${confirm.instance.data.form.contact}`,
  );
  if (orderNote) {
    pageDefinitions.confirm.onInput.call(confirm.instance, {
      currentTarget: { dataset: { k: 'orderNote' } }, detail: { value: orderNote },
    });
    if (confirm.instance.data.form.orderNote !== orderNote) throw new Error('order note did not reach checkout state');
  }
  lastNavigation = null;
  return confirm;
}

if (composedFlow === 'customer' && paymentExpectation === 'success') describe('mini-program UI1 against composed local API and MySQL', () => {
  it('routes the server identity into user browsing, pickup menu, categories, and products', async () => {
    document.body.innerHTML = '';
    lastNavigation = null;
    app.onLaunch();
    const suffix = `composed-${Date.now()}`;
    const common = globalComponents(suffix);
    const launch = renderPage({ definition: pageDefinitions.launch, template: launchTemplate, id: `launch-${suffix}`, usingComponents: common });

    await waitFor(() => app.globalData.entryRouting.state !== 'loading',
      () => `server entry routing remained ${app.globalData.entryRouting.state}`);
    await waitFor(
      () => app.globalData.session.state !== 'loading',
      () => `silent session remained ${app.globalData.session.state}`,
    );
    if (app.globalData.session.state !== 'ready') {
      throw new Error(`real silent session ended ${app.globalData.session.state}`);
    }
    if (entryExpectation === 'user') {
      if (lastNavigation !== '/pages/home/home' || app.globalData.entryRouting.state !== 'user'
        || launch.querySelector('.cards') || launch.querySelector('button[open-type="getPhoneNumber"]')) {
        throw new Error(`unbound cold start ended ${lastNavigation}/${app.globalData.entryRouting.state}`);
      }
    } else {
      await waitFor(() => launch.instance.data.storefrontState !== 'loading',
        () => `bound storefront remained ${launch.instance.data.storefrontState}`);
      const userEntry = launch.querySelector('.id-card.primary');
      if (app.globalData.entryRouting.state !== 'merchant' || !userEntry || !launch.querySelector('.id-plain')
        || launch.querySelector('button[open-type="getPhoneNumber"]')) {
        throw new Error(`bound cold start ended ${lastNavigation}/${app.globalData.entryRouting.state}`);
      }
      const beforeSelection = requestObservations.length;
      const entered = pageDefinitions.launch.go.call(launch.instance, {
        currentTarget: { dataset: { to: userEntry.dom.dataset.to } },
      });
      if (!entered || lastNavigation !== '/pages/home/home' || requestObservations.length !== beforeSelection) {
        throw new Error(`bound user selection ended ${entered}/${lastNavigation || 'nothing'}`);
      }
    }

    const home = renderPage({ definition: pageDefinitions.home, template: homeTemplate, id: `home-${suffix}`, usingComponents: common });
    await waitFor(
      () => home.instance.data.settingsState !== 'loading',
      () => `home remained ${home.instance.data.settingsState}`,
    );
    if (home.instance.data.settingsState !== 'ready') throw new Error(`real home ended ${home.instance.data.settingsState}`);
    if (home.querySelector('button[open-type="getPhoneNumber"]')) throw new Error('home rendered phone authorization before checkout');

    const menuEntry = home.querySelector('.search');
    menuEntry.dispatchEvent('touchstart');
    menuEntry.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== '/pages/menu/menu') throw new Error(`home menu entry navigated to ${lastNavigation || 'nothing'}`);

    const menu = renderPage({ definition: pageDefinitions.menu, template: menuTemplate, id: `menu-${suffix}`, usingComponents: globalComponents(`${suffix}-menu`, true) });
    await waitFor(
      () => menu.instance.data.listState !== 'loading',
      () => `menu remained ${menu.instance.data.listState}`,
    );
    if (menu.instance.data.optionState !== 'ready' || menu.instance.data.listState !== 'ready') {
      throw new Error(`real menu ended option=${menu.instance.data.optionState} list=${menu.instance.data.listState}`);
    }
    const categories = menu.querySelectorAll('.seg');
    const products = menu.querySelectorAll('.act-btn');
    if (categories.length < 2) throw new Error(`real category count was ${categories.length}`);
    if (products.length < 3) throw new Error(`real product count was ${products.length}`);
    if (!menu.dom.textContent.includes('本地午餐套餐') || !menu.dom.textContent.includes('无糖饮品')) {
      throw new Error(`real seeded products were absent: ${menu.dom.textContent}`);
    }
  });

  it('keeps the user in place when the composed API returns an HTTP error', async () => {
    document.body.innerHTML = '';
    lastNavigation = null;
    globalThis.__ui1ComposedHTTPFault = true;
    try {
      app.onLaunch();
      const suffix = `composed-http-error-${Date.now()}`;
      const launch = renderPage({
        definition: pageDefinitions.launch,
        template: launchTemplate,
        id: `launch-${suffix}`,
        usingComponents: globalComponents(suffix),
      });
      await waitFor(
        () => launch.instance.data.storefrontState !== 'loading',
        () => `faulted storefront remained ${launch.instance.data.storefrontState}`,
      );
      if (launch.instance.data.storefrontState !== 'error') {
        throw new Error(`faulted storefront rendered ${launch.instance.data.storefrontState}`);
      }
      const result = pageDefinitions.launch.go.call(launch.instance, { currentTarget: { dataset: { to: 'home' } } });
      if (result !== false || lastNavigation !== null) {
        throw new Error(`HTTP failure navigated to ${lastNavigation || 'an unknown success state'}`);
      }
      if (!launch.dom.textContent.includes('门店信息加载失败')) {
        throw new Error(`HTTP failure did not render an error state: ${launch.dom.textContent}`);
      }
    } finally {
      globalThis.__ui1ComposedHTTPFault = false;
    }
  });

  it('binds a trusted phone and completes cart, payment, result, list, detail, and eligible refund', async () => {
    const suffix = `composed-transaction-${Date.now()}`;
    const confirm = await openCheckout(suffix, 'UI1验收用户', 'UI1 composed');
    const pay = confirm.querySelector('.pay-btn');
    pay.dispatchEvent('touchstart');
    pay.dispatchEvent('touchend');
    await waitFor(
      () => lastNavigation !== null || confirm.instance.data.paymentState === 'error',
      () => `payment remained ${confirm.instance.data.paymentState}`,
    );
    if (!lastNavigation || !lastNavigation.startsWith('/pages/result/result?id=')) {
      throw new Error(`payment ended ${confirm.instance.data.paymentState} at ${lastNavigation || 'no navigation'}`);
    }
    if (Object.keys(app.globalData.cart).length !== 0) throw new Error('created order did not clear the cart');
    if (paymentCalls.length !== 1) throw new Error(`requestPayment calls were ${paymentCalls.length}`);
    const orderID = new URL(`http://ui1.local${lastNavigation}`).searchParams.get('id');
    if (!/^[1-9]\d*$/.test(orderID || '')) throw new Error(`created order id was ${orderID || 'missing'}`);

    document.body.innerHTML = '';
    const result = renderPage({
      definition: pageDefinitions.result,
      template: resultTemplate,
      id: `result-${suffix}`,
      usingComponents: globalComponents(`${suffix}-result`, false, true),
      loadOptions: { id: orderID },
    });
    await waitFor(
      () => result.instance.data.state !== 'loading',
      () => `result remained ${result.instance.data.state}`,
    );
    if (result.instance.data.state !== 'ready' || !result.dom.textContent.includes('订单已创建')) {
      throw new Error(`result ended ${result.instance.data.state}: ${result.dom.textContent}`);
    }

    document.body.innerHTML = '';
    const orders = renderPage({
      definition: pageDefinitions.orders,
      template: ordersTemplate,
      id: `orders-${suffix}`,
      usingComponents: globalComponents(`${suffix}-orders`, false, true),
    });
    await waitFor(
      () => orders.instance.data.listState !== 'loading',
      () => `orders remained ${orders.instance.data.listState}`,
    );
    if (!orders.instance.data.list.some(order => order.id === orderID)) {
      throw new Error(`created order ${orderID} was absent from the real list`);
    }

    document.body.innerHTML = '';
    const detail = renderPage({
      definition: pageDefinitions['order-detail'],
      template: orderDetailTemplate,
      id: `order-detail-${suffix}`,
      usingComponents: globalComponents(`${suffix}-detail`, false, true),
      loadOptions: { id: orderID },
    });
    await waitFor(
      () => detail.instance.data.detailState !== 'loading',
      () => `order detail remained ${detail.instance.data.detailState}`,
    );
    if (detail.instance.data.detailState !== 'ready' || detail.instance.data.o.id !== orderID || !detail.instance.data.rows.length) {
      throw new Error(`real order detail ended ${detail.instance.data.detailState}`);
    }
    if (!detail.instance.data.canCancel || detail.instance.data.o.state !== 'RESERVED') {
      throw new Error(`created order was not eligible for the promised refund path: ${detail.instance.data.o.state}`);
    }

    const openCancel = detail.querySelector('.cancel-btn.on');
    openCancel.dispatchEvent('touchstart');
    openCancel.dispatchEvent('touchend');
    await waitFor(
      () => detail.instance.data.cancelSheet === true,
      () => `cancel sheet remained ${detail.instance.data.cancelSheet}`,
    );
    const confirmCancel = detail.querySelector('.cs-confirm');
    confirmCancel.dispatchEvent('touchstart');
    confirmCancel.dispatchEvent('touchend');
    await waitFor(
      () => detail.instance.data.o && detail.instance.data.o.state === 'REFUNDING',
      () => `cancel ended ${detail.instance.data.o && detail.instance.data.o.state}`,
    );

    const refundDeadline = Date.now() + 12000;
    while (detail.instance.data.o.state !== 'REFUNDED' && Date.now() < refundDeadline) {
      await simulate.sleep(250);
      await pageDefinitions['order-detail'].load.call(detail.instance);
    }
    if (detail.instance.data.o.state !== 'REFUNDED') {
      throw new Error(`local refund worker ended ${detail.instance.data.o.state}`);
    }
  });

  it('keeps the cart and never renders success when real confirm HTTP fails', async () => {
    const suffix = `composed-confirm-error-${Date.now()}`;
    const confirm = await openCheckout(suffix, 'UI1失败保护', 'confirm HTTP error');

    globalThis.__ui1ComposedHTTPFaultPath = '/api/v1/orders/confirm';
    try {
      const pay = confirm.querySelector('.pay-btn');
      pay.dispatchEvent('touchstart');
      pay.dispatchEvent('touchend');
      await waitFor(
        () => confirm.instance.data.paymentState === 'error' || lastNavigation !== null,
        () => `faulted confirm remained ${confirm.instance.data.paymentState}`,
      );
      if (confirm.instance.data.paymentState !== 'error') {
        throw new Error(`faulted confirm rendered ${confirm.instance.data.paymentState}`);
      }
      if (lastNavigation !== null) throw new Error(`faulted confirm navigated to ${lastNavigation}`);
      if (Object.keys(app.globalData.cart).length !== 1 || confirm.instance.data.count !== 1) {
        throw new Error('faulted confirm cleared the cart');
      }
      if (paymentCalls.length !== 1) throw new Error(`faulted requestPayment calls were ${paymentCalls.length}`);
    } finally {
      globalThis.__ui1ComposedHTTPFaultPath = '';
    }
  });
});

if (composedFlow === 'consent') describe('mini-program UI1 consent and profile against composed local API and MySQL', () => {
  it('persists READY/refund decisions, drives their notifications, and keeps merchant identity server-owned', async () => {
    if (!composedRunID || !merchantSetup.pickup_time || !merchantSetup.far_pickup_time) {
      throw new Error('consent composed setup was incomplete');
    }
    const payOrder = async (suffix, contact, pickupTime, orderNote) => {
      const confirm = await openCheckout(suffix, contact, 'consent composed', pickupTime, orderNote);
      const pay = confirm.querySelector('.pay-btn');
      pay.dispatchEvent('touchstart');
      pay.dispatchEvent('touchend');
      await waitFor(
        () => lastNavigation !== null || confirm.instance.data.paymentState === 'error',
        () => `consent payment remained ${confirm.instance.data.paymentState}`,
      );
      if (!lastNavigation || !lastNavigation.startsWith('/pages/result/result?id=')) {
        throw new Error(`consent payment ended ${confirm.instance.data.paymentState} at ${lastNavigation || 'no navigation'}`);
      }
      const orderID = new URL(`http://ui1.local${lastNavigation}`).searchParams.get('id');
      if (!/^[1-9]\d*$/.test(orderID || '')) throw new Error(`consent order id was ${orderID || 'missing'}`);
      return orderID;
    };
    const waitRefunded = async detail => {
      const deadline = Date.now() + 12000;
      while (detail.instance.data.o.state !== 'REFUNDED' && Date.now() < deadline) {
        await simulate.sleep(250);
        await pageDefinitions['order-detail'].load.call(detail.instance);
      }
      if (detail.instance.data.o.state !== 'REFUNDED') {
        throw new Error(`consent refund worker ended ${detail.instance.data.o.state}`);
      }
    };

    const readyID = await payOrder(
      `${composedRunID}-ready`, 'UI1订阅备好用户', merchantSetup.pickup_time, `${composedRunID}-ready`,
    );
    subscribeDecisions.push('accept');
    document.body.innerHTML = '';
    const result = renderPage({
      definition: pageDefinitions.result,
      template: resultTemplate,
      id: `consent-result-${composedRunID}`,
      usingComponents: globalComponents(`${composedRunID}-result`, false, true),
      loadOptions: { id: readyID },
    });
    await waitFor(
      () => result.instance.data.state !== 'loading'
        && requestObservations.some(item => item.path === `/api/v1/orders/${readyID}/subscriptions`),
      () => `READY consent result ended ${result.instance.data.state}`,
    );
    if (result.instance.data.state !== 'ready' || result.instance.data.o.state !== 'PREPARING') {
      throw new Error(`READY consent order ended ${result.instance.data.state}/${result.instance.data.o && result.instance.data.o.state}`);
    }
    const readyConsent = requestObservations.find(item => item.path === `/api/v1/orders/${readyID}/subscriptions`);
    if (!readyConsent || readyConsent.status !== 200 || subscribeCalls.length !== 1
      || subscribeCalls[0].tmplIds[0] !== 'ui1-ready-template') {
      throw new Error(`READY consent was not durably recorded: ${readyConsent && readyConsent.status}`);
    }

    document.body.innerHTML = '';
    const merchantReady = renderPage({
      definition: pageDefinitions['admin-order-detail'],
      template: adminOrderDetailTemplate,
      id: `consent-ready-${composedRunID}`,
      usingComponents: globalComponents(`${composedRunID}-ready`, false, true),
      loadOptions: { id: readyID },
    });
    await waitFor(
      () => merchantReady.instance.data.detailState !== 'loading',
      () => `consent merchant detail remained ${merchantReady.instance.data.detailState}`,
    );
    const readyControl = merchantReady.querySelector('.foot-main');
    readyControl.dispatchEvent('touchstart');
    readyControl.dispatchEvent('touchend');
    await waitFor(
      () => merchantReady.instance.data.o && merchantReady.instance.data.o.state === 'READY_FOR_PICKUP',
      () => `consent mark-ready ended ${merchantReady.instance.data.o && merchantReady.instance.data.o.state}`,
    );

    const acceptedRefundID = await payOrder(
      `${composedRunID}-refund-accept`, 'UI1订阅退款用户', merchantSetup.far_pickup_time, `${composedRunID}-refund-accept`,
    );
    document.body.innerHTML = '';
    const acceptedDetail = renderPage({
      definition: pageDefinitions['order-detail'], template: orderDetailTemplate,
      id: `consent-refund-accept-${composedRunID}`,
      usingComponents: globalComponents(`${composedRunID}-refund-accept`, false, true),
      loadOptions: { id: acceptedRefundID },
    });
    await waitFor(
      () => acceptedDetail.instance.data.detailState !== 'loading',
      () => `accepted refund detail remained ${acceptedDetail.instance.data.detailState}`,
    );
    if (!acceptedDetail.instance.data.canCancel || acceptedDetail.instance.data.o.state !== 'RESERVED') {
      throw new Error(`accepted refund order was ${acceptedDetail.instance.data.o.state}/${acceptedDetail.instance.data.canCancel}`);
    }
    subscribeDecisions.push('accept');
    const acceptedBefore = requestObservations.length;
    const acceptedCancel = acceptedDetail.querySelector('.cancel-btn.on');
    acceptedCancel.dispatchEvent('touchstart');
    acceptedCancel.dispatchEvent('touchend');
    await waitFor(
      () => acceptedDetail.instance.data.cancelSheet === true && !!acceptedDetail.querySelector('.cs-confirm'),
      () => `accepted refund sheet remained ${acceptedDetail.instance.data.cancelSheet}`,
    );
    const acceptedConfirm = acceptedDetail.querySelector('.cs-confirm');
    acceptedConfirm.dispatchEvent('touchstart');
    acceptedConfirm.dispatchEvent('touchend');
    await waitFor(
      () => acceptedDetail.instance.data.o && acceptedDetail.instance.data.o.state === 'REFUNDING',
      () => `accepted refund cancel ended ${acceptedDetail.instance.data.o && acceptedDetail.instance.data.o.state}`,
    );
    const acceptedWrites = requestObservations.slice(acceptedBefore)
      .filter(item => item.path === `/api/v1/orders/${acceptedRefundID}/subscriptions`
        || item.path === `/api/v1/orders/${acceptedRefundID}/cancel`);
    if (acceptedWrites.length !== 2 || !acceptedWrites[0].path.endsWith('/subscriptions')
      || acceptedWrites[0].status !== 200 || !acceptedWrites[1].path.endsWith('/cancel') || acceptedWrites[1].status !== 200) {
      throw new Error(`accepted refund write order was ${JSON.stringify(acceptedWrites)}`);
    }
    await waitRefunded(acceptedDetail);

    const rejectedRefundID = await payOrder(
      `${composedRunID}-refund-reject`, 'UI1拒绝退款提醒用户', merchantSetup.far_pickup_time, `${composedRunID}-refund-reject`,
    );
    document.body.innerHTML = '';
    const rejectedDetail = renderPage({
      definition: pageDefinitions['order-detail'], template: orderDetailTemplate,
      id: `consent-refund-reject-${composedRunID}`,
      usingComponents: globalComponents(`${composedRunID}-refund-reject`, false, true),
      loadOptions: { id: rejectedRefundID },
    });
    await waitFor(
      () => rejectedDetail.instance.data.detailState !== 'loading',
      () => `rejected refund detail remained ${rejectedDetail.instance.data.detailState}`,
    );
    subscribeDecisions.push('reject');
    const rejectedBefore = requestObservations.length;
    rejectedDetail.querySelector('.cancel-btn.on').dispatchEvent('touchstart');
    rejectedDetail.querySelector('.cancel-btn.on').dispatchEvent('touchend');
    await waitFor(
      () => rejectedDetail.instance.data.cancelSheet === true && !!rejectedDetail.querySelector('.cs-confirm'),
      () => `rejected refund sheet remained ${rejectedDetail.instance.data.cancelSheet}`,
    );
    rejectedDetail.querySelector('.cs-confirm').dispatchEvent('touchstart');
    rejectedDetail.querySelector('.cs-confirm').dispatchEvent('touchend');
    await waitFor(
      () => rejectedDetail.instance.data.o && rejectedDetail.instance.data.o.state === 'REFUNDING',
      () => `rejected refund cancel ended ${rejectedDetail.instance.data.o && rejectedDetail.instance.data.o.state}`,
    );
    const rejectedWrites = requestObservations.slice(rejectedBefore)
      .filter(item => item.path === `/api/v1/orders/${rejectedRefundID}/subscriptions`
        || item.path === `/api/v1/orders/${rejectedRefundID}/cancel`);
    if (rejectedWrites.length !== 2 || rejectedWrites.some(item => item.status !== 200)) {
      throw new Error(`rejected refund did not continue through cancel: ${JSON.stringify(rejectedWrites)}`);
    }
    await waitRefunded(rejectedDetail);

    document.body.innerHTML = '';
    lastNavigation = null;
    const profile = renderPage({
      definition: pageDefinitions.profile, template: profileTemplate,
      id: `consent-profile-${composedRunID}`,
      usingComponents: globalComponents(`${composedRunID}-profile`, false, true),
    });
    await waitFor(
      () => profile.instance.data.identityState !== 'loading',
      () => `merchant profile remained ${profile.instance.data.identityState}`,
    );
    if (profile.instance.data.identityState !== 'ready' || !profile.instance.data.merchantBound
      || !profile.querySelector('.switch-id') || !profile.querySelector('.merchant-login')) {
      throw new Error(`merchant profile was not server-bound: ${profile.instance.data.identityState}/${profile.instance.data.merchantBound}`);
    }
    profile.instance.setData({ merchantBound: false });
    await waitFor(
      () => !profile.querySelector('.switch-id') && !!profile.querySelector('.merchant-login'),
      () => 'ordinary-user projection still rendered identity switching',
    );
    profile.instance.setData({ merchantBound: true });
    await waitFor(
      () => !!profile.querySelector('.switch-id'),
      () => 'merchant-bound projection did not restore identity switching',
    );
    const loginRequestCount = requestObservations.filter(item => item.path === '/api/v1/me/merchant-login').length;
    const loggedIn = await pageDefinitions.profile.onMerchantPhone.call(profile.instance, { detail: { code: 'ui1-consent-merchant-code' } });
    if (!loggedIn || lastNavigation !== '/pages/launch/launch'
      || requestObservations.filter(item => item.path === '/api/v1/me/merchant-login').length !== loginRequestCount + 1) {
      throw new Error(`profile merchant login ended ${loggedIn}/${lastNavigation || 'no navigation'}`);
    }

    await simulate.sleep(1500);
  });
});

if (composedFlow === 'customer' && paymentExpectation === 'pending') describe('mini-program UI1 against composed local API and MySQL in pending mode', () => {
  it('keeps the cart, suppresses success, and rotates the confirm key after real pending responses', async () => {
    const suffix = `composed-pending-${Date.now()}`;
    const confirm = await openCheckout(suffix, 'UI1待确认用户', 'pending expectation');
    const pay = confirm.querySelector('.pay-btn');

    pay.dispatchEvent('touchstart');
    pay.dispatchEvent('touchend');
    await waitFor(
      () => confirmResponseStatuses.length === 1 && (confirm.instance.data.paymentState === 'pending' || lastNavigation !== null),
      () => `first confirm remained ${confirm.instance.data.paymentState}`,
    );
    if (confirmResponseStatuses[0] !== 202) throw new Error(`first confirm returned ${confirmResponseStatuses[0]}`);
    if (confirm.instance.data.paymentState !== 'pending') {
      throw new Error(`first confirm rendered ${confirm.instance.data.paymentState}`);
    }
    if (lastNavigation !== null) throw new Error(`first pending confirm navigated to ${lastNavigation}`);
    if (Object.keys(app.globalData.cart).length !== 1 || confirm.instance.data.count !== 1) {
      throw new Error('first pending confirm cleared the cart');
    }
    if (paymentCalls.length !== 1) throw new Error(`pending requestPayment calls were ${paymentCalls.length}`);
    if (confirmRequestKeys.length !== 1 || !confirmRequestKeys[0]) {
      throw new Error(`first confirm key count was ${confirmRequestKeys.length}`);
    }

    pay.dispatchEvent('touchstart');
    pay.dispatchEvent('touchend');
    await waitFor(
      () => confirmResponseStatuses.length === 2 && confirm.instance.data.paymentState === 'pending',
      () => `second confirm count/state was ${confirmResponseStatuses.length}/${confirm.instance.data.paymentState}`,
    );
    if (confirmResponseStatuses[1] !== 202) throw new Error(`second confirm returned ${confirmResponseStatuses[1]}`);
    if (!confirmRequestKeys[1] || confirmRequestKeys[1] === confirmRequestKeys[0]) {
      throw new Error('second pending confirm did not use a fresh idempotency key');
    }
    if (lastNavigation !== null) throw new Error(`second pending confirm navigated to ${lastNavigation}`);
    if (Object.keys(app.globalData.cart).length !== 1 || confirm.instance.data.count !== 1) {
      throw new Error('second pending confirm cleared the cart');
    }
    if (paymentCalls.length !== 1) throw new Error(`retry repeated requestPayment ${paymentCalls.length} times`);
  });
});

if (composedFlow === 'merchant') describe('mini-program UI1 merchant fulfillment against composed local API and MySQL', () => {
  it('creates a preparing order, marks it ready, scans it completed, and restores live controls', async () => {
    const suffix = `composed-merchant-${Date.now()}`;
    const confirm = await openCheckout(suffix, 'UI1商户联调用户', 'merchant composed');
    const pay = confirm.querySelector('.pay-btn');
    pay.dispatchEvent('touchstart');
    pay.dispatchEvent('touchend');
    await waitFor(
      () => lastNavigation !== null || confirm.instance.data.paymentState === 'error',
      () => `merchant setup payment remained ${confirm.instance.data.paymentState}`,
    );
    if (!lastNavigation || !lastNavigation.startsWith('/pages/result/result?id=')) {
      throw new Error(`merchant setup payment ended at ${lastNavigation || confirm.instance.data.paymentState}`);
    }
    const orderID = new URL(`http://ui1.local${lastNavigation}`).searchParams.get('id');
    if (!/^[1-9]\d*$/.test(orderID || '')) throw new Error(`merchant order id was ${orderID || 'missing'}`);
    if (paymentCalls.length !== 1 || Object.keys(app.globalData.cart).length !== 0) {
      throw new Error(`merchant payment/cart ended ${paymentCalls.length}/${Object.keys(app.globalData.cart).length}`);
    }

    document.body.innerHTML = '';
    const initialUserDetail = renderPage({
      definition: pageDefinitions['order-detail'],
      template: orderDetailTemplate,
      id: `merchant-user-initial-${suffix}`,
      usingComponents: globalComponents(`${suffix}-user-initial`, false, true),
      loadOptions: { id: orderID },
    });
    await waitFor(
      () => initialUserDetail.instance.data.detailState !== 'loading',
      () => `merchant initial user detail remained ${initialUserDetail.instance.data.detailState}`,
    );
    if (initialUserDetail.instance.data.o.state !== 'PREPARING' || initialUserDetail.instance.data.showQr) {
      throw new Error(`merchant order initial state/token was ${initialUserDetail.instance.data.o.state}/${initialUserDetail.instance.data.showQr}`);
    }
    if (initialUserDetail.instance.data.o.pickupDate !== merchantSetup.service_date
      || initialUserDetail.instance.data.o.pickupTime !== merchantSetup.pickup_time) {
      throw new Error(`merchant order pickup was ${initialUserDetail.instance.data.o.pickupDate}/${initialUserDetail.instance.data.o.pickupTime}`);
    }

    document.body.innerHTML = '';
    lastNavigation = null;
    const launch = renderPage({
      definition: pageDefinitions.launch,
      template: launchTemplate,
      id: `merchant-entry-${suffix}`,
      usingComponents: globalComponents(`${suffix}-entry`),
    });
    await waitFor(
      () => launch.instance.data.storefrontState !== 'loading',
      () => `merchant entry remained ${launch.instance.data.storefrontState}`,
    );
    if (!launch.querySelector('.id-plain') || launch.querySelector('button[open-type="getPhoneNumber"]')) {
      throw new Error('bound merchant entry repeated phone authorization or was not rendered');
    }
    const entered = pageDefinitions.launch.goMerchant.call(launch.instance);
    if (!entered || lastNavigation !== '/pages/admin-orders/admin-orders') {
      throw new Error(`merchant entry ended ${entered}/${lastNavigation || 'no navigation'}`);
    }

    document.body.innerHTML = '';
    lastNavigation = null;
    const merchantOrders = renderPage({
      definition: pageDefinitions['admin-orders'],
      template: adminOrdersTemplate,
      id: `merchant-orders-${suffix}`,
      usingComponents: globalComponents(`${suffix}-orders`, false, true),
      loadOptions: { lane: '制作中' },
    });
    await waitFor(
      () => merchantOrders.instance.data.listState !== 'loading',
      () => `merchant orders remained ${merchantOrders.instance.data.listState}`,
    );
    if (!merchantOrders.instance.data.list.some(order => order.id === orderID && order.state === 'PREPARING')) {
      throw new Error(`merchant PREPARING order ${orderID} was absent`);
    }
    const bizControls = merchantOrders.querySelectorAll('.biz-seg');
    const closed = bizControls.find(control => control.dom.dataset.b === 'closed');
    const open = bizControls.find(control => control.dom.dataset.b === 'open');
    if (!closed || !open) throw new Error('merchant store status controls were absent');
    closed.dispatchEvent('touchstart');
    closed.dispatchEvent('touchend');
    await waitFor(
      () => merchantOrders.instance.data.storeStatus === 'closed',
      () => `merchant store close ended ${merchantOrders.instance.data.storeStatus}`,
    );
    open.dispatchEvent('touchstart');
    open.dispatchEvent('touchend');
    await waitFor(
      () => merchantOrders.instance.data.storeStatus === 'open',
      () => `merchant store restore ended ${merchantOrders.instance.data.storeStatus}`,
    );
    const orderCard = merchantOrders.querySelectorAll('.aorder').find(card => card.dom.dataset.id === orderID);
    if (!orderCard) throw new Error(`merchant order card ${orderID} was absent`);
    orderCard.dispatchEvent('touchstart');
    orderCard.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== `/pages/admin-order-detail/admin-order-detail?id=${orderID}`) {
      throw new Error(`merchant order card navigated to ${lastNavigation || 'nothing'}`);
    }

    document.body.innerHTML = '';
    const merchantDetail = renderPage({
      definition: pageDefinitions['admin-order-detail'],
      template: adminOrderDetailTemplate,
      id: `merchant-detail-${suffix}`,
      usingComponents: globalComponents(`${suffix}-detail`, false, true),
      loadOptions: { id: orderID },
    });
    await waitFor(
      () => merchantDetail.instance.data.detailState !== 'loading',
      () => `merchant detail remained ${merchantDetail.instance.data.detailState}`,
    );
    if (merchantDetail.instance.data.o.state !== 'PREPARING') {
      throw new Error(`merchant detail state was ${merchantDetail.instance.data.o.state}`);
    }
    const ready = merchantDetail.querySelector('.foot-main');
    ready.dispatchEvent('touchstart');
    ready.dispatchEvent('touchend');
    await waitFor(
      () => merchantDetail.instance.data.o && merchantDetail.instance.data.o.state === 'READY_FOR_PICKUP',
      () => `merchant ready ended ${merchantDetail.instance.data.o && merchantDetail.instance.data.o.state}`,
    );
    if (merchantDetail.instance.data.meta.isView || merchantDetail.instance.data.meta.label !== '核销'
      || !merchantDetail.querySelector('.foot-main').dom.textContent.includes('核销')) {
      throw new Error('READY merchant detail did not expose direct redeem for cross-date orders');
    }

    document.body.innerHTML = '';
    const readyUserDetail = renderPage({
      definition: pageDefinitions['order-detail'],
      template: orderDetailTemplate,
      id: `merchant-user-ready-${suffix}`,
      usingComponents: globalComponents(`${suffix}-user-ready`, false, true),
      loadOptions: { id: orderID },
    });
    await waitFor(
      () => readyUserDetail.instance.data.detailState !== 'loading',
      () => `ready user detail remained ${readyUserDetail.instance.data.detailState}`,
    );
    scanToken = readyUserDetail.instance.data.qrToken;
    if (readyUserDetail.instance.data.o.state !== 'READY_FOR_PICKUP' || !readyUserDetail.instance.data.showQr || !scanToken) {
      throw new Error(`ready user token ended ${readyUserDetail.instance.data.o.state}/${readyUserDetail.instance.data.showQr}`);
    }

    document.body.innerHTML = '';
    lastNavigation = null;
    const verify = renderPage({
      definition: pageDefinitions['admin-verify'],
      template: adminVerifyTemplate,
      id: `merchant-verify-${suffix}`,
      usingComponents: globalComponents(`${suffix}-verify`, false, true),
    });
    const scan = verify.querySelector('.verify-center .btn');
    scan.dispatchEvent('touchstart');
    scan.dispatchEvent('touchend');
    await waitFor(
      () => verify.instance.data.lookupState !== 'loading' && verify.instance.data.lookupState !== 'idle',
      () => `merchant scan remained ${verify.instance.data.lookupState}`,
    );
    if (verify.instance.data.lookupState !== 'completed' || !verify.instance.data.lastResult
      || verify.instance.data.lastResult.id !== orderID || verify.instance.data.lastResult.state !== 'COMPLETED') {
      const scanRequest = requestObservations.filter(item => item.path === '/api/v1/verify/scan').at(-1);
      throw new Error(`merchant scan ended ${verify.instance.data.lookupState}/${scanRequest && scanRequest.status}`);
    }
    if (!verify.dom.textContent.includes('核销成功')) throw new Error('merchant scan did not render the completed result');
    const backToOrders = verify.querySelector('.vr-back-orders');
    backToOrders.dispatchEvent('touchstart');
    backToOrders.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== '/pages/admin-orders/admin-orders?lane=%E5%B7%B2%E5%AE%8C%E6%88%90') {
      throw new Error(`completed scan navigated to ${lastNavigation || 'nothing'}`);
    }

    document.body.innerHTML = '';
    const products = renderPage({
      definition: pageDefinitions['admin-products'],
      template: adminProductsTemplate,
      id: `merchant-products-${suffix}`,
      usingComponents: globalComponents(`${suffix}-products`, true, true),
    });
    await waitFor(
      () => products.instance.data.listState !== 'loading',
      () => `merchant products remained ${products.instance.data.listState}`,
    );
    const product = products.instance.data.list.find(item => item.id === merchantSetup.product_id);
    if (!product || product.soldOut !== merchantSetup.prepared_sold_out) {
      throw new Error(`merchant sold-out baseline was ${product && product.soldOut}`);
    }
    const soldout = products.querySelectorAll('.pa-soldout').find(control => control.dom.dataset.id === merchantSetup.product_id);
    if (!soldout) throw new Error(`merchant sold-out control ${merchantSetup.product_id} was absent`);
    soldout.dispatchEvent('touchstart');
    soldout.dispatchEvent('touchend');
    await waitFor(
      () => products.instance.data.list.find(item => item.id === merchantSetup.product_id).soldOut === true,
      () => 'merchant sold-out true did not persist',
    );
    soldout.dispatchEvent('touchstart');
    soldout.dispatchEvent('touchend');
    await waitFor(
      () => products.instance.data.list.find(item => item.id === merchantSetup.product_id).soldOut === false,
      () => 'merchant sold-out false did not persist',
    );
  });
});
