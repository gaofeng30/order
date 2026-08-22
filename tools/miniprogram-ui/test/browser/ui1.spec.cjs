/* global App, Behavior, Component, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const appConfig = require('../../../../apps/wechat-miniprogram/app.json');
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
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateTo: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  request: options => {
    fetch(options.url, { method: options.method || 'GET' })
      .then(async response => {
        const data = await response.json();
        options.success({ statusCode: response.status, data });
      })
      .catch(error => options.fail(error));
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

function globalComponents(componentSuffix, includeMenuComponents = false) {
  const iconID = registerComponent({
    definition: iconDefinition,
    template: iconTemplate,
    id: `icon-${componentSuffix}`,
    tagName: 'icon',
  });
  const components = {
    icon: iconID,
    navbar: registerComponent({
      definition: navbarDefinition,
      template: navbarTemplate,
      id: `navbar-${componentSuffix}`,
      tagName: 'navbar',
      usingComponents: { icon: iconID },
    }),
    tabbar: registerComponent({
      definition: tabbarDefinition,
      template: tabbarTemplate,
      id: `tabbar-${componentSuffix}`,
      tagName: 'tabbar',
      usingComponents: { icon: iconID },
    }),
    toast: registerComponent({
      definition: toastDefinition,
      template: toastTemplate,
      id: `toast-${componentSuffix}`,
      tagName: 'toast',
      usingComponents: { icon: iconID },
    }),
  };
  if (!includeMenuComponents) return components;

  components.money = registerComponent({
    definition: moneyDefinition,
    template: moneyTemplate,
    id: `money-${componentSuffix}`,
    tagName: 'money',
  });
  components.stepper = registerComponent({
    definition: stepperDefinition,
    template: stepperTemplate,
    id: `stepper-${componentSuffix}`,
    tagName: 'stepper',
  });
  const imagephID = registerComponent({
    definition: imagephDefinition,
    template: imagephTemplate,
    id: `imageph-${componentSuffix}`,
    tagName: 'imageph',
  });
  components.customize = registerComponent({
    definition: customizeDefinition,
    template: customizeTemplate,
    id: `customize-${componentSuffix}`,
    tagName: 'customize',
    usingComponents: {
      icon: iconID,
      imageph: imagephID,
      money: components.money,
      stepper: components.stepper,
    },
  });
  return components;
}

describe('mini-program UI1 in real Chromium simulator', () => {
  it('cold-starts into anonymous browse and exposes the menu entry without phone authorization', async () => {
    document.body.innerHTML = '';
    lastNavigation = null;
    app.onLaunch();

    const componentSuffix = Date.now();
    const components = globalComponents(componentSuffix);
    if (appConfig.pages[0] !== 'pages/launch/launch') {
      throw new Error(`configured first route was ${appConfig.pages[0] || 'missing'}`);
    }

    const launch = renderPage({
      definition: pageDefinitions.launch,
      template: launchTemplate,
      id: `launch-page-${componentSuffix}`,
      usingComponents: components,
    });
    if (!launch.dom.textContent.includes('用户端')) {
      throw new Error(`cold start did not render the user entry: ${launch.dom.textContent}`);
    }
    const userEntry = launch.querySelector('.id-card.primary');
    if (!userEntry) throw new Error('rendered launch page has no user entry');
    userEntry.dispatchEvent('touchstart');
    userEntry.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== '/pages/home/home') {
      throw new Error(`user entry navigated to ${lastNavigation || 'nothing'}`);
    }

    const home = renderPage({
      definition: pageDefinitions.home,
      template: homeTemplate,
      id: `home-page-${componentSuffix}`,
      usingComponents: components,
    });

    const visibleText = home.dom.textContent;
    if (!visibleText.includes('你好，欢迎光临')) throw new Error(`home greeting was not rendered: ${visibleText}`);
    if (!visibleText.includes('进入菜单查看当日商品目录')) throw new Error(`menu entry was not rendered: ${visibleText}`);
    if (visibleText.includes('手机号授权')) throw new Error(`cold start rendered a phone authorization prompt: ${visibleText}`);
    if (home.querySelector('button[open-type="getPhoneNumber"]')) {
      throw new Error('user home rendered a phone authorization control');
    }

    const menuEntry = home.querySelector('.search');
    if (!menuEntry) throw new Error('rendered menu entry is not interactive');
    menuEntry.dispatchEvent('touchstart');
    menuEntry.dispatchEvent('touchend');
    await simulate.sleep(10);
    if (lastNavigation !== '/pages/menu/menu') {
      throw new Error(`menu entry navigated to ${lastNavigation || 'nothing'}`);
    }
  });

  it('renders a network error and recovers after the user taps retry', async () => {
    document.body.innerHTML = '';
    app.onLaunch();
    const componentSuffix = `network-${Date.now()}`;
    const menu = renderPage({
      definition: pageDefinitions.menu,
      template: menuTemplate,
      id: `menu-page-${componentSuffix}`,
      usingComponents: globalComponents(componentSuffix, true),
    });

    await simulate.sleep(20);
    if (!menu.dom.textContent.includes('目录加载失败')) {
      throw new Error(`network failure was not rendered: ${menu.dom.textContent}`);
    }

    const retry = menu.querySelector('.catalog-state .btn');
    if (!retry) throw new Error('rendered network error has no retry interaction');
    retry.dispatchEvent('touchstart');
    retry.dispatchEvent('touchend');
    await simulate.sleep(30);

    if (!menu.dom.textContent.includes('恢复后的热菜')) {
      throw new Error(`retry did not render the recovered catalog: ${menu.dom.textContent}`);
    }
  });

  it('changes menu category and opens the rendered product-selection sheet', async () => {
    document.body.innerHTML = '';
    app.onLaunch();
    const componentSuffix = `interaction-${Date.now()}`;
    const menu = renderPage({
      definition: pageDefinitions.menu,
      template: menuTemplate,
      id: `menu-page-${componentSuffix}`,
      usingComponents: globalComponents(componentSuffix, true),
    });

    await simulate.sleep(30);
    const categories = menu.querySelectorAll('.seg');
    if (categories.length !== 2) throw new Error(`rendered category count was ${categories.length}`);
    categories[1].dispatchEvent('touchstart');
    categories[1].dispatchEvent('touchend');
    await simulate.sleep(10);
    if (!categories[1].dom.className.includes('on')) {
      throw new Error(`second category did not become active: ${categories[1].dom.className}`);
    }

    const productChoices = menu.querySelectorAll('.act-btn');
    if (productChoices.length !== 2) throw new Error(`rendered product choice count was ${productChoices.length}`);
    productChoices[1].dispatchEvent('touchstart');
    productChoices[1].dispatchEvent('touchend');
    await simulate.sleep(20);
    if (!menu.dom.textContent.includes('口味偏好')) {
      throw new Error(`product-selection sheet was not rendered: ${menu.dom.textContent}`);
    }
  });
});
