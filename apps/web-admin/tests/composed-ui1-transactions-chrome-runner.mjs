import http from 'node:http';
import { execFileSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, statSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash, createHmac, randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || 'http://127.0.0.1:8080');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chrome is missing; reuse the configured Playwright cache');

const pendingIDs = exactIDs(process.env.ORDER_PC_PENDING_IDS || '');
const mysql = {
  container: exactIdentifier(process.env.ORDER_COMPOSED_MYSQL_CONTAINER, 'MySQL container', true),
  database: exactIdentifier(process.env.ORDER_COMPOSED_MYSQL_DATABASE, 'MySQL database'),
  user: exactIdentifier(process.env.ORDER_COMPOSED_MYSQL_USER, 'MySQL user'),
  password: exactString(process.env.ORDER_COMPOSED_MYSQL_PASSWORD, 'MySQL password'),
};
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const evidenceRoot = process.env.ORDER_PC_EVIDENCE_ROOT || path.join('/private/tmp', `order-pc-transactions-ui1-${candidateSHA.slice(0, 12)}`);
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const checks = [];
const requests = [];
let pcToken = '';
let proxy;
let browser;
let context;
let page;
let failure;

process.stdout.write(`PC_TRANSACTION_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin, pending_ids: pendingIDs })}\n`);

try {
  const facts = readPendingFacts();
  record('two HTTP-created prepayments are awaiting local fake payment', facts.length === 2 && facts.every(f => f.materialization_state === 'AWAITING_PAYMENT' && f.provider_state === 'NOT_PAID'));
  pcToken = (await acquirePCSession()).pcToken;
  await waitForEffectiveDeadline(facts);
  for (const fact of facts) await sendLatePaymentCallback(fact);
  await waitFor(async () => {
    const body = await adminRequest('/api/v1/admin/pending-payments', { method: 'GET' }, 200);
    return pendingIDs.every(id => body.prepayments.some(item => String(item.id) === id));
  }, 12_000, 'late callbacks did not become durable pending payments');
  record('late signed fake-provider callbacks persist before PC processing', true);

  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext({ acceptDownloads: true });
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  page = await context.newPage();
  page.on('request', request => {
    const url = new URL(request.url());
    if (url.pathname.startsWith('/api/v1/')) requests.push({ type: 'request', method: request.method(), path: url.pathname, query: url.search, headers: request.headers(), body: request.postData() || '' });
  });
  page.on('response', response => {
    const url = new URL(response.url());
    if (url.pathname.startsWith('/api/v1/')) requests.push({ type: 'response', method: response.request().method(), path: url.pathname, query: url.search, status: response.status() });
  });
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner' && document.querySelector('#tb-title')?.textContent === '工作台');
  record('Chrome renders an authenticated real OWNER PC session', true);

  const materialized = await materializePending(pendingIDs[0]);
  const pendingRefund = await refundPending(pendingIDs[1]);
  const orderRefund = await exerciseOrders(materialized.order);
  await waitForRefunds(materialized.order, [pendingRefund.refund.id, orderRefund.refund.id]);
  await exerciseDashboard();
  await exerciseFinance(materialized.order, [pendingRefund.refund, orderRefund.refund]);
  await page.screenshot({ path: path.join(evidenceRoot, 'final-finance.png'), fullPage: true });
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
  if (page) await page.screenshot({ path: path.join(evidenceRoot, 'failure.png'), fullPage: true }).catch(() => {});
} finally {
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
}

for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
const passed = !failure && checks.every(item => item.ok);
const receipt = {
  schema: 'order.pc-transactions-ui1.v1', candidate_sha: candidateSHA, generated_at: new Date().toISOString(),
  browser: browserVersion, upstream: apiOrigin, evidence_level: 'L3_LOCAL_COMPOSED', payment_provider: 'local-fake-auto-pay-disabled',
  callback_success_time: 'actual local time after effective_deadline; synthetic signed local callback only',
  pending_ids: pendingIDs, setup: 'real Mini UI1 HTTP Quote -> prepay -> two 202 confirms; no SQL mutation',
  cleanup: { temporary_catalog: 'none created', settings_and_launch: 'not mutated', api_mode_restore: 'controller-owned after runner' },
  status: passed ? 'PASS' : 'FAIL', checks,
  requests: requests.filter(item => item.type === 'response'),
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_TRANSACTION_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, receipt: path.join(evidenceRoot, 'receipt.json') })}\n`);
if (!passed) process.exitCode = 1;

async function materializePending(id) {
  await navigate('pending', '支付待处理', '#pd-host');
  const row = page.locator(`tr[data-id="${id}"]`);
  await row.waitFor();
  const response = await uiMutation('POST', `/api/v1/admin/pending-payments/${id}`, 200, async () => {
    await row.locator('[data-act="build"]').click();
    await page.locator('.modal [data-a="ok"]').click();
  });
  const order = exactNested(response.body, 'order');
  await page.waitForFunction(value => !document.querySelector(`tr[data-id="${value}"]`), id);
  record('PC04 visible MATERIALIZE creates one server order and removes the pending row', !!order.order_no && !!order.pickup_number);

  const replay = await adminRequest(`/api/v1/admin/pending-payments/${id}`, {
    method: 'POST', idempotencyKey: response.key, body: response.requestBody,
  }, 200);
  record('PC04 MATERIALIZE idempotent replay returns the same order', replay.order?.id === order.id && replay.order?.order_no === order.order_no);
  return { order };
}

async function refundPending(id) {
  const row = page.locator(`tr[data-id="${id}"]`);
  await row.waitFor();
  await row.locator('[data-act="void"]').click();
  const invalid = await uiMutation('POST', `/api/v1/admin/pending-payments/${id}`, 400, () => page.locator('.modal [data-a="ok"]').click());
  record('PC04 blank pending-refund reason fails closed and keeps the modal/row',
    invalid.body?.error?.code === 'INVALID_REQUEST' && await page.locator('.modal').isVisible() && await row.isVisible());

  const reason = `UI1待支付作废-${candidateSHA.slice(0, 8)}`;
  await page.locator('.modal #pv-why').fill(reason);
  const response = await uiMutation('POST', `/api/v1/admin/pending-payments/${id}`, 200, () => page.locator('.modal [data-a="ok"]').click());
  const refund = exactNested(response.body, 'refund');
  await page.waitForFunction(value => document.querySelector('#toast-host')?.textContent.includes(value), '已作废');
  record('PC04 visible REFUND creates one durable refund and shows the accepted result', refund.state === '退款中' && Number(refund.amount_cents) > 0);

  const replay = await adminRequest(`/api/v1/admin/pending-payments/${id}`, {
    method: 'POST', idempotencyKey: response.key, body: response.requestBody,
  }, 200);
  const conflict = await adminRequest(`/api/v1/admin/pending-payments/${id}`, {
    method: 'POST', idempotencyKey: response.key, body: { action: 'REFUND', reason: `${reason}-不同` },
  }, 409);
  record('PC04 pending REFUND duplicate is stable and conflicting reuse is 409', replay.refund?.id === refund.id && conflict.error?.code === 'COMMAND_CONFLICT');
  return { refund };
}

async function exerciseOrders(order) {
  await navigate('orders', '订单管理', '#tbl-host');
  const laneText = await page.locator('#lanes').innerText();
  record('PC02 all six order states plus all-lane are visible', ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款', '全部'].every(label => laneText.includes(label)));
  const searchResponse = page.waitForResponse(response => new URL(response.url()).pathname === '/api/v1/admin/orders' && new URL(response.url()).searchParams.get('q') === order.order_no && response.status() === 200);
  await page.locator('#f-kw').fill(order.order_no);
  await searchResponse;
  await page.waitForFunction(value => document.querySelector('#detail')?.textContent.includes(value), order.order_no);
  const detail = await page.locator('#detail').innerText();
  record('PC02 search renders immutable item/payment detail without invalid money', detail.includes(order.order_no) && detail.includes('实付') && !detail.includes('¥—'));

  await page.locator('#detail [data-refund]').click();
  const invalid = await uiMutation('POST', `/api/v1/admin/orders/${order.id}/refund`, 400, () => page.locator('.modal [data-a="ok"]').click());
  record('PC02 blank OWNER order-refund reason fails closed', invalid.body?.error?.code === 'INVALID_REQUEST' && await page.locator('.modal').isVisible());
  const reason = `UI1订单退款-${candidateSHA.slice(0, 8)}`;
  await page.locator('.modal #rf-why').fill(reason);
  const response = await uiMutation('POST', `/api/v1/admin/orders/${order.id}/refund`, 200, () => page.locator('.modal [data-a="ok"]').click());
  const refund = exactNested(response.body, 'refund');
  const projected = exactNested(response.body, 'order');
  await page.waitForFunction(value => document.querySelector('#detail')?.textContent.includes(value), '退款中');
  record('PC02 visible OWNER full refund projects the order to REFUNDING', projected.state === '退款中' && refund.state === '退款中');

  const replay = await adminRequest(`/api/v1/admin/orders/${order.id}/refund`, {
    method: 'POST', idempotencyKey: response.key, body: response.requestBody,
  }, 200);
  const conflict = await adminRequest(`/api/v1/admin/orders/${order.id}/refund`, {
    method: 'POST', idempotencyKey: response.key, body: { reason: `${reason}-不同` },
  }, 409);
  record('PC02 order REFUND duplicate is stable and conflicting reuse is 409', replay.refund?.id === refund.id && conflict.error?.code === 'COMMAND_CONFLICT');

  await page.locator('[data-lane="待取餐"]').click();
  await page.waitForResponse(response => {
    const url = new URL(response.url());
    return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('state') === '待取餐' && response.status() === 200;
  });
  const unclaimedResponse = page.waitForResponse(response => {
    const url = new URL(response.url());
    return url.pathname === '/api/v1/admin/orders' && url.searchParams.get('unclaimed') === 'true' && response.status() === 200;
  });
  await page.locator('[data-unc]').click();
  await unclaimedResponse;
  record('PC02 unclaimed is a real server query, not a seventh persisted state', (await page.locator('#unc-host').innerText()).includes('营业日已结束'));
  return { refund };
}

async function waitForRefunds(order, refundIDs) {
  await waitFor(async () => {
    const orders = await adminRequest(`/api/v1/admin/orders?q=${encodeURIComponent(order.order_no)}`, { method: 'GET' }, 200);
    const refunds = await adminRequest('/api/v1/admin/finance/refunds?from=2026-01-01&to=2026-12-31', { method: 'GET' }, 200);
    const orderDone = orders.orders.some(item => String(item.id) === String(order.id) && item.state === '已退款');
    const refundDone = refundIDs.every(id => refunds.refunds.some(item => String(item.id) === String(id) && item.state === '已退款'));
    return orderDone && refundDone;
  }, 20_000, 'local refund worker did not reach REFUNDED');

  await navigate('pending', '支付待处理', '#pd-host');
  await page.waitForFunction(values => values.every(value => !document.querySelector(`tr[data-id="${value}"]`)), pendingIDs);
  record('PC04 completed pending refund disappears after server refund finality is re-read', true);

  await navigate('orders', '订单管理', '#tbl-host');
  const response = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/admin/orders' && new URL(r.url()).searchParams.get('q') === order.order_no && r.status() === 200);
  await page.locator('#f-kw').fill(order.order_no);
  await response;
  await page.waitForFunction(() => document.querySelector('#detail')?.textContent.includes('已退款'));
  record('PC02 local refund worker reaches final REFUNDED visible in PC UI', true);
}

async function exerciseDashboard() {
  const stats = await adminRequest('/api/v1/admin/stats', { method: 'GET' }, 200);
  await navigate('dashboard', '工作台', '.kpi');
  const text = await page.locator('#content').innerText();
  const expected = [
    `¥${yuan(stats.today_revenue_cents)}`, String(stats.today_orders),
    `¥${yuan(stats.month_revenue_cents)}`, String(stats.month_orders), `¥${yuan(stats.refund_cents)}`,
  ];
  record('PC01 dashboard matches server day/month revenue/orders and refunds', expected.every(value => text.includes(value)));
  record('PC01 pending-production and product sales are server-backed and visible',
    Number(stats.pending_production) === 0 ? text.includes('暂无待办') : text.includes(String(stats.pending_production)) && text.includes('单待制作'),
  );
}

async function exerciseFinance(order, refunds) {
  await navigate('finance', '财务与对账', '#sum-host');
  const responsesBeforeInvalid = () => requests.filter(item => item.type === 'response' && item.path.startsWith('/api/v1/admin/finance/')).length;
  const rangeResponse = page.waitForResponse(r => new URL(r.url()).pathname === '/api/v1/admin/finance/summary' && r.status() === 200);
  await page.locator('[data-quick="7"]').click();
  await rangeResponse;
  await page.waitForTimeout(250);
  const from = await page.locator('#f-from').inputValue();
  const to = await page.locator('#f-to').inputValue();
  const summary = await adminRequest(`/api/v1/admin/finance/summary?from=${from}&to=${to}`, { method: 'GET' }, 200);
  const summaryText = await page.locator('#sum-host').innerText();
  record('PC03 finance summary shows exact gross/refund/net and counts',
    summaryText.includes(`¥${yuan(summary.gross)}`) && summaryText.includes(`−¥${yuan(summary.refundAmount)}`) && summaryText.includes(`${summary.count} 笔收款`));
  await page.waitForFunction(value => document.querySelector('#fin-host')?.textContent.includes(value), order.order_no);
  record('PC03 payment detail visibly includes the materialized real order', (await page.locator('#fin-host').innerText()).includes(order.order_no));

  await page.locator('[data-tab="refund"]').click();
  await page.waitForFunction(values => values.every(value => document.querySelector('#fin-host')?.textContent.includes(value)), refunds.map(item => item.reason));
  record('PC03 refund detail visibly includes both real full refunds', true);

  const downloadPromise = page.waitForEvent('download');
  await page.locator('[data-export]').click();
  const download = await downloadPromise;
  const downloadPath = await download.path();
  const csv = downloadPath ? execFileSync('/bin/cat', [downloadPath], { encoding: 'utf8' }) : '';
  record('PC03 export downloads server CSV containing the real order', csv.includes('订单号') && csv.includes(order.order_no));

  await page.waitForTimeout(250);
  const before = responsesBeforeInvalid();
  await page.evaluate(({ invalidFrom, invalidTo }) => {
    const fromInput = document.querySelector('#f-from');
    const toInput = document.querySelector('#f-to');
    fromInput.value = invalidFrom;
    toInput.value = invalidTo;
    toInput.dispatchEvent(new Event('change', { bubbles: true }));
  }, { invalidFrom: to, invalidTo: addISODate(to, -1) });
  await page.waitForTimeout(150);
  const after = responsesBeforeInvalid();
  record('PC03 reversed date range is rejected before any finance request', before === after && (await page.locator('#toast-host').innerText()).includes('起始日期不能晚于结束日期'));
}

async function navigate(route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).first().waitFor();
  await page.waitForTimeout(80);
}

async function uiMutation(method, pathname, expectedStatus, trigger) {
  const responsePromise = page.waitForResponse(response => response.request().method() === method && new URL(response.url()).pathname === pathname);
  await trigger();
  const response = await responsePromise;
  const body = await response.json().catch(() => ({}));
  if (response.status() !== expectedStatus) throw new Error(`${method} ${pathname} returned ${response.status()}`);
  const request = response.request();
  const key = (await request.allHeaders())['idempotency-key'] || '';
  let requestBody = {};
  try { requestBody = request.postDataJSON(); } catch {}
  return { body, key, requestBody };
}

function readPendingFacts() {
  const list = pendingIDs.join(',');
  const sql = `SELECT JSON_OBJECT('id',CAST(id AS CHAR),'materialization_state',materialization_state,'provider_state',provider_state,'effective_deadline',DATE_FORMAT(effective_deadline,'%Y-%m-%dT%H:%i:%s.%fZ'),'request',provider_create_request_json) FROM prepayments WHERE id IN (${list}) ORDER BY id`;
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysql.password}`, mysql.container,
    'mysql', '--batch', '--raw', '--skip-column-names', `-u${mysql.user}`, `--database=${mysql.database}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim();
  const facts = output ? output.split('\n').map(line => JSON.parse(line)) : [];
  if (facts.length !== 2 || facts.some(fact => !pendingIDs.includes(String(fact.id)))) throw new Error('requested prepayment facts are missing');
  return facts;
}

async function waitForEffectiveDeadline(facts) {
  const deadline = Math.max(...facts.map(fact => Date.parse(fact.effective_deadline)));
  if (!Number.isFinite(deadline)) throw new Error('prepayment deadline is invalid');
  let lastNotice = 0;
  while (Date.now() <= deadline + 100) {
    const remaining = deadline + 101 - Date.now();
    if (Date.now() - lastNotice >= 30_000 || lastNotice === 0) {
      process.stdout.write(`PC_TRANSACTION_UI1_WAIT ${JSON.stringify({ seconds_remaining: Math.max(0, Math.ceil(remaining / 1000)) })}\n`);
      lastNotice = Date.now();
    }
    await new Promise(resolve => setTimeout(resolve, Math.min(1000, Math.max(20, remaining))));
  }
}

async function sendLatePaymentCallback(fact) {
  const request = fact.request;
  const amount = Number(request.amount_cents);
  const successTime = new Date().toISOString();
  const providerExpiry = Date.parse(request.time_expire);
  if (!Number.isFinite(Date.parse(successTime)) || !Number.isFinite(providerExpiry) || Date.parse(successTime) < Date.parse(fact.effective_deadline)) throw new Error('callback success time is before effective deadline');
  const body = Buffer.from(JSON.stringify({
    ID: `pc-ui1-late-${fact.id}-${randomUUID()}`,
    Transaction: {
      AppID: request.appid, MerchantID: request.mchid, OutTradeNo: request.out_trade_no,
      TransactionID: `LOCAL-PC-UI1-${fact.id}`, TradeType: 'JSAPI', TradeState: 'SUCCESS',
      TradeStateDescription: '', BankType: '', Attach: '', SuccessTime: successTime,
      Payer: { openid: request.payer_openid },
      Amount: { Total: amount, PayerTotal: amount, Currency: request.currency, PayerCurrency: request.currency },
    },
  }));
  const secret = createHash('sha256').update('order-paymentorder-local-fake-v1').digest();
  const signature = createHmac('sha256', secret).update(body).digest('hex');
  const response = await fetch(`${apiOrigin}/api/v1/payments/wechat/notify`, {
    method: 'POST', body,
    headers: {
      'Content-Type': 'application/json', 'Wechatpay-Serial': 'FAKE', 'Wechatpay-Signature': signature,
      'Wechatpay-Timestamp': String(Math.floor(Date.now() / 1000)), 'Wechatpay-Nonce': 'pc-ui1-late',
    },
    redirect: 'error',
  });
  if (response.status !== 204) throw new Error(`late callback for prepayment ${fact.id} returned ${response.status}`);
}

async function acquirePCSession() {
  const session = await jsonRequest('/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-transactions-${randomUUID()}` } }, 201);
  const miniToken = exactToken(session.access_token, 'Mini session');
  await jsonRequest('/api/v1/me/bind-phone', { method: 'POST', bearer: miniToken, idempotencyKey: randomUUID(), body: { code: `pc-transactions-phone-${randomUUID()}` } }, 200);
  const login = await jsonRequest('/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  const loginID = exactString(login.login_id, 'login_id');
  await jsonRequest('/api/v1/me/admin-login/approve', {
    method: 'POST', bearer: miniToken,
    body: { login_id: loginID, approval_secret: exactString(payload.searchParams.get('approval_secret'), 'approval_secret'), code: `pc-transactions-approve-${randomUUID()}` },
  }, 200);
  const poll = await jsonRequest('/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll_secret') } }, 200);
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return { miniToken, pcToken: exactToken(poll.session.token, 'PC session') };
}

async function adminRequest(pathname, options, expectedStatus) {
  return jsonRequest(pathname, Object.assign({}, options, { bearer: pcToken }), expectedStatus);
}

async function jsonRequest(pathname, options, expectedStatus) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
  });
  const raw = await response.text();
  let body = {};
  if (raw) { try { body = JSON.parse(raw); } catch { body = raw; } }
  if (response.status !== expectedStatus) {
    const code = body?.error?.code || '';
    throw new Error(`${options.method} ${pathname} returned ${response.status}${code ? `/${code}` : ''}`);
  }
  return body;
}

async function waitFor(predicate, timeout, message) {
  const deadline = Date.now() + timeout;
  while (!(await predicate())) {
    if (Date.now() >= deadline) throw new Error(message);
    await new Promise(resolve => setTimeout(resolve, 250));
  }
}

function exactNested(body, key) {
  const value = body && body[key];
  if (!value || typeof value !== 'object' || Array.isArray(value) || !value.id) throw new Error(`response omitted nested ${key}`);
  return value;
}

function record(name, ok) { checks.push({ name, ok: Boolean(ok) }); }
function safeMessage(error) { return String(error?.message || error || 'unknown').replace(/[\r\n]+/g, ' ').slice(0, 240); }
function yuan(cents) { const value = Math.abs(Math.round(Number(cents) || 0)); return `${Math.floor(value / 100)}.${String(value % 100).padStart(2, '0')}`; }
function addISODate(value, days) { const date = new Date(`${value}T00:00:00Z`); date.setUTCDate(date.getUTCDate() + days); return date.toISOString().slice(0, 10); }

function exactIDs(value) {
  const ids = String(value).split(',').map(item => item.trim());
  if (ids.length !== 2 || ids.some(id => !/^[1-9]\d*$/.test(id)) || ids[0] === ids[1]) throw new Error('ORDER_PC_PENDING_IDS must contain two distinct decimal IDs');
  return ids;
}
function exactIdentifier(value, label, allowDash = false) {
  const pattern = allowDash ? /^[A-Za-z0-9_.-]+$/ : /^[A-Za-z0-9_]+$/;
  if (typeof value !== 'string' || !pattern.test(value)) throw new Error(`${label} is invalid`);
  return value;
}
function exactToken(value, label) { const token = exactString(value, label); if (token.length < 32) throw new Error(`${label} is malformed`); return token; }
function exactString(value, label) { if (typeof value !== 'string' || value.trim() === '') throw new Error(`${label} is missing`); return value; }
function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) throw new Error('ORDER_COMPOSED_API_ORIGIN must be exact loopback');
  return parsed.origin;
}

async function startSameOriginProxy(upstreamOrigin) {
  const server = http.createServer(async (request, response) => {
    try {
      if (request.url.startsWith('/api/')) { await proxyAPI(request, response, upstreamOrigin); return; }
      serveStatic(request, response);
    } catch {
      response.writeHead(502, { 'Content-Type': 'text/plain; charset=utf-8' });
      response.end('local composed proxy unavailable');
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
  response.writeHead(upstream.status, { ...(contentType ? { 'Content-Type': contentType } : {}), 'Content-Length': body.length, 'Cache-Control': 'no-store' });
  response.end(body);
}

function serveStatic(request, response) {
  const pathname = decodeURIComponent(new URL(request.url, 'http://127.0.0.1').pathname);
  const relative = pathname === '/' ? 'web-admin/index.html' : pathname.replace(/^\/+/, '');
  const target = path.resolve(appsRoot, relative);
  if (!target.startsWith(`${appsRoot}${path.sep}`) || !existsSync(target) || !statSync(target).isFile()) { response.writeHead(404); response.end('not found'); return; }
  const contentType = target.endsWith('.js') ? 'text/javascript; charset=utf-8' : target.endsWith('.css') ? 'text/css; charset=utf-8' : target.endsWith('.png') ? 'image/png' : 'text/html; charset=utf-8';
  response.writeHead(200, { 'Content-Type': contentType, 'Cache-Control': 'no-store' });
  createReadStream(target).pipe(response);
}
