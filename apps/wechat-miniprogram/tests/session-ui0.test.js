const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');

const accessToken = 'A'.repeat(43);

function sessionResponse(token = accessToken) {
  return {
    statusCode: 201,
    data: {
      access_token: token,
      token_type: 'Bearer',
      expires_at: '2999-08-24T08:00:00Z',
    },
  };
}

test('ready runtime endpoint exchanges one fresh code at the resolved origin', async () => {
  const harness = createHarness({
    logins: [{ code: 'fresh-develop-code' }],
    requests: [sessionResponse()],
  });
  global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'develop' } });

  const app = harness.loadApp();
  assert.deepEqual(app.globalData.session, { state: 'loading', accessToken: '', expiresAt: '' });
  assert.equal(harness.loginCalls.length, 1);

  await harness.flush();

  assert.equal(harness.requestCalls.length, 1);
  assert.equal(harness.requestCalls[0].url, 'http://127.0.0.1:8080/api/v1/auth/miniprogram/session');
  assert.equal(harness.requestCalls[0].method, 'POST');
  assert.deepEqual(harness.requestCalls[0].data, { code: 'fresh-develop-code' });
  assert.deepEqual(app.globalData.session, {
    state: 'ready', accessToken, expiresAt: '2999-08-24T08:00:00Z',
  });
  assert.equal(Object.hasOwn(global.wx, 'setStorageSync'), false);
  assert.equal(Object.hasOwn(global.wx, 'setStorage'), false);
});

test('non-ready runtime endpoint performs zero login and zero request', async t => {
  for (const envVersion of ['trial', 'release']) {
    await t.test(envVersion, async () => {
      const harness = createHarness({ logins: [{ code: 'must-not-be-used' }] });
      global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion } });

      const app = harness.loadApp();
      await harness.flush();

      assert.deepEqual(app.globalData.session, { state: 'error', accessToken: '', expiresAt: '' });
      assert.equal(harness.loginCalls.length, 0);
      assert.equal(harness.requestCalls.length, 0);
    });
  }
});

test('login failure and blank code fail closed without request or automatic retry', async t => {
  for (const login of [{ networkError: true }, { code: '   ' }]) {
    await t.test(JSON.stringify(login), async () => {
      const harness = createHarness({ logins: [login] });
      global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'develop' } });

      const app = harness.loadApp();
      await harness.flush();
      await harness.flush();

      assert.deepEqual(app.globalData.session, { state: 'error', accessToken: '', expiresAt: '' });
      assert.equal(harness.loginCalls.length, 1);
      assert.equal(harness.requestCalls.length, 0);
    });
  }
});

test('rejected or malformed exchange clears credentials without replay', async t => {
  const cases = [
    { statusCode: 503, data: {} },
    { statusCode: 201, data: { access_token: accessToken, token_type: 'bearer', expires_at: '2999-08-24T08:00:00Z' } },
    { statusCode: 201, data: Object.assign({}, sessionResponse().data, { user_id: 1 }) },
  ];
  for (const response of cases) {
    await t.test(String(response.statusCode) + JSON.stringify(response.data), async () => {
      const harness = createHarness({ logins: [{ code: 'single-code' }], requests: [response] });
      global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'develop' } });

      const app = harness.loadApp();
      await harness.flush();
      await harness.flush();

      assert.deepEqual(app.globalData.session, { state: 'error', accessToken: '', expiresAt: '' });
      assert.equal(harness.loginCalls.length, 1);
      assert.equal(harness.requestCalls.length, 1);
    });
  }
});

test('explicit retry uses a new login code and replaces only the failed runtime session', async () => {
  const retryToken = 'B'.repeat(43);
  const harness = createHarness({
    logins: [{ networkError: true }, { code: 'retry-code' }],
    requests: [sessionResponse(retryToken)],
  });
  global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'develop' } });

  const app = harness.loadApp();
  await harness.flush();
  assert.equal(app.globalData.session.state, 'error');

  await app.startSession();

  assert.equal(harness.loginCalls.length, 2);
  assert.equal(harness.requestCalls.length, 1);
  assert.deepEqual(harness.requestCalls[0].data, { code: 'retry-code' });
  assert.deepEqual(app.globalData.session, {
    state: 'ready', accessToken: retryToken, expiresAt: '2999-08-24T08:00:00Z',
  });
});
