/* global App, Behavior, Component, ORDER_DETAIL_MEDIA_SETUP, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const detailTemplate = require('../../../../apps/wechat-miniprogram/pages/detail/detail.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const setup = ORDER_DETAIL_MEDIA_SETUP;
const pages = {};
const observations = [];
const previews = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let unavailablePath = '';

globalThis.App = definition => { appDefinition = definition; };
globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages[registeringPage] = definition; };
globalThis.wx = {
  login: options => queueMicrotask(() => options.success({ code: `${setup.run_id}-browser-session` })),
  getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }),
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  navigateTo: () => {}, navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  previewImage: options => previews.push({ current: options.current, urls: options.urls.slice() }),
  request: options => {
    const url = new URL(options.url);
    const headers = Object.assign({}, options.header || {});
    if (url.pathname === unavailablePath) headers['X-Detail-Media-Force-Status'] = '503';
    fetch(url.toString(), {
      method: options.method || 'GET', headers,
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let data = {};
      if (raw) { try { data = JSON.parse(raw); } catch {} }
      observations.push({ method: options.method || 'GET', path: url.pathname, query: url.search, status: response.status });
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

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/menu/menu' }];

describe('BE-34/35 and PAGE-U04 composed detail media boundaries', () => {
  it('renders root/MySQL zero and one image facts and fails closed for missing/unknown current facts', async () => {
    app.onLaunch();
    await waitFor(() => app.globalData.session.state !== 'loading', () => `session stayed ${app.globalData.session.state}`);
    if (app.globalData.session.state !== 'ready') throw new Error(`session ended ${app.globalData.session.state}`);

    const menu = render('menu', menuTemplate, 'menu');
    await waitFor(() => menu.instance.data.listState !== 'loading', () => `menu stayed ${menu.instance.data.listState}`);
    if (menu.instance.data.optionState !== 'ready' || menu.instance.data.listState !== 'ready') {
      throw new Error(`menu ended ${menu.instance.data.optionState}/${menu.instance.data.listState}`);
    }
    const menuProducts = menu.instance.data.groups.flatMap(group => group.products);
    const zeroMenu = menuProducts.find(product => product.id === setup.zero.id);
    const singleMenu = menuProducts.find(product => product.id === setup.single.id);
    if (!zeroMenu || zeroMenu.images.length !== 0 || !singleMenu || singleMenu.images.length !== 1) {
      throw new Error(`root menu media facts were ${JSON.stringify(menuProducts.map(item => [item.id, item.images.length]))}`);
    }
    const zeroRow = findTreeByClassAndText(menu.toJSON(), 'dish-row', setup.zero.name);
    if (!zeroRow || !findTreeTag(zeroRow, 'imageph') || JSON.stringify(zeroRow).includes('暂无图片')) {
      throw new Error('BE-34 root menu row did not render ImagePH');
    }
    assertPlaceholder(findTreeTag(zeroRow, 'imageph'), firstCharacter(setup.zero.name), 'BE-34 menu');
    if (externalImageSources(zeroRow).length) throw new Error('BE-34 zero-image menu row invented an image URL');

    const selected = menu.instance.data.pickup;
    if (!selected || selected.date !== setup.pickup.date || selected.time !== setup.pickup.time) {
      throw new Error(`server pickup selection was ${JSON.stringify(selected)}`);
    }

    const zeroDetail = render('detail', detailTemplate, 'zero', { id: setup.zero.id });
    await waitFor(() => zeroDetail.instance.data.detailState !== 'loading', () => `zero detail stayed ${zeroDetail.instance.data.detailState}`);
    if (zeroDetail.instance.data.detailState !== 'ready' || zeroDetail.instance.data.m.images.length !== 0) {
      throw new Error(`zero detail ended ${zeroDetail.instance.data.detailState}`);
    }
    const zeroPlaceholder = findTreeTag(zeroDetail.toJSON(), 'imageph');
    if (!zeroPlaceholder) throw new Error('BE-34 root detail did not render ImagePH');
    assertPlaceholder(zeroPlaceholder, firstCharacter(setup.zero.name), 'BE-34 detail');
    if (externalImageSources(zeroDetail.toJSON()).length || pages.detail.previewImage.call(zeroDetail.instance, { currentTarget: { dataset: { url: '' } } }) !== false
      || previews.length !== 0) {
      throw new Error('BE-34 zero-image detail invented or previewed an image');
    }

    const singleDetail = render('detail', detailTemplate, 'single', { id: setup.single.id });
    await waitFor(() => singleDetail.instance.data.detailState !== 'loading', () => `single detail stayed ${singleDetail.instance.data.detailState}`);
    if (singleDetail.instance.data.detailState !== 'ready' || singleDetail.instance.data.m.images.length !== 1
      || singleDetail.querySelector('swiper') || singleDetail.querySelector('.image-count') || !singleDetail.querySelector('.detail-single-image')) {
      throw new Error('BE-35 root single image was not static/count-free');
    }
    const onlyURL = singleDetail.instance.data.m.images[0].url;
    if (pages.detail.onImageChange.call(singleDetail.instance, { detail: { current: 1 } }) !== false
      || !pages.detail.previewImage.call(singleDetail.instance, { currentTarget: { dataset: { url: onlyURL } } })) {
      throw new Error('BE-35 root single image accepted a second position or could not preview');
    }
    if (previews.length !== 1 || previews[0].current !== onlyURL
      || previews[0].urls.length !== 1 || previews[0].urls[0] !== onlyURL) {
      throw new Error(`BE-35 preview escaped the only URL: ${JSON.stringify(previews)}`);
    }

    app.globalData.pickup = null;
    const missingStart = observations.length;
    const missingDetail = render('detail', detailTemplate, 'missing', { id: setup.zero.id });
    await simulate.sleep(20);
    if (missingDetail.instance.data.detailState !== 'selection_required'
      || !missingDetail.dom.textContent.includes('请先从菜单选择取餐时间')
      || observations.slice(missingStart).some(item => item.path === `/api/v1/catalog/products/${setup.zero.id}`)) {
      throw new Error('PAGE-U04 missing selection was not visibly fail-closed before HTTP');
    }
    const missingResponse = await fetch(`${setup.api_origin}/api/v1/catalog/products/${setup.zero.id}`, {
      headers: { Authorization: `Bearer ${app.globalData.session.accessToken}` },
    });
    const missingBody = await missingResponse.json();
    if (missingResponse.status !== 400 || !missingBody.error || missingBody.error.code !== 'INVALID_MENU_SELECTION') {
      throw new Error(`PAGE-U04 missing date/time returned ${missingResponse.status}/${JSON.stringify(missingBody)}`);
    }

    app.globalData.pickup = { date: setup.pickup.date, mealPeriod: setup.pickup.meal_period, time: setup.pickup.time };
    unavailablePath = `/api/v1/catalog/products/${setup.zero.id}`;
    const unavailableDetail = render('detail', detailTemplate, 'unavailable', { id: setup.zero.id });
    await waitFor(() => unavailableDetail.instance.data.detailState !== 'loading', () => `503 detail stayed ${unavailableDetail.instance.data.detailState}`);
    unavailablePath = '';
    if (unavailableDetail.instance.data.detailState !== 'error' || unavailableDetail.instance.data.m !== null
      || !unavailableDetail.dom.textContent.includes('商品加载失败') || !unavailableDetail.querySelector('.detail-state .btn')
      || externalImageSources(unavailableDetail.toJSON()).length) {
      throw new Error('PAGE-U04 503 rendered stale content or no visible retry');
    }
  });
});

function render(name, template, suffix, loadOptions = {}) {
  document.body.innerHTML = '';
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
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
    customize: registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, imageph, money, stepper },
    }),
  };
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 5000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}

function firstCharacter(value) { return Array.from(value || '食')[0] || '食'; }

function assertPlaceholder(tree, expectedChar, label) {
  const serialized = JSON.stringify(tree);
  if (!serialized.includes('linear-gradient') || !serialized.includes(expectedChar) || serialized.includes('ph-img')) {
    throw new Error(`${label} placeholder was ${serialized}`);
  }
}

function findTreeTag(node, tagName) {
  if (!node || typeof node !== 'object') return null;
  if (node.tagName === tagName) return node;
  if (!Array.isArray(node.children)) return null;
  for (const child of node.children) {
    const found = findTreeTag(child, tagName);
    if (found) return found;
  }
  return null;
}

function findTreeByClassAndText(node, className, text) {
  if (!node || typeof node !== 'object') return null;
  const hasClass = Array.isArray(node.attrs) && node.attrs.some(attr => attr.name === 'class'
    && String(attr.value).split(/\s+/).includes(className));
  if (hasClass && JSON.stringify(node).includes(text)) return node;
  if (!Array.isArray(node.children)) return null;
  for (const child of node.children) {
    const found = findTreeByClassAndText(child, className, text);
    if (found) return found;
  }
  return null;
}

function externalImageSources(node) {
  if (!node || typeof node !== 'object') return [];
  const own = ['image', 'wx-image'].includes(node.tagName) && Array.isArray(node.attrs)
    ? node.attrs.filter(attr => attr.name === 'src' && typeof attr.value === 'string' && !attr.value.startsWith('data:image/')).map(attr => attr.value)
    : [];
  return own.concat(Array.isArray(node.children) ? node.children.flatMap(externalImageSources) : []);
}
