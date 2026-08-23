import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || toolRoot;
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const upstreamOrigin = process.env.ORDER_COMPOSED_API_ORIGIN;
const paymentExpectation = process.env.ORDER_COMPOSED_PAYMENT_EXPECTATION || 'success';
const composedFlow = process.env.ORDER_COMPOSED_FLOW || 'customer';
const composedRunID = `ui1-consent-${Date.now()}-${Math.random().toString(16).slice(2)}`;
const mysqlContainer = process.env.ORDER_COMPOSED_MYSQL_CONTAINER || '';
const mysqlDatabase = process.env.ORDER_COMPOSED_MYSQL_DATABASE || '';
const mysqlUser = process.env.ORDER_COMPOSED_MYSQL_USER || '';
const mysqlPassword = process.env.ORDER_COMPOSED_MYSQL_PASSWORD || '';

if (!existsSync(browserPath)) {
  throw new Error('locked Chromium is missing; reuse the configured MINIPROGRAM_UI_DEPS cache');
}
if (!/^http:\/\/127\.0\.0\.1:\d{1,5}$/.test(upstreamOrigin || '')) {
  throw new Error('ORDER_COMPOSED_API_ORIGIN must be an explicit http://127.0.0.1:<port> origin');
}
if (!['success', 'pending'].includes(paymentExpectation)) {
  throw new Error('ORDER_COMPOSED_PAYMENT_EXPECTATION must be success or pending');
}
if (!['customer', 'merchant', 'consent'].includes(composedFlow)
  || (['merchant', 'consent'].includes(composedFlow) && paymentExpectation !== 'success')) {
  throw new Error('ORDER_COMPOSED_FLOW must be customer, merchant, or consent; merchant/consent require success payment expectation');
}
if (composedFlow === 'consent' && (!/^[A-Za-z0-9_.-]+$/.test(mysqlContainer)
  || !/^[A-Za-z0-9_]+$/.test(mysqlDatabase) || !/^[A-Za-z0-9_]+$/.test(mysqlUser) || !mysqlPassword)) {
  throw new Error('consent flow requires explicit ORDER_COMPOSED_MYSQL_CONTAINER/DATABASE/USER/PASSWORD evidence settings');
}

process.env.CHROME_BIN = browserPath;
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key',
    'access-control-expose-headers': 'x-request-id',
  };
}

async function requestJSON(origin, log, method, requestPath, options = {}) {
  const headers = {};
  if (options.token) headers.authorization = `Bearer ${options.token}`;
  if (options.data !== undefined) headers['content-type'] = 'application/json';
  if (options.idempotencyKey) headers['idempotency-key'] = options.idempotencyKey;
  const response = await fetch(`${origin}${requestPath}`, {
    method,
    headers,
    body: options.data === undefined ? undefined : JSON.stringify(options.data),
  });
  let body = {};
  const raw = await response.text();
  if (raw) {
    try { body = JSON.parse(raw); } catch (error) { body = {}; }
  }
  log.push({ method, path: requestPath, status: response.status, request_id: response.headers.get('x-request-id') || '' });
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  if (!expected.includes(response.status)) {
    const code = body && body.error && body.error.code || 'UNKNOWN';
    throw new Error(`${method} ${requestPath} returned ${response.status}/${code}`);
  }
  return body;
}

function shanghaiParts(value) {
  const shifted = new Date(value.getTime() + 8 * 60 * 60 * 1000);
  const pad = number => String(number).padStart(2, '0');
  return {
    date: `${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}`,
    time: `${pad(shifted.getUTCHours())}:${pad(shifted.getUTCMinutes())}`,
  };
}

function settingsWrite(view) {
  return {
    store_status: view.store_status,
    pickup_point: view.pickup_point,
    notice: view.notice,
    pickup_step_min: view.pickup_step_min,
    meal_periods: view.meal_periods,
    service_dates: view.service_dates,
  };
}

function merchantKey(scope) {
  return `ui1-${scope}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function restoreMerchantSetup(origin, setup) {
  const failures = [];
  if (setup.productChanged) {
    try {
      const restoredProduct = await requestJSON(origin, setup.requests, 'PUT', `/api/v1/merchant/products/${setup.product.id}/soldout`, {
        token: setup.miniToken,
        idempotencyKey: merchantKey('restore-soldout'),
        data: { service_date: setup.serviceDate, sold_out: setup.product.baselineSoldOut },
      });
      if (restoredProduct.product_id !== setup.product.id || restoredProduct.service_date !== setup.serviceDate
        || restoredProduct.sold_out !== setup.product.baselineSoldOut) {
        throw new Error('sold-out cleanup response did not match the saved baseline');
      }
    } catch (error) { failures.push(error.message); }
  }
  if (setup.settingsChanged) {
    try {
      const restoredSettings = await requestJSON(origin, setup.requests, 'PUT', '/api/v1/admin/settings', {
        token: setup.pcToken,
        idempotencyKey: merchantKey('restore-settings'),
        data: settingsWrite(setup.baselineSettings),
      });
      if (JSON.stringify(settingsWrite(restoredSettings)) !== JSON.stringify(settingsWrite(setup.baselineSettings))) {
        throw new Error('settings cleanup response did not match the saved baseline');
      }
      const reread = await requestJSON(origin, setup.requests, 'GET', '/api/v1/admin/settings', { token: setup.pcToken });
      if (JSON.stringify(settingsWrite(reread)) !== JSON.stringify(settingsWrite(setup.baselineSettings))) {
        throw new Error('settings cleanup reread did not match the saved baseline');
      }
    } catch (error) { failures.push(error.message); }
  }
  return failures;
}

async function prepareMerchantSetup(origin, includeFarPickup = false) {
  const setup = { requests: [], settingsChanged: false, productChanged: false, product: null };
  try {
    const session = await requestJSON(origin, setup.requests, 'POST', '/api/v1/auth/miniprogram/session', {
      expected: 201,
      data: { code: 'ui1-composed-owner-setup-code' },
    });
    setup.miniToken = session.access_token;
    if (typeof setup.miniToken !== 'string' || !setup.miniToken) throw new Error('owner setup session omitted token');

    const login = await requestJSON(origin, setup.requests, 'POST', '/api/v1/admin/auth/qrcode', { expected: 201, data: {} });
    const qr = new URL(login.qr_payload);
    const approvalSecret = qr.searchParams.get('approval_secret');
    if (!login.login_id || !login.poll_secret || !approvalSecret) throw new Error('owner setup QR proof incomplete');
    await requestJSON(origin, setup.requests, 'POST', '/api/v1/me/admin-login/approve', {
      token: setup.miniToken,
      data: { login_id: login.login_id, approval_secret: approvalSecret, code: 'ui1-composed-owner-phone-code' },
    });
    const poll = await requestJSON(origin, setup.requests, 'POST', '/api/v1/admin/auth/poll', {
      data: { login_id: login.login_id, poll_secret: login.poll_secret },
    });
    setup.pcToken = poll && poll.session && poll.session.token;
    if (poll.state !== 'APPROVED' || typeof setup.pcToken !== 'string' || !setup.pcToken) {
      throw new Error('owner setup PC session was not approved');
    }
    const owner = await requestJSON(origin, setup.requests, 'GET', '/api/v1/admin/me', { token: setup.pcToken });
    if (!owner.account || owner.account.role !== 'OWNER') throw new Error('owner setup was not an OWNER session');

    setup.baselineSettings = await requestJSON(origin, setup.requests, 'GET', '/api/v1/admin/settings', { token: setup.pcToken });
    const now = new Date();
    const target = new Date(Math.ceil((now.getTime() + 20 * 60 * 1000) / (5 * 60 * 1000)) * 5 * 60 * 1000);
    const farTarget = new Date(target.getTime() + 40 * 60 * 1000);
    const cutoff = new Date(target.getTime() - 5 * 60 * 1000);
    const current = shanghaiParts(now);
    const pickup = shanghaiParts(target);
    const farPickup = shanghaiParts(farTarget);
    if (pickup.date !== current.date || (includeFarPickup && farPickup.date !== current.date)) {
      throw new Error('merchant composed pickup crossed the Shanghai service date');
    }
    const dates = setup.baselineSettings.service_dates.map(date => Object.assign({}, date));
    const today = dates.find(date => date.date === current.date);
    if (!today) throw new Error('baseline admin settings omitted the current service date; refusing non-restorable setup');
    today.status = 'open';
    const meals = setup.baselineSettings.meal_periods.map(meal => Object.assign({}, meal));
    const lunch = meals.find(meal => meal.code === 'lunch');
    if (!lunch) throw new Error('baseline admin settings omitted lunch');
    lunch.cutoff_time = shanghaiParts(cutoff).time;
    lunch.pickup_from = pickup.time;
    lunch.pickup_to = includeFarPickup ? farPickup.time : pickup.time;
    const temporary = settingsWrite(Object.assign({}, setup.baselineSettings, {
      store_status: 'open',
      pickup_step_min: 5,
      meal_periods: meals,
      service_dates: dates,
    }));
    setup.settingsChanged = true;
    await requestJSON(origin, setup.requests, 'PUT', '/api/v1/admin/settings', {
      token: setup.pcToken,
      idempotencyKey: merchantKey('temporary-settings'),
      data: temporary,
    });
    setup.serviceDate = pickup.date;
    setup.pickupTime = pickup.time;
    setup.farPickupTime = includeFarPickup ? farPickup.time : '';

    const menu = await requestJSON(origin, setup.requests, 'GET', `/api/v1/menu?date=${encodeURIComponent(setup.serviceDate)}&time=${encodeURIComponent(setup.pickupTime)}`, {
      token: setup.miniToken,
    });
    const products = (menu.categories || []).flatMap(category => category.products || []);
    const product = products.find(item => item && item.listed === true);
    if (!product || typeof product.id !== 'string' || typeof product.sold_out !== 'boolean') {
      throw new Error('merchant setup menu had no listed product');
    }
    setup.product = { id: product.id, baselineSoldOut: product.sold_out, preparedSoldOut: false };
    if (product.sold_out) {
      setup.productChanged = true;
      await requestJSON(origin, setup.requests, 'PUT', `/api/v1/merchant/products/${product.id}/soldout`, {
        token: setup.miniToken,
        idempotencyKey: merchantKey('prepare-soldout'),
        data: { service_date: setup.serviceDate, sold_out: false },
      });
    }
    return setup;
  } catch (error) {
    const restoreFailures = await restoreMerchantSetup(origin, setup);
    if (restoreFailures.length) throw new Error(`${error.message}; cleanup failed: ${restoreFailures.join('; ')}`);
    throw error;
  }
}

function verifyConsentMySQL() {
  const notes = ['ready', 'refund-accept', 'refund-reject'].map(suffix => `${composedRunID}-${suffix}`);
  const quoted = notes.map(note => `'${note}'`).join(',');
  const sql = `SELECT o.order_note,c.kind,c.decision,IF(c.consumed_at IS NULL,'0','1'),COALESCE(x.kind,''),COALESCE(x.state,''),COALESCE(x.provider_message_id,'') FROM orders o JOIN notification_consents c ON c.order_id=o.id LEFT JOIN notification_outbox x ON x.order_id=o.id AND x.kind=c.kind WHERE o.order_note IN (${quoted}) ORDER BY o.order_note,c.kind,c.grant_sequence`;
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser, `--database=${mysqlDatabase}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim();
  const rows = output ? output.split('\n').map(line => {
    const columns = line.split('\t');
    while (columns.length < 7) columns.push('');
    return columns;
  }) : [];
  const byNote = new Map(rows.map(row => [row[0], row]));
  const ready = byNote.get(notes[0]);
  const accepted = byNote.get(notes[1]);
  const rejected = byNote.get(notes[2]);
  if (rows.length !== 3
    || !ready || ready[1] !== 'READY' || ready[2] !== 'ACCEPTED' || ready[3] !== '1'
      || ready[4] !== 'READY' || ready[5] !== 'SENT' || !ready[6]
    || !accepted || accepted[1] !== 'REFUND_RESULT' || accepted[2] !== 'ACCEPTED' || accepted[3] !== '1'
      || accepted[4] !== 'REFUND_RESULT' || accepted[5] !== 'SENT' || !accepted[6]
    || !rejected || rejected[1] !== 'REFUND_RESULT' || rejected[2] !== 'REJECTED' || rejected[3] !== '0'
      || rejected[4] !== '' || rejected[5] !== '' || rejected[6] !== '') {
    throw new Error(`consent MySQL evidence mismatch: ${JSON.stringify(rows)}`);
  }
  return rows.map(row => ({
    order_note: row[0], kind: row[1], decision: row[2], consumed: row[3] === '1',
    outbox_kind: row[4], outbox_state: row[5], provider_message: !!row[6],
  }));
}

async function startTransparentProxy(origin) {
  const target = new URL(origin);
  const requests = [];
  const server = http.createServer((request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, corsHeaders());
      response.end();
      return;
    }
    const headers = Object.assign({}, request.headers, { host: target.host });
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname,
      port: target.port,
      method: request.method,
      path: request.url,
      headers,
    }, upstreamResponse => {
      const responseHeaders = Object.assign({}, upstreamResponse.headers, corsHeaders());
      delete responseHeaders.connection;
      requests.push({
        method: request.method,
        path: request.url,
        status: upstreamResponse.statusCode,
        request_id: upstreamResponse.headers['x-request-id'] || '',
      });
      response.writeHead(upstreamResponse.statusCode || 502, responseHeaders);
      upstreamResponse.pipe(response);
    });
    upstream.on('error', error => {
      requests.push({ method: request.method, path: request.url, status: 0, error: error.code || 'UPSTREAM_ERROR' });
      if (!response.headersSent) response.writeHead(502, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
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
    requests,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

const proxy = await startTransparentProxy(upstreamOrigin);
let merchantSetup;
try {
  merchantSetup = ['merchant', 'consent'].includes(composedFlow)
    ? await prepareMerchantSetup(upstreamOrigin, composedFlow === 'consent') : null;
} catch (error) {
  await proxy.close();
  throw error;
}
process.env.ORDER_COMPOSED_PROXY_ORIGIN = proxy.origin;
process.env.ORDER_COMPOSED_PAYMENT_EXPECTATION = paymentExpectation;
process.env.ORDER_COMPOSED_FLOW = composedFlow;
process.env.ORDER_COMPOSED_RUN_ID = composedRunID;
process.env.ORDER_COMPOSED_MERCHANT_SETUP = merchantSetup ? JSON.stringify({
  service_date: merchantSetup.serviceDate,
  pickup_time: merchantSetup.pickupTime,
  far_pickup_time: merchantSetup.farPickupTime,
  product_id: merchantSetup.product.id,
  prepared_sold_out: merchantSetup.product.preparedSoldOut,
}) : '{}';
console.log('UI1_COMPOSED_ENV', JSON.stringify({
  runner: 'order-miniprogram-ui-gates@1.0.0',
  simulator: 'miniprogram-simulate@1.6.2',
  browser: browserVersion,
  upstream: upstreamOrigin,
  proxy: `${proxy.origin} (random loopback)`,
  payment_expectation: paymentExpectation,
  flow: composedFlow,
  merchant_setup: merchantSetup ? {
    service_date: merchantSetup.serviceDate,
    pickup_time: merchantSetup.pickupTime,
    far_pickup_time: merchantSetup.farPickupTime || undefined,
    product_id: merchantSetup.product.id,
  } : null,
}));

let exitCode;
let cleanupFailures = [];
let consentMySQLEvidence = [];
let executionFailure = '';
try {
  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
  if (exitCode === 0 && composedFlow === 'consent') consentMySQLEvidence = verifyConsentMySQL();
} catch (error) {
  executionFailure = error && error.message || 'composed runner failed';
  exitCode = 1;
} finally {
  if (merchantSetup) cleanupFailures = await restoreMerchantSetup(upstreamOrigin, merchantSetup);
  await proxy.close();
}

if (cleanupFailures.length) {
  console.error('UI1_COMPOSED_CLEANUP', JSON.stringify({ status: 'FAIL', failures: cleanupFailures }));
  exitCode = exitCode || 1;
}
if (executionFailure) console.error('UI1_COMPOSED_EXECUTION', JSON.stringify({ status: 'FAIL', error: executionFailure }));

console.log('UI1_COMPOSED_RESULT', JSON.stringify({
  status: exitCode === 0 ? 'PASS' : 'FAIL',
  scenarios: composedFlow === 'merchant' || composedFlow === 'consent' || paymentExpectation === 'pending' ? 1 : 4,
  evidence_level: 'L3_LOCAL_COMPOSED',
  setup_requests: merchantSetup ? merchantSetup.requests : [],
  upstream_requests: proxy.requests,
  mysql_evidence: consentMySQLEvidence,
}));
if (exitCode !== 0) process.exitCode = exitCode;
