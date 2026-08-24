/* global App, Behavior, Component, ORDER_STAFF_PROFILE_SETUP, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const detailTemplate = require('../../../../apps/wechat-miniprogram/pages/detail/detail.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const profileTemplate = require('../../../../apps/wechat-miniprogram/pages/profile/profile.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const setup = ORDER_STAFF_PROFILE_SETUP;
const pages = {};
const observations = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let currentRoute = 'pages/menu/menu';
let identityMode = '';

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages[registeringPage] = definition; };
globalThis.wx = {
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  navigateTo: () => {}, redirectTo: () => {}, reLaunch: () => {}, navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  getUserProfile: options => queueMicrotask(() => options.fail({ errMsg: 'getUserProfile:fail user deny' })),
  request: options => {
    const url = new URL(options.url);
    const headers = Object.assign({}, options.header || {});
    if (url.pathname === '/api/v1/me/identity' && identityMode) headers['X-Staff-Profile-Identity-Mode'] = identityMode;
    fetch(url.toString(), {
      method: options.method || 'GET', headers,
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let body = {};
      if (raw) { try { body = JSON.parse(raw); } catch {} }
      observations.push({ method: options.method || 'GET', path: url.pathname, query: url.search, status: response.status, identity_mode: identityMode || undefined });
      options.success({ statusCode: response.status, data: body });
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
require('../../../../apps/wechat-miniprogram/components/stepper/stepper.js');
const stepperDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/customize/customize.js');
const customizeDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/app.js');
registeringPage = 'menu'; require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'detail'; require('../../../../apps/wechat-miniprogram/pages/detail/detail.js');
registeringPage = 'profile'; require('../../../../apps/wechat-miniprogram/pages/profile/profile.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
app.globalData.runtimeEndpoint = { state: 'ready', envVersion: 'develop', origin: setup.api_origin };
app.globalData.apiBaseUrl = setup.api_origin;
app.globalData.storefrontFlavors = setup.flavors.slice();
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: currentRoute }];

describe('PAGE-U03/U09 and AC-04/05 staff pricing composed closure', () => {
  it('keeps anonymous, unknown identity and trusted staff pricing fail closed against one root product', async () => {
    setAnonymous();
    const anonymousMenu = render('menu', menuTemplate, 'anonymous-menu');
    await readyMenu(anonymousMenu, 'anonymous menu');
    const anonymousMenuProduct = productFromMenu(anonymousMenu);
    assertVisitorPrice(anonymousMenuProduct, anonymousMenu, 'anonymous menu');
    const anonymousDetail = render('detail', detailTemplate, 'anonymous-detail', { id: setup.product.id });
    await readyDetail(anonymousDetail, 'anonymous detail');
    assertVisitorPrice(anonymousDetail.instance.data.m, anonymousDetail, 'anonymous detail');
    if (setup.visitor_quote.product_id !== setup.product.id
      || setup.visitor_quote.original_unit_price_cents !== setup.product.original_unit_price_cents
      || setup.visitor_quote.discounted_unit_price_cents !== setup.product.original_unit_price_cents
      || setup.visitor_quote.identity_kind !== 'VISITOR') {
      throw new Error(`visitor Quote was not the same original-price product: ${JSON.stringify(setup.visitor_quote)}`);
    }

    setStaff('503');
    const unavailableMenu = render('menu', menuTemplate, 'identity-503-menu');
    await readyMenu(unavailableMenu, 'identity 503 menu');
    assertVisitorPrice(productFromMenu(unavailableMenu), unavailableMenu, 'identity 503 menu');
    if (!observations.some(item => item.path === '/api/v1/me/identity' && item.status === 503 && item.identity_mode === '503')) {
      throw new Error('identity 503 was not observed before fail-closed pricing');
    }

    setStaff('malformed');
    app.globalData.pickup = {
      date: setup.pickup.date, mealPeriod: setup.pickup.meal_period, time: setup.pickup.time,
    };
    const malformedDetail = render('detail', detailTemplate, 'identity-malformed-detail', { id: setup.product.id });
    await readyDetail(malformedDetail, 'identity malformed detail');
    assertVisitorPrice(malformedDetail.instance.data.m, malformedDetail, 'identity malformed detail');
    if (!observations.some(item => item.path === '/api/v1/me/identity' && item.status === 200 && item.identity_mode === 'malformed')) {
      throw new Error('malformed identity was not observed before fail-closed pricing');
    }

    setStaff('');
    const staffMenu = render('menu', menuTemplate, 'staff-menu');
    await readyMenu(staffMenu, 'staff menu');
    const staffMenuProduct = productFromMenu(staffMenu);
    assertStaffPrice(staffMenuProduct, staffMenu, 'staff menu');
    const staffDetail = render('detail', detailTemplate, 'staff-detail', { id: setup.product.id });
    await readyDetail(staffDetail, 'staff detail');
    assertStaffPrice(staffDetail.instance.data.m, staffDetail, 'staff detail');

    const quoteStart = observations.length;
    const quote = await directJSON('POST', '/api/v1/quotes', setup.staff_quote_body, setup.staff_quote_key, 201);
    const projection = quote.quote;
    const line = projection && projection.items && projection.items[0];
    if (!projection || !line || projection.identity.kind !== 'STAFF' || line.product_id !== setup.product.id
      || line.original_unit_price_cents !== setup.product.original_unit_price_cents
      || line.discounted_unit_price_cents !== setup.product.staff_unit_price_cents
      || projection.payable_cents !== setup.product.staff_unit_price_cents * 2
      || staffMenuProduct.price_cents !== line.discounted_unit_price_cents
      || staffDetail.instance.data.m.price_cents !== line.discounted_unit_price_cents) {
      throw new Error(`menu/detail/Quote price drifted: ${JSON.stringify(projection)}`);
    }
    for (const tamper of setup.tamper_bodies) {
      const rejected = await directJSON('POST', '/api/v1/quotes', tamper.body, tamper.key, 400);
      if (!rejected.error || rejected.error.code !== 'INVALID_REQUEST') {
        throw new Error(`${tamper.kind} tamper returned ${JSON.stringify(rejected)}`);
      }
    }
    const quoteRequests = observations.slice(quoteStart).filter(item => item.path === '/api/v1/quotes');
    if (quoteRequests.length !== 4 || quoteRequests.filter(item => item.status === 201).length !== 1
      || quoteRequests.filter(item => item.status === 400).length !== 3) {
      throw new Error(`Quote/tamper request evidence was ${JSON.stringify(quoteRequests)}`);
    }
  });

  it('renders neutral cosmetic/profile/contact failure without HTTP or false identity', async () => {
    setStaff('503');
    const profile = render('profile', profileTemplate, 'profile-503');
    await waitFor(() => profile.instance.data.identityState !== 'loading', () => 'profile identity stayed loading');
    if (profile.instance.data.identityState !== 'error' || profile.instance.data.pricingKind !== 'VISITOR'
      || profile.instance.data.nick !== '微信用户' || profile.instance.data.avatarText !== '客'
      || profile.instance.data.avatarUrl !== '' || profile.dom.textContent.includes('员工价')
      || !profile.dom.textContent.includes('联系客服')) {
      throw new Error(`profile 503 exposed a false identity: ${profile.dom.textContent}`);
    }
    const before = observations.length;
    const selected = await pages.profile.chooseProfile.call(profile.instance);
    await simulate.sleep(10);
    if (selected !== false || observations.length !== before || profile.instance.data.nick !== '微信用户'
      || profile.instance.data.avatarText !== '客' || profile.instance.data.avatarUrl !== '') {
      throw new Error('profile rejection changed identity or issued business HTTP');
    }
    const contact = profile.querySelector('.prow.last');
    if (!contact) throw new Error('native contact surface was not rendered');
    await contact.dispatchEvent('touchstart');
    await simulate.sleep(10);
    if (observations.length !== before || !profile.dom.textContent.includes('联系客服')) {
      throw new Error('native contact failure issued business HTTP or removed the neutral surface');
    }

    setStaff('malformed');
    const malformed = render('profile', profileTemplate, 'profile-malformed');
    await waitFor(() => malformed.instance.data.identityState !== 'loading', () => 'malformed profile stayed loading');
    if (malformed.instance.data.identityState !== 'error' || malformed.instance.data.pricingKind !== 'VISITOR'
      || malformed.dom.textContent.includes('员工价') || !malformed.dom.textContent.includes('微信用户')) {
      throw new Error('malformed identity projected staff or removed the neutral placeholder');
    }
  });
});

function setAnonymous() {
  identityMode = '';
  app.globalData.session = { state: 'failed', accessToken: '' };
  app.globalData.pickup = null;
  app.globalData.cart = {};
}
function setStaff(mode) {
  identityMode = mode;
  app.globalData.session = { state: 'ready', accessToken: setup.staff_token };
  app.globalData.pickup = null;
  app.globalData.cart = {};
}
function render(name, template, suffix, loadOptions = {}) {
  document.body.innerHTML = '';
  currentRoute = `pages/${name}/${name}`;
  return renderPage({
    definition: pages[name], template, id: `${name}-${suffix}`,
    usingComponents: components(`${name}-${suffix}`), loadOptions,
  });
}
function components(suffix) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
  const money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  const stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
  return {
    icon, imageph, money, stepper,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    pill: registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
    customize: registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, imageph, money, stepper },
    }),
  };
}
async function readyMenu(page, label) {
  await waitFor(() => page.instance.data.listState !== 'loading', () => `${label} stayed loading`);
  if (page.instance.data.optionState !== 'ready' || page.instance.data.listState !== 'ready') {
    throw new Error(`${label} ended ${page.instance.data.optionState}/${page.instance.data.listState}`);
  }
}
async function readyDetail(page, label) {
  await waitFor(() => page.instance.data.detailState !== 'loading', () => `${label} stayed loading`);
  if (page.instance.data.detailState !== 'ready') throw new Error(`${label} ended ${page.instance.data.detailState}`);
}
function productFromMenu(page) {
  const products = page.instance.data.groups.flatMap(group => group.products);
  const product = products.find(item => item.id === setup.product.id);
  if (!product) throw new Error(`menu omitted product ${setup.product.id}`);
  return product;
}
function assertVisitorPrice(product, page, label) {
  if (product.id !== setup.product.id || product.original_unit_price_cents !== setup.product.original_unit_price_cents
    || product.price_cents !== setup.product.original_unit_price_cents || product.isStaffPrice
    || product.staff_unit_price_cents !== undefined || page.querySelectorAll('.staff-price-label').length !== 0) {
    throw new Error(`${label} exposed staff price: ${JSON.stringify(product)}`);
  }
}
function assertStaffPrice(product, page, label) {
  const renderedText = page.dom.textContent;
  if (product.id !== setup.product.id || product.original_unit_price_cents !== setup.product.original_unit_price_cents
    || product.staff_unit_price_cents !== setup.product.staff_unit_price_cents
    || product.price_cents !== setup.product.staff_unit_price_cents || !product.isStaffPrice
    || page.querySelectorAll('.staff-price-label').length === 0 || !renderedText.includes(product.original_price_text)) {
    throw new Error(`${label} omitted exact staff/original prices: ${JSON.stringify(product)}`);
  }
}
async function directJSON(method, pathname, body, idempotencyKey, expectedStatus) {
  const headers = {
    Accept: 'application/json', Authorization: `Bearer ${setup.staff_token}`,
    'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey,
  };
  const response = await fetch(`${setup.api_origin}${pathname}`, { method, headers, body: JSON.stringify(body) });
  const result = await response.json();
  observations.push({ method, path: pathname, query: '', status: response.status, direct: true });
  if (response.status !== expectedStatus) {
    throw new Error(`${method} ${pathname} returned ${response.status}/${result && result.error && result.error.code || 'UNKNOWN'}`);
  }
  return result;
}
async function waitFor(predicate, message) {
  const deadline = Date.now() + 10000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}
