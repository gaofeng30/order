#!/usr/bin/env node
/* PC 后台六态状态机门禁。用法: node check_order_lifecycle.js <repo-root> */
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
const SIX = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款'];
const RETIRED = ['待支付', '已支付待接单', '已取消', '待制作', '异常'];
const read = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const ctx = () => {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/seed.js', 'data/api.js']) vm.runInContext(read(rel), c, { filename: rel });
  return sb.window;
};

check('state machine exposes exactly the merchant transitions', () => {
  const { Api } = ctx();
  // vm 沙箱内的对象与本 realm 原型不同，deepEqual 会误报，改用序列化比较
  assert.equal(JSON.stringify(Api.NEXT), JSON.stringify({ 制作中: '待取餐', 待取餐: '已完成' }));
  assert.equal(Object.hasOwn(Api.NEXT, '已预约'), false, 'frontend must not advance 已预约');
  assert.equal(JSON.stringify(Api.LANES), JSON.stringify(['已预约', '制作中', '待取餐', '已完成', '已退款', '全部']));
  for (const s of RETIRED) assert.equal(Api.LANES.includes(s), false, `LANES still offers ${s}`);
});

check('undo is removed from the contract and every caller', () => {
  const { Api } = ctx();
  assert.equal(Object.hasOwn(Api, 'revertOrder'), false, 'contract still exports revertOrder');
  for (const rel of ['pages/orders.js', 'pages/dashboard.js']) {
    assert.doesNotMatch(read(rel), /revertOrder|onUndo/, `${rel} still offers an undo`);
  }
});

check('non-advanceable rows read naturally', () => {
  // 状态名本身以「已」开头，`该订单已${status}` 会拼出「该订单已已完成」。
  // 六态把 已预约/退款中/已退款 也纳入只读分支，放大了这处既有文案缺陷。
  assert.doesNotMatch(read('pages/orders.js'), /该订单已\$\{/, 'non-advanceable row text duplicates 已');
});

check('status tone covers the six states only', () => {
  const { Api } = ctx();
  assert.equal(Api.statusTone('退款中'), 'warn');
  assert.equal(Api.statusTone('已退款'), 'mute');
  for (const s of ['已预约', '制作中', '待取餐']) assert.equal(Api.statusTone(s), 'info', `${s} tone`);
});

check('seed orders use only the six states', () => {
  const { Seed } = ctx();
  for (const o of Seed.ADMIN_ORDERS) {
    assert.equal(SIX.includes(o.status), true, `${o.id} has retired status ${o.status}`);
  }
  assert.equal(Seed.ADMIN_ORDERS.some(o => o.status === '已预约'), true, 'no 已预约 order to populate the lane');
});

check('advancing walks forward only and stops at the terminal state', () => {
  const w = ctx();
  w.__store = { aOrders: JSON.parse(JSON.stringify(w.Seed.ADMIN_ORDERS)) };
  const making = w.__store.aOrders.find(o => o.status === '制作中');
  const reserved = w.__store.aOrders.find(o => o.status === '已预约');
  return w.Api.advanceOrder(making.id)
    .then(() => { assert.equal(making.status, '待取餐'); return w.Api.advanceOrder(making.id); })
    .then(() => { assert.equal(making.status, '已完成'); return w.Api.advanceOrder(making.id).then(
      () => { throw new Error('terminal order advanced further'); },
      () => {}); })
    .then(() => w.Api.advanceOrder(reserved.id).then(
      () => { throw new Error('已预约 advanced by the frontend'); },
      () => assert.equal(reserved.status, '已预约')));
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
  if (fails.length) { console.log(fails.map(f => `  ${f}`).join('\n')); console.log('ORDER_LIFECYCLE_GATE=FAIL'); process.exit(1); }
  console.log('ORDER_LIFECYCLE_GATE=PASS');
});
