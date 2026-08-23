const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'product-public-read', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

function frozenProduct(images) {
  return {
    id: '70', category_id: '7', name: '红烧肉', description: '慢炖', specification: '份',
    meal_period: 'all', images, listed: true, sold_out: false, original_unit_price_cents: 1800,
  };
}

async function readyHarness(requests) {
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION].concat(requests || []) });
  const app = harness.loadApp();
  await harness.flush();
  return { app, harness };
}

test('frozen product accepts all-meal and ordered zero to three images', async () => {
  await readyHarness();
  const store = require('../utils/productStore.js');
  const cases = [
    [],
    [{ object_key: 'images/one.png', url: '/api/v1/objects/images/one.png' }],
    [
      { object_key: 'images/one.png', url: '/api/v1/objects/images/one.png' },
      { object_key: 'images/two.jpg', url: 'https://cdn.example.com/images/two.jpg?version=2' },
      { object_key: 'images/three.png', url: '/api/v1/objects/images/three.png' },
    ],
  ];
  for (const images of cases) {
    const product = store.parse({ product: frozenProduct(images) });
    assert.equal(product.mealPeriod, 'all');
    assert.equal(product.images.length, images.length);
    if (images.length) assert.equal(product.cover.objectKey, images[0].object_key);
  }
});

test('bad product URL and legacy price fail closed without cart mutation', async () => {
  const bad = frozenProduct([{ object_key: 'images/one.png', url: 'http://cdn.example.com/images/one.png' }]);
  const { app, harness } = await readyHarness([{ statusCode: 200, data: { product: bad } }]);
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
  const page = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(page, 'onLoad', { id: '70' });
  await harness.flush();

  assert.equal(page.data.detailState, 'error');
  assert.equal(page.add(), false);
  assert.deepEqual(app.globalData.cart, {});

  const store = require('../utils/productStore.js');
  const legacy = frozenProduct([]);
  delete legacy.original_unit_price_cents;
  legacy.price_cents = 1800;
  assert.throws(() => store.parse({ product: legacy }), /CATALOG_UNAVAILABLE/);

  const normalized = store.parse({ product: frozenProduct([]) });
  delete normalized.meal_period;
  assert.throws(() => require('../utils/catalogStore.js').snapshotProduct(normalized), /CATALOG_UNAVAILABLE/);
});
