const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'checkout-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

const PHONE = { statusCode: 200, data: { primary_phone_bound: true, masked_phone: '+*********5678' } };
const QUOTE = {
  statusCode: 201,
  data: {
    quote: {
      id: '91', contact: { name: '林先生', masked_phone: '+*********5678' }, identity: { kind: 'VISITOR' },
      discount: { rate_percent: 100 }, store: { name: '绥安食品', address: '老食堂' },
      pickup: { date: '2026-08-25', time: '17:30', meal_period: 'dinner', point: '北门' },
      order_note: '', items: [{ product_id: '70', name: '红烧肉', quantity: 1, unit_price_cents: 1800 }],
      original_subtotal_cents: 1800, discount_cents: 0, payable_cents: 1800,
      created_at: '2026-08-25T08:00:00Z', expires_at: '2026-08-25T08:10:00Z',
    },
  },
};
const PREPAY = {
  statusCode: 201,
  data: {
    prepayment: {
      id: '101', state: 'CREATED', expires_at: '2026-08-25T08:10:00Z',
      wx_request_payment: {
        timeStamp: '1787644800', nonceStr: 'nonce', package: 'prepay_id=wx123', signType: 'RSA', paySign: 'signature',
      },
    },
  },
};

function readyCheckout(requests, payments) {
  const harness = createHarness({
    logins: [{ code: 'code' }], requests: [SESSION].concat(requests), payments: payments || [],
  });
  const app = harness.loadApp();
  return { app, harness };
}

function seedCart(app) {
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
  app.globalData.cart = {
    70: {
      product: {
        id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份',
        meal_period: 'all', images: [], listed: true, sold_out: false,
        original_unit_price_cents: 1800, price_cents: 1800, isStaffPrice: false,
      },
      qty: 1, flavors: ['少盐'], note: '分装',
    },
  };
}

test('PAGE-U05/U06 checkout uses Quote -> durable wx_request_payment -> server confirm only', async () => {
  const confirm = { statusCode: 200, data: { state: 'ORDER_CREATED', order_id: '301' } };
  const { app, harness } = readyCheckout([PHONE, QUOTE, PREPAY, confirm], [{ ok: true }]);
  await harness.flush();
  seedCart(app);
  const page = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(page, 'onLoad');
  await harness.invoke(page, 'onShow');
  page.onInput({ currentTarget: { dataset: { k: 'contact' } }, detail: { value: '林先生' } });

  const paid = await page.pay();
  await harness.flush();

  assert.equal(paid, true);
  assert.deepEqual(harness.requestCalls[2].data, {
    contact_name: '林先生', pickup_date: '2026-08-25', pickup_time: '17:30', order_note: '',
    items: [{ product_id: '70', quantity: 1, flavors: ['少盐'], note: '分装' }],
  });
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^quote-/);
  assert.deepEqual(harness.requestCalls[3].data, { quote_id: '91' });
  assert.deepEqual(harness.paymentCalls[0], PREPAY.data.prepayment.wx_request_payment);
  assert.deepEqual(harness.requestCalls[4].data, { prepayment_id: '101' });
  assert.deepEqual(app.globalData.cart, {});
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/result/result?id=301');
});

test('BE-07/BE-08 native cancel retries the same payment before confirm-only pending checks', async () => {
  const pending = { statusCode: 202, data: { state: 'PENDING' } };
  const { app, harness } = readyCheckout(
    [PHONE, QUOTE, PREPAY, pending, pending],
    [{ errMsg: 'requestPayment:fail cancel' }, { ok: true }],
  );
  await harness.flush();
  seedCart(app);
  const page = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(page, 'onLoad');
  await harness.invoke(page, 'onShow');
  page.onInput({ currentTarget: { dataset: { k: 'contact' } }, detail: { value: '林先生' } });

  const paid = await page.pay();
  await harness.flush();

  assert.equal(paid, false);
  assert.equal(page.data.paymentState, 'error');
  assert.equal(Object.keys(app.globalData.cart).length, 1);
  assert.equal(harness.navigationCalls.length, 0);
  assert.equal(page._prepayment.id, '101');
  assert.equal(harness.requestCalls.filter(call => call.url.endsWith('/api/v1/orders/confirm')).length, 0);
  assert.equal(harness.paymentCalls.length, 1);

  assert.equal(await page.pay(), false);
  assert.equal(page.data.paymentState, 'pending');
  assert.equal(Object.keys(app.globalData.cart).length, 1);
  assert.equal(harness.navigationCalls.length, 0);
  const firstConfirmKey = harness.requestCalls.at(-1).header['Idempotency-Key'];
  assert.equal(harness.paymentCalls.length, 2);
  assert.deepEqual(harness.paymentCalls[0], harness.paymentCalls[1]);

  assert.equal(await page.pay(), false);
  const secondConfirmKey = harness.requestCalls.at(-1).header['Idempotency-Key'];
  assert.notEqual(secondConfirmKey, firstConfirmKey, 'a durable PENDING receipt requires a new key for the next observation query');
  assert.equal(harness.paymentCalls.length, 2, 'durable PENDING retries confirm only');
  assert.equal(harness.requestCalls.filter(call => call.url.endsWith('/api/v1/orders/prepay')).length, 1);
});

test('BE-08 confirm retry preserves the key for transport ambiguity and rotates it only after durable PENDING', async () => {
  const pending = { statusCode: 202, data: { state: 'PENDING' } };
  const created = { statusCode: 200, data: { state: 'ORDER_CREATED', order_id: '302' } };
  const { app, harness } = readyCheckout(
    [PHONE, QUOTE, PREPAY, { networkError: true }, pending, created],
    [{ ok: true }],
  );
  await harness.flush();
  seedCart(app);
  const page = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(page, 'onLoad');
  await harness.invoke(page, 'onShow');
  page.onInput({ currentTarget: { dataset: { k: 'contact' } }, detail: { value: '林先生' } });

  assert.equal(await page.pay(), false);
  const ambiguousKey = harness.requestCalls.at(-1).header['Idempotency-Key'];
  assert.equal(page.data.paymentState, 'error');
  assert.equal(Object.keys(app.globalData.cart).length, 1);
  assert.equal(harness.navigationCalls.length, 0);

  assert.equal(await page.pay(), false);
  const pendingKey = harness.requestCalls.at(-1).header['Idempotency-Key'];
  assert.equal(pendingKey, ambiguousKey, 'transport ambiguity must retry the same logical attempt');
  assert.equal(page.data.paymentState, 'pending');
  assert.equal(Object.keys(app.globalData.cart).length, 1);
  assert.equal(harness.navigationCalls.length, 0);

  assert.equal(await page.pay(), true);
  const postPendingKey = harness.requestCalls.at(-1).header['Idempotency-Key'];
  assert.notEqual(postPendingKey, pendingKey, 'durable PENDING starts a new observation attempt');
  assert.deepEqual(app.globalData.cart, {});
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/result/result?id=302');
});

test('BE-25 empty cart makes zero checkout request', async () => {
  const { app, harness } = readyCheckout([], []);
  await harness.flush();
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
  const page = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(page, 'onLoad');
  const paid = await page.pay();
  assert.equal(paid, false);
  assert.equal(harness.requestCalls.length, 1);
  assert.equal(harness.paymentCalls.length, 0);
});
