#!/usr/bin/env node
/* 小程序商户端订单搜索与取餐号营业日口径（PRD §6.6、§7.8）。
   用法: node check_merchant_search.js <repo-root>
   断言能力事实而非断言词：搜索框断「是 input 且绑定事件」，不断「搜索」二字；
   营业日限定断「解析实现只有一份」，不断注释文字。 */
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
const CODE_RE = /^\d{4}$/;

/* ---- 最小页面宿主。不复用 tests/page-harness.js：归档产物必须自足。 ---- */
function host() {
  for (const k of Object.keys(require.cache)) {
    if (k.startsWith(MP + path.sep)) delete require.cache[k];
  }
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
    navigateTo() {}, redirectTo() {}, reLaunch() {}, navigateBack() {},
  };
  const data = require(path.join(MP, 'utils/data.js'));
  const cl = v => JSON.parse(JSON.stringify(v));
  const globalData = {
    statusBarHeight: 20, navBarHeight: 44, navTotalHeight: 64, safeBottom: 0,
    aOrders: cl(data.ADMIN_ORDERS), orders: cl(data.USER_ORDERS), menu: cl(data.MENU),
    store: { status: '营业中' },
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
      selectComponent: () => ({ show: (m, c) => toasts.push({ message: m, config: c }) }),
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

const type = (page, value) => page.onKw({ detail: { value } });
const ids = page => page.data.list.map(o => o.id);

/* ---- 种子事实：跨营业日重复号、只属于旧营业日的号、退款中的单 ---- */
function facts(h) {
  const all = h.data.ADMIN_ORDERS;
  const today = h.data.BUSINESS_DAY;
  const byCode = new Map();
  for (const o of all) {
    if (!byCode.has(o.code)) byCode.set(o.code, []);
    byCode.get(o.code).push(o);
  }
  const dup = [...byCode.values()].find(g => g.length > 1 && new Set(g.map(o => o.pickupDate)).size > 1);
  const staleOnly = all.find(o => o.pickupDate !== today && !all.some(x => x.code === o.code && x.pickupDate === today));
  const refunding = all.find(o => o.status === '退款中');
  return { all, today, dup, staleOnly, refunding };
}

/* ================= R1 商户订单列表可跨泳道搜索 ================= */

check('the search box is a bound input, not a static label', () => {
  const wxml = read('pages/admin-orders/admin-orders.wxml');
  const box = wxml.slice(wxml.indexOf('search-box'), wxml.indexOf('search-box') + 600);
  assert.match(box, /<input\b/, 'the search area still has no input element');
  assert.match(box, /bindinput\s*=\s*"onKw"/, 'the input is not bound to an input handler');
  assert.match(box, /value\s*=\s*"\{\{\s*kw\s*\}\}"/, 'the input is not controlled by page state');
  const h = host();
  const page = h.load('pages/admin-orders/admin-orders.js');
  assert.equal(typeof page.onKw, 'function', 'the page exposes no input handler');
});

check('search spans every lane', () => {
  const h = host();
  const f = facts(h);
  const page = h.load('pages/admin-orders/admin-orders.js');
  h.invoke(page, 'onShow');
  // 站在 已预约 泳道，搜一张 已完成 的单
  page.switchLane({ currentTarget: { dataset: { l: '已预约' } } });
  const done = f.all.find(o => o.status === '已完成' && o.pickupDate === f.today);
  type(page, done.code);
  assert.ok(ids(page).includes(done.id),
    `a 已完成 order was not reachable from the 已预约 lane by its code ${done.code}`);
  const states = new Set(page.data.list.map(o => o.status));
  assert.ok(!states.has('已预约') || states.size > 1 || page.data.lane === '已预约',
    'search results appear filtered by the selected lane');
  // 同一个手机号跨状态时结果必须同时出现
  const phone = done.phone;
  const expect = f.all.filter(o => o.phone === phone).map(o => o.id).sort();
  type(page, phone);
  assert.equal(JSON.stringify(ids(page).sort()), JSON.stringify(expect),
    'a phone search did not return every order holding that phone');
});

check('search is a query, and choosing a lane leaves it', () => {
  const h = host();
  const f = facts(h);
  const page = h.load('pages/admin-orders/admin-orders.js');
  h.invoke(page, 'onShow');
  type(page, f.all[0].no);
  assert.ok(page.data.kw, 'the keyword did not reach page state');
  assert.equal('kw' in h.globalData, false, 'the keyword leaked into globalData');
  for (const o of h.globalData.aOrders) {
    assert.equal('kw' in o, false, 'the keyword became a field on the order model');
    assert.equal('matched' in o, false, 'search stamped a flag onto the order model');
  }
  page.switchLane({ currentTarget: { dataset: { l: '制作中' } } });
  assert.equal(page.data.kw, '', 'choosing a lane did not clear the search keyword');
  assert.equal(JSON.stringify(ids(page).sort()),
    JSON.stringify(f.all.filter(o => o.status === '制作中').map(o => o.id).sort()),
    'leaving search did not restore the full lane');
});

/* ================= R2 取餐号 4 位数字、按取餐日期累计 ================= */

check('every pickup code is four digits on both surfaces', () => {
  const h = host();
  for (const [label, list] of [['merchant', h.data.ADMIN_ORDERS], ['user', h.data.USER_ORDERS]]) {
    for (const o of list) {
      assert.match(String(o.code), CODE_RE, `${label} order ${o.id} carries a non-conforming pickup code ${o.code}`);
    }
  }
  // 遗留前缀不得残留在任何商户端模板或页面逻辑里
  for (const rel of ['pages/admin-verify/admin-verify.js', 'pages/admin-verify/admin-verify.wxml']) {
    assert.doesNotMatch(read(rel), /\b[A-Z]\d{3}\b/, `${rel} still shows a prefixed pickup code`);
  }
});

check('one pickup code repeats across business days', () => {
  const h = host();
  const f = facts(h);
  assert.ok(f.dup, 'no pickup code repeats across two business days — the business-day rule is unfalsifiable');
  assert.ok(f.dup.some(o => o.pickupDate === f.today), 'the repeated code has no instance on the current business day');
  assert.ok(f.staleOnly, 'no pickup code exists only outside the current business day');
  const perDay = new Map();
  for (const o of f.all) {
    const k = o.pickupDate + '#' + o.code;
    assert.equal(perDay.has(k), false, `pickup code ${o.code} is reused within ${o.pickupDate}`);
    perDay.set(k, o.id);
  }
});

check('the business day is a constant, not the clock', () => {
  const h = host();
  assert.match(String(h.data.BUSINESS_DAY), /^\d{4}-\d{2}-\d{2}$/, 'BUSINESS_DAY is not an exported ISO date');
  for (const rel of ['utils/data.js', 'utils/util.js']) {
    const src = read(rel);
    assert.doesNotMatch(src, /new Date\(|Date\.now\(/, `${rel} derives time from the runtime clock`);
  }
  for (const o of h.data.ADMIN_ORDERS) {
    assert.match(String(o.pickupDate), /^\d{4}-\d{2}-\d{2}$/, `merchant order ${o.id} has no pickup date`);
  }
});

/* ================= R3 取餐号解析限定当前营业日 ================= */

check('pickup code resolution lives in exactly one place', () => {
  const h = host();
  assert.equal(typeof h.util.findByCode, 'function', 'the shared pickup-code resolver does not exist');
  assert.equal(typeof h.util.codeHint, 'function', 'the cross-day hint does not exist');
  for (const rel of ['pages/admin-verify/admin-verify.js', 'pages/admin-orders/admin-orders.js']) {
    const src = read(rel);
    assert.doesNotMatch(src, /\.code\s*===|\.code\.toUpperCase\(\)/,
      `${rel} resolves pickup codes on its own instead of using the shared resolver`);
  }
  assert.match(read('pages/admin-verify/admin-verify.js'), /findByCode/,
    'the verify screen does not use the shared resolver');
});

check('a four-digit code resolves only within the current business day', () => {
  const h = host();
  const f = facts(h);
  const hit = h.util.findByCode(f.dup[0].code);
  assert.ok(hit, 'a code present on the current business day resolved to nothing');
  assert.equal(hit.pickupDate, f.today, 'a pickup code resolved to an order from another business day');
  assert.equal(h.util.findByCode(f.staleOnly.code), null,
    'a code belonging only to another business day still resolved');
});

check('a stale code is refused with a hint naming its business day', () => {
  const h = host();
  const f = facts(h);
  const hint = h.util.codeHint(f.staleOnly.code);
  assert.ok(hint && hint.length, 'a stale pickup code produced no hint');
  assert.ok(hint.includes(f.staleOnly.pickupDate), 'the hint does not name the business day the code belongs to');
  assert.match(hint, /订单号|手机号/, 'the hint offers no alternative way to locate the order');
  assert.equal(h.util.codeHint(f.dup[0].code), '', 'a code valid today still produced a cross-day hint');
  // 核销路径必须拒绝该号，且拒绝理由不是「无效取餐号」
  const page = h.load('pages/admin-verify/admin-verify.js');
  page.tryVerify(f.staleOnly.code);
  const said = h.toasts.map(t => t.message).join(' ') + JSON.stringify(page.data.match || {});
  assert.ok(said.includes(f.staleOnly.pickupDate),
    'manual verification rejected a stale code without naming its business day');
});

check('a four-digit phone fragment still finds its order', () => {
  const h = host();
  const f = facts(h);
  const page = h.load('pages/admin-orders/admin-orders.js');
  h.invoke(page, 'onShow');
  const src = f.all.find(o => /\d{4}$/.test(String(o.phone)));
  const tail = String(src.phone).slice(-4);
  assert.match(tail, CODE_RE, 'the chosen phone fragment is not four digits');
  type(page, tail);
  assert.ok(ids(page).includes(src.id),
    `a four-digit phone fragment ${tail} was swallowed by the pickup-code branch`);
});

check('an order under refund is findable although it has no lane', () => {
  const h = host();
  const f = facts(h);
  assert.ok(f.refunding, 'the seed carries no 退款中 order, so its visibility cannot be verified');
  const page = h.load('pages/admin-orders/admin-orders.js');
  h.invoke(page, 'onShow');
  // 泳道集合保持 §6.6 的五档 + 全部，本 change 不得引入 退款中 泳道
  assert.equal(JSON.stringify(page.data.lanes),
    JSON.stringify(['已预约', '制作中', '待取餐', '已完成', '已退款', '全部']),
    'the lane set drifted from the five lanes PRD §6.6 enumerates');
  type(page, f.refunding.no);
  assert.ok(ids(page).includes(f.refunding.id), 'a 退款中 order could not be found by its order number');
  page.switchLane({ currentTarget: { dataset: { l: '全部' } } });
  assert.ok(ids(page).includes(f.refunding.id), 'a 退款中 order is missing from the 全部 lane');
});

check('all javascript parses', () => {
  const files = [];
  (function walk(d) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      if (e.name === 'node_modules') continue;
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p); else if (e.name.endsWith('.js')) files.push(p);
    }
  })(MP);
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`MERCHANT_SEARCH_GATE=FAIL (${fails.length}/12)`);
  process.exit(1);
}
console.log('MERCHANT_SEARCH_GATE=PASS');
