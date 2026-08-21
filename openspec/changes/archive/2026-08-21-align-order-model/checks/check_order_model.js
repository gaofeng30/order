#!/usr/bin/env node
/* PC 订单模型对齐门禁（PRD §15.6.2）。用法: node check_order_model.js <repo-root>
   注意：跨 vm realm 的对象不能用 assert.deepEqual（结构相同但引用不同），一律用 JSON.stringify 比较。 */
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
    settings: JSON.parse(JSON.stringify(w.Seed.SETTINGS)),
    store: { status: '营业中' },
  };
  return w;
};

const SIX = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款'];

check('every order carries the settlement facts PRD 15.6.2 requires', () => {
  const w = ctx();
  const list = w.Seed.ADMIN_ORDERS;
  assert.ok(Array.isArray(list) && list.length > 0, 'no order seed');
  for (const o of list) {
    for (const k of ['id', 'no', 'code', 'status', 'pickupDate', 'pickupTime', 'mealPeriod',
                     'pickupPoint', 'paidAt', 'txnId', 'subtotal', 'discountRate', 'discountCut',
                     'total', 'isStaff', 'contact', 'phone', 'items']) {
      assert.equal(Object.hasOwn(o, k), true, `${o.no} missing ${k}`);
    }
    assert.equal(SIX.includes(o.status), true, `${o.no}.status = ${o.status}`);
    assert.equal(['lunch', 'dinner'].includes(o.mealPeriod), true, `${o.no}.mealPeriod = ${o.mealPeriod}`);
    assert.match(o.pickupDate, /^\d{4}-\d{2}-\d{2}$/, `${o.no}.pickupDate = ${o.pickupDate}`);
    assert.match(o.pickupTime, /^\d{2}:\d{2}$/, `${o.no}.pickupTime = ${o.pickupTime}`);
    assert.match(o.paidAt, /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/, `${o.no}.paidAt = ${o.paidAt}`);
    assert.equal(typeof o.isStaff, 'boolean', `${o.no}.isStaff not boolean`);
  }
});

check('money is stored in cents and every order settles exactly', () => {
  const w = ctx();
  for (const o of w.Seed.ADMIN_ORDERS) {
    for (const k of ['subtotal', 'discountCut', 'total']) {
      assert.equal(Number.isInteger(o[k]), true, `${o.no}.${k} = ${o[k]} is not an integer`);
    }
    /* 分为单位 —— 元为单位时 total 会是 26~76 这样的两位数 */
    assert.ok(o.total >= 1000, `${o.no}.total = ${o.total} looks like yuan, not cents`);
    assert.equal(o.subtotal - o.discountCut, o.total, `${o.no} does not settle: ${o.subtotal} - ${o.discountCut} != ${o.total}`);
    /* items 行的小计必须等于 subtotal，且逐行折后价之和等于 total */
    const sub = o.items.reduce((s, it) => s + it[1] * it[2], 0);
    const paid = o.items.reduce((s, it) => s + it[1] * it[3], 0);
    assert.equal(sub, o.subtotal, `${o.no} item subtotal ${sub} != ${o.subtotal}`);
    assert.equal(paid, o.total, `${o.no} item paid ${paid} != ${o.total}`);
  }
});

check('the staff discount snapshot is consistent with isStaff', () => {
  const w = ctx();
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.equal(Number.isInteger(o.discountRate), true, `${o.no}.discountRate not an integer`);
    assert.ok(o.discountRate >= 1 && o.discountRate <= 100, `${o.no}.discountRate = ${o.discountRate}`);
    if (o.isStaff) assert.ok(o.discountCut > 0, `${o.no} is a staff order with no discount`);
    else {
      assert.equal(o.discountRate, 100, `${o.no} is not a staff order but rate = ${o.discountRate}`);
      assert.equal(o.discountCut, 0, `${o.no} is not a staff order but cut = ${o.discountCut}`);
    }
  }
  assert.ok(w.Seed.ADMIN_ORDERS.some(o => o.isStaff), 'seed needs at least one staff-priced order');
});

check('the wechat transaction id is present exactly when the order was paid', () => {
  const w = ctx();
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.match(o.txnId, /^42000\d{13,}$/, `${o.no}.txnId = ${o.txnId} is not a wechat transaction id`);
  }
  const ids = new Set(w.Seed.ADMIN_ORDERS.map(o => o.txnId));
  assert.equal(ids.size, w.Seed.ADMIN_ORDERS.length, 'transaction ids are not unique');
});

check('refunded orders carry a refund record and others do not', () => {
  const w = ctx();
  for (const o of w.Seed.ADMIN_ORDERS) {
    const refunding = o.status === '退款中' || o.status === '已退款';
    assert.equal(Object.hasOwn(o, 'refund') && o.refund != null, refunding,
      `${o.no} status=${o.status} but refund=${JSON.stringify(o.refund)}`);
    if (!refunding) continue;
    for (const k of ['no', 'amount', 'status', 'operator', 'at', 'reason']) {
      assert.equal(Object.hasOwn(o.refund, k), true, `${o.no}.refund missing ${k}`);
    }
    assert.match(o.refund.no, /^5000\d{13,}$/, `${o.no}.refund.no = ${o.refund.no}`);
    assert.equal(Number.isInteger(o.refund.amount), true, `${o.no}.refund.amount not an integer`);
    assert.ok(o.refund.amount > 0 && o.refund.amount <= o.total,
      `${o.no}.refund.amount = ${o.refund.amount} out of range (total ${o.total})`);
    assert.equal(o.refund.status === (o.status === '已退款' ? '已退款' : '退款中'), true,
      `${o.no}.refund.status = ${o.refund.status} disagrees with order status ${o.status}`);
    assert.ok(String(o.refund.operator).length > 0, `${o.no}.refund.operator empty`);
  }
  const st = w.Seed.ADMIN_ORDERS.filter(o => o.status === '退款中').length;
  const done = w.Seed.ADMIN_ORDERS.filter(o => o.status === '已退款').length;
  assert.ok(st > 0 && done > 0, `seed needs both 退款中 (${st}) and 已退款 (${done}) orders`);
});

check('the order carries no whole-order flavor', () => {
  const w = ctx();
  /* PRD §15.6.2：口味与备注绑定在 items 行内，整单级只有 orderNote */
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.equal(Object.hasOwn(o, 'flavor'), false, `${o.no} still carries a whole-order flavor`);
  }
  for (const rel of ['pages/orders.js', 'pages/dashboard.js']) {
    assert.doesNotMatch(read(rel), /\bo\.flavor\b|\border\.flavor\b/, `${rel} still reads a whole-order flavor`);
  }
});

check('the contract formats cents and never leaks raw cents into a page', () => {
  const w = ctx();
  assert.equal(typeof w.Api.yuan, 'function', 'contract missing yuan(cents)');
  assert.equal(w.Api.yuan(0), '0.00');
  assert.equal(w.Api.yuan(5), '0.05');
  assert.equal(w.Api.yuan(2600), '26.00');
  assert.equal(w.Api.yuan(123456), '1234.56');
  /* 页面 MUST NOT 自己做分转元的算术 */
  for (const rel of ['pages/orders.js', 'pages/dashboard.js']) {
    assert.doesNotMatch(read(rel), /\/\s*100\b/, `${rel} converts cents to yuan by hand`);
  }
});

check('every page rendering an order total goes through the formatter', () => {
  const src = read('pages/orders.js');
  const hits = [...src.matchAll(/T\.money\(([^)]*)\)/g)].map(m => m[1].trim());
  assert.ok(hits.length > 0, 'orders.js renders no money at all');
  for (const arg of hits) {
    assert.match(arg, /Api\.yuan\(/, `T.money(${arg}) is not fed through Api.yuan`);
  }
});

check('the order lifecycle still exposes only the six states', () => {
  const w = ctx();
  /* ACT 覆盖全部六态（终态也有「查看」），是权威的状态集合 */
  assert.equal(JSON.stringify(Object.keys(w.Api.ACT).sort()), JSON.stringify([...SIX].sort()), 'state set drifted');
  assert.equal(JSON.stringify(w.Api.NEXT), JSON.stringify({ 制作中: '待取餐', 待取餐: '已完成' }), 'advance graph drifted');
});

/* 本项接管 archive/2026-08-20-adopt-six-state-order-lifecycle 的
   'state machine exposes exactly the merchant transitions'。
   那条把 LANES 钉死为五态（缺 退款中），与 PRD §15.5.3「六态泳道」不符 ——
   是那个 change 少给了一格，不是本 change 越界。原断言中除泳道集合外的部分
   （NEXT 图、已预约 不可推进、废弃状态不得出现）在此原样保留。 */
const RETIRED = ['待支付', '已支付待接单', '已取消', '待制作', '异常'];

check('every one of the six states has its own lane', () => {
  const w = ctx();
  assert.equal(JSON.stringify(w.Api.LANES), JSON.stringify([...SIX, '全部']), 'lane set is not the six states plus 全部');
  assert.equal(Object.hasOwn(w.Api.NEXT, '已预约'), false, 'frontend must not advance 已预约');
  for (const st of RETIRED) assert.equal(w.Api.LANES.includes(st), false, `LANES still offers ${st}`);
  const counts = w.Api.laneCounts();
  for (const l of SIX) {
    assert.equal(counts[l], w.Seed.ADMIN_ORDERS.filter(o => o.status === l).length, `lane ${l} miscounts`);
    assert.ok(counts[l] > 0, `lane ${l} has no seed order to exercise it`);
  }
  assert.equal(counts['全部'], w.Seed.ADMIN_ORDERS.length, 'lane 全部 miscounts');
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('ORDER_MODEL_GATE=FAIL'); process.exit(1); }
  console.log('ORDER_MODEL_GATE=PASS');
});
