import { spawn, execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { createRequire } from 'node:module';
import { createReadStream, existsSync, mkdirSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolRoot, '../..');
const appsRoot = path.join(repositoryRoot, 'apps');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || '/Users/vivix/.codex/worktrees/a61d/order/tools/miniprogram-ui';
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('reuse the locked MINIPROGRAM_UI_DEPS Chromium cache');
if (process.env.ORDER_TEST_MYSQL_INSTANCE !== 'order-mysql-w3' || process.env.ORDER_TEST_MYSQL_ISOLATED !== 'YES') {
  throw new Error('refund/unclaimed L3 requires the single isolated order-mysql-w3');
}

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const runID = `${process.pid}-${randomUUID().replaceAll('-', '')}`;
const infoFile = `/private/tmp/order-refund-unclaimed-l3-${runID}.json`;
const stopFile = `/private/tmp/order-refund-unclaimed-l3-${runID}.stop`;
const evidenceRoot = process.env.ORDER_REFUND_UNCLAIMED_L3_EVIDENCE_ROOT || `/private/tmp/order-refund-unclaimed-l3-${candidateSHA.slice(0, 12)}`;
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const fixture = { product: refundProduct(), past_a: {}, past_b: {}, user_order: {} };
const requests = [];
const checks = [];
let harness;
let harnessOutput = '';
let apiOrigin = '';
let proxy;
let failure;
let karmaExitCode = 1;

process.stdout.write(`REFUND_UNCLAIMED_L3_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, mysql_instance: 'order-mysql-w3' })}\n`);

try {
  harness = startHarness();
  const info = await waitForHarnessInfo();
  apiOrigin = exactLoopbackOrigin(info.origin);
  fixture.schema = exactIdentifier(info.schema, 'private schema');
  if (!fixture.schema.startsWith('order_acceptance_')) throw new Error('harness did not create a fresh acceptance schema');
  fixture.past_date = exactDate(info.past_date, 'past date');
  fixture.current_date = exactDate(info.current_date, 'current date');
  fixture.pickup_time = exactTime(info.pickup_time, 'pickup time');
  fixture.user_token = await acquireMini(info.user_login_code, info.user_phone_code, false);
  fixture.owner_token = await acquireMini(info.owner_login_code, info.owner_phone_code, true);
  fixture.pc_token = await acquirePCSession(fixture.owner_token, info.owner_phone_code);

  await setClock(exactTimestamp(info.past_clock, 'past clock'));
  fixture.past_a = await createOrder('past-unclaimed-a', fixture.past_date, 1);
  fixture.past_b = await createOrder('past-unclaimed-b', fixture.past_date, 2);
  await setClock(`${fixture.past_date}T03:00:00Z`);
  const production = await request('POST', '/api/v1/__acceptance/refund-unclaimed/production-worker');
  if (production.advanced !== 2) throw new Error(`past production advanced ${production.advanced}`);
  fixture.past_a = Object.assign(fixture.past_a, await merchantReady(fixture.past_a, 'past-a-ready'));
  fixture.past_b = Object.assign(fixture.past_b, await merchantReady(fixture.past_b, 'past-b-ready'));

  await setClock(exactTimestamp(info.current_clock, 'current clock'));
  fixture.user_order = await createOrder('user-future-cancel', fixture.current_date, 1);
  if (fixture.user_order.state !== 'RESERVED') throw new Error(`user cancellation order started ${fixture.user_order.state}`);

  proxy = await startProxy(apiOrigin, requests);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_REFUND_UNCLAIMED_L3_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_REFUND_UNCLAIMED_L3_FIXTURE = JSON.stringify(fixture);
  const config = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.refund-unclaimed-l3.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  karmaExitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(config, resolve);
    server.start().catch(reject);
  });
  if (karmaExitCode !== 0) throw new Error(`rendered Mini Karma gate exited ${karmaExitCode}`);
  record('Mini rendered user refund and Merchant cross-day redeem completed', true);

  await exercisePC();
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
} finally {
  if (proxy) await proxy.close().catch(() => {});
  if (harness) {
    writeFileSync(stopFile, 'stop\n', { mode: 0o600 });
    await waitForHarnessExit(harness, 30_000).catch(error => {
      if (!failure) failure = error;
      harness.kill('SIGTERM');
    });
  }
  rmSync(infoFile, { force: true });
  rmSync(stopFile, { force: true });
}

const passed = !failure && karmaExitCode === 0 && checks.every(item => item.ok);
const receipt = {
  schema: 'order.refund-unclaimed-l3.ui1.v1', candidate_sha: candidateSHA, generated_at: new Date().toISOString(),
  status: passed ? 'PASS' : 'FAIL', evidence_level: 'L3_LOCAL_COMPOSED', browser: browserVersion,
  mysql: 'one random fresh-v44 schema on order-mysql-w3',
  cases: ['AC-14', 'BE-14', 'BE-19', 'INV-10'],
  fixture: {
    schema: fixture.schema, user_order_id: fixture.user_order.id,
    past_unclaimed_id: fixture.past_a.id, past_redeemed_id: fixture.past_b.id,
  },
  claims: [
    'user and PC initiate immutable full refunds from rendered clients',
    'UNKNOWN and PROCESSING remain REFUNDING with one provider Create',
    'only provider SUCCESS reaches REFUNDED once',
    'past READY orders are rendered as unclaimed and never count before redeem',
    'COMPLETED counts once then REFUNDING/REFUNDED are excluded from revenue and sales',
    'refund clears redemption capability and cannot invent inventory return',
  ],
  checks, requests,
  external: [{ level: 'UI3/L4', status: 'BLOCKED_EXTERNAL', reason: 'real WeChat refund, funds arrival, callback and device confirmation require customer platform assets' }],
  error: failure ? safeMessage(failure) : undefined,
};
const receiptPath = path.join(evidenceRoot, 'receipt.json');
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`REFUND_UNCLAIMED_L3_RESULT ${JSON.stringify({ status: receipt.status, candidate_sha: candidateSHA, checks: checks.length, requests: requests.length, receipt: receiptPath, error: receipt.error })}\n`);
if (!passed) {
  if (harnessOutput) process.stderr.write(`REFUND_UNCLAIMED_L3_HARNESS ${harnessOutput.slice(-5000)}\n`);
  process.exitCode = 1;
}

function startHarness() {
  const child = spawn('go', ['test', '-p=1', './cmd/order-api', '-run', '^TestRefundUnclaimedL3Server$', '-count=1', '-v'], {
    cwd: path.join(repositoryRoot, 'services/api'),
    env: {
      ...process.env,
      ORDER_REFUND_UNCLAIMED_L3_SERVE: 'YES',
      ORDER_REFUND_UNCLAIMED_L3_INFO_FILE: infoFile,
      ORDER_REFUND_UNCLAIMED_L3_STOP_FILE: stopFile,
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

async function acquireMini(loginCode, phoneCode, owner) {
  const session = await request('POST', '/api/v1/auth/miniprogram/session', { expected: 201, body: { code: loginCode } });
  const token = exactToken(session.access_token, owner ? 'owner Mini token' : 'user Mini token');
  await request('POST', '/api/v1/me/bind-phone', { token, body: { code: phoneCode } });
  if (owner) {
    const login = await request('POST', '/api/v1/me/merchant-login', { token, body: { code: phoneCode } });
    if (login.merchant?.role !== 'OWNER') throw new Error('owner Mini session did not derive OWNER role');
  }
  return token;
}

async function acquirePCSession(ownerToken, phoneCode) {
  const login = await request('POST', '/api/v1/admin/auth/qrcode', { expected: 201, body: {} });
  const payload = new URL(exactString(login.qr_payload, 'PC qr_payload'));
  const loginID = exactString(login.login_id, 'PC login id');
  await request('POST', '/api/v1/me/admin-login/approve', {
    token: ownerToken,
    body: { login_id: loginID, approval_secret: exactString(payload.searchParams.get('approval_secret'), 'approval secret'), code: phoneCode },
  });
  const poll = await request('POST', '/api/v1/admin/auth/poll', {
    body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll secret') },
  });
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session token');
}

async function setClock(now) {
  const result = await request('PUT', '/api/v1/__acceptance/refund-unclaimed/clock', { body: { now } });
  if (result.now !== now) throw new Error(`clock ended ${result.now}`);
}

async function createOrder(label, pickupDate, quantity) {
  const key = `refund-${runID}-${label}`;
  const quote = await request('POST', '/api/v1/quotes', {
    token: fixture.user_token, key: `${key}-quote`, expected: 201,
    body: {
      contact_name: '退款闭环用户', pickup_date: pickupDate, pickup_time: fixture.pickup_time, order_note: label,
      items: [{ product_id: fixture.product.id, quantity, flavors: ['少饭'], note: label }],
    },
  });
  const quoteID = exactID(quote.quote?.id, `${label} quote id`);
  const prepay = await request('POST', '/api/v1/orders/prepay', {
    token: fixture.user_token, key: `${key}-prepay`, expected: 201, body: { quote_id: quoteID },
  });
  const prepaymentID = exactID(prepay.prepayment?.id, `${label} prepayment id`);
  const confirmed = await request('POST', '/api/v1/orders/confirm', {
    token: fixture.user_token, key: `${key}-confirm`, body: { prepayment_id: prepaymentID },
  });
  if (confirmed.state !== 'ORDER_CREATED') throw new Error(`${label} confirm ended ${confirmed.state}`);
  const id = exactID(confirmed.order_id, `${label} order id`);
  const detail = await request('GET', `/api/v1/orders/${id}`, { token: fixture.user_token });
  return {
    id, quantity, prepayment_id: prepaymentID, state: detail.order?.state,
    order_no: exactString(detail.order?.order_no, `${label} order no`),
    pickup_number: exactString(detail.order?.pickup_number, `${label} pickup number`),
    amount_cents: Number(detail.order?.payable_cents),
  };
}

async function merchantReady(order, suffix) {
  const view = await request('POST', `/api/v1/merchant/orders/${order.id}/ready`, {
    token: fixture.owner_token, key: `refund-${runID}-${suffix}`, body: {},
  });
  if (view.order?.state !== 'READY_FOR_PICKUP') throw new Error(`${suffix} ended ${view.order?.state}`);
  return { state: view.order.state };
}

async function exercisePC() {
  await request('PUT', '/api/v1/__acceptance/refund-unclaimed/provider-mode', { body: { mode: 'PROCESSING' } });
  const browser = await chromium.launch({ executablePath: browserPath, headless: true });
  let context;
  try {
    context = await browser.newContext();
    await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), fixture.pc_token);
    const page = await context.newPage();
    page.on('response', response => {
      const url = new URL(response.url());
      if (url.pathname.startsWith('/api/v1/')) requests.push({ phase: 'PC-rendered', method: response.request().method(), path: url.pathname, query: url.search, status: response.status() });
    });
    await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
    await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner' && document.querySelector('#tb-title')?.textContent === '工作台');
    record('PC renders one authenticated OWNER session', true);

    await navigate(page, 'orders', '订单管理', '#tbl-host');
    let responsePromise = page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('state') === '待取餐' && response.status() === 200;
    });
    await page.locator('[data-lane="待取餐"]').click();
    await responsePromise;
    responsePromise = page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('unclaimed') === 'true' && response.status() === 200;
    });
    await page.locator('[data-unc]').click();
    await responsePromise;
    await page.waitForFunction(value => document.querySelector('#detail')?.textContent.includes(value), fixture.past_a.order_no);
    const unclaimedText = await page.locator('#content').innerText();
    record('PC rendered unclaimed filter returns exact past READY order only', unclaimedText.includes(fixture.past_a.order_no) && !unclaimedText.includes(fixture.past_b.order_no));

    const pcA = await refundFromPC(page, fixture.past_a, 'PC未取餐全额退款');
    if (pcA.refund.amount_cents !== fixture.past_a.amount_cents || pcA.order.state !== '退款中') throw new Error('PC past-order refund changed full amount/state');
    let worker = await request('POST', '/api/v1/__acceptance/refund-unclaimed/refund-worker');
    if (worker.pending !== 1 || worker.applied !== 0) throw new Error(`PC PROCESSING worker diverged ${JSON.stringify(worker)}`);
    let facts = await orderFacts(fixture.past_a.id);
    assertOrderRefund(facts, 'REFUNDING', 'PROCESSING', 1, 1, 1, 'PC PROCESSING');
    const redeemA = await request('POST', `/api/v1/merchant/orders/${fixture.past_a.id}/redeem`, {
      token: fixture.owner_token, key: `refund-${runID}-redeem-refunding-a`, body: {}, expected: 409,
    });
    if (!redeemA.error?.code) throw new Error('REFUNDING unclaimed order did not reject redeem');
    await request('PUT', '/api/v1/__acceptance/refund-unclaimed/provider-mode', { body: { mode: 'SUCCESS' } });
    worker = await request('POST', '/api/v1/__acceptance/refund-unclaimed/refund-worker');
    if (worker.applied !== 1) throw new Error('PC unclaimed SUCCESS was not applied once');
    facts = await orderFacts(fixture.past_a.id);
    assertOrderRefund(facts, 'REFUNDED', 'SUCCESS', 1, 2, 2, 'PC unclaimed SUCCESS');
    if (facts.order.redemption_cipher_present || facts.order.product_sold_out) throw new Error('PC refund retained redeem cipher or mutated sold-out truth');

    await navigate(page, 'orders', '订单管理', '#tbl-host');
    responsePromise = page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('q') === fixture.past_b.order_no && response.status() === 200;
    });
    await page.locator('#f-kw').fill(fixture.past_b.order_no);
    await responsePromise;
    await page.waitForFunction(value => document.querySelector('#detail')?.textContent.includes(value), fixture.past_b.order_no);
    const beforeCompletedRefund = await orderFacts(fixture.past_b.id);
    if (beforeCompletedRefund.historical_stats.today_orders !== 1 || beforeCompletedRefund.historical_stats.product_sales !== fixture.past_b.quantity) {
      throw new Error('COMPLETED order did not count exactly once before refund');
    }
    const pcB = await refundFromPC(page, fixture.past_b, 'PC已核销全额退款');
    if (pcB.refund.amount_cents !== fixture.past_b.amount_cents || pcB.order.state !== '退款中') throw new Error('PC completed-order refund changed full amount/state');
    facts = await orderFacts(fixture.past_b.id);
    if (facts.historical_stats.today_orders !== 0 || facts.historical_stats.today_revenue_cents !== 0 || facts.historical_stats.product_sales !== 0) {
      throw new Error(`REFUNDING completed order remained effective revenue/sales ${JSON.stringify(facts.historical_stats)}`);
    }
    worker = await request('POST', '/api/v1/__acceptance/refund-unclaimed/refund-worker');
    if (worker.applied !== 1) throw new Error('PC completed-order SUCCESS was not applied once');
    facts = await orderFacts(fixture.past_b.id);
    assertOrderRefund(facts, 'REFUNDED', 'SUCCESS', 1, 1, 1, 'PC completed SUCCESS');
    if (facts.historical_stats.today_orders !== 0 || facts.historical_stats.today_revenue_cents !== 0 || facts.historical_stats.product_sales !== 0) {
      throw new Error('REFUNDED order returned to effective revenue/sales');
    }
    const duplicate = await request('POST', `/api/v1/admin/orders/${fixture.past_b.id}/refund`, {
      token: fixture.pc_token, key: `refund-${runID}-duplicate-b`, body: { reason: '重复退款必须拒绝' }, expected: 409,
    });
    if (!duplicate.error?.code) throw new Error('terminal order accepted a second refund');
    const redeemB = await request('POST', `/api/v1/merchant/orders/${fixture.past_b.id}/redeem`, {
      token: fixture.owner_token, key: `refund-${runID}-redeem-refunded-b`, body: {}, expected: 409,
    });
    if (!redeemB.error?.code) throw new Error('REFUNDED completed order accepted redeem');

    await navigate(page, 'dashboard', '工作台', '.kpi');
    const stats = await request('GET', '/api/v1/admin/stats', { token: fixture.pc_token });
    const dashboard = await page.locator('#content').innerText();
    record('PC dashboard excludes all unclaimed/refunding/refunded orders from effective revenue', stats.month_orders === 0 && stats.month_revenue_cents === 0 && dashboard.includes('¥0.00'));
    record('refund finality preserves no-inventory model and legal fulfillment states', facts.inventory_table_count === 0 && !facts.order.redemption_cipher_present && !facts.order.product_sold_out);
    await page.screenshot({ path: path.join(evidenceRoot, 'refund-unclaimed-final.png'), fullPage: true });
  } finally {
    if (context) await context.close().catch(() => {});
    await browser.close().catch(() => {});
  }
}

async function refundFromPC(page, order, reason) {
  await page.locator('#detail [data-refund]').click();
  await page.locator('.modal #rf-why').fill(reason);
  const responsePromise = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === `/api/v1/admin/orders/${order.id}/refund`);
  await page.locator('.modal [data-a="ok"]').click();
  const response = await responsePromise;
  const body = await response.json();
  if (response.status() !== 200 || !body.order || !body.refund) throw new Error(`PC refund ${order.id} returned ${response.status()}`);
  await page.waitForFunction(() => document.querySelector('#content')?.textContent.includes('退款中'));
  return body;
}

async function navigate(page, route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).first().waitFor();
  await page.waitForTimeout(80);
}

async function orderFacts(orderID) {
  return request('GET', `/api/v1/__acceptance/refund-unclaimed/facts?order_id=${encodeURIComponent(orderID)}`);
}

function assertOrderRefund(view, state, provider, creates, queries, observations, label) {
  const actual = view?.order?.refund;
  if (!actual || view.order.state !== state || actual.provider_state !== provider
    || actual.provider_create_count !== creates || actual.provider_query_count !== queries
    || actual.observation_count !== observations) throw new Error(`${label} facts diverged ${JSON.stringify(view)}`);
}

async function request(method, pathname, options = {}) {
  const headers = { Accept: 'application/json' };
  if (options.token) headers.authorization = `Bearer ${options.token}`;
  if (options.key) headers['idempotency-key'] = options.key;
  if (options.body !== undefined) headers['content-type'] = 'application/json';
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
  });
  const raw = await response.text();
  let body = {};
  if (raw) try { body = JSON.parse(raw); } catch {}
  requests.push({ phase: 'setup/control', method, path: pathname, status: response.status });
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  if (!expected.includes(response.status)) throw new Error(`${method} ${pathname} returned ${response.status}/${body.error?.code || 'UNKNOWN'}`);
  return body;
}

async function startProxy(origin, log) {
  const target = new URL(origin);
  const server = http.createServer(async (incoming, outgoing) => {
    try {
      if (incoming.url.startsWith('/api/')) {
        if (incoming.method === 'OPTIONS') {
          outgoing.writeHead(204, corsHeaders());
          outgoing.end();
          return;
        }
        const chunks = [];
        for await (const chunk of incoming) chunks.push(chunk);
        const headers = {};
        for (const [name, value] of Object.entries(incoming.headers)) if (value !== undefined && !['host', 'content-length', 'connection'].includes(name)) headers[name] = value;
        const upstream = await fetch(`${target.origin}${incoming.url}`, {
          method: incoming.method, headers, body: chunks.length ? Buffer.concat(chunks) : undefined, redirect: 'error',
        });
        const body = Buffer.from(await upstream.arrayBuffer());
        log.push({ phase: 'rendered-ui', method: incoming.method, path: incoming.url, status: upstream.status });
        outgoing.writeHead(upstream.status, { 'content-type': upstream.headers.get('content-type') || 'application/octet-stream', 'content-length': body.length, 'cache-control': 'no-store', ...corsHeaders() });
        outgoing.end(body);
        return;
      }
      serveStatic(incoming, outgoing);
    } catch {
      outgoing.writeHead(502, { 'content-type': 'text/plain; charset=utf-8', ...corsHeaders() });
      outgoing.end('private composed proxy unavailable');
    }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { origin: `http://127.0.0.1:${server.address().port}`, close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())) };
}

function serveStatic(request, response) {
  const pathname = decodeURIComponent(new URL(request.url, 'http://127.0.0.1').pathname);
  const relative = pathname === '/' ? 'web-admin/index.html' : pathname.replace(/^\/+/, '');
  const target = path.resolve(appsRoot, relative);
  if (!target.startsWith(`${appsRoot}${path.sep}`) || !existsSync(target) || !statSync(target).isFile()) {
    response.writeHead(404); response.end('not found'); return;
  }
  const contentType = target.endsWith('.js') ? 'text/javascript; charset=utf-8'
    : target.endsWith('.css') ? 'text/css; charset=utf-8'
      : target.endsWith('.png') ? 'image/png' : 'text/html; charset=utf-8';
  response.writeHead(200, { 'content-type': contentType, 'cache-control': 'no-store' });
  createReadStream(target).pipe(response);
}

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key',
  };
}

function refundProduct() {
  return {
    id: '1', category_id: '1', name: '工作餐', description: '两荤一素', specification: '份', meal_period: 'lunch',
    images: [], listed: true, sold_out: false, original_unit_price_cents: 1250,
    staff_unit_price_cents: 1000, isStaffPrice: true, price_cents: 1000,
  };
}

function record(name, ok) { checks.push({ name, ok: Boolean(ok) }); }
function exactLoopbackOrigin(value) { const parsed = new URL(value); if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.pathname !== '/' || parsed.search || parsed.hash || parsed.username || parsed.password) throw new Error('private origin must be exact loopback HTTP'); return parsed.origin; }
function exactIdentifier(value, label) { if (typeof value !== 'string' || !/^[A-Za-z0-9_]+$/.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactDate(value, label) { if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactTime(value, label) { if (typeof value !== 'string' || !/^(?:[01]\d|2[0-3]):[0-5]\d$/.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactTimestamp(value, label) { if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:00Z$/.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactToken(value, label) { const token = exactString(value, label); if (token.length < 16 || /\s/.test(token)) throw new Error(`${label} is invalid`); return token; }
function exactID(value, label) { if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactString(value, label) { if (typeof value !== 'string' || value.trim() === '') throw new Error(`${label} is missing`); return value; }
function safeMessage(error) { return String(error?.message || error || 'unknown failure').replace(/[\r\n]+/g, ' ').slice(0, 500); }
function delay(milliseconds) { return new Promise(resolve => setTimeout(resolve, milliseconds)); }
