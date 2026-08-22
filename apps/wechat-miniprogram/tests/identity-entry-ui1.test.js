const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const { createHarness, miniprogramRoot } = require('./page-harness.js');

// 0818 PRD §4.4（2026-08-22 决策）：入口为身份选择页；商户端入口触发微信手机号授权。
// §14：用户能免手机号浏览，启动时不弹手机号授权。
const read = rel => fs.readFileSync(path.join(miniprogramRoot, rel), 'utf8');

function launch() {
  const harness = createHarness();
  harness.loadApp();
  const page = harness.loadPage('pages/launch/launch.js');
  harness.invoke(page, 'onLoad', {});
  return { harness, page };
}
const HANDLER = () => read('pages/launch/launch.wxml').match(/bindgetphonenumber="(\w+)"/)[1];

test('the entry page is the identity screen and home is still reachable', () => {
  const app = JSON.parse(read('app.json'));
  assert.equal(app.pages[0], 'pages/launch/launch');
  assert.ok(app.pages.includes('pages/home/home'));
});

test('the user entry browses with no identity request at all', () => {
  const { harness, page } = launch();
  page.go({ currentTarget: { dataset: { to: 'home' } } });
  assert.match(harness.navigationCalls.at(-1).url, /pages\/home\/home/);
  const wxml = read('pages/launch/launch.wxml');
  const userCard = wxml.slice(Math.max(0, wxml.indexOf('用户端') - 400), wxml.indexOf('用户端'));
  assert.doesNotMatch(userCard, /getPhoneNumber|openAuth/, '用户端一侧仍在索取身份');
});

test('the merchant entry is the real wechat control, not a drawn dialog', () => {
  const wxml = read('pages/launch/launch.wxml');
  assert.match(wxml, /<button[^>]*open-type="getPhoneNumber"/s, '商户端入口不是微信授权控件');
  assert.doesNotMatch(wxml, /auth-mask|auth-card/, '自绘授权弹层回来了');
  assert.doesNotMatch(read('pages/launch/launch.wxss'), /\.auth-/, '自绘弹层样式回来了');
  const { page } = launch();
  assert.equal(typeof page[HANDLER()], 'function');
});

test('declining keeps the user on the page and is not an error', () => {
  const { harness, page } = launch();
  page[HANDLER()].call(page, { detail: { errMsg: 'getPhoneNumber:fail user deny' } });
  assert.ok(!harness.navigationCalls.some(c => String(c.url).includes('admin-orders')),
    '拒绝授权后仍进了商户端');
  assert.ok(page.data.hint, '拒绝后没有任何说明');
  assert.match(page.data.hint, /商户端/);
  assert.doesNotMatch(page.data.hint, /失败|错误/, '把合法的拒绝渲染成了失败');
});

test('allowing enters the merchant end and says the check has not happened', () => {
  const said = [];
  const harness = createHarness();
  harness.loadApp();
  const realToast = global.wx.showToast;
  global.wx.showToast = o => said.push(o && o.title);
  try {
    const page = harness.loadPage('pages/launch/launch.js');
    harness.invoke(page, 'onLoad', {});
    page[HANDLER()].call(page, { detail: { errMsg: 'getPhoneNumber:ok', code: 'c', encryptedData: 'e' } });
    assert.ok(harness.navigationCalls.some(c => String(c.url).includes('admin-orders')), '授权后没进商户端');
    const text = said.join(' ');
    assert.match(text, /服务端/, '没有告知校验仍依赖服务端');
    assert.doesNotMatch(text, /验证通过|身份已确认/, '声称了一个并未发生的校验');
  } finally {
    global.wx.showToast = realToast;
  }
});

test('the front end implements no role check of its own', () => {
  for (const rel of ['pages/launch/launch.js', 'app.js', 'utils/util.js']) {
    assert.doesNotMatch(read(rel), /isMerchant|hasMerchantRole|checkRole|verifyMerchant/,
      `${rel} 自行实现了角色判定；鉴权属服务端（§4.4 末条）`);
  }
});
