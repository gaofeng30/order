const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 生效 spec mvp-product-baseline：
//   Orders use one six-state production state machine
//   Every first-phase order uses one discrete pickup time（一期无即时取餐）
//   Cancellation and refund rules are deterministic（无待支付、无已取消）
const SIX = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款'];
const RETIRED_STATES = ['待支付', '已支付待接单', '已取消', '待制作', '异常'];
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('status map covers exactly the six production states', () => {
  createHarness().loadApp();
  const { STATUS_MAP, statusTone } = require('../utils/util.js');
  for (const s of SIX) assert.equal(Object.hasOwn(STATUS_MAP, s), true, `STATUS_MAP lost ${s}`);
  for (const s of RETIRED_STATES) assert.equal(Object.hasOwn(STATUS_MAP, s), false, `STATUS_MAP still maps ${s}`);
  assert.equal(statusTone('退款中'), 'warn');
  assert.equal(statusTone('已退款'), 'mute');
});

test('merchant advancement is single-directional with no undo', () => {
  createHarness().loadApp();
  const util = require('../utils/util.js');
  assert.deepEqual(util.NEXT, { 制作中: '待取餐', 待取餐: '已完成' });
  // 断言的是「撤销能力」不存在，不是「撤销」二字不出现——注释里说明禁止撤销是正当的
  assert.doesNotMatch(read('utils/util.js'), /onUndo/, 'util.js still offers an undo');
  assert.doesNotMatch(read('components/toast/toast.js'), /onUndo|undoable/, 'toast still supports undo');
  assert.doesNotMatch(read('components/toast/toast.wxml'), /撤销/, 'toast template still renders an undo action');
  // 已预约 → 制作中 由服务端定时推进，前端不得提供该转换
  assert.equal(Object.hasOwn(util.NEXT, '已预约'), false, 'frontend must not advance 已预约');
});

test('advancing an order never walks backwards', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const util = require('../utils/util.js');
  const order = app.globalData.aOrders.find(o => o.status === '制作中');
  assert.ok(order, 'seed has no 制作中 order');
  util.advanceOrder(order.id, null, null);
  assert.equal(order.status, '待取餐');
  util.advanceOrder(order.id, null, null);
  assert.equal(order.status, '已完成');
  util.advanceOrder(order.id, null, null);
  assert.equal(order.status, '已完成', 'terminal state advanced further');
});

test('seed orders use only the six states and carry no order type', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const o of [...data.USER_ORDERS, ...data.ADMIN_ORDERS]) {
    assert.equal(SIX.includes(o.status), true, `${o.id} has retired status ${o.status}`);
    assert.equal(Object.hasOwn(o, 'type'), false, `${o.id} still carries an order type`);
  }
});

test('immediate ordering is gone from every surface', () => {
  createHarness().loadApp();
  const util = require('../utils/util.js');
  assert.equal(Object.hasOwn(util, 'orderMode'), false, 'util still exports orderMode');
  for (const rel of ['pages/home/home.js', 'pages/confirm/confirm.js', 'pages/confirm/confirm.wxml',
                     'pages/result/result.js', 'pages/result/result.wxml',
                     'pages/orders/orders.js', 'pages/order-detail/order-detail.js']) {
    assert.doesNotMatch(read(rel), /orderMode|'now'|尽快|到店点单/, `${rel} still offers immediate ordering`);
    // 订单类型字段随即时单删除，任何页面都不得再读它
    assert.doesNotMatch(read(rel), /o\.type|item\.type|order\.type/, `${rel} still reads the deleted order type`);
  }
});

test('checkout creates a reserved order with a pickup time', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart, pickup } = require('../utils/util.js');
  // 显式选定取餐时间：不依赖默认值，否则该用例会随默认规则变化而漂移
  pickup.set({ off: 1, period: 'lunch', time: '12:00' });
  cart.setPrefs({ id: '9007199254740993', category_id: '9007199254740995', name: 'X', description: 'd', specification: 's', price_cents: 1000 },
    { qty: 1, flavors: [], note: '' });
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  confirm.pay();
  const order = app.globalData.orders[0];
  assert.equal(order.status, '已预约');
  assert.equal(Object.hasOwn(order, 'type'), false);
  // pickupLabel 已按 §15.6.2 删除；取餐文案改由 pickupDate + pickupTime 现算
  assert.equal(Object.hasOwn(order, 'pickupLabel'), false, 'order still stores a frozen pickup label');
  const label = require('../utils/data.js').orderPickupLabel(order);
  assert.ok(label && !/尽快/.test(label), 'order has no reserved pickup label');
});

test('checkout inside the 30-minute window creates a producing order', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart, pickup } = require('../utils/util.js');
  const data = require('../utils/data.js');
  // 演示时钟 16:48，今天晚餐 17:00 距取餐 12 分钟 —— 按 §7.4 支付成功即进 制作中
  pickup.set({ off: 0, period: 'dinner', time: '17:00' });
  assert.ok(data.slotMins(0, '17:00') <= data.CANCEL_LIMIT_MIN);
  cart.setPrefs({ id: '9007199254740993', category_id: '9007199254740995', name: 'X', description: 'd', specification: 's', price_cents: 1000 },
    { qty: 1, flavors: [], note: '' });
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  confirm.pay();
  assert.equal(app.globalData.orders[0].status, '制作中');
});

test('user order filters expose only first-phase states', () => {
  const harness = createHarness();
  harness.loadApp();
  const orders = harness.loadPage('pages/orders/orders.js');
  harness.invoke(orders, 'onShow');
  const tabs = orders.data.tabs.map(t => (typeof t === 'string' ? t : t.k || t.label));
  for (const retired of ['待支付', '已取消']) {
    assert.equal(tabs.includes(retired), false, `order filter still offers ${retired}`);
  }
  assert.equal(tabs.includes('已退款'), true, 'order filter lost 已退款');
});

test('every surface that names a state uses the six-state vocabulary', () => {
  // 独立验证发现首轮断言只覆盖了 util/种子/结算/订单筛选，漏掉了商户端泳道、
  // TabBar 角标、核销判定与取消流程。此处按「凡是出现状态名的地方」逐一覆盖。
  const SURFACES = [
    'pages/admin-orders/admin-orders.js',
    'pages/admin-verify/admin-verify.js',
    'pages/order-detail/order-detail.js',
    'pages/orders/orders.wxml',
    'components/tabbar/tabbar.js',
    'utils/util.js',
    'utils/data.js',
  ];
  for (const rel of SURFACES) {
    for (const retired of RETIRED_STATES) {
      assert.doesNotMatch(read(rel), new RegExp(`'${retired}'|「${retired}」|status === '${retired}'`),
        `${rel} still names ${retired}`);
    }
  }
});

test('merchant lanes and tab badges follow the six states', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  harness.invoke(page, 'onLoad', {});
  assert.deepEqual(page.data.lanes, ['已预约', '制作中', '待取餐', '已完成', '已退款', '全部']);
  assert.equal(page.data.lane, '已预约');
  const src = read('components/tabbar/tabbar.js');
  assert.doesNotMatch(src, /'待制作'|'待支付'/, 'tab badge still counts a retired state');
  assert.match(src, /'制作中'/, 'merchant badge lost 制作中');
});

test('cancelling a paid order moves it to refunding, never to a deleted state', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const target = app.globalData.orders.find(o => o.status === '已预约');
  assert.ok(target, 'seed has no 已预约 order');
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  harness.invoke(page, 'onLoad', { id: target.id });
  page.doCancel();
  const after = app.globalData.orders.find(o => o.id === target.id);
  assert.equal(after.status, '退款中');
  // 断言的是「写入」，不是「词出现」——注释里说明一期没有该状态是正当的
  assert.doesNotMatch(read('pages/order-detail/order-detail.js'), /status: '已取消'/,
    'cancel still writes a deleted state');
});

test('no template calls a handler its page no longer defines', () => {
  const wxml = read('pages/orders/orders.wxml');
  const js = read('pages/orders/orders.js');
  const handlers = [...wxml.matchAll(/(?:bind|catch)tap="(\w+)"/g)].map(m => m[1]);
  for (const h of handlers) {
    assert.match(js, new RegExp(`\\b${h}\\s*\\(`), `orders.wxml calls ${h} but orders.js does not define it`);
  }
});

test('the pickup QR code is bound to the ready state only', () => {
  const wxml = read('pages/order-detail/order-detail.wxml');
  assert.match(wxml, /status === '待取餐'/, 'order detail no longer gates the QR code on 待取餐');
});
