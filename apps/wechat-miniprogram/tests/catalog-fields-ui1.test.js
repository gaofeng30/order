const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 评审记录 §9（不展示标签）、§10（不展示过敏原）、§16（不展示月售）、§3（不做数量库存）。
// 生效 spec mvp-product-baseline 的 Product availability uses a per-service-date sellout
// switch 规定商品可售性只由上下架与按取餐日期的售罄开关决定，不得有份数或剩余量。
const RETIRED_FIELDS = ['tags', 'allergens', 'sold', 'stock'];
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('seed products carry no retired catalog field', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const product of data.MENU) {
    for (const field of RETIRED_FIELDS) {
      assert.equal(Object.hasOwn(product, field), false, `${product.id}.${field} still seeded`);
    }
    assert.equal(Object.hasOwn(product, 'status'), true, `${product.id} lost its sale status`);
  }
});

test('product contract neither accepts nor produces a quantity', () => {
  createHarness().loadApp();
  const api = require('../utils/api.js');
  const src = read('utils/api.js');
  assert.doesNotMatch(src, /\bstock\b|库存/, 'api.js still handles a quantity');
  assert.doesNotMatch(src, /\btags\b|\ballergens\b|\bsold\b/, 'api.js still seeds a retired field');
  assert.equal(typeof api.setProductStatus, 'function', 'sale-status contract broken');
});

test('merchant product list drops quantity and monthly sales', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  harness.invoke(page, 'onShow');
  const row = page.data.list[0];
  for (const field of [...RETIRED_FIELDS, 'low']) {
    assert.equal(Object.hasOwn(row, field), false, `admin list row still carries ${field}`);
  }
  assert.equal(app.globalData.menu.length > 0, true);
  const wxml = read('pages/admin-products/admin-products.wxml');
  assert.doesNotMatch(wxml, /库存|月售|item\.stock\b|item\.sold\b|item\.low\b/, 'admin list still renders a retired field');
  assert.match(wxml, /toggleSoldout/, 'admin list lost the sale-status control');
});

test('merchant product editor has no quantity field', () => {
  const js = read('pages/admin-product-edit/admin-product-edit.js');
  const wxml = read('pages/admin-product-edit/admin-product-edit.wxml');
  assert.doesNotMatch(js, /\bstock\b/, 'editor script still holds a quantity');
  assert.doesNotMatch(wxml, /库存|f\.stock/, 'editor template still renders a quantity');
});

test('sale status still works end to end after the fields are gone', async () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const api = require('../utils/api.js');
  const id = app.globalData.menu[0].id;
  await api.setProductStatus(id, 'soldout');
  assert.equal(app.globalData.menu[0].status, 'soldout');
  await api.setProductStatus(id, 'on');
  assert.equal(app.globalData.menu[0].status, 'on');
});
