#!/usr/bin/env node
/* web-admin 排除能力一致性门禁。用法: node check_admin_scope.js <repo-root>
 *
 * 数据层（data/seed.js、data/api.js）为无 DOM 依赖的纯 JS，在 Node 的 window 垫片下
 * 真实加载并断言导出，属运行态证据。页面、导航与 index.html 依赖浏览器 DOM，
 * 仓库内无浏览器 runner，只做静态断言——该边界在 tasks.md 中记为 BLOCKED_EXTERNAL。
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
    if (r && typeof r.then === 'function') {
      pending.push(r.catch(e => { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }));
    }
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};

const EXCLUDED_PAGES = ['levels.js', 'members.js', 'member-import.js', 'coupons.js'];
const EXCLUDED_SEEDS = ['LEVELS', 'MEMBERS', 'COUPONS', 'MY_COUPON_USED'];
const EXCLUDED_API = /level|member|coupon/i;
const EXCLUDED_STORE = ['levels', 'members', 'coupons', 'couponUsed'];
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');

// ---- 静态：页面文件与 index.html 脚本标签 ----
check('page files absent', () => {
  for (const f of EXCLUDED_PAGES) {
    assert.equal(fs.existsSync(path.join(WA, 'pages', f)), false, `pages/${f} still exists`);
  }
});
check('index.html drops script tags', () => {
  const html = read('index.html');
  for (const f of EXCLUDED_PAGES) {
    assert.equal(html.includes(`pages/${f}`), false, `index.html still loads pages/${f}`);
  }
});

// ---- 静态：导航与内存态初始化 ----
check('nav drops excluded group', () => {
  const app = read('app.js');
  assert.equal(/会员与营销/.test(app), false, 'app.js still declares the 会员与营销 nav group');
  assert.equal(/\bp2\b/.test(app), false, 'app.js still carries the p2 second-phase flag');
  for (const key of EXCLUDED_STORE) {
    assert.equal(new RegExp(`\\b${key}\\s*:`).test(app), false, `__store still initializes ${key}`);
  }
});

// ---- 静态：残留引用 ----
check('modules drop coupon references', () => {
  for (const rel of ['pages/products.js', 'ui/drawer.js', 'pages/dashboard.js', 'pages/orders.js']) {
    if (!fs.existsSync(path.join(WA, rel))) continue;
    assert.equal(EXCLUDED_API.test(read(rel)), false, `${rel} still references an excluded capability`);
  }
});

// ---- 运行态：数据层在 window 垫片下真实加载 ----
check('seed and api runtime exports', () => {
  const sandbox = { window: {}, console, setTimeout, clearTimeout, Promise };
  sandbox.globalThis = sandbox;
  const ctx = vm.createContext(sandbox);
  for (const rel of ['data/seed.js', 'data/api.js']) {
    vm.runInContext(read(rel), ctx, { filename: rel });
  }
  const { Seed, Api } = sandbox.window;
  assert.ok(Seed, 'window.Seed missing');
  assert.ok(Api, 'window.Api missing');
  for (const s of EXCLUDED_SEEDS) {
    assert.equal(Object.hasOwn(Seed, s), false, `Seed.${s} still exported`);
  }
  const leaked = Object.keys(Api).filter(n => EXCLUDED_API.test(n));
  assert.deepEqual(leaked, [], `Api still exports ${leaked.join(', ')}`);
  // 菜品与订单契约必须完好，删除不得连带打断
  for (const kept of ['listProducts', 'deleteProduct', 'listOrders', 'listCategories', 'getSettings']) {
    assert.equal(typeof Api[kept], 'function', `Api.${kept} broken by removal`);
  }
});

// ---- 运行态：删除菜品不再返回摘券结果 ----
check('deleteProduct drops coupon side effect', () => {
  const sandbox = { window: {}, console, setTimeout, clearTimeout, Promise };
  sandbox.globalThis = sandbox;
  const ctx = vm.createContext(sandbox);
  for (const rel of ['data/seed.js', 'data/api.js']) {
    vm.runInContext(read(rel), ctx, { filename: rel });
  }
  const { Seed, Api } = sandbox.window;
  sandbox.window.__store = { menu: JSON.parse(JSON.stringify(Seed.MENU)) };
  const id = sandbox.window.__store.menu[0].id;
  const before = sandbox.window.__store.menu.length;
  return Api.deleteProduct(id).then(r => {
    assert.equal(sandbox.window.__store.menu.length, before - 1, 'product not removed');
    assert.equal(Object.hasOwn(r || {}, 'disabledCoupons'), false, 'deleteProduct still returns disabledCoupons');
  });
});

// ---- 静态：注释与文案不得再提及排除能力 ----
check('no stale prose about excluded capability', () => {
  const RESIDUE = ['会员等级', '会员名单', '优惠券', '二期能力', 'MY_COUPON_USED', 'LEVELS', 'MEMBERS', 'COUPONS'];
  const hits = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) { walk(p); continue; }
      if (!/\.(js|html|css)$/.test(e.name)) continue;
      const text = fs.readFileSync(p, 'utf8');
      for (const term of RESIDUE) {
        if (text.includes(term)) hits.push(`${path.relative(WA, p)} :: ${term}`);
      }
    }
  })(WA);
  assert.deepEqual(hits, [], `stale prose: ${hits.join(' | ')}`);
});

// ---- 全部 JS 可解析 ----
check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p);
      else if (e.name.endsWith('.js')) files.push(p);
    }
  })(WA);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

Promise.all(pending).then(() => {
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('ADMIN_SCOPE_GATE=FAIL'); process.exit(1); }
  console.log('ADMIN_SCOPE_GATE=PASS');
});
