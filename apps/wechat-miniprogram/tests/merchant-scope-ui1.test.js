const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 2026-08-19 客户评审 §26：完整商品配置、退款、名单和财务功能留在 PC 后台。
// §38：底部只保留订单 / 核销 / 菜品；菜品页只允许切换可售 / 售罄。
// 0818 PRD §3.5：小程序商户端共 4 屏。
const MERCHANT_SCREENS = ['admin-orders', 'admin-order-detail', 'admin-verify', 'admin-products'];
const PC_ONLY_SCREENS = ['admin-product-edit', 'admin-categories', 'admin-settings', 'admin-layer', 'admin-profile'];
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('the mini program merchant surface is exactly four screens', () => {
  const app = JSON.parse(read('app.json'));
  const merchant = app.pages.filter(p => p.includes('admin-')).map(p => p.split('/')[1]);
  assert.deepEqual(merchant.sort(), [...MERCHANT_SCREENS].sort());
  for (const s of PC_ONLY_SCREENS) {
    assert.equal(fs.existsSync(path.join(miniprogramRoot, 'pages', s)), false, `${s} still exists`);
  }
});

test('the product screen offers only the sell-out toggle', () => {
  const harness = createHarness();
  harness.loadApp();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  harness.invoke(page, 'onShow');
  for (const forbidden of ['newProduct', 'edit', 'toggleShelf']) {
    assert.equal(typeof page[forbidden], 'undefined', `product screen still exposes ${forbidden}`);
  }
  assert.equal(typeof page.toggleSoldout, 'function', 'product screen lost the sell-out toggle');
  const wxml = read('pages/admin-products/admin-products.wxml');
  assert.doesNotMatch(wxml, /newProduct|"edit"|toggleShelf|上架新菜|编辑/, 'product template still offers配置 entries');
  assert.match(wxml, /toggleSoldout/, 'product template lost the sell-out toggle');
});

test('the sell-out toggle still works end to end', async () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  harness.invoke(page, 'onShow');
  /* 售罄按取餐日期写独立记录，不动 status（§6.5、§15.6.1）。 */
  const data = require('../utils/data.js');
  const id = app.globalData.menu[0].id;
  const shelf = app.globalData.menu[0].status;
  const day = data.BUSINESS_DAY;
  const before = data.isSoldOut(id, day);
  page.toggleSoldout({ currentTarget: { dataset: { id } } });
  await harness.flush(90);
  assert.equal(app.globalData.menu[0].status, shelf, '售罄开关改动了上下架');
  assert.notEqual(data.isSoldOut(id, day), before, '售罄记录没有变化');
  assert.equal(data.isSoldOut(id, '2026-08-22'), false, '当日售罄影响了次日');
});

test('business status can be switched from the orders screen', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  harness.invoke(page, 'onShow');
  assert.deepEqual(page.data.biz, ['营业中', '休息中', '已截单']);
  page.setBiz({ currentTarget: { dataset: { b: '休息中' } } });
  assert.equal(app.globalData.store.status, '休息中');
  assert.equal(page.data.storeStatus, '休息中');
  assert.match(read('pages/admin-orders/admin-orders.wxml'), /setBiz/, 'orders screen has no status switch');
});

test('the orders screen returns to the identity screen', () => {
  const harness = createHarness();
  harness.loadApp();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  assert.equal(typeof page.reset, 'function', 'orders screen cannot return to the identity screen');
  assert.equal(typeof page.toProfile, 'undefined', 'orders screen still routes to a removed merchant centre');
  assert.doesNotMatch(read('pages/admin-orders/admin-orders.wxml'), /toProfile|商户中心/);
});

test('no merchant surface references a removed screen or placeholder', () => {
  const walk = dir => fs.readdirSync(dir, { withFileTypes: true }).flatMap(e => {
    const p = path.join(dir, e.name);
    return e.isDirectory() ? walk(p) : [p];
  });
  const files = walk(path.join(miniprogramRoot, 'pages'))
    .concat(walk(path.join(miniprogramRoot, 'components')))
    .concat([path.join(miniprogramRoot, 'utils', 'util.js')]);
  for (const f of files) {
    const src = fs.readFileSync(f, 'utf8');
    const rel = path.relative(miniprogramRoot, f);
    for (const gone of PC_ONLY_SCREENS) {
      assert.equal(src.includes(gone), false, `${rel} still references ${gone}`);
    }
    for (const placeholder of ['交班对账', '成员与权限', '建设中']) {
      assert.equal(src.includes(placeholder), false, `${rel} still renders placeholder ${placeholder}`);
    }
  }
});
