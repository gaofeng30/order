#!/usr/bin/env node
/* 单取餐点（PRD §3.1、§5.5、§7.2、§13.3）。
   用法: node check_pickup_point.js <repo-root> */
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
const walk = (d, out = []) => {
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    if (e.name === 'node_modules' || e.name === '__pycache__' || e.name === 'tests') continue;
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p, out); else out.push(p);
  }
  return out;
};

const EXPECTED = '党政办公中心后院老食堂北门';
/* 只清扫取餐点语境的演示文案。'县前直营店' 同时是 STORE.branch（门店名），
   客户这次只给了取餐地点，门店名另行索取，不在本 change 范围 —— 一刀切扫
   会逼我凭空替换一个更难被发现的虚构值。 */
const RETIRED_PICKUP = ['县前大厦'];

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
    settings: cl(w.Seed.SETTINGS), store: cl(w.Seed.STORE), voidedPending: [],
    soldOut: cl(w.Seed.PRODUCT_SOLD_OUT_DATES),
  };
  return w;
}

function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
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
    soldOut: cl(data.PRODUCT_SOLD_OUT_DATES), cart: {}, store: cl(data.STORE),
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
      selectComponent: () => ({ show() {} }),
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
  return { data, globalData, load, invoke };
}

const CATALOG_ITEM = { id: '1', category_id: '9', name: '目录来的双拼饭',
                       description: 'd', specification: 's', price_cents: 3200 };

check('both ends export the same pickup point constant', () => {
  const h = mp(), w = pc();
  assert.equal(h.data.PICKUP_POINT, EXPECTED, `mini program constant = ${h.data.PICKUP_POINT}`);
  assert.equal(w.Seed.PICKUP_POINT, EXPECTED, `pc constant = ${w.Seed.PICKUP_POINT}`);
});

check('there is exactly one pickup point and it carries no separate address', () => {
  const h = mp(), w = pc();
  for (const [label, list] of [['mini program', h.data.PICKUP_POINTS], ['pc', w.Seed.PICKUP_POINTS]]) {
    assert.equal(list.length, 1, `${label} declares ${list.length} pickup points`);
    assert.equal(list[0].name, EXPECTED, `${label} point name = ${list[0].name}`);
    assert.equal(Object.hasOwn(list[0], 'addr'), false,
      `${label} still splits the pickup point into name and address`);
  }
});

check('every seeded order snapshots the pickup point', () => {
  const h = mp(), w = pc();
  const sets = [['mp user', h.data.USER_ORDERS], ['mp merchant', h.data.ADMIN_ORDERS],
                ['pc', w.Seed.ADMIN_ORDERS], ['pc pending', w.Seed.PENDING_PAYMENTS]];
  for (const [label, list] of sets) {
    for (const o of list) {
      assert.equal(o.pickupPoint, EXPECTED, `${label} ${o.id || o.outTradeNo}.pickupPoint = ${o.pickupPoint}`);
    }
  }
});

check('checkout writes the same value the seed carries', () => {
  const h = mp();
  require(path.join(MP, 'utils/util.js')).cart.add(CATALOG_ITEM);
  const confirm = h.load('pages/confirm/confirm.js');
  h.invoke(confirm, 'onLoad');
  confirm.pay();
  const fresh = h.globalData.orders[0];
  assert.equal(fresh.pickupPoint, EXPECTED,
    `checkout wrote ${fresh.pickupPoint}, the seed carries ${EXPECTED}`);
  assert.equal(fresh.pickupPoint, h.data.USER_ORDERS[0].pickupPoint,
    'checkout and the seed disagree on what pickupPoint means');
});

check('the order detail shows the pickup location once', () => {
  const h = mp();
  const detail = h.load('pages/order-detail/order-detail.js');
  h.invoke(detail, 'onLoad', { id: h.globalData.orders[0].id });
  assert.equal(detail.data.o.pickupPoint, EXPECTED, 'the order detail lost the pickup point');
  assert.equal(Object.hasOwn(detail.data, 'ptAddr'), false,
    'the order detail still carries a separate pickup address');
  const wxml = fs.readFileSync(path.join(MP, 'pages/order-detail/order-detail.wxml'), 'utf8');
  const bindings = (wxml.match(/\{\{\s*(?:o\.)?pickup(?:Point|Window)|\{\{\s*ptAddr/g) || []).length;
  assert.equal(bindings, 1, `the template outputs the pickup location ${bindings} times`);
});

check('the pc settings page renders the constant and offers no choice', () => {
  const src = readWA('pages/settings.js');
  assert.doesNotMatch(src, /STORE\.pickupWindow/, 'settings still renders a separate pickup window field');
  assert.match(src, /Seed\.PICKUP_POINT/, 'settings does not render the shared constant');
  /* §3.1「不提供多点选择」：单点没有可选性，选择器本身就是多点模型的残留 */
  const fld = src.slice(src.indexOf('取餐地点'), src.indexOf('取餐地点') + 420);
  assert.doesNotMatch(fld, /<select/, 'settings still offers a pickup point selector');
  const w = pc();
  assert.equal(Object.hasOwn(w.Seed.STORE, 'pickupWindow'), false,
    'the store still keeps a duplicate pickup field alongside the constant');
});

check('no pickup point string is hardcoded outside the constant', () => {
  for (const base of [WA, MP]) {
    for (const f of walk(base)) {
      if (!/\.(js|wxml|json)$/.test(f)) continue;
      const rel = path.relative(root, f);
      const src = fs.readFileSync(f, 'utf8');
      for (const bad of RETIRED_PICKUP) {
        assert.ok(!src.includes(bad), `${rel} still carries the demo pickup text ${bad}`);
      }
      // pickupPoint 的每一次赋值都必须引用常量，不得写字面量
      for (const m of src.matchAll(/pickupPoint:\s*(['"][^'"]*['"])/g)) {
        assert.fail(`${rel} hardcodes a pickup point literal ${m[1]} instead of the shared constant`);
      }
    }
  }
});

check('the PRD records the confirmed pickup point', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const sec = prd.slice(prd.indexOf('### 13.3'), prd.indexOf('## 14.'));
  assert.ok(sec.includes(EXPECTED), '13.3 does not record the confirmed pickup point');
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

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`PICKUP_POINT_GATE=FAIL (${fails.length}/9)`);
  process.exit(1);
}
console.log('PICKUP_POINT_GATE=PASS');
