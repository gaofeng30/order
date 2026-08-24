/* global App, Behavior, Component, ORDER_AC17_PROXY_ORIGIN, ORDER_AC17_SETUP, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const launchTemplate = require('../../wechat-miniprogram/pages/launch/launch.wxml');
const menuTemplate = require('../../wechat-miniprogram/pages/menu/menu.wxml');
const detailTemplate = require('../../wechat-miniprogram/pages/detail/detail.wxml');
const merchantTemplate = require('../../wechat-miniprogram/pages/admin-products/admin-products.wxml');
const iconTemplate = require('../../wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../wechat-miniprogram/components/imageph/imageph.wxml');
const moneyTemplate = require('../../wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../wechat-miniprogram/components/pill/pill.wxml');
const stepperTemplate = require('../../wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../wechat-miniprogram/components/toast/toast.wxml');
const customizeTemplate = require('../../wechat-miniprogram/components/customize/customize.wxml');
const { registerComponent, renderPage } = require('../../../tools/miniprogram-ui/test/browser/page-adapter.cjs');

const setup = ORDER_AC17_SETUP;
const pageDefinitions = {};
const requests = [];
let appDefinition;
let componentDefinition;
let registeringPage = '';
let lastNavigation = '';
let lastStorefront = null;

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pageDefinitions[registeringPage] = definition; };
globalThis.wx = {
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  navigateTo: options => { lastNavigation = options.url; },
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  previewImage: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  request: options => {
    const requestURL = new URL(options.url);
    if (setup.fail_path && requestURL.pathname === setup.fail_path) {
      requests.push({ method: options.method || 'GET', path: requestURL.pathname, status: 503, injected: true });
      queueMicrotask(() => options.success({ statusCode: 503, data: { error: { code: 'FAULT_INJECTED' } } }));
      return;
    }
    fetch(requestURL.toString(), {
      method: options.method || 'GET', headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let body = {};
      if (raw) try { body = JSON.parse(raw); } catch (error) { body = {}; }
      if (setup.mode === 'customer-bad-object' && requestURL.pathname === '/api/v1/storefront/settings'
        && body?.storefront?.launch_layer?.image) {
        body.storefront.launch_layer.image.url = '/api/v1/objects/ac17-mismatched-object.png';
      }
      if (requestURL.pathname === '/api/v1/storefront/settings') lastStorefront = body;
      requests.push({ method: options.method || 'GET', path: requestURL.pathname, status: response.status });
      options.success({ statusCode: response.status, data: body });
    }).catch(error => options.fail(error));
  },
};

require('../../wechat-miniprogram/components/icon/icon.js');
const iconDefinition = componentDefinition;
require('../../wechat-miniprogram/components/imageph/imageph.js');
const imagephDefinition = componentDefinition;
require('../../wechat-miniprogram/components/money/money.js');
const moneyDefinition = componentDefinition;
require('../../wechat-miniprogram/components/navbar/navbar.js');
const navbarDefinition = componentDefinition;
require('../../wechat-miniprogram/components/pill/pill.js');
const pillDefinition = componentDefinition;
require('../../wechat-miniprogram/components/stepper/stepper.js');
const stepperDefinition = componentDefinition;
require('../../wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../wechat-miniprogram/components/customize/customize.js');
const customizeDefinition = componentDefinition;
const componentSources = [
  ['icon', iconDefinition, iconTemplate], ['imageph', imagephDefinition, imagephTemplate],
  ['money', moneyDefinition, moneyTemplate], ['navbar', navbarDefinition, navbarTemplate],
  ['pill', pillDefinition, pillTemplate], ['stepper', stepperDefinition, stepperTemplate],
  ['tabbar', tabbarDefinition, tabbarTemplate], ['toast', toastDefinition, toastTemplate],
  ['customize', customizeDefinition, customizeTemplate],
];

require('../../wechat-miniprogram/app.js');
registeringPage = 'launch'; require('../../wechat-miniprogram/pages/launch/launch.js');
registeringPage = 'menu'; require('../../wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'detail'; require('../../wechat-miniprogram/pages/detail/detail.js');
registeringPage = 'admin-products'; require('../../wechat-miniprogram/pages/admin-products/admin-products.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
app.globalData.apiBaseUrl = ORDER_AC17_PROXY_ORIGIN;
app.globalData.runtimeEndpoint = { state: 'ready', envVersion: 'develop', origin: ORDER_AC17_PROXY_ORIGIN, errorCode: '' };
app.globalData.session = { state: 'ready', accessToken: setup.mini_token, expiresAt: setup.expires_at };
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/launch/launch' }];

function components(suffix) {
  const loaded = {};
  for (const [name, definition, template] of componentSources) {
    loaded[name] = registerComponent({
      definition, template,
      id: `ac17-${name}-${suffix}`, tagName: name,
      usingComponents: name === 'navbar' || name === 'tabbar' || name === 'toast' || name === 'customize'
        ? { icon: loaded.icon, money: loaded.money, stepper: loaded.stepper }
        : {},
    });
  }
  return loaded;
}

function render(name, suffix, loadOptions = {}) {
  document.body.innerHTML = '';
  return renderPage({
    definition: pageDefinitions[name],
    template: name === 'launch' ? launchTemplate : name === 'menu' ? menuTemplate : name === 'detail' ? detailTemplate : merchantTemplate,
    id: `ac17-${name}-${suffix}`,
    usingComponents: components(`${name}-${suffix}`),
    loadOptions,
  });
}

async function waitFor(predicate, message, timeout = 8000) {
  const deadline = Date.now() + timeout;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(20);
  }
}

function productFrom(page) {
  return page.instance.data.groups.flatMap(group => group.products).find(product => product.id === setup.product.id);
}

describe('AC-17 three rendered clients share one root fact source', () => {
  it(setup.mode, async () => {
    if (setup.mode === 'customer-before' || setup.mode === 'customer-after') {
      const launch = render('launch', `${setup.mode}-launch`);
      await waitFor(() => launch.instance.data.storefrontState !== 'loading', () => `launch ${launch.instance.data.storefrontState}`);
      if (launch.instance.data.storefrontState !== 'ready' || launch.instance.data.launchLayer?.image?.objectKey !== setup.launch_object_key
        || !launch.dom.textContent.includes(setup.store_name)) {
        throw new Error(`customer launch fact state=${launch.instance.data.storefrontState} object=${launch.instance.data.launchLayer?.image?.objectKey || ''} expected=${setup.launch_object_key} store=${launch.instance.data.storeName} runtime=${JSON.stringify(app.globalData.runtimeEndpoint)} api=${app.globalData.apiBaseUrl} body=${JSON.stringify(lastStorefront)} text=${launch.dom.textContent}`);
      }

      const menu = render('menu', `${setup.mode}-menu`);
      await waitFor(() => menu.instance.data.listState !== 'loading', () => `menu ${menu.instance.data.optionState}/${menu.instance.data.listState}`);
      const product = productFrom(menu);
      const expectedSoldOut = setup.mode === 'customer-after';
      if (menu.instance.data.listState !== 'ready' || !menu.instance.data.cats.some(category => category.id === setup.category.id && category.name === setup.category.name)
        || !product || product.name !== setup.product.name || product.soldOut !== expectedSoldOut
        || product.orderable === expectedSoldOut) {
        throw new Error(`customer menu fact mismatch product=${JSON.stringify(product)} state=${menu.instance.data.optionState}/${menu.instance.data.listState} pickup=${JSON.stringify(menu.instance.data.pickup)} cats=${JSON.stringify(menu.instance.data.cats)} groups=${JSON.stringify(menu.instance.data.groups)} requests=${JSON.stringify(requests)}`);
      }

      const detail = render('detail', `${setup.mode}-detail`, { id: setup.product.id });
      await waitFor(() => detail.instance.data.detailState !== 'loading', () => `detail ${detail.instance.data.detailState}`);
      if (detail.instance.data.detailState !== 'ready' || detail.instance.data.m.id !== setup.product.id
        || detail.instance.data.m.name !== setup.product.name || detail.instance.data.m.soldOut !== expectedSoldOut
        || detail.instance.data.m.orderable === expectedSoldOut) throw new Error('customer detail did not read the same dated product fact');
      return;
    }

    if (setup.mode === 'merchant-soldout' || setup.mode === 'merchant-http-failure') {
      const merchant = render('admin-products', setup.mode);
      await waitFor(() => merchant.instance.data.listState !== 'loading', () => `merchant ${merchant.instance.data.listState}`);
      const product = merchant.instance.data.list.find(item => item.id === setup.product.id);
      if (!product) throw new Error('merchant rendered menu omitted the PC-created product');
      const changed = await pageDefinitions['admin-products'].toggleSoldout.call(merchant.instance, {
        currentTarget: { dataset: { id: setup.product.id } },
      });
      const refreshed = merchant.instance.data.list.find(item => item.id === setup.product.id);
      if (setup.mode === 'merchant-soldout') {
        if (!changed || merchant.instance.data.actionState !== 'ready' || !refreshed?.soldOut) {
          throw new Error(`merchant sold-out UI changed=${changed} state=${merchant.instance.data.actionState} product=${JSON.stringify(refreshed)} requests=${JSON.stringify(requests)}`);
        }
      } else if (changed || merchant.instance.data.actionState !== 'error' || !refreshed?.soldOut) {
        throw new Error('merchant HTTP failure produced false success or local fact drift');
      }
      return;
    }

    if (setup.mode === 'customer-http-failure') {
      app.globalData.cart = {};
      const menu = render('menu', setup.mode);
      await waitFor(() => menu.instance.data.listState !== 'loading', () => `failed menu ${menu.instance.data.listState}`);
      lastNavigation = '';
      const opened = pageDefinitions.menu.goDetail.call(menu.instance, { currentTarget: { dataset: { id: setup.product.id } } });
      if (menu.instance.data.listState !== 'error' || menu.instance.data.groups.length || menu.instance.data.canCheckout
        || opened !== false || lastNavigation || Object.keys(app.globalData.cart).length) throw new Error('customer HTTP failure produced a false-success menu');
      return;
    }

    if (setup.mode === 'customer-bad-object') {
      const launch = render('launch', setup.mode);
      await waitFor(() => launch.instance.data.storefrontState !== 'loading', () => `bad object launch ${launch.instance.data.storefrontState}`);
      lastNavigation = '';
      const opened = pageDefinitions.launch.go.call(launch.instance, { currentTarget: { dataset: { to: 'home' } } });
      if (launch.instance.data.storefrontState !== 'error' || launch.instance.data.launchLayer !== null || opened !== false || lastNavigation) {
        throw new Error('bad object metadata produced a false-success storefront');
      }
      return;
    }
    throw new Error(`unknown AC-17 mode ${setup.mode}`);
  });
});
