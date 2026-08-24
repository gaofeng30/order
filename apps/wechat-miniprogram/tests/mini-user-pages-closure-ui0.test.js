const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'user-pages-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

function product(images) {
  return {
    statusCode: 200,
    data: {
      product: {
        id: '70', category_id: '7', name: '三图菜品', description: '真实商品说明', specification: '每份 300 克',
        meal_period: 'lunch', images, listed: true, sold_out: false, original_unit_price_cents: 1888,
      },
    },
  };
}

async function loadDetail(images) {
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION, product(images)] });
  const app = harness.loadApp();
  await harness.flush();
  app.globalData.pickup = { date: '2026-08-25', mealPeriod: 'lunch', time: '11:30' };
  const page = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(page, 'onLoad', { id: '70' });
  await harness.flush();
  return page;
}

test('PAGE-U04 multi-image detail exposes the current gallery position while a single image has no counter', async () => {
  const images = [1, 2, 3].map(index => ({
    object_key: `products/70-${index}.png`,
    url: `https://img.example.com/products/70-${index}.png`,
  }));
  const page = await loadDetail(images);

  assert.equal(page.data.detailState, 'ready');
  assert.equal(page.data.imageIndex, 0);
  page.onImageChange({ detail: { current: 2 } });
  assert.equal(page.data.imageIndex, 2);

  const wxml = fs.readFileSync(path.resolve(__dirname, '../pages/detail/detail.wxml'), 'utf8');
  assert.match(wxml, /m\.images\.length\s*>\s*1/);
  assert.match(wxml, /imageIndex\s*\+\s*1/);
});
