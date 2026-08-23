const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: {
    access_token: 'boundaries-session-token',
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
      announcement: '',
      business_status: 'open',
      launch_layer: null,
      flavors: [],
    },
  },
};

function homeHarness(orders) {
  const harness = createHarness({
    logins: [{ code: 'boundaries-session-code' }],
    requests: [SESSION, STOREFRONT, { statusCode: 200, data: { orders } }],
  });
  harness.loadApp();
  return harness;
}

test('BE-26 home take-code entry navigates only to a READY_FOR_PICKUP order', async () => {
  const harness = homeHarness([{
    id: '51', state: 'RESERVED', pickup_date: '2026-08-25', pickup_time: '18:00',
    pickup_number: '0042', payable_cents: 1800,
  }]);
  await harness.flush();
  const page = harness.loadPage('pages/home/home.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  const before = harness.navigationCalls.length;
  const handled = page.gridTap({ currentTarget: { dataset: { k: 'pickup' } } });

  assert.equal(handled, true);
  assert.equal(harness.navigationCalls.length, before, 'non-READY order must not open a stale take-code page');
  assert.equal(harness.toastCalls.at(-1).message, '暂无可取餐订单');
});

test('BE-01 closed store still renders the server menu but exposes no checkout path', async () => {
  const options = {
    statusCode: 200,
    data: {
      dates: [{
        date: '2026-08-25', available: false,
        meal_periods: [
          { meal_period: 'lunch', cutoff_time: '11:30', available: false, pickup_times: ['11:30'] },
          { meal_period: 'dinner', cutoff_time: '17:00', available: false, pickup_times: ['17:00'] },
        ],
      }, {
        date: '2026-08-26', available: false,
        meal_periods: [
          { meal_period: 'lunch', cutoff_time: '11:30', available: false, pickup_times: ['11:30'] },
          { meal_period: 'dinner', cutoff_time: '17:00', available: false, pickup_times: ['17:00'] },
        ],
      }],
    },
  };
  const menu = {
    statusCode: 200,
    data: {
      selection: { date: '2026-08-25', time: '11:30', meal_period: 'lunch' },
      store_status: { business_status: 'closed', service_date_available: true, meal_available: false, cutoff_passed: false },
      categories: [{
        id: '7', name: '午餐', products: [{
          id: '70', category_id: '7', name: '红烧肉', description: '', specification: '',
          meal_period: 'lunch', images: [], listed: true, sold_out: false, original_unit_price_cents: 1800,
        }],
      }],
    },
  };
  const harness = createHarness({
    logins: [{ code: 'boundaries-session-code' }],
    requests: [SESSION, options, menu],
  });
  const app = harness.loadApp();
  app.globalData.cart = {
    70: {
      product: {
        id: '70', category_id: '7', name: '红烧肉', description: '', specification: '', meal_period: 'lunch',
        images: [], listed: true, sold_out: false, original_unit_price_cents: 1800, price_cents: 1800, isStaffPrice: false,
      },
      qty: 1, flavors: [], note: '',
    },
  };
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.equal(page.data.listState, 'ready');
  assert.equal(page.data.groups[0].products[0].orderable, false);
  assert.equal(page.data.groups[0].products[0].availabilityLabel, '当前不可下单');
  assert.equal(page.data.count, 1, 'closed store must preserve an already selected item');
  assert.equal(page.data.canCheckout, false);
  assert.equal(page.add({ currentTarget: { dataset: { id: '70' } } }), false);
  assert.equal(page.data.count, 1, 'browse-only facts must also block incrementing a stale cart line');
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/menu?date=2026-08-25&time=11%3A30');
  assert.equal(page.goConfirm(), undefined);
  assert.equal(Object.keys(app.globalData.cart).length, 1);
  assert.equal(harness.navigationCalls.length, 0);

  page._menuOrderable = true;
  page.setData({ canCheckout: true });
  const refresh = page.loadOptionsAndMenu();
  assert.equal(page.data.canCheckout, false, 'uncertain refreshed facts must synchronously close checkout');
  await refresh;
});

test('BE-23 extra phone is visibly STAFF only after the server confirms both phone and name', async () => {
  const visitor = {
    statusCode: 200,
    data: {
      extra_phone: { set: true, masked_phone: '188****0001', name: '错名' },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
    },
  };
  const staff = {
    statusCode: 200,
    data: {
      extra_phone: { set: true, masked_phone: '188****0001', name: '正确姓名' },
      pricing_identity: { kind: 'STAFF', rate_percent: 80 },
    },
  };
  const harness = createHarness({
    logins: [{ code: 'boundaries-session-code' }],
    requests: [SESSION, visitor, staff],
  });
  harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');

  page.setData({ extraForm: { phone: '18800000001', name: '错名' } });
  assert.equal(await page.saveExtraPhone(), true);
  await harness.flush();
  assert.equal(page.data.pricingKind, 'VISITOR');
  assert.equal(page.data.extraState, 'unmatched');

  page.setData({ extraForm: { phone: '18800000001', name: '正确姓名' } });
  assert.equal(await page.saveExtraPhone(), true);
  await harness.flush();
  assert.equal(page.data.pricingKind, 'STAFF');
  assert.equal(page.data.extraState, 'matched');
});
