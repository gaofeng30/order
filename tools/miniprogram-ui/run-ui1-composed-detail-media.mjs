import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { randomUUID } from 'node:crypto';
import { deflateSync } from 'node:zlib';
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
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || '');
const receiptPath = path.resolve(process.env.ORDER_DETAIL_MEDIA_RECEIPT_PATH
  || '/private/tmp/order-mini-detail-media/receipt.json');
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = existsSync(browserPath)
  ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const runID = `detail-media-${Date.now()}-${randomUUID().slice(0, 8)}`;
const cases = ['PAGE-U04', 'BE-34', 'BE-35'];
const state = {
  pcToken: '', miniToken: '', baselineSettings: null, settingsDirty: false,
  categoryID: '', productIDs: [], setupRequests: [], browserRequests: [], cleanup: [],
};
let proxy;
let karmaExit = 1;
let failure = '';

if (process.env.ORDER_DETAIL_MEDIA_FRESH_DB !== 'YES') {
  throw new Error('ORDER_DETAIL_MEDIA_FRESH_DB=YES is required');
}
if (!browserVersion) throw new Error('locked Chromium is missing; reuse MINIPROGRAM_UI_DEPS and PLAYWRIGHT_BROWSERS_PATH=0');

try {
  const setup = await prepareFacts();
  proxy = await startTransparentProxy(apiOrigin, state.browserRequests);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_DETAIL_MEDIA_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_DETAIL_MEDIA_SETUP = JSON.stringify({
    run_id: runID,
    api_origin: proxy.origin,
    pickup: setup.pickup,
    zero: setup.zero,
    single: setup.single,
  });
  process.stdout.write(`MINI_DETAIL_MEDIA_UI1_ENV ${JSON.stringify({
    candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin,
    proxy: `${proxy.origin} (random loopback)`, cases,
  })}\n`);
  const processed = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-detail-media.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  karmaExit = await new Promise((resolve, reject) => {
    const server = new karma.Server(processed, resolve);
    server.start().catch(reject);
  });
  if (karmaExit !== 0) throw new Error(`detail media Chrome Gate exited ${karmaExit}`);
} catch (error) {
  failure = safeMessage(error);
} finally {
  if (proxy) {
    try {
      await proxy.close();
      state.cleanup.push({ target: 'random loopback proxy', ok: true });
    } catch (error) {
      state.cleanup.push({ target: 'random loopback proxy', ok: false, error: safeMessage(error) });
    }
  }
  await cleanupFacts();
}

const passed = karmaExit === 0 && !failure && state.cleanup.length >= 5 && state.cleanup.every(item => item.ok)
  && state.browserRequests.some(item => item.status === 400 && item.path.startsWith('/api/v1/catalog/products/'))
  && state.browserRequests.some(item => item.status === 503 && item.injected_transport_failure === true);
const receipt = {
  schema: 'order.mini-detail-media.ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED',
  database_schema_version: 44,
  cases,
  status: passed ? 'PASS' : 'FAIL',
  case_evidence: {
    'PAGE-U04': 'rendered production detail against private root HTTP/MySQL; missing date/time is 400 before fact load and injected 503 visibly clears stale content with retry',
    'BE-34': 'root facts contain zero images; rendered menu/detail use production ImagePH gradient and serif first character with no invented external src or preview',
    'BE-35': 'root facts contain exactly one image; rendered detail has no swiper/count/second position and preview receives exactly that one public URL',
  },
  setup_requests: state.setupRequests,
  browser_requests: state.browserRequests,
  cleanup: state.cleanup,
  external_only: ['WeChat Developer Tools UI2', 'physical-device UI3'],
  forbidden_claims: ['Chrome/miniprogram-simulate is not WeChat Developer Tools UI2', 'local composed evidence is not submission readiness'],
  error: failure || undefined,
};
mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`MINI_DETAIL_MEDIA_UI1_RESULT ${JSON.stringify({
  status: receipt.status, candidate_sha: candidateSHA, cases: cases.length,
  browser: browserVersion, requests: state.browserRequests.length,
  cleanup: state.cleanup, receipt: receiptPath, error: failure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

async function prepareFacts() {
  const sessions = await acquirePCSession();
  state.pcToken = sessions.pcToken;
  state.miniToken = sessions.miniToken;
  state.baselineSettings = await requestJSON('GET', '/api/v1/admin/settings', { bearer: state.pcToken });
  const today = shanghaiDate(0);
  const tomorrow = shanghaiDate(1);
  const settings = {
    store_status: 'open',
    pickup_point: state.baselineSettings.pickup_point,
    notice: state.baselineSettings.notice,
    pickup_step_min: 30,
    meal_periods: [
      { code: 'lunch', name: '午餐', cutoff_time: '11:00', pickup_from: '11:30', pickup_to: '11:30' },
      { code: 'dinner', name: '晚餐', cutoff_time: '17:00', pickup_from: '17:30', pickup_to: '17:30' },
    ],
    service_dates: [{ date: today, status: 'closed' }, { date: tomorrow, status: 'open' }],
  };
  state.settingsDirty = true;
  const saved = await requestJSON('PUT', '/api/v1/admin/settings', {
    bearer: state.pcToken, key: idempotencyKey('settings'), body: settings,
  });
  if (JSON.stringify(settingsWrite(saved)) !== JSON.stringify(settings)) {
    throw new Error('Admin settings did not preserve the exact media Gate selection');
  }
  const category = payloadOf(await requestJSON('POST', '/api/v1/admin/categories', {
    bearer: state.pcToken, key: idempotencyKey('category'), expected: 201,
    body: { name: `媒体边界-${runID}` },
  }), 'category');
  state.categoryID = exactID(category.id, 'category');
  const zero = await createProduct('零图套餐', []);
  const upload = await uploadImage(solidPNG(38, 112, 188, 255));
  const single = await createProduct('单图套餐', [{ object_key: upload.object_key, sort_order: 0 }]);
  const query = `date=${encodeURIComponent(tomorrow)}&time=11%3A30`;
  const zeroRead = payloadOf(await requestJSON('GET', `/api/v1/catalog/products/${zero.id}?${query}`, {
    bearer: state.miniToken,
  }), 'product');
  const singleRead = payloadOf(await requestJSON('GET', `/api/v1/catalog/products/${single.id}?${query}`, {
    bearer: state.miniToken,
  }), 'product');
  if (!Array.isArray(zeroRead.images) || zeroRead.images.length !== 0) throw new Error('zero-image root fact was not exact');
  if (!Array.isArray(singleRead.images) || singleRead.images.length !== 1 || !singleRead.images[0].url) {
    throw new Error('single-image root fact was not exact');
  }
  return {
    pickup: { date: tomorrow, meal_period: 'lunch', time: '11:30' },
    zero: { id: zeroRead.id, name: zeroRead.name },
    single: { id: singleRead.id, name: singleRead.name },
  };
}

async function acquirePCSession() {
  const session = await requestJSON('POST', '/api/v1/auth/miniprogram/session', {
    expected: 201, body: { code: `${runID}-owner-session` },
  });
  const miniToken = exactToken(session.access_token, 'Mini session');
  await requestJSON('POST', '/api/v1/me/bind-phone', {
    bearer: miniToken, key: idempotencyKey('bind-owner'), body: { code: `${runID}-owner-phone` },
  });
  const login = await requestJSON('POST', '/api/v1/admin/auth/qrcode', { expected: 201, body: {} });
  const qr = new URL(exactString(login.qr_payload, 'qr_payload'));
  await requestJSON('POST', '/api/v1/me/admin-login/approve', {
    bearer: miniToken,
    body: {
      login_id: exactString(login.login_id, 'login_id'),
      approval_secret: exactString(qr.searchParams.get('approval_secret'), 'approval_secret'),
      code: `${runID}-owner-approval`,
    },
  });
  const poll = await requestJSON('POST', '/api/v1/admin/auth/poll', {
    body: { login_id: login.login_id, poll_secret: exactString(login.poll_secret, 'poll_secret') },
  });
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC session was not approved');
  return { miniToken, pcToken: exactToken(poll.session.token, 'PC session') };
}

async function createProduct(name, images) {
  const product = payloadOf(await requestJSON('POST', '/api/v1/admin/products', {
    bearer: state.pcToken, key: idempotencyKey('product'), expected: 201,
    body: {
      name: `${name}-${runID}`, price_cents: 1888, category_id: state.categoryID,
      meal_period: 'all', description: 'PAGE-U04 媒体边界', images,
    },
  }), 'product');
  const id = exactID(product.id, name);
  state.productIDs.push(id);
  return { id, name: product.name };
}

async function uploadImage(bytes) {
  const form = new FormData();
  form.append('file', new Blob([bytes], { type: 'image/png' }), `${randomUUID()}.png`);
  const response = await fetch(`${apiOrigin}/api/v1/upload`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${state.pcToken}`, 'Idempotency-Key': idempotencyKey('upload') },
    body: form,
  });
  const raw = await response.text();
  let body = {};
  if (raw) { try { body = JSON.parse(raw); } catch {} }
  state.setupRequests.push({ phase: 'setup', method: 'POST', path: '/api/v1/upload', status: response.status });
  if (response.status !== 201 || !body.image || !body.image.object_key || !body.image.url) {
    throw new Error(`image upload returned ${response.status}`);
  }
  return body.image;
}

async function cleanupFacts() {
  if (!state.pcToken) return;
  for (const id of state.productIDs.slice().reverse()) {
    await cleanupRequest('DELETE', `/api/v1/admin/products/${id}`, `product ${id}`);
  }
  if (state.categoryID) await cleanupRequest('DELETE', `/api/v1/admin/categories/${state.categoryID}`, `category ${state.categoryID}`);
  if (state.settingsDirty && state.baselineSettings) {
    try {
      await requestJSON('PUT', '/api/v1/admin/settings', {
        bearer: state.pcToken, key: idempotencyKey('restore-settings'), body: settingsWrite(state.baselineSettings),
      });
      const reread = await requestJSON('GET', '/api/v1/admin/settings', { bearer: state.pcToken });
      assertRestoredSettings(reread, state.baselineSettings);
      state.cleanup.push({ target: 'settings baseline fields; dedicated service-date rows drop with fresh schema', ok: true });
    } catch (error) {
      state.cleanup.push({ target: 'settings baseline', ok: false, error: safeMessage(error) });
    }
  }
}

async function cleanupRequest(method, pathname, target) {
  try {
    await requestJSON(method, pathname, {
      bearer: state.pcToken, key: idempotencyKey('cleanup'), expected: [200, 404],
    });
    state.cleanup.push({ target, ok: true });
  } catch (error) {
    state.cleanup.push({ target, ok: false, error: safeMessage(error) });
  }
}

async function requestJSON(method, pathname, options = {}) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
  });
  const raw = await response.text();
  let body = {};
  if (raw) { try { body = JSON.parse(raw); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); } }
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  state.setupRequests.push({ phase: 'setup-cleanup', method, path: pathname, status: response.status });
  if (!expected.includes(response.status)) {
    throw new Error(`${method} ${pathname} returned ${response.status}/${body && body.error && body.error.code || 'UNKNOWN'}`);
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
    const requestURL = new URL(request.url, origin);
    if (request.headers['x-detail-media-force-status'] === '503') {
      observations.push({ method: request.method, path: requestURL.pathname, query: requestURL.search, status: 503, injected_transport_failure: true });
      response.writeHead(503, { ...corsHeaders(), 'content-type': 'application/json' });
      response.end(JSON.stringify({ error: { code: 'CATALOG_UNAVAILABLE' } }));
      return;
    }
    const headers = { ...request.headers, host: target.host };
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname, port: target.port, method: request.method, path: request.url, headers,
    }, incoming => {
      observations.push({
        method: request.method, path: requestURL.pathname, query: requestURL.search,
        status: incoming.statusCode, request_id: incoming.headers['x-request-id'] || '',
      });
      const responseHeaders = { ...incoming.headers, ...corsHeaders() };
      delete responseHeaders.connection;
      response.writeHead(incoming.statusCode || 502, responseHeaders);
      incoming.pipe(response);
    });
    upstream.on('error', error => {
      observations.push({ method: request.method, path: requestURL.pathname, status: 0, error: error.code || 'UPSTREAM_ERROR' });
      response.writeHead(502, { ...corsHeaders(), 'content-type': 'application/json' });
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
    'access-control-allow-headers': 'authorization, content-type, idempotency-key, x-detail-media-force-status',
    'access-control-expose-headers': 'x-request-id',
  };
}

function solidPNG(red, green, blue, alpha) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  const header = Buffer.alloc(13);
  header.writeUInt32BE(2, 0); header.writeUInt32BE(2, 4); header[8] = 8; header[9] = 6;
  const pixel = [red, green, blue, alpha];
  const rows = Buffer.from([0, ...pixel, ...pixel, 0, ...pixel, ...pixel]);
  return Buffer.concat([signature, pngChunk('IHDR', header), pngChunk('IDAT', deflateSync(rows)), pngChunk('IEND', Buffer.alloc(0))]);
}

function pngChunk(kind, data) {
  const type = Buffer.from(kind, 'ascii');
  const length = Buffer.alloc(4); length.writeUInt32BE(data.length);
  const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(Buffer.concat([type, data])));
  return Buffer.concat([length, type, data, crc]);
}

function crc32(data) {
  let value = 0xffffffff;
  for (const byte of data) {
    value ^= byte;
    for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ ((value & 1) ? 0xedb88320 : 0);
  }
  return (value ^ 0xffffffff) >>> 0;
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
function assertRestoredSettings(actual, baseline) {
  const withoutDates = value => {
    const copy = settingsWrite(value);
    delete copy.service_dates;
    return copy;
  };
  if (JSON.stringify(withoutDates(actual)) !== JSON.stringify(withoutDates(baseline))) {
    throw new Error('restorable settings fields differ from baseline');
  }
  const currentDates = new Map(actual.service_dates.map(item => [item.date, item.status]));
  if (baseline.service_dates.some(item => currentDates.get(item.date) !== item.status)) {
    throw new Error('baseline service-date facts were not restored');
  }
}
function shanghaiDate(dayOffset) {
  const shifted = new Date(Date.now() + 8 * 60 * 60 * 1000);
  const midnight = Date.UTC(shifted.getUTCFullYear(), shifted.getUTCMonth(), shifted.getUTCDate() + dayOffset);
  return new Date(midnight).toISOString().slice(0, 10);
}
function payloadOf(body, key) {
  const value = body && body[key];
  if (!value || typeof value !== 'object') throw new Error(`${key} response was malformed`);
  return value;
}
function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) throw new Error(`${label} id was malformed`);
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
function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || !parsed.port
    || parsed.pathname !== '/' || parsed.search || parsed.hash || parsed.username || parsed.password) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}
function idempotencyKey(scope) { return `detail-media-${scope}-${randomUUID()}`; }
function safeMessage(error) {
  return String(error && error.message || 'unknown error')
    .replace(/Bearer\s+\S+/gi, 'Bearer [REDACTED]')
    .replace(/MYSQL_PWD=\S+/gi, 'MYSQL_PWD=[REDACTED]');
}
