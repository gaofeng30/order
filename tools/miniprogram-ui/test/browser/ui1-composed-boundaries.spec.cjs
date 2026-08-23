/* global App, Behavior, Component, ORDER_BOUNDARIES_PROXY_ORIGIN, ORDER_BOUNDARIES_RUN_ID, ORDER_BOUNDARIES_SETUP, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const profileTemplate = require('../../../../apps/wechat-miniprogram/pages/profile/profile.wxml');
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const setup = ORDER_BOUNDARIES_SETUP;
const proxyOrigin = ORDER_BOUNDARIES_PROXY_ORIGIN;
const runID = ORDER_BOUNDARIES_RUN_ID;
const pageDefinitions = {};
const observations = [];
const quotes = [];
const paymentCalls = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let loginCode = `${runID}-browser-1`;
let lastNavigation = null;
let afterQuoteHook = null;
let malformedPath = '';
let unauthorizedPath = '';
let unavailablePath = '';
let forceUnboundPhone = false;
let filterReadyOrders = false;

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: loginCode })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  requestPayment: options => {
    paymentCalls.push({ package: options.package, paySign: options.paySign });
    queueMicrotask(() => options.success({ errMsg: 'requestPayment:ok' }));
  },
  request: options => {
    const requestURL = new URL(options.url);
    const headers = Object.assign({}, options.header || {});
    if (requestURL.pathname === unauthorizedPath) headers.Authorization = 'Bearer invalid-boundary-token';
    if (requestURL.pathname === unavailablePath) headers['X-Boundary-Force-Status'] = '503';
    fetch(requestURL.toString(), {
      method: options.method || 'GET', headers,
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let body = {};
      if (raw) {
        try { body = JSON.parse(raw); } catch (error) { body = {}; }
      }
      observations.push({ method: options.method || 'GET', path: requestURL.pathname, status: response.status });
      if (requestURL.pathname === '/api/v1/quotes' && response.status === 201 && body.quote) {
        quotes.push(body.quote);
        if (afterQuoteHook) {
          const hook = afterQuoteHook;
          afterQuoteHook = null;
          await hook(body.quote);
        }
      }
      let projected = requestURL.pathname === malformedPath ? {} : body;
      if (forceUnboundPhone && requestURL.pathname === '/api/v1/me/primary-phone' && response.status === 200) {
        projected = { primary_phone_bound: false };
      }
      if (filterReadyOrders && requestURL.pathname === '/api/v1/orders' && requestURL.searchParams.get('active') === 'true'
        && projected && Array.isArray(projected.orders)) {
        projected = Object.assign({}, projected, { orders: projected.orders.filter(order => order.state !== 'READY_FOR_PICKUP') });
      }
      options.success({ statusCode: response.status, data: projected });
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
require('../../../../apps/wechat-miniprogram/components/stepper/stepper.js');
const stepperDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/customize/customize.js');
const customizeDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/app.js');
registeringPage = 'home';
require('../../../../apps/wechat-miniprogram/pages/home/home.js');
registeringPage = 'menu';
require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'confirm';
require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.js');
registeringPage = 'profile';
require('../../../../apps/wechat-miniprogram/pages/profile/profile.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/home/home' }];

describe('Mini user fail-closed boundaries against composed root HTTP and MySQL', () => {
  it('covers BE-01..06 and BE-22..26 through rendered Chrome controls', async () => {
    await freshSession('initial');

    const firstMenu = await openMenu('be02-partial');
    if (firstMenu.instance.data.pickup.date !== setup.dates[0]
      || firstMenu.instance.data.pickup.mealPeriod !== 'dinner') {
      throw new Error(`BE-02 partial-cutoff default was ${JSON.stringify(firstMenu.instance.data.pickup)}`);
    }
    await openPicker(firstMenu);
    if (!firstMenu.dom.textContent.includes('已截单 · 00:00 截止')
      || firstMenu.querySelectorAll('.pk-time').some(node => node.dom.dataset.t === '11:30')) {
      throw new Error('BE-02 cut-off lunch was not visibly folded as one group');
    }

    await writeSettings(fullCutoffSettings('open'));
    const fullCutoffMenu = await openMenu('be02-full');
    if (fullCutoffMenu.instance.data.pickup.date !== setup.dates[1]
      || fullCutoffMenu.instance.data.pickerDates[0].available !== false) {
      throw new Error(`BE-02 full cutoff did not default to tomorrow: ${JSON.stringify(fullCutoffMenu.instance.data.pickup)}`);
    }

    const beforeEmpty = observations.length;
    lastNavigation = null;
    tap(fullCutoffMenu.querySelector('.cart-go'));
    await simulate.sleep(20);
    if (lastNavigation !== null || observations.slice(beforeEmpty).some(item => ['/api/v1/quotes', '/api/v1/orders/prepay'].includes(item.path))) {
      throw new Error('BE-25 empty cart navigated or wrote a quote/prepayment');
    }

    await chooseProduct(fullCutoffMenu, setup.products.all.id);
    if (Object.keys(app.globalData.cart).length !== 1) throw new Error('BE-01 setup cart was not created visibly');
    await writeSettings(fullCutoffSettings('closed'));
    const closedMenu = await openMenu('be01-closed');
    if (closedMenu.instance.data.listState !== 'ready' || closedMenu.instance.data.canCheckout !== false
      || closedMenu.instance.data.count !== 1 || !closedMenu.dom.textContent.includes(setup.products.all.name)) {
      throw new Error(`BE-01 closed browse state was ${closedMenu.instance.data.listState}/${closedMenu.instance.data.canCheckout}`);
    }
    const beforeClosed = observations.length;
    lastNavigation = null;
    tap(closedMenu.querySelector('.cart-go'));
    await simulate.sleep(20);
    if (lastNavigation !== null || Object.keys(app.globalData.cart).length !== 1
      || observations.slice(beforeClosed).some(item => ['/api/v1/quotes', '/api/v1/orders/prepay'].includes(item.path))) {
      throw new Error('BE-01 closed checkout was not fail-closed');
    }

    await writeSettings(fullCutoffSettings('open'));
    await admin('PUT', `/api/v1/admin/products/${setup.products.all.id}/soldout`, {
      service_date: setup.dates[1], sold_out: true,
    });
    const soldoutMenu = await openMenu('be03-soldout');
    const soldoutProduct = soldoutMenu.instance.data.groups.flatMap(group => group.products)
      .find(product => product.id === setup.products.all.id);
    if (!soldoutProduct || soldoutProduct.orderable || !soldoutMenu.dom.textContent.includes('已售罄')
      || soldoutMenu.instance.data.count !== 1) {
      throw new Error('BE-03 sold-out product was not visible, disabled, and retained in cart');
    }
    const soldoutConfirm = await openConfirm('be03-confirm', '边界用户');
    await bindPhone(soldoutConfirm, `${runID}-primary-1`);
    const soldoutStart = observations.length;
    tap(soldoutConfirm.querySelector('.pay-btn'));
    await waitFor(() => soldoutConfirm.instance.data.paymentState === 'error', () => `BE-03 payment remained ${soldoutConfirm.instance.data.paymentState}`);
    assertOnlyQuoteFailure('BE-03', soldoutStart, soldoutConfirm);
    await admin('PUT', `/api/v1/admin/products/${setup.products.all.id}/soldout`, {
      service_date: setup.dates[1], sold_out: false,
    });
    await admin('PUT', `/api/v1/admin/products/${setup.products.lunch.id}/status`, { status: 'OFF' });
    app.globalData.cart = {};
    const shelfMenu = await openMenu('be03-off');
    if (shelfMenu.dom.textContent.includes(setup.products.lunch.name)) throw new Error('BE-03 off-shelf product remained visible');
    await admin('PUT', `/api/v1/admin/products/${setup.products.lunch.id}/status`, { status: 'ON' });

    app.globalData.cart = {};
    const dinnerMenu = await openMenu('be04-dinner-source');
    await selectPickup(dinnerMenu, setup.dates[1], 'dinner', '23:59');
    await chooseProduct(dinnerMenu, setup.products.dinner.id);
    await selectPickup(dinnerMenu, setup.dates[1], 'lunch', '11:30');
    if (dinnerMenu.dom.textContent.includes(setup.products.dinner.name) || Object.keys(app.globalData.cart).length !== 1) {
      throw new Error('BE-04 lunch menu did not hide the dinner product while retaining the stale cart item');
    }
    const mismatchConfirm = await openConfirm('be04-confirm', '餐段边界');
    const mismatchStart = observations.length;
    tap(mismatchConfirm.querySelector('.pay-btn'));
    await waitFor(() => mismatchConfirm.instance.data.paymentState === 'error', () => `BE-04 payment remained ${mismatchConfirm.instance.data.paymentState}`);
    assertOnlyQuoteFailure('BE-04', mismatchStart, mismatchConfirm);

    app.globalData.cart = {};
    await writeSettings(setup.baseline_settings);
    const cutoffMenu = await openMenu('be05-source');
    await selectPickup(cutoffMenu, setup.dates[0], 'dinner', '23:59');
    if (cutoffMenu.instance.data.pickup.date !== setup.dates[0] || cutoffMenu.instance.data.pickup.mealPeriod !== 'dinner') {
      throw new Error(`BE-05 could not establish today's orderable dinner: ${JSON.stringify(cutoffMenu.instance.data.pickup)}`);
    }
    await chooseProduct(cutoffMenu, setup.products.all.id);
    const cutoffConfirm = await openConfirm('be05-confirm', '截单边界');
    await writeSettings(fullCutoffSettings('open'));
    const cutoffStart = observations.length;
    tap(cutoffConfirm.querySelector('.pay-btn'));
    await waitFor(() => cutoffConfirm.instance.data.paymentState === 'error', () => `BE-05 payment remained ${cutoffConfirm.instance.data.paymentState}`);
    assertOnlyQuoteFailure('BE-05', cutoffStart, cutoffConfirm);

    app.globalData.cart = {};
    const driftMenu = await openMenu('be06-source');
    await chooseProduct(driftMenu, setup.products.all.id);
    const driftConfirm = await openConfirm('be06-confirm', '变价边界');
    const originalPrice = setup.products.all.price_cents;
    afterQuoteHook = async () => admin('PUT', `/api/v1/admin/products/${setup.products.all.id}`, {
      name: setup.products.all.name,
      price_cents: originalPrice + 1,
      category_id: setup.category_id,
      meal_period: 'all',
      description: 'UI1 boundary all price drift',
      images: [],
    });
    const driftStart = observations.length;
    tap(driftConfirm.querySelector('.pay-btn'));
    await waitFor(() => driftConfirm.instance.data.paymentState === 'error', () => `BE-06 payment remained ${driftConfirm.instance.data.paymentState}`);
    const driftRequests = observations.slice(driftStart);
    if (!driftRequests.some(item => item.path === '/api/v1/quotes' && item.status === 201)
      || !driftRequests.some(item => item.path === '/api/v1/orders/prepay' && item.status === 409)
      || paymentCalls.length !== 0 || Object.keys(app.globalData.cart).length !== 1) {
      throw new Error(`BE-06 drift shield evidence was ${JSON.stringify(driftRequests)}`);
    }

    app.globalData.cart = {};
    await freshSession('unbound');
    const unboundMenu = await openMenu('be22-menu');
    await chooseProduct(unboundMenu, setup.products.all.id);
    forceUnboundPhone = true;
    const unboundConfirm = await openConfirm('be22-confirm', '未绑定用户', false);
    forceUnboundPhone = false;
    if (unboundConfirm.instance.data.phoneState !== 'unbound') throw new Error(`BE-22 phone state was ${unboundConfirm.instance.data.phoneState}`);
    const unboundStart = observations.length;
    tap(unboundConfirm.querySelector('.pay-btn'));
    await simulate.sleep(20);
    if (observations.slice(unboundStart).some(item => item.path === '/api/v1/quotes')
      || Object.keys(app.globalData.cart).length !== 1 || lastNavigation !== null) {
      throw new Error('BE-22 unbound checkout wrote a quote, cleared cart, or navigated');
    }
    await bindPhone(unboundConfirm, `${runID}-primary-2`);
    if (unboundConfirm.instance.data.phoneState !== 'bound' || Object.keys(app.globalData.cart).length !== 1) {
      throw new Error('BE-22 trusted binding did not return to the same checkout with cart intact');
    }

    const profile = await openProfile('be23-profile');
    await fillExtra(profile, setup.staff.phone, `${setup.staff.name}错`);
    tap(profile.querySelector('.extra-save'));
    await waitFor(() => profile.instance.data.extraState === 'unmatched', () => `BE-23 wrong name remained ${profile.instance.data.extraState}`);
    if (profile.instance.data.pricingKind !== 'VISITOR' || !profile.dom.textContent.includes('姓名未匹配员工名单')) {
      throw new Error('BE-23 phone-only match granted STAFF or omitted the visible correction');
    }
    await fillExtra(profile, setup.staff.phone, setup.staff.name);
    tap(profile.querySelector('.extra-save'));
    await waitFor(() => profile.instance.data.extraState === 'matched', () => `BE-23 exact name remained ${profile.instance.data.extraState}`);
    if (profile.instance.data.pricingKind !== 'STAFF' || !profile.dom.textContent.includes('已命中员工名单')) {
      throw new Error('BE-23 exact two-factor match did not visibly become STAFF');
    }

    await fillExtra(profile, setup.staff.phone, `${setup.staff.name}再错`);
    tap(profile.querySelector('.extra-save'));
    await waitFor(() => profile.instance.data.extraState === 'unmatched', () => `BE-24 visitor reset remained ${profile.instance.data.extraState}`);
    app.globalData.cart = {};
    const visitorMenu = await openMenu('be24-menu');
    await chooseProduct(visitorMenu, setup.products.all.id);
    const visitorConfirm = await openConfirm('be24-confirm', '访客原价');
    const quoteCount = quotes.length;
    const visitorPaymentCount = paymentCalls.length;
    lastNavigation = null;
    tap(visitorConfirm.querySelector('.pay-btn'));
    await waitFor(() => lastNavigation && lastNavigation.startsWith('/pages/result/result?id='), () => `BE-24 payment ended ${visitorConfirm.instance.data.paymentState}/${lastNavigation}`);
    const visitorQuote = quotes[quoteCount];
    if (!visitorQuote || visitorQuote.identity.kind !== 'VISITOR' || visitorQuote.discount_cents !== 0
      || visitorQuote.payable_cents !== visitorQuote.original_subtotal_cents
      || paymentCalls.length !== visitorPaymentCount + 1) {
      throw new Error(`BE-24 visitor quote was ${JSON.stringify(visitorQuote)}`);
    }

    filterReadyOrders = true;
    const home = await openHome('be26-home');
    filterReadyOrders = false;
    if (!home.instance.data.ongoing || home.instance.data.ongoing.ready !== false) {
      throw new Error(`BE-26 required a real non-READY order, got ${JSON.stringify(home.instance.data.ongoing)}`);
    }
    const pickupEntry = home.querySelectorAll('.grid-item').find(node => node.dom.dataset.k === 'pickup');
    lastNavigation = null;
    tap(pickupEntry);
    await simulate.sleep(20);
    if (lastNavigation !== null || !home.dom.textContent.includes('暂无可取餐订单')) {
      throw new Error(`BE-26 non-READY take-code entry navigated to ${lastNavigation || 'nothing without toast'}`);
    }

    unauthorizedPath = '/api/v1/orders';
    const unauthorizedHome = await openHome('failure-401');
    unauthorizedPath = '';
    if (unauthorizedHome.instance.data.ordersState !== 'error' || unauthorizedHome.instance.data.ongoing !== null) {
      throw new Error('real 401 rendered a false active-order success');
    }
    unavailablePath = '/api/v1/menu/pickup-options';
    app.globalData.pickup = null;
    const unavailableMenu = await openMenu('failure-503', 'error');
    unavailablePath = '';
    if (unavailableMenu.instance.data.listState !== 'error' || unavailableMenu.instance.data.canCheckout) {
      throw new Error('503 pickup facts rendered an orderable menu');
    }
    malformedPath = '/api/v1/menu/pickup-options';
    const malformedMenu = await openMenu('failure-unknown', 'error');
    malformedPath = '';
    if (malformedMenu.instance.data.listState !== 'error' || malformedMenu.instance.data.canCheckout) {
      throw new Error('unknown pickup facts rendered an orderable menu');
    }
  });
});

function globalComponents(suffix, includeMenu = false) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const components = {
    icon,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
    pill: registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' }),
  };
  if (includeMenu) {
    components.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
    components.stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
    components.customize = registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, money: components.money, stepper: components.stepper },
    });
  }
  return components;
}

async function freshSession(suffix) {
  document.body.innerHTML = '';
  loginCode = `${runID}-${suffix}-${Date.now()}`;
  app.globalData.session = { state: 'idle', accessToken: '', expiresAt: '' };
  app.globalData.pickup = null;
  app.globalData.cart = {};
  app.onLaunch();
  await waitFor(() => app.globalData.session.state !== 'loading', () => `session ${suffix} remained loading`);
  if (app.globalData.session.state !== 'ready') throw new Error(`session ${suffix} ended ${app.globalData.session.state}`);
  lastNavigation = null;
}

async function openMenu(suffix, expected = 'ready') {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pageDefinitions.menu, template: menuTemplate, id: `menu-${suffix}`,
    usingComponents: globalComponents(`${suffix}-menu`, true),
  });
  await waitFor(() => page.instance.data.listState !== 'loading', () => `menu ${suffix} remained loading`);
  if (page.instance.data.listState !== expected) throw new Error(`menu ${suffix} ended ${page.instance.data.listState}, expected ${expected}`);
  return page;
}

async function openConfirm(suffix, contactName, requireBound = true) {
  document.body.innerHTML = '';
  lastNavigation = null;
  const page = renderPage({
    definition: pageDefinitions.confirm, template: confirmTemplate, id: `confirm-${suffix}`,
    usingComponents: globalComponents(`${suffix}-confirm`, true),
  });
  await waitFor(() => page.instance.data.phoneState !== 'loading', () => `confirm ${suffix} phone remained loading`);
  if (requireBound && page.instance.data.phoneState !== 'bound') throw new Error(`confirm ${suffix} phone ended ${page.instance.data.phoneState}`);
  const contact = page.querySelector('.field-in');
  contact.dispatchEvent('input', { detail: { value: contactName } });
  await waitFor(() => page.instance.data.form.contact === contactName, () => `confirm ${suffix} contact did not update`);
  return page;
}

async function openProfile(suffix) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pageDefinitions.profile, template: profileTemplate, id: `profile-${suffix}`,
    usingComponents: globalComponents(`${suffix}-profile`),
  });
  await waitFor(() => page.instance.data.identityState !== 'loading', () => `profile ${suffix} remained loading`);
  if (page.instance.data.identityState !== 'ready') throw new Error(`profile ${suffix} ended ${page.instance.data.identityState}`);
  return page;
}

async function openHome(suffix) {
  document.body.innerHTML = '';
  const page = renderPage({
    definition: pageDefinitions.home, template: homeTemplate, id: `home-${suffix}`,
    usingComponents: globalComponents(`${suffix}-home`),
  });
  await waitFor(() => page.instance.data.settingsState !== 'loading' && page.instance.data.ordersState !== 'loading',
    () => `home ${suffix} remained ${page.instance.data.settingsState}/${page.instance.data.ordersState}`);
  return page;
}

async function chooseProduct(page, productID) {
  const control = page.querySelectorAll('.act-btn').find(node => node.dom.dataset.id === productID);
  if (!control || !control.dom.textContent.includes('选择')) throw new Error(`orderable product ${productID} was absent`);
  tap(control);
  await waitFor(() => page.instance.data.czVisible === true, () => `customize ${productID} did not open`);
  pageDefinitions.menu.onCzConfirm.call(page.instance, { detail: { qty: 1, flavors: [], note: '' } });
  await simulate.sleep(20);
}

async function openPicker(page) {
  tap(page.querySelector('.pickup-bar'));
  await waitFor(() => page.instance.data.pickerVisible === true, () => 'pickup picker did not open');
}

async function selectPickup(page, date, period, time) {
  await openPicker(page);
  if (page.instance.data.pickerDate !== date) {
    const dateControl = page.querySelectorAll('.pk-date').find(node => node.dom.dataset.date === date);
    tap(dateControl);
    await simulate.sleep(10);
  }
  const timeControl = page.querySelectorAll('.pk-time').find(node => node.dom.dataset.t === time
    && node.dom.dataset.period === period && node.dom.dataset.date === date);
  if (!timeControl) throw new Error(`pickup ${date}/${period}/${time} was absent`);
  tap(timeControl);
  await waitFor(() => page.instance.data.listState === 'ready'
    && page.instance.data.pickup.date === date && page.instance.data.pickup.time === time,
  () => `pickup remained ${JSON.stringify(page.instance.data.pickup)}`);
}

async function bindPhone(confirm, code) {
  const bound = await pageDefinitions.confirm.onGetPhoneNumber.call(confirm.instance, { detail: { code } });
  if (!bound || confirm.instance.data.phoneState !== 'bound') throw new Error(`phone bind ended ${confirm.instance.data.phoneState}`);
}

async function fillExtra(profile, phone, name) {
  const phoneInput = profile.querySelector('.extra-phone');
  const nameInput = profile.querySelector('.extra-name');
  if (!phoneInput || !nameInput) throw new Error('extra-phone rendered inputs were absent');
  phoneInput.dispatchEvent('input', { detail: { value: phone } });
  nameInput.dispatchEvent('input', { detail: { value: name } });
  await waitFor(() => profile.instance.data.extraForm.phone === phone && profile.instance.data.extraForm.name === name,
    () => `extra form was ${JSON.stringify(profile.instance.data.extraForm)}`);
}

function assertOnlyQuoteFailure(caseID, start, confirm) {
  const requests = observations.slice(start);
  if (!requests.some(item => item.path === '/api/v1/quotes' && item.status === 409)
    || requests.some(item => item.path === '/api/v1/orders/prepay')
    || Object.keys(app.globalData.cart).length !== 1 || paymentCalls.length !== 0
    || confirm.instance.data.paymentState !== 'error' || lastNavigation !== null) {
    throw new Error(`${caseID} quote failure shield was ${JSON.stringify(requests)}`);
  }
}

function fullCutoffSettings(status) {
  return Object.assign({}, setup.baseline_settings, {
    store_status: status,
    meal_periods: setup.baseline_settings.meal_periods.map(meal => Object.assign({}, meal, {
      cutoff_time: '00:00',
      pickup_from: meal.code === 'lunch' ? '11:30' : '23:59',
      pickup_to: meal.code === 'lunch' ? '11:30' : '23:59',
    })),
  });
}

function writeSettings(value) { return admin('PUT', '/api/v1/admin/settings', settingsBody(value)); }
function settingsBody(value) {
  return {
    store_status: value.store_status,
    pickup_point: value.pickup_point,
    notice: value.notice,
    pickup_step_min: value.pickup_step_min,
    meal_periods: value.meal_periods,
    service_dates: value.service_dates,
  };
}

async function admin(method, pathname, body) {
  const headers = { Accept: 'application/json', Authorization: `Bearer ${setup.pc_token}` };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (method !== 'GET') headers['Idempotency-Key'] = `boundary-${crypto.randomUUID()}`;
  const response = await fetch(`${proxyOrigin}${pathname}`, {
    method, headers, body: body === undefined ? undefined : JSON.stringify(body),
  });
  const raw = await response.text();
  let data = {};
  if (raw) {
    try { data = JSON.parse(raw); } catch (error) { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  }
  if (!response.ok) throw new Error(`${method} ${pathname} returned ${response.status}/${data && data.error && data.error.code || 'UNKNOWN'}`);
  return data;
}

function tap(node) {
  if (!node) throw new Error('required rendered control was absent');
  node.dispatchEvent('touchstart');
  node.dispatchEvent('touchend');
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 5000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}
