/* global Behavior, Component, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const profileTemplate = require('../../../../apps/wechat-miniprogram/pages/profile/profile.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const pages = {};
const observations = [];
let componentDefinition;
let profileMode = 'fail';

globalThis.Behavior = definition => definition;
globalThis.Component = definition => { componentDefinition = definition; };
globalThis.Page = definition => { pages.profile = definition; };
globalThis.wx = {
  getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getSystemInfoSync: () => ({ statusBarHeight: 20, screenWidth: 375, screenHeight: 812, safeArea: { bottom: 778 } }),
  getMenuButtonBoundingClientRect: () => ({ top: 24, left: 278, width: 87, height: 32 }),
  navigateTo: () => {}, redirectTo: () => {}, reLaunch: () => {}, navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  getUserProfile: options => queueMicrotask(() => {
    if (profileMode === 'success') {
      options.success({ userInfo: { nickName: '本次昵称', avatarUrl: 'https://avatar.example.test/once.png' } });
    } else options.fail({ errMsg: 'getUserProfile:fail user deny' });
  }),
  request: options => {
    const url = new URL(options.url);
    observations.push({ method: options.method || 'GET', path: url.pathname });
    queueMicrotask(() => {
      if (url.pathname === '/api/v1/me/identity') {
        options.success({ statusCode: 200, data: { identity: { malformed: true } } });
      } else options.success({ statusCode: 200, data: { orders: [] } });
    });
  },
};

require('../../../../apps/wechat-miniprogram/components/icon/icon.js');
const iconDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/navbar/navbar.js');
const navbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/pill/pill.js');
const pillDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.js');
const tabbarDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/components/toast/toast.js');
const toastDefinition = componentDefinition;
require('../../../../apps/wechat-miniprogram/pages/profile/profile.js');

const app = {
  globalData: {
    runtimeEndpoint: { state: 'ready', envVersion: 'develop', origin: 'http://127.0.0.1:18082' },
    apiBaseUrl: 'http://127.0.0.1:18082',
    session: { state: 'ready', accessToken: 'A'.repeat(43) },
  },
};
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: 'pages/profile/profile' }];

describe('PAGE-U09 cosmetic profile and native contact boundary', () => {
  it('keeps identity failures neutral and profile/contact failures at zero business HTTP', async () => {
    profileMode = 'fail';
    const profile = render('failure');
    await waitFor(() => profile.instance.data.identityState !== 'loading');
    if (profile.instance.data.identityState !== 'error' || profile.instance.data.pricingKind !== 'VISITOR'
      || !profile.dom.textContent.includes('微信用户') || !profile.dom.textContent.includes('客')
      || profile.dom.textContent.includes('员工价') || !profile.querySelector('.prow.last')) {
      throw new Error(`identity failure rendered a false identity: ${profile.dom.textContent}`);
    }
    const before = observations.length;
    const selected = await pages.profile.chooseProfile.call(profile.instance);
    await simulate.sleep(10);
    if (selected !== false || observations.length !== before || profile.instance.data.nick !== '微信用户'
      || profile.instance.data.avatarText !== '客' || profile.instance.data.avatarUrl !== '') {
      throw new Error('rejected cosmetic profile changed identity or issued business HTTP');
    }
    const contact = profile.querySelector('.prow.last');
    await contact.dispatchEvent('touchstart');
    await simulate.sleep(10);
    if (observations.length !== before || !profile.dom.textContent.includes('联系客服')) {
      throw new Error('native contact failure issued business HTTP or removed the neutral surface');
    }
  });

  it('renders a successful one-time cosmetic avatar locally without business HTTP', async () => {
    profileMode = 'success';
    const profile = render('success');
    await waitFor(() => profile.instance.data.identityState !== 'loading');
    const before = observations.length;
    const selected = await pages.profile.chooseProfile.call(profile.instance);
    await simulate.sleep(10);
    const avatar = findTreeByClass(profile.toJSON(), 'avatar-img');
    const avatarSource = avatar && Array.isArray(avatar.attrs)
      ? avatar.attrs.find(attribute => attribute.name === 'src') : null;
    if (!selected || observations.length !== before || profile.instance.data.nick !== '本次昵称'
      || !profile.dom.textContent.includes('本次昵称') || !avatar
      || !avatarSource || avatarSource.value !== 'https://avatar.example.test/once.png') {
      throw new Error(`one-time cosmetic avatar was not visibly rendered without business HTTP: ${JSON.stringify(avatar)}`);
    }
  });
});

function render(suffix) {
  document.body.innerHTML = '';
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  return renderPage({
    definition: pages.profile, template: profileTemplate, id: `profile-${suffix}`,
    usingComponents: {
      icon,
      navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
      pill: registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' }),
      tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
      toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
    },
  });
}

async function waitFor(predicate) {
  const deadline = Date.now() + 5000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error('profile remained loading');
    await simulate.sleep(10);
  }
}

function findTreeByClass(node, className) {
  if (!node || typeof node !== 'object') return null;
  const hasClass = Array.isArray(node.attrs) && node.attrs.some(attribute => attribute.name === 'class'
    && String(attribute.value).split(/\s+/).includes(className));
  if (hasClass) return node;
  if (!Array.isArray(node.children)) return null;
  for (const child of node.children) {
    const found = findTreeByClass(child, className);
    if (found) return found;
  }
  return null;
}
