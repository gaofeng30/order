#!/usr/bin/env node
/* 门店标识单一来源；订单的取餐地点取自订单快照（PRD §3.1、§7.2、§13.3）。
   用法: node check_store_identity.js <repo-root> */
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

const NAME = '绥安食品';
const ADDR = '党政办公中心后院老食堂';
const POINT = '党政办公中心后院老食堂北门';
const RETIRED = ['县前直营店', '绥芬河市青云镇通商路', '县前大厦'];

function pc() {
  const sb = { window: {}, console, setTimeout, clearTimeout, Promise,
               DecompressionStream, TextDecoder, Response, Uint8Array, DataView, ArrayBuffer };
  sb.globalThis = sb;
  const c = vm.createContext(sb);
  for (const rel of ['data/xlsx.js', 'data/seed.js', 'data/api.js']) vm.runInContext(readWA(rel), c, { filename: rel });
  return sb.window;
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

check('the confirmed store name and address are in place on both ends', () => {
  const h = mp(), w = pc();
  for (const [label, store] of [['mini program', h.data.STORE], ['pc', w.Seed.STORE]]) {
    assert.equal(store.name, NAME, `${label} store name = ${store.name}`);
    assert.equal(store.addr, ADDR, `${label} store address = ${store.addr}`);
  }
});

check('no branch field survives', () => {
  const h = mp(), w = pc();
  for (const [label, store] of [['mini program', h.data.STORE], ['pc', w.Seed.STORE]]) {
    assert.equal(Object.hasOwn(store, 'branch'), false,
      `${label} still carries a branch field; a value identical to the store name makes writers pick again`);
  }
  for (const base of [WA, MP]) {
    for (const f of walk(base)) {
      if (!/\.(js|wxml|html)$/.test(f)) continue;
      assert.doesNotMatch(fs.readFileSync(f, 'utf8'), /\bSTORE\.branch\b|\bstore\.branch\b/,
        `${path.relative(root, f)} still reads a branch field`);
    }
  }
});

check('no retired demo identity text survives', () => {
  for (const base of [WA, MP]) {
    for (const f of walk(base)) {
      if (!/\.(js|wxml|html|json)$/.test(f)) continue;
      const src = fs.readFileSync(f, 'utf8');
      for (const bad of RETIRED) {
        assert.ok(!src.includes(bad), `${path.relative(root, f)} still carries the demo text ${bad}`);
      }
    }
  }
});

check('the merchant order detail reads the pickup point off the order', () => {
  const h = mp();
  const target = h.globalData.aOrders[0];
  const page = h.load('pages/admin-order-detail/admin-order-detail.js');
  h.invoke(page, 'onLoad', { id: target.id });
  assert.equal(page.data.o.pickupPoint, POINT, 'the order lost its pickup point snapshot');
  const wxml = fs.readFileSync(path.join(MP, 'pages/admin-order-detail/admin-order-detail.wxml'), 'utf8');
  const row = wxml.slice(wxml.indexOf('取餐地点'), wxml.indexOf('取餐地点') + 220);
  assert.match(row, /\{\{\s*o\.pickupPoint\s*\}\}/,
    'the merchant order detail does not render the order pickup snapshot');
  assert.doesNotMatch(row, /store\./,
    'the merchant order detail renders the pickup location from store configuration');
});

check('both ends show the same pickup location for the same order', () => {
  const h = mp();
  const user = h.load('pages/order-detail/order-detail.js');
  h.invoke(user, 'onLoad', { id: h.globalData.orders[0].id });
  const merchant = h.load('pages/admin-order-detail/admin-order-detail.js');
  h.invoke(merchant, 'onLoad', { id: h.globalData.aOrders[0].id });
  assert.equal(user.data.o.pickupPoint, merchant.data.o.pickupPoint,
    'user and merchant surfaces disagree on the pickup location');
});

check('the home screen reads the store name from configuration', () => {
  const h = mp();
  h.globalData.store.name = '测试门店名';
  const page = h.load('pages/home/home.js');
  h.invoke(page, 'onLoad', {});
  h.invoke(page, 'onShow');
  assert.equal(page.data.store.name, '测试门店名', 'the home screen does not read the store name');
  const wxml = fs.readFileSync(path.join(MP, 'pages/home/home.wxml'), 'utf8');
  assert.doesNotMatch(wxml, new RegExp(NAME), 'the home template still hardcodes the store name');
});

check('the PRD records the confirmed store identity', () => {
  const prd = fs.readFileSync(path.join(root, 'docs/product/online-ordering-system-prd-0818.md'), 'utf8');
  const sec = prd.slice(prd.indexOf('### 13.3'), prd.indexOf('## 14.'));
  assert.ok(sec.includes(ADDR), '13.3 does not record the confirmed address');
  assert.doesNotMatch(sec, /门店名称与门店地址[^|]*\|[^|]*\|[^|]*\|[^|]*演示值/,
    '13.3 still lists the store identity as pending');
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
  console.log(`STORE_IDENTITY_GATE=FAIL (${fails.length}/8)`);
  process.exit(1);
}
console.log('STORE_IDENTITY_GATE=PASS');
