const assert = require('node:assert/strict');
const test = require('node:test');

const { createHarness } = require('./page-harness.js');
const { isRuntimeOrigin, resolveRuntimeEndpoint } = require('../utils/runtimeEndpoint.js');

test('deployment config keeps only the develop loopback ready', () => {
  assert.deepEqual(require('../utils/runtimeEndpointConfig.js'), {
    develop: 'http://127.0.0.1:8080',
    trial: '',
    release: '',
  });
});

test('unconfigured trial and release cold launches expose endpoint error without login or request', async t => {
  for (const envVersion of ['trial', 'release']) {
    await t.test(envVersion, async () => {
      const harness = createHarness();
      const loginCalls = [];
      global.wx.login = request => { loginCalls.push(request); };
      global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion } });

      const app = harness.loadApp();

      assert.deepEqual(app.globalData.runtimeEndpoint, {
        state: 'error',
        envVersion,
        origin: '',
        errorCode: 'RUNTIME_ENDPOINT_UNCONFIGURED',
      });
      assert.equal(app.globalData.apiBaseUrl, '');
      assert.equal(loginCalls.length, 0);
      assert.equal(harness.requestCalls.length, 0);
      await harness.flush();
      assert.equal(loginCalls.length, 0);
      assert.equal(harness.requestCalls.length, 0);
    });
  }
});

test('unknown or invalid runtime endpoint never triggers login or request on cold launch', async t => {
  const cases = [
    ['unknown envVersion',
      () => ({ miniProgram: { envVersion: 'unknown' } }),
      {},
      'unknown',
      'RUNTIME_ENDPOINT_ENV_UNSUPPORTED'],
    ['account-info failure',
      () => { throw new Error('provider failed'); },
      {},
      'unknown',
      'RUNTIME_ENDPOINT_ENV_UNSUPPORTED'],
    ['invalid trial origin',
      () => ({ miniProgram: { envVersion: 'trial' } }),
      { trial: 'http://trial-api.example.test' },
      'trial',
      'RUNTIME_ENDPOINT_INVALID'],
    ['remote develop origin',
      () => ({ miniProgram: { envVersion: 'develop' } }),
      { develop: 'http://api.example.test' },
      'develop',
      'RUNTIME_ENDPOINT_INVALID'],
  ];

  for (const [name, accountInfo, configOverride, envVersion, errorCode] of cases) {
    await t.test(name, async () => {
      const harness = createHarness();
      const loginCalls = [];
      global.wx.login = request => { loginCalls.push(request); };
      global.wx.getAccountInfoSync = accountInfo;
      Object.assign(require('../utils/runtimeEndpointConfig.js'), configOverride);

      const app = harness.loadApp();

      assert.deepEqual(app.globalData.runtimeEndpoint, {
        state: 'error', envVersion, origin: '', errorCode,
      });
      assert.equal(app.globalData.apiBaseUrl, '');
      assert.equal(loginCalls.length, 0);
      assert.equal(harness.requestCalls.length, 0);
      await harness.flush();
      assert.equal(loginCalls.length, 0);
      assert.equal(harness.requestCalls.length, 0);
    });
  }
});

test('account info selects develop, trial and release without allowing unknown fallback', async t => {
  const deploymentConfig = {
    develop: 'http://127.0.0.1:8080',
    trial: 'https://trial-api.example.test',
    release: 'https://release-api.example.test/',
  };

  await t.test('missing API is the explicit develop seam', () => {
    assert.deepEqual(resolveRuntimeEndpoint({}, deploymentConfig), {
      state: 'ready',
      envVersion: 'develop',
      origin: 'http://127.0.0.1:8080',
      errorCode: '',
    });
  });

  for (const [envVersion, origin] of [
    ['develop', 'http://127.0.0.1:8080'],
    ['trial', 'https://trial-api.example.test'],
    ['release', 'https://release-api.example.test'],
  ]) {
    await t.test(envVersion, () => {
      const wxApi = { getAccountInfoSync: () => ({ miniProgram: { envVersion } }) };
      assert.deepEqual(resolveRuntimeEndpoint(wxApi, deploymentConfig), {
        state: 'ready',
        envVersion,
        origin,
        errorCode: '',
      });
    });
  }

  for (const [name, wxApi, envVersion] of [
    ['unknown value', { getAccountInfoSync: () => ({ miniProgram: { envVersion: 'unknown' } }) }, 'unknown'],
    ['provider error', { getAccountInfoSync: () => { throw new Error('provider failed'); } }, 'unknown'],
  ]) {
    await t.test(name, () => {
      assert.deepEqual(resolveRuntimeEndpoint(wxApi, deploymentConfig), {
        state: 'error',
        envVersion,
        origin: '',
        errorCode: 'RUNTIME_ENDPOINT_ENV_UNSUPPORTED',
      });
    });
  }
});

test('trial and release fail closed unless configured with a deployable HTTPS origin', async t => {
  const invalidCases = [
    ['HTTP trial', 'trial', 'http://trial-api.example.test'],
    ['IPv4 loopback', 'trial', 'https://127.0.0.1:8443'],
    ['hexadecimal IPv4 loopback', 'trial', 'https://0x7f.0.0.1'],
    ['mixed hexadecimal IPv4 loopback', 'release', 'https://127.0x0.0.1'],
    ['octal IPv4 loopback', 'release', 'https://0177.0.0.1'],
    ['integer IPv4 loopback', 'trial', 'https://2130706433'],
    ['short IPv4 loopback', 'trial', 'https://127.1'],
    ['IPv6 loopback', 'release', 'https://[::1]'],
    ['localhost', 'release', 'https://localhost'],
    ['localhost subdomain', 'trial', 'https://foo.localhost'],
    ['path', 'release', 'https://api.example.test/v1'],
    ['query', 'release', 'https://api.example.test?env=release'],
    ['fragment', 'release', 'https://api.example.test#release'],
    ['userinfo', 'release', 'https://user@api.example.test'],
    ['whitespace', 'release', ' https://api.example.test'],
    ['invalid host', 'release', 'https://-api.example.test'],
    ['invalid port', 'release', 'https://api.example.test:70000'],
  ];

  for (const [name, envVersion, configured] of invalidCases) {
    await t.test(name, () => {
      const wxApi = { getAccountInfoSync: () => ({ miniProgram: { envVersion } }) };
      assert.deepEqual(resolveRuntimeEndpoint(wxApi, { [envVersion]: configured }), {
        state: 'error',
        envVersion,
        origin: '',
        errorCode: 'RUNTIME_ENDPOINT_INVALID',
      });
      assert.equal(isRuntimeOrigin(envVersion, configured), false);
    });
  }

  await t.test('single trailing slash is normalized away', () => {
    const wxApi = { getAccountInfoSync: () => ({ miniProgram: { envVersion: 'trial' } }) };
    assert.deepEqual(resolveRuntimeEndpoint(wxApi, { trial: 'https://API.Example.Test:8443/' }), {
      state: 'ready',
      envVersion: 'trial',
      origin: 'https://api.example.test:8443',
      errorCode: '',
    });
    assert.equal(isRuntimeOrigin('trial', 'https://api.example.test:8443'), true);
  });
});

test('develop accepts only its explicit local loopback origin', async t => {
  const wxApi = { getAccountInfoSync: () => ({ miniProgram: { envVersion: 'develop' } }) };
  assert.deepEqual(resolveRuntimeEndpoint(wxApi, { develop: 'http://127.0.0.1:8080/' }), {
    state: 'ready',
    envVersion: 'develop',
    origin: 'http://127.0.0.1:8080',
    errorCode: '',
  });
  assert.equal(isRuntimeOrigin('develop', 'http://127.0.0.1:8080'), true);

  for (const configured of [
    'ftp://127.0.0.1:8080',
    'http://127.0.0.1:8080/v1',
    'http://api.example.test',
    'https://api.example.test',
  ]) {
    await t.test(configured, () => {
      assert.deepEqual(resolveRuntimeEndpoint(wxApi, { develop: configured }), {
        state: 'error',
        envVersion: 'develop',
        origin: '',
        errorCode: 'RUNTIME_ENDPOINT_INVALID',
      });
      assert.equal(isRuntimeOrigin('develop', configured), false);
    });
  }
});

test('endpoint error keeps catalog browsing retryable without login, request or mock fallback', async () => {
  const harness = createHarness();
  const loginCalls = [];
  global.wx.login = request => { loginCalls.push(request); };
  global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'trial' } });
  const app = harness.loadApp();
  const page = harness.loadPage('pages/menu/menu.js');

  const first = harness.invoke(page, 'onShow');
  assert.equal(page.data.listState, 'loading');
  await first;
  assert.equal(page.data.listState, 'error');
  assert.deepEqual(page.data.groups, []);
  assert.equal(Object.hasOwn(app.globalData, 'menu'), false);
  assert.equal(loginCalls.length, 0);
  assert.equal(harness.requestCalls.length, 0);

  const retry = page.retryCatalog();
  assert.equal(page.data.listState, 'loading');
  await retry;
  assert.equal(page.data.listState, 'error');
  assert.deepEqual(page.data.groups, []);
  assert.equal(loginCalls.length, 0);
  assert.equal(harness.requestCalls.length, 0);
});

test('catalog consumer rejects forged or mismatched ready endpoints before request', async t => {
  for (const [name, runtimeEndpoint, apiBaseUrl] of [
    ['develop remote HTTP', {
      state: 'ready', envVersion: 'develop', origin: 'http://api.example.test', errorCode: '',
    }, 'http://api.example.test'],
    ['trial HTTP', {
      state: 'ready', envVersion: 'trial', origin: 'http://api.example.test', errorCode: '',
    }, 'http://api.example.test'],
    ['origin mismatch', {
      state: 'ready', envVersion: 'trial', origin: 'https://api.example.test', errorCode: '',
    }, 'https://other.example.test'],
  ]) {
    await t.test(name, async () => {
      const harness = createHarness();
      global.wx.getAccountInfoSync = () => ({ miniProgram: { envVersion: 'trial' } });
      const app = harness.loadApp();
      app.globalData.runtimeEndpoint = runtimeEndpoint;
      app.globalData.apiBaseUrl = apiBaseUrl;
      const catalogApi = require('../utils/catalogApi.js');

      await assert.rejects(catalogApi.listCatalog(), error => (
        error && error.code === 'CATALOG_UNAVAILABLE'
      ));
      assert.equal(harness.requestCalls.length, 0);
    });
  }
});
