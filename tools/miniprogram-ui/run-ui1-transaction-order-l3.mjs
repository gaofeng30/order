import { spawn, execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { createRequire } from 'node:module';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolRoot, '../..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || '/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui';
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('reuse the locked MINIPROGRAM_UI_DEPS Chromium cache');
if (process.env.ORDER_TEST_MYSQL_INSTANCE !== 'order-mysql-w3' || process.env.ORDER_TEST_MYSQL_ISOLATED !== 'YES') {
  throw new Error('transaction/order L3 requires the single isolated order-mysql-w3');
}

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const runID = `${process.pid}-${randomUUID().replaceAll('-', '')}`;
const infoFile = `/private/tmp/order-transaction-l3-${runID}.json`;
const stopFile = `/private/tmp/order-transaction-l3-${runID}.stop`;
const evidenceRoot = process.env.ORDER_TRANSACTION_L3_EVIDENCE_ROOT || `/private/tmp/order-transaction-l3-${candidateSHA.slice(0, 12)}`;
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const fixture = { exact: {}, near: {}, notification: {}, product: transactionProduct() };
const requests = [];
let harness;
let harnessOutput = '';
let apiOrigin = '';
let proxy;
let failure;
let exitCode = 1;

process.stdout.write(`TRANSACTION_ORDER_L3_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, mysql_instance: 'order-mysql-w3' })}\n`);

try {
  harness = startHarness();
  const info = await waitForHarnessInfo();
  apiOrigin = exactLoopbackOrigin(info.origin);
  fixture.schema = exactIdentifier(info.schema, 'private schema');
  if (!fixture.schema.startsWith('order_acceptance_')) throw new Error('harness did not create a fresh acceptance schema');
  fixture.pickup_date = exactDate(info.pickup_date, 'pickup date');
  fixture.pickup_time = exactTime(info.pickup_time, 'pickup time');

  fixture.user_token = await acquireUser(info.user_login_code, info.user_phone_code);
  fixture.owner_token = await acquireOwner(info.owner_login_code, info.owner_phone_code);

  await setClock(exactTimestamp(info.exact_now, 'exact clock'));
  fixture.exact = await createOrder('exact-30m');
  if (fixture.exact.state !== 'PREPARING') throw new Error(`exact 30-minute order started ${fixture.exact.state}`);
  fixture.exact.cancel_status = await rejectCancellation(fixture.exact.id, 'exact');

  await setClock(exactTimestamp(info.near_now, 'near clock'));
  fixture.near = await createOrder('near-29m');
  if (fixture.near.state !== 'PREPARING') throw new Error(`near order started ${fixture.near.state}`);
  fixture.near.cancel_status = await rejectCancellation(fixture.near.id, 'near');
  fixture.notification = await createOrder('notification-provider-failure');
  if (fixture.notification.state !== 'PREPARING') throw new Error(`notification order started ${fixture.notification.state}`);

  proxy = await startProxy(apiOrigin, requests);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_TRANSACTION_L3_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_TRANSACTION_L3_FIXTURE = JSON.stringify(fixture);
  const config = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.transaction-order-l3.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(config, resolve);
    server.start().catch(reject);
  });
  if (exitCode !== 0) throw new Error(`rendered Karma gate exited ${exitCode}`);
  if (requests.some(item => item.path === '/api/v1/payments/wechat/notify')) {
    throw new Error('lost-callback scenarios unexpectedly used the callback route');
  }
} catch (error) {
  failure = error;
} finally {
  if (proxy) await proxy.close().catch(() => {});
  if (harness) {
    writeFileSync(stopFile, 'stop\n', { mode: 0o600 });
    await waitForHarnessExit(harness, 30_000).catch(error => {
      if (!failure) failure = error;
      harness.kill('SIGTERM');
    });
  }
}

const passed = !failure && exitCode === 0;
const receipt = {
  schema: 'order.transaction-order-l3.ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  status: passed ? 'PASS' : 'FAIL',
  evidence_level: 'L3_LOCAL_COMPOSED',
  browser: browserVersion,
  mysql: 'one random fresh-v44 schema on order-mysql-w3',
  cases: ['AC-06', 'AC-07', 'AC-15', 'BE-07', 'BE-13', 'BE-21', 'INV-09'],
  fixture: {
    exact: fixture.exact, near: fixture.near, notification: fixture.notification,
    product_id: fixture.product.id,
  },
  claims: [
    'payment retry reuses one provider Create',
    'lost callback is recovered only by Query',
    'apply SQL failure keeps a durable observation and zero order',
    'exact 30-minute cancellation stays unavailable',
    'rejected READY consent has a supplemental entry',
    'notification provider failure does not change order state',
  ],
  requests,
  external: [
    { level: 'UI3', status: 'BLOCKED_EXTERNAL', reason: 'real wx.requestPayment and subscription consent require WeChat assets' },
  ],
  error: failure ? safeMessage(failure) : undefined,
};
const receiptPath = path.join(evidenceRoot, 'receipt.json');
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`TRANSACTION_ORDER_L3_RESULT ${JSON.stringify({ status: receipt.status, candidate_sha: candidateSHA, requests: requests.length, receipt: receiptPath, error: receipt.error })}\n`);
if (!passed) {
  if (harnessOutput) process.stderr.write(`TRANSACTION_ORDER_L3_HARNESS ${harnessOutput.slice(-5000)}\n`);
  process.exitCode = 1;
}

function startHarness() {
  const child = spawn('go', ['test', '-p=1', './cmd/order-api', '-run', '^TestTransactionOrderL3Server$', '-count=1', '-v'], {
    cwd: path.join(repositoryRoot, 'services/api'),
    env: {
      ...process.env,
      ORDER_TRANSACTION_L3_SERVE: 'YES',
      ORDER_TRANSACTION_L3_INFO_FILE: infoFile,
      ORDER_TRANSACTION_L3_STOP_FILE: stopFile,
      GOPROXY: process.env.GOPROXY || 'off',
      GOTOOLCHAIN: process.env.GOTOOLCHAIN || 'go1.26.5',
      GOCACHE: process.env.GOCACHE || '/Users/vivix/Library/Caches/go-build',
      GOMODCACHE: process.env.GOMODCACHE || '/Users/vivix/go/pkg/mod',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  child.stdout.on('data', chunk => { harnessOutput += chunk.toString(); });
  child.stderr.on('data', chunk => { harnessOutput += chunk.toString(); });
  return child;
}

async function waitForHarnessInfo() {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    if (existsSync(infoFile)) return JSON.parse(readFileSync(infoFile, 'utf8'));
    if (harness.exitCode !== null) throw new Error(`private Go harness exited ${harness.exitCode}`);
    await delay(50);
  }
  throw new Error('private Go harness did not publish its origin');
}

async function waitForHarnessExit(child, timeout) {
  const deadline = Date.now() + timeout;
  while (child.exitCode === null && Date.now() < deadline) await delay(50);
  if (child.exitCode === null) throw new Error('private Go harness did not stop');
  if (child.exitCode !== 0) throw new Error(`private Go harness exited ${child.exitCode}`);
}

async function acquireUser(loginCode, phoneCode) {
  const session = await request('POST', '/api/v1/auth/miniprogram/session', { expected: 201, body: { code: loginCode } });
  const token = exactToken(session.access_token, 'user Mini token');
  await request('POST', '/api/v1/me/bind-phone', { token, body: { code: phoneCode } });
  return token;
}

async function acquireOwner(loginCode, phoneCode) {
  const session = await request('POST', '/api/v1/auth/miniprogram/session', { expected: 201, body: { code: loginCode } });
  const token = exactToken(session.access_token, 'owner Mini token');
  await request('POST', '/api/v1/me/bind-phone', { token, body: { code: phoneCode } });
  const login = await request('POST', '/api/v1/me/merchant-login', { token, body: { code: phoneCode } });
  if (login.merchant?.role !== 'OWNER') throw new Error('owner Mini session did not derive OWNER role');
  return token;
}

async function setClock(now) {
  const view = await request('PUT', '/api/v1/__acceptance/transaction-order/clock', { body: { now } });
  if (view.now !== now) throw new Error(`clock ended ${view.now}`);
}

async function createOrder(label) {
  const prefix = `transaction-${runID}-${label}`;
  const quote = await request('POST', '/api/v1/quotes', {
    token: fixture.user_token, key: `${prefix}-quote`, expected: 201,
    body: {
      contact_name: '交易闭环用户', pickup_date: fixture.pickup_date, pickup_time: fixture.pickup_time, order_note: prefix,
      items: [{ product_id: fixture.product.id, quantity: 1, flavors: ['少饭'], note: label }],
    },
  });
  const quoteID = exactID(quote.quote?.id, `${label} quote id`);
  const prepay = await request('POST', '/api/v1/orders/prepay', {
    token: fixture.user_token, key: `${prefix}-prepay`, expected: 201, body: { quote_id: quoteID },
  });
  const prepaymentID = exactID(prepay.prepayment?.id, `${label} prepayment id`);
  const confirm = await request('POST', '/api/v1/orders/confirm', {
    token: fixture.user_token, key: `${prefix}-confirm`, body: { prepayment_id: prepaymentID },
  });
  if (confirm.state !== 'ORDER_CREATED') throw new Error(`${label} confirm ended ${confirm.state}`);
  const id = exactID(confirm.order_id, `${label} order id`);
  const detail = await request('GET', `/api/v1/orders/${id}`, { token: fixture.user_token });
  return { id, prepayment_id: prepaymentID, state: detail.order?.state };
}

async function rejectCancellation(orderID, label) {
  const response = await request('POST', `/api/v1/orders/${orderID}/cancel`, {
    token: fixture.user_token, key: `transaction-${runID}-cancel-${label}`,
    body: { reason: 'USER_REQUEST' }, expected: 409,
  });
  if (!response.error?.code) throw new Error(`${label} cancel omitted its fail-closed code`);
  return 409;
}

async function request(method, pathname, options = {}) {
  const headers = {};
  if (options.token) headers.authorization = `Bearer ${options.token}`;
  if (options.key) headers['idempotency-key'] = options.key;
  if (options.body !== undefined) headers['content-type'] = 'application/json';
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  const raw = await response.text();
  let body = {};
  if (raw) try { body = JSON.parse(raw); } catch {}
  requests.push({ phase: 'setup', method, path: pathname, status: response.status });
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  if (!expected.includes(response.status)) throw new Error(`${method} ${pathname} returned ${response.status}/${body.error?.code || 'UNKNOWN'}`);
  return body;
}

async function startProxy(origin, log) {
  const target = new URL(origin);
  const server = http.createServer((incoming, outgoing) => {
    if (incoming.method === 'OPTIONS') {
      outgoing.writeHead(204, corsHeaders());
      outgoing.end();
      return;
    }
    const headers = { ...incoming.headers, host: target.host };
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname, port: target.port, method: incoming.method, path: incoming.url, headers,
    }, response => {
      const responseHeaders = { ...response.headers, ...corsHeaders() };
      delete responseHeaders.connection;
      log.push({ phase: 'rendered-ui', method: incoming.method, path: incoming.url, status: response.statusCode || 0 });
      outgoing.writeHead(response.statusCode || 502, responseHeaders);
      response.pipe(outgoing);
    });
    upstream.on('error', error => {
      log.push({ phase: 'rendered-ui', method: incoming.method, path: incoming.url, status: 0, error: error.code || 'UPSTREAM_ERROR' });
      if (!outgoing.headersSent) outgoing.writeHead(502, { 'content-type': 'application/json', ...corsHeaders() });
      outgoing.end(JSON.stringify({ error: { code: 'PRIVATE_UPSTREAM_UNAVAILABLE' } }));
    });
    incoming.pipe(upstream);
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  return {
    origin: `http://127.0.0.1:${server.address().port}`,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key',
  };
}

function transactionProduct() {
  return {
    id: '1', category_id: '1', name: '工作餐', description: '两荤一素', specification: '份', meal_period: 'lunch',
    images: [], listed: true, sold_out: false, original_unit_price_cents: 1250,
    staff_unit_price_cents: 1000, isStaffPrice: true, price_cents: 1000,
  };
}

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password
    || parsed.pathname !== '/' || parsed.search || parsed.hash) throw new Error('private origin must be exact loopback HTTP');
  return parsed.origin;
}

function exactIdentifier(value, label) {
  if (typeof value !== 'string' || !/^[A-Za-z0-9_]+$/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function exactDate(value, label) {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function exactTime(value, label) {
  if (typeof value !== 'string' || !/^(?:[01]\d|2[0-3]):[0-5]\d$/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function exactTimestamp(value, label) {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:00Z$/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function exactToken(value, label) {
  if (typeof value !== 'string' || value.length < 16 || /\s/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw new Error(`${label} is invalid`);
  return value;
}

function safeMessage(error) {
  return String(error?.message || error || 'unknown failure').replace(/[\r\n]+/g, ' ').slice(0, 500);
}

function delay(milliseconds) { return new Promise(resolve => setTimeout(resolve, milliseconds)); }
