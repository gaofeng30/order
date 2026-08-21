#!/usr/bin/env node
/* 订单项自带商品名称快照（PRD §15.6.2、§5.6）。
   用法: node check_item_name.js <repo-root>
   本门禁接管 archive/2026-08-21-align-order-model/checks/check_order_model.js
   的 PC 订单模型断言集 —— 见 proposal.md「与已归档门禁的冲突」。
   跨 vm realm 一律用 JSON.stringify 比较，不用 deepEqual。 */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const MP = path.join(root, 'apps/wechat-miniprogram');
const fails = [];
const check = (label, fn) => {
  try { fn(); } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const readWA = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const readMP = rel => fs.readFileSync(path.join(MP, rel), 'utf8');
const SIX = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款'];

/* ---- PC 侧宿主 ---- */
function pc() {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise,
               DecompressionStream, TextDecoder, Response, Uint8Array, DataView, ArrayBuffer };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/xlsx.js', 'data/seed.js', 'data/api.js']) vm.runInContext(readWA(rel), c, { filename: rel });
  const w = sb.window;
  const cl = v => JSON.parse(JSON.stringify(v));
  w.__store = {
    aOrders: cl(w.Seed.ADMIN_ORDERS), pending: cl(w.Seed.PENDING_PAYMENTS),
    menu: cl(w.Seed.MENU), accounts: cl(w.Seed.MERCHANT_ACCOUNTS),
    settings: cl(w.Seed.SETTINGS), store: { status: '营业中' }, voidedPending: [],
  };
  return w;
}

/* ---- 小程序侧宿主。归档产物必须自足，不复用 tests/page-harness.js ---- */
function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
  const toasts = [];
  global.Behavior = d => d;
  global.Component = d => d;
  global.App = () => {};
  global.Page = d => { pageDef = d; };
  global.wx = {
    getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, safeArea: { bottom: 778 } }),
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect: () => ({ top: 26, left: 278, height: 32 }),
    navigateTo() {}, redirectTo() {}, reLaunch() {}, navigateBack() {}, setClipboardData() {},
    request(r) { queueMicrotask(() => r.fail({ errMsg: 'offline' })); return { abort() {} }; },
  };
  const data = require(path.join(MP, 'utils/data.js'));
  const cl = v => JSON.parse(JSON.stringify(v));
  const globalData = {
    statusBarHeight: 20, navBarHeight: 44, navTotalHeight: 64, safeBottom: 0,
    aOrders: cl(data.ADMIN_ORDERS), orders: cl(data.USER_ORDERS), menu: cl(data.MENU),
    cart: {}, store: { status: '营业中' },
  };
  global.getApp = () => ({ globalData });
  const load = rel => {
    pageDef = null;
    require(path.join(MP, rel));
    if (!pageDef) throw new Error(`page not registered: ${rel}`);
    const behaviors = (pageDef.behaviors || []).filter(Boolean);
    const page = {
      data: Object.assign({}, ...behaviors.map(b => cl(b.data || {})), cl(pageDef.data || {})),
      setData(patch, cb) {
        for (const [k, v] of Object.entries(patch || {})) {
          const parts = k.split('.');
          let cur = this.data;
          for (let i = 0; i < parts.length - 1; i += 1) cur = (cur[parts[i]] ||= {});
          cur[parts[parts.length - 1]] = cl(v);
        }
        if (cb) cb.call(this);
      },
      selectComponent: () => ({ show: (m, o) => toasts.push({ message: m, config: o }) }),
      createSelectorQuery: () => {
        const q = { select: () => q, selectAll: () => q, boundingClientRect: () => q, exec: cb => cb([]) };
        return q;
      },
    };
    for (const b of behaviors) Object.assign(page, b.methods || {});
    for (const [k, v] of Object.entries(pageDef)) if (k !== 'data' && k !== 'behaviors') page[k] = v;
    page.__behaviors = behaviors;
    page.__def = pageDef;
    return page;
  };
  const invoke = (page, hook, arg) => {
    for (const b of page.__behaviors) if (typeof b[hook] === 'function') b[hook].call(page, arg);
    if (typeof page.__def[hook] === 'function') return page.__def[hook].call(page, arg);
    return undefined;
  };
  const util = require(path.join(MP, 'utils/util.js'));
  return { data, util, globalData, load, invoke, toasts };
}

// 服务端目录形态的商品：数字 id、price_cents。本地种子里没有这个 id。
const CATALOG_ITEM = { id: '1', category_id: '9', name: '目录来的双拼饭',
                       description: 'd', specification: 's', price_cents: 3200 };

const lines = o => o.items;
const assertLine = (line, where) => {
  assert.ok(Array.isArray(line) && line.length >= 5, `${where} line is not a 5+ tuple: ${JSON.stringify(line)}`);
  assert.equal(typeof line[1], 'string', `${where} line has no name at index 1: ${JSON.stringify(line)}`);
  assert.ok(line[1].trim().length > 0, `${where} line carries an empty name`);
  assert.equal(Number.isInteger(line[2]), true, `${where} qty is not an integer: ${JSON.stringify(line)}`);
  /* 单价与折后价只断言是有限数值。分为单位的整性由 [takeover] 的结算恒等式负责，
     而那条只在 PC 上成立 —— 小程序改分是紧随其后的另一个 change。 */
  for (const i of [3, 4]) {
    assert.equal(typeof line[i] === 'number' && Number.isFinite(line[i]), true,
      `${where} line index ${i} is not a finite number: ${JSON.stringify(line)}`);
  }
};

/* ================= 名称快照 ================= */

check('mini program order lines carry a name snapshot', () => {
  const h = mp();
  for (const [label, list] of [['user', h.data.USER_ORDERS], ['merchant', h.data.ADMIN_ORDERS]]) {
    assert.ok(list.length, `${label} order seed is empty`);
    for (const o of list) for (const line of lines(o)) assertLine(line, `${label} ${o.no}`);
  }
});

check('pc order lines carry a name snapshot', () => {
  const w = pc();
  for (const o of w.Seed.ADMIN_ORDERS) for (const line of lines(o)) assertLine(line, `pc ${o.no}`);
  for (const p of w.Seed.PENDING_PAYMENTS) for (const line of lines(p)) assertLine(line, `pending ${p.outTradeNo}`);
});

check('no render path resolves a product name by id', () => {
  /* 行为断言：给一个本地目录里查不到的 id 加上名称，摘要必须报出该名称。
     若实现仍在回查，这里要么抛错要么给出兜底文案。 */
  const GHOST = ['999', '目录外的菜', 2, 1000, 1000];
  const h = mp();
  const mpSummary = h.util.itemsSummary([GHOST]);
  assert.ok(mpSummary.includes('目录外的菜'),
    `mini program summary did not use the snapshot name: ${mpSummary}`);
  const w = pc();
  const pcSummary = w.Api.itemsSummary([GHOST]);
  assert.ok(pcSummary.includes('目录外的菜'),
    `pc summary did not use the snapshot name: ${pcSummary}`);
  /* 结构断言：摘要实现内不得再出现按 id 取商品的调用，兜底分支随之消失 */
  const mpBody = readMP('utils/util.js');
  const mpFn = mpBody.slice(mpBody.indexOf('function itemsSummary'), mpBody.indexOf('function itemsSummary') + 260);
  assert.doesNotMatch(mpFn, /itemById/, 'mini program itemsSummary still resolves the product by id');
  const pcBody = readWA('data/api.js');
  const pcFn = pcBody.slice(pcBody.indexOf('function itemsSummary'), pcBody.indexOf('function itemsSummary') + 260);
  assert.doesNotMatch(pcFn, /itemById/, 'pc itemsSummary still resolves the product by id');
});

/* ================= 下单后可打开 ================= */

check('a freshly placed order opens on both user surfaces', () => {
  const h = mp();
  h.util.cart.add(CATALOG_ITEM);
  const confirm = h.load('pages/confirm/confirm.js');
  h.invoke(confirm, 'onLoad');
  confirm.pay();
  const fresh = h.globalData.orders[0];
  assert.ok(fresh && fresh.items.length, 'checkout produced no order');
  assertLine(fresh.items[0], 'checkout');
  assert.ok(fresh.items[0][1].includes('目录'), 'the checkout order did not snapshot the catalog name');

  const list = h.load('pages/orders/orders.js');
  h.invoke(list, 'onShow');
  const row = list.data.list.find(o => o.id === fresh.id);
  assert.ok(row, 'the fresh order is missing from the order list');
  assert.ok(row.summary.includes('目录'), `order list summary lost the name: ${row.summary}`);

  const detail = h.load('pages/order-detail/order-detail.js');
  h.invoke(detail, 'onLoad', { id: fresh.id });
  assert.ok(detail.data.rows.length, 'order detail rendered no rows');
  assert.ok(detail.data.rows[0].name.includes('目录'), 'order detail lost the snapshot name');
});

check('checkout and seed agree on field types', () => {
  const h = mp();
  h.util.cart.add(CATALOG_ITEM);
  const confirm = h.load('pages/confirm/confirm.js');
  h.invoke(confirm, 'onLoad');
  confirm.pay();
  const fresh = h.globalData.orders[0];
  const seeded = h.data.USER_ORDERS[0];
  for (const k of ['total', 'code', 'status']) {
    assert.equal(typeof fresh[k], typeof seeded[k],
      `${k} is ${typeof fresh[k]} on a fresh order but ${typeof seeded[k]} on a seeded one`);
  }
});

/* ========== 以下五项接管自 archive/2026-08-21-align-order-model ========== */

check('[takeover] every pc order carries the settlement facts PRD 15.6.2 requires', () => {
  const w = pc();
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

check('[takeover] money is in cents and every pc order settles exactly on the new tuple', () => {
  const w = pc();
  for (const o of w.Seed.ADMIN_ORDERS) {
    for (const k of ['subtotal', 'discountCut', 'total']) {
      assert.equal(Number.isInteger(o[k]), true, `${o.no}.${k} = ${o[k]} is not an integer`);
    }
    assert.ok(o.total >= 1000, `${o.no}.total = ${o.total} looks like yuan, not cents`);
    assert.equal(o.subtotal - o.discountCut, o.total,
      `${o.no} does not settle: ${o.subtotal} - ${o.discountCut} != ${o.total}`);
    /* 具名解构而非裸下标：下一次元组变形时以「字段缺失」而不是「算错了」的形式失败 */
    let sub = 0, paid = 0;
    for (const [, , qty, price, discounted] of o.items) { sub += qty * price; paid += qty * discounted; }
    assert.equal(sub, o.subtotal, `${o.no} item subtotal ${sub} != ${o.subtotal}`);
    assert.equal(paid, o.total, `${o.no} item paid ${paid} != ${o.total}`);
  }
});

check('[takeover] the staff discount snapshot is consistent with isStaff', () => {
  const w = pc();
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.equal(Number.isInteger(o.discountRate), true, `${o.no}.discountRate not an integer`);
    assert.ok(o.discountRate >= 1 && o.discountRate <= 100, `${o.no}.discountRate = ${o.discountRate}`);
    if (o.isStaff) {
      assert.ok(o.discountCut > 0, `${o.no} is a staff order with no discount`);
      for (const [, , , price, discounted] of o.items) {
        assert.equal(discounted, Math.round(price * o.discountRate / 100),
          `${o.no} line discount is not rounded per unit`);
      }
    } else {
      assert.equal(o.discountRate, 100, `${o.no} is not a staff order but rate = ${o.discountRate}`);
      assert.equal(o.discountCut, 0, `${o.no} is not a staff order but cut = ${o.discountCut}`);
    }
  }
  assert.ok(w.Seed.ADMIN_ORDERS.some(o => o.isStaff), 'seed needs at least one staff-priced order');
});

check('[takeover] transaction id and refund record stay bidirectional', () => {
  const w = pc();
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.match(o.txnId, /^42000\d{13,}$/, `${o.no}.txnId = ${o.txnId} is not a wechat transaction id`);
    const settled = o.status === '退款中' || o.status === '已退款';
    assert.equal(!!o.refund, settled, `${o.no}.status = ${o.status} but refund record = ${!!o.refund}`);
  }
});

check('[takeover] no order carries a whole-order flavor', () => {
  const w = pc();
  for (const o of w.Seed.ADMIN_ORDERS) {
    assert.equal(Object.hasOwn(o, 'flavor'), false, `${o.no} still carries a whole-order flavor`);
  }
  for (const rel of ['pages/orders.js', 'pages/dashboard.js']) {
    assert.doesNotMatch(readWA(rel), /\bo\.flavor\b|\border\.flavor\b/, `${rel} still reads a whole-order flavor`);
  }
});

/* ================= PRD ================= */

check('the PRD defines the name column in 15.6.2', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const sec = prd.slice(prd.indexOf('### 15.6.2'), prd.indexOf('### 15.6.3'));
  const m = sec.match(/items:\s*\[([^\]]*)\]/);
  assert.ok(m, '15.6.2 no longer defines the items tuple');
  const cols = m[1].split(',').map(s => s.trim());
  assert.equal(cols[0], 'id', `items tuple starts with ${cols[0]}`);
  assert.equal(cols[1], 'name', `items tuple has ${cols[1]} at index 1, not name`);
  assert.ok(cols.length >= 5, `items tuple declares only ${cols.length} columns`);
});

check('all javascript parses', () => {
  let n = 0;
  for (const base of [WA, MP]) {
    (function walk(d) {
      for (const e of fs.readdirSync(d, { withFileTypes: true })) {
        if (e.name === 'node_modules' || e.name === '__pycache__') continue;
        const p = path.join(d, e.name);
        if (e.isDirectory()) walk(p);
        else if (e.name.endsWith('.js')) { new vm.Script(fs.readFileSync(p, 'utf8'), { filename: p }); n += 1; }
      }
    })(base);
  }
  console.log(`  parsed ${n} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`ITEM_NAME_GATE=FAIL (${fails.length}/12)`);
  process.exit(1);
}
console.log('ITEM_NAME_GATE=PASS');
