#!/usr/bin/env node
/* 入口页为身份选择页；商户端入口触发微信真实授权且不声称校验已发生
   （PRD §4.4、§14、项目方 2026-08-22 决策）。
   用法: node check_identity_entry.js <repo-root>

   本门禁接管 archive/2026-08-22-browse-before-identity/checks/check_browse_first.js
   —— 见 proposal.md「与已归档门禁的冲突」。 */
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
const BROWSING = ['home', 'menu', 'detail'];

function mp() {
  for (const k of Object.keys(require.cache)) if (k.startsWith(MP + path.sep)) delete require.cache[k];
  let pageDef = null;
  const navs = [];
  const toasts = [];
  global.Behavior = d => d; global.Component = d => d; global.App = () => {};
  global.Page = d => { pageDef = d; };
  global.wx = {
    getWindowInfo: () => ({ statusBarHeight: 20, screenWidth: 375, safeArea: { bottom: 778 } }),
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect: () => ({ top: 26, left: 278, height: 32 }),
    navigateTo(r) { navs.push(r.url); }, redirectTo(r) { navs.push(r.url); },
    reLaunch(r) { navs.push(r.url); }, navigateBack() { navs.push('back'); },
    setClipboardData() {},
    showToast(o) { toasts.push(o && o.title); },
    showModal(o) { toasts.push(o && o.content); },
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
      selectComponent: () => ({ show: m => toasts.push(m) }),
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
  return { data, globalData, load, invoke, navs, toasts };
}
const openLaunch = h => { const p = h.load('pages/launch/launch.js'); h.invoke(p, 'onLoad', {}); return p; };

/* ================= 入口页（推翻归档断言） ================= */

check('the entry page is the identity screen', () => {
  const app = JSON.parse(read('app.json'));
  assert.equal(app.pages[0], 'pages/launch/launch',
    `the entry page is ${app.pages[0]}, not the identity screen`);
  assert.ok(app.pages.includes('pages/home/home'), 'the user home screen left the page list');
});

/* ================= 商户端入口 ================= */

check('the merchant entry uses the real wechat authorisation control', () => {
  const wxml = read('pages/launch/launch.wxml');
  const i = wxml.indexOf('商户端');
  assert.ok(i > 0, 'the merchant entry disappeared from the identity screen');
  const card = wxml.slice(Math.max(0, i - 600), i + 200);
  assert.match(card, /open-type\s*=\s*"getPhoneNumber"/,
    'the merchant entry does not trigger the wechat phone authorisation');
  assert.match(card, /bindgetphonenumber\s*=\s*"(\w+)"/, 'the authorisation result is not handled');
  assert.doesNotMatch(card, /data-to="admin-orders"[^>]*bindtap/,
    'the merchant entry still navigates straight through without authorisation');
  const h = mp();
  const page = openLaunch(h);
  const handler = card.match(/bindgetphonenumber\s*=\s*"(\w+)"/)[1];
  assert.equal(typeof page[handler], 'function', `the page has no ${handler} handler`);
});

check('declining stays on the page without an error', () => {
  const h = mp();
  const page = openLaunch(h);
  const wxml = read('pages/launch/launch.wxml');
  const handler = wxml.match(/bindgetphonenumber\s*=\s*"(\w+)"/)[1];
  // 微信在拒绝时回调 errMsg 且不带 code / encryptedData
  page[handler].call(page, { detail: { errMsg: 'getPhoneNumber:fail user deny' } });
  assert.ok(!h.navs.some(u => String(u).includes('admin-orders')),
    'declining still opened the merchant end');
  const said = JSON.stringify(page.data) + h.toasts.join(' ');
  assert.match(said, /商户端|验证|身份/, 'declining produced no explanation');
  assert.doesNotMatch(said, /失败|错误|出错/, 'declining was rendered as a failure; it is a legitimate choice');
});

check('allowing enters the merchant end and states verification has not happened', () => {
  const h = mp();
  const page = openLaunch(h);
  const wxml = read('pages/launch/launch.wxml');
  const handler = wxml.match(/bindgetphonenumber\s*=\s*"(\w+)"/)[1];
  page[handler].call(page, { detail: { errMsg: 'getPhoneNumber:ok', code: 'abc123', encryptedData: 'x' } });
  assert.ok(h.navs.some(u => String(u).includes('admin-orders')), 'allowing did not open the merchant end');
  const said = (JSON.stringify(page.data) + ' ' + h.toasts.join(' '));
  assert.match(said, /服务端|后端/, 'nothing tells the user the check still depends on the server');
  assert.doesNotMatch(said, /验证通过|身份已确认|校验通过/,
    'the interface claims a verification that never happened');
});

check('the front end claims no access control over the merchant screens', () => {
  /* §4.4 末条：客户端菜单隐藏不能代替鉴权。前端不得出现任何角色判定。 */
  for (const f of walk(MP)) {
    if (!f.endsWith('.js')) continue;
    const rel = path.relative(MP, f);
    const src = fs.readFileSync(f, 'utf8');
    assert.doesNotMatch(src, /isMerchant|hasMerchantRole|checkRole|verifyMerchant/,
      `${rel} implements a front-end role check; authorisation belongs to the server`);
  }
});

/* ========== 以下六项接管自 archive/2026-08-22-browse-before-identity ========== */

check('[takeover] the launch screen holds no self-drawn authorisation popup', () => {
  const h = mp();
  const page = openLaunch(h);
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

check('[takeover] the user entry opens the home screen with no gate', () => {
  const wxml = read('pages/launch/launch.wxml');
  const userCard = wxml.slice(Math.max(0, wxml.indexOf('用户端') - 400), wxml.indexOf('用户端'));
  assert.doesNotMatch(userCard, /openAuth|getPhoneNumber/, 'the user entry now asks for identity');
  assert.match(userCard, /bindtap="go"/, 'the user entry is not wired to plain navigation');
  assert.match(userCard, /data-to="home"/, 'the user entry does not point at the home screen');
});

check('[takeover] no browsing surface asks for identity', () => {
  for (const id of BROWSING) {
    for (const ext of ['js', 'wxml']) {
      const rel = `pages/${id}/${id}.${ext}`;
      const src = read(rel);
      assert.doesNotMatch(src, /getPhoneNumber/, `${rel} requests a phone number during browsing`);
      assert.doesNotMatch(src, /授权登录|申请获取/, `${rel} shows an authorization prompt during browsing`);
    }
  }
});

check('[takeover] no fake authorisation control survives anywhere', () => {
  /* 假授权 = 模板里写着授权话术，脚本里却没有任何微信授权接口调用。
     只找「请求授权」的话术，不找「已授权」这类状态标签。 */
  for (const f of walk(MP)) {
    if (!f.endsWith('.wxml')) continue;
    const rel = path.relative(MP, f);
    const src = fs.readFileSync(f, 'utf8');
    if (!/申请获取|授权登录|获取[^。]{0,8}手机号/.test(src)) continue;
    const js = f.replace(/\.wxml$/, '.js');
    const jsSrc = fs.existsSync(js) ? fs.readFileSync(js, 'utf8') : '';
    assert.match(jsSrc, /getPhoneNumber|wx\.login|requestSubscribeMessage/,
      `${rel} shows an authorization prompt with no wechat call behind it`);
  }
});

check('[takeover] the home screen offers no route to the identity screen', () => {
  const wxml = read('pages/home/home.wxml');
  const nav = wxml.slice(wxml.indexOf('<navbar'), wxml.indexOf('>', wxml.indexOf('<navbar')) + 1);
  assert.doesNotMatch(nav, /\bexit\b/, 'the home navbar offers a jump back to the identity screen');
});

check('[takeover] the launch back control points somewhere reachable', () => {
  const h = mp();
  const page = openLaunch(h);
  const wxml = read('pages/launch/launch.wxml');
  for (const m of wxml.matchAll(/bindtap="(\w+)"/g)) {
    assert.equal(typeof page[m[1]], 'function', `the launch template binds ${m[1]} but there is no such handler`);
  }
  const util = require(path.join(MP, 'utils/util.js'));
  assert.equal(typeof util.nav.toBrand, 'undefined',
    'nav.toBrand reappeared; the brand screen was removed with the section 0.2 list');
  assert.doesNotMatch(read('pages/launch/launch.js'), /toBrand/,
    'the launch screen still calls a handler removed with the brand screen');
});

check('all javascript parses', () => {
  const files = walk(MP).filter(f => f.endsWith('.js'));
  for (const f of files) new vm.Script(fs.readFileSync(f, 'utf8'), { filename: f });
  console.log(`  parsed ${files.length} javascript files`);
});

if (fails.length) {
  console.log(fails.map(f => `  ${f}`).join('\n'));
  console.log(`IDENTITY_ENTRY_GATE=FAIL (${fails.length}/12)`);
  process.exit(1);
}
console.log('IDENTITY_ENTRY_GATE=PASS');
