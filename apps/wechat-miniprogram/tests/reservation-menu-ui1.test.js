const assert = require('node:assert/strict');
const test = require('node:test');
const { createHarness } = require('./page-harness.js');

// The merged public contract uses server-provided dates and frozen product prices.
const selected = { date: '2026-08-25', mealPeriod: 'dinner', time: '17:30' };
function fixture(name = 'Fresh', overrides = {}) {
  return {
    selection: { date: selected.date, meal_period: selected.mealPeriod, time: selected.time },
    store_status: { business_status: 'open', service_date_available: true, meal_available: true, cutoff_passed: false },
    categories: [{ id: '9007199254740995', name: 'Dinner', products: [{
      id: '9007199254740993', category_id: '9007199254740995', name,
      description: '', specification: '份', meal_period: 'all', images: [],
      listed: true, sold_out: false, original_unit_price_cents: 12345,
      ...overrides,
    }] }],
  };
}
function setup() {
  const harness = createHarness();
  const app = harness.loadApp();
  app.globalData.session = { state: 'error' };
  const store = require('../utils/reservationMenuStore.js');
  const { pickup } = require('../utils/util.js');
  pickup.set(selected);
  const page = harness.loadPage('pages/menu/menu.js');
  return { harness, app, store, page };
}

test('reservation parser preserves large IDs, cents, order and frozen availability', () => {
  const { store } = setup();
  const menu = store.parseMenu(fixture());
  assert.deepEqual(menu.selection, selected);
  const product = menu.categories[0].products[0];
  assert.equal(product.id, '9007199254740993');
  assert.equal(product.category_id, '9007199254740995');
  assert.equal(product.price_cents, 12345);
  assert.equal(product.price_text, '123.45');
  assert.equal(product.orderable, true);
  assert.equal(store.parseMenu(fixture('Sold', { sold_out: true })).categories[0].products[0].orderable, false);
  for (const change of [{ original_unit_price_cents: 1.5 }, { category_id: '2' }, { listed: false }]) {
    assert.throws(() => store.parseMenu(fixture('Invalid', change)), /MENU_UNAVAILABLE/);
  }
});

test('pickup chooses server dates and skips unavailable meals without client date synthesis', () => {
  const { store } = setup();
  const options = store.parsePickupOptions({ dates: [{ date: selected.date, available: true, meal_periods: [
    { meal_period: 'lunch', available: false, cutoff_time: '11:00', pickup_times: ['11:30'] },
    { meal_period: 'dinner', available: true, cutoff_time: '17:00', pickup_times: ['17:30'] },
  ] }] });
  assert.deepEqual(store.firstAvailable(options), selected);
});

test('menu failure clears content, explicit retry recovers, empty response stays empty', async () => {
  const { page, store } = setup();
  store.loadPickupOptions = async () => ({ dates: [{ date: selected.date, available: true,
    mealPeriods: [{ mealPeriod: 'dinner', available: true, cutoffTime: '17:00', pickupTimes: ['17:30'] }] }] });
  store.loadMenu = async () => { throw new Error('unavailable'); };
  assert.equal(await page.loadMenu(), false);
  assert.equal(page.data.listState, 'error');
  assert.deepEqual(page.data.groups, []);
  store.loadMenu = async () => store.parseMenu(fixture());
  await page.retryCatalog();
  assert.equal(page.data.listState, 'ready');
  store.loadMenu = async () => store.parseMenu({ ...fixture(), categories: [] });
  await page.loadMenu();
  assert.equal(page.data.listState, 'empty');
});

for (const staleFailure of [false, true]) {
  test(`older menu ${staleFailure ? 'failure' : 'success'} cannot replace a newer pickup result`, async () => {
    const { page, store } = setup();
    const pending = [];
    store.loadMenu = selection => new Promise((resolve, reject) => pending.push({ selection, resolve, reject }));
    const oldLoad = page.loadMenu();
    await Promise.resolve();
    page._options = { dates: [{ date: selected.date, available: true,
      mealPeriods: [{ mealPeriod: 'dinner', available: true, pickupTimes: ['17:30', '18:00'] }] }] };
    const newLoad = page.pickPickerTime({ currentTarget: { dataset: { date: selected.date, period: 'dinner', t: '18:00' } } });
    await Promise.resolve();
    assert.equal(pending.length, 2);
    const fresh = store.parseMenu(fixture());
    fresh.selection = pending[1].selection;
    pending[1].resolve(fresh);
    await newLoad;
    if (staleFailure) pending[0].reject(new Error('old failure'));
    else pending[0].resolve(store.parseMenu(fixture('Stale')));
    await oldLoad;
    assert.equal(page.data.listState, 'ready');
    assert.equal(page.data.groups[0].products[0].name, 'Fresh');
    assert.equal(page.data.pickup.time, '18:00');
  });
}

test('sold-out products stay browsable but cannot be added through customization', async () => {
  const { page, store, app, harness } = setup();
  store.loadMenu = async () => store.parseMenu(fixture('Sold', { sold_out: true }));
  await page.loadMenu();
  const product = page.data.groups[0].products[0];
  page.add({ currentTarget: { dataset: { id: product.id } } });
  page.openCustomize({ currentTarget: { dataset: { id: product.id } } });
  assert.equal(page.data.czVisible, false);
  page.setData({ czItem: product });
  page.onCzConfirm({ detail: { qty: 1, flavors: [], note: '' } });
  assert.deepEqual(app.globalData.cart, {});
  page.goDetail({ currentTarget: { dataset: { id: product.id } } });
  assert.match(harness.navigationCalls.at(-1).url, /detail\?id=9007199254740993$/);
});
