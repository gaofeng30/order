#!/usr/bin/env node
/* 小程序订单模型对齐 §15.6.2（金额整数分、结算事实齐备、剩余时间现算、
   整单只有 orderNote、金额单一格式化入口）。
   用法: node check_mp_order_model.js <repo-root> */
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

const CATALOG_ITEM = { id: '1', category_id: '9', name: '目录来的双拼饭',
                       description: 'd', specification: 's', price_cents: 3250 };

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
  return { data, util: require(path.join(MP, 'utils/util.js')), globalData, load, invoke, toasts };
}

function placeOrder(h) {
  h.util.cart.add(CATALOG_ITEM);
  const confirm = h.load('pages/confirm/confirm.js');
  h.invoke(confirm, 'onLoad');
  confirm.pay();
  return h.globalData.orders[0];
}

const FACTS = ['id', 'no', 'code', 'status', 'pickupDate', 'pickupTime', 'mealPeriod',
               'pickupPoint', 'paidAt', 'subtotal', 'discountRate', 'discountCut',
               'total', 'isStaff', 'contact', 'phone', 'items'];

function settles(o, where) {
  for (const k of ['subtotal', 'discountCut', 'total']) {
    assert.equal(Number.isInteger(o[k]), true, `${where} ${o.id}.${k} = ${o[k]} is not an integer`);
  }
  /* 分为单位 —— 元为单位时 total 会是 26~76 这样的两位数 */
  assert.ok(o.total >= 1000, `${where} ${o.id}.total = ${o.total} looks like yuan, not cents`);
  assert.equal(o.subtotal - o.discountCut, o.total,
    `${where} ${o.id} does not settle: ${o.subtotal} - ${o.discountCut} != ${o.total}`);
  let sub = 0, paid = 0;
  for (const [, , qty, price, discounted] of o.items) {
    assert.equal(Number.isInteger(price), true, `${where} ${o.id} line price ${price} is not an integer`);
    assert.equal(Number.isInteger(discounted), true, `${where} ${o.id} line discounted ${discounted} is not an integer`);
    sub += qty * price; paid += qty * discounted;
  }
  assert.equal(sub, o.subtotal, `${where} ${o.id} item subtotal ${sub} != ${o.subtotal}`);
  assert.equal(paid, o.total, `${where} ${o.id} item paid ${paid} != ${o.total}`);
}

/* ================= 结算事实与整数分 ================= */

check('every seeded order carries the settlement facts in integer cents', () => {
  const h = mp();
  for (const [label, list] of [['user', h.data.USER_ORDERS], ['merchant', h.data.ADMIN_ORDERS]]) {
    assert.ok(list.length, `${label} seed is empty`);
    for (const o of list) {
      for (const k of FACTS) assert.equal(Object.hasOwn(o, k), true, `${label} ${o.id} missing ${k}`);
      assert.match(o.pickupDate, /^\d{4}-\d{2}-\d{2}$/, `${label} ${o.id}.pickupDate = ${o.pickupDate}`);
      assert.match(o.pickupTime, /^\d{2}:\d{2}$/, `${label} ${o.id}.pickupTime = ${o.pickupTime}`);
      assert.match(o.paidAt, /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/, `${label} ${o.id}.paidAt = ${o.paidAt}`);
      assert.equal(['lunch', 'dinner'].includes(o.mealPeriod), true, `${label} ${o.id}.mealPeriod = ${o.mealPeriod}`);
      assert.equal(typeof o.isStaff, 'boolean', `${label} ${o.id}.isStaff not boolean`);
      settles(o, label);
    }
  }
});

check('checkout writes the same fields and the same unit as the seed', () => {
  const h = mp();
  const o = placeOrder(h);
  for (const k of FACTS) assert.equal(Object.hasOwn(o, k), true, `checkout order missing ${k}`);
  settles(o, 'checkout');
  const seeded = h.data.USER_ORDERS[0];
  for (const k of FACTS) {
    assert.equal(typeof o[k], typeof seeded[k], `${k} is ${typeof o[k]} fresh but ${typeof seeded[k]} seeded`);
  }
});

check('checkout keeps cents integral through a half-yuan price', () => {
  /* 3250 分 × 1 —— 走元的往返会得到 32.5，再乘数量就会出现浮点尾数 */
  const h = mp();
  const o = placeOrder(h);
  assert.equal(o.total, 3250, `total = ${o.total}, expected 3250 cents`);
  assert.equal(o.items[0][3], 3250, `line price = ${o.items[0][3]}, expected 3250 cents`);
});

check('no order is priced in yuan anywhere', () => {
  const h = mp();
  for (const list of [h.data.USER_ORDERS, h.data.ADMIN_ORDERS]) {
    for (const o of list) {
      assert.ok(o.total >= 1000, `${o.id}.total = ${o.total} is still yuan`);
      for (const [, , , price, discounted] of o.items) {
        assert.ok(price >= 100, `${o.id} line price ${price} is still yuan`);
        assert.ok(discounted >= 100, `${o.id} line discounted ${discounted} is still yuan`);
      }
    }
  }
});

check('the guest pricing snapshot is honest, not a placeholder', () => {
  const h = mp();
  const all = [...h.data.USER_ORDERS, ...h.data.ADMIN_ORDERS, placeOrder(mp())];
  for (const o of all) {
    assert.equal(o.isStaff, false, `${o.id}.isStaff = ${o.isStaff}; the identity chain is not wired yet`);
    assert.equal(o.discountRate, 100, `${o.id}.discountRate = ${o.discountRate}`);
    assert.equal(o.discountCut, 0, `${o.id}.discountCut = ${o.discountCut}`);
  }
});

/* ================= 剩余时间现算 ================= */

check('no order stores a frozen time to pickup or pickup label', () => {
  const h = mp();
  const all = [...h.data.USER_ORDERS, ...h.data.ADMIN_ORDERS, placeOrder(mp())];
  for (const o of all) {
    for (const k of ['minsToPickup', 'pickupLabel']) {
      assert.equal(Object.hasOwn(o, k), false, `${o.id} still carries ${k}`);
    }
  }
  for (const f of walk(MP)) {
    if (!/\.(js|wxml)$/.test(f)) continue;
    const src = fs.readFileSync(f, 'utf8');
    assert.doesNotMatch(src, /\bo\.minsToPickup\b|\border\.minsToPickup\b|\bo\.pickupLabel\b|\border\.pickupLabel\b/,
      `${path.relative(MP, f)} still reads a frozen pickup field off an order`);
  }
});

check('the cancel window follows the clock, not the order record', () => {
  const h = mp();
  assert.equal(typeof h.data.minsToPickup, 'function', 'there is no derivation for time to pickup');
  const base = { status: '已预约', pickupDate: h.data.BUSINESS_DAY };
  const now = h.data.NOW_MINS;
  const at = m => {
    const t = now + m;
    return `${String(Math.floor(t / 60)).padStart(2, '0')}:${String(t % 60).padStart(2, '0')}`;
  };
  const far = Object.assign({}, base, { pickupTime: at(90) });
  const near = Object.assign({}, base, { pickupTime: at(10) });
  assert.equal(h.data.canCancelReserve(far), true, 'an order 90 minutes out is not cancellable');
  assert.equal(h.data.canCancelReserve(near), false, 'an order 10 minutes out is still cancellable');
  assert.ok(h.data.minsToPickup(far) > h.data.minsToPickup(near), 'the derivation ignores pickup time');
  // 判定只吃取餐时刻，不吃记录上的任何冻结值
  const poisoned = Object.assign({}, near, { minsToPickup: 999 });
  assert.equal(h.data.canCancelReserve(poisoned), false,
    'the cancel window honoured a stale field instead of the clock');
});

/* ================= 整单只有 orderNote ================= */

check('no order carries an order-level flavor', () => {
  const h = mp();
  const all = [...h.data.USER_ORDERS, ...h.data.ADMIN_ORDERS, placeOrder(mp())];
  for (const o of all) {
    for (const k of ['flavor', 'flavors']) {
      assert.equal(Object.hasOwn(o, k), false, `${o.id} still carries an order-level ${k}`);
    }
    assert.equal(Object.hasOwn(o, 'orderNote'), true, `${o.id} has no orderNote`);
  }
  for (const f of walk(MP)) {
    if (!/\.(js|wxml)$/.test(f)) continue;
    assert.doesNotMatch(fs.readFileSync(f, 'utf8'), /\bo\.flavors?\b|\bitem\.flavor\b/,
      `${path.relative(MP, f)} still reads an order-level flavor`);
  }
});

check('per-item flavours survive the removal', () => {
  const h = mp();
  const target = h.globalData.aOrders.find(o => o.items.some(it => it[5] && String(it[5]).trim()));
  assert.ok(target, 'seed has no order with a per-item flavour to display');
  const page = h.load('pages/admin-orders/admin-orders.js');
  h.invoke(page, 'onShow');
  const row = page.data.list.find(r => r.id === target.id);
  assert.ok(row, 'the order vanished from the merchant list');
  const flavour = String(target.items.find(it => it[5] && String(it[5]).trim())[5]);
  assert.ok(row.band && row.band.includes(flavour),
    `per-item flavour "${flavour}" is not shown after dropping the order-level field: band=${row.band}`);
});

/* ================= 金额单一格式化入口 ================= */

check('money is formatted through exactly one entry point', () => {
  const h = mp();
  const money = require(path.join(MP, 'utils/money.js'));
  assert.equal(typeof money.yuan, 'function', 'there is no single money formatter');
  assert.equal(money.yuan(3250), '32.50');
  assert.equal(money.yuan(0), '0.00');
  assert.equal(money.yuan(7), '0.07');
  // 非法输入不得伪装成目录/网络故障，也不得显示成 0.00
  /* 断结构不断词：注释里解释「为什么不复用目录格式化」会命中任何朴素正则。
     真正的事实是它不依赖目录模块，且不抛目录错误。 */
  assert.doesNotMatch(read('utils/money.js'), /require\(\s*['"][^'"]*catalog/i,
    'the money formatter depends on the catalog layer');
  for (const bad of [null, undefined, '', NaN, 'abc', {}]) {
    let out;
    assert.doesNotThrow(() => { out = money.yuan(bad); }, `the formatter throws on ${String(bad)}`);
    assert.equal(out, '—', `the formatter returned ${out} for ${String(bad)} instead of a dash`);
  }
});

check('no page divides money by hand', () => {
  for (const f of walk(MP)) {
    if (!/\.(js|wxml)$/.test(f)) continue;
    const rel = path.relative(MP, f);
    if (rel === 'utils/money.js' || rel.startsWith('utils/catalog')) continue;
    const src = fs.readFileSync(f, 'utf8');
    /* 只针对金额语境。裸的 `/ 100` 会命中日历算法里的 Math.floor(doe / 100)，
       那是纪元换算不是分转元 —— 断言必须指向事实，不是指向一个除号。 */
    assert.doesNotMatch(src, /(cents|price|total|subtotal|amount|sub|paid)\w*\s*\/\s*100\b/i,
      `${rel} converts money by hand`);
    assert.doesNotMatch(src, /toFixed\(2\)/, `${rel} formats money with toFixed`);
  }
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`MP_ORDER_MODEL_GATE=FAIL (${fails.length}/12)`);
  process.exit(1);
}
console.log('MP_ORDER_MODEL_GATE=PASS');
