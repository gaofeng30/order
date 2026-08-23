/* global App, Behavior, Component, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const launchTemplate = require('../../../../apps/wechat-miniprogram/pages/launch/launch.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

let appDefinition;
let componentDefinition;
const pageDefinitions = {};
let registeringPage;
let lastNavigation = null;

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
    if (globalThis.__ui1ComposedHTTPFault) {
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

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/home/home' }];

function globalComponents(suffix, includeMenuComponents = false) {
  const iconID = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const components = {
    icon: iconID,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon: iconID } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon: iconID } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon: iconID } }),
  };
  if (!includeMenuComponents) return components;
  components.money = registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' });
  components.stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
  const imagephID = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
  components.customize = registerComponent({
    definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
    usingComponents: { icon: iconID, imageph: imagephID, money: components.money, stepper: components.stepper },
  });
  return components;
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 3000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
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
});
