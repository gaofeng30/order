const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 评审记录 §1（不展示洗衣洗车）、§34（删营销 Banner）、§35（删入群与会员中心）、
// §36 批注（推荐商品都删掉）、§38（删商户端工作台，底部只保留订单/核销/菜品）。
const REMOVED_ROUTES = ['pages/brand/brand', 'pages/admin-dashboard/admin-dashboard'];
const ADMIN_TABS = ['admin-orders', 'admin-verify', 'admin-products'];
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('retired entry screens are gone from routes and disk', () => {
  const app = JSON.parse(read('app.json'));
  for (const route of REMOVED_ROUTES) {
    assert.equal(app.pages.includes(route), false, `app.json still lists ${route}`);
    assert.equal(fs.existsSync(path.join(miniprogramRoot, path.dirname(route))), false, `${route} dir still exists`);
  }
  assert.doesNotMatch(read('utils/util.js'), /toBrand|pages\/brand/, 'util.js still routes to the brand screen');
});

test('home drops every marketing surface', () => {
  const js = read('pages/home/home.js');
  const wxml = read('pages/home/home.wxml');
  for (const token of ['CAMPAIGNS', 'bannerIdx', 'onBanner', 'dotTap', 'openBanner', 'signature']) {
    assert.doesNotMatch(js, new RegExp(token), `home.js still carries ${token}`);
  }
  assert.doesNotMatch(wxml, /banner|swiper|dots|入群|会员|今日招牌|signature/,
    'home.wxml still renders a retired marketing surface');
});

test('home grid keeps only first-phase entries', () => {
  const harness = createHarness();
  harness.loadApp();
  const home = harness.loadPage('pages/home/home.js');
  const keys = home.data.grid.map(g => g.k);
  assert.equal(keys.includes('member'), false, 'home grid still offers 会员中心');
  assert.equal(keys.includes('service'), false, 'home grid still offers 联系客服');
  for (const kept of ['reserve', 'orders', 'pickup']) {
    assert.equal(keys.includes(kept), true, `home grid lost ${kept}`);
  }
});

test('merchant tab bar collapses to three tabs', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  app.globalData.aOrders = [];
  const src = read('components/tabbar/tabbar.js');
  assert.doesNotMatch(src, /admin-dashboard|admin-profile/, 'tabbar still lists a removed merchant tab');
  for (const id of ADMIN_TABS) {
    assert.match(src, new RegExp(`'${id}'`), `tabbar lost ${id}`);
  }
});

test('merchant center stays reachable after the tab bar shrinks', () => {
  const wxml = read('pages/admin-orders/admin-orders.wxml');
  assert.match(wxml, /admin-profile|toProfile/, 'admin-profile became unreachable');
  assert.equal(fs.existsSync(path.join(miniprogramRoot, 'pages/admin-profile')), true);
});

test('seed drops data owned only by the removed dashboard', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  assert.equal(Object.hasOwn(data, 'RANK'), false, 'data.js still exports RANK');
});

test('home no longer touches the catalog', async () => {
  const harness = createHarness();
  harness.loadApp();
  const home = harness.loadPage('pages/home/home.js');
  harness.invoke(home, 'onShow');
  await harness.flush(90);
  assert.equal(harness.requestCalls.length, 0, 'home still requests the catalog');
  assert.equal(Object.hasOwn(home.data, 'listState'), false, 'home still carries a list state');
  assert.equal(typeof home.retryCatalog, 'undefined', 'home still exposes a catalog retry');
});

test('the identity screen routes merchants to a page that exists', () => {
  const wxml = read('pages/launch/launch.wxml');
  assert.doesNotMatch(wxml, /admin-dashboard/, 'identity screen still routes to the removed dashboard');
  assert.match(wxml, /data-to="admin-orders"/, 'identity screen lost the merchant entry');
});

test('the launch-layer editor previews only screens that exist', () => {
  const js = read('pages/admin-layer/admin-layer.js');
  const wxml = read('pages/admin-layer/admin-layer.wxml');
  assert.doesNotMatch(js, /'brand'|switchMock/, 'layer editor still targets the removed brand screen');
  assert.doesNotMatch(wxml, /mock === 'brand'|业务选择页/, 'layer editor still previews the removed brand screen');
});
