const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = { statusCode: 201, data: { access_token: 'merchant-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' } };
const ORDER = {
  id: '401', order_no: 'SA202608250002', state: 'PREPARING', pickup_date: '2026-08-25', pickup_time: '17:30',
  pickup_point: '北门', pickup_number: '0013', payable_cents: 2100,
  materialized_at: '2026-08-25T08:00:00Z', available_actions: ['READY'],
};
const DETAIL = Object.assign({}, ORDER, {
  contact: { name: '王女士', masked_phone: '+*********9988' }, identity: { kind: 'STAFF' }, discount: { rate_percent: 85 },
  items: [{ product_id: '70', name: '红烧肉', quantity: 1, unit_price_cents: 2100, line_total_cents: 2100, flavors: [], note: '' }],
  transaction_id: 'tx', paid_at: '2026-08-25T08:00:00Z', redemption_token: null,
  transition_times: {}, notification_options: [], order_note: '',
});
function readyHarness(requests, native) {
  const harness = createHarness(Object.assign({ logins: [{ code: 'code' }], requests: [SESSION].concat(requests || []) }, native || {}));
  harness.loadApp();
  return harness;
}

test('PAGE-M02 has exactly five lanes, server search and server store-status write', async () => {
  const harness = readyHarness([
    { statusCode: 200, data: { orders: [ORDER] } },
    { statusCode: 200, data: { orders: [ORDER] } },
    { statusCode: 200, data: { store_status: 'closed' } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-orders/admin-orders.js');
  await harness.invoke(page, 'onShow');
  assert.deepEqual(page.data.lanes, ['已预约', '制作中', '待取餐', '已完成', '已退款']);
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/merchant/orders?state=RESERVED&limit=20');
  await page.onKw({ detail: { value: '0013' } });
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/merchant/orders?q=0013&limit=20');
  assert.equal(await page.setBiz({ currentTarget: { dataset: { b: 'closed' } } }), true);
  assert.deepEqual(harness.requestCalls[3].data, { status: 'closed' });
  assert.equal(page.data.storeStatus, 'closed');
});

test('PAGE-M03 marks only PREPARING order ready from server response', async () => {
  const ready = Object.assign({}, DETAIL, { state: 'READY_FOR_PICKUP', redemption_token: 'token', available_actions: ['REDEEM'] });
  const completed = Object.assign({}, ready, { state: 'COMPLETED', redemption_token: null, available_actions: [] });
  const harness = readyHarness([
    { statusCode: 200, data: { order: DETAIL } },
    { statusCode: 200, data: { order: ready } },
    { statusCode: 200, data: { order: completed } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-order-detail/admin-order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '401' });
  assert.equal(page.data.o.state, 'PREPARING');
  assert.equal(page.data.meta.label, '备好');
  assert.equal(await page.advance(), true);
  assert.equal(page.data.o.state, 'READY_FOR_PICKUP');
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^ready-/);
  assert.equal(page.data.meta.label, '核销');
  assert.equal(page.data.meta.isView, false);
  assert.equal(await page.advance(), true);
  assert.equal(page.data.o.state, 'COMPLETED');
  assert.match(harness.requestCalls[3].header['Idempotency-Key'], /^redeem-/);
});

test('PAGE-M04 scan/manual atomically redeem with keys; invalid code makes zero request', async () => {
  const completed = Object.assign({}, DETAIL, { state: 'COMPLETED', redemption_token: null, available_actions: [] });
  const harness = readyHarness([
    { statusCode: 200, data: { order: completed } },
    { statusCode: 200, data: { order: completed } },
  ], { scans: [{ result: 'opaque-token' }] });
  await harness.flush();
  const page = harness.loadPage('pages/admin-verify/admin-verify.js');
  assert.equal(await page.scan(), true);
  assert.deepEqual(harness.requestCalls[1].data, { token: 'opaque-token' });
  assert.match(harness.requestCalls[1].header['Idempotency-Key'], /^redeem-scan-/);
  assert.equal(page.data.lookupState, 'completed');
  assert.equal(page.data.lastResult.state, 'COMPLETED');
  const count = harness.requestCalls.length;
  page.setData({ code: '13' });
  assert.equal(await page.manual(), false);
  assert.equal(harness.requestCalls.length, count);
  page.setData({ code: '0013', lastResult: null, lookupState: 'idle' });
  assert.equal(await page.manual(), true);
  assert.deepEqual(harness.requestCalls.at(-1).data, { pickup_number: '0013' });
  assert.match(harness.requestCalls.at(-1).header['Idempotency-Key'], /^redeem-code-/);
  assert.equal(page.data.lastResult.state, 'COMPLETED');
});

test('PAGE-M05 reads server menu and today sold-out toggle updates only from response', async () => {
  const options = { statusCode: 200, data: { dates: [{
    date: '2026-08-25', available: true, meal_periods: [{ meal_period: 'dinner', cutoff_time: '17:00', available: true, pickup_times: ['17:30'] }],
  }] } };
  const menu = { statusCode: 200, data: {
    selection: { date: '2026-08-25', time: '17:30', meal_period: 'dinner' },
    store_status: { business_status: 'open', service_date_available: true, meal_available: true, cutoff_passed: false },
    categories: [{ id: '7', name: '晚餐', products: [{ id: '70', category_id: '7', name: '红烧肉', description: '', specification: '份', meal_period: 'all', images: [], listed: true, sold_out: false, original_unit_price_cents: 1800 }] }],
  } };
  const harness = readyHarness([options, menu, {
    statusCode: 200, data: { product_id: '70', service_date: '2026-08-25', sold_out: true },
  }]);
  await harness.flush();
  const page = harness.loadPage('pages/admin-products/admin-products.js');
  await harness.invoke(page, 'onShow');
  assert.equal(page.data.list[0].soldOut, false);
  assert.equal(await page.toggleSoldout({ currentTarget: { dataset: { id: '70' } } }), true);
  assert.equal(page.data.list[0].soldOut, true);
  assert.deepEqual(harness.requestCalls[3].data, { service_date: '2026-08-25', sold_out: true });
});
