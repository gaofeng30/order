const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// 0818 PRD §5.1 首页内容；§6.9 营业状态可人工覆盖；§5.10 订阅被拒时的兜底。
const ONGOING = ['已预约', '制作中', '待取餐'];

function home(mutate) {
  const harness = createHarness();
  const app = harness.loadApp();
  if (mutate) mutate(app.globalData);
  const page = harness.loadPage('pages/home/home.js');
  harness.invoke(page, 'onLoad', {});
  harness.invoke(page, 'onShow');
  return { harness, app, page };
}
const DAY = '2026-08-21';
const order = (id, status, time) => ({
  id, code: '00' + id.slice(-2), status, pickupDate: DAY, pickupTime: time,
  paidAt: `${DAY} 09:00:00`, pickupPoint: '县前直营店', items: [], total: 1000,
});

test('the notice comes from configuration, not from the page', () => {
  const { page } = home(g => { g.store.notice = '今日提前打烊，18:00 停止取餐'; });
  assert.equal(page.data.notice, '今日提前打烊，18:00 停止取餐');
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  for (const rel of ['pages/home/home.js', 'pages/home/home.wxml']) {
    assert.ok(!fs.readFileSync(path.join(root, rel), 'utf8').includes('今日卤味'),
      `${rel} 把公告写死了`);
  }
});

test('the business status follows the merchant switch', () => {
  for (const st of ['营业中', '休息中', '已截单']) {
    const { page } = home(g => { g.store.status = st; });
    assert.equal(page.data.bizStatus, st);
  }
});

test('no in-flight strip when nothing is in flight', () => {
  const { page } = home(g => { g.orders = g.orders.filter(o => !ONGOING.includes(o.status)); });
  assert.equal(page.data.ongoing, null);
});

test('the strip counts the three in-flight states only', () => {
  const { page } = home(g => {
    g.orders = [order('m1', '已预约', '18:30'), order('m2', '制作中', '17:30'),
                order('m3', '已完成', '12:00'), order('m4', '已退款', '12:00')];
  });
  assert.equal(page.data.ongoing.count, 2);
  assert.equal(page.data.ongoing.ready, false);
});

test('the strip names the earliest pickup, not the earliest order', () => {
  const { page } = home(g => {
    const late = order('m1', '已预约', '19:00');
    late.paidAt = `${DAY} 08:00:00`;            // 更早下单
    const soon = order('m2', '已预约', '17:30');
    soon.paidAt = `${DAY} 10:00:00`;            // 更晚下单，但更早取餐
    g.orders = [late, soon];
  });
  assert.equal(page.data.ongoing.orderId, 'm2');
  assert.match(page.data.ongoing.text, /17:30/);
  assert.match(page.data.ongoing.text, /你有 2 单进行中/);
});

test('a ready order flips the wording and drops the count', () => {
  const { page } = home(g => {
    g.orders = [order('m1', '已预约', '17:30'), order('m2', '待取餐', '19:00')];
  });
  assert.equal(page.data.ongoing.ready, true);
  assert.match(page.data.ongoing.text, /已备好，可取餐/);
  assert.equal(page.data.ongoing.orderId, 'm2', '提示没有指向已备好的那一单');
  assert.doesNotMatch(page.data.ongoing.text, /\d+\s*单/, '已备好的文案仍在强调单数');
});

test('tapping the strip opens that order', () => {
  const { harness, page } = home();
  const id = page.data.ongoing.orderId;
  page.tapOngoing();
  const url = harness.navigationCalls.at(-1).url;
  assert.match(url, /order-detail/);
  assert.ok(url.includes(id), `跳转没带上该单：${url}`);
});

test('the home screen keeps three entries and no placeholder', () => {
  const { page } = home();
  assert.deepEqual(page.data.grid.map(g => g.k), ['reserve', 'orders', 'pickup']);
  for (const g of page.data.grid) assert.equal(Object.hasOwn(g, 'off'), false);
  const fs = require('node:fs');
  const path = require('node:path');
  const wxml = fs.readFileSync(path.join(path.resolve(__dirname, '..'), 'pages/home/home.wxml'), 'utf8');
  assert.doesNotMatch(wxml, /未开放|即将上线|item\.off/);
});
