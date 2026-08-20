const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

const HUGE_PRODUCT_ID = '9007199254740993';
const HUGE_CATEGORY_ID = '9007199254740995';
const UNSUPPORTED_PRODUCT_FIELDS = ['stock', 'sold', 'status', 'tags', 'image', 'availability', 'orderable'];

function assertUnsupportedProductFieldsAbsent(value, label) {
  for (const field of UNSUPPORTED_PRODUCT_FIELDS) {
    assert.equal(Object.hasOwn(value, field), false, `${label}.${field}`);
  }
}

function product(id, categoryID, name, cents) {
  return {
    id,
    category_id: categoryID,
    name,
    description: `${name} description`,
    specification: `${name} specification`,
    price_cents: cents,
  };
}

function catalogFixture() {
  return {
    categories: [
      {
        id: HUGE_CATEGORY_ID,
        name: 'Server First',
        products: [
          Object.assign(product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Server One', 12345), {
            stock: 99,
            status: 'soldout',
            tags: ['mock'],
            image: '/mock.jpg',
            availability: 'AVAILABLE',
            orderable: true,
          }),
          product('2', HUGE_CATEGORY_ID, 'Server Two', 250),
          product('3', HUGE_CATEGORY_ID, 'Server Three', 300),
        ],
      },
      {
        id: '7',
        name: 'Server Second',
        products: [
          product('4', '7', 'Server Four', 401),
          product('5', '7', 'Server Five', 599),
        ],
      },
    ],
  };
}

test('legacy behavior boundary: list lifecycle sends a catalog request', () => {
  const harness = createHarness({ requests: [{ statusCode: 200, data: { categories: [] } }] });
  harness.loadApp();
  const page = harness.loadPage('pages/home/home.js');
  harness.invoke(page, 'onShow');
  assert.equal(
    harness.requestCalls.length,
    1,
    `request count = ${harness.requestCalls.length}, want 1`,
  );
});

test('legacy behavior boundary: network failure is retryable without mock fallback', async () => {
  const harness = createHarness({
    requests: [{ networkError: true }, { statusCode: 200, data: catalogFixture() }],
  });
  harness.loadApp();
  const page = harness.loadPage('pages/home/home.js');

  const first = harness.invoke(page, 'onShow');
  assert.equal(
    harness.requestCalls.length,
    1,
    `network failure request count = ${harness.requestCalls.length}, want 1`,
  );
  assert.equal(page.data.listState, 'loading');
  await first;
  assert.equal(page.data.listState, 'error');
  assert.deepEqual(page.data.signature, []);

  const retry = page.retryCatalog();
  assert.equal(page.data.listState, 'loading');
  await retry;
  assert.equal(page.data.listState, 'ready');
  assert.equal(page.data.signature[0].id, HUGE_PRODUCT_ID);
  assert.equal(harness.requestCalls.length, 2);
});

test('legacy behavior boundary: unknown detail never falls back to p001', async () => {
  const harness = createHarness({
    requests: [{
      statusCode: 404,
      data: { error: { code: 'PRODUCT_NOT_FOUND', message: 'product not found' } },
    }],
  });
  harness.loadApp();
  const page = harness.loadPage('pages/detail/detail.js');
  harness.invoke(page, 'onLoad', { id: HUGE_PRODUCT_ID });
  await harness.flush();
  const currentID = page.data.m && page.data.m.id;
  assert.equal(currentID, null, `fallback product id = ${currentID}, want null`);
  assert.equal(page.data.detailState, 'not_found');
});

test('catalog transport and catalog store preserve URL, types, cents, order and error classes', async () => {
  const fixture = catalogFixture();
  const harness = createHarness({ requests: [{ statusCode: 200, data: fixture }] });
  harness.loadApp();
  const store = require('../utils/catalogStore.js');

  const catalog = await store.loadCatalog();
  assert.equal(harness.requestCalls.length, 1);
  assert.equal(harness.requestCalls[0].url, 'http://127.0.0.1:8080/api/v1/catalog');
  assert.equal(harness.requestCalls[0].method, 'GET');
  assert.equal(Object.hasOwn(harness.requestCalls[0], 'data'), false);
  assert.equal(Object.hasOwn(harness.requestCalls[0], 'header'), false);
  assert.deepEqual(catalog.categories.map(category => category.id), [HUGE_CATEGORY_ID, '7']);
  assert.deepEqual(store.flattenProducts(catalog.categories).map(item => item.id), [HUGE_PRODUCT_ID, '2', '3', '4', '5']);
  assert.equal(catalog.categories[0].products[0].id, HUGE_PRODUCT_ID);
  assert.equal(catalog.categories[0].products[0].price_cents, 12345);
  assert.equal(store.formatCents(12345), '123.45');
  assertUnsupportedProductFieldsAbsent(catalog.categories[0].products[0], 'catalog product');

  harness.enqueueRequest({
    statusCode: 404,
    data: { error: { code: 'PRODUCT_NOT_FOUND', message: 'product not found' } },
  });
  await assert.rejects(
    store.loadProduct(HUGE_PRODUCT_ID),
    error => error && error.code === 'PRODUCT_NOT_FOUND',
  );
  assert.equal(harness.requestCalls[1].url, `http://127.0.0.1:8080/api/v1/catalog/products/${HUGE_PRODUCT_ID}`);

  harness.enqueueRequest({ statusCode: 503, data: { internal: 'not exposed' } });
  await assert.rejects(
    store.loadCatalog(),
    error => error && error.code === 'CATALOG_UNAVAILABLE' && !String(error.message).includes('internal'),
  );

  harness.enqueueRequest({ statusCode: 200, data: { categories: [{ id: 1 }] } });
  await assert.rejects(store.loadCatalog(), error => error && error.code === 'CATALOG_UNAVAILABLE');
});

test('home list lifecycle covers loading, error, retry, ready and stable first four', async () => {
  const harness = createHarness({
    requests: [{ networkError: true }, { statusCode: 200, data: catalogFixture() }],
  });
  harness.loadApp();
  const page = harness.loadPage('pages/home/home.js');

  const first = harness.invoke(page, 'onShow');
  assert.equal(page.data.listState, 'loading');
  await first;
  assert.equal(page.data.listState, 'error');
  assert.deepEqual(page.data.signature, []);

  const retry = page.retryCatalog();
  assert.equal(page.data.listState, 'loading');
  await retry;
  assert.equal(page.data.listState, 'ready');
  assert.deepEqual(page.data.signature.map(item => item.id), [HUGE_PRODUCT_ID, '2', '3', '4']);
  assertUnsupportedProductFieldsAbsent(page.data.signature[0], 'home product view');
  assert.equal(Object.hasOwn(page.data.signature[0], 'price'), false, 'home product view.price');
  assert.equal(harness.requestCalls.length, 2);
});

test('non-200 2xx list and detail responses remain unavailable without fallback or retry', async () => {
  const homeHarness = createHarness({ requests: [{ statusCode: 201, data: catalogFixture() }] });
  homeHarness.loadApp();
  const home = homeHarness.loadPage('pages/home/home.js');
  await homeHarness.invoke(home, 'onShow');
  assert.equal(home.data.listState, 'error');
  assert.deepEqual(home.data.signature, []);
  assert.equal(homeHarness.requestCalls.length, 1);

  const detailProduct = product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Unexpected 204 Product', 12345);
  const detailHarness = createHarness({ requests: [{ statusCode: 204, data: { product: detailProduct } }] });
  detailHarness.loadApp();
  const detail = detailHarness.loadPage('pages/detail/detail.js');
  await detailHarness.invoke(detail, 'onLoad', { id: HUGE_PRODUCT_ID });
  assert.equal(detail.data.detailState, 'error');
  assert.equal(detail.data.m, null);
  assert.equal(detailHarness.requestCalls.length, 1);
});

test('list empty is only categories empty and enabled empty category remains ready', async () => {
  const homeHarness = createHarness({
    requests: [
      { statusCode: 200, data: { categories: [] } },
      { statusCode: 200, data: catalogFixture() },
    ],
  });
  homeHarness.loadApp();
  const home = homeHarness.loadPage('pages/home/home.js');
  await homeHarness.invoke(home, 'onShow');
  assert.equal(home.data.listState, 'empty');
  assert.deepEqual(home.data.signature, []);
  const homeRetry = home.retryCatalog();
  assert.equal(home.data.listState, 'loading');
  await homeRetry;
  assert.equal(home.data.listState, 'ready');
  assert.equal(home.data.signature[0].id, HUGE_PRODUCT_ID);
  assert.equal(homeHarness.requestCalls.length, 2);

  const menuHarness = createHarness({
    requests: [
      { statusCode: 200, data: { categories: [] } },
      { statusCode: 200, data: catalogFixture() },
    ],
  });
  menuHarness.loadApp();
  const retriedMenu = menuHarness.loadPage('pages/menu/menu.js');
  await menuHarness.invoke(retriedMenu, 'onShow');
  assert.equal(retriedMenu.data.listState, 'empty');
  assert.deepEqual(retriedMenu.data.groups, []);
  const menuRetry = retriedMenu.retryCatalog();
  assert.equal(retriedMenu.data.listState, 'loading');
  await menuRetry;
  assert.equal(retriedMenu.data.listState, 'ready');
  assert.equal(retriedMenu.data.groups[0].products[0].id, HUGE_PRODUCT_ID);
  assert.equal(menuHarness.requestCalls.length, 2);

  const groupHarness = createHarness({
    requests: [{ statusCode: 200, data: { categories: [{ id: '8', name: 'Empty Server Group', products: [] }] } }],
  });
  groupHarness.loadApp();
  const menu = groupHarness.loadPage('pages/menu/menu.js');
  await groupHarness.invoke(menu, 'onShow');
  assert.equal(menu.data.listState, 'ready');
  assert.equal(menu.data.groups[0].id, '8');
  assert.deepEqual(menu.data.groups[0].products, []);
  assert.equal(menu.data.active, '8');

  const homeGroupHarness = createHarness({
    requests: [{ statusCode: 200, data: { categories: [{ id: '8', name: 'Empty Server Group', products: [] }] } }],
  });
  homeGroupHarness.loadApp();
  const groupedHome = homeGroupHarness.loadPage('pages/home/home.js');
  await homeGroupHarness.invoke(groupedHome, 'onShow');
  assert.equal(groupedHome.data.listState, 'ready');
  assert.deepEqual(groupedHome.data.signature, []);

  const viewHarness = createHarness({ requests: [{ statusCode: 200, data: catalogFixture() }] });
  viewHarness.loadApp();
  const readyMenu = viewHarness.loadPage('pages/menu/menu.js');
  await viewHarness.invoke(readyMenu, 'onShow');
  assertUnsupportedProductFieldsAbsent(readyMenu.data.groups[0].products[0], 'menu product view');
  assert.equal(Object.hasOwn(readyMenu.data.groups[0].products[0], 'price'), false, 'menu product view.price');
});

test('detail lifecycle retries the same huge string ID without fallback', async () => {
  const readyProduct = Object.assign(product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Detail Server Product', 12345), {
    stock: 99,
    sold: 88,
    status: 'soldout',
    tags: ['mock'],
    image: '/mock.jpg',
    availability: 'AVAILABLE',
    orderable: true,
  });
  const harness = createHarness({
    requests: [
      { statusCode: 503, data: { error: { code: 'CATALOG_UNAVAILABLE' } } },
      { statusCode: 200, data: { product: readyProduct } },
    ],
  });
  harness.loadApp();
  const page = harness.loadPage('pages/detail/detail.js');

  const first = harness.invoke(page, 'onLoad', { id: HUGE_PRODUCT_ID });
  assert.equal(page.data.detailState, 'loading');
  await first;
  assert.equal(page.data.detailState, 'error');
  assert.equal(page.data.m, null);

  const retry = page.retryProduct();
  assert.equal(page.data.detailState, 'loading');
  await retry;
  assert.equal(page.data.detailState, 'ready');
  assert.equal(page.data.m.id, HUGE_PRODUCT_ID);
  assert.equal(page.data.m.price_cents, 12345);
  assert.equal(page.data.m.price_text, '123.45');
  assertUnsupportedProductFieldsAbsent(page.data.m, 'detail product view');
  assert.equal(Object.hasOwn(page.data.m, 'price'), false, 'detail product view.price');
  assert.equal(page.data.qty, 0);
  assert.deepEqual(
    harness.requestCalls.map(call => call.url),
    Array(2).fill(`http://127.0.0.1:8080/api/v1/catalog/products/${HUGE_PRODUCT_ID}`),
  );
});

test('detail selection handler carries the API snapshot through cart and confirm after response mutation and failure', async () => {
  const responseProduct = Object.assign(product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'API Selected Product', 12345), {
    stock: 99,
    sold: 88,
    status: 'soldout',
    tags: ['mock'],
    image: '/mock.jpg',
    availability: 'AVAILABLE',
    orderable: true,
  });
  const harness = createHarness({
    requests: [
      { statusCode: 200, data: { product: responseProduct } },
      { networkError: true },
    ],
  });
  const app = harness.loadApp();
  const detail = harness.loadPage('pages/detail/detail.js');
  await harness.invoke(detail, 'onLoad', { id: HUGE_PRODUCT_ID });
  assert.equal(detail.data.detailState, 'ready');
  assert.equal(detail.data.qty, 0);

  detail.openCustomize();
  assert.equal(detail.data.czVisible, true);
  detail.onCzConfirm({ detail: { qty: 2, flavors: ['少盐'], note: 'from page handler' } });
  assert.equal(detail.data.czVisible, false);
  assert.equal(detail.data.qty, 2);
  detail.add();
  assert.equal(detail.data.qty, 3);

  const expectedSnapshot = product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'API Selected Product', 12345);
  assert.deepEqual(app.globalData.cart[HUGE_PRODUCT_ID], {
    product: expectedSnapshot,
    qty: 3,
    flavors: ['少盐'],
    note: 'from page handler',
  });
  assertUnsupportedProductFieldsAbsent(app.globalData.cart[HUGE_PRODUCT_ID].product, 'page-selected cart snapshot');
  assert.equal(Object.hasOwn(app.globalData.cart[HUGE_PRODUCT_ID].product, 'price'), false);
  assert.equal(Object.hasOwn(app.globalData.cart[HUGE_PRODUCT_ID].product, 'price_text'), false);

  responseProduct.name = 'Mutated API Response';
  responseProduct.price_cents = 1;
  await detail.retryProduct();
  assert.equal(detail.data.detailState, 'error');
  assert.deepEqual(app.globalData.cart[HUGE_PRODUCT_ID].product, expectedSnapshot);
  assert.equal(harness.requestCalls.length, 2);

  let menuReads = 0;
  Object.defineProperty(app.globalData, 'menu', {
    configurable: true,
    get() { menuReads += 1; throw new Error('legacy menu read'); },
  });
  const requestCountBeforeConfirm = harness.requestCalls.length;
  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  await harness.flush(90);
  assert.equal(harness.requestCalls.length, requestCountBeforeConfirm);
  assert.equal(menuReads, 0);
  assert.equal(confirm.data.items[0].item.id, HUGE_PRODUCT_ID);
  assert.equal(confirm.data.items[0].item.name, 'API Selected Product');
  assert.equal(confirm.data.items[0].item.price_cents, 12345);
  assert.equal(confirm.data.items[0].item.price_text, '123.45');
  assert.equal(confirm.data.items[0].line_total_cents, 37035);
  assert.equal(confirm.data.items[0].line_total_text, '370.35');
  assert.deepEqual(confirm.data.items[0].flavors, ['少盐']);
  assert.equal(confirm.data.items[0].note, 'from page handler');
});

test('cart snapshot drives confirm, subtotal-only pricing and mock pay without menu or catalog reads', async () => {
  const original = Object.assign(product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Snapshot Product', 12345), {
    stock: 99,
    sold: 88,
    status: 'soldout',
    tags: ['mock'],
    image: '/mock.jpg',
    availability: 'AVAILABLE',
    orderable: true,
  });
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart } = require('../utils/util.js');

  cart.setPrefs(original, { qty: 2, flavors: ['少盐'], note: 'separate' });
  original.name = 'Mutated Response';
  original.price_cents = 1;

  assert.deepEqual(app.globalData.cart[HUGE_PRODUCT_ID], {
    product: product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Snapshot Product', 12345),
    qty: 2,
    flavors: ['少盐'],
    note: 'separate',
  });
  assert.equal(cart.totalCents(), 24690);
  assert.equal(cart.list()[0].item.price_text, '123.45');
  assert.equal(cart.list()[0].item.price, 123.45);
  assert.equal(Object.hasOwn(app.globalData.cart[HUGE_PRODUCT_ID].product, 'price'), false);
  assert.equal(Object.hasOwn(app.globalData.cart[HUGE_PRODUCT_ID].product, 'price_text'), false);
  assertUnsupportedProductFieldsAbsent(app.globalData.cart[HUGE_PRODUCT_ID].product, 'cart product snapshot');

  assert.equal(Object.hasOwn(app.globalData, 'coupons'), false);
  let menuReads = 0;
  Object.defineProperty(app.globalData, 'menu', {
    configurable: true,
    get() { menuReads += 1; throw new Error('legacy menu read'); },
  });

  const confirm = harness.loadPage('pages/confirm/confirm.js');
  harness.invoke(confirm, 'onLoad');
  await harness.flush(90);
  assert.equal(harness.requestCalls.length, 0);
  assert.equal(menuReads, 0);
  assert.equal(confirm.data.items[0].item.id, HUGE_PRODUCT_ID);
  assert.equal(confirm.data.items[0].item.name, 'Snapshot Product');
  assert.equal(confirm.data.items[0].item.price_cents, 12345);
  assert.equal(confirm.data.items[0].line_total_cents, 24690);
  assert.equal(confirm.data.items[0].line_total_text, '246.90');
  assert.equal(confirm.data.subtotal_cents, 24690);
  assert.equal(confirm.data.subtotal_text, '246.90');
  assert.equal(confirm.data.payable_cents, confirm.data.subtotal_cents);
  assert.equal(confirm.data.payable_text, '246.90');
  assert.equal(typeof confirm.openCoupon, 'undefined');

  confirm.pay();
  assert.equal(menuReads, 0);
  assert.equal(harness.requestCalls.length, 0);
  assert.equal(app.globalData.orders[0].items[0][0], HUGE_PRODUCT_ID);
  assert.equal(app.globalData.orders[0].items[0][2], 123.45);
  assert.equal(app.globalData.orders[0].total, '246.90');
  assert.equal(app.globalData.orders[0].subtotal, '246.90');
  assert.equal(harness.navigationCalls.at(-1).type, 'redirectTo');
  assert.equal(harness.navigationCalls.at(-1).url, '/pages/result/result');
});

test('existing cart snapshot quantity remains editable after menu catalog failure', async () => {
  const selected = product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Selected Snapshot', 12345);
  const harness = createHarness({ requests: [{ networkError: true }] });
  const app = harness.loadApp();
  const { cart } = require('../utils/util.js');
  cart.setPrefs(selected, { qty: 1, flavors: ['少盐'], note: 'keep' });
  const snapshotBefore = JSON.stringify(app.globalData.cart[HUGE_PRODUCT_ID].product);

  let menuReads = 0;
  Object.defineProperty(app.globalData, 'menu', {
    configurable: true,
    get() { menuReads += 1; throw new Error('legacy menu read'); },
  });

  const menu = harness.loadPage('pages/menu/menu.js');
  await harness.invoke(menu, 'onShow');
  assert.equal(menu.data.listState, 'error');
  assert.equal(harness.requestCalls.length, 1);

  menu.add({ currentTarget: { dataset: { id: HUGE_PRODUCT_ID } } });
  assert.equal(cart.qty(HUGE_PRODUCT_ID), 2);
  assert.equal(JSON.stringify(app.globalData.cart[HUGE_PRODUCT_ID].product), snapshotBefore);
  assert.deepEqual(app.globalData.cart[HUGE_PRODUCT_ID].product, selected);
  assert.equal(harness.requestCalls.length, 1);
  assert.equal(menuReads, 0);
});

test('cart rejects invalid quantities and unsafe integer amounts without mutation', () => {
  const selected = product(HUGE_PRODUCT_ID, HUGE_CATEGORY_ID, 'Validated Snapshot', 12345);
  const harness = createHarness();
  const app = harness.loadApp();
  const { cart } = require('../utils/util.js');
  cart.setPrefs(selected, { qty: 2, flavors: ['少盐'], note: 'keep' });
  const validEntry = JSON.stringify(app.globalData.cart[HUGE_PRODUCT_ID]);

  for (const qty of [0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1]) {
    assert.throws(
      () => cart.setPrefs(selected, { qty, flavors: [], note: '' }),
      error => error && error.code === 'CART_INVALID' && error.message === 'cart unavailable',
    );
    assert.equal(JSON.stringify(app.globalData.cart[HUGE_PRODUCT_ID]), validEntry);
  }

  const maxPrice = product('11', HUGE_CATEGORY_ID, 'Max Price Snapshot', Number.MAX_SAFE_INTEGER);
  cart.setPrefs(maxPrice, { qty: 1, flavors: [], note: '' });
  const maxEntry = JSON.stringify(app.globalData.cart['11']);
  assert.throws(() => cart.add(maxPrice), error => error && error.code === 'CART_INVALID');
  assert.equal(JSON.stringify(app.globalData.cart['11']), maxEntry);
  assert.throws(
    () => cart.setPrefs(maxPrice, { qty: 2, flavors: [], note: '' }),
    error => error && error.code === 'CART_INVALID',
  );
  assert.equal(JSON.stringify(app.globalData.cart['11']), maxEntry);

  cart.clear();
  cart.setPrefs(product('12', HUGE_CATEGORY_ID, 'Zero One', 0), { qty: Number.MAX_SAFE_INTEGER, flavors: [], note: '' });
  cart.setPrefs(product('13', HUGE_CATEGORY_ID, 'Zero Two', 0), { qty: 1, flavors: [], note: '' });
  assert.throws(() => cart.count(), error => error && error.code === 'CART_INVALID');

  cart.clear();
  cart.setPrefs(product('14', HUGE_CATEGORY_ID, 'Sum Max', Number.MAX_SAFE_INTEGER), { qty: 1, flavors: [], note: '' });
  cart.setPrefs(product('15', HUGE_CATEGORY_ID, 'Sum One', 1), { qty: 1, flavors: [], note: '' });
  assert.throws(() => cart.totalCents(), error => error && error.code === 'CART_INVALID');
});

test('WXML exposes exact recoverable states and public product files contain no legacy catalog fields', () => {
  const read = relativePath => fs.readFileSync(path.join(miniprogramRoot, relativePath), 'utf8');
  const homeWXML = read('pages/home/home.wxml');
  const menuWXML = read('pages/menu/menu.wxml');
  const detailWXML = read('pages/detail/detail.wxml');
  const confirmWXML = read('pages/confirm/confirm.wxml');

  for (const source of [homeWXML, menuWXML]) {
    for (const state of ['loading', 'empty', 'error', 'ready']) assert.match(source, new RegExp(`listState === '${state}'`));
    assert.match(source, /bindtap="retryCatalog"/);
    assert.match(
      source,
      /<view wx:elif="\{\{listState === 'empty'\}\}" class="catalog-state">\s*<text>当前目录为空<\/text>\s*<view class="btn btn--ghost-blue btn--sm" bindtap="retryCatalog">再次加载<\/view>\s*<\/view>/,
    );
  }
  assert.match(
    homeWXML,
    /<block wx:elif="\{\{listState === 'ready'\}\}">\s*<view wx:if="\{\{signature\.length === 0\}\}" class="catalog-state">暂无招牌商品<\/view>\s*<scroll-view wx:else class="signature"/,
  );
  for (const state of ['loading', 'not_found', 'error', 'ready']) {
    assert.match(detailWXML, new RegExp(`detailState === '${state}'`));
  }
  assert.match(detailWXML, /bindtap="retryProduct"/);
  assert.match(detailWXML, /<stepper wx:if="\{\{qty > 0\}\}" value="\{\{qty\}\}" bind:sub="sub" bind:add="add" \/>/);
  assert.match(detailWXML, /<view wx:else class="have-qty">尚未选择<\/view>/);
  assert.doesNotMatch(detailWXML, /购物车已有[\s\S]*\{\{qty\}\}[\s\S]*份/);
  assert.doesNotMatch(confirmWXML, /bindtap="openCoupon"/);
  assert.match(confirmWXML, /\{\{subtotal_text\}\}/);
  assert.match(confirmWXML, /<money v="\{\{payable_text\}\}"/);
  assert.match(confirmWXML, /bindtap="pay"/);
  assert.match(
    confirmWXML,
    /<view class="snapshot-notice faint"[^>]*>目录商品为本地快照，不锁定价格或库存；真实结算与下单需服务端再次校验。<\/view>/,
  );

  const publicScripts = [
    'pages/home/home.js', 'pages/menu/menu.js', 'pages/detail/detail.js', 'pages/confirm/confirm.js',
  ].map(read).join('\n');
  for (const forbidden of [/menuList\s*\(/, /itemById\s*\(/, /data\.CATS/]) {
    assert.doesNotMatch(publicScripts, forbidden);
  }

  const productMarkup = [
    'pages/home/home.wxml', 'pages/menu/menu.wxml', 'pages/detail/detail.wxml', 'pages/confirm/confirm.wxml',
  ].map(read).join('\n');
  for (const field of UNSUPPORTED_PRODUCT_FIELDS) {
    assert.doesNotMatch(
      productMarkup,
      new RegExp(`\\b(?:item|m|ci\\.item)\\.${field}\\b`),
      `product rendering must not bind ${field}`,
    );
  }
  for (const forbidden of [/p001/, /p005/, /<imageph\b/]) {
    assert.doesNotMatch(publicScripts + productMarkup, forbidden);
  }
});
