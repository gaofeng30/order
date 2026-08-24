/* global Behavior, Component, Page, describe, getApp, getCurrentPages, it, simulate, wx */
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

const pages = {};
const previews = [];
let componentDefinition;
let registeringPage;

globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages[registeringPage] = definition; };
globalThis.wx = {
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  navigateTo: () => {}, navigateBack: () => {},
  previewImage: options => previews.push({ current: options.current, urls: options.urls.slice() }),
  request: options => queueMicrotask(() => options.fail({ errMsg: 'static rendered seam has no HTTP' })),
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
registeringPage = 'menu'; require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'detail'; require('../../../../apps/wechat-miniprogram/pages/detail/detail.js');

const app = { globalData: { cart: {}, pickup: null, storefrontFlavors: [] } };
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [];

describe('BE-34/35 detail media rendered contract', () => {
  it('renders the production gradient/serif placeholder on zero-image menu and detail', async () => {
    const zero = product('70', '无图套餐', []);
    const menu = render('menu', menuTemplate, 'zero-menu');
    await simulate.sleep(10);
    menu.instance.setData({ optionState: 'ready', listState: 'ready', groups: [{ id: '7', name: '套餐', products: [zero] }] });
    await simulate.sleep(10);
    const menuPlaceholder = placeholderTree(menu);
    if (!menuPlaceholder || menu.dom.textContent.includes('暂无图片')) {
      throw new Error(`BE-34 menu did not render ImagePH: ${menu.dom.textContent}`);
    }
    assertPlaceholder(menuPlaceholder, '无', 'BE-34 menu');

    const detail = render('detail', detailTemplate, 'zero-detail');
    detail.instance.setData({ detailState: 'ready', m: zero, qty: 0, imageIndex: 0 });
    await simulate.sleep(10);
    const detailPlaceholder = placeholderTree(detail);
    if (!detailPlaceholder) throw new Error(`BE-34 detail did not render ImagePH: ${detail.dom.textContent}`);
    assertPlaceholder(detailPlaceholder, '无', 'BE-34 detail');
    const externalSources = externalImageSources(detail.toJSON());
    if (externalSources.length) {
      throw new Error(`BE-34 zero-image detail rendered an external image: ${JSON.stringify(externalSources)}`);
    }
  });

  it('keeps a single image static and previews exactly that one URL', async () => {
    previews.length = 0;
    const onlyURL = 'https://images.example.test/product-71.png';
    const single = product('71', '单图套餐', [{ objectKey: 'products/71.png', url: onlyURL }]);
    const detail = render('detail', detailTemplate, 'single-detail');
    detail.instance.setData({ detailState: 'ready', m: single, qty: 0, imageIndex: 0 });
    await simulate.sleep(10);
    if (detail.querySelector('swiper') || detail.querySelector('.image-count') || !detail.querySelector('.detail-single-image')) {
      throw new Error('BE-35 single image rendered slider/count or omitted the static image');
    }
    if (pages.detail.onImageChange.call(detail.instance, { detail: { current: 1 } }) !== false) {
      throw new Error('BE-35 single image accepted a second position');
    }
    if (!pages.detail.previewImage.call(detail.instance, { currentTarget: { dataset: { url: onlyURL } } })) {
      throw new Error('BE-35 single image was not previewable');
    }
    if (previews.length !== 1 || previews[0].current !== onlyURL
      || previews[0].urls.length !== 1 || previews[0].urls[0] !== onlyURL) {
      throw new Error(`BE-35 preview escaped the single URL: ${JSON.stringify(previews)}`);
    }
  });
});

function render(name, template, suffix) {
  document.body.innerHTML = '';
  return renderPage({
    definition: pages[name], template, id: `${name}-${suffix}`,
    usingComponents: components(`${name}-${suffix}`), loadOptions: { id: '70' },
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

function product(id, name, images) {
  return {
    id, category_id: '7', name, description: '媒体边界', specification: '每份 300 克', meal_period: 'all',
    images, cover: images[0] || null, listed: true, sold_out: false, orderable: true,
    original_unit_price_cents: 1888, price_cents: 1888, isStaffPrice: false,
    price_text: '18.88', original_price_text: '18.88', staff_price_text: '',
  };
}

function assertPlaceholder(tree, expectedChar, label) {
  const serialized = JSON.stringify(tree);
  if (!serialized.includes('linear-gradient') || !serialized.includes(expectedChar) || serialized.includes('ph-img')) {
    throw new Error(`${label} placeholder was ${serialized}`);
  }
}

function placeholderTree(page) { return findTreeTag(page.toJSON(), 'imageph'); }

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

function externalImageSources(node) {
  if (!node || typeof node !== 'object') return [];
  const own = ['image', 'wx-image'].includes(node.tagName) && Array.isArray(node.attrs)
    ? node.attrs.filter(attr => attr.name === 'src' && typeof attr.value === 'string' && !attr.value.startsWith('data:image/')).map(attr => attr.value)
    : [];
  return own.concat(Array.isArray(node.children) ? node.children.flatMap(externalImageSources) : []);
}
