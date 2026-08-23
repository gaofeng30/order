const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: {
    access_token: 'overnight-session-token',
    token_type: 'Bearer',
    expires_at: '2999-08-25T08:00:00Z',
  },
};

const STOREFRONT = {
  statusCode: 200,
  data: {
    settings: {
      store_name: '绥安食品',
      store_address: '党政办公中心后院老食堂',
      pickup_point: '北门',
      announcement: '今日公告',
      business_status: 'open',
      launch_layer: null,
    },
  },
};

function readyHarness(requests) {
  const harness = createHarness({
    logins: [{ code: 'fresh-session-code' }],
    requests: [SESSION].concat(requests || []),
  });
  const app = harness.loadApp();
  return { app, harness };
}

test('PAGE-U01 app owns only client cart/selection and never seeds business truth', () => {
  const { app } = readyHarness();
  for (const key of ['store', 'orders', 'lastOrder', 'aOrders', 'menu', 'soldOut']) {
    assert.equal(Object.hasOwn(app.globalData, key), false, `${key} is still local business truth`);
  }
  assert.deepEqual(app.globalData.cart, {});
  assert.equal(app.globalData.pickup, null);
});
test('PAGE-U01 anonymous user entry remains available while merchant login is server-confirmed', async () => {
  const { harness } = readyHarness([
    STOREFRONT,
    { statusCode: 200, data: { user: { primary_phone_bound: true }, merchant: { role: 'OWNER', auth_version: 3 } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/launch/launch.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  page.go({ currentTarget: { dataset: { to: 'home' } } });
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/home/home');

  const denied = await page.onMerchantPhone({ detail: { errMsg: 'getPhoneNumber:fail user deny' } });
  assert.equal(denied, false);
  assert.match(page.data.hint, /用户端/);
  assert.equal(harness.requestCalls.length, 2, 'denial must not call merchant-login');

  const allowed = await page.onMerchantPhone({ detail: { code: 'merchant-phone-code' } });
  assert.equal(allowed, true);
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/me/merchant-login');
  assert.deepEqual(harness.requestCalls[2].header, {
    Authorization: 'Bearer overnight-session-token',
    'content-type': 'application/json',
  });
  assert.deepEqual(harness.requestCalls[2].data, { code: 'merchant-phone-code' });
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/admin-orders/admin-orders');
});

test('PAGE-U02 home reads storefront and active orders without local fallback', async () => {
  const { harness } = readyHarness([
    STOREFRONT,
    {
      statusCode: 200,
      data: {
        orders: [{
          id: '41', state: 'READY_FOR_PICKUP', pickup_date: '2026-08-25', pickup_time: '12:00',
          pickup_number: '0012', payable_cents: 1800,
        }],
      },
    },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/home/home.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.equal(page.data.settingsState, 'ready');
  assert.equal(page.data.storeName, '绥安食品');
  assert.equal(page.data.ongoing.orderId, '41');
  assert.match(page.data.ongoing.text, /0012/);
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/orders?active=true');
  assert.deepEqual(harness.requestCalls[2].header, { Authorization: 'Bearer overnight-session-token' });
});
