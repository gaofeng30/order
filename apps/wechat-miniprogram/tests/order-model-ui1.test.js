const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// 0818 PRD §15.6.2 订单模型；§5.6 金额一律以整数分保存与计算；
// §7.6 取餐前 30 分钟外可自助取消；§15.6.4 口味绑定在菜品行内。

// 3250 分：走元的往返会得到 32.5，是浮点尾数的来源
const HALF_YUAN = { id: '1', category_id: '9', name: '半元定价菜',
                    description: 'd', specification: 's', price_cents: 3250 };

function placeOrder(product) {
  const harness = createHarness();
  const app = harness.loadApp();
  require('../utils/util.js').cart.add(product || HALF_YUAN);
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  confirm.pay();
  return { harness, app, order: app.globalData.orders[0] };
}

test('checkout writes整数分 and settles exactly', () => {
  const { order } = placeOrder();
  assert.equal(order.subtotal, 3250);
  assert.equal(order.total, 3250);
  assert.equal(order.discountCut, 0);
  assert.equal(order.subtotal - order.discountCut, order.total);
  assert.equal(order.items[0][3], 3250);
  assert.equal(Number.isInteger(order.total), true, '金额不是整数分');
});

test('every seeded order settles in cents', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const o of [...data.USER_ORDERS, ...data.ADMIN_ORDERS]) {
    const sub = o.items.reduce((s, it) => s + it[2] * it[3], 0);
    const paid = o.items.reduce((s, it) => s + it[2] * it[4], 0);
    assert.equal(sub, o.subtotal, `${o.id} 行小计对不上 subtotal`);
    assert.equal(paid, o.total, `${o.id} 行折后价对不上 total`);
    assert.equal(o.subtotal - o.discountCut, o.total, `${o.id} 不结算`);
    assert.ok(o.total >= 1000, `${o.id}.total = ${o.total} 仍是元`);
  }
});

test('the money formatter is the only conversion', () => {
  const { yuan } = require('../utils/money.js');
  assert.equal(yuan(3250), '32.50');
  assert.equal(yuan(7), '0.07');
  assert.equal(yuan(0), '0.00');
  assert.equal(yuan(-1250), '-12.50');
  for (const bad of [null, undefined, '', NaN, 'abc']) assert.equal(yuan(bad), '—');
});

test('the cancel window follows the clock, not a stored field', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  const far = { status: '已预约', pickupDate: data.BUSINESS_DAY, pickupTime: '18:30' };
  const near = { status: '已预约', pickupDate: data.BUSINESS_DAY, pickupTime: '17:06' };
  assert.equal(data.minsToPickup(far), 102);          // NOW_MINS = 16:48
  assert.equal(data.minsToPickup(near), 18);
  assert.equal(data.canCancelReserve(far), true);
  assert.equal(data.canCancelReserve(near), false);
  // 记录上塞一个陈旧值也不该改变判定
  assert.equal(data.canCancelReserve({ ...near, minsToPickup: 999 }), false);
});

test('no order stores a frozen pickup label or time to pickup', () => {
  const { app, order } = placeOrder();
  const data = require('../utils/data.js');
  for (const o of [order, ...app.globalData.orders, ...app.globalData.aOrders]) {
    assert.equal(Object.hasOwn(o, 'pickupLabel'), false, `${o.id} 仍带 pickupLabel`);
    assert.equal(Object.hasOwn(o, 'minsToPickup'), false, `${o.id} 仍带 minsToPickup`);
  }
  assert.equal(data.orderPickupLabel(app.globalData.orders[1]), '今天 17:00');
  assert.equal(data.orderPickupLabel(app.globalData.orders.find(o => o.id === 'o3')), '昨天 12:30');
});

test('order level carries only the note, per-item flavours survive', () => {
  const { harness, app, order } = placeOrder();
  for (const o of [order, ...app.globalData.orders, ...app.globalData.aOrders]) {
    assert.equal(Object.hasOwn(o, 'flavor'), false, `${o.id} 仍带整单级 flavor`);
    assert.equal(Object.hasOwn(o, 'flavors'), false, `${o.id} 仍带整单级 flavors`);
    assert.equal(Object.hasOwn(o, 'orderNote'), true, `${o.id} 没有 orderNote`);
  }
  // a0 的口味 少盐 原本挂在整单级，现在在第一行
  const a0 = app.globalData.aOrders.find(o => o.id === 'a0');
  assert.equal(a0.items[0][5], '少盐');
  const list = harness.loadPage('pages/admin-orders/admin-orders.js');
  harness.invoke(list, 'onShow');
  list.switchLane({ currentTarget: { dataset: { l: '全部' } } });
  const row = list.data.list.find(r => r.id === 'a0');
  assert.ok(row.band.includes('少盐'), `行内口味没有展示出来: ${row.band}`);
  assert.ok(row.band.includes('预约 18:00 取'), `整单备注没有展示出来: ${row.band}`);
});

test('user surfaces render a cents order end to end', () => {
  const { harness, order } = placeOrder();
  const list = harness.loadPage('pages/orders/orders.js');
  harness.invoke(list, 'onShow');
  const row = list.data.list.find(o => o.id === order.id);
  assert.equal(row.total, 3250, '列表拿到的仍不是整数分');
  assert.match(row.timeText, /^预约 (今天|明天) \d{2}:\d{2} · /);
  const detail = harness.loadPage('pages/order-detail/order-detail.js');
  harness.invoke(detail, 'onLoad', { id: order.id });
  assert.equal(detail.data.rows[0].p, 3250);
  assert.equal(detail.data.rows[0].sub, 3250);
  assert.match(detail.data.pickupText, /^(今天|明天) \d{2}:\d{2}$/);
});
