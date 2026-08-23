import { randomInt, randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolRoot, '../..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || toolRoot;
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || 'http://127.0.0.1:8080');
const runID = `mini-boundaries-${Date.now()}-${randomUUID().slice(0, 8)}`;
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = existsSync(browserPath)
  ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const caseIDs = ['BE-01', 'BE-02', 'BE-03', 'BE-04', 'BE-05', 'BE-06', 'BE-22', 'BE-23', 'BE-24', 'BE-25', 'BE-26'];
const receiptPath = process.env.ORDER_BOUNDARIES_RECEIPT_PATH
  ? path.resolve(process.env.ORDER_BOUNDARIES_RECEIPT_PATH)
  : path.join(repositoryRoot, '.scratch/mini-user-boundaries-ui1/receipt.json');

if (!browserVersion) throw new Error('locked Chromium is missing; reuse the configured MINIPROGRAM_UI_DEPS cache');

const state = {
  pcToken: '', baselineSettings: null, settingsDirty: false,
  categoryID: '', productIDs: [], staffID: '', primaryStaff: null, primaryStaffDirty: false,
  requests: [], cleanup: [],
};
let proxy;
let setup;
let exitCode = 1;
let executionFailure = '';

try {
  state.pcToken = await acquirePCSession(apiOrigin);
  state.baselineSettings = await requestJSON('GET', '/api/v1/admin/settings', { bearer: state.pcToken });
  const dates = state.baselineSettings.service_dates.map(item => ({ date: item.date, status: 'open' }));
  if (dates.length !== 2 || !dates.every(item => /^\d{4}-\d{2}-\d{2}$/.test(item.date))) {
    throw new Error('admin settings did not expose exactly today and tomorrow; refusing non-restorable setup');
  }
  const temporarySettings = {
    store_status: 'open',
    pickup_point: state.baselineSettings.pickup_point,
    notice: state.baselineSettings.notice,
    pickup_step_min: 30,
    meal_periods: [
      { code: 'lunch', name: '午餐', cutoff_time: '00:00', pickup_from: '11:30', pickup_to: '11:30' },
      { code: 'dinner', name: '晚餐', cutoff_time: '23:59', pickup_from: '23:59', pickup_to: '23:59' },
    ],
    service_dates: dates,
  };
  state.settingsDirty = true;
  await requestJSON('PUT', '/api/v1/admin/settings', {
    bearer: state.pcToken, key: newKey('settings'), body: temporarySettings,
  });

  const primaryStaffList = payloadOf(await requestJSON('GET', '/api/v1/admin/staff-whitelist?q=13800000000', {
    bearer: state.pcToken,
  }), 'staff');
  if (!Array.isArray(primaryStaffList) || primaryStaffList.length > 1) {
    throw new Error('local primary phone staff baseline was ambiguous');
  }
  if (primaryStaffList.length === 1) {
    const existing = primaryStaffList[0];
    state.primaryStaff = { id: exactID(existing.id, 'local primary staff'), enabled: existing.enabled === true };
    if (state.primaryStaff.enabled) {
      state.primaryStaffDirty = true;
      await requestJSON('PUT', `/api/v1/admin/staff-whitelist/${state.primaryStaff.id}`, {
        bearer: state.pcToken, key: newKey('disable-primary-staff'), body: { enabled: false },
      });
    }
  }

  const category = payloadOf(await requestJSON('POST', '/api/v1/admin/categories', {
    bearer: state.pcToken, key: newKey('category'), body: { name: `UI1边界-${runID}` }, expected: 201,
  }), 'category');
  state.categoryID = exactID(category.id, 'category');
  const productSpecs = [
    { key: 'all', name: `UI1全天-${runID}`, price: 1234 },
    { key: 'lunch', name: `UI1午餐-${runID}`, price: 2345 },
    { key: 'dinner', name: `UI1晚餐-${runID}`, price: 3456 },
  ];
  const products = {};
  for (const item of productSpecs) {
    const product = payloadOf(await requestJSON('POST', '/api/v1/admin/products', {
      bearer: state.pcToken,
      key: newKey(`product-${item.key}`),
      expected: 201,
      body: {
        name: item.name, price_cents: item.price, category_id: state.categoryID,
        meal_period: item.key, description: `UI1 boundary ${item.key}`, images: [],
      },
    }), 'product');
    const id = exactID(product.id, `product ${item.key}`);
    state.productIDs.push(id);
    products[item.key] = { id, name: item.name, price_cents: item.price };
  }

  const staffPhone = `188${String(randomInt(0, 100000000)).padStart(8, '0')}`;
  const staffName = `UI1员工-${runID}`;
  const staff = await requestJSON('POST', '/api/v1/admin/staff-whitelist', {
    bearer: state.pcToken, key: newKey('staff'), expected: 201, body: { phone: staffPhone, name: staffName },
  });
  state.staffID = exactID(staff.id, 'staff');

  proxy = await startTransparentProxy(apiOrigin, state.requests);
  setup = {
    pc_token: state.pcToken,
    dates: dates.map(item => item.date),
    baseline_settings: temporarySettings,
    category_id: state.categoryID,
    products,
    staff: { phone: staffPhone, name: staffName },
  };
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_BOUNDARIES_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_BOUNDARIES_RUN_ID = runID;
  process.env.ORDER_BOUNDARIES_SETUP = JSON.stringify(setup);

  process.stdout.write(`MINI_BOUNDARIES_UI1_ENV ${JSON.stringify({
    candidate_sha: candidateSHA,
    browser: browserVersion,
    upstream: apiOrigin,
    proxy: `${proxy.origin} (random loopback)`,
    cases: caseIDs,
  })}\n`);

  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-boundaries.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
} catch (error) {
  executionFailure = safeMessage(error);
  exitCode = 1;
} finally {
  if (proxy) await proxy.close().catch(error => state.cleanup.push({ target: 'proxy', ok: false, error: safeMessage(error) }));
  await cleanup();
}

const passed = exitCode === 0 && !executionFailure && state.cleanup.every(item => item.ok);
const receipt = {
  schema: 'order.mini-user-boundaries.ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED',
  upstream: apiOrigin,
  cases: caseIDs,
  case_evidence: Object.fromEntries(caseIDs.map(caseID => [caseID,
    ['BE-22', 'BE-26'].includes(caseID) ? 'L3_RENDERED_SUPPORTING_PROJECTION' : 'L3_LOCAL_COMPOSED'])),
  status: passed ? 'PASS' : 'FAIL',
  requests: state.requests,
  cleanup: state.cleanup,
  error: executionFailure || undefined,
};
mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
const receiptDisplay = receiptPath.startsWith(`${repositoryRoot}${path.sep}`)
  ? path.relative(repositoryRoot, receiptPath) : receiptPath;
process.stdout.write(`MINI_BOUNDARIES_UI1_RESULT ${JSON.stringify({
  status: receipt.status,
  candidate_sha: candidateSHA,
  scenarios: caseIDs.length,
  requests: state.requests.length,
  cleanup: state.cleanup,
  receipt: receiptDisplay,
  error: executionFailure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

async function cleanup() {
  if (!state.pcToken) return;
  for (const id of state.productIDs.slice().reverse()) {
    await cleanupRequest('DELETE', `/api/v1/admin/products/${id}`, `product ${id}`);
  }
  if (state.staffID) await cleanupRequest('DELETE', `/api/v1/admin/staff-whitelist/${state.staffID}`, `staff ${state.staffID}`);
  if (state.primaryStaffDirty && state.primaryStaff) {
    try {
      await requestJSON('PUT', `/api/v1/admin/staff-whitelist/${state.primaryStaff.id}`, {
        bearer: state.pcToken, key: newKey('restore-primary-staff'), body: { enabled: state.primaryStaff.enabled },
      });
      state.cleanup.push({ target: `staff ${state.primaryStaff.id} enabled baseline`, ok: true });
    } catch (error) {
      state.cleanup.push({ target: `staff ${state.primaryStaff.id} enabled baseline`, ok: false, error: safeMessage(error) });
    }
  }
  if (state.categoryID) await cleanupRequest('DELETE', `/api/v1/admin/categories/${state.categoryID}`, `category ${state.categoryID}`);
  if (state.settingsDirty && state.baselineSettings) {
    try {
      await requestJSON('PUT', '/api/v1/admin/settings', {
        bearer: state.pcToken, key: newKey('restore-settings'), body: settingsWrite(state.baselineSettings),
      });
      const reread = await requestJSON('GET', '/api/v1/admin/settings', { bearer: state.pcToken });
      if (JSON.stringify(settingsWrite(reread)) !== JSON.stringify(settingsWrite(state.baselineSettings))) {
        throw new Error('settings restore reread differs from baseline');
      }
      state.cleanup.push({ target: 'settings baseline', ok: true });
    } catch (error) {
      state.cleanup.push({ target: 'settings baseline', ok: false, error: safeMessage(error) });
    }
  }
}

async function cleanupRequest(method, pathname, target) {
  try {
    await requestJSON(method, pathname, { bearer: state.pcToken, key: newKey('cleanup'), expected: [200, 404] });
    state.cleanup.push({ target, ok: true });
  } catch (error) {
    state.cleanup.push({ target, ok: false, error: safeMessage(error) });
  }
}

function settingsWrite(value) {
  return {
    store_status: value.store_status,
    pickup_point: value.pickup_point,
    notice: value.notice,
    pickup_step_min: value.pickup_step_min,
    meal_periods: value.meal_periods,
    service_dates: value.service_dates,
  };
}

async function acquirePCSession(origin) {
  const mini = await json(origin, 'POST', '/api/v1/auth/miniprogram/session', { body: { code: `${runID}-owner-session` }, expected: 201 });
  const bearer = exactToken(mini.access_token, 'mini session');
  await json(origin, 'POST', '/api/v1/me/bind-phone', { bearer, body: { code: `${runID}-owner-phone` } });
  const login = await json(origin, 'POST', '/api/v1/admin/auth/qrcode', { body: {}, expected: 201 });
  const qr = new URL(exactString(login.qr_payload, 'qr_payload'));
  await json(origin, 'POST', '/api/v1/me/admin-login/approve', {
    bearer,
    body: {
      login_id: exactString(login.login_id, 'login_id'),
      approval_secret: exactString(qr.searchParams.get('approval_secret'), 'approval_secret'),
      code: `${runID}-owner-approval`,
    },
  });
  const poll = await json(origin, 'POST', '/api/v1/admin/auth/poll', {
    body: { login_id: login.login_id, poll_secret: exactString(login.poll_secret, 'poll_secret') },
  });
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session');
}

function requestJSON(method, pathname, options = {}) {
  return json(apiOrigin, method, pathname, options).then(body => {
    state.requests.push({ phase: 'setup-cleanup', method, path: pathname, status: options.expected || 200 });
    return body;
  });
}

async function json(origin, method, pathname, options = {}) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  const response = await fetch(`${origin}${pathname}`, {
    method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
  });
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  const raw = await response.text();
  let body = {};
  if (raw) {
    try { body = JSON.parse(raw); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  }
  if (!expected.includes(response.status)) {
    const code = body && body.error && body.error.code || 'UNKNOWN';
    throw new Error(`${method} ${pathname} returned ${response.status}/${code}`);
  }
  return body;
}

async function startTransparentProxy(origin, observations) {
  const target = new URL(origin);
  const server = http.createServer((request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, corsHeaders());
      response.end();
      return;
    }
    if (request.headers['x-boundary-force-status'] === '503') {
      observations.push({ phase: 'browser', method: request.method, path: request.url, status: 503, injected_transport_failure: true });
      response.writeHead(503, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
      response.end(JSON.stringify({ error: { code: 'BOUNDARY_PROVIDER_UNAVAILABLE' } }));
      return;
    }
    const headers = Object.assign({}, request.headers, { host: target.host });
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname, port: target.port, method: request.method, path: request.url, headers,
    }, upstreamResponse => {
      observations.push({
        phase: 'browser', method: request.method, path: request.url,
        status: upstreamResponse.statusCode, request_id: upstreamResponse.headers['x-request-id'] || '',
      });
      const responseHeaders = Object.assign({}, upstreamResponse.headers, corsHeaders());
      delete responseHeaders.connection;
      response.writeHead(upstreamResponse.statusCode || 502, responseHeaders);
      upstreamResponse.pipe(response);
    });
    upstream.on('error', error => {
      observations.push({ phase: 'browser', method: request.method, path: request.url, status: 0, error: error.code || 'UPSTREAM_ERROR' });
      response.writeHead(502, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
      response.end(JSON.stringify({ error: { code: 'COMPOSED_UPSTREAM_UNAVAILABLE' } }));
    });
    request.pipe(upstream);
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  return {
    origin: `http://127.0.0.1:${address.port}`,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key, x-boundary-force-status',
    'access-control-expose-headers': 'x-request-id',
  };
}

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password
    || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}

function payloadOf(body, key) {
  const value = body && body[key];
  if (!value || typeof value !== 'object') throw new Error(`${key} response is malformed`);
  return value;
}
function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) throw new Error(`${label} id is malformed`);
  return value;
}
function exactString(value, label) {
  if (typeof value !== 'string' || !value.trim()) throw new Error(`${label} is missing`);
  return value;
}
function exactToken(value, label) {
  const token = exactString(value, label);
  if (token.length < 32) throw new Error(`${label} is malformed`);
  return token;
}
function newKey(scope) { return `${scope}-${randomUUID()}`; }
function safeMessage(error) {
  return error && error.message ? String(error.message).replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : 'unknown error';
}
