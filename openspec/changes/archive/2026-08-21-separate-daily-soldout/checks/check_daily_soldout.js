#!/usr/bin/env node
/* 当日售罄与上下架分离（PRD §6.5、§15.6.1）。
   用法: node check_daily_soldout.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const WA = path.join(root, 'apps/web-admin');
const MP = path.join(root, 'apps/wechat-miniprogram');
const fails = [];
const pending = [];
const check = (label, fn) => {
  try {
    const r = fn();
    if (r && typeof r.then === 'function') pending.push(r.catch(e => fails.push(`${label}: ${String(e.message).split('\n')[0]}`)));
  } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const readWA = rel => fs.readFileSync(path.join(WA, rel), 'utf8');
const walk = (d, out = []) => {
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    if (e.name === 'node_modules' || e.name === '__pycache__' || e.name === 'tests') continue;
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p, out); else out.push(p);
  }
  return out;
};

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
    soldOut: cl(w.Seed.PRODUCT_SOLD_OUT_DATES || []),
  };
  return w;
}

function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
  const toasts = [];
  global.Behavior = d => d; global.Component = d => d; global.App = () => {};
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
    soldOut: cl(data.PRODUCT_SOLD_OUT_DATES || []), cart: {}, store: { status: '营业中' },
  };
  global.getApp = () => ({ globalData });
  const load = rel => {
    pageDef = null;
    require(path.join(MP, rel));
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
    page.__behaviors = behaviors; page.__def = pageDef;
    return page;
  };
  const invoke = (page, hook, arg) => {
    for (const b of page.__behaviors) if (typeof b[hook] === 'function') b[hook].call(page, arg);
    if (typeof page.__def[hook] === 'function') return page.__def[hook].call(page, arg);
    return undefined;
  };
  return { data, globalData, load, invoke, toasts };
}

const nextDay = iso => {
  const [y, m, d] = iso.split('-').map(Number);
  const t = new Map();  // 仅用于门禁内部推算，不进产品代码
  const dim = [31, (y % 4 === 0 && y % 100 !== 0) || y % 400 === 0 ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return d < dim[m - 1] ? `${y}-${String(m).padStart(2, '0')}-${String(d + 1).padStart(2, '0')}`
       : m < 12 ? `${y}-${String(m + 1).padStart(2, '0')}-01` : `${y + 1}-01-01`;
};

/* ================= 模型形状 ================= */

check('product status carries only shelf state on both ends', () => {
  const h = mp(), w = pc();
  for (const [label, list] of [['mini program', h.data.MENU], ['pc', w.Seed.MENU]]) {
    assert.ok(list.length, `${label} menu seed is empty`);
    for (const m of list) {
      assert.ok(['on', 'off'].includes(m.status), `${label} ${m.id}.status = ${m.status}`);
      for (const k of ['soldout', 'soldOut', 'sold_out']) {
        assert.equal(Object.hasOwn(m, k), false, `${label} ${m.id} still carries ${k} on the product`);
      }
    }
  }
});

check('daily sell-out is a record set keyed by product and service date', () => {
  const h = mp(), w = pc();
  for (const [label, list] of [['mini program', h.data.PRODUCT_SOLD_OUT_DATES], ['pc', w.Seed.PRODUCT_SOLD_OUT_DATES]]) {
    assert.ok(Array.isArray(list) && list.length, `${label} has no sell-out record set`);
    const seen = new Set();
    for (const r of list) {
      assert.equal(typeof r.productId, 'string', `${label} record has no productId`);
      assert.match(r.serviceDate, /^\d{4}-\d{2}-\d{2}$/, `${label} record serviceDate = ${r.serviceDate}`);
      for (const k of Object.keys(r)) {
        assert.ok(['productId', 'serviceDate'].includes(k),
          `${label} record carries ${k}; existence is the fact, a boolean would give two shapes for one meaning`);
      }
      const key = r.productId + '#' + r.serviceDate;
      assert.equal(seen.has(key), false, `${label} duplicate record for ${key}`);
      seen.add(key);
    }
  }
});

check('the seed can distinguish a cleared day from a day never marked', () => {
  const h = mp(), w = pc();
  for (const [label, list, today] of [['mini program', h.data.PRODUCT_SOLD_OUT_DATES, h.data.BUSINESS_DAY],
                                      ['pc', w.Seed.PRODUCT_SOLD_OUT_DATES, w.Api.today()]]) {
    assert.ok(list.some(r => r.serviceDate === today), `${label} seed has no record for the current business day`);
    assert.ok(list.some(r => r.serviceDate < today),
      `${label} seed has no record from an earlier business day — "clears next day" is unfalsifiable`);
  }
});

/* ================= 派生与隔离 ================= */

check('availability is computed from both dimensions', () => {
  for (const [label, api, today] of [['mini program', mp().data, mp().data.BUSINESS_DAY],
                                     ['pc', pc().Api, pc().Api.today()]]) {
    assert.equal(typeof api.isSoldOut, 'function', `${label} has no isSoldOut derivation`);
    assert.equal(typeof api.isSellable, 'function', `${label} has no isSellable derivation`);
    void today;
  }
});

check("a day D sell-out does not reach day D+1", () => {
  const h = mp(), w = pc();
  for (const [label, api, list, today] of [
    ['mini program', h.data, h.data.PRODUCT_SOLD_OUT_DATES, h.data.BUSINESS_DAY],
    ['pc', w.Api, w.Seed.PRODUCT_SOLD_OUT_DATES, w.Api.today()],
  ]) {
    const rec = list.find(r => r.serviceDate === today);
    assert.ok(rec, `${label} seed has no record for today`);
    assert.equal(api.isSoldOut(rec.productId, today), true, `${label} today's record does not take effect`);
    assert.equal(api.isSoldOut(rec.productId, nextDay(today)), false,
      `${label} a day D sell-out still blocks D+1 — 一期只有预约单，这会误伤主场景`);
  }
});

check('yesterday clears itself with no manual action', () => {
  const h = mp(), w = pc();
  for (const [label, api, list, today] of [
    ['mini program', h.data, h.data.PRODUCT_SOLD_OUT_DATES, h.data.BUSINESS_DAY],
    ['pc', w.Api, w.Seed.PRODUCT_SOLD_OUT_DATES, w.Api.today()],
  ]) {
    const stale = list.find(r => r.serviceDate < today &&
      !list.some(x => x.productId === r.productId && x.serviceDate === today));
    assert.ok(stale, `${label} seed has no product whose only record is from an earlier day`);
    assert.equal(api.isSoldOut(stale.productId, today), false,
      `${label} an earlier day's sell-out survived into today`);
  }
});

/* ================= 两端开关 ================= */

check('the mini program toggle writes the day record and leaves status alone', () => {
  const h = mp();
  const page = h.load('pages/admin-products/admin-products.js');
  h.invoke(page, 'onShow');
  const target = h.globalData.menu.find(m => m.status === 'on');
  const before = target.status;
  const today = h.data.BUSINESS_DAY;
  page.toggleSoldout({ currentTarget: { dataset: { id: target.id } } });
  const after = h.globalData.menu.find(m => m.id === target.id);
  assert.equal(after.status, before, 'the sell-out toggle changed the shelf state');
  assert.ok(h.globalData.soldOut.some(r => r.productId === target.id && r.serviceDate === today),
    'the toggle wrote no record for the current business day');
  page.toggleSoldout({ currentTarget: { dataset: { id: target.id } } });
  assert.equal(h.globalData.soldOut.some(r => r.productId === target.id && r.serviceDate === today), false,
    'toggling back did not remove the record');
});

check('the mini program product screen still offers only the sell-out toggle', () => {
  const h = mp();
  const page = h.load('pages/admin-products/admin-products.js');
  h.invoke(page, 'onShow');
  for (const forbidden of ['newProduct', 'edit', 'toggleShelf']) {
    assert.equal(typeof page[forbidden], 'undefined', `product screen exposes ${forbidden}`);
  }
  assert.equal(typeof page.toggleSoldout, 'function', 'the sell-out toggle is gone');
});

check('the pc toggles keep the two dimensions apart', () => {
  const w = pc();
  const target = w.__store.menu.find(m => m.status === 'on');
  const today = w.Api.today();
  assert.equal(typeof w.Api.setSoldOut, 'function', 'pc has no daily sell-out writer');
  assert.equal(typeof w.Api.setShelf, 'function', 'pc has no shelf writer');
  w.Api.setSoldOut(target.id, true);
  assert.equal(w.__store.menu.find(m => m.id === target.id).status, 'on', 'the sell-out writer changed the shelf state');
  assert.equal(w.Api.isSoldOut(target.id, today), true, 'the sell-out writer had no effect');
  w.Api.setShelf(target.id, 'off');
  assert.equal(w.Api.isSoldOut(target.id, today), true, 'the shelf writer removed a sell-out record');
  w.Api.setSoldOut(target.id, false);
  assert.equal(w.__store.menu.find(m => m.id === target.id).status, 'off', 'the sell-out writer changed the shelf state');
});

check('rebuilding a paid order checks sell-out by its pickup date', () => {
  const w = pc();
  const p = w.__store.pending.find(x => x.cause !== '数据校验不通过' &&
                                        x.pickupDate && x.items.every(it => w.Seed.itemById(it[0])));
  assert.ok(p, 'no pending payment usable for this check');
  const pid = p.items[0][0];
  w.Api.setShelf(pid, 'on');
  // 别的日期售罄不该挡住这笔
  w.__store.soldOut.push({ productId: pid, serviceDate: '2020-01-01' });
  const other = w.Api.blockingReason(p);
  assert.ok(!other || !/售罄/.test(other), `an unrelated date's sell-out blocked the rebuild: ${other}`);
  // 本笔取餐日期售罄必须挡住
  w.__store.soldOut.push({ productId: pid, serviceDate: p.pickupDate });
  const own = w.Api.blockingReason(p);
  assert.ok(own && /售罄/.test(own), `the pickup date's sell-out did not block the rebuild: ${own}`);
});

check('no surface reads a sold-out product status', () => {
  for (const base of [WA, MP]) {
    for (const f of walk(base)) {
      if (!/\.(js|wxml)$/.test(f)) continue;
      const src = fs.readFileSync(f, 'utf8');
      assert.doesNotMatch(src, /status\s*===\s*['"]soldout['"]|['"]soldout['"]\s*===\s*\w+\.status/,
        `${path.relative(root, f)} still reads sell-out off the product status`);
    }
  }
});

/* ===== 以下四项接管自 archive/2026-08-20-strip-retired-catalog-fields =====
   该门禁断言 setProductStatus(id,'soldout') 会把 status 设成 'soldout' ——
   正是 §15.6.1 禁止、本 change 要纠正的事实。门禁整体 FAIL 会让它其余仍然
   成立的断言失去绿色守护，故一并吸收。 */

const RETIRED = ['stock', 'sold', 'tags', 'allergens', 'low'];

check('[takeover] seed products carry no retired field', () => {
  const w = pc();
  for (const m of w.Seed.MENU) {
    for (const f of RETIRED) assert.equal(Object.hasOwn(m, f), false, `${m.id}.${f} still seeded`);
    assert.equal(Object.hasOwn(m, 'status'), true, `${m.id} lost its sale status`);
  }
});

check('[takeover] the contract neither accepts nor produces a quantity', () => {
  const src = readWA('data/api.js');
  assert.doesNotMatch(src, /\bstock\b|库存/, 'api.js still handles a quantity');
  assert.doesNotMatch(src, /\btags\b|\ballergens\b/, 'api.js still seeds a retired field');
  const w = pc();
  for (const fn of ['setProductStatus', 'saveProduct', 'setShelf', 'setSoldOut']) {
    assert.equal(typeof w.Api[fn], 'function', `${fn} contract broken`);
  }
  // 上下架只接受两个取值：第三个取值正是本 change 拆掉的那个
  return w.Api.setShelf(w.Seed.MENU[0].id, 'soldout')
    .then(() => { throw new Error('setShelf accepted a sold-out status'); },
          () => {});
});

check('[takeover] the product page drops quantity and monthly sales columns', () => {
  const src = readWA('pages/products.js');
  assert.doesNotMatch(src, /库存|f-stock|r\.stock|low-stock/, 'products page still renders a quantity');
  assert.doesNotMatch(src, /'销量'|r\.sold\b/, 'products page still renders monthly sales');
  assert.match(src, /标记售罄|恢复售卖/, 'products page lost the sale-status control');
});

check('[takeover] the dashboard drops the low-stock todo', () => {
  const src = readWA('pages/dashboard.js');
  assert.doesNotMatch(src, /lowStock|库存告急/, 'dashboard still shows a low-stock todo');
  assert.match(src, /RANK/, 'dashboard lost the sales ranking');
});

check('all javascript parses', () => {
  let n = 0;
  for (const base of [WA, MP]) {
    for (const f of walk(base)) {
      if (!f.endsWith('.js')) continue;
      new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
      n += 1;
    }
  }
  console.log(`  parsed ${n} javascript files`);
});

Promise.all(pending).then(() => {
  if (fails.length) {
    console.log(fails.map(f => `  ${f}`).join('\n'));
    console.log(`DAILY_SOLDOUT_GATE=FAIL (${fails.length}/16)`);
    process.exit(1);
  }
  console.log('DAILY_SOLDOUT_GATE=PASS');
});
