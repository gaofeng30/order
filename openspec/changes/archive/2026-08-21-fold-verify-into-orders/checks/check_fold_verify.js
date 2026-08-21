#!/usr/bin/env node
/* 删除 PC 扫码核销页、手工核销并入订单管理（PRD §15.5.3、§6.6、§6.7、§7.8）。
   用法: node check_fold_verify.js <repo-root>
   跨 vm realm 一律用 JSON.stringify 比较，不用 deepEqual。 */
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
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise,
               DecompressionStream, TextDecoder, Response, Uint8Array, DataView, ArrayBuffer };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/xlsx.js', 'data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  const w = sb.window;
  w.__store = {
    aOrders: JSON.parse(JSON.stringify(w.Seed.ADMIN_ORDERS)),
    pending: JSON.parse(JSON.stringify(w.Seed.PENDING_PAYMENTS)),
    menu: JSON.parse(JSON.stringify(w.Seed.MENU)),
    accounts: JSON.parse(JSON.stringify(w.Seed.MERCHANT_ACCOUNTS)),
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};

check('the standalone verify page is gone from every surface', () => {
  /* §15.5.3：评审 §23「扫码是在手机上扫」，PC 侧该页删除。
     断言的是能力不存在，不是「扫码核销」四个字不出现 —— 页面上说明手工核销
     去了哪里、以及扫码在手机端进行，都是正当的。 */
  assert.equal(fs.existsSync(path.join(WA, 'pages/verify.js')), false, 'pages/verify.js still exists');
  assert.doesNotMatch(read('index.html'), /pages\/verify\.js/, 'index.html still loads the verify page');
  assert.doesNotMatch(read('app.js'), /r:\s*'verify'/, 'sidebar still routes to verify');
  const files = fs.readdirSync(path.join(WA, 'pages'));
  assert.equal(files.includes('verify.js'), false, 'a verify page is still registered');
});

check('the PC page set matches the PRD page list exactly', () => {
  const app = read('app.js');
  const routes = [...app.matchAll(/\{\s*r:\s*'([a-z-]+)'/g)].map(m => m[1]);
  const expect = ['dashboard', 'orders', 'finance', 'pending', 'products', 'product-import',
                  'categories', 'staff', 'staff-import', 'accounts', 'settings', 'layer'];
  assert.equal(JSON.stringify(routes.slice().sort()), JSON.stringify(expect.slice().sort()),
    `routes = ${JSON.stringify(routes)}`);
  /* §3.5 写的是 12 页，两边必须对上 */
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const line = prd.split('\n').find(l => l.includes('PC 网页后台（') && l.includes('页）'));
  assert.ok(line, 'PRD has no PC page list');
  const n = Number(line.match(/PC 网页后台（(\d+)\s*页）/)[1]);
  assert.equal(n, routes.length, `PRD says ${n} pages, the sidebar has ${routes.length}`);
});

check('the order list is searchable by pickup code, order number and phone', () => {
  const w = ctx();
  assert.equal(typeof w.Api.searchOrders, 'function', 'contract missing searchOrders');
  const o = w.__store.aOrders.find(x => x.status === '待取餐' && x.pickupDate === w.Api.today());
  assert.ok(o, 'seed has no 待取餐 order on the business day');
  return Promise.all([
    w.Api.searchOrders(o.code),
    w.Api.searchOrders(o.no),
    w.Api.searchOrders(o.phone.slice(-4)),
  ]).then(([byCode, byNo, byPhone]) => {
    assert.equal(byCode.some(x => x.id === o.id), true, 'search by pickup code missed the order');
    assert.equal(byNo.some(x => x.id === o.id), true, 'search by order number missed the order');
    assert.equal(byPhone.some(x => x.id === o.id), true, 'search by phone missed the order');
  });
});

check('a four-digit code only matches the current business day', () => {
  const w = ctx();
  /* §6.6：手工输入只匹配当前营业日期的订单。§7.8：跨营业日的取餐号可能重复 ——
     不限定营业日的话，一个 4 位数字会同时命中两天的单，核销就核错了。 */
  const stale = w.__store.aOrders.find(x => x.pickupDate < w.Api.today());
  assert.ok(stale, 'seed has no order from an earlier business day');
  return w.Api.searchOrders(stale.code).then(list => {
    assert.equal(list.some(x => x.id === stale.id), false,
      `code ${stale.code} matched an order from ${stale.pickupDate}`);
    /* 4 位数字也可能是手机尾号，那部分不受营业日限制；受限的是「以取餐号命中」这一半 */
    for (const hit of list.filter(o => o.code === stale.code)) {
      assert.equal(hit.pickupDate, w.Api.today(), `a same-code order from ${hit.pickupDate} was returned`);
    }
    /* 但按订单号仍应找得到 —— 订单号全局唯一，没有跨日歧义 */
    return w.Api.searchOrders(stale.no).then(byNo =>
      assert.equal(byNo.some(x => x.id === stale.id), true, 'order number search cannot reach an earlier day'));
  });
});

check('a code that only exists on an earlier day says so instead of coming up empty', () => {
  const w = ctx();
  /* 搜不到就以为没有，是这条规则最容易造成的误判。契约必须把「该取餐号在其他
     营业日存在」这个事实报出来，并指出可用的定位方式。 */
  assert.equal(typeof w.Api.codeHint, 'function', 'contract missing codeHint');
  const stale = w.__store.aOrders.find(x => x.pickupDate < w.Api.today());
  const hint = w.Api.codeHint(stale.code);
  assert.ok(hint && hint.length > 0, `no hint for a code that exists only on ${stale.pickupDate}`);
  assert.ok(hint.includes(stale.pickupDate), 'the hint does not name the business day it was found on');
  const live = w.__store.aOrders.find(x => x.pickupDate === w.Api.today());
  assert.equal(w.Api.codeHint(live.code), '', 'a same-day code should need no hint');
});

check('searching does not become a seventh lane or a state', () => {
  const w = ctx();
  assert.equal(w.Api.LANES.includes('搜索'), false, 'search leaked into the lane set');
  assert.equal(Object.hasOwn(w.Api.ACT, '搜索'), false, 'search leaked into the state set');
  /* 搜索是跨泳道的：核销时并不知道单子在哪个状态 */
  return w.Api.searchOrders('1').then(list => {
    const states = new Set(list.map(o => o.status));
    assert.ok(states.size >= 2, `search spans only ${states.size} state(s); it must cross lanes`);
  });
});

check('the orders page wires the search and keeps the verify action', () => {
  const src = read('pages/orders.js');
  assert.match(src, /Api\.searchOrders\(/, 'orders page never calls searchOrders');
  assert.match(src, /Api\.codeHint\(/, 'orders page ignores the cross-day code hint');
  /* 核销仍是 待取餐 → 已完成 的唯一路径（§6.6），入口留在订单详情 */
  assert.match(src, /Api\.advanceOrder\(/, 'orders page lost the advance/verify action');
  assert.doesNotMatch(src, /revertOrder|onUndo/, 'orders page offers an undo');
});

check('the manual verify path still refuses a refunded order', () => {
  const w = ctx();
  /* §6.6：已退款订单不得核销。这条能力原本靠 verify 页把关，
     并入订单管理后必须由契约层承担，否则删页就是删了校验。 */
  return Promise.all(['退款中', '已退款'].map(st => {
    const o = w.__store.aOrders.find(x => x.status === st);
    assert.ok(o, `seed has no ${st} order`);
    return w.Api.advanceOrder(o.id).then(
      () => { throw new Error(`a ${st} order was advanced`); },
      () => assert.equal(w.__store.aOrders.find(x => x.id === o.id).status, st, `${st} order changed state`));
  }));
});

check('verifying is idempotent and never double-counts', () => {
  const w = ctx();
  /* §6.6：重复核销返回第一次结果，不重复计算营收或订单量 */
  const o = w.__store.aOrders.find(x => x.status === '待取餐');
  return w.Api.advanceOrder(o.id).then(first => {
    assert.equal(first.next, '已完成', `verify moved it to ${first.next}`);
    return w.Api.advanceOrder(o.id).then(
      () => { throw new Error('a completed order was advanced again'); },
      () => assert.equal(w.__store.aOrders.find(x => x.id === o.id).status, '已完成', 'state drifted on retry'));
  });
});

check('the PRD records the deletion and where manual verify went', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const row = prd.split('\n').find(l => l.includes('扫码核销') && l.includes('删除'));
  assert.ok(row, '§15.5.3 no longer records the deletion');
  const gaps = prd.slice(prd.indexOf('## 16.5'), prd.indexOf('## 16.6'));
  assert.doesNotMatch(gaps, /PC.{0,10}扫码核销.{0,20}未建/, 'the gap list still expects a PC verify page');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('FOLD_VERIFY_GATE=FAIL'); process.exit(1); }
  console.log('FOLD_VERIFY_GATE=PASS');
});
