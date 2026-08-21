const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// 0818 PRD §6.6：商户端订单列表提供按取餐号、订单号、手机号搜索；
//                手工输入只匹配当前营业日期的订单。
// 0818 PRD §7.8：取餐号为 4 位数字，按取餐日期从 0001 累计，跨营业日可能重复。

function orders() {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  harness.invoke(page, 'onLoad', {});
  harness.invoke(page, 'onShow');
  return { harness, app, page };
}
const type = (page, value) => page.onKw({ detail: { value } });
const ids = page => page.data.list.map(o => o.id);

test('a pickup code reported at the window finds the order from any lane', () => {
  const { page } = orders();
  page.switchLane({ currentTarget: { dataset: { l: '已预约' } } });
  type(page, '0090');                       // a7，已完成
  assert.deepEqual(ids(page), ['a7']);
  page.switchLane({ currentTarget: { dataset: { l: '已完成' } } });
  type(page, '0131');                       // a1，制作中
  assert.deepEqual(ids(page), ['a1']);
});

test('an order number and a phone fragment both locate the order', () => {
  const { page } = orders();
  type(page, 'SA2406100112');
  assert.deepEqual(ids(page), ['a6']);
  type(page, '3322');                       // 手机尾号，不是取餐号
  assert.deepEqual(ids(page).sort(), ['a0', 'a8']);
});

test('choosing a lane leaves search and restores the lane', () => {
  const { app, page } = orders();
  type(page, 'SA2406100112');
  assert.equal(page.data.kw, 'SA2406100112');
  page.switchLane({ currentTarget: { dataset: { l: '制作中' } } });
  assert.equal(page.data.kw, '');
  assert.equal(page.data.hint, '');
  assert.deepEqual(ids(page).sort(),
    app.globalData.aOrders.filter(o => o.status === '制作中').map(o => o.id).sort());
});

test('a code from another business day reports the fact instead of an empty list', () => {
  const { page } = orders();
  type(page, '0203');                       // a10，属于 2026-08-20
  assert.deepEqual(ids(page), []);
  assert.match(page.data.hint, /2026-08-20/);
  assert.match(page.data.hint, /订单号|手机号/);
});

test('a repeated pickup code resolves to the current business day only', () => {
  const { page } = orders();
  type(page, '0118');                       // a5 (08-21) 与 a9 (08-20) 同号
  assert.deepEqual(ids(page), ['a5']);
  assert.equal(page.data.hint, '');
});

test('manual verification refuses a stale code and names its business day', () => {
  const harness = createHarness();
  harness.loadApp();
  const page = harness.loadPage('pages/admin-verify/admin-verify.js');
  harness.invoke(page, 'onLoad', {});
  page.tryVerify('0203');
  assert.equal(page.data.match, null, 'a stale code opened the verification sheet');
  assert.match(harness.toastCalls.at(-1).message, /2026-08-20/);
  page.tryVerify('0118');
  assert.ok(page.data.match, 'the current business day code did not resolve');
  assert.equal(page.data.match.o.id, 'a5');
  assert.equal(page.data.match.err, '');
});

test('an order under refund is reachable without a lane of its own', () => {
  const { page } = orders();
  assert.deepEqual(page.data.lanes, ['已预约', '制作中', '待取餐', '已完成', '已退款', '全部']);
  type(page, 'SA2406100102');
  assert.deepEqual(ids(page), ['a11']);
  assert.equal(page.data.list[0].status, '退款中');
  page.switchLane({ currentTarget: { dataset: { l: '全部' } } });
  assert.ok(ids(page).includes('a11'), '退款中 order is missing from the 全部 lane');
});

test('search never leaks into the order model', () => {
  const { app, page } = orders();
  type(page, '0118');
  assert.equal('kw' in app.globalData, false);
  for (const o of app.globalData.aOrders) {
    assert.equal('kw' in o, false);
    assert.equal('matched' in o, false);
  }
});
