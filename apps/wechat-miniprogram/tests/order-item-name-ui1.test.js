const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// 0818 PRD §15.6.2：items = [id, name, qty, price, discountedPrice, flavors?, note?]
//                   name 是下单当刻固化的快照，渲染不按 id 回查商品表。
// 0818 PRD §5.6：订单固化原价、折扣率、折后价、身份与价格版本。

// 服务端目录形态的商品：数字 id，本地种子 MENU 里查不到这个 id。
const CATALOG_ITEM = {
  id: '1', category_id: '9', name: '目录来的双拼饭',
  description: 'd', specification: 's', price_cents: 3200,
};

function placeOrder() {
  const harness = createHarness();
  const app = harness.loadApp();
  require('../utils/util.js').cart.add(CATALOG_ITEM);
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  confirm.pay();
  return { harness, app, order: app.globalData.orders[0] };
}

test('checkout snapshots the product name into the order line', () => {
  const { order } = placeOrder();
  const [id, name, qty, price, discounted] = order.items[0];
  assert.equal(id, '1');
  assert.equal(name, '目录来的双拼饭');
  assert.equal(qty, 1);
  assert.equal(price, 3200);
  assert.equal(discounted, 3200);
});

test('the order list opens on a product the local seed does not know', () => {
  const { harness, order } = placeOrder();
  const list = harness.loadPage('pages/orders/orders.js');
  harness.invoke(list, 'onShow');                       // 改动前此处抛 undefined.name
  const row = list.data.list.find(o => o.id === order.id);
  assert.ok(row, 'the fresh order is missing from the list');
  assert.equal(row.summary, '目录来的双拼饭×1');
});

test('the order detail opens and falls back to a placeholder image', () => {
  const { harness, order } = placeOrder();
  const detail = harness.loadPage('pages/order-detail/order-detail.js');
  harness.invoke(detail, 'onLoad', { id: order.id });   // 改动前此处抛 undefined.name
  assert.equal(detail.data.rows.length, 1);
  assert.equal(detail.data.rows[0].name, '目录来的双拼饭');
  assert.equal(detail.data.rows[0].m, null, '订单没有固化图片，本地目录查不到时应回落占位图');
});

test('a fresh order and a seeded order agree on field types', () => {
  const { app, order } = placeOrder();
  const seeded = app.globalData.orders.find(o => o.id !== order.id);
  // subtotal / discountRate 等字段种子订单尚未携带，补齐属 align-miniprogram-order-model
  for (const k of ['total', 'code', 'status']) {
    assert.equal(typeof order[k], typeof seeded[k], `${k} type drifted`);
  }
  assert.equal(order.total, 3200);
});

test('renaming the catalog does not rewrite history', () => {
  const { harness, app, order } = placeOrder();
  // 商品改名并从本地目录移除，模拟目录随时间变化
  app.globalData.menu = app.globalData.menu.filter(m => m.id !== '1');
  const list = harness.loadPage('pages/orders/orders.js');
  harness.invoke(list, 'onShow');
  const row = list.data.list.find(o => o.id === order.id);
  assert.equal(row.summary, '目录来的双拼饭×1', '历史订单显示的名称随目录变了');
});

test('every seeded order line carries a name snapshot', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const o of [...data.USER_ORDERS, ...data.ADMIN_ORDERS]) {
    for (const line of o.items) {
      assert.ok(line.length >= 5, `${o.id} line is not a 5+ tuple: ${JSON.stringify(line)}`);
      assert.equal(typeof line[1], 'string', `${o.id} line has no name: ${JSON.stringify(line)}`);
      assert.ok(line[1].trim(), `${o.id} line carries an empty name`);
      assert.equal(Number.isInteger(line[2]), true, `${o.id} qty is not an integer`);
    }
  }
});

test('the merchant surfaces read the snapshot, not the product table', () => {
  const harness = createHarness();
  const app = harness.loadApp();
  const target = app.globalData.aOrders.find(o => o.status === '待取餐');
  app.globalData.menu = [];                             // 菜品表清空
  const detail = harness.loadPage('pages/admin-order-detail/admin-order-detail.js');
  harness.invoke(detail, 'onLoad', { id: target.id });
  assert.ok(detail.data.rows.every(r => r.name && r.name.trim()), '商户端订单详情丢了名称');
  const verify = harness.loadPage('pages/admin-verify/admin-verify.js');
  harness.invoke(verify, 'onLoad', {});
  verify.tryVerify(target.code);
  assert.ok(verify.data.match.rows.every(r => r.name && r.name.trim()), '核销页丢了名称');
});
