const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'menu-public-read', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

const OPTIONS = {
  statusCode: 200,
  data: {
    dates: [{
      date: '2026-08-25', available: true,
      meal_periods: [{
        meal_period: 'dinner', available: true, cutoff_time: '17:00', pickup_times: ['17:30', '18:00'],
      }],
    }],
  },
};

const MENU = {
  statusCode: 200,
  data: {
    selection: { date: '2026-08-25', time: '17:30', meal_period: 'dinner' },
    store_status: {
      business_status: 'open', service_date_available: true, meal_available: true, cutoff_passed: false,
    },
    categories: [{
      id: '7', name: '晚餐', products: [{
        id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份',
        meal_period: 'all', images: [{ object_key: 'images/70.png', url: '/api/v1/objects/images/70.png' }],
        listed: true, sold_out: false, original_unit_price_cents: 1800, staff_unit_price_cents: 1530,
      }],
    }],
  },
};

test('frozen menu uses staff unit price in display and the cart while retaining original price', async () => {
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION, OPTIONS, MENU] });
  const app = harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.equal(page.data.listState, 'ready');
  const product = page.data.groups[0].products[0];
  assert.equal(product.price_cents, 1530);
  assert.equal(product.original_unit_price_cents, 1800);
  assert.equal(product.isStaffPrice, true);
  assert.equal(product.cover.url, 'http://127.0.0.1:8080/api/v1/objects/images/70.png');

  page.add({ currentTarget: { dataset: { id: '70' } } });
  assert.equal(app.globalData.cart['70'].product.price_cents, 1530);
  assert.equal(app.globalData.cart['70'].product.original_unit_price_cents, 1800);
  assert.equal(page.data.total, 15.3);

  assert.equal(page.goDetail({ currentTarget: { dataset: { id: 'missing' } } }), false);
  assert.equal(harness.navigationCalls.length, 0);
  assert.equal(page.goDetail({ currentTarget: { dataset: { id: '70' } } }), true);
  assert.match(harness.navigationCalls.at(-1).url, /\/pages\/detail\/detail\?id=70$/);

  page.openPicker();
  assert.equal(page.pickPickerDate({ currentTarget: { dataset: { date: '2026-08-25' } } }), true);
});

test('legacy pickup facts fail closed before menu request and never write cart or selection', async () => {
  const legacy = {
    statusCode: 200,
    data: {
      timezone: 'Asia/Shanghai',
      dates: [{
        date: '2026-08-25', orderable: true,
        meals: [{ code: 'dinner', orderable: true, cutoff_at: '2026-08-25T17:00:00+08:00', pickup_times: ['17:30'] }],
      }],
    },
  };
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION, legacy] });
  const app = harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.equal(page.data.listState, 'error');
  assert.equal(app.globalData.pickup, null);
  assert.deepEqual(app.globalData.cart, {});
  assert.equal(harness.requestCalls.length, 2);
});

test('closed store remains browsable but every product is non-orderable', async () => {
  const closed = JSON.parse(JSON.stringify(MENU));
  closed.data.store_status.business_status = 'closed';
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION, OPTIONS, closed] });
  const app = harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.equal(page.data.groups[0].products[0].orderable, false);
  page.add({ currentTarget: { dataset: { id: '70' } } });
  assert.deepEqual(app.globalData.cart, {});
});
