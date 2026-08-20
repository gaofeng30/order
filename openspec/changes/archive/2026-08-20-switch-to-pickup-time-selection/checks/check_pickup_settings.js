#!/usr/bin/env node
/* PC 后台餐段与取餐时间配置门禁。用法: node check_pickup_settings.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const fails = [], pending = [];
const check = (label, fn) => {
  try {
    const r = fn();
    if (r && typeof r.then === 'function') pending.push(r.catch(e => fails.push(`${label}: ${String(e.message).split('\n')[0]}`)));
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sb.window;
};

check('settings carry per-period cutoffs and a configurable step', () => {
  const { Seed } = ctx();
  const s = Seed.SETTINGS;
  assert.equal(Object.hasOwn(s, 'cutoff'), false, 'settings still carry a single global cutoff');
  assert.equal(Object.hasOwn(s, 'openTime'), false, 'settings still carry a store-wide open time');
  assert.equal(typeof s.pickupStepMin, 'number');
  assert.equal(JSON.stringify(s.mealPeriods.map(p => p.key)), JSON.stringify(['lunch', 'dinner']));
  for (const p of s.mealPeriods) {
    for (const k of ['cutoff', 'from', 'to', 'name']) {
      assert.equal(typeof p[k], 'string', `meal period ${p.key} missing ${k}`);
    }
  }
});

check('settings contract validates the period configuration', () => {
  const w = ctx();
  w.__store = { store: { status: '营业中' }, settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)) };
  const base = { status: '营业中', notice: 'n', pickupPoint: '县前直营店', pickupStepMin: 30 };
  const periods = JSON.parse(JSON.stringify(w.Seed.SETTINGS.mealPeriods));
  return w.Api.saveSettings(Object.assign({}, base, { mealPeriods: periods }))
    .then(saved => {
      assert.equal(saved.pickupStepMin, 30);
      assert.equal(saved.mealPeriods.length, 2);
      // 结束早于开始必须被拒绝
      const bad = JSON.parse(JSON.stringify(periods));
      bad[0].to = '10:00';
      return w.Api.saveSettings(Object.assign({}, base, { mealPeriods: bad }))
        .then(() => { throw new Error('accepted a period whose end precedes its start'); }, () => {});
    })
    .then(() => w.Api.saveSettings(Object.assign({}, base, { mealPeriods: periods, pickupStepMin: 0 }))
      .then(() => { throw new Error('accepted a non-positive step'); }, () => {}))
    .then(() => w.Api.saveSettings(Object.assign({}, base, { mealPeriods: [] }))
      .then(() => { throw new Error('accepted an empty period list'); }, () => {}));
});

check('settings page edits periods, not a single cutoff', () => {
  const src = read('pages/settings.js');
  assert.doesNotMatch(src, /f-open|f-close|f-cut\b/, 'settings page still edits a store-wide cutoff');
  assert.match(src, /data-mp=/, 'settings page does not edit per-period times');
  assert.match(src, /f-step/, 'settings page does not expose the pickup step');
});

check('no module reads the removed single cutoff', () => {
  for (const rel of ['data/api.js', 'pages/settings.js', 'pages/dashboard.js', 'pages/orders.js', 'app.js']) {
    assert.doesNotMatch(read(rel), /openTime|closeTime|settings\.cutoff|s\.cutoff\b/,
      `${rel} still reads the removed store-wide cutoff`);
  }
});

check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p); else if (e.name.endsWith('.js')) files.push(p);
    }
  })(WA);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

Promise.all(pending).then(() => {
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('PICKUP_SETTINGS_GATE=FAIL'); process.exit(1); }
  console.log('PICKUP_SETTINGS_GATE=PASS');
});
