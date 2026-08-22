#!/usr/bin/env node
/* 免手机号浏览、启动不弹授权（PRD §14、§4.4、§5.6、§5.9）。
   用法: node check_browse_first.js <repo-root> */
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
  global.getCurrentPages = () => [];
  global.wx = {
    getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, safeArea: { bottom: 778 } }),
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect: () => ({ top: 26, left: 278, height: 32 }),
    navigateTo(r) { navs.push(['navigateTo', r.url]); },
    redirectTo(r) { navs.push(['redirectTo', r.url]); },
    reLaunch(r) { navs.push(['reLaunch', r.url]); },
    navigateBack(r) { navs.push(['navigateBack', (r && r.delta) || 1]); },
    setClipboardData() {},
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

// 浏览路径：启动到提交订单之前
const BROWSING = ['home', 'menu', 'detail'];

/* ================= 入口 ================= */

check('the entry page is the user home screen', () => {
  const app = JSON.parse(read('app.json'));
  assert.equal(app.pages[0], 'pages/home/home',
    `the entry page is ${app.pages[0]}; PRD 4.4 sends unbound users straight to the home screen`);
  assert.ok(app.pages.includes('pages/launch/launch'),
    'the identity screen was dropped from the page list; it is still one of the nine user screens');
});

/* ================= 无假授权 ================= */

check('the launch screen holds no authorization popup', () => {
  const h = mp();
  const page = h.load('pages/launch/launch.js');
  h.invoke(page, 'onLoad', {});
  for (const k of ['auth', 'wxIcon']) {
    assert.equal(Object.hasOwn(page.data, k), false, `the launch screen still carries ${k} state`);
  }
  for (const fn of ['openAuth', 'closeAuth', 'allowAuth']) {
    assert.equal(typeof page[fn], 'undefined', `the launch screen still exposes ${fn}`);
  }
  const wxml = read('pages/launch/launch.wxml');
  assert.doesNotMatch(wxml, /auth-mask|auth-card|auth-btn/, 'the launch template still renders the popup');
  assert.doesNotMatch(read('pages/launch/launch.wxss'), /\.auth-/, 'the popup styles survived');
});

check('the user entry opens the home screen with no gate', () => {
  const h = mp();
  const page = h.load('pages/launch/launch.js');
  h.invoke(page, 'onLoad', {});
  const wxml = read('pages/launch/launch.wxml');
  const userCard = wxml.slice(wxml.indexOf('用户端') - 400, wxml.indexOf('用户端'));
  assert.doesNotMatch(userCard, /openAuth/, 'the user entry still opens an authorization popup');
  assert.match(userCard, /bindtap="go"/, 'the user entry is not wired to plain navigation');
  assert.match(userCard, /data-to="home"/, 'the user entry does not point at the home screen');
});

check('no browsing surface asks for identity', () => {
  for (const id of BROWSING) {
    for (const ext of ['js', 'wxml']) {
      const rel = `pages/${id}/${id}.${ext}`;
      const src = read(rel);
      assert.doesNotMatch(src, /getPhoneNumber|open-type\s*=\s*"getPhoneNumber"/,
        `${rel} requests a phone number during browsing`);
      assert.doesNotMatch(src, /授权登录|申请获取/, `${rel} shows an authorization prompt during browsing`);
    }
  }
});

check('no fake authorization control survives anywhere', () => {
  /* 假授权 = 模板里写着授权字样，脚本里却没有任何微信授权接口调用。
     两者都不存在才算干净；只删文案会留下控件，只删调用则不成立。 */
  for (const f of walk(MP)) {
    if (!f.endsWith('.wxml')) continue;
    const rel = path.relative(MP, f);
    const src = fs.readFileSync(f, 'utf8');
    /* 只找「请求授权」的话术，不找「已授权」这类状态标签 ——
       后者是展示，不是控件；按「授权」二字一刀切会误伤它。 */
    if (!/申请获取|授权登录|获取[^。]{0,8}手机号/.test(src)) continue;
    const js = f.replace(/\.wxml$/, '.js');
    const jsSrc = fs.existsSync(js) ? fs.readFileSync(js, 'utf8') : '';
    assert.match(jsSrc, /getPhoneNumber|wx\.login|requestSubscribeMessage/,
      `${rel} shows an authorization prompt with no wechat call behind it`);
  }
});

/* ================= 身份选择页的可达性 ================= */

check('the home screen offers no route to the identity screen', () => {
  const wxml = read('pages/home/home.wxml');
  const nav = wxml.slice(wxml.indexOf('<navbar'), wxml.indexOf('>', wxml.indexOf('<navbar')) + 1);
  assert.doesNotMatch(nav, /\bexit\b/, 'the home navbar still offers a jump back to the identity screen');
  assert.doesNotMatch(wxml, /launch/, 'the home template still links to the identity screen');
});

check('the launch back control points somewhere reachable', () => {
  const h = mp();
  const page = h.load('pages/launch/launch.js');
  h.invoke(page, 'onLoad', {});
  const util = require(path.join(MP, 'utils/util.js'));
  const wxml = read('pages/launch/launch.wxml');
  for (const m of wxml.matchAll(/bindtap="(\w+)"/g)) {
    assert.equal(typeof page[m[1]], 'function', `the launch template binds ${m[1]} but the page has no such handler`);
  }
  assert.equal(typeof util.nav.toBrand, 'undefined',
    'nav.toBrand reappeared; the brand screen was removed with the section 0.2 list');
  assert.doesNotMatch(read('pages/launch/launch.js'), /toBrand/,
    'the launch screen still calls a handler removed with the brand screen');
});

check('the merchant entry is unchanged and still reachable from the identity screen', () => {
  const h = mp();
  const page = h.load('pages/launch/launch.js');
  h.invoke(page, 'onLoad', {});
  const wxml = read('pages/launch/launch.wxml');
  assert.match(wxml, /data-to="admin-orders"/, 'the merchant entry disappeared from the identity screen');
  page.go({ currentTarget: { dataset: { to: 'admin-orders' } } });
  assert.ok(h.navs.some(([, url]) => String(url).includes('admin-orders')),
    'the merchant entry no longer navigates');
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`BROWSE_FIRST_GATE=FAIL (${fails.length}/9)`);
  process.exit(1);
}
console.log('BROWSE_FIRST_GATE=PASS');
