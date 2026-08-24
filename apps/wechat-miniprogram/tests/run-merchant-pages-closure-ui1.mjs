import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { createRequire } from 'node:module';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const testRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(testRoot, '../../..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS;
const apiOrigin = exactOrigin(process.env.ORDER_MERCHANT_CLOSURE_API_ORIGIN);
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const receiptPath = path.resolve(process.env.ORDER_MERCHANT_CLOSURE_RECEIPT_PATH || `/private/tmp/order-merchant-closure-${candidateSHA}/receipt.json`);
const runID = `merchant-closure-${Date.now()}-${randomUUID().slice(0, 8)}`;
const requestLog = [];
const dependencyRequire = createRequire(path.join(dependencyRoot || '', 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const browserVersion = existsSync(browserPath) ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const fixture = { mini_token: '', expires_at: '', lanes: {}, products: {} };
let proxy;
let exitCode = 1;
let failure = '';

if (!dependencyRoot || !browserVersion) throw new Error('reuse MINIPROGRAM_UI_DEPS with its locked Chromium');

try {
  const session = await request('POST', '/api/v1/auth/miniprogram/session', { body: { code: `${runID}-session` }, expected: 201 });
  fixture.mini_token = exactString(session.access_token, 'mini access token');
  fixture.expires_at = exactString(session.expires_at, 'mini expiry');
  await request('POST', '/api/v1/me/bind-phone', { bearer: fixture.mini_token, body: { code: `${runID}-phone` } });
  await request('POST', '/api/v1/me/merchant-login', { bearer: fixture.mini_token, body: { code: `${runID}-merchant-phone` } });
  const pcToken = await acquirePCSession(fixture.mini_token);
  const settings = await request('GET', '/api/v1/admin/settings', { bearer: pcToken });
  const schedule = closureSchedule(settings);
  await request('PUT', '/api/v1/admin/settings', {
    bearer: pcToken, key: key('settings'), body: schedule.settings,
  });

  const categoryView = await request('POST', '/api/v1/admin/categories', {
    bearer: pcToken, key: key('category'), expected: 201, body: { name: `商户闭环-${runID}` },
  });
  const categoryID = exactID(exactObject(categoryView.category, 'category').id, 'category id');
  for (const meal of ['lunch', 'dinner']) {
    const productView = await request('POST', '/api/v1/admin/products', {
      bearer: pcToken, key: key(`product-${meal}`), expected: 201,
      body: {
        name: `${meal === 'lunch' ? '午餐' : '晚餐'}闭环-${runID}`, price_cents: meal === 'lunch' ? 1280 : 1680,
        category_id: categoryID, meal_period: meal, description: '商户页面本地闭环', images: [],
      },
    });
    fixture.products[meal] = exactID(exactObject(productView.product, `product ${meal}`).id, `product ${meal} id`);
  }

  const make = async (name, time, productID) => materializeOrder(fixture.mini_token, schedule.today, time, productID, name);
  const reserved = await make('reserved', schedule.farTime, fixture.products.dinner);
  const preparing = await make('preparing', schedule.nearTime, fixture.products.lunch);
  const readyPage = await make('ready-page', schedule.nearTime, fixture.products.lunch);
  const scan = await make('scan', schedule.nearTime, fixture.products.lunch);
  const manual = await make('manual', schedule.nearTime, fixture.products.lunch);
  const completed = await make('completed', schedule.nearTime, fixture.products.lunch);
  const refunded = await make('refunded', schedule.nearTime, fixture.products.lunch);
  for (const order of [preparing, readyPage, scan, manual, completed, refunded]) {
    await waitOrderState(order.id, 'PREPARING');
  }
  for (const order of [scan, manual, completed, refunded]) await markReady(order.id);

  const scanDetail = await userOrder(scan.id);
  const manualDetail = await userOrder(manual.id);
  const completedDetail = await userOrder(completed.id);
  const refundedDetail = await userOrder(refunded.id);
  await request('POST', '/api/v1/verify/scan', {
    bearer: fixture.mini_token, key: key('complete-seed'), body: { token: exactString(completedDetail.redemption_token, 'completed token') },
  });
  await request('POST', '/api/v1/verify/scan', {
    bearer: fixture.mini_token, key: key('refund-complete'), body: { token: exactString(refundedDetail.redemption_token, 'refunded token') },
  });
  await request('POST', `/api/v1/admin/orders/${refunded.id}/refund`, {
    bearer: pcToken, key: key('refund'), body: { reason: '商户页面闭环退款' },
  });
  await waitOrderState(refunded.id, 'REFUNDED', 15000);

  fixture.lanes = {
    已预约: reserved.id, 制作中: preparing.id, 待取餐: scan.id, 已完成: completed.id, 已退款: refunded.id,
  };
  fixture.search_order_id = completed.id;
  fixture.search_order_no = completed.orderNo;
  fixture.ready_order_id = readyPage.id;
  fixture.scan_order_id = scan.id;
  fixture.scan_token = exactString(scanDetail.redemption_token, 'scan token');
  fixture.manual_order_id = manual.id;
  fixture.manual_code = exactCode(manualDetail.pickup_number);
  fixture.refunded_token = exactString(refundedDetail.redemption_token, 'refunded replay token');
  fixture.today = schedule.today;
  fixture.tomorrow = schedule.tomorrow;
  fixture.near_time = schedule.nearTime;
  fixture.far_time = schedule.farTime;

  proxy = await startProxy(apiOrigin, requestLog);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_MERCHANT_CLOSURE_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_MERCHANT_CLOSURE_FIXTURE = JSON.stringify(fixture);
  process.stdout.write(`MERCHANT_CLOSURE_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin, proxy: proxy.origin, schema: process.env.ORDER_MERCHANT_CLOSURE_MYSQL_DATABASE })}\n`);
  const config = await karma.config.parseConfig(
    path.join(testRoot, 'merchant-pages-closure-karma.cjs'), { singleRun: true }, { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(config, resolve);
    server.start().catch(reject);
  });
  if (exitCode !== 0) throw new Error(`rendered Karma gate exited ${exitCode}`);

  const tomorrowMenu = await request('GET', `/api/v1/menu?date=${encodeURIComponent(schedule.tomorrow)}&time=${encodeURIComponent(schedule.nearTime)}`, { bearer: fixture.mini_token });
  const tomorrowProduct = (tomorrowMenu.categories || []).flatMap(item => item.products || []).find(item => item.id === fixture.products.lunch);
  if (!tomorrowProduct || tomorrowProduct.sold_out !== false) throw new Error('PAGE-M05 today sold-out leaked into tomorrow');
  const mysqlEvidence = verifyMySQL(fixture);
  const requiredPaths = [
    '/api/v1/merchant/orders', '/api/v1/merchant/store-status', `/api/v1/merchant/orders/${fixture.ready_order_id}/ready`,
    '/api/v1/verify/scan', '/api/v1/verify/code', `/api/v1/merchant/products/${fixture.products.lunch}/soldout`,
  ];
  for (const pathname of requiredPaths) {
    if (!requestLog.some(item => item.path.startsWith(pathname) && item.status >= 200 && item.status < 300)) {
      throw new Error(`rendered gate omitted successful root request ${pathname}`);
    }
  }
  writeReceipt('PASS', mysqlEvidence);
} catch (error) {
  failure = safeMessage(error);
  writeReceipt('FAIL', []);
  process.exitCode = 1;
} finally {
  if (proxy) await proxy.close().catch(() => {});
}

function writeReceipt(status, mysqlEvidence) {
  const receipt = {
    schema: 'order.merchant-pages-closure.ui1.v1', candidate_sha: candidateSHA,
    generated_at: new Date().toISOString(), status, evidence_level: 'L3_LOCAL_COMPOSED', browser: browserVersion,
    cases: ['PAGE-M02', 'PAGE-M03', 'PAGE-M04', 'PAGE-M05'], fixture: redactFixture(fixture),
    requests: requestLog, mysql_evidence: mysqlEvidence,
    external: [{ case: 'PAGE-M04', level: 'UI3', status: 'BLOCKED_EXTERNAL', reason: 'real camera requires logged-in WeChat DevTools or device' }],
    error: failure || undefined,
  };
  mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
  writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
  process.stdout.write(`MERCHANT_CLOSURE_UI1_RESULT ${JSON.stringify({ status, candidate_sha: candidateSHA, scenarios: 4, requests: requestLog.length, receipt: receiptPath, error: failure || undefined })}\n`);
}

async function acquirePCSession(miniToken) {
  const login = await request('POST', '/api/v1/admin/auth/qrcode', { body: {}, expected: 201 });
  const qr = new URL(exactString(login.qr_payload, 'admin QR payload'));
  await request('POST', '/api/v1/me/admin-login/approve', {
    bearer: miniToken,
    body: { login_id: login.login_id, approval_secret: qr.searchParams.get('approval_secret'), code: `${runID}-admin-phone` },
  });
  const poll = await request('POST', '/api/v1/admin/auth/poll', { body: { login_id: login.login_id, poll_secret: login.poll_secret } });
  if (poll.state !== 'APPROVED') throw new Error(`PC session ended ${poll.state}`);
  return exactString(exactObject(poll.session, 'PC session').token, 'PC token');
}

async function materializeOrder(token, date, pickupTime, productID, label) {
  const prefix = `${runID}-${label}`;
  const quoteView = await request('POST', '/api/v1/quotes', {
    bearer: token, key: `${prefix}-quote`, expected: 201,
    body: { contact_name: `闭环${label}`, pickup_date: date, pickup_time: pickupTime, order_note: prefix,
      items: [{ product_id: productID, quantity: 1, flavors: [], note: '' }] },
  });
  const quoteID = exactID(exactObject(quoteView.quote, 'quote').id, 'quote id');
  const prepayView = await request('POST', '/api/v1/orders/prepay', {
    bearer: token, key: `${prefix}-prepay`, expected: 201, body: { quote_id: quoteID },
  });
  const prepaymentID = exactID(exactObject(prepayView.prepayment, 'prepayment').id, 'prepayment id');
  const confirm = await request('POST', '/api/v1/orders/confirm', {
    bearer: token, key: `${prefix}-confirm`, body: { prepayment_id: prepaymentID },
  });
  if (confirm.state !== 'ORDER_CREATED') throw new Error(`${label} confirmation ended ${confirm.state}`);
  const id = exactID(confirm.order_id, `${label} order id`);
  const detail = await userOrder(id);
  return { id, orderNo: exactString(detail.order_no, `${label} order number`) };
}

async function markReady(id) {
  const view = await request('POST', `/api/v1/merchant/orders/${id}/ready`, { bearer: fixture.mini_token, key: key(`ready-${id}`), body: {} });
  if (exactObject(view.order, 'ready order').state !== 'READY_FOR_PICKUP') throw new Error(`order ${id} did not become READY`);
}

async function userOrder(id) {
  const view = await request('GET', `/api/v1/orders/${id}`, { bearer: fixture.mini_token });
  return exactObject(view.order, `order ${id}`);
}

async function waitOrderState(id, wanted, timeout = 8000) {
  const deadline = Date.now() + timeout;
  let state = '';
  do {
    state = exactString((await userOrder(id)).state, `order ${id} state`);
    if (state === wanted) return;
    await new Promise(resolve => setTimeout(resolve, 250));
  } while (Date.now() < deadline);
  throw new Error(`order ${id} ended ${state}, want ${wanted}`);
}

function closureSchedule(settings) {
  const now = new Date();
  const near = new Date(Math.ceil((now.getTime() + 12 * 60000) / (5 * 60000)) * 5 * 60000);
  const far = new Date(near.getTime() + 60 * 60000);
  const cutoff = new Date(near.getTime() - 5 * 60000);
  const today = shanghai(now).date;
  const tomorrow = shanghai(new Date(now.getTime() + 24 * 60 * 60000)).date;
  if (shanghai(near).date !== today || shanghai(far).date !== today) throw new Error('closure schedule crossed Shanghai day');
  const nearTime = shanghai(near).time;
  const farTime = shanghai(far).time;
  return {
    today, tomorrow, nearTime, farTime,
    settings: {
      store_status: 'open', pickup_point: settings.pickup_point, notice: settings.notice, pickup_step_min: 5,
      meal_periods: [
        { code: 'lunch', name: '午餐', cutoff_time: shanghai(cutoff).time, pickup_from: nearTime, pickup_to: nearTime },
        { code: 'dinner', name: '晚餐', cutoff_time: shanghai(cutoff).time, pickup_from: farTime, pickup_to: farTime },
      ],
      service_dates: [{ date: today, status: 'open' }, { date: tomorrow, status: 'open' }],
    },
  };
}

function shanghai(value) {
  const date = new Date(value.getTime() + 8 * 60 * 60000);
  const pad = number => String(number).padStart(2, '0');
  return { date: `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`, time: `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}` };
}

async function request(method, pathname, options = {}) {
  const headers = {};
  if (options.bearer) headers.authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['idempotency-key'] = options.key;
  if (options.body !== undefined) headers['content-type'] = 'application/json';
  const response = await fetch(`${apiOrigin}${pathname}`, { method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body) });
  const raw = await response.text();
  let body = {};
  if (raw) try { body = JSON.parse(raw); } catch (error) { body = {}; }
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  requestLog.push({ phase: 'setup-or-evidence', method, path: pathname, status: response.status });
  if (!expected.includes(response.status)) throw new Error(`${method} ${pathname} returned ${response.status}/${body.error && body.error.code || 'UNKNOWN'}`);
  return body;
}

function verifyMySQL(values) {
  const container = process.env.ORDER_MERCHANT_CLOSURE_MYSQL_CONTAINER;
  const database = process.env.ORDER_MERCHANT_CLOSURE_MYSQL_DATABASE;
  const user = process.env.ORDER_MERCHANT_CLOSURE_MYSQL_USER;
  const password = process.env.ORDER_MERCHANT_CLOSURE_MYSQL_PASSWORD;
  if (!/^[A-Za-z0-9_.-]+$/.test(container || '') || !/^[A-Za-z0-9_]+$/.test(database || '') || !user || !password) throw new Error('MySQL evidence settings invalid');
  const sql = `SELECT state,COUNT(*) FROM orders GROUP BY state ORDER BY state; SELECT action,COUNT(*) FROM action_audits WHERE action IN ('store.status.set','fulfillment.mark_ready','fulfillment.redeem_token','fulfillment.redeem_current_date_code','product.sold_out.set') GROUP BY action ORDER BY action; SELECT service_date FROM product_sold_out_dates WHERE product_id=${Number(values.products.lunch)} ORDER BY service_date;`;
  const output = execFileSync('/opt/homebrew/bin/docker', ['exec', '-e', `MYSQL_PWD=${password}`, container,
    'mysql', '--batch', '--raw', '--skip-column-names', '-u', user, `--database=${database}`, '--execute', sql], { encoding: 'utf8' }).trim();
  const lines = output.split('\n').filter(Boolean);
  for (const state of ['RESERVED', 'PREPARING', 'READY_FOR_PICKUP', 'COMPLETED', 'REFUNDED']) {
    if (!lines.some(line => line.startsWith(`${state}\t`))) throw new Error(`MySQL omitted ${state}`);
  }
  if (!lines.includes(values.today) || lines.includes(values.tomorrow)) throw new Error('MySQL sold-out date scope mismatch');
  return lines;
}

function redactFixture(values) {
  return {
    lanes: values.lanes, search_order_id: values.search_order_id, ready_order_id: values.ready_order_id,
    scan_order_id: values.scan_order_id, manual_order_id: values.manual_order_id,
    today: values.today, tomorrow: values.tomorrow, products: values.products,
  };
}

async function startProxy(origin, log) {
  const target = new URL(origin);
  const server = http.createServer((incoming, outgoing) => {
    if (incoming.method === 'OPTIONS') {
      outgoing.writeHead(204, corsHeaders()); outgoing.end(); return;
    }
    const headers = Object.assign({}, incoming.headers, { host: target.host });
    delete headers.connection;
    const upstream = http.request({ hostname: target.hostname, port: target.port, method: incoming.method, path: incoming.url, headers }, response => {
      log.push({ phase: 'rendered-ui', method: incoming.method, path: incoming.url, status: response.statusCode || 0 });
      const responseHeaders = Object.assign({}, response.headers, corsHeaders());
      delete responseHeaders.connection;
      outgoing.writeHead(response.statusCode || 502, responseHeaders); response.pipe(outgoing);
    });
    upstream.on('error', () => { outgoing.writeHead(502, corsHeaders()); outgoing.end('{}'); });
    incoming.pipe(upstream);
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { origin: `http://127.0.0.1:${server.address().port}`, close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())) };
}

function corsHeaders() {
  return { 'access-control-allow-origin': '*', 'access-control-allow-methods': 'GET,POST,PUT,OPTIONS', 'access-control-allow-headers': 'authorization,content-type,idempotency-key' };
}
function key(scope) { return `${runID}-${scope}-${randomUUID()}`; }
function exactOrigin(value) { if (!/^http:\/\/127\.0\.0\.1:\d{1,5}$/.test(value || '')) throw new Error('explicit loopback API origin required'); return value; }
function exactObject(value, name) { if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${name} missing`); return value; }
function exactString(value, name) { if (typeof value !== 'string' || !value) throw new Error(`${name} missing`); return value; }
function exactID(value, name) { const id = exactString(String(value || ''), name); if (!/^[1-9]\d*$/.test(id)) throw new Error(`${name} invalid`); return id; }
function exactCode(value) { const code = exactString(value, 'pickup number'); if (!/^\d{4}$/.test(code)) throw new Error('pickup number is not four digits'); return code; }
function safeMessage(error) { return String(error && error.message || 'unknown failure').replace(/[\r\n]+/g, ' ').slice(0, 500); }
