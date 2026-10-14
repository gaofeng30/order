const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const calls = [];
const options = { dates: ['2026-10-13', '2026-10-14'].map(date => ({ date, available: true, meal_periods: ['lunch', 'dinner'].map(meal_period => ({ meal_period, available: true, cutoff_time: '10:00', pickup_times: ['11:30'] })) })) };
const api = { APIError: Error, getOptional: async url => { calls.push(url); if (url !== '/api/v1/menu/pickup-options') throw Error('Full menu must not be fetched to discover availability'); return options; } };
const context = { module: { exports: {} }, require: name => name === './apiClient.js' ? api : {} };
vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../../apps/wechat-miniprogram/utils/reservationMenuStore.js'), 'utf8'), context);
(async () => {
  const store = context.module.exports;
  const actual = await store.loadOrderableOptions();
  assert.equal(calls.length, 1);
  assert.equal(store.firstAvailable(actual).date, '2026-10-13');
  options.dates[0].meal_periods.forEach(m => m.available = false);
  assert.equal(store.firstAvailable(await store.loadOrderableOptions()).date, '2026-10-14');
  console.log('PASS: availability requires one lightweight request, no menu probes; empty meals skipped');
})().catch(error => { console.error(error); process.exitCode = 1; });
