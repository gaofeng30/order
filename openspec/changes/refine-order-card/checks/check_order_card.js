#!/usr/bin/env node
/* 订单卡片展示与订单项数值聚合（PRD §5.8）。
   用法: node check_order_card.js <repo-root> */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = process.argv[2];
const MP = path.join(root, 'apps/wechat-miniprogram');
const fails = [];
const check = (label, fn) => {
  try { fn(); } catch (e) { fails.push(`${label}: ${String(e.message).split('\n')[0]}`); }
};
const read = rel => fs.readFileSync(path.join(MP, rel), 'utf8');
const walk = (d, out = []) => {
  for (const e of fs.readdirSync(d, { withFileTypes: true })) {
    if (e.name === 'node_modules' || e.name === '__pycache__' || e.name === 'tests') continue;
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p, out); else out.push(p);
  }
  return out;
};

function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
  const navs = [];
  global.Behavior = d => d; global.Component = d => d; global.App = () => {};
  global.Page = d => { pageDef = d; };
  global.wx = {
    getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, safeArea: { bottom: 778 } }),
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect: () => ({ top: 26, left: 278, height: 32 }),
    navigateTo(r) { navs.push(r.url); }, redirectTo(r) { navs.push(r.url); },
    reLaunch(r) { navs.push(r.url); }, navigateBack() {}, setClipboardData() {},
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
  return { data, globalData, load, invoke, navs };
}
const openOrders = h => { const p = h.load('pages/orders/orders.js'); h.invoke(p, 'onLoad', {}); h.invoke(p, 'onShow'); return p; };

/* ================= 件数已删除 ================= */

check('the order card renders no piece count', () => {
  const wxml = read('pages/orders/orders.wxml');
  assert.doesNotMatch(wxml, /共\s*\{\{[^}]*\}\}\s*件/, 'the card still renders a piece count');
  assert.doesNotMatch(wxml, /itemCount/, 'the card template still references a piece count field');
  assert.doesNotMatch(read('pages/orders/orders.wxss'), /\.oc-count\b/, 'the piece-count style survived');
});

check('the page state carries no piece count field', () => {
  const h = mp();
  const page = openOrders(h);
  assert.ok(page.data.list.length, 'the order list is empty');
  for (const row of page.data.list) {
    assert.equal(Object.hasOwn(row, 'itemCount'), false, `${row.id} still carries itemCount`);
  }
});

/* ================= 聚合必须选数值列 ================= */

check('no aggregation over items selects the id or the name column', () => {
  /* 缺陷成因：名称插入 index 1 后，所有基于下标的聚合都要跟着右移，
     当时改对了两处中的一处。这里锁的是这一类写法，不是某个字段。 */
  const RE = /items\s*\.\s*reduce\s*\(([\s\S]{0,160}?)\)\s*;/g;
  for (const f of walk(MP)) {
    if (!f.endsWith('.js')) continue;
    const rel = path.relative(MP, f);
    const src = fs.readFileSync(f, 'utf8');
    for (const m of src.matchAll(RE)) {
      for (const idx of m[1].matchAll(/\[\s*(\d+)\s*\]/g)) {
        assert.ok(['2', '3', '4'].includes(idx[1]),
          `${rel} aggregates items by column ${idx[1]}; only 2 (qty) / 3 (price) / 4 (discounted) are numeric`);
      }
    }
  }
});

check('every derived number on the card is actually a number', () => {
  const h = mp();
  const page = openOrders(h);
  for (const row of page.data.list) {
    assert.equal(typeof row.total, 'number', `${row.id}.total is ${typeof row.total}`);
    for (const [k, v] of Object.entries(row)) {
      /* 缺陷的实际形态：数字与商品名被 + 拼在一起。日期 2026-08-21、
         时刻 17:00 都以数字开头，用「数字紧跟中文」才指向真正的失败模式。 */
      assert.ok(!(typeof v === 'string' && /^\d+[\u4e00-\u9fa5]/.test(v)),
        `${row.id}.${k} looks like a number concatenated with a product name: ${v}`);
    }
  }
});

/* ================= 徽章与按钮 ================= */

check('the pickup code badge shows the number alone', () => {
  const wxml = read('pages/orders/orders.wxml');
  const badge = wxml.slice(wxml.indexOf('oc-code'), wxml.indexOf('oc-code') + 320);
  assert.doesNotMatch(badge, /oc-code-lbl/, 'the badge still renders a label element');
  assert.doesNotMatch(badge, />\s*号\s*</, 'the badge still renders a 号 label');
  assert.match(badge, /\{\{\s*item\.code\s*\}\}/, 'the badge lost the pickup code');
  assert.doesNotMatch(read('pages/orders/orders.wxss'), /\.oc-code-lbl\b/, 'the label style survived');
});

check('the badge centres its content', () => {
  const wxss = read('pages/orders/orders.wxss');
  const rule = wxss.slice(wxss.indexOf('.oc-code {'), wxss.indexOf('}', wxss.indexOf('.oc-code {')));
  assert.match(rule, /align-items:\s*center/, 'the badge does not centre horizontally');
  assert.match(rule, /justify-content:\s*center/, 'the badge does not centre vertically');
});

check('the total group stays right aligned after the count is gone', () => {
  /* 件数原本是行首元素，「合计」靠 margin-left:auto 把整组推到右边。
     删件数时若把 auto 一并删掉，整行会塌成左对齐。 */
  const wxss = read('pages/orders/orders.wxss');
  const i = wxss.indexOf('.oc-total-lbl');
  assert.ok(i >= 0, 'the total label has no style rule');
  assert.match(wxss.slice(i, wxss.indexOf('}', i)), /margin-left:\s*auto/,
    'the total group no longer pushes to the right edge');
});

check('the cancel button stays on one line', () => {
  const wxss = read('pages/orders/orders.wxss');
  const i = wxss.indexOf('.oc-cancel');
  assert.ok(i >= 0, 'the cancel button has no style rule');
  const rule = wxss.slice(i, wxss.indexOf('}', i));
  assert.match(rule, /white-space:\s*nowrap/, 'the cancel button may wrap');
  assert.match(rule, /flex:\s*0\s+0\s+auto|flex-shrink:\s*0/,
    'the cancel button may be squeezed until the text overflows');
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`ORDER_CARD_GATE=FAIL (${fails.length}/9)`);
  process.exit(1);
}
console.log('ORDER_CARD_GATE=PASS');
