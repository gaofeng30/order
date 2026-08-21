const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// 0818 PRD §6.5 售罄按取餐日期生效，只影响当天，次日自然清零；
//           §15.6.1 status 只表达上下架，soldout 移出该字段。

function screen() {
  const harness = createHarness();
  const app = harness.loadApp();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  harness.invoke(page, 'onShow');
  return { harness, app, page, data: require('../utils/data.js') };
}
const TOMORROW = '2026-08-22';

test('product status carries only shelf state', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const m of data.MENU) {
    assert.ok(['on', 'off'].includes(m.status), `${m.id}.status = ${m.status}`);
    for (const k of ['soldout', 'soldOut', 'sold_out']) {
      assert.equal(Object.hasOwn(m, k), false, `${m.id} 仍把售罄挂在商品上`);
    }
  }
});

test("today's sell-out applies today and not tomorrow", () => {
  const { data } = screen();
  assert.equal(data.isSoldOut('p003', data.BUSINESS_DAY), true);
  assert.equal(data.isSoldOut('p003', TOMORROW), false, 'D 日售罄挡住了 D+1 的预约');
  assert.equal(data.isSellable('p003', data.BUSINESS_DAY), false);
  assert.equal(data.isSellable('p003', TOMORROW), true);
});

test("yesterday's sell-out clears itself", () => {
  const { data } = screen();
  // p006 只有 2026-08-20 的记录
  assert.equal(data.isSoldOut('p006', '2026-08-20'), true);
  assert.equal(data.isSoldOut('p006', data.BUSINESS_DAY), false, '昨日售罄活到了今天');
  assert.equal(data.isSellable('p006', data.BUSINESS_DAY), true);
});

test('a shelved-off product is never sellable regardless of sell-out', () => {
  const { data } = screen();
  assert.equal(data.itemById('p007').status, 'off');
  assert.equal(data.isSoldOut('p007', data.BUSINESS_DAY), false);
  assert.equal(data.isSellable('p007', data.BUSINESS_DAY), false, '下架商品因为没有售罄记录就被判可售');
});

test('the merchant toggle writes the day record and leaves the shelf alone', async () => {
  const { harness, app, page, data } = screen();
  const target = app.globalData.menu.find(m => m.status === 'on' && !data.isSoldOut(m.id, data.BUSINESS_DAY));
  const shelf = target.status;
  page.toggleSoldout({ currentTarget: { dataset: { id: target.id } } });
  await harness.flush(90);
  assert.equal(app.globalData.menu.find(m => m.id === target.id).status, shelf, '售罄开关改了上下架');
  assert.equal(data.isSoldOut(target.id, data.BUSINESS_DAY), true);
  assert.equal(data.isSoldOut(target.id, TOMORROW), false, '写记录时误伤了次日');
  assert.match(harness.toastCalls.at(-1).message, /仅限 2026-08-21/);
  // 切回可售应移除记录，而不是留一条 false
  page.toggleSoldout({ currentTarget: { dataset: { id: target.id } } });
  await harness.flush(90);
  assert.equal(data.isSoldOut(target.id, data.BUSINESS_DAY), false);
  assert.equal(app.globalData.soldOut.some(r => r.productId === target.id && r.serviceDate === data.BUSINESS_DAY),
    false, '恢复售卖后记录仍在');
});

test('the product row derives its label from both dimensions', () => {
  const { page } = screen();
  const soldRow = page.data.list.find(r => r.id === 'p003');
  assert.equal(soldRow.pillLabel, '售罄');
  assert.equal(soldRow.soldOut, true);
  assert.equal(soldRow.on, true, '售罄的商品被当成了下架');
  const offRow = page.data.list.find(r => r.id === 'p007');
  assert.equal(offRow.pillLabel, '已下架');
  assert.equal(offRow.soldOut, false);
  const okRow = page.data.list.find(r => r.id === 'p001');
  assert.equal(okRow.pillLabel, '可购');
});

test('the sell-out record set stores existence, not a boolean', () => {
  createHarness().loadApp();
  const data = require('../utils/data.js');
  for (const r of data.PRODUCT_SOLD_OUT_DATES) {
    assert.deepEqual(Object.keys(r).sort(), ['productId', 'serviceDate'],
      `记录多带了字段：${JSON.stringify(r)}`);
  }
});
