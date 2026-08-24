const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'menu-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

const VISITOR_IDENTITY = {
  statusCode: 200,
  data: {
    identity: {
      primary_phone: { bound: true, masked_phone: '+******0000' },
      extra_phone: { set: false, masked_phone: '' },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
      merchant: { bound: false },
    },
  },
};

const OPTIONS = {
  statusCode: 200,
  data: {
    dates: [
      {
        date: '2026-08-25', available: true,
        meal_periods: [
          { meal_period: 'lunch', cutoff_time: '10:45', available: false, pickup_times: ['11:40'] },
          { meal_period: 'dinner', cutoff_time: '17:00', available: true, pickup_times: ['17:30', '18:00'] },
        ],
      },
      {
        date: '2026-08-26', available: true,
        meal_periods: [{ meal_period: 'lunch', cutoff_time: '10:45', available: true, pickup_times: ['11:40'] }],
      },
    ],
  },
};

const MENU = {
  statusCode: 200,
  data: {
    selection: { date: '2026-08-25', time: '17:30', meal_period: 'dinner' },
    store_status: { business_status: 'open', service_date_available: true, meal_available: true, cutoff_passed: false },
    categories: [{
      id: '7', name: '晚餐', products: [
        { id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份', meal_period: 'all', images: [], listed: true, sold_out: false, original_unit_price_cents: 1800 },
        { id: '71', category_id: '7', name: '米饭', description: '', specification: '碗', meal_period: 'dinner', images: [], listed: true, sold_out: true, original_unit_price_cents: 200 },
      ],
    }],
  },
};

function readyHarness(requests) {
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION].concat(requests) });
  const app = harness.loadApp();
  return { app, harness };
}

test('PAGE-U03 pickup options skip cut-off meals and menu sold-out stays non-addable', async () => {
  const { app, harness } = readyHarness([OPTIONS, VISITOR_IDENTITY, MENU]);
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.deepEqual(app.globalData.pickup, { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' });
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/menu/pickup-options');
  assert.equal(harness.requestCalls[3].url, 'http://127.0.0.1:8080/api/v1/menu?date=2026-08-25&time=17%3A30');
  assert.equal(page.data.listState, 'ready');
  assert.equal(page.data.groups[0].products[1].availabilityLabel, '已售罄');

  page.add({ currentTarget: { dataset: { id: '71' } } });
  assert.equal(page.data.count, 0);
  page.add({ currentTarget: { dataset: { id: '70' } } });
  assert.equal(page.data.count, 1);

  page.onSearch({ detail: { value: '红烧' } });
  assert.deepEqual(page.data.groups[0].products.map(product => product.id), ['70']);
  page.onSearch({ detail: { value: '不存在' } });
  assert.equal(page.data.listState, 'empty');
});
test('BE-02 selecting a different available point reloads /menu and never synthesizes a time', async () => {
  const secondMenu = JSON.parse(JSON.stringify(MENU));
  secondMenu.data.selection = { date: '2026-08-26', time: '11:40', meal_period: 'lunch' };
  secondMenu.data.categories[0].products = secondMenu.data.categories[0].products.filter(product => product.meal_period === 'all' || product.meal_period === 'lunch');
  const { app, harness } = readyHarness([OPTIONS, VISITOR_IDENTITY, MENU, VISITOR_IDENTITY, secondMenu]);
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  const changed = await page.pickPickerTime({
    currentTarget: { dataset: { date: '2026-08-26', period: 'lunch', t: '11:40' } },
  });
  await harness.flush();

  assert.equal(changed, true);
  assert.deepEqual(app.globalData.pickup, { date: '2026-08-26', mealPeriod: 'lunch', time: '11:40' });
  assert.equal(harness.requestCalls.at(-1).url, 'http://127.0.0.1:8080/api/v1/menu?date=2026-08-26&time=11%3A40');
});

test('PAGE-U04 detail requires pickup facts, renders images/specification and blocks unknown availability', async () => {
  const detail = {
    statusCode: 200,
    data: {
      product: {
        id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份',
        meal_period: 'dinner', images: [{ object_key: 'p/70.png', url: 'https://img.example.com/p/70.png' }],
        listed: true, sold_out: false, original_unit_price_cents: 1800,
      },
    },
  };
  const { app, harness } = readyHarness([VISITOR_IDENTITY, detail]);
  await harness.flush();
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
  const page = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(page, 'onLoad', { id: '70' });
  await harness.flush();

  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/catalog/products/70?date=2026-08-25&time=17%3A30');
  assert.equal(page.data.detailState, 'ready');
  assert.equal(page.data.m.images.length, 1);
  assert.equal(page.data.m.orderable, true);
  page.previewImage({ currentTarget: { dataset: { url: 'https://img.example.com/p/70.png' } } });
  assert.deepEqual(harness.previewCalls.at(-1), {
    current: 'https://img.example.com/p/70.png', urls: ['https://img.example.com/p/70.png'],
  });
});

test('BE-03 detail without a server pickup selection makes zero product request', async () => {
  const { harness } = readyHarness([]);
  await harness.flush();
  const page = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(page, 'onLoad', { id: '70' });
  await harness.flush();
  assert.equal(page.data.detailState, 'selection_required');
  assert.equal(harness.requestCalls.length, 1);
});
