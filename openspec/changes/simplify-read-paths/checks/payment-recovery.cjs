// Isolated read-only diagnostic. No network, wx API, credentials, or production data.
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const assert = require('node:assert/strict');
const root = path.resolve(__dirname, '../../../../apps/wechat-miniprogram');
const selection = { date: '2026-10-13', mealPeriod: 'lunch', time: '12:00', timeConfirmed: true };
const product = { id: '1', category_id: '1', name: 'Diagnostic fixture', description: '', specification: '', meal_period: 'lunch', images: [], listed: true, sold_out: false, original_unit_price_cents: 1000, price_cents: 1000, isStaffPrice: false };
let app;
function load(relative, dependencies) {
  const module = { exports: {} };
  vm.runInNewContext(fs.readFileSync(path.join(root, relative), 'utf8'), {
    module, exports: module.exports, getApp: () => app,
    require: name => {
      if (!Object.hasOwn(dependencies, name)) throw new Error(`Unexpected dependency ${relative}: ${name}`);
      return dependencies[name];
    },
  }, { filename: relative });
  return module.exports;
}
const catalog = load('utils/catalogStore.js', { './apiClient.js': { APIError: Error }, './catalogApi.js': {} });
const dates = load('utils/dateLabel.js', {});
const util = load('utils/util.js', { './catalogStore.js': catalog, './dateLabel.js': dates });
let confirmCalls = 0;
let page;
const deps = {
  '../../utils/apiClient.js': { newIdempotencyKey: () => 'diagnostic-key' },
  '../../utils/phoneStore.js': {}, '../../utils/identityStore.js': {}, '../../utils/storefrontStore.js': {},
  '../../utils/transactionStore.js': { confirm: async () => { confirmCalls++; return { state: 'ORDER_CREATED', orderID: 'fixture-order' }; } },
  '../../utils/catalogStore.js': catalog, '../../utils/reservationMenuStore.js': {},
  '../../utils/dateLabel.js': dates, '../../utils/util.js': util, '../../utils/navBehavior.js': {},
};
vm.runInNewContext(fs.readFileSync(path.join(root, 'pages/confirm/confirm.js'), 'utf8'), {
  Page: value => { page = value; }, getApp: () => app,
  require: name => { if (!Object.hasOwn(deps, name)) throw new Error(`Unexpected dependency ${name}`); return deps[name]; },
}, { filename: 'pages/confirm/confirm.js' });
async function scenario(name, availability, existing = true) {
  app = { globalData: { cart: {}, pickup: selection, menuState: { date: selection.date, mealPeriod: selection.mealPeriod, availability } } };
  util.cart.add(product, selection);
  if (existing) util.cart.savePayment(selection, { prepayment: { id: 'fixture-prepayment' }, needsPayment: false });
  confirmCalls = 0;
  const ctx = Object.assign({}, page, {
    _groupKey: util.cart.groupKey(selection), _selection: selection,
    _prepayment: existing ? { id: 'fixture-prepayment' } : null, _confirmKey: 'diagnostic-key', _needsPayment: false,
    _visible: false, data: { paymentState: 'pending', count: 1, pickupState: 'loading' },
    setData(patch) { Object.assign(this.data, patch); },
  });
  ctx.syncPayDisabled();
  const before = { cartCount: util.cart.count(ctx.availability(), ctx.selection()), payDisabled: ctx.data.payDisabled, locked: util.cart.isLocked(selection) };
  const result = await ctx.payOnce();
  return { name, before, result, transactionConfirmCalls: confirmCalls, lockedAfter: util.cart.isLocked(selection) };
}
(async () => {
  const results = [
    await scenario('existing prepayment + available item', { '1': { soldOut: false, orderable: true } }),
    await scenario('existing prepayment + sold out item', { '1': { soldOut: true, orderable: false } }),
    await scenario('existing prepayment + cutoff item', { '1': { soldOut: false, orderable: false } }),
  ];
  assert.equal(results[0].transactionConfirmCalls, 1);
  assert.equal(results[0].result, true);
  for (const blocked of results.slice(1)) {
    assert.equal(blocked.before.payDisabled, false);
    assert.equal(blocked.transactionConfirmCalls, 1);
    assert.equal(blocked.result, true);
    assert.equal(blocked.lockedAfter, false);
  }
  for (const availability of [{ '1': { soldOut: true, orderable: false } }, { '1': { soldOut: false, orderable: false } }]) {
    const fresh = await scenario('new purchase unavailable', availability, false);
    assert.equal(fresh.result, false);
    assert.equal(fresh.transactionConfirmCalls, 0);
    assert.equal(fresh.lockedAfter, false);
    results.push(fresh);
  }
  console.log(JSON.stringify({ candidate: 'local working tree', scope: 'isolated VM using real confirm.js, util.js, catalogStore.js and dateLabel.js; transaction.confirm mocked; no external services', results }, null, 2));
})().catch(error => { console.error(error); process.exitCode = 1; });
