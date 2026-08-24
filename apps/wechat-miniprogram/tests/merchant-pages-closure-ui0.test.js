const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

function readyHarness(requests) {
  const harness = createHarness({
    logins: [{ code: 'merchant-closure-login' }],
    requests: [{ statusCode: 201, data: {
      access_token: 'merchant-closure-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z',
    } }].concat(requests),
  });
  harness.loadApp();
  return harness;
}

function orderSummary(state = 'PREPARING') {
  return {
    id: '41', order_no: 'ORD-MERCHANT-41', state,
    pickup_date: '2026-08-24', pickup_time: '12:00', pickup_point: '北门', pickup_number: '0041',
    payable_cents: 1880, available_actions: state === 'PREPARING' ? ['READY'] : [], materialized_at: '2026-08-24T03:20:00Z',
  };
}

function menu(date, time, mealPeriod, products) {
  return { statusCode: 200, data: {
    selection: { date, time, meal_period: mealPeriod },
    store_status: { business_status: 'closed', service_date_available: true, meal_available: false, cutoff_passed: true },
    categories: [{ id: '7', name: '全部菜品', products }],
  } };
}

function product(id, name, mealPeriod) {
  return {
    id, category_id: '7', name, description: '', specification: '份', meal_period: mealPeriod,
    images: [], listed: true, sold_out: false, original_unit_price_cents: 1800,
  };
}

test('PAGE-M02 reads the current store status together with the selected merchant lane', async () => {
  const harness = readyHarness([
    { statusCode: 200, data: { orders: [orderSummary()] } },
    { statusCode: 200, data: { storefront: {
      name: '绥安食品', address: '党政办公中心后院老食堂', pickup_point: '北门', announcement: '',
      business_status: 'open', flavors: [],
    } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  page.onLoad({ lane: '制作中' });
  assert.equal(await page.load(), true);
  assert.equal(page.data.storeStatus, 'open');
  assert.equal(page.data.listState, 'ready');
  assert.equal(page.data.list[0].state, 'PREPARING');
  assert.equal(harness.requestCalls[1].url.includes('state=PREPARING'), true);
  assert.equal(harness.requestCalls[2].url.endsWith('/api/v1/storefront/settings'), true);
});

test('PAGE-M02 failed store-status write stays visibly failed without changing the selected fact', async () => {
  const harness = readyHarness([{ statusCode: 503, data: { error: { code: 'STORE_STATUS_UNAVAILABLE' } } }]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  page.setData({ storeStatus: 'open' });
  assert.equal(await page.setBiz({ currentTarget: { dataset: { b: 'closed' } } }), false);
  assert.equal(page.data.storeStatus, 'open');
  assert.equal(page.data.statusWriteState, 'error');
});

test('PAGE-M03 ready failure is visible and never projects a local transition', async () => {
  const detail = Object.assign(orderSummary(), {
    items: [{ product_id: '7', name: '午餐', quantity: 1, unit_price_cents: 1880, line_total_cents: 1880, flavors: [], note: '' }],
    contact: { name: '顾客', masked_phone: '138****0000' }, order_note: '', notification_options: [],
  });
  const harness = readyHarness([
    { statusCode: 200, data: { order: detail } },
    { statusCode: 503, data: { error: { code: 'FULFILLMENT_UNAVAILABLE' } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-order-detail/admin-order-detail.js');
  assert.equal(await page.onLoad({ id: '41' }), true);
  assert.equal(await page.markReady(), false);
  assert.equal(page.data.o.state, 'PREPARING');
  assert.equal(page.data.actionState, 'error');
});

test('PAGE-M05 remains usable while closed and merges lunch/dinner products without inventing availability', async () => {
  const date = '2026-08-24';
  const options = { statusCode: 200, data: { dates: [{
    date, available: false, meal_periods: [
      { meal_period: 'lunch', cutoff_time: '11:30', available: false, pickup_times: ['11:30'] },
      { meal_period: 'dinner', cutoff_time: '17:00', available: false, pickup_times: ['17:00'] },
    ],
  }] } };
  const harness = readyHarness([
    options,
    menu(date, '11:30', 'lunch', [product('70', '全天菜', 'all'), product('71', '午餐菜', 'lunch')]),
    menu(date, '17:00', 'dinner', [product('70', '全天菜', 'all'), product('72', '晚餐菜', 'dinner')]),
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  assert.equal(await page.onShow(), true);
  assert.equal(page.data.listState, 'ready');
  assert.deepEqual(page.data.list.map(item => item.name).sort(), ['全天菜', '午餐菜', '晚餐菜']);
  assert.equal(page.data.list.every(item => item.orderable === false), true);
  assert.equal(harness.requestCalls.length, 4);
});

test('PAGE-M05 contains sale-status controls only and no shelf control', () => {
  const wxml = fs.readFileSync(path.join(__dirname, '../pages/admin-products/admin-products.wxml'), 'utf8');
  assert.doesNotMatch(wxml, /上架|下架|pa-shelf/u);
  assert.match(wxml, /标记售罄|soldoutLabel/u);
});
