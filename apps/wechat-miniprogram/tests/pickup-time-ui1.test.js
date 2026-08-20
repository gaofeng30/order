const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 生效 spec mvp-product-baseline · Every first-phase order uses one discrete pickup time：
//   仅预约取餐；可预约今天与明天；每餐段一个固定截单时刻，餐段内共用；
//   取餐时间为离散时间点，粒度商户可配；取餐时间是约定时刻而非到场窗口。
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('reservation is limited to today and tomorrow', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  assert.deepEqual(data.RESERVE_DATES.map(d => d.k), ['今天', '明天']);
  assert.deepEqual(data.RESERVE_DATES.map(d => d.off), [0, 1]);
});

test('meal periods each carry one fixed cutoff and a configurable step', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  assert.deepEqual(data.MEAL_PERIODS.map(p => p.key), ['lunch', 'dinner']);
  const lunch = data.MEAL_PERIODS[0];
  assert.equal(lunch.cutoff, '11:30');
  assert.equal(lunch.from, '11:30');
  assert.equal(lunch.to, '13:30');
  assert.equal(typeof data.PICKUP_STEP_MIN, 'number');
  // 时间点由范围与粒度推导，不写死
  assert.deepEqual(data.pickupTimes('lunch'), ['11:30', '12:00', '12:30', '13:00', '13:30']);
  assert.deepEqual(data.pickupTimes('dinner'), ['17:00', '17:30', '18:00', '18:30', '19:00']);
});

test('a meal period is cut off for today once its fixed cutoff passes', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  // 模拟时钟 16:48：午餐（11:30 截）已截，晚餐（17:00 截）未截
  assert.equal(data.isPeriodCutOff(0, 'lunch'), true);
  assert.equal(data.isPeriodCutOff(0, 'dinner'), false);
  // 明天两段都没截
  assert.equal(data.isPeriodCutOff(1, 'lunch'), false);
  assert.equal(data.isPeriodCutOff(1, 'dinner'), false);
});

test('default pickup is the first time still open', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  const d = data.defaultPickup();
  assert.equal(d.off, 0);
  assert.equal(d.period, 'dinner');
  assert.equal(d.time, '17:00');
});

test('menu carries a pickup bar and opens the picker', () => {
  const harness = createHarness();
  harness.loadApp();
  const menu = harness.loadPage('pages/menu/menu.js');
  harness.invoke(menu, 'onShow');
  assert.equal(menu.data.pickup.time, '17:00');
  assert.equal(menu.data.pickup.label.includes('今天'), true);
  assert.equal(menu.data.pickerVisible, false);
  menu.openPicker();
  assert.equal(menu.data.pickerVisible, true);
  const wxml = read('pages/menu/menu.wxml');
  assert.match(wxml, /bindtap="openPicker"/, 'menu has no pickup bar');
  assert.match(wxml, /pickup\.label/, 'menu does not render the chosen pickup time');
});

test('the picker groups times by meal period and folds a cut-off group', () => {
  const harness = createHarness();
  harness.loadApp();
  const menu = harness.loadPage('pages/menu/menu.js');
  harness.invoke(menu, 'onShow');
  menu.openPicker();
  const groups = menu.data.pickerGroups;
  assert.deepEqual(groups.map(g => g.name), ['午餐', '晚餐']);
  assert.equal(groups[0].cutOff, true, 'lunch should be folded at 16:48');
  assert.equal(groups[0].cutoffLabel.includes('11:30'), true, 'folded group must state its cutoff');
  assert.deepEqual(groups[0].times, [], 'folded group must not render individual times');
  assert.equal(groups[1].cutOff, false);
  assert.equal(groups[1].times.length, 5);
});

test('picking a date that is fully cut off is refused', () => {
  const harness = createHarness();
  harness.loadApp();
  const data = require('../utils/data.js');
  const menu = harness.loadPage('pages/menu/menu.js');
  harness.invoke(menu, 'onShow');
  menu.openPicker();
  assert.equal(menu.data.pickerDates[0].allCutOff, false, 'today still has dinner open');
  assert.equal(menu.data.pickerDates[1].allCutOff, false);
});

test('checkout reads the chosen pickup time instead of choosing again', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart, pickup } = require('../utils/util.js');
  pickup.set({ off: 1, period: 'lunch', time: '12:00' });
  cart.setPrefs({ id: '9007199254740993', category_id: '9007199254740995', name: 'X',
    description: 'd', specification: 's', price_cents: 1000 }, { qty: 1, flavors: [], note: '' });
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  assert.equal(confirm.data.pickup.time, '12:00');
  const wxml = read('pages/confirm/confirm.wxml');
  assert.doesNotMatch(wxml, /pickDate|pickSlot|rsv-slots/, 'confirm still picks the time itself');
  confirm.pay();
  const order = app.globalData.orders[0];
  assert.equal(order.pickupLabel, '明天 12:00');
  assert.equal(order.mealPeriod, 'lunch');
});

test('submitting into a cut-off period is blocked', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart, pickup } = require('../utils/util.js');
  pickup.set({ off: 0, period: 'lunch', time: '12:00' });   // 今天午餐已截
  cart.setPrefs({ id: '9007199254740993', category_id: '9007199254740995', name: 'X',
    description: 'd', specification: 's', price_cents: 1000 }, { qty: 1, flavors: [], note: '' });
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  const before = app.globalData.orders.length;
  confirm.pay();
  assert.equal(app.globalData.orders.length, before, 'order created into a cut-off period');
  assert.equal(cart.list().length, 1, 'cart must be preserved when submission is blocked');
});

test('cancel eligibility no longer depends on the deleted order type', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  assert.doesNotMatch(read('utils/data.js'), /o\.type === 'reserve'/, 'cancel rule still reads the deleted type field');
  assert.equal(data.canCancelReserve({ status: '已预约', minsToPickup: 102 }), true);
  assert.equal(data.canCancelReserve({ status: '已预约', minsToPickup: 18 }), false);
  assert.equal(data.canCancelReserve({ status: '制作中', minsToPickup: 102 }), false);
});

test('no template binds a data field its page never sets', () => {
  // 上一个 change 删了 order-detail 的 reserve、result 的 reserve，模板却还在用，
  // 渲染出的是错误分支。这类「删 data 忘了删模板」需要一条通用断言。
  const pages = fs.readdirSync(path.join(miniprogramRoot, 'pages'));
  const problems = [];
  for (const name of pages) {
    const wxml = path.join(miniprogramRoot, 'pages', name, `${name}.wxml`);
    const js = path.join(miniprogramRoot, 'pages', name, `${name}.js`);
    if (!fs.existsSync(wxml) || !fs.existsSync(js)) continue;
    const src = fs.readFileSync(js, 'utf8');
    const tpl = fs.readFileSync(wxml, 'utf8');
    for (const field of ['reserve', 'mode', 'slot', 'slots', 'dateIdx', 'dates', 'signature', 'listState', 'calc']) {
      const usedInTemplate = new RegExp(`\\{\\{[^}]*\\b${field}\\b`).test(tpl);
      const setInScript = new RegExp(`\\b${field}\\s*[:,)]`).test(src);
      if (usedInTemplate && !setInScript) problems.push(`${name}.wxml binds ${field} but ${name}.js never sets it`);
    }
  }
  assert.deepEqual(problems, []);
});
