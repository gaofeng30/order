const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'menu-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

const OPTIONS = {
  statusCode: 200,
  data: {
    timezone: 'Asia/Shanghai',
    dates: [
      {
        date: '2026-08-25', orderable: true,
        meals: [
          { code: 'lunch', cutoff_at: '2026-08-25T10:45:00+08:00', orderable: false, pickup_times: ['11:40'] },
          { code: 'dinner', cutoff_at: '2026-08-25T17:00:00+08:00', orderable: true, pickup_times: ['17:30', '18:00'] },
        ],
      },
      {
        date: '2026-08-26', orderable: true,
        meals: [{ code: 'lunch', cutoff_at: '2026-08-26T10:45:00+08:00', orderable: true, pickup_times: ['11:40'] }],
      },
    ],
  },
};

const MENU = {
  statusCode: 200,
  data: {
    selection: { date: '2026-08-25', time: '17:30', timezone: 'Asia/Shanghai' },
    meal: { code: 'dinner', cutoff_at: '2026-08-25T17:00:00+08:00', orderable: true },
    categories: [{
      id: '7', name: '晚餐', products: [
        { id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份', price_cents: 1800, sold_out: false, orderable: true },
        { id: '71', category_id: '7', name: '米饭', description: '', specification: '碗', price_cents: 200, sold_out: true, orderable: false },
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
  const { app, harness } = readyHarness([OPTIONS, MENU]);
  await harness.flush();
  const page = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();

  assert.deepEqual(app.globalData.pickup, { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' });
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/menu/pickup-options');
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/menu?date=2026-08-25&time=17%3A30');
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
  secondMenu.data.selection = { date: '2026-08-26', time: '11:40', timezone: 'Asia/Shanghai' };
  secondMenu.data.meal.code = 'lunch';
  const { app, harness } = readyHarness([OPTIONS, MENU, secondMenu]);
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
  const { app, harness } = readyHarness([detail]);
  await harness.flush();
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
  const page = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(page, 'onLoad', { id: '70' });
  await harness.flush();

  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/catalog/products/70?date=2026-08-25&time=17%3A30');
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
