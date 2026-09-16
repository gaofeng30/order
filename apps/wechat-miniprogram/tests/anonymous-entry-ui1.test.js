const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

const launchRoot = path.join(miniprogramRoot, 'pages/launch/launch');
const read = extension => fs.readFileSync(`${launchRoot}.${extension}`, 'utf8');

test('user card directly enters the existing anonymous home route', () => {
  const wxml = read('wxml');
  assert.match(
    wxml,
    /class="id-card primary fade-up"[^>]*data-to="home"[^>]*bindtap="go"/,
  );

  const harness = createHarness();
  harness.loadApp();
  const page = harness.loadPage('pages/launch/launch.js');

  assert.equal(page.data.auth, undefined);
  assert.equal(page.openAuth, undefined);
  assert.equal(page.closeAuth, undefined);
  assert.equal(page.allowAuth, undefined);

  page.go({ currentTarget: { dataset: { to: 'home' } } });
  assert.deepEqual(harness.navigationCalls, [
    { type: 'navigateTo', url: '/pages/home/home', delta: undefined },
  ]);
});

test('launch page contains no interactive profile or phone authorization surface', () => {
  const source = ['js', 'wxml', 'wxss'].map(read).join('\n');
  for (const retired of [
    'openAuth', 'closeAuth', 'allowAuth', 'auth-mask', 'auth-card',
    '微信授权登录', '申请获取并使用你的', '昵称、头像与手机号',
  ]) {
    assert.equal(source.includes(retired), false, `retired launch authorization remains: ${retired}`);
  }
  for (const forbidden of [
    ['get', 'PhoneNumber'].join(''),
    ['get', 'UserProfile'].join(''),
    ['set', 'Storage'].join(''),
    'wxIcon',
    'WX_SVG',
  ]) {
    assert.equal(source.includes(forbidden), false, `forbidden launch API or state remains: ${forbidden}`);
  }
});

test('launch brand and both existing entry cards remain declared', () => {
  const wxml = read('wxml');
  assert.match(wxml, /绥安食品/);
  assert.match(wxml, />用户端</);
  assert.match(wxml, />商户端</);
  assert.match(wxml, /data-to="admin-orders"[^>]*bindtap="go"/);
});
