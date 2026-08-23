/* global App, Behavior, Component, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const launchTemplate = require('../../../../apps/wechat-miniprogram/pages/launch/launch.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const resultTemplate = require('../../../../apps/wechat-miniprogram/pages/result/result.wxml');
const ordersTemplate = require('../../../../apps/wechat-miniprogram/pages/orders/orders.wxml');
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

let appDefinition;
let componentDefinition;
const pageDefinitions = {};
let registeringPage;
let lastNavigation = null;
const paymentCalls = [];

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
  request: options => {
    const requestURL = new URL(options.url);
    if (globalThis.__ui1ComposedHTTPFault || requestURL.pathname === globalThis.__ui1ComposedHTTPFaultPath) {
      requestURL.pathname = `${requestURL.pathname}/__composed_ui1_http_error__`;
    }
    fetch(requestURL.toString(), {
      method: options.method || 'GET',
      headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    })
      .then(async response => {
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

async function openCheckout(suffix, contactName, note) {
  document.body.innerHTML = '';
  lastNavigation = null;
  paymentCalls.length = 0;
  app.globalData.cart = {};
  app.globalData.pickup = null;
  app.onLaunch();
  await waitFor(
    () => app.globalData.session.state !== 'loading',
    () => `checkout session remained ${app.globalData.session.state}`,
  );
  if (app.globalData.session.state !== 'ready') throw new Error(`checkout session ended ${app.globalData.session.state}`);

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
  lastNavigation = null;
  return confirm;
}

describe('mini-program UI1 against composed local API and MySQL', () => {
  it('lets an anonymous user browse the storefront, pickup menu, categories, and products', async () => {
    document.body.innerHTML = '';
    lastNavigation = null;
    app.onLaunch();
    const suffix = `composed-${Date.now()}`;
    const common = globalComponents(suffix);
    const launch = renderPage({ definition: pageDefinitions.launch, template: launchTemplate, id: `launch-${suffix}`, usingComponents: common });

    await waitFor(
      () => launch.instance.data.storefrontState !== 'loading',
      () => `storefront remained ${launch.instance.data.storefrontState}`,
    );
    if (launch.instance.data.storefrontState !== 'ready') {
      throw new Error(`real storefront ended ${launch.instance.data.storefrontState}`);
    }
    await waitFor(
      () => app.globalData.session.state !== 'loading',
      () => `silent session remained ${app.globalData.session.state}`,
    );
    if (app.globalData.session.state !== 'ready') {
      throw new Error(`real silent session ended ${app.globalData.session.state}`);
    }
    if (!launch.dom.textContent.includes('绥安食品')) throw new Error(`real store name was absent: ${launch.dom.textContent}`);

    const userEntry = launch.querySelector('.id-card.primary');
    if (!userEntry) throw new Error('anonymous user entry was not rendered');
    if (userEntry.dom.closest('button[open-type="getPhoneNumber"]')) {
      throw new Error('anonymous user entry was nested in a phone authorization control');
    }
    userEntry.dispatchEvent('touchstart');
    userEntry.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== '/pages/home/home') throw new Error(`user entry navigated to ${lastNavigation || 'nothing'}`);

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
