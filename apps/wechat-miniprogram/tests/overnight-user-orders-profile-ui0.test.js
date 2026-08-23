const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = { statusCode: 201, data: { access_token: 'user-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' } };

const SUMMARY = {
  id: '301', order_no: 'SA202608250001', state: 'READY_FOR_PICKUP', pickup_date: '2026-08-25',
  pickup_time: '17:30', pickup_point: '北门', pickup_number: '0012', payable_cents: 1800,
  materialized_at: '2026-08-25T08:01:00Z', available_actions: ['SUBSCRIBE_READY'],
};

const DETAIL = Object.assign({}, SUMMARY, {
  contact: { name: '林先生', masked_phone: '+*********5678' }, identity: { kind: 'VISITOR' },
  discount: { rate_percent: 100 },
  items: [{ product_id: '70', name: '红烧肉', quantity: 1, unit_price_cents: 1800, line_total_cents: 1800, flavors: ['少盐'], note: '' }],
  transaction_id: 'wx-tx', paid_at: '2026-08-25T08:01:00Z', redemption_token: 'opaque-redemption-token',
  transition_times: {}, notification_options: ['READY'], order_note: '',
});

function readyHarness(requests, native) {
  const harness = createHarness(Object.assign({
    logins: [{ code: 'code' }], requests: [SESSION].concat(requests || []),
  }, native || {}));
  const app = harness.loadApp();
  return { app, harness };
}

test('PAGE-U08 exposes fixed six filters, pages server results, and has no refunding filter', async () => {
  const { harness } = readyHarness([
    { statusCode: 200, data: { orders: [SUMMARY], next_after_id: '301' } },
    { statusCode: 200, data: { orders: [], next_after_id: null } },
    { statusCode: 200, data: { orders: [Object.assign({}, SUMMARY, { id: '302', state: 'REFUNDED' })] } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/orders/orders.js');
  await harness.invoke(page, 'onShow');
  assert.deepEqual(page.data.tabs, ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款']);
  assert.equal(page.data.list[0].displayState, '待取餐');
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20');
  await page.loadMore();
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20&after_id=301');
  await page.switchTab({ currentTarget: { dataset: { t: '已退款' } } });
  assert.equal(harness.requestCalls[3].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20&state=REFUNDED');
});

test('PAGE-U07 READY detail shows only server token and records native subscription decision', async () => {
  const { app, harness } = readyHarness([
    { statusCode: 200, data: { order: DETAIL } },
    { statusCode: 200, data: { subscription: { kind: 'READY', decision: 'ACCEPTED', available: true } } },
  ], { subscriptions: [{ 'ready-template': 'accept' }] });
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { READY: 'ready-template' };
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '301' });

  assert.equal(page.data.showQr, true);
  assert.equal(page.data.qrToken, 'opaque-redemption-token');
  assert.equal(page.data.canCancel, false);
  const subscribed = await page.subscribeReady();
  assert.equal(subscribed, true);
  assert.deepEqual(harness.subscribeCalls[0].tmplIds, ['ready-template']);
  assert.deepEqual(harness.requestCalls[2].data, { kind: 'READY', decision: 'ACCEPTED' });
});

test('PAGE-U07 cancellation is only a server available_action and becomes REFUNDING from response', async () => {
  const reserved = Object.assign({}, DETAIL, {
    state: 'RESERVED', redemption_token: null, available_actions: ['CANCEL'], notification_options: [],
  });
  const refunding = Object.assign({}, reserved, { state: 'REFUNDING', available_actions: [] });
  const { harness } = readyHarness([
    { statusCode: 200, data: { order: reserved } },
    { statusCode: 200, data: { order: refunding, refund: { id: '501', state: 'REQUESTED', amount_cents: 1800 } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  assert.equal(page.data.showQr, false);
  assert.equal(page.data.canCancel, true);
  const cancelled = await page.doCancel();
  assert.equal(cancelled, true);
  assert.equal(page.data.o.state, 'REFUNDING');
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^cancel-/);
});

test('PAGE-U06 result reloads the created order instead of global lastOrder', async () => {
  const { harness } = readyHarness([{ statusCode: 200, data: { order: DETAIL } }]);
  await harness.flush();
  const page = harness.loadPage('pages/result/result.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  assert.equal(page.data.state, 'ready');
  assert.equal(page.data.o.id, '301');
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/orders/301');
});

test('PAGE-U09 profile uses server identity, optional cosmetic profile and native customer-service surface', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' },
      extra_phone: { set: false }, pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
      merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [SUMMARY] } },
  ], { profiles: [{ userInfo: { nickName: '用户昵称', avatarUrl: 'https://avatar.example.com/a.png' } }] });
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');
  assert.equal(page.data.identityState, 'ready');
  assert.equal(page.data.phoneMask, '+*********5678');
  assert.equal(page.data.pend, 1);
  assert.equal(await page.chooseProfile(), true);
  assert.equal(page.data.nick, '用户昵称');
});
