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
    storefront: {
      name: '绥安食品',
      address: '党政办公中心后院老食堂',
      pickup_point: '北门',
      announcement: '今日公告',
      business_status: 'open',
      launch_layer: null,
      flavors: [],
    },
  },
};

function identity(bound) {
  return {
    statusCode: 200,
    data: {
      identity: {
        primary_phone: { bound: false, masked_phone: '' },
        extra_phone: { set: false, masked_phone: '' },
        pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
        merchant: { bound, role: bound ? 'OWNER' : '' },
      },
    },
  };
}

function readyHarness(requests) {
  const harness = createHarness({
    logins: [{ code: 'fresh-session-code' }],
    requests: [SESSION].concat(requests || []),
  });
  const app = harness.loadApp();
  return { app, harness };
}

test('PAGE-U01 app owns only client cart/selection and never seeds business truth', async () => {
  const { app } = readyHarness();
  await app.sessionPromise;
  for (const key of ['store', 'orders', 'lastOrder', 'aOrders', 'menu', 'soldOut']) {
    assert.equal(Object.hasOwn(app.globalData, key), false, `${key} is still local business truth`);
  }
  assert.deepEqual(app.globalData.cart, {});
  assert.equal(app.globalData.pickup, null);
});

test('PAGE-U01 cold start sends an unbound user directly home and keeps a bound merchant on identity selection', async () => {
  const anonymous = readyHarness([identity(false)]);
  const anonymousLaunch = anonymous.harness.loadPage('pages/launch/launch.js');
  await anonymous.harness.invoke(anonymousLaunch, 'onShow');
  assert.equal(anonymous.harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/me/identity');
  assert.deepEqual(anonymous.harness.navigationCalls.at(-1), {
    type: 'reLaunch', url: '/pages/home/home', delta: undefined,
  });
  assert.deepEqual(anonymous.app.globalData.entryRouting, { state: 'user' });

  const merchant = readyHarness([identity(true), STOREFRONT]);
  const merchantLaunch = merchant.harness.loadPage('pages/launch/launch.js');
  await merchant.harness.invoke(merchantLaunch, 'onShow');
  assert.equal(merchant.harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/me/identity');
  assert.equal(merchant.harness.navigationCalls.length, 0);
  assert.deepEqual(merchant.app.globalData.entryRouting, { state: 'merchant', role: 'OWNER' });
  assert.equal(merchantLaunch.data.storefrontState, 'ready');
});
test('PAGE-U01 bound merchant chooses either side without a second phone authorization', async () => {
  const { harness } = readyHarness([
    identity(true),
    STOREFRONT,
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/launch/launch.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  page.go({ currentTarget: { dataset: { to: 'home' } } });
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/home/home');

  assert.equal(page.goMerchant(), true);
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/admin-orders/admin-orders');
  assert.equal(harness.requestCalls.length, 3, 'selection must not repeat merchant phone authorization');
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
