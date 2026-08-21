#!/usr/bin/env node
/* 首页补齐 §5.1：门店公告、当前营业状态、进行中订单提示条。
   用法: node check_home_screen.js <repo-root> */
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
  const toasts = [];
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
  return { data, globalData, load, invoke, toasts, navs };
}

const openHome = h => { const p = h.load('pages/home/home.js'); h.invoke(p, 'onLoad', {}); h.invoke(p, 'onShow'); return p; };
const ONGOING = ['已预约', '制作中', '待取餐'];

/* ================= 公告与营业状态 ================= */

check('the home screen shows the configured store notice', () => {
  const h = mp();
  assert.equal(typeof h.data.STORE.notice, 'string', 'the store seed has no notice field');
  assert.ok(h.data.STORE.notice.trim(), 'the store notice is empty');
  h.globalData.store.notice = '门店今日提前打烊';
  const page = openHome(h);
  const shown = JSON.stringify(page.data);
  assert.ok(shown.includes('门店今日提前打烊'), 'the home screen did not read the notice from configuration');
  const wxml = read('pages/home/home.wxml');
  assert.ok(/notice/.test(wxml), 'the home template renders no notice');
  assert.ok(!wxml.includes(h.data.STORE.notice), 'the notice text is hardcoded in the template');
  assert.ok(!read('pages/home/home.js').includes(h.data.STORE.notice), 'the notice text is hardcoded in the script');
});

check('the home screen shows the business status and follows the merchant', () => {
  /* 宿主必须串行创建：mp() 会重置 global.Page 与 require.cache，
     交错持有两个宿主会让先建的那个再也拿不到页面定义。 */
  for (const st of ['营业中', '休息中', '已截单']) {
    const g = mp();
    g.globalData.store.status = st;
    assert.ok(JSON.stringify(openHome(g).data).includes(st), `the home screen does not surface ${st}`);
  }
  // 不得由截单时刻派生：手动置为 营业中 时不能显示成 已截单
  const h = mp();
  h.globalData.store.status = '营业中';
  assert.ok(!JSON.stringify(openHome(h).data).includes('已截单'),
    'the home screen derived the status instead of reading the merchant override');
});

/* ================= 进行中订单提示条 ================= */

check('the in-flight strip is absent when nothing is in flight', () => {
  const h = mp();
  h.globalData.orders = h.globalData.orders.filter(o => !ONGOING.includes(o.status));
  const page = openHome(h);
  assert.ok(!page.data.ongoing, `an in-flight strip was rendered with no order in flight: ${JSON.stringify(page.data.ongoing)}`);
});

check('the in-flight strip counts exactly the three in-flight states', () => {
  const h = mp();
  h.globalData.orders = h.globalData.orders.map(o =>
    Object.assign({}, o, { status: o.id === 'o1' ? '已预约' : o.status }));
  const page = openHome(h);
  const expect = h.globalData.orders.filter(o => ONGOING.includes(o.status)).length;
  assert.ok(page.data.ongoing, 'no in-flight strip with orders in flight');
  assert.equal(page.data.ongoing.count, expect,
    `strip counts ${page.data.ongoing.count}, expected ${expect}`);
  // 已完成 / 已退款 / 退款中 不得计入
  const g = mp();
  g.globalData.orders = g.globalData.orders.map(o => Object.assign({}, o, { status: '已完成' }));
  assert.ok(!openHome(g).data.ongoing, 'completed orders produced an in-flight strip');
});

check('the in-flight strip names the earliest pickup, not the earliest order', () => {
  const h = mp();
  const day = h.data.BUSINESS_DAY;
  h.globalData.orders = [
    { id: 'x1', code: '0001', status: '已预约', pickupDate: day, pickupTime: '19:00',
      paidAt: `${day} 09:00:00`, pickupPoint: '县前直营店', items: [], total: 100 },
    { id: 'x2', code: '0002', status: '已预约', pickupDate: day, pickupTime: '17:30',
      paidAt: `${day} 10:00:00`, pickupPoint: '县前直营店', items: [], total: 100 },
  ];
  const page = openHome(h);
  assert.equal(page.data.ongoing.orderId, 'x2',
    'the strip followed the order time instead of the pickup time');
  assert.ok(page.data.ongoing.text.includes('17:30'),
    `the strip does not name the earliest pickup: ${page.data.ongoing.text}`);
});

check('a ready order changes the wording and highlights', () => {
  const h = mp();
  const day = h.data.BUSINESS_DAY;
  h.globalData.orders = [
    { id: 'y1', code: '0001', status: '已预约', pickupDate: day, pickupTime: '17:30',
      paidAt: `${day} 09:00:00`, pickupPoint: '县前直营店', items: [], total: 100 },
    { id: 'y2', code: '0002', status: '待取餐', pickupDate: day, pickupTime: '19:00',
      paidAt: `${day} 10:00:00`, pickupPoint: '县前直营店', items: [], total: 100 },
  ];
  const page = openHome(h);
  assert.equal(page.data.ongoing.ready, true, 'a 待取餐 order did not put the strip in the ready state');
  assert.ok(page.data.ongoing.text.includes('已备好'), `ready wording missing: ${page.data.ongoing.text}`);
  assert.equal(page.data.ongoing.orderId, 'y2', 'the ready order is not the one the strip points at');
  assert.ok(!/\d+\s*单/.test(page.data.ongoing.text),
    `the ready wording still counts orders and dilutes the call to action: ${page.data.ongoing.text}`);
});

check('tapping the in-flight strip opens that order', () => {
  const h = mp();
  const page = openHome(h);
  assert.ok(page.data.ongoing, 'no in-flight strip to tap');
  assert.equal(typeof page.tapOngoing, 'function', 'the strip has no tap handler');
  page.tapOngoing();
  assert.ok(h.navs.some(u => u.includes('order-detail') && u.includes(page.data.ongoing.orderId)),
    `tapping the strip did not open the order: ${JSON.stringify(h.navs)}`);
  const wxml = read('pages/home/home.wxml');
  assert.match(wxml, /bindtap="tapOngoing"/, 'the strip is not bound in the template');
});

/* ================= 残留与范围 ================= */

check('no coming-soon placeholder survives on the home screen', () => {
  const wxml = read('pages/home/home.wxml');
  assert.doesNotMatch(wxml, /未开放|即将上线|敬请期待/, 'the home template still renders a coming-soon badge');
  assert.doesNotMatch(wxml, /item\.off/, 'the home template still branches on a disabled entry');
  assert.doesNotMatch(read('pages/home/home.js'), /未开放|即将上线/, 'the home script still carries a coming-soon label');
});

check('the home screen keeps exactly the three first-phase entries', () => {
  const h = mp();
  const page = openHome(h);
  assert.equal(JSON.stringify(page.data.grid.map(g => g.k)), JSON.stringify(['reserve', 'orders', 'pickup']),
    `the entry set drifted: ${JSON.stringify(page.data.grid.map(g => g.k))}`);
  for (const g of page.data.grid) {
    assert.equal(Object.hasOwn(g, 'off'), false, `entry ${g.k} still carries a disabled flag`);
  }
});

check('the home screen still carries no marketing surface', () => {
  const js = read('pages/home/home.js');
  const wxml = read('pages/home/home.wxml');
  for (const token of ['banners', 'bannerIdx', 'swiper', '入群', '会员', '推荐商品', '今日招牌']) {
    assert.ok(!js.includes(token), `home.js still carries ${token}`);
    assert.ok(!wxml.includes(token), `home.wxml still carries ${token}`);
  }
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`HOME_SCREEN_GATE=FAIL (${fails.length}/11)`);
  process.exit(1);
}
console.log('HOME_SCREEN_GATE=PASS');
