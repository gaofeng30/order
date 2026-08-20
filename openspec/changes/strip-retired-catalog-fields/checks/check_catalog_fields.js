#!/usr/bin/env node
/* PC 后台目录字段门禁。用法: node check_catalog_fields.js <repo-root>
 * 数据层（data/seed.js、data/api.js）无 DOM 依赖，在 Node vm 的 window 垫片下真实加载；
 * 页面层依赖 DOM，做静态断言。UI1 由浏览器实际运行补充，见 tasks.md。
 */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const fails = [];
const pending = [];
const check = (label, fn) => {
  try {
    const r = fn();
    if (r && typeof r.then === 'function') pending.push(r.catch(e => fails.push(`${label}: ${String(e.message).split('\n')[0]}`)));
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const RETIRED = ['tags', 'allergens', 'sold', 'stock'];
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sandbox = { window: {}, console, setTimeout, clearTimeout, Promise };
  sandbox.globalThis = sandbox;
  const c = vm.createContext(sandbox);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sandbox.window;
};

check('seed products carry no retired field', () => {
  const { Seed } = ctx();
  for (const p of Seed.MENU) {
    for (const f of RETIRED) assert.equal(Object.hasOwn(p, f), false, `${p.id}.${f} still seeded`);
    assert.equal(Object.hasOwn(p, 'status'), true, `${p.id} lost its sale status`);
  }
});

check('contract neither accepts nor produces a quantity', () => {
  const src = read('data/api.js');
  assert.doesNotMatch(src, /\bstock\b|库存/, 'api.js still handles a quantity');
  assert.doesNotMatch(src, /\btags\b|\ballergens\b/, 'api.js still seeds a retired field');
  const { Api } = ctx();
  assert.equal(typeof Api.setProductStatus, 'function', 'sale-status contract broken');
  assert.equal(typeof Api.saveProduct, 'function', 'saveProduct contract broken');
});

check('product page drops quantity and monthly sales columns', () => {
  const src = read('pages/products.js');
  assert.doesNotMatch(src, /库存|f-stock|r\.stock|low-stock/, 'products page still renders a quantity');
  assert.doesNotMatch(src, /'销量'|r\.sold/, 'products page still renders monthly sales');
  assert.match(src, /标记售罄|恢复售卖/, 'products page lost the sale-status control');
});

check('dashboard drops the low-stock todo', () => {
  const src = read('pages/dashboard.js');
  assert.doesNotMatch(src, /lowStock|库存告急/, 'dashboard still shows a low-stock todo');
  assert.match(src, /RANK/, 'dashboard lost the sales ranking');
});

check('sale status still works after the fields are gone', () => {
  const w = ctx();
  w.__store = { menu: JSON.parse(JSON.stringify(w.Seed.MENU)) };
  const id = w.__store.menu[0].id;
  return w.Api.setProductStatus(id, 'soldout')
    .then(() => { assert.equal(w.__store.menu[0].status, 'soldout'); return w.Api.setProductStatus(id, 'on'); })
    .then(() => assert.equal(w.__store.menu[0].status, 'on'));
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('CATALOG_FIELDS_GATE=FAIL'); process.exit(1); }
  console.log('CATALOG_FIELDS_GATE=PASS');
});
