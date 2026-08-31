const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const SESSION = { statusCode: 201, data: { access_token: 'user-token', token_type: 'Bearer', expires_at: '2999-08-25T08:00:00Z' } };

const SUMMARY = {
  id: '301', order_no: 'SA202608250001', state: 'READY_FOR_PICKUP', pickup_date: '2026-08-25',
  pickup_time: '17:30', pickup_point: '北门', pickup_number: '0012', payable_cents: 1800,
  materialized_at: '2026-08-25T08:01:00Z', available_actions: ['SUBSCRIBE_READY'],
};

const DETAIL = Object.assign({}, SUMMARY, {
  contact: { name: '林先生', masked_phone: '+*********5678' }, identity: { kind: 'VISITOR' },
  discount: { rate_percent: 100 },
  items: [{ product_id: '70', name: '红烧肉', quantity: 1, unit_price_cents: 1800, line_total_cents: 1800, flavors: ['少盐'], note: '' }],
  transaction_id: 'wx-tx', paid_at: '2026-08-25T08:01:00Z', redemption_token: 'opaque-redemption-token',
  transition_times: {}, notification_options: ['READY'], order_note: '',
});

function readyHarness(requests, native) {
  const harness = createHarness(Object.assign({
    logins: [{ code: 'code' }], requests: [SESSION].concat(requests || []),
  }, native || {}));
  const app = harness.loadApp();
  return { app, harness };
}

test('PAGE-U08 exposes fixed six filters, pages server results, and has no refunding filter', async () => {
  const { harness } = readyHarness([
    { statusCode: 200, data: { orders: [SUMMARY], next_after_id: '301' } },
    { statusCode: 200, data: { orders: [], next_after_id: null } },
    { statusCode: 200, data: { orders: [Object.assign({}, SUMMARY, { id: '302', state: 'REFUNDED' })] } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/orders/orders.js');
  await harness.invoke(page, 'onShow');
  assert.deepEqual(page.data.tabs, ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款']);
  assert.equal(page.data.list[0].displayState, '待取餐');
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20');
  await page.loadMore();
  assert.equal(harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20&after_id=301');
  await page.switchTab({ currentTarget: { dataset: { t: '已退款' } } });
  assert.equal(harness.requestCalls[3].url, 'http://127.0.0.1:8080/api/v1/orders?limit=20&state=REFUNDED');
});

test('PAGE-U07 READY detail shows only server token and records native subscription decision', async () => {
  const { app, harness } = readyHarness([
    { statusCode: 200, data: { order: DETAIL } },
    { statusCode: 200, data: { subscription: { kind: 'READY', decision: 'ACCEPTED', available: true } } },
  ], { subscriptions: [{ 'ready-template': 'accept' }] });
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { READY: 'ready-template' };
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '301' });

  assert.equal(page.data.showQr, true);
  assert.equal(page.data.qrToken, 'opaque-redemption-token');
  assert.equal(page.data.canCancel, false);
  const subscribed = await page.subscribeReady();
  assert.equal(subscribed, true);
  assert.deepEqual(harness.subscribeCalls[0].tmplIds, ['ready-template']);
  assert.deepEqual(harness.requestCalls[2].data, { kind: 'READY', decision: 'ACCEPTED' });
});

test('PAGE-U07 persists a rejected refund subscription before cancelling and becomes REFUNDING from response', async () => {
  const reserved = Object.assign({}, DETAIL, {
    state: 'RESERVED', redemption_token: null, available_actions: ['CANCEL'], notification_options: [],
  });
  const refunding = Object.assign({}, reserved, { state: 'REFUNDING', available_actions: [] });
  const { app, harness } = readyHarness([
    { statusCode: 200, data: { order: reserved } },
    { statusCode: 200, data: { subscription: { kind: 'REFUND_RESULT', decision: 'REJECTED', available: false } } },
    { statusCode: 200, data: { order: refunding, refund: { id: '501', state: 'REQUESTED', amount_cents: 1800 } } },
  ], { subscriptions: [{ 'refund-template': 'reject' }] });
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { REFUND_RESULT: 'refund-template' };
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  assert.equal(page.data.showQr, false);
  assert.equal(page.data.canCancel, true);
  const cancelled = await page.doCancel();
  assert.equal(cancelled, true);
  assert.equal(page.data.o.state, 'REFUNDING');
  assert.deepEqual(harness.subscribeCalls[0].tmplIds, ['refund-template']);
  assert.deepEqual(harness.requestCalls[2].data, { kind: 'REFUND_RESULT', decision: 'REJECTED' });
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^subscription-/);
  assert.match(harness.requestCalls[3].header['Idempotency-Key'], /^cancel-/);
});

test('PAGE-U07 native subscription failure still executes the confirmed cancellation without a consent write', async () => {
  const reserved = Object.assign({}, DETAIL, {
    state: 'RESERVED', redemption_token: null, available_actions: ['CANCEL'], notification_options: [],
  });
  const refunding = Object.assign({}, reserved, { state: 'REFUNDING', available_actions: [] });
  const { app, harness } = readyHarness([
    { statusCode: 200, data: { order: reserved } },
    { statusCode: 200, data: { order: refunding, refund: { id: '502', state: 'REQUESTED', amount_cents: 1800 } } },
  ]);
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { REFUND_RESULT: 'refund-template' };
  const page = harness.loadPage('pages/order-detail/order-detail.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  assert.equal(await page.doCancel(), true);
  assert.equal(page.data.o.state, 'REFUNDING');
  assert.equal(harness.subscribeCalls.length, 1);
  assert.equal(harness.requestCalls.length, 3);
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^cancel-/);
});

test('PAGE-U07 consent persistence failure still cancels, while cancel failure never advances the local order', async () => {
  const reserved = Object.assign({}, DETAIL, {
    state: 'RESERVED', redemption_token: null, available_actions: ['CANCEL'], notification_options: [],
  });
  const refunding = Object.assign({}, reserved, { state: 'REFUNDING', available_actions: [] });
  const first = readyHarness([
    { statusCode: 200, data: { order: reserved } },
    { statusCode: 503, data: { error: { code: 'SUBSCRIPTION_UNAVAILABLE' } } },
    { statusCode: 200, data: { order: refunding, refund: { id: '503', state: 'REQUESTED', amount_cents: 1800 } } },
  ], { subscriptions: [{ 'refund-template': 'accept' }] });
  await first.harness.flush();
  first.app.globalData.subscriptionTemplateIds = { REFUND_RESULT: 'refund-template' };
  const firstPage = first.harness.loadPage('pages/order-detail/order-detail.js');
  await first.harness.invoke(firstPage, 'onLoad', { id: '301' });
  assert.equal(await firstPage.doCancel(), true);
  assert.equal(firstPage.data.o.state, 'REFUNDING');
  assert.equal(first.harness.requestCalls[2].url, 'http://127.0.0.1:8080/api/v1/orders/301/subscriptions');
  assert.equal(first.harness.requestCalls[3].url, 'http://127.0.0.1:8080/api/v1/orders/301/cancel');

  const second = readyHarness([
    { statusCode: 200, data: { order: reserved } },
    { statusCode: 200, data: { subscription: { kind: 'REFUND_RESULT', decision: 'ACCEPTED', available: true } } },
    { statusCode: 503, data: { error: { code: 'REFUND_UNAVAILABLE' } } },
  ], { subscriptions: [{ 'refund-template': 'accept' }] });
  await second.harness.flush();
  second.app.globalData.subscriptionTemplateIds = { REFUND_RESULT: 'refund-template' };
  const secondPage = second.harness.loadPage('pages/order-detail/order-detail.js');
  await second.harness.invoke(secondPage, 'onLoad', { id: '301' });
  assert.equal(await secondPage.doCancel(), false);
  assert.equal(secondPage.data.o.state, 'RESERVED');
  assert.equal(secondPage.data.canceling, false);
});

test('PAGE-U06 reloads the created order then automatically persists one READY acceptance', async () => {
  const { app, harness } = readyHarness([
    { statusCode: 200, data: { order: DETAIL } },
    { statusCode: 200, data: { subscription: { kind: 'READY', decision: 'ACCEPTED', available: true } } },
  ], { subscriptions: [{ 'ready-template': 'accept' }] });
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { READY: 'ready-template' };
  const page = harness.loadPage('pages/result/result.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  await harness.flush();
  assert.equal(page.data.state, 'ready');
  assert.equal(page.data.o.id, '301');
  assert.equal(harness.requestCalls[1].url, 'http://127.0.0.1:8080/api/v1/orders/301');
  assert.deepEqual(harness.subscribeCalls[0].tmplIds, ['ready-template']);
  assert.deepEqual(harness.requestCalls[2].data, { kind: 'READY', decision: 'ACCEPTED' });
  assert.match(harness.requestCalls[2].header['Idempotency-Key'], /^subscription-/);
  await page.requestReadySubscription();
  assert.equal(harness.subscribeCalls.length, 1);
});

test('PAGE-U06 native subscription failure leaves the confirmed order visible and performs no consent write', async () => {
  const { app, harness } = readyHarness([{ statusCode: 200, data: { order: DETAIL } }]);
  await harness.flush();
  app.globalData.subscriptionTemplateIds = { READY: 'ready-template' };
  const page = harness.loadPage('pages/result/result.js');
  await harness.invoke(page, 'onLoad', { id: '301' });
  await harness.flush();
  assert.equal(page.data.state, 'ready');
  assert.equal(page.data.o.id, '301');
  assert.equal(harness.subscribeCalls.length, 1);
  assert.equal(harness.requestCalls.length, 2);
});

test('PAGE-U09 profile uses server identity, optional cosmetic profile and native customer-service surface', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' },
      extra_phone: { set: false }, pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
      merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [SUMMARY] } },
  ], { profiles: [{ userInfo: { nickName: '用户昵称', avatarUrl: 'https://avatar.example.com/a.png' } }] });
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');
  assert.equal(page.data.identityState, 'ready');
  assert.equal(page.data.phoneMask, '+*********5678');
  assert.equal(page.data.pend, 1);
  assert.equal(page.data.merchantBound, false);
  assert.equal(await page.chooseProfile(), true);
  assert.equal(page.data.nick, '用户昵称');
});

test('PAGE-U09 merchant phone authorization binds through the server and resets to identity selection', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' },
      extra_phone: { set: false }, pricing_identity: { kind: 'VISITOR', rate_percent: 100 },
      merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [] } },
    { statusCode: 200, data: { merchant: { bound: true, role: 'OWNER' } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');
  assert.equal(await page.onMerchantPhone({ detail: { code: 'merchant-code' } }), true);
  assert.equal(page.data.merchantBound, true);
  assert.equal(harness.requestCalls[3].url, 'http://127.0.0.1:8080/api/v1/me/merchant-login');
  assert.deepEqual(harness.requestCalls[3].data, { code: 'merchant-code' });
  assert.deepEqual(harness.navigationCalls.at(-1), { type: 'reLaunch', url: '/pages/launch/launch', delta: undefined });
});

test('PAGE-U09 rejected merchant phone authorization is a zero-write user-side outcome', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: false }, extra_phone: { set: false },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 }, merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [] } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');
  assert.equal(await page.onMerchantPhone({ detail: { errMsg: 'getPhoneNumber:fail user deny' } }), false);
  assert.equal(page.data.merchantBound, false);
  assert.equal(harness.requestCalls.length, 3);
  assert.equal(harness.navigationCalls.length, 0);
});

test('PAGE-U09 merchant identity server failure remains fail closed in the user side', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' }, extra_phone: { set: false },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 }, merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [] } },
    { statusCode: 503, data: { error: { code: 'MERCHANT_IDENTITY_UNAVAILABLE' } } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');
  assert.equal(await page.onMerchantPhone({ detail: { code: 'merchant-code' } }), false);
  assert.equal(page.data.merchantBound, false);
  assert.equal(page.data.merchantLoginState, 'error');
  assert.equal(harness.navigationCalls.length, 0);
});

// P0-6：附加手机号表单曾把标题、两个输入和保存挤进同一个 flex 行，标题逐字换行。
// 表单改为默认收起的折叠面板；这里锁住状态机，几何由 UI2 断言。
test('PAGE-U09 extra phone form stays collapsed until opened and re-collapses after a save', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' },
      extra_phone: { set: true, masked_phone: '139****0001', name: '李四' },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 }, merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([
    identity,
    { statusCode: 200, data: { orders: [] } },
    { statusCode: 200, data: {
      extra_phone: { set: true, masked_phone: '139****0002', name: '王五' },
      pricing_identity: { kind: 'STAFF', rate_percent: 88 },
    } },
  ]);
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');

  // 收起态是默认，且不泄露已保存结果。
  assert.equal(page.data.extraOpen, false);
  // 姓名服务端返回明文，可预填；手机号只有脱敏值，不预填。
  assert.equal(page.data.extraName, '李四');
  assert.equal(page.data.extraForm.name, '李四');
  assert.equal(page.data.extraForm.phone, '');

  page.toggleExtra();
  assert.equal(page.data.extraOpen, true);

  page.onExtraInput({ currentTarget: { dataset: { k: 'phone' } }, detail: { value: '+8613900000002' } });
  page.onExtraInput({ currentTarget: { dataset: { k: 'name' } }, detail: { value: '王五' } });
  assert.equal(page.data.extraSavable, true);

  assert.equal(await page.saveExtraPhone(), true);
  assert.equal(page.data.extraState, 'matched');
  // 保存成功后自动收起。
  assert.equal(page.data.extraOpen, false);
});

test('PAGE-U09 extra phone save stays disabled until both factors are present', async () => {
  const identity = {
    statusCode: 200,
    data: { identity: {
      primary_phone: { bound: true, masked_phone: '+*********5678' }, extra_phone: { set: false },
      pricing_identity: { kind: 'VISITOR', rate_percent: 100 }, merchant: { bound: false },
    } },
  };
  const { harness } = readyHarness([identity, { statusCode: 200, data: { orders: [] } }]);
  await harness.flush();
  const page = harness.loadPage('pages/profile/profile.js');
  await harness.invoke(page, 'onShow');

  assert.equal(page.data.extraSavable, false);
  page.onExtraInput({ currentTarget: { dataset: { k: 'phone' } }, detail: { value: '+8613900000002' } });
  // 双要素缺姓名，仍不可保存。
  assert.equal(page.data.extraSavable, false);
  page.onExtraInput({ currentTarget: { dataset: { k: 'name' } }, detail: { value: '王五' } });
  assert.equal(page.data.extraSavable, true);
});

// Node harness 只加载 JS，看不见 WXML/WXSS。这两条锁住结构与样式存在性，
// 因为两处根因分别是「类零定义」和「button 承担布局」，都不在 JS 里。
test('PAGE-U09 profile styles define every class the extra phone panel renders', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  const wxss = fs.readFileSync(path.join(root, 'pages/profile/profile.wxss'), 'utf8');
  for (const selector of ['.prow--extra', '.prow-head', '.extra-panel', '.extra-fields',
    '.extra-in', '.extra-in--phone', '.extra-in--name', '.extra-save', '.prow-tap']) {
    assert.equal(wxss.includes(selector), true, `profile.wxss is missing ${selector}`);
  }
  // 点击层必须真正退出布局流，而不是靠压平原生样式。
  assert.match(wxss, /\.prow-tap\s*\{[^}]*position:\s*absolute/);
  assert.match(wxss, /\.prow--tappable\s*\{[^}]*position:\s*relative/);
});

test('PAGE-U09 interactive rows never let a native button carry the row layout', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  const wxml = fs.readFileSync(path.join(root, 'pages/profile/profile.wxml'), 'utf8');
  // 布局类不得再扣在 button 上。按 class token 精确比对：
  // \bprow\b 会在 prow-tap 里命中，因为连字符也是词边界。
  for (const match of wxml.matchAll(/<button[^>]*class="([^"]*)"/g)) {
    assert.equal(match[1].split(/\s+/).includes('prow'), false,
      'a native button still carries the .prow layout class');
  }
  // 三处能力必须保留：两个 getPhoneNumber、一个 contact。
  assert.equal((wxml.match(/open-type="getPhoneNumber"/g) || []).length, 2);
  assert.equal((wxml.match(/open-type="contact"/g) || []).length, 1);
  // 三处交互行都通过透明点击层承载按钮。
  assert.equal((wxml.match(/class="prow-tap"/g) || []).length, 3);
  // 收起态不得渲染输入框。
  assert.match(wxml, /wx:if="\{\{extraOpen\}\}"/);
});

// 卡片内五行必须风格一致地各带一个 .prow-ico，否则未绑定主手机号的用户
// 会看到一行没有图标的缺口。
test('PAGE-U09 every row in the profile card carries a same-style icon', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  const wxml = fs.readFileSync(path.join(root, 'pages/profile/profile.wxml'), 'utf8');
  assert.equal((wxml.match(/class="prow-ico"/g) || []).length, 5,
    'profile card rows and icons are out of sync');
  // 语义各异，主手机号与附加手机号不撞义。
  for (const name of ['receipt', 'phone', 'user', 'store', 'headset']) {
    assert.equal(wxml.includes(`name="${name}"`), true, `missing row icon: ${name}`);
  }
  // 图标名必须来自既有图标表，不新增资源。
  const icons = require('../utils/icons.js');
  for (const match of wxml.matchAll(/<icon[^>]*name="([a-zA-Z]+)"/g)) {
    assert.equal(Object.prototype.hasOwnProperty.call(icons, match[1]), true,
      `profile.wxml uses an undefined icon: ${match[1]}`);
  }
});

// 首页「服务功能」标题被 .body 的负边距拉进深蓝 hero，深字压深底近乎隐形。
// 删除后 hero 收紧，卡片紧随不重叠。
test('PAGE-U01 home drops the invisible section heading and tightens the hero', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  const wxml = fs.readFileSync(path.join(root, 'pages/home/home.wxml'), 'utf8');
  const wxss = fs.readFileSync(path.join(root, 'pages/home/home.wxss'), 'utf8');

  assert.equal(wxml.includes('服务功能'), false, 'home still renders the 服务功能 heading');
  assert.equal(wxml.includes('sec-h'), false, 'home still renders a section heading row');
  // 负边距曾把首个元素推进 hero；改为正向留白后圆角才完整可见。
  assert.match(wxss, /\.body\s*\{[^}]*margin-top:\s*24rpx/);
  assert.equal(/\.body\s*\{[^}]*margin-top:\s*-/.test(wxss), false,
    'home body still pulls its first child into the hero');
  assert.match(wxss, /\.home-hero\s*\{[^}]*padding:\s*0 36rpx 48rpx/);
});

// 冷启动改为直接进首页后启动页不再露面，logo 因此完全没有曝光位。
test('PAGE-U01 home hero shows the store emblem beside the store name', () => {
  const fs = require('node:fs');
  const path = require('node:path');
  const root = path.resolve(__dirname, '..');
  const wxml = fs.readFileSync(path.join(root, 'pages/home/home.wxml'), 'utf8');
  const wxss = fs.readFileSync(path.join(root, 'pages/home/home.wxss'), 'utf8');

  assert.match(wxml, /class="hero-brand"/);
  assert.match(wxml, /src="\/assets\/emblem\.png"/);
  // 完整展示而非裁切。
  assert.match(wxml, /class="hero-emblem"[^>]*mode="aspectFit"|mode="aspectFit"[^>]*class="hero-emblem"/);
  // 徽标与门店名同行才构成品牌锁定。
  assert.match(wxml, /class="hero-brand"[\s\S]{0,400}class="store-name serif"/);
  for (const selector of ['.hero-brand', '.hero-logo', '.hero-emblem']) {
    assert.equal(wxss.includes(selector), true, `home.wxss is missing ${selector}`);
  }
});
