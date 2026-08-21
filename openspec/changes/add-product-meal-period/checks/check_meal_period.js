#!/usr/bin/env node
/* PC 菜品餐段可售门禁。用法: node check_meal_period.js <repo-root> */
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
const MEALS = ['all', 'lunch', 'dinner'];
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sb.window;
};

check('every seeded product declares a meal period', () => {
  const { Seed } = ctx();
  for (const p of Seed.MENU) {
    assert.equal(MEALS.includes(p.meal), true, `${p.id}.meal = ${p.meal}`);
  }
});

check('the contract requires and validates the meal period', () => {
  const w = ctx();
  w.__store = { menu: JSON.parse(JSON.stringify(w.Seed.MENU)), cats: JSON.parse(JSON.stringify(w.Seed.ADMIN_CATS)) };
  const base = { name: '门禁测试菜', price: 20, cat: w.Seed.MENU[0].cat, desc: '', imgs: [] };
  return w.Api.saveProduct(Object.assign({}, base, { meal: 'lunch' }))
    .then(created => {
      assert.equal(created.meal, 'lunch', 'saved product lost its meal period');
      return w.Api.saveProduct(Object.assign({}, base, { name: '缺餐段' }))
        .then(() => { throw new Error('accepted a product without a meal period'); }, () => {});
    })
    .then(() => w.Api.saveProduct(Object.assign({}, base, { name: '非法餐段', meal: '中午' }))
      .then(() => { throw new Error('accepted an invalid meal period'); }, () => {}));
});

check('the contract is the single source of meal-period labels', () => {
  // 标签由契约层提供，页面从 Api.MEAL_LABEL 渲染 —— 因此断言写在运行态，
  // 而不是去页面里找「全天」这三个字。
  const { Api } = ctx();
  assert.equal(JSON.stringify(Api.MEALS), JSON.stringify(MEALS));
  assert.equal(JSON.stringify(Api.MEAL_LABEL), JSON.stringify({ all: '全天', lunch: '午餐', dinner: '晚餐' }));
});

check('the product form and table expose the meal period', () => {
  const src = read('pages/products.js');
  assert.match(src, /id="f-meal"/, 'edit form has no meal-period control');
  assert.match(src, /餐段可售/, 'edit form does not label the meal-period field');
  assert.match(src, /Api\.MEALS\.map/, 'form does not render options from the contract');
  assert.match(src, /\{ t: '餐段'/, 'product table has no meal-period column');
  assert.match(src, /Api\.MEAL_LABEL\[r\.meal\]/, 'table does not render the label from the contract');
  assert.match(src, /meal: root\.querySelector\('#f-meal'\)\.value/, 'form does not submit the meal period');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('MEAL_PERIOD_GATE=FAIL'); process.exit(1); }
  console.log('MEAL_PERIOD_GATE=PASS');
});
