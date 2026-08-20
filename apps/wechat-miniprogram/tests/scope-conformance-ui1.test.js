const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 生效 spec mvp-product-baseline 把会员等级与优惠券列入一期排除范围，
// 且明确不得以任何形式预留。以下断言逐项指名残留位置，删了一半会被精确点出来。
const EXCLUDED_ROUTES = [
  'pages/my-coupons/my-coupons',
  'pages/admin-levels/admin-levels',
  'pages/admin-members/admin-members',
  'pages/admin-member-edit/admin-member-edit',
  'pages/admin-member-import/admin-member-import',
  'pages/admin-coupons/admin-coupons',
  'pages/admin-coupon-edit/admin-coupon-edit',
];
const EXCLUDED_DIRS = EXCLUDED_ROUTES.map(route => path.dirname(route));
const EXCLUDED_MODULES = ['utils/promo.js', 'components/coupon-card'];
const EXCLUDED_GLOBALS = ['levels', 'members', 'coupons', 'couponUsed'];
const EXCLUDED_SEEDS = ['LEVELS', 'MEMBERS', 'COUPONS', 'MY_COUPON_USED'];
const EXCLUDED_API = /level|member|coupon/i;
const EXCLUDED_ORDER_FIELDS = ['levelName', 'levelLabel', 'levelCut', 'couponName', 'couponCut', 'totalCut'];

function read(rel) {
  return fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');
}

test('app.json exposes no route for an excluded capability', () => {
  const app = JSON.parse(read('app.json'));
  for (const route of EXCLUDED_ROUTES) {
    assert.equal(app.pages.includes(route), false, `app.json still lists ${route}`);
  }
  for (const dir of EXCLUDED_DIRS) {
    assert.equal(fs.existsSync(path.join(miniprogramRoot, dir)), false, `${dir} still exists`);
  }
});

test('excluded capability modules are absent', () => {
  for (const rel of EXCLUDED_MODULES) {
    assert.equal(fs.existsSync(path.join(miniprogramRoot, rel)), false, `${rel} still exists`);
  }
});

test('globalData holds no excluded capability state', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  for (const key of EXCLUDED_GLOBALS) {
    assert.equal(Object.hasOwn(app.globalData, key), false, `globalData.${key} still present`);
  }
});

test('seed and contract layers expose no excluded capability', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const seed of EXCLUDED_SEEDS) {
    assert.equal(Object.hasOwn(data, seed), false, `data.${seed} still exported`);
  }
  const api = require('../utils/api.js');
  const leaked = Object.keys(api).filter(name => EXCLUDED_API.test(name));
  assert.deepEqual(leaked, [], `api still exports ${leaked.join(', ')}`);
});

test('checkout completes with subtotal as payable and no discount fields', async () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart } = require('../utils/util.js');
  const item = {
    id: '9007199254740993',
    category_id: '9007199254740995',
    name: 'Scope Product',
    description: 'd',
    specification: 's',
    price_cents: 12345,
  };
  cart.setPrefs(item, { qty: 2, flavors: ['少盐'], note: 'keep' });

  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  await harness.flush(90);

  assert.equal(confirm.data.items[0].line_total_cents, 24690);
  assert.equal(confirm.data.payable_text, '246.90');
  for (const key of ['couponId', 'cpVisible', 'level', 'isMember', 'calc']) {
    assert.equal(Object.hasOwn(confirm.data, key), false, `confirm.data.${key} still present`);
  }
  assert.equal(typeof confirm.openCoupon, 'undefined', 'openCoupon handler still present');

  confirm.pay();
  const order = app.globalData.orders[0];
  assert.equal(order.items[0][0], item.id);
  for (const field of EXCLUDED_ORDER_FIELDS) {
    assert.equal(Object.hasOwn(order, field), false, `order.${field} still present`);
  }
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/result/result');
});

test('templates drop excluded capability entries', () => {
  const confirmWXML = read('pages/confirm/confirm.wxml');
  assert.doesNotMatch(confirmWXML, /openCoupon|coupon-card|等级折扣|优惠券/);
  assert.doesNotMatch(read('pages/confirm/confirm.js'), /promo/);

  const profileWXML = read('pages/profile/profile.wxml');
  assert.doesNotMatch(profileWXML, /我的优惠券|levelName|levelLabel|couponCount/);

  // 商户中心与菜品编辑页均已迁往 PC，排除能力的缺席由页面本身不存在保证
  assert.equal(fs.existsSync(path.join(miniprogramRoot, 'pages/admin-profile')), false);
  assert.equal(fs.existsSync(path.join(miniprogramRoot, 'pages/admin-product-edit')), false);
});
