import { execFileSync } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { createRequire } from 'node:module';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolRoot, '../..');
const dependencyRoot = exactString(process.env.MINIPROGRAM_UI_DEPS, 'MINIPROGRAM_UI_DEPS');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const browserVersion = existsSync(browserPath) ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const apiOrigin = exactOrigin(process.env.ORDER_MERCHANT_FAILURE_API_ORIGIN);
const mysqlContainer = exactName(process.env.ORDER_MERCHANT_FAILURE_MYSQL_CONTAINER, 'MySQL container');
const mysqlDatabase = exactName(process.env.ORDER_MERCHANT_FAILURE_MYSQL_DATABASE, 'MySQL database');
const mysqlUser = exactName(process.env.ORDER_MERCHANT_FAILURE_MYSQL_USER, 'MySQL user');
const mysqlPassword = exactString(process.env.ORDER_MERCHANT_FAILURE_MYSQL_PASSWORD, 'MySQL password');
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const receiptPath = path.resolve(process.env.ORDER_MERCHANT_FAILURE_RECEIPT_PATH
  || `/private/tmp/order-merchant-failure-${candidateSHA}/receipt.json`);
const runID = `merchant-failure-${Date.now()}-${randomUUID().slice(0, 8)}`;
const cases = ['PAGE-M02', 'PAGE-M03', 'PAGE-M05'];
const requestLog = [];
const fixture = { mini_token: '', expires_at: '', lanes: {}, ready: {}, product: {} };
let proxy;
let karmaExit = 1;
let failure = '';
let mysqlEvidence = {};
let outboxHidden = false;

if (process.env.ORDER_MERCHANT_FAILURE_FRESH_DB !== 'YES') throw new Error('ORDER_MERCHANT_FAILURE_FRESH_DB=YES is required');
if (!browserVersion) throw new Error('locked Chromium is missing');
if (!/^order_merchant_failure_[a-z0-9_]+$/.test(mysqlDatabase)) throw new Error('dedicated order_merchant_failure_* schema required');

try {
  await prepareFacts();
  proxy = await startProxy(apiOrigin);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_MERCHANT_FAILURE_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_MERCHANT_FAILURE_FIXTURE = JSON.stringify(redactFixture(fixture));
  process.stdout.write(`MERCHANT_FAILURE_UI1_ENV ${JSON.stringify({
    candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin,
    proxy: `${proxy.origin} (random loopback)`, schema: 'order_merchant_failure_*', cases,
  })}\n`);
  const config = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-merchant-failure.conf.cjs'), { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  karmaExit = await new Promise((resolve, reject) => {
    const server = new karma.Server(config, resolve);
    server.start().catch(reject);
  });
  if (karmaExit !== 0) throw new Error(`merchant failure Chrome Gate exited ${karmaExit}`);
  if (outboxHidden) restoreOutbox();

  const tomorrowMenu = await request('GET', `/api/v1/menu?date=${fixture.tomorrow}&time=${encodeURIComponent(fixture.near_time)}`, {
    bearer: fixture.mini_token,
  });
  const tomorrowProduct = findMenuProduct(tomorrowMenu, fixture.product.id);
  if (tomorrowProduct.sold_out !== false) throw new Error('today sold-out leaked into tomorrow root menu');
  mysqlEvidence = verifyMySQL();
  verifyRenderedRequests();
} catch (error) {
  failure = safeMessage(error);
} finally {
  if (outboxHidden) {
    try { restoreOutbox(); } catch (error) { failure ||= `outbox restore failed: ${safeMessage(error)}`; }
  }
  if (proxy) await proxy.close().catch(error => { failure ||= `proxy cleanup failed: ${safeMessage(error)}`; });
}

const passed = karmaExit === 0 && !failure
  && mysqlEvidence.five_lane_states === true
  && mysqlEvidence.store_close_exact === true
  && mysqlEvidence.ready_success_exact === true
  && mysqlEvidence.ready_failures_unchanged === true
  && mysqlEvidence.today_only_soldout === true;
const receipt = {
  schema: 'order.merchant-failure.ui1.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), status: passed ? 'PASS' : 'FAIL',
  evidence_level: 'L3_LOCAL_COMPOSED', browser: browserVersion, database_schema_version: 44,
  database_name_redacted: 'order_merchant_failure_*', cases,
  case_evidence: {
    'PAGE-M02': 'rendered five live lanes and cross-lane search; injected close 503 stayed visibly open with no write, then one root close succeeded; identity reset rendered',
    'PAGE-M03': 'rendered RESERVED zero-request shield, injected READY 503 and real notification-enqueue SQL failure with PREPARING/token/receipt/outbox rollback, then one legal READY with token/receipt/outbox',
    'PAGE-M05': 'rendered today product; injected 503 and mismatched-date fact stayed unchanged/error; one root today sold-out succeeded and tomorrow root menu remained available',
  },
  root_requests: requestLog,
  mysql_evidence: mysqlEvidence,
  controlled_fixture: 'notification enqueue failure temporarily renames only the fresh schema notification_outbox table, exactly as the fulfillment MySQL integration seam; it is restored before evidence and the whole schema is dropped by Writer Gate',
  cleanup_contract: 'verify-writer.sh stops the private API and drops the dedicated fresh schema',
  external_only: [], error: failure || undefined,
};
mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`MERCHANT_FAILURE_UI1_RESULT ${JSON.stringify({
  status: receipt.status, candidate_sha: candidateSHA, cases: cases.length, browser: browserVersion,
  requests: requestLog.length, mysql_evidence: mysqlEvidence, receipt: receiptPath, error: failure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

async function prepareFacts() {
  const session = await request('POST', '/api/v1/auth/miniprogram/session', {
    expected: 201, body: { code: `${runID}-session` },
  });
  fixture.mini_token = exactToken(session.access_token, 'Mini session');
  fixture.expires_at = exactString(session.expires_at, 'Mini expiry');
  await request('POST', '/api/v1/me/bind-phone', {
    bearer: fixture.mini_token, body: { code: `${runID}-phone` },
  });
  await request('POST', '/api/v1/me/merchant-login', {
    bearer: fixture.mini_token, body: { code: `${runID}-merchant-phone` },
  });
  const pcToken = await acquirePCSession(fixture.mini_token);
  const baseline = await request('GET', '/api/v1/admin/settings', { bearer: pcToken });
  const schedule = closureSchedule(baseline);
  await request('PUT', '/api/v1/admin/settings', {
    bearer: pcToken, key: key('settings'), body: schedule.settings,
  });
  fixture.today = schedule.today;
  fixture.tomorrow = schedule.tomorrow;
  fixture.near_time = schedule.nearTime;
  fixture.far_time = schedule.farTime;

  const category = exactObject((await request('POST', '/api/v1/admin/categories', {
    bearer: pcToken, key: key('category'), expected: 201, body: { name: `商户失败闭环-${runID}` },
  })).category, 'category');
  const categoryID = exactID(category.id, 'category id');
  const products = {};
  for (const [meal, price] of [['lunch', 1280], ['dinner', 1680]]) {
    const view = await request('POST', '/api/v1/admin/products', {
      bearer: pcToken, key: key(`product-${meal}`), expected: 201,
      body: {
        name: `${meal === 'lunch' ? '午餐' : '晚餐'}失败闭环-${runID}`, price_cents: price,
        category_id: categoryID, meal_period: meal, description: '商户失败保护本地闭环', images: [],
      },
    });
    products[meal] = exactObject(view.product, `${meal} product`);
  }
  fixture.product = { id: exactID(products.lunch.id, 'lunch product id'), name: products.lunch.name };
  const dinnerProductID = exactID(products.dinner.id, 'dinner product id');

  const reserved = await materializeOrder(schedule.today, schedule.farTime, dinnerProductID, 'reserved');
  const readyLane = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'ready-lane');
  const readySuccess = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'ready-success');
  const ready503 = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'ready-503');
  const readyEnqueue = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'ready-enqueue');
  const completed = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'completed');
  const refunded = await materializeOrder(schedule.today, schedule.nearTime, fixture.product.id, 'refunded');
  const preparingOrders = [readyLane, readySuccess, ready503, readyEnqueue, completed, refunded];
  for (const order of preparingOrders) await waitOrderState(order.id, 'PREPARING');
  if ((await userOrder(reserved.id)).state !== 'RESERVED') throw new Error('illegal-state fixture is not RESERVED');

  await markReady(readyLane.id, 'ready-lane');
  await markReady(completed.id, 'completed');
  await markReady(refunded.id, 'refunded');
  await redeemWithUserToken(completed.id, 'completed');
  await redeemWithUserToken(refunded.id, 'refunded');
  await request('POST', `/api/v1/admin/orders/${refunded.id}/refund`, {
    bearer: pcToken, key: key('refund'), body: { reason: '商户失败闭环退款' },
  });
  await waitOrderState(refunded.id, 'REFUNDED', 15000);
  await recordReadyConsent(readySuccess.id, 'success');
  await recordReadyConsent(readyEnqueue.id, 'enqueue-fail');

  const completedDetail = await userOrder(completed.id);
  fixture.lanes = {
    已预约: reserved.id,
    制作中: ready503.id,
    待取餐: readyLane.id,
    已完成: completed.id,
    已退款: refunded.id,
  };
  fixture.search_order_id = completed.id;
  fixture.search_order_no = completed.orderNo;
  fixture.search_pickup_number = exactCode(completedDetail.pickup_number);
  fixture.ready = {
    illegal_order_id: reserved.id,
    http_503_order_id: ready503.id,
    enqueue_fail_order_id: readyEnqueue.id,
    success_order_id: readySuccess.id,
    ready_lane_order_id: readyLane.id,
    completed_order_id: completed.id,
    refunded_order_id: refunded.id,
  };
}

async function acquirePCSession(miniToken) {
  const login = await request('POST', '/api/v1/admin/auth/qrcode', { expected: 201, body: {} });
  const qr = new URL(exactString(login.qr_payload, 'QR payload'));
  await request('POST', '/api/v1/me/admin-login/approve', {
    bearer: miniToken,
    body: {
      login_id: exactString(login.login_id, 'login id'),
      approval_secret: exactString(qr.searchParams.get('approval_secret'), 'approval secret'),
      code: `${runID}-admin-phone`,
    },
  });
  const poll = await request('POST', '/api/v1/admin/auth/poll', {
    body: { login_id: login.login_id, poll_secret: exactString(login.poll_secret, 'poll secret') },
  });
  if (poll.state !== 'APPROVED') throw new Error(`PC session ended ${poll.state}`);
  return exactToken(exactObject(poll.session, 'PC session').token, 'PC token');
}

async function materializeOrder(date, pickupTime, productID, label) {
  const prefix = `${runID}-${label}`;
  const quote = exactObject((await request('POST', '/api/v1/quotes', {
    bearer: fixture.mini_token, key: `${prefix}-quote`, expected: 201,
    body: {
      contact_name: `闭环${label}`, pickup_date: date, pickup_time: pickupTime, order_note: prefix,
      items: [{ product_id: productID, quantity: 1, flavors: [], note: '' }],
    },
  })).quote, `${label} quote`);
  const prepayment = exactObject((await request('POST', '/api/v1/orders/prepay', {
    bearer: fixture.mini_token, key: `${prefix}-prepay`, expected: 201,
    body: { quote_id: exactID(quote.id, `${label} quote id`) },
  })).prepayment, `${label} prepayment`);
  const confirm = await request('POST', '/api/v1/orders/confirm', {
    bearer: fixture.mini_token, key: `${prefix}-confirm`,
    body: { prepayment_id: exactID(prepayment.id, `${label} prepayment id`) },
  });
  if (confirm.state !== 'ORDER_CREATED') throw new Error(`${label} confirm ended ${confirm.state}`);
  const id = exactID(confirm.order_id, `${label} order id`);
  const detail = await userOrder(id);
  return { id, orderNo: exactString(detail.order_no, `${label} order number`) };
}

async function markReady(id, label) {
  const body = await request('POST', `/api/v1/merchant/orders/${id}/ready`, {
    bearer: fixture.mini_token, key: key(`ready-${label}`), body: {},
  });
  if (exactObject(body.order, `${label} ready order`).state !== 'READY_FOR_PICKUP') {
    throw new Error(`${label} did not become READY`);
  }
}

async function redeemWithUserToken(id, label) {
  const detail = await userOrder(id);
  const body = await request('POST', '/api/v1/verify/scan', {
    bearer: fixture.mini_token, key: key(`redeem-${label}`),
    body: { token: exactString(detail.redemption_token, `${label} redemption token`) },
  });
  if (exactObject(body.order, `${label} completed order`).state !== 'COMPLETED') {
    throw new Error(`${label} did not become COMPLETED`);
  }
}

async function recordReadyConsent(id, label) {
  const body = await request('POST', `/api/v1/orders/${id}/subscriptions`, {
    bearer: fixture.mini_token, key: key(`consent-${label}`),
    body: { kind: 'READY', decision: 'ACCEPTED' },
  });
  if (!body.subscription || body.subscription.available !== true) throw new Error(`${label} READY consent unavailable`);
}

async function userOrder(id) {
  return exactObject((await request('GET', `/api/v1/orders/${id}`, { bearer: fixture.mini_token })).order, `order ${id}`);
}

async function waitOrderState(id, wanted, timeout = 8000) {
  const deadline = Date.now() + timeout;
  let state = '';
  do {
    state = exactString((await userOrder(id)).state, `order ${id} state`);
    if (state === wanted) return;
    await new Promise(resolve => setTimeout(resolve, 200));
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
  if (shanghai(near).date !== today || shanghai(far).date !== today) throw new Error('merchant failure schedule crossed Shanghai day');
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
  const shifted = new Date(value.getTime() + 8 * 60 * 60000);
  const pad = number => String(number).padStart(2, '0');
  return {
    date: `${shifted.getUTCFullYear()}-${pad(shifted.getUTCMonth() + 1)}-${pad(shifted.getUTCDate())}`,
    time: `${pad(shifted.getUTCHours())}:${pad(shifted.getUTCMinutes())}`,
  };
}

async function request(method, pathname, options = {}) {
  const headers = { Accept: 'application/json' };
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  const raw = await response.text();
  let body = {};
  if (raw) { try { body = JSON.parse(raw); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); } }
  const expected = Array.isArray(options.expected) ? options.expected : [options.expected || 200];
  const parsed = new URL(pathname, apiOrigin);
  requestLog.push({ phase: 'setup-or-evidence', method, path: parsed.pathname, query: parsed.search, status: response.status });
  if (!expected.includes(response.status)) {
    throw new Error(`${method} ${pathname} returned ${response.status}/${body.error && body.error.code || 'UNKNOWN'}`);
  }
  return body;
}

async function startProxy(origin) {
  const target = new URL(origin);
  const server = http.createServer((incoming, outgoing) => {
    if (incoming.method === 'OPTIONS') {
      outgoing.writeHead(204, corsHeaders()); outgoing.end(); return;
    }
    const requestURL = new URL(incoming.url, origin);
    const rawMode = incoming.headers['x-merchant-failure-mode'];
    const mode = Array.isArray(rawMode) ? rawMode[0] : rawMode || '';
    if (directFailure(mode, incoming.method, requestURL.pathname, outgoing)) return;

    let hideForRequest = false;
    try {
      if (mode === 'ready-enqueue-fail' && incoming.method === 'POST' && /\/merchant\/orders\/[1-9]\d*\/ready$/.test(requestURL.pathname)) {
        hideOutbox();
        hideForRequest = true;
      }
      const headers = Object.assign({}, incoming.headers, { host: target.host });
      delete headers.connection;
      delete headers['x-merchant-failure-mode'];
      const upstream = http.request({
        hostname: target.hostname, port: target.port, method: incoming.method, path: incoming.url, headers,
      }, response => {
        const chunks = [];
        response.on('data', chunk => chunks.push(chunk));
        response.on('end', () => {
          try {
            if (hideForRequest) restoreOutbox();
            const rawKey = incoming.headers['idempotency-key'];
            const idempotencyKey = Array.isArray(rawKey) ? rawKey[0] : rawKey;
            requestLog.push({
              phase: 'rendered-ui', method: incoming.method, path: requestURL.pathname, query: requestURL.search,
              status: response.statusCode || 0, injected: mode || undefined,
              idempotency_key_hash: typeof idempotencyKey === 'string' && idempotencyKey
                ? createHash('sha256').update(idempotencyKey).digest('hex') : undefined,
            });
            const responseHeaders = Object.assign({}, response.headers, corsHeaders());
            delete responseHeaders.connection;
            outgoing.writeHead(response.statusCode || 502, responseHeaders);
            outgoing.end(Buffer.concat(chunks));
          } catch (error) {
            outgoing.writeHead(502, { ...corsHeaders(), 'content-type': 'application/json' });
            outgoing.end(JSON.stringify({ error: { code: 'FAILURE_FIXTURE_UNAVAILABLE' } }));
          }
        });
      });
      upstream.on('error', error => {
        if (hideForRequest && outboxHidden) { try { restoreOutbox(); } catch {} }
        requestLog.push({ phase: 'rendered-ui', method: incoming.method, path: requestURL.pathname, status: 0, error: error.code || 'UPSTREAM_ERROR' });
        outgoing.writeHead(502, { ...corsHeaders(), 'content-type': 'application/json' });
        outgoing.end(JSON.stringify({ error: { code: 'COMPOSED_UPSTREAM_UNAVAILABLE' } }));
      });
      incoming.pipe(upstream);
    } catch (error) {
      if (hideForRequest && outboxHidden) { try { restoreOutbox(); } catch {} }
      outgoing.writeHead(502, { ...corsHeaders(), 'content-type': 'application/json' });
      outgoing.end(JSON.stringify({ error: { code: 'FAILURE_FIXTURE_UNAVAILABLE' } }));
    }
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

function directFailure(mode, method, pathname, response) {
  const matches503 = (mode === 'store-status-503' && method === 'PUT' && pathname === '/api/v1/merchant/store-status')
    || (mode === 'ready-503' && method === 'POST' && /\/api\/v1\/merchant\/orders\/[1-9]\d*\/ready$/.test(pathname))
    || (mode === 'soldout-503' && method === 'PUT' && /\/api\/v1\/merchant\/products\/[1-9]\d*\/soldout$/.test(pathname));
  if (matches503) {
    requestLog.push({ phase: 'rendered-ui', method, path: pathname, status: 503, injected: mode });
    response.writeHead(503, { ...corsHeaders(), 'content-type': 'application/json' });
    response.end(JSON.stringify({ error: { code: 'TEMPORARILY_UNAVAILABLE' } }));
    return true;
  }
  if (mode === 'soldout-drift' && method === 'PUT' && /\/api\/v1\/merchant\/products\/[1-9]\d*\/soldout$/.test(pathname)) {
    const id = pathname.split('/').at(-2);
    requestLog.push({ phase: 'rendered-ui', method, path: pathname, status: 200, injected: mode });
    response.writeHead(200, { ...corsHeaders(), 'content-type': 'application/json' });
    response.end(JSON.stringify({ product_id: id, service_date: fixture.tomorrow, sold_out: true }));
    return true;
  }
  return false;
}

function hideOutbox() {
  if (outboxHidden) throw new Error('notification_outbox already hidden');
  mysqlExecute('RENAME TABLE notification_outbox TO notification_outbox_merchant_failure');
  outboxHidden = true;
}
function restoreOutbox() {
  if (!outboxHidden) return;
  mysqlExecute('RENAME TABLE notification_outbox_merchant_failure TO notification_outbox');
  outboxHidden = false;
}
function mysqlExecute(sql) {
  execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser,
    `--database=${mysqlDatabase}`, '--execute', sql,
  ], { encoding: 'utf8' });
}

function verifyMySQL() {
  const ids = fixture.ready;
  const sql = `SELECT
    (SELECT business_status FROM storefront_settings WHERE id=1),
    (SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action='merchant.store.operating_status.write'),
    (SELECT state FROM orders WHERE id=${Number(ids.success_order_id)}),
    (SELECT COUNT(*) FROM orders WHERE id=${Number(ids.success_order_id)} AND state='READY_FOR_PICKUP' AND ready_at IS NOT NULL AND redemption_token_ciphertext IS NOT NULL AND redemption_token_hash IS NOT NULL AND redemption_key_version IS NOT NULL AND redemption_issued_at IS NOT NULL),
    (SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action='fulfillment.mark_ready' AND target_id=${Number(ids.success_order_id)}),
    (SELECT COUNT(*) FROM notification_outbox WHERE order_id=${Number(ids.success_order_id)} AND kind='READY'),
    (SELECT COUNT(*) FROM notification_consents WHERE order_id=${Number(ids.success_order_id)} AND kind='READY' AND consumed_at IS NOT NULL),
    (SELECT COUNT(*) FROM orders WHERE id IN (${Number(ids.http_503_order_id)},${Number(ids.enqueue_fail_order_id)}) AND state='PREPARING' AND ready_at IS NULL AND redemption_token_ciphertext IS NULL AND redemption_token_hash IS NULL AND redemption_key_version IS NULL AND redemption_issued_at IS NULL),
    (SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action='fulfillment.mark_ready' AND target_id IN (${Number(ids.http_503_order_id)},${Number(ids.enqueue_fail_order_id)})),
    (SELECT COUNT(*) FROM notification_outbox WHERE order_id IN (${Number(ids.http_503_order_id)},${Number(ids.enqueue_fail_order_id)})),
    (SELECT COUNT(*) FROM notification_consents WHERE order_id=${Number(ids.enqueue_fail_order_id)} AND kind='READY' AND decision='ACCEPTED' AND consumed_at IS NULL),
    (SELECT COUNT(*) FROM product_sold_out_dates WHERE product_id=${Number(fixture.product.id)} AND service_date='${fixture.today}'),
    (SELECT COUNT(*) FROM product_sold_out_dates WHERE product_id=${Number(fixture.product.id)} AND service_date='${fixture.tomorrow}'),
    (SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action='merchant.product.sold_out.write' AND target_id=${Number(fixture.product.id)}),
    (SELECT IF(
      (SELECT state FROM orders WHERE id=${Number(fixture.lanes.已预约)})='RESERVED' AND
      (SELECT state FROM orders WHERE id=${Number(fixture.lanes.制作中)})='PREPARING' AND
      (SELECT state FROM orders WHERE id=${Number(fixture.lanes.待取餐)})='READY_FOR_PICKUP' AND
      (SELECT state FROM orders WHERE id=${Number(fixture.lanes.已完成)})='COMPLETED' AND
      (SELECT state FROM orders WHERE id=${Number(fixture.lanes.已退款)})='REFUNDED',1,0))
  `;
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser,
    `--database=${mysqlDatabase}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim().split('\t');
  if (output.length !== 15) throw new Error(`MySQL evidence shape ${JSON.stringify(output)}`);
  return {
    store_close_exact: output[0] === 'closed' && output[1] === '1',
    ready_success_exact: output[2] === 'READY_FOR_PICKUP' && output.slice(3, 7).every(value => value === '1'),
    ready_failures_unchanged: output[7] === '2' && output[8] === '0' && output[9] === '0' && output[10] === '1',
    today_only_soldout: output[11] === '1' && output[12] === '0' && output[13] === '1',
    five_lane_states: output[14] === '1',
    raw_counts: output,
  };
}

function verifyRenderedRequests() {
  for (const [mode, status] of [
    ['store-status-503', 503], ['ready-503', 503], ['ready-enqueue-fail', 503],
    ['soldout-503', 503], ['soldout-drift', 200],
  ]) {
    if (!requestLog.some(item => item.phase === 'rendered-ui' && item.injected === mode && item.status === status)) {
      throw new Error(`rendered request omitted ${mode}/${status}`);
    }
  }
  for (const [method, pathname] of [
    ['PUT', '/api/v1/merchant/store-status'],
    ['POST', `/api/v1/merchant/orders/${fixture.ready.success_order_id}/ready`],
    ['PUT', `/api/v1/merchant/products/${fixture.product.id}/soldout`],
  ]) {
    if (!requestLog.some(item => item.phase === 'rendered-ui' && item.method === method && item.path === pathname
      && !item.injected && item.status >= 200 && item.status < 300)) {
      throw new Error(`rendered request omitted successful ${method} ${pathname}`);
    }
  }
}

function findMenuProduct(body, id) {
  const products = body && Array.isArray(body.categories)
    ? body.categories.flatMap(category => Array.isArray(category.products) ? category.products : []) : [];
  const product = products.find(item => item.id === id);
  if (!product) throw new Error(`root menu omitted product ${id}`);
  return product;
}
function redactFixture(value) {
  return {
    mini_token: value.mini_token, expires_at: value.expires_at,
    lanes: value.lanes, search_order_id: value.search_order_id,
    search_order_no: value.search_order_no, search_pickup_number: value.search_pickup_number,
    ready: value.ready, product: value.product, today: value.today, tomorrow: value.tomorrow,
    near_time: value.near_time, far_time: value.far_time,
  };
}
function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET,POST,PUT,OPTIONS',
    'access-control-allow-headers': 'authorization,content-type,idempotency-key,x-merchant-failure-mode',
    'access-control-expose-headers': 'x-request-id',
  };
}
function key(scope) { return `${runID}-${scope}-${randomUUID()}`; }
function exactOrigin(value) {
  if (!/^http:\/\/127\.0\.0\.1:\d{1,5}$/.test(value || '')) throw new Error('exact loopback API origin required');
  return value;
}
function exactName(value, label) {
  if (typeof value !== 'string' || !/^[A-Za-z0-9_.-]+$/.test(value)) throw new Error(`${label} invalid`);
  return value;
}
function exactObject(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${label} missing`);
  return value;
}
function exactString(value, label) {
  if (typeof value !== 'string' || !value.trim()) throw new Error(`${label} missing`);
  return value;
}
function exactToken(value, label) {
  const token = exactString(value, label);
  if (token.length < 32) throw new Error(`${label} malformed`);
  return token;
}
function exactID(value, label) {
  const id = String(value || '');
  if (!/^[1-9]\d*$/.test(id)) throw new Error(`${label} invalid`);
  return id;
}
function exactCode(value) {
  const code = exactString(value, 'pickup number');
  if (!/^\d{4}$/.test(code)) throw new Error('pickup number malformed');
  return code;
}
function safeMessage(error) {
  return String(error && error.message || 'unknown failure')
    .replace(/Bearer\s+\S+/gi, 'Bearer [REDACTED]')
    .replace(/MYSQL_PWD=\S+/gi, 'MYSQL_PWD=[REDACTED]')
    .replace(/[\r\n]+/g, ' ').slice(0, 800);
}
