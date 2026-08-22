const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 0818 PRD §14：用户能免手机号浏览，启动时不弹手机号授权。
//          §4.4：未绑定商户手机号的用户直接进用户端首页。
//          §5.9：商户经个人中心的「切换身份」回身份选择页。
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

test('the entry page is the home screen, not the identity screen', () => {
  const app = JSON.parse(read('app.json'));
  assert.equal(app.pages[0], 'pages/home/home');
  assert.ok(app.pages.includes('pages/launch/launch'), '身份选择页仍是用户端 9 屏之一');
});

test('the home screen opens with no authorization of any kind', () => {
  const harness = createHarness();
  harness.loadApp();
  const home = harness.loadPage('pages/home/home.js');
  harness.invoke(home, 'onLoad', {});
  harness.invoke(home, 'onShow');
  assert.ok(home.data.grid.length, '首页没有渲染');
  for (const k of Object.keys(home.data)) {
    assert.doesNotMatch(k, /auth/i, `首页仍带授权状态 ${k}`);
  }
  assert.equal(harness.navigationCalls.length, 0, '首页把用户导走了');
});

test('the browsing path never requests a phone number', () => {
  for (const id of ['home', 'menu', 'detail', 'confirm']) {
    for (const ext of ['js', 'wxml']) {
      const src = read(`pages/${id}/${id}.${ext}`);
      assert.doesNotMatch(src, /getPhoneNumber/, `pages/${id} 在浏览路径上索取手机号`);
      assert.doesNotMatch(src, /申请获取|授权登录/, `pages/${id} 弹出授权提示`);
    }
  }
});

test('the identity screen holds no authorization popup', () => {
  const harness = createHarness();
  harness.loadApp();
  const launch = harness.loadPage('pages/launch/launch.js');
  harness.invoke(launch, 'onLoad', {});
  for (const fn of ['openAuth', 'closeAuth', 'allowAuth']) {
    assert.equal(typeof launch[fn], 'undefined', `身份选择页仍有 ${fn}`);
  }
  assert.equal(Object.hasOwn(launch.data, 'auth'), false);
  assert.doesNotMatch(read('pages/launch/launch.wxml'), /auth-card|auth-mask/);
  assert.doesNotMatch(read('pages/launch/launch.wxss'), /\.auth-/);
});

test('the user entry navigates straight to the home screen', () => {
  const harness = createHarness();
  harness.loadApp();
  const launch = harness.loadPage('pages/launch/launch.js');
  harness.invoke(launch, 'onLoad', {});
  launch.go({ currentTarget: { dataset: { to: 'home' } } });
  assert.match(harness.navigationCalls.at(-1).url, /pages\/home\/home/);
});

test('the merchant entry still works from the identity screen', () => {
  const harness = createHarness();
  harness.loadApp();
  const launch = harness.loadPage('pages/launch/launch.js');
  harness.invoke(launch, 'onLoad', {});
  launch.go({ currentTarget: { dataset: { to: 'admin-orders' } } });
  assert.match(harness.navigationCalls.at(-1).url, /admin-orders/);
});

test('every handler the identity template binds actually exists', () => {
  const harness = createHarness();
  harness.loadApp();
  const launch = harness.loadPage('pages/launch/launch.js');
  harness.invoke(launch, 'onLoad', {});
  for (const m of read('pages/launch/launch.wxml').matchAll(/bindtap="(\w+)"/g)) {
    assert.equal(typeof launch[m[1]], 'function', `模板绑定了不存在的 ${m[1]}`);
  }
  // 品牌选择页随 §0.2 删除，nav.toBrand 早已不存在
  assert.equal(typeof require('../utils/util.js').nav.toBrand, 'undefined');
});

test('the merchant path is still reachable through the profile screen', () => {
  const harness = createHarness();
  harness.loadApp();
  const profile = harness.loadPage('pages/profile/profile.js');
  harness.invoke(profile, 'onLoad', {});
  profile.reset();
  assert.match(harness.navigationCalls.at(-1).url, /pages\/launch\/launch/,
    '身份选择页失去了唯一入口');
});

test('the home screen offers no shortcut to the identity screen', () => {
  const wxml = read('pages/home/home.wxml');
  const nav = wxml.slice(wxml.indexOf('<navbar'), wxml.indexOf('>', wxml.indexOf('<navbar')) + 1);
  assert.doesNotMatch(nav, /\bexit\b/, '首页导航栏仍提供回身份选择页的入口');
});
