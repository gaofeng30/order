import http from 'node:http';
import { execFileSync, spawn } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chrome is missing; reuse the configured Playwright cache');

const mysql = {
  container: exactIdentifier(process.env.ORDER_TEST_MYSQL_INSTANCE, 'MySQL instance', true),
  user: exactIdentifier(process.env.ORDER_TEST_MYSQL_USER, 'MySQL user'),
  password: exactString(process.env.ORDER_TEST_MYSQL_PASSWORD, 'MySQL password'),
};
if (mysql.container !== 'order-mysql-w3' || process.env.ORDER_TEST_MYSQL_ISOLATED !== 'YES') throw new Error('the selector requires the single isolated order-mysql-w3');

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const runID = `${process.pid}-${randomUUID().replaceAll('-', '')}`;
const infoFile = `/private/tmp/order-pc-closure-${runID}.json`;
const stopFile = `/private/tmp/order-pc-closure-${runID}.stop`;
const evidenceRoot = process.env.ORDER_PC_CLOSURE_EVIDENCE_ROOT || `/private/tmp/order-pc-closure-ui1-${candidateSHA.slice(0, 12)}`;
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const checks = [];
const requests = [];
let harness;
let harnessOutput = '';
let apiOrigin = '';
let schema = '';
let miniToken = '';
let pcToken = '';
let proxy;
let browser;
let context;
let page;
let failure;

process.stdout.write(`PC_CLOSURE_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, mysql_instance: mysql.container })}\n`);

try {
  harness = startHarness();
  const info = await waitForHarnessInfo();
  apiOrigin = exactLoopbackOrigin(info.origin);
  schema = exactIdentifier(info.schema, 'private schema');
  if (!schema.startsWith('order_acceptance_')) throw new Error('harness did not create a private acceptance schema');

  miniToken = await acquireMiniOwner(info.login_code, info.phone_code);
  pcToken = await acquirePCSession(info.phone_code);
  const yesterdayOrder = await createPaidOrder('2026-08-23', 'unclaimed', 2, false);
  const completedOrder = await createPaidOrder('2026-08-23', 'completed', 1, true);
  const corrupt = await createCorruptPending('2026-08-24');

  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext({ acceptDownloads: true });
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  page = await context.newPage();
  page.on('response', response => {
    const url = new URL(response.url());
    if (url.pathname.startsWith('/api/v1/')) requests.push({ method: response.request().method(), path: url.pathname, query: url.search, status: response.status() });
  });

  const factsBeforeDashboard = transactionFacts();
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner' && document.querySelector('#tb-title')?.textContent === '工作台');
  record('Chrome renders an authenticated real OWNER PC session', true);
  await exerciseDashboard(completedOrder, yesterdayOrder, factsBeforeDashboard);
  await exerciseOrders(completedOrder);
  await exerciseFinance(completedOrder);
  await exerciseCorruptPending(corrupt);
  await page.screenshot({ path: path.join(evidenceRoot, 'final-pending.png'), fullPage: true });
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
  if (page) await page.screenshot({ path: path.join(evidenceRoot, 'failure.png'), fullPage: true }).catch(() => {});
} finally {
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
  if (harness) {
    writeFileSync(stopFile, 'stop\n', { mode: 0o600 });
    await waitForHarnessExit(harness, 30_000).catch(error => {
      if (!failure) failure = error;
      harness.kill('SIGTERM');
    });
  }
}

for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
const passed = !failure && checks.every(item => item.ok);
const receipt = {
  schema: 'order.pc01-pc04-closure-ui1.v1', candidate_sha: candidateSHA, generated_at: new Date().toISOString(),
  browser: browserVersion, evidence_level: 'L3_LOCAL_COMPOSED', mysql: 'one random fresh-v44 schema on order-mysql-w3',
  setup: 'test-only private Go server; bootstrap fixture only; all normal orders/refunds use root HTTP; one labeled quote-digest fault injection',
  cleanup: 'server stopped and private schema dropped by Go test cleanup', status: passed ? 'PASS' : 'FAIL', checks, requests,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_CLOSURE_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, receipt: path.join(evidenceRoot, 'receipt.json') })}\n`);
if (!passed) {
  if (harnessOutput) process.stderr.write(`PC_CLOSURE_HARNESS ${harnessOutput.slice(-4000)}\n`);
  process.exitCode = 1;
}

function startHarness() {
  const child = spawn('go', ['test', './cmd/order-api', '-run', '^TestPCClosureUI1Server$', '-count=1', '-v'], {
    cwd: path.join(repositoryRoot, 'services/api'),
    env: {
      ...process.env,
      ORDER_PC_CLOSURE_SERVE: 'YES', ORDER_PC_CLOSURE_INFO_FILE: infoFile, ORDER_PC_CLOSURE_STOP_FILE: stopFile,
      GOPROXY: process.env.GOPROXY || 'off', GOTOOLCHAIN: process.env.GOTOOLCHAIN || 'go1.26.5',
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

async function acquireMiniOwner(loginCode, phoneCode) {
  const session = await apiRequest('/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: loginCode }, status: 201 });
  const token = exactToken(session.access_token, 'Mini token');
  await apiRequest('/api/v1/me/bind-phone', { method: 'POST', token, body: { code: phoneCode }, status: 200 });
  const merchant = await apiRequest('/api/v1/me/merchant-login', { method: 'POST', token, body: { code: phoneCode }, status: 200 });
  record('one real Mini session binds the bootstrap OWNER identity', merchant.merchant?.bound === true && merchant.merchant?.role === 'OWNER');
  return token;
}

async function acquirePCSession(phoneCode) {
  const qr = await apiRequest('/api/v1/admin/auth/qrcode', { method: 'POST', body: {}, status: 201 });
  const approvalSecret = new URL(qr.qr_payload).searchParams.get('approval_secret');
  await apiRequest('/api/v1/me/admin-login/approve', { method: 'POST', token: miniToken, body: { login_id: qr.login_id, approval_secret: approvalSecret, code: phoneCode }, status: 200 });
  const poll = await apiRequest('/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: qr.login_id, poll_secret: qr.poll_secret }, status: 200 });
  return exactToken(poll.session?.token, 'PC token');
}

async function createPaidOrder(date, key, quantity, complete) {
  await apiRequest('/api/v1/__acceptance/clock', { method: 'POST', body: { date }, status: 200 });
  const quote = await apiRequest('/api/v1/quotes', {
    method: 'POST', token: miniToken, key: `pc-ui1-quote-${key}`,
    body: { contact_name: 'PC UI1顾客', pickup_date: date, pickup_time: '11:30', order_note: key, items: [{ product_id: '1', quantity, flavors: ['少饭'], note: key }] }, status: 201,
  });
  const prepay = await apiRequest('/api/v1/orders/prepay', { method: 'POST', token: miniToken, key: `pc-ui1-prepay-${key}`, body: { quote_id: quote.quote.id }, status: 201 });
  const confirm = await apiRequest('/api/v1/orders/confirm', { method: 'POST', token: miniToken, key: `pc-ui1-confirm-${key}`, body: { prepayment_id: prepay.prepayment.id }, status: 200 });
  let detail = await apiRequest(`/api/v1/orders/${confirm.order_id}`, { token: miniToken, status: 200 });
  if (detail.order.state !== 'PREPARING') throw new Error(`${key} order did not catch up to PREPARING`);
  await apiRequest(`/api/v1/merchant/orders/${confirm.order_id}/ready`, { method: 'POST', token: miniToken, key: `pc-ui1-ready-${key}`, body: {}, status: 200 });
  detail = await apiRequest(`/api/v1/orders/${confirm.order_id}`, { token: miniToken, status: 200 });
  if (complete) {
    await apiRequest('/api/v1/verify/scan', { method: 'POST', token: miniToken, key: `pc-ui1-redeem-${key}`, body: { token: detail.order.redemption_token }, status: 200 });
    detail = await apiRequest(`/api/v1/orders/${confirm.order_id}`, { token: miniToken, status: 200 });
  }
  record(`${key} normal business order is server-backed in ${detail.order.state}`, detail.order.state === (complete ? 'COMPLETED' : 'READY_FOR_PICKUP'));
  return { ...detail.order, prepaymentID: String(prepay.prepayment.id), quantity };
}

async function createCorruptPending(date) {
  await apiRequest('/api/v1/__acceptance/clock', { method: 'POST', body: { date }, status: 200 });
  const quote = await apiRequest('/api/v1/quotes', {
    method: 'POST', token: miniToken, key: 'pc-ui1-quote-corrupt',
    body: { contact_name: 'PC UI1顾客', pickup_date: date, pickup_time: '11:30', order_note: 'corrupt', items: [{ product_id: '1', quantity: 1, flavors: [], note: 'corrupt' }] }, status: 201,
  });
  const prepay = await apiRequest('/api/v1/orders/prepay', { method: 'POST', token: miniToken, key: 'pc-ui1-prepay-corrupt', body: { quote_id: quote.quote.id }, status: 201 });
  const quoteID = exactID(quote.quote.id, 'corrupt quote');
  mysqlQuery(`UPDATE quotes SET snapshot_digest=UNHEX(REPEAT('00',32)) WHERE id=${quoteID}`);
  const confirm = await apiRequest('/api/v1/orders/confirm', { method: 'POST', token: miniToken, key: 'pc-ui1-confirm-corrupt', body: { prepayment_id: prepay.prepayment.id }, status: 202 });
  record('fault-injected corrupt snapshot becomes durable PENDING without an order', confirm.state === 'PENDING');
  return { id: String(prepay.prepayment.id), quoteID };
}

async function exerciseDashboard(completedOrder, unclaimedOrder, factsBefore) {
  const stats = await adminRequest('/api/v1/admin/stats');
  const derived = JSON.parse(mysqlQuery(`SELECT JSON_OBJECT('completed_count',COUNT(*),'completed_amount',COALESCE(SUM(payable_cents),0)) FROM orders WHERE state='COMPLETED' AND YEAR(paid_at)=2026 AND MONTH(paid_at)=8`));
  const unclaimed = JSON.parse(mysqlQuery(`SELECT JSON_OBJECT('count',COUNT(*),'amount',COALESCE(SUM(payable_cents),0)) FROM orders WHERE state='READY_FOR_PICKUP' AND pickup_date<'2026-08-24'`));
  const text = await page.locator('#content').innerText();
  record('PC01 rendered month revenue and order count equal only COMPLETED MySQL facts',
    Number(stats.month_orders) === Number(derived.completed_count) && Number(stats.month_revenue_cents) === Number(derived.completed_amount) &&
    text.includes(`¥${yuan(derived.completed_amount)}`) && text.includes(String(derived.completed_count)));
  record('PC01 rendered sales section excludes the past READY order and shows the truthful empty-day state',
    unclaimed.count === 1 && stats.product_sales?.length === 0 && text.includes('暂无销量数据') && !text.includes(unclaimedOrder.order_no));
  record('PC01 dashboard queries do not mutate any transaction fact', JSON.stringify(factsBefore) === JSON.stringify(transactionFacts()));
}

async function exerciseOrders(order) {
  await navigate('orders', '订单管理', '#tbl-host');
  const lanes = await page.locator('#lanes').innerText();
  record('PC02 all six public order states plus all-lane are rendered', ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款', '全部'].every(label => lanes.includes(label)));
  for (const [label, value] of [['order number', order.order_no], ['pickup number', order.pickup_number], ['phone', '+8613800000003']]) {
    const response = page.waitForResponse(r => {
      const url = new URL(r.url());
      return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('q') === value && url.searchParams.get('date') === order.pickup_date && r.status() === 200;
    });
    await setDateAndSearch(order.pickup_date, value);
    await response;
    await page.waitForFunction(value => document.querySelector('#detail')?.textContent.includes(value), order.order_no);
    record(`PC02 visible ${label} search carries the selected date and returns the immutable order`, (await page.locator('#detail').innerText()).includes(`¥${yuan(order.payable_cents)}`));
  }
  await page.locator('#detail [data-refund]').click();
  await page.locator('.modal #rf-why').fill('PC02 UI1全额退款');
  const refundResponse = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/v1/admin/orders/${order.id}/refund`);
  await page.locator('.modal [data-a="ok"]').click();
  const refundHTTP = await refundResponse;
  const refundBody = await refundHTTP.json();
  record('PC02 visible full refund preserves the immutable paid amount and first shows REFUNDING', refundHTTP.status() === 200 && Number(refundBody.refund?.amount_cents) === Number(order.payable_cents) && refundBody.order?.state === '退款中');
  await waitFor(async () => {
    const body = await adminRequest(`/api/v1/admin/orders?q=${encodeURIComponent(order.order_no)}&date=${order.pickup_date}`);
    return body.orders?.[0]?.state === '已退款';
  }, 12_000, 'PC02 refund worker did not reach REFUNDED');
  await setDateAndSearch(order.pickup_date, '');
  await setDateAndSearch(order.pickup_date, order.order_no);
  await page.waitForFunction(() => document.querySelector('#detail')?.textContent.includes('已退款'));
  record('PC02 final REFUNDED state is visible after provider finality', true);
}

async function exerciseFinance(order) {
  await navigate('finance', '财务与对账', '#sum-host');
  const rangeResponse = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/admin/finance/summary' && new URL(r.url()).searchParams.get('from') === '2026-08-18' && r.status() === 200);
  await page.locator('[data-quick="7"]').click();
  await rangeResponse;
  await page.waitForFunction(value => document.querySelector('#fin-host')?.textContent.includes(value), order.order_no);
  const summary = await adminRequest('/api/v1/admin/finance/summary?from=2026-08-18&to=2026-08-24');
  const text = await page.locator('#content').innerText();
  record('PC03 rendered summary/payment detail use real server gross, refund and net', text.includes(`¥${yuan(summary.gross)}`) && text.includes(`−¥${yuan(summary.refundAmount)}`) && text.includes(order.order_no));
  await page.locator('[data-tab="refund"]').click();
  await page.waitForFunction(() => document.querySelector('#fin-host')?.textContent.includes('PC02 UI1全额退款'));
  record('PC03 rendered refund detail includes the final full refund', true);
  const downloadPromise = page.waitForEvent('download');
  await page.locator('[data-export]').click();
  const download = await downloadPromise;
  const downloadPath = await download.path();
  const csv = downloadPath ? readFileSync(downloadPath, 'utf8') : '';
  record('PC03 visible export downloads server CSV with the real order', csv.includes('订单号') && csv.includes(order.order_no));
  record('PC03 page explicitly refuses to call system totals a reconciled WeChat bill', text.includes('本页数字只汇总本系统事实') && text.includes('不单独代表已与微信核平'));
  const before = requests.filter(item => item.path.startsWith('/api/v1/admin/finance/')).length;
  await page.evaluate(() => {
    const from = document.querySelector('#f-from');
    const to = document.querySelector('#f-to');
    from.value = '2026-08-25'; to.value = '2026-08-24'; to.dispatchEvent(new Event('change', { bubbles: true }));
  });
  await page.waitForTimeout(120);
  const after = requests.filter(item => item.path.startsWith('/api/v1/admin/finance/')).length;
  record('PC03 reversed date range is rejected before any finance request', before === after && (await page.locator('#toast-host').innerText()).includes('起始日期不能晚于结束日期'));
}

async function exerciseCorruptPending(pending) {
  await navigate('pending', '支付待处理', '#pd-host');
  const row = page.locator(`tr[data-id="${pending.id}"]`);
  await row.waitFor();
  record('PC04 corrupt snapshot is visibly labeled data-validation failure', (await row.innerText()).includes('数据校验不通过'));
  const before = orderSequenceFacts();
  await row.locator('[data-act="build"]').click();
  const rejected = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/v1/admin/pending-payments/${pending.id}`);
  await page.locator('.modal [data-a="ok"]').click();
  const rejectedHTTP = await rejected;
  await page.waitForTimeout(80);
  record('PC04 visible MATERIALIZE rejects corrupt snapshot and keeps the modal/row', rejectedHTTP.status() === 409 && await page.locator('.modal').isVisible() && await row.isVisible());
  record('PC04 corrupt MATERIALIZE creates zero abnormal orders and burns zero pickup numbers', JSON.stringify(before) === JSON.stringify(orderSequenceFacts()));
  await page.locator('.modal [data-a="cancel"]').click();
  await row.locator('[data-act="void"]').click();
  await page.locator('.modal #pv-why').fill('PC04 UI1快照损坏退款');
  const refunded = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/v1/admin/pending-payments/${pending.id}`);
  await page.locator('.modal [data-a="ok"]').click();
  const refundedHTTP = await refunded;
  const refundedBody = await refundedHTTP.json();
  record('PC04 visible REFUND accepts exactly one full paid-prepayment refund', refundedHTTP.status() === 200 && Number(refundedBody.refund?.amount_cents) === 1250);
  await waitFor(async () => {
    const body = await adminRequest('/api/v1/admin/finance/refunds?from=2026-08-24&to=2026-08-24');
    return body.refunds?.some(item => item.reason === 'PC04 UI1快照损坏退款' && item.state === '已退款');
  }, 12_000, 'PC04 pending refund did not reach finality');
  await navigate('orders', '订单管理', '#tbl-host');
  await navigate('pending', '支付待处理', '#pd-host');
  record('PC04 final refund removes the pending row and still has zero order/sequence side effects', !(await page.locator(`tr[data-id="${pending.id}"]`).count()) && JSON.stringify(before) === JSON.stringify(orderSequenceFacts()));
}

async function setDateAndSearch(date, query) {
  await page.evaluate(({ date, query }) => {
    const dateInput = document.querySelector('#f-date');
    const search = document.querySelector('#f-kw');
    dateInput.value = date; dateInput.dispatchEvent(new Event('change', { bubbles: true }));
    search.value = query; search.dispatchEvent(new Event('input', { bubbles: true }));
  }, { date, query });
}

async function navigate(route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).first().waitFor();
  await page.waitForTimeout(80);
}

async function adminRequest(pathname, options = {}) { return apiRequest(pathname, { ...options, token: pcToken, status: options.status || 200 }); }

async function apiRequest(pathname, { method = 'GET', token = '', key = '', body, status = 200 } = {}) {
  const headers = { Accept: 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (key) headers['Idempotency-Key'] = key;
  let encoded;
  if (body !== undefined) { headers['Content-Type'] = 'application/json'; encoded = JSON.stringify(body); }
  const response = await fetch(`${apiOrigin}${pathname}`, { method, headers, body: encoded });
  const text = await response.text();
  let parsed = {};
  try { parsed = text ? JSON.parse(text) : {}; } catch { parsed = text; }
  if (response.status !== status) throw new Error(`${method} ${pathname} returned ${response.status}, want ${status}: ${safeMessage(parsed)}`);
  return parsed;
}

function transactionFacts() {
  return JSON.parse(mysqlQuery(`SELECT JSON_OBJECT('orders',(SELECT COUNT(*) FROM orders),'items',(SELECT COUNT(*) FROM order_items),'prepayments',(SELECT COUNT(*) FROM prepayments),'observations',(SELECT COUNT(*) FROM payment_observations),'refunds',(SELECT COUNT(*) FROM refunds),'sequences',(SELECT COUNT(*) FROM pickup_sequences),'audits',(SELECT COUNT(*) FROM action_audits),'versions',(SELECT COALESCE(SUM(record_version),0) FROM orders)+(SELECT COALESCE(SUM(record_version),0) FROM prepayments)+(SELECT COALESCE(SUM(record_version),0) FROM payment_observations)+(SELECT COALESCE(SUM(record_version),0) FROM refunds))`));
}

function orderSequenceFacts() {
  return JSON.parse(mysqlQuery(`SELECT JSON_OBJECT('orders',(SELECT COUNT(*) FROM orders),'last_number',(SELECT COALESCE(SUM(last_number),0) FROM pickup_sequences))`));
}

function mysqlQuery(sql) {
  return execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysql.password}`, mysql.container, 'mysql', '--batch', '--raw', '--skip-column-names', `-u${mysql.user}`, `--database=${schema}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim();
}

async function startSameOriginProxy(upstreamOrigin) {
  const server = http.createServer(async (request, response) => {
    try {
      if (request.url.startsWith('/api/')) { await proxyAPI(request, response, upstreamOrigin); return; }
      serveStatic(request, response);
    } catch {
      response.writeHead(502, { 'Content-Type': 'text/plain; charset=utf-8' }); response.end('local composed proxy unavailable');
    }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  const address = server.address();
  return { origin: `http://127.0.0.1:${address.port}`, close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())) };
}

async function proxyAPI(request, response, upstreamOrigin) {
  const chunks = [];
  for await (const chunk of request) chunks.push(chunk);
  const headers = {};
  for (const [name, value] of Object.entries(request.headers)) if (value !== undefined && !['host', 'content-length', 'connection'].includes(name)) headers[name] = value;
  const upstream = await fetch(`${upstreamOrigin}${request.url}`, { method: request.method, headers, body: chunks.length ? Buffer.concat(chunks) : undefined, redirect: 'error' });
  const body = Buffer.from(await upstream.arrayBuffer());
  const contentType = upstream.headers.get('content-type');
  response.writeHead(upstream.status, { ...(contentType ? { 'Content-Type': contentType } : {}), 'Content-Length': body.length, 'Cache-Control': 'no-store' }); response.end(body);
}

function serveStatic(request, response) {
  const pathname = decodeURIComponent(new URL(request.url, 'http://127.0.0.1').pathname);
  const relative = pathname === '/' ? 'web-admin/index.html' : pathname.replace(/^\/+/, '');
  const target = path.resolve(appsRoot, relative);
  if (!target.startsWith(`${appsRoot}${path.sep}`) || !existsSync(target) || !statSync(target).isFile()) { response.writeHead(404); response.end('not found'); return; }
  const contentType = target.endsWith('.js') ? 'text/javascript; charset=utf-8' : target.endsWith('.css') ? 'text/css; charset=utf-8' : 'text/html; charset=utf-8';
  response.writeHead(200, { 'Content-Type': contentType, 'Cache-Control': 'no-store' }); createReadStream(target).pipe(response);
}

function record(name, ok) { checks.push({ name, ok: Boolean(ok) }); }
function yuan(cents) { const value = Math.abs(Number(cents)); return `${Math.floor(value / 100)}.${String(value % 100).padStart(2, '0')}`; }
function exactID(value, label) { const id = String(value); if (!/^[1-9]\d*$/.test(id)) throw new Error(`${label} ID is invalid`); return id; }
function exactIdentifier(value, label, allowDash = false) { const pattern = allowDash ? /^[A-Za-z0-9_.-]+$/ : /^[A-Za-z0-9_]+$/; if (typeof value !== 'string' || !pattern.test(value)) throw new Error(`${label} is invalid`); return value; }
function exactString(value, label) { if (typeof value !== 'string' || value.trim() === '') throw new Error(`${label} is missing`); return value; }
function exactToken(value, label) { const token = exactString(value, label); if (token.length < 32) throw new Error(`${label} is malformed`); return token; }
function exactLoopbackOrigin(value) { const parsed = new URL(value); if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.pathname !== '/' || parsed.search || parsed.hash) throw new Error('harness origin must be exact loopback'); return parsed.origin; }
function safeMessage(value) { if (value instanceof Error) return value.message; try { return JSON.stringify(value); } catch { return String(value); } }
function delay(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }
async function waitFor(check, timeout, message) { const deadline = Date.now() + timeout; while (Date.now() < deadline) { if (await check()) return; await delay(100); } throw new Error(message); }
