const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = {
  statusCode: 201,
  data: { access_token: 'public-read-session', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' },
};

async function readyStore() {
  const harness = createHarness({ logins: [{ code: 'public-read-code' }], requests: [SESSION] });
  harness.loadApp();
  await harness.flush();
  return require('../utils/storefrontStore.js');
}

test('public storefront accepts only the frozen envelope and resolves a local object URL', async () => {
  const store = await readyStore();
  const parsed = store.parse({
    storefront: {
      name: '绥安食品',
      address: '党政办公中心后院老食堂',
      pickup_point: '北门',
      announcement: '今日公告',
      business_status: 'open',
      flavors: ['少盐', '免葱'],
      launch_layer: {
        image: { object_key: 'images/launch.png', url: '/api/v1/objects/images/launch.png' },
        center_x: 0.5,
        center_y: 0.35,
        width_ratio: 0.6,
        aspect_ratio: 2,
      },
    },
  });

  assert.equal(parsed.launchLayer.image.objectKey, 'images/launch.png');
  assert.equal(parsed.launchLayer.image.url, 'http://127.0.0.1:8080/api/v1/objects/images/launch.png');
  assert.deepEqual(parsed.flavors, ['少盐', '免葱']);
  assert.equal(parsed.launchLayer.widthPercent, 60);
  assert.equal(parsed.launchLayer.heightVw, 30);

  assert.throws(() => store.parse({ settings: { store_name: '旧契约' } }), /STOREFRONT_UNAVAILABLE/);
});

test('public object URLs reject absolute HTTP, credentials, fragments and traversal', async () => {
  const store = await readyStore();
  const base = {
    name: '绥安食品', address: '地址', pickup_point: '北门', announcement: '',
    business_status: 'open', flavors: [],
  };
  for (const url of [
    'http://evil.example/image.png',
    'https://user:pass@img.example.com/image.png',
    'https://img.example.com/image.png#fragment',
    '/api/v1/objects/../secret',
  ]) {
    assert.throws(() => store.parse({
      storefront: Object.assign({}, base, {
        launch_layer: {
          image: { object_key: 'images/launch.png', url },
          center_x: 0.5, center_y: 0.5, width_ratio: 0.5, aspect_ratio: 1,
        },
      }),
    }), /STOREFRONT_UNAVAILABLE/, url);
  }
});

test('launch page exposes the server layer and blocks navigation when storefront facts are invalid', async () => {
  const frozen = {
    statusCode: 200,
    data: {
      storefront: {
        name: '绥安食品', address: '地址', pickup_point: '北门', announcement: '',
        business_status: 'open', flavors: [],
        launch_layer: {
          image: { object_key: 'images/launch.png', url: '/api/v1/objects/images/launch.png' },
          center_x: 0.5, center_y: 0.4, width_ratio: 0.6, aspect_ratio: 2,
        },
      },
    },
  };
  const harness = createHarness({ logins: [{ code: 'code' }], requests: [SESSION, frozen] });
  harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/launch/launch.js');
  await harness.invoke(page, 'onShow');
  await harness.flush();
  assert.equal(page.data.launchLayer.image.objectKey, 'images/launch.png');
  assert.equal(page.go({ currentTarget: { dataset: { to: 'home' } } }), true);
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/home/home');
  page.dismissLaunchLayer();
  assert.equal(page.data.launchLayer, null);
  harness.invoke(page, 'onUnload');

  const badHarness = createHarness({
    logins: [{ code: 'code' }],
    requests: [SESSION, { statusCode: 200, data: { settings: { store_name: '旧契约' } } }],
  });
  badHarness.loadApp();
  await badHarness.flush();
  const badPage = badHarness.loadPage('pages/launch/launch.js');
  await badHarness.invoke(badPage, 'onShow');
  await badHarness.flush();
  assert.equal(badPage.data.storefrontState, 'error');
  assert.equal(badPage.go({ currentTarget: { dataset: { to: 'home' } } }), false);
  assert.equal(badHarness.navigationCalls.length, 0);
});

test('home shows the same server launch layer and does not navigate before storefront readiness', async () => {
  const frozen = {
    statusCode: 200,
    data: {
      storefront: {
        name: '绥安食品', address: '地址', pickup_point: '北门', announcement: '',
        business_status: 'open', flavors: [],
        launch_layer: {
          image: { object_key: 'images/launch.png', url: 'https://cdn.example.com/launch.png' },
          center_x: 0.5, center_y: 0.3, width_ratio: 0.5, aspect_ratio: 1,
        },
      },
    },
  };
  const harness = createHarness({
    logins: [{ code: 'code' }], requests: [SESSION, frozen, { statusCode: 200, data: { orders: [] } }],
  });
  harness.loadApp();
  await harness.flush();
  const page = harness.loadPage('pages/home/home.js');
  assert.equal(page.toMenu(), false);
  assert.equal(harness.navigationCalls.length, 0);
  await harness.invoke(page, 'onShow');
  await harness.flush();
  assert.equal(page.data.launchLayer.image.url, 'https://cdn.example.com/launch.png');
  assert.equal(page.toMenu(), true);
  page.dismissLaunchLayer();
  harness.invoke(page, 'onUnload');
});
