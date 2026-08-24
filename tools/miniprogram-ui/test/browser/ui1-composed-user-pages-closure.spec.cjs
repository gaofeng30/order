/* global App, Behavior, Component, ORDER_USER_PAGES_SETUP, Page, describe, getApp, getCurrentPages, it, simulate, wx */
const confirmTemplate = require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.wxml');
const customizeTemplate = require('../../../../apps/wechat-miniprogram/components/customize/customize.wxml');
const detailTemplate = require('../../../../apps/wechat-miniprogram/pages/detail/detail.wxml');
const homeTemplate = require('../../../../apps/wechat-miniprogram/pages/home/home.wxml');
const iconTemplate = require('../../../../apps/wechat-miniprogram/components/icon/icon.wxml');
const imagephTemplate = require('../../../../apps/wechat-miniprogram/components/imageph/imageph.wxml');
const launchTemplate = require('../../../../apps/wechat-miniprogram/pages/launch/launch.wxml');
const menuTemplate = require('../../../../apps/wechat-miniprogram/pages/menu/menu.wxml');
const moneyTemplate = require('../../../../apps/wechat-miniprogram/components/money/money.wxml');
const navbarTemplate = require('../../../../apps/wechat-miniprogram/components/navbar/navbar.wxml');
const ordersTemplate = require('../../../../apps/wechat-miniprogram/pages/orders/orders.wxml');
const pillTemplate = require('../../../../apps/wechat-miniprogram/components/pill/pill.wxml');
const profileTemplate = require('../../../../apps/wechat-miniprogram/pages/profile/profile.wxml');
const stepperTemplate = require('../../../../apps/wechat-miniprogram/components/stepper/stepper.wxml');
const tabbarTemplate = require('../../../../apps/wechat-miniprogram/components/tabbar/tabbar.wxml');
const toastTemplate = require('../../../../apps/wechat-miniprogram/components/toast/toast.wxml');
const { registerComponent, renderPage } = require('./page-adapter.cjs');

const setup = ORDER_USER_PAGES_SETUP;
const pageDefinitions = {};
const observations = [];
const paymentCalls = [];
const previewCalls = [];
const profileCalls = [];
let appDefinition;
let componentDefinition;
let registeringPage;
let lastNavigation = null;
let currentRoute = 'pages/launch/launch';

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
  navigateTo: options => { lastNavigation = options.url; },
  redirectTo: options => { lastNavigation = options.url; },
  reLaunch: options => { lastNavigation = options.url; },
  navigateBack: () => {},
  getRandomValues: bytes => { crypto.getRandomValues(bytes); return bytes; },
  request: options => {
    const requestURL = new URL(options.url);
    fetch(requestURL.toString(), {
      method: options.method || 'GET', headers: options.header || {},
      body: options.data === undefined ? undefined : JSON.stringify(options.data),
    }).then(async response => {
      const raw = await response.text();
      let body = {};
      if (raw) {
        try { body = JSON.parse(raw); } catch (error) { body = {}; }
      }
      observations.push({
        method: options.method || 'GET', path: requestURL.pathname,
        query: requestURL.search, status: response.status,
      });
      options.success({ statusCode: response.status, data: body });
    }).catch(error => options.fail(error));
  },
  requestPayment: options => {
    paymentCalls.push({
      timeStamp: options.timeStamp, nonceStr: options.nonceStr, package: options.package,
      signType: options.signType, paySign: options.paySign,
    });
    queueMicrotask(() => options.success({ errMsg: 'requestPayment:ok' }));
  },
  previewImage: options => { previewCalls.push({ current: options.current, urls: options.urls.slice() }); },
  getUserProfile: options => {
    profileCalls.push({ desc: options.desc });
    queueMicrotask(() => options.success({ userInfo: { nickName: '本次微信昵称', avatarUrl: 'https://avatar.example.com/once.png' } }));
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
registeringPage = 'launch';
require('../../../../apps/wechat-miniprogram/pages/launch/launch.js');
registeringPage = 'home';
require('../../../../apps/wechat-miniprogram/pages/home/home.js');
registeringPage = 'menu';
require('../../../../apps/wechat-miniprogram/pages/menu/menu.js');
registeringPage = 'detail';
require('../../../../apps/wechat-miniprogram/pages/detail/detail.js');
registeringPage = 'confirm';
require('../../../../apps/wechat-miniprogram/pages/confirm/confirm.js');
registeringPage = 'orders';
require('../../../../apps/wechat-miniprogram/pages/orders/orders.js');
registeringPage = 'profile';
require('../../../../apps/wechat-miniprogram/pages/profile/profile.js');

const app = Object.assign({}, appDefinition, { globalData: JSON.parse(JSON.stringify(appDefinition.globalData)) });
globalThis.getApp = () => app;
globalThis.getCurrentPages = () => [{ route: currentRoute }];

describe('PAGE-U01 through PAGE-U09 strict composed rendered closure', () => {
  it('closes the remaining user-visible page operations against one root API and fresh MySQL', async () => {
    app.onLaunch();
    await waitFor(() => app.globalData.session.state !== 'loading', () => `session remained ${app.globalData.session.state}`);
    if (app.globalData.session.state !== 'ready') throw new Error(`session ended ${app.globalData.session.state}`);

    const launch = render('launch', launchTemplate, 'u01');
    await waitFor(() => launch.instance.data.storefrontState !== 'loading', () => 'PAGE-U01 storefront stayed loading');
    if (launch.instance.data.storefrontState !== 'ready' || !launch.dom.textContent.includes('绥安食品')) {
      throw new Error(`PAGE-U01 storefront was ${launch.instance.data.storefrontState}`);
    }
    const userEntry = launch.querySelector('.id-card.primary');
    const merchantEntry = launch.querySelector('.id-plain');
    if (!userEntry || !merchantEntry || launch.querySelector('button[open-type="getPhoneNumber"]')) {
      throw new Error('PAGE-U01 bound identity selection repeated phone authorization or omitted an entry');
    }
    const beforeSelection = observations.length;
    lastNavigation = null;
    const userNavigated = pageDefinitions.launch.go.call(launch.instance, {
      currentTarget: { dataset: { to: userEntry.dom.dataset.to } },
    });
    if (!userNavigated || lastNavigation !== '/pages/home/home') {
      throw new Error(`PAGE-U01 user entry navigated to ${lastNavigation || 'nothing'}`);
    }
    lastNavigation = null;
    const merchantNavigated = pageDefinitions.launch.goMerchant.call(launch.instance);
    if (!merchantNavigated || lastNavigation !== '/pages/admin-orders/admin-orders'
      || observations.length !== beforeSelection) {
      throw new Error(`PAGE-U01 merchant selection ended ${merchantNavigated}/${lastNavigation || 'nothing'}`);
    }

    const home = render('home', homeTemplate, 'u02');
    await waitFor(() => home.instance.data.settingsState !== 'loading' && home.instance.data.ordersState !== 'loading',
      () => `PAGE-U02 home stayed ${home.instance.data.settingsState}/${home.instance.data.ordersState}`);
    if (home.instance.data.settingsState !== 'ready' || home.instance.data.ordersState !== 'ready'
      || !home.instance.data.ongoing || home.querySelectorAll('.grid-item').length !== 3
      || !home.dom.textContent.includes(setup.notice)) {
      throw new Error(`PAGE-U02 did not render shared storefront/active-order facts: ${home.dom.textContent}`);
    }
    lastNavigation = null;
    const ongoingBar = home.querySelector('.ob');
    const ongoingOpened = ongoingBar && pageDefinitions.home.tapOngoing.call(home.instance);
    if (!ongoingOpened || !lastNavigation || !lastNavigation.startsWith('/pages/order-detail/order-detail?id=')) {
      throw new Error(`PAGE-U02 ongoing bar navigated to ${lastNavigation || 'nothing'}`);
    }
    const gridEntries = home.querySelectorAll('.grid-item');
    if (gridEntries.some((node, index) => !node.dom.textContent.includes(home.instance.data.grid[index].cn))) {
      throw new Error('PAGE-U02 rendered grid did not match its server-backed page state');
    }
    lastNavigation = null;
    const menuOpened = pageDefinitions.home.gridTap.call(home.instance, {
      currentTarget: { dataset: { k: home.instance.data.grid[0].k } },
    });
    if (!menuOpened || lastNavigation !== '/pages/menu/menu') throw new Error(`PAGE-U02 menu entry navigated to ${lastNavigation || 'nothing'}`);
    lastNavigation = null;
    const ordersOpened = pageDefinitions.home.gridTap.call(home.instance, {
      currentTarget: { dataset: { k: home.instance.data.grid[1].k } },
    });
    if (!ordersOpened || lastNavigation !== '/pages/orders/orders') throw new Error(`PAGE-U02 order entry navigated to ${lastNavigation || 'nothing'}`);
    lastNavigation = null;
    const pickupOpened = pageDefinitions.home.gridTap.call(home.instance, {
      currentTarget: { dataset: { k: home.instance.data.grid[2].k } },
    });
    if (!pickupOpened) throw new Error('PAGE-U02 pickup-code entry was not actionable');
    if (home.instance.data.ongoing.ready) {
      if (!lastNavigation || !lastNavigation.startsWith('/pages/order-detail/order-detail?id=')) {
        throw new Error(`PAGE-U02 pickup-code entry navigated to ${lastNavigation || 'nothing'}`);
      }
    } else if (lastNavigation !== null || !home.dom.textContent.includes('暂无可取餐订单')) {
      throw new Error('PAGE-U02 pickup-code entry neither opened READY detail nor rendered its empty state');
    }

    app.globalData.cart = {};
    const menu = render('menu', menuTemplate, 'u03', true);
    await waitFor(() => menu.instance.data.listState !== 'loading', () => `PAGE-U03 menu stayed ${menu.instance.data.listState}`);
    if (menu.instance.data.optionState !== 'ready' || menu.instance.data.listState !== 'ready') {
      throw new Error(`PAGE-U03 menu ended ${menu.instance.data.optionState}/${menu.instance.data.listState}`);
    }
    if (!menu.querySelector('.pickup-bar')) throw new Error('PAGE-U03 pickup bar was not rendered');
    pageDefinitions.menu.openPicker.call(menu.instance);
    await waitFor(() => menu.instance.data.pickerVisible === true, () => 'PAGE-U03 pickup picker did not open');
    if (menu.instance.data.pickerDates.length !== 2 || menu.instance.data.pickerDates.some(item => !item.available)) {
      throw new Error(`PAGE-U03 service dates were ${JSON.stringify(menu.instance.data.pickerDates)}`);
    }
    const selectableDate = menu.instance.data.pickerDates[1];
    if (!selectableDate || !pageDefinitions.menu.pickPickerDate.call(menu.instance, {
      currentTarget: { dataset: { date: selectableDate.date } },
    })) throw new Error('PAGE-U03 had no selectable service date');
    const selectableMeal = menu.instance.data.pickerGroups.find(item => !item.cutOff && item.times.length);
    if (!selectableMeal) throw new Error('PAGE-U03 had no selectable server pickup time');
    const pickupRequestStart = observations.length;
    const picked = await pageDefinitions.menu.pickPickerTime.call(menu.instance, {
      currentTarget: { dataset: { date: selectableMeal.date, period: selectableMeal.key, t: selectableMeal.times[0] } },
    });
    if (!picked || menu.instance.data.pickerVisible || menu.instance.data.pickup.date !== selectableMeal.date
      || menu.instance.data.pickup.time !== selectableMeal.times[0]
      || !observations.slice(pickupRequestStart).some(item => item.path === '/api/v1/menu' && item.status === 200)) {
      throw new Error(`PAGE-U03 pickup selection ended ${JSON.stringify(menu.instance.data.pickup)}`);
    }
    const search = menu.querySelector('.search-in');
    if (!search) throw new Error('PAGE-U03 search input was not rendered');
    pageDefinitions.menu.onSearch.call(menu.instance, { detail: { value: setup.product.name } });
    await waitFor(() => menu.instance.data.search === setup.product.name, () => 'PAGE-U03 search did not update');
    const visibleIDs = menu.instance.data.groups.flatMap(group => group.products.map(product => product.id));
    if (visibleIDs.length !== 1 || visibleIDs[0] !== setup.product.id) {
      throw new Error(`PAGE-U03 search returned ${JSON.stringify(visibleIDs)}`);
    }
    pageDefinitions.menu.onSearch.call(menu.instance, { detail: { value: '' } });
    const choose = menu.querySelectorAll('.act-btn').find(node => node.dom.textContent.includes('选择'));
    if (!choose) throw new Error('PAGE-U03 rendered no product choose control');
    pageDefinitions.menu.openCustomize.call(menu.instance, {
      currentTarget: { dataset: { id: setup.product.id } },
    });
    await waitFor(() => menu.instance.data.czVisible === true,
      () => `PAGE-U03 customize did not open: ${menu.instance.data.czVisible}`);
    const renderedCustomizeParts = components('u03-rendered-customize', true);
    const customize = simulate.render(renderedCustomizeParts.customize, {
      visible: true, item: menu.instance.data.czItem, init: null,
      flavorOptions: menu.instance.data.flavors, confirmLabel: menu.instance.data.czLabel,
    });
    customize.attach(document.body);
    await waitFor(() => customize.querySelectorAll('.chip').length === 8,
      () => `PAGE-U03 rendered flavors ended ${customize.querySelectorAll('.chip').length}`);
    for (const flavor of ['少饭', '酱汁分装']) {
      const chip = customize.querySelectorAll('.chip').find(node => node.dom.textContent.includes(flavor));
      if (!chip) throw new Error(`PAGE-U03 flavor ${flavor} was not rendered`);
      customize.instance.toggleFlavor({ currentTarget: { dataset: { f: flavor } } });
      await simulate.sleep(10);
    }
    if (!customize.querySelector('.cz-note')) throw new Error('PAGE-U03 item-note input was not rendered');
    customize.instance.onNote({ detail: { value: '菜品备注保留' } });
    const renderedCustomizeTree = customize.toJSON();
    if (!customize.querySelector('stepper') && !customize.querySelector('wx-stepper')
      && !treeHasTag(renderedCustomizeTree, 'stepper') && !treeHasTag(renderedCustomizeTree, 'wx-stepper')) {
      throw new Error('PAGE-U03 quantity stepper was not rendered');
    }
    customize.instance.add();
    if (!customize.querySelector('.cz-confirm')) throw new Error('PAGE-U03 customize confirmation was not rendered');
    pageDefinitions.menu.onCzConfirm.call(menu.instance, {
      detail: { qty: customize.instance.data.qty, flavors: customize.instance.data.flavors, note: customize.instance.data.note },
    });
    await waitFor(() => menu.instance.data.count === 2, () => `PAGE-U03 cart count stayed ${menu.instance.data.count}`);
    if (menu.instance.data.count !== 2 || app.globalData.cart[setup.product.id].flavors.length !== 2) {
      throw new Error('PAGE-U03 add-to-cart did not retain quantity/flavors');
    }

    const detail = render('detail', detailTemplate, 'u04', true, { id: setup.product.id });
    await waitFor(() => detail.instance.data.detailState !== 'loading', () => `PAGE-U04 detail stayed ${detail.instance.data.detailState}`);
    if (detail.instance.data.detailState !== 'ready' || detail.instance.data.m.images.length !== 3
      || !detail.dom.textContent.includes(setup.product.specification)
      || !detail.dom.textContent.includes('1 / 3')) {
      throw new Error(`PAGE-U04 multi-image detail was incomplete: ${detail.dom.textContent}`);
    }
    const detailImages = detail.querySelector('.detail-images');
    if (!detailImages || !pageDefinitions.detail.onImageChange.call(detail.instance, { detail: { current: 2 } })) {
      throw new Error('PAGE-U04 gallery was not actionable');
    }
    await waitFor(() => detail.instance.data.imageIndex === 2 && detail.dom.textContent.includes('3 / 3'),
      () => `PAGE-U04 gallery stayed ${detail.instance.data.imageIndex}`);
    const renderedSwiperTree = detailImages.toJSON();
    if (treeTagCount(renderedSwiperTree, 'image') + treeTagCount(renderedSwiperTree, 'wx-image') < 3) {
      throw new Error('PAGE-U04 three image hosts were not rendered');
    }
    if (!pageDefinitions.detail.previewImage.call(detail.instance, {
      currentTarget: { dataset: { url: detail.instance.data.m.images[2].url } },
    })) throw new Error('PAGE-U04 third image was not previewable');
    if (previewCalls.length !== 1 || previewCalls[0].urls.length !== 3
      || previewCalls[0].current !== detail.instance.data.m.images[2].url) {
      throw new Error(`PAGE-U04 preview evidence was ${JSON.stringify(previewCalls)}`);
    }

    const confirm = render('confirm', confirmTemplate, 'u05', true);
    await waitFor(() => confirm.instance.data.phoneState !== 'loading', () => `PAGE-U05 phone stayed ${confirm.instance.data.phoneState}`);
    if (confirm.instance.data.phoneState !== 'bound' || confirm.instance.data.count !== 2) {
      throw new Error(`PAGE-U05 checkout state was ${confirm.instance.data.phoneState}/${confirm.instance.data.count}`);
    }
    inputByKey(confirm, 'contact', 'PAGE-U05 联系人');
    inputByKey(confirm, 'extraPhone', setup.staff.phone);
    inputByKey(confirm, 'orderNote', 'PAGE-U05 整单备注');
    if (confirm.instance.data.form.contact !== 'PAGE-U05 联系人'
      || confirm.instance.data.form.extraPhone !== setup.staff.phone
      || confirm.instance.data.form.orderNote !== 'PAGE-U05 整单备注') {
      throw new Error(`PAGE-U05 rendered form did not reach production state: ${JSON.stringify(confirm.instance.data.form)}`);
    }
    const extraStart = observations.length;
    const extraSaved = await pageDefinitions.confirm.saveExtraPhone.call(confirm.instance);
    if (!extraSaved || !observations.slice(extraStart).some(item => item.path === '/api/v1/me/extra-phone' && item.status === 200)) {
      throw new Error('PAGE-U05 extra phone/name did not reach the root API');
    }
    const requestStart = observations.length;
    lastNavigation = null;
    if (!confirm.querySelector('.pay-btn')) throw new Error('PAGE-U05 payment control was not rendered');
    const paid = await pageDefinitions.confirm.pay.call(confirm.instance);
    await waitFor(() => lastNavigation || confirm.instance.data.paymentState === 'error',
      () => `PAGE-U05 payment stayed ${confirm.instance.data.paymentState}`);
    const transactionRequests = observations.slice(requestStart);
    if (!paid || !lastNavigation || !lastNavigation.startsWith('/pages/result/result?id=')
      || paymentCalls.length !== 1
      || !['/api/v1/quotes', '/api/v1/orders/prepay', '/api/v1/orders/confirm'].every(path =>
        transactionRequests.some(item => item.path === path && [200, 201].includes(item.status)))) {
      throw new Error(`PAGE-U05 transaction evidence was ${JSON.stringify(transactionRequests)}`);
    }

    const cancelled = await directRequest('POST', `/api/v1/orders/${setup.cancel_order_id}/cancel`, {
      reason: 'USER_REQUEST',
    }, `u08-cancel-${crypto.randomUUID()}`);
    if (!cancelled.order || cancelled.order.state !== 'REFUNDING') {
      throw new Error(`PAGE-U08 refund setup was ${JSON.stringify(cancelled)}`);
    }
    const orders = render('orders', ordersTemplate, 'u08', false);
    await waitFor(() => orders.instance.data.listState !== 'loading', () => `PAGE-U08 list stayed ${orders.instance.data.listState}`);
    const expectedTabs = ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款'];
    if (JSON.stringify(orders.instance.data.tabs) !== JSON.stringify(expectedTabs)
      || orders.instance.data.list.length !== 20 || !orders.instance.data.nextAfterID
      || !orders.instance.data.list.some(order => order.id === setup.cancel_order_id)) {
      throw new Error(`PAGE-U08 first page was ${orders.instance.data.list.length}/${orders.instance.data.nextAfterID}`);
    }
    const firstIDs = orders.instance.data.list.map(order => order.id);
    if (!orders.querySelector('.btn.btn--ghost-blue')) throw new Error('PAGE-U08 load-more control was not rendered');
    await pageDefinitions.orders.loadMore.call(orders.instance);
    await waitFor(() => orders.instance.data.loadingMore === false && orders.instance.data.list.length > 20,
      () => `PAGE-U08 pagination stayed ${orders.instance.data.list.length}/${orders.instance.data.loadingMore}`);
    if (new Set(orders.instance.data.list.map(order => order.id)).size !== orders.instance.data.list.length
      || !firstIDs.every(id => orders.instance.data.list.some(order => order.id === id))) {
      throw new Error('PAGE-U08 pagination duplicated or replaced the first page');
    }
    const stateByLabel = { 已预约: 'RESERVED', 制作中: 'PREPARING', 待取餐: 'READY_FOR_PICKUP', 已完成: 'COMPLETED', 已退款: 'REFUNDED' };
    for (const label of expectedTabs.slice(1)) {
      const before = observations.length;
      await pageDefinitions.orders.switchTab.call(orders.instance, { currentTarget: { dataset: { t: label } } });
      await waitFor(() => observations.length > before && orders.instance.data.loadingMore === false,
        () => `PAGE-U08 ${label} did not query`);
      const expectedState = stateByLabel[label];
      const request = observations.slice(before).find(item => item.path === '/api/v1/orders');
      if (!request || !request.query.includes(`state=${expectedState}`)
        || orders.instance.data.list.some(order => order.state !== expectedState)) {
        throw new Error(`PAGE-U08 ${label} evidence was ${JSON.stringify(request)}`);
      }
    }
    if (orders.instance.data.tabs.includes('退款中')) throw new Error('PAGE-U08 rendered a REFUNDING filter');

    const profile = render('profile', profileTemplate, 'u09', false);
    await waitFor(() => profile.instance.data.identityState !== 'loading', () => `PAGE-U09 identity stayed ${profile.instance.data.identityState}`);
    if (profile.instance.data.identityState !== 'ready' || !profile.instance.data.phoneMask
      || !profile.instance.data.merchantBound || !profile.querySelector('.switch-id')
      || !profile.querySelector('.prow.last')) {
      throw new Error(`PAGE-U09 server identity/contact was incomplete: ${profile.dom.textContent}`);
    }
    const beforeCosmetic = observations.length;
    const cosmetic = await pageDefinitions.profile.chooseProfile.call(profile.instance);
    await simulate.sleep(10);
    if (!cosmetic || observations.length !== beforeCosmetic || profileCalls.length !== 1
      || profile.instance.data.nick !== '本次微信昵称' || !profile.dom.textContent.includes('本次微信昵称')) {
      throw new Error('PAGE-U09 cosmetic profile changed server identity or was not visibly rendered');
    }
    const phoneInput = profile.querySelector('.extra-phone');
    const nameInput = profile.querySelector('.extra-name');
    if (!phoneInput || !nameInput || !profile.querySelector('.extra-save')) {
      throw new Error('PAGE-U09 extra-phone controls were not rendered');
    }
    pageDefinitions.profile.onExtraInput.call(profile.instance, {
      currentTarget: { dataset: { k: phoneInput.dom.dataset.k } }, detail: { value: setup.staff.phone },
    });
    pageDefinitions.profile.onExtraInput.call(profile.instance, {
      currentTarget: { dataset: { k: nameInput.dom.dataset.k } }, detail: { value: setup.staff.name },
    });
    await pageDefinitions.profile.saveExtraPhone.call(profile.instance);
    await waitFor(() => profile.instance.data.extraState !== 'saving', () => `PAGE-U09 extra phone stayed ${profile.instance.data.extraState}`);
    if (profile.instance.data.extraState !== 'matched' || profile.instance.data.pricingKind !== 'STAFF'
      || !profile.dom.textContent.includes('已命中员工名单')) {
      throw new Error(`PAGE-U09 extra phone ended ${profile.instance.data.extraState}/${profile.instance.data.pricingKind}`);
    }
    lastNavigation = null;
    const relogged = await pageDefinitions.profile.onMerchantPhone.call(profile.instance, { detail: { code: 'u09-merchant-phone' } });
    if (!relogged || lastNavigation !== '/pages/launch/launch') {
      throw new Error(`PAGE-U09 merchant switch ended ${relogged}/${lastNavigation || 'nothing'}`);
    }
  });
});

function render(name, template, suffix, includeProduct = false, loadOptions = {}) {
  document.body.innerHTML = '';
  currentRoute = `pages/${name}/${name}`;
  return renderPage({
    definition: pageDefinitions[name], template, id: `${name}-${suffix}`,
    usingComponents: components(`${name}-${suffix}`, includeProduct), loadOptions,
  });
}

function components(suffix, includeProduct) {
  const icon = registerComponent({ definition: iconDefinition, template: iconTemplate, id: `icon-${suffix}`, tagName: 'icon' });
  const values = {
    icon,
    navbar: registerComponent({ definition: navbarDefinition, template: navbarTemplate, id: `navbar-${suffix}`, tagName: 'navbar', usingComponents: { icon } }),
    tabbar: registerComponent({ definition: tabbarDefinition, template: tabbarTemplate, id: `tabbar-${suffix}`, tagName: 'tabbar', usingComponents: { icon } }),
    toast: registerComponent({ definition: toastDefinition, template: toastTemplate, id: `toast-${suffix}`, tagName: 'toast', usingComponents: { icon } }),
    pill: registerComponent({ definition: pillDefinition, template: pillTemplate, id: `pill-${suffix}`, tagName: 'pill' }),
    money: registerComponent({ definition: moneyDefinition, template: moneyTemplate, id: `money-${suffix}`, tagName: 'money' }),
  };
  if (includeProduct) {
    values.stepper = registerComponent({ definition: stepperDefinition, template: stepperTemplate, id: `stepper-${suffix}`, tagName: 'stepper' });
    const imageph = registerComponent({ definition: imagephDefinition, template: imagephTemplate, id: `imageph-${suffix}`, tagName: 'imageph' });
    values.imageph = imageph;
    values.customize = registerComponent({
      definition: customizeDefinition, template: customizeTemplate, id: `customize-${suffix}`, tagName: 'customize',
      usingComponents: { icon, imageph, money: values.money, stepper: values.stepper },
    });
  }
  return values;
}

function inputByKey(page, key, value) {
  const input = page.querySelectorAll('.field-in').find(node => node.dom.dataset.k === key);
  if (!input) throw new Error(`input ${key} was absent`);
  pageDefinitions.confirm.onInput.call(page.instance, {
    currentTarget: { dataset: { k: input.dom.dataset.k } }, detail: { value },
  });
}

async function directRequest(method, pathname, body, idempotencyKey) {
  const headers = {
    Accept: 'application/json', Authorization: `Bearer ${app.globalData.session.accessToken}`,
    'Content-Type': 'application/json', ...(idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : {}),
  };
  const response = await fetch(`${setup.api_origin}${pathname}`, { method, headers, body: JSON.stringify(body) });
  const result = await response.json();
  if (!response.ok) throw new Error(`${method} ${pathname} returned ${response.status}/${result && result.error && result.error.code || 'UNKNOWN'}`);
  return result;
}

function treeHasTag(node, tagName) {
  if (!node || typeof node !== 'object') return false;
  if (node.tagName === tagName) return true;
  return Array.isArray(node.children) && node.children.some(child => treeHasTag(child, tagName));
}

function treeTagCount(node, tagName) {
  if (!node || typeof node !== 'object') return 0;
  const own = node.tagName === tagName ? 1 : 0;
  return own + (Array.isArray(node.children)
    ? node.children.reduce((sum, child) => sum + treeTagCount(child, tagName), 0) : 0);
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 10000;
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error(message());
    await simulate.sleep(10);
  }
}
