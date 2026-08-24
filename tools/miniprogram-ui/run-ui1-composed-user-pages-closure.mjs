import { execFileSync, spawnSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { randomInt, randomUUID } from 'node:crypto';
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
const mysqlContainer = exactName(process.env.ORDER_COMPOSED_MYSQL_CONTAINER, 'MySQL container');
const mysqlDatabase = exactName(process.env.ORDER_COMPOSED_MYSQL_DATABASE, 'MySQL database');
const mysqlUser = exactName(process.env.ORDER_COMPOSED_MYSQL_USER, 'MySQL user');
const mysqlPassword = exactString(process.env.ORDER_COMPOSED_MYSQL_PASSWORD, 'MySQL password');
const evidenceRoot = path.resolve(process.env.ORDER_USER_PAGES_EVIDENCE_DIR || '/private/tmp/order-mini-user-pages-closure');
const receiptPath = path.resolve(process.env.ORDER_USER_PAGES_RECEIPT_PATH || path.join(evidenceRoot, 'receipt.json'));
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = existsSync(browserPath) ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const caseIDs = ['PAGE-U01', 'PAGE-U02', 'PAGE-U03', 'PAGE-U04', 'PAGE-U05', 'PAGE-U06', 'PAGE-U07', 'PAGE-U08', 'PAGE-U09'];
const runID = `user-pages-${Date.now()}-${randomUUID().slice(0, 8)}`;
const requests = [];
const childResults = [];
let proxy;
let failure = '';
let setup;
let mysqlEvidence = {};
let karmaExit = 1;

if (process.env.ORDER_USER_PAGES_FRESH_DB !== 'YES') {
  throw new Error('ORDER_USER_PAGES_FRESH_DB=YES is required; this Gate must never mutate a shared acceptance database');
}
if (!/^order_user_pages_[a-z0-9_]+$/.test(mysqlDatabase)) {
  throw new Error('ORDER_COMPOSED_MYSQL_DATABASE must be a dedicated order_user_pages_* schema');
}
if (!browserVersion) throw new Error('locked Chromium is missing; reuse MINIPROGRAM_UI_DEPS and PLAYWRIGHT_BROWSERS_PATH=0');

mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });
process.stdout.write(`MINI_USER_PAGES_UI1_ENV ${JSON.stringify({
  candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin,
  database: mysqlDatabase, cases: caseIDs,
})}\n`);

try {
  setup = await prepareFacts();
  runChild('customer', 'run-ui1-composed.mjs', {
    ORDER_COMPOSED_FLOW: 'customer', ORDER_COMPOSED_PAYMENT_EXPECTATION: 'success',
  });
  runChild('consent', 'run-ui1-composed.mjs', {
    ORDER_COMPOSED_FLOW: 'consent', ORDER_COMPOSED_PAYMENT_EXPECTATION: 'success',
  });
  runChild('boundaries', 'run-ui1-composed-boundaries.mjs', {
    ORDER_BOUNDARIES_RECEIPT_PATH: path.join(evidenceRoot, 'boundaries-receipt.json'),
  });

  const paginationOrders = await seedPaginationOrders(setup.miniToken, setup.product.id, setup.pickup);
  setup.cancelOrderID = paginationOrders.at(-1);
  proxy = await startTransparentProxy(apiOrigin, requests);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_USER_PAGES_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_USER_PAGES_SETUP = JSON.stringify({
    api_origin: proxy.origin,
    notice: setup.notice,
    product: {
      id: setup.product.id, name: setup.product.name, specification: setup.product.specification,
      images: setup.product.images,
    },
    staff: setup.staff,
    cancel_order_id: setup.cancelOrderID,
  });
  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-user-pages-closure.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  karmaExit = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
  if (karmaExit !== 0) throw new Error(`PAGE-U01..U09 focused Chrome Gate exited ${karmaExit}`);
  mysqlEvidence = verifyMySQL();
} catch (error) {
  failure = safeMessage(error);
} finally {
  if (proxy) await proxy.close().catch(error => { failure ||= `proxy cleanup failed: ${safeMessage(error)}`; });
}

const passed = !failure && karmaExit === 0 && childResults.length === 3 && childResults.every(result => result.status === 'PASS')
  && mysqlEvidence.quote_snapshot === true && mysqlEvidence.pagination_orders >= 21
  && mysqlEvidence.identity_shared_facts === true;
const receipt = {
  schema: 'order.mini-user-pages.ui1.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED', database_schema_version: 44,
  database_name_redacted: 'order_user_pages_*', status: passed ? 'PASS' : 'FAIL',
  cases: caseIDs,
  case_evidence: {
    'PAGE-U01': 'rendered anonymous/rejected/local-fake merchant entry + root HTTP; real WeChat phone remains L4',
    'PAGE-U02': 'rendered storefront, notice, active-order strip and three entries from root HTTP',
    'PAGE-U03': 'rendered pickup facts, search and flavored cart against server menu',
    'PAGE-U04': 'rendered three ordered server images, specification, manual position and full preview',
    'PAGE-U05': 'rendered contact/time/flavors/item note/order note through Quote, Prepay, Confirm and MySQL snapshots',
    'PAGE-U06': 'existing customer+consent rendered result and READY decision against root HTTP/MySQL; real payment/subscription remain L4',
    'PAGE-U07': 'existing customer+consent rendered detail, subscription, cancellation and refund result against root HTTP/MySQL',
    'PAGE-U08': 'rendered exact filters, 20+cursor page, appended page, per-state root queries and REFUNDING setup response',
    'PAGE-U09': 'rendered masked identity, cosmetic zero-HTTP, native contact, exact extra-phone match, merchant login/switch',
  },
  child_results: childResults,
  focused_requests: requests,
  mysql_evidence: mysqlEvidence,
  fixture_exceptions: [{
    marker: 'BOOTSTRAP_CONTENT_FIXTURE_NOT_EVIDENCE',
    scope: 'one HTTP-created product specification only',
    reason: 'canonical PRD section 10 keeps specification outside the phase-one PC write contract; this fixture proves only PAGE-U04 read-only rendering',
  }],
  cleanup_contract: 'private API and dedicated fresh schema are terminated/dropped by verify-writer.sh',
  external_only: [
    'WeChat Developer Tools UI2', 'real getPhoneNumber/profile/contact UI',
    'real wx.requestPayment/funds', 'real subscription templates and consent UI', 'physical-device UI3',
  ],
  error: failure || undefined,
};
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`MINI_USER_PAGES_UI1_RESULT ${JSON.stringify({
  status: receipt.status, candidate_sha: candidateSHA, cases: caseIDs.length,
  child_flows: childResults.length, requests: requests.length,
  mysql_evidence: mysqlEvidence, receipt: receiptPath, error: failure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

async function prepareFacts() {
  const sessions = await acquirePCSession();
  const baseline = await requestJSON('GET', '/api/v1/admin/settings', { bearer: sessions.pcToken });
  if (typeof baseline.pickup_point !== 'string' || !baseline.pickup_point.trim()) {
    throw new Error('fresh bootstrap did not expose a pickup point');
  }
  const serviceDates = [shanghaiDate(0), shanghaiDate(1)].map(date => ({ date, status: 'open' }));
  const notice = `PAGE-U02 服务端公告 ${runID}`;
  const configured = {
    store_status: 'open', pickup_point: baseline.pickup_point, notice, pickup_step_min: 30,
    meal_periods: [
      { code: 'lunch', name: '午餐', cutoff_time: '23:00', pickup_from: '23:30', pickup_to: '23:30' },
      { code: 'dinner', name: '晚餐', cutoff_time: '23:10', pickup_from: '23:40', pickup_to: '23:40' },
    ],
    service_dates: serviceDates,
  };
  const saved = await requestJSON('PUT', '/api/v1/admin/settings', {
    bearer: sessions.pcToken, key: key('settings'), body: configured,
  });
  if (!Array.isArray(saved.service_dates) || saved.service_dates.length !== 2
    || saved.service_dates.some((item, index) => item.date !== serviceDates[index].date || item.status !== 'open')) {
    throw new Error('Admin settings did not persist exact today/tomorrow service dates');
  }
  const images = [];
  for (const color of [[220, 45, 45, 255], [45, 180, 75, 230], [45, 90, 220, 200]]) {
    images.push(await uploadImage(sessions.pcToken, solidPNG(...color)));
  }
  const firstCategory = payloadOf(await requestJSON('POST', '/api/v1/admin/categories', {
    bearer: sessions.pcToken, key: key('category-a'), body: { name: `本地套餐-${runID}` }, expected: 201,
  }), 'category');
  const secondCategory = payloadOf(await requestJSON('POST', '/api/v1/admin/categories', {
    bearer: sessions.pcToken, key: key('category-b'), body: { name: `本地饮品-${runID}` }, expected: 201,
  }), 'category');
  const firstAdmin = await createProduct(sessions.pcToken, firstCategory.id, '本地午餐套餐', images, '全天商品与三图详情');
  await createProduct(sessions.pcToken, secondCategory.id, '无糖饮品', [], '无糖饮品');
  await createProduct(sessions.pcToken, firstCategory.id, '本地晚餐套餐', [], '晚餐套餐');
  const staffPhone = `188${String(randomInt(0, 100000000)).padStart(8, '0')}`;
  const staffName = 'PAGE-U05 联系人';
  await requestJSON('POST', '/api/v1/admin/staff-whitelist', {
    bearer: sessions.pcToken, key: key('staff'), expected: 201, body: { phone: staffPhone, name: staffName },
  });
  const discount = await requestJSON('PUT', '/api/v1/admin/discount-rate', {
    bearer: sessions.pcToken, key: key('discount-rate'), body: { rate_percent: 88 },
  });
  if (discount.rate_percent !== 88) {
    throw new Error(`Admin discount rate was ${JSON.stringify(discount.rate_percent)}`);
  }
  const current = saved.service_dates[0];
  applySpecificationFixture(firstAdmin.id);
  const first = payloadOf(await requestJSON(
    'GET',
    `/api/v1/catalog/products/${firstAdmin.id}?date=${encodeURIComponent(current.date)}&time=23%3A30`,
    { bearer: sessions.miniToken },
  ), 'product');
  if (first.specification !== '每份 300 克') {
    throw new Error(`PAGE-U04 public specification was ${JSON.stringify(first.specification)}`);
  }
  if (first.images.length !== 3) {
    throw new Error(`PAGE-U04 public image count was ${first.images.length}`);
  }
  return {
    pcToken: sessions.pcToken, miniToken: sessions.miniToken,
    notice, product: first, staff: { phone: staffPhone, name: staffName },
    pickup: { date: current.date, time: '23:30' },
  };
}

function applySpecificationFixture(productID) {
  if (typeof productID !== 'string' || !/^[1-9]\d*$/.test(productID)) {
    throw new Error('specification fixture target is invalid');
  }
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--default-character-set=utf8mb4', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser,
    `--database=${mysqlDatabase}`, '--execute',
    `UPDATE products SET specification='每份 300 克' WHERE id=${productID}; SELECT ROW_COUNT()`,
  ], { encoding: 'utf8' }).trim();
  if (output !== '1') throw new Error(`specification fixture affected ${JSON.stringify(output)} rows`);
}

async function createProduct(pcToken, categoryID, name, images, description) {
  return payloadOf(await requestJSON('POST', '/api/v1/admin/products', {
    bearer: pcToken, key: key('product'), expected: 201,
    body: {
      name, price_cents: 1888, category_id: String(categoryID), meal_period: 'all', description,
      images: images.map((image, sort) => ({ object_key: image.object_key, sort_order: sort })),
    },
  }), 'product');
}

async function seedPaginationOrders(token, productID, pickup) {
  const ids = [];
  for (let index = 1; index <= 21; index += 1) {
    const quote = payloadOf(await requestJSON('POST', '/api/v1/quotes', {
      bearer: token, key: key(`page-quote-${index}`), expected: 201,
      body: {
        contact_name: 'PAGE-U08 分页用户', pickup_date: pickup.date, pickup_time: pickup.time,
        order_note: `PAGE-U08 pagination ${index}`,
        items: [{ product_id: String(productID), quantity: 1, flavors: [], note: '' }],
      },
    }), 'quote');
    const prepayment = payloadOf(await requestJSON('POST', '/api/v1/orders/prepay', {
      bearer: token, key: key(`page-prepay-${index}`), expected: 201, body: { quote_id: quote.id },
    }), 'prepayment');
    const confirmed = await requestJSON('POST', '/api/v1/orders/confirm', {
      bearer: token, key: key(`page-confirm-${index}`), body: { prepayment_id: prepayment.id },
    });
    if (confirmed.state !== 'ORDER_CREATED' || !/^[1-9]\d*$/.test(confirmed.order_id || '')) {
      throw new Error(`pagination order ${index} was not created`);
    }
    ids.push(confirmed.order_id);
  }
  return ids;
}

function runChild(name, script, extraEnv) {
  const result = spawnSync(process.execPath, [path.join(toolRoot, script)], {
    cwd: repositoryRoot,
    env: {
      ...process.env, ...extraEnv,
      ORDER_COMPOSED_API_ORIGIN: apiOrigin,
      ORDER_COMPOSED_MYSQL_CONTAINER: mysqlContainer,
      ORDER_COMPOSED_MYSQL_DATABASE: mysqlDatabase,
      ORDER_COMPOSED_MYSQL_USER: mysqlUser,
      ORDER_COMPOSED_MYSQL_PASSWORD: mysqlPassword,
      MINIPROGRAM_UI_DEPS: dependencyRoot,
      PLAYWRIGHT_BROWSERS_PATH: '0',
    },
    encoding: 'utf8', maxBuffer: 20 * 1024 * 1024,
  });
  if (result.stdout) process.stdout.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);
  const marker = name === 'boundaries' ? 'MINI_BOUNDARIES_UI1_RESULT ' : 'UI1_COMPOSED_RESULT ';
  const line = (result.stdout || '').split('\n').filter(value => value.startsWith(marker)).at(-1);
  let parsed = {};
  try { parsed = line ? JSON.parse(line.slice(marker.length)) : {}; } catch (error) { parsed = {}; }
  const status = result.status === 0 && parsed.status === 'PASS' ? 'PASS' : 'FAIL';
  childResults.push({ name, status, scenarios: parsed.scenarios || parsed.cases || 0 });
  if (status !== 'PASS') throw new Error(`${name} child Gate failed with exit ${result.status}`);
}

function verifyMySQL() {
  const sql = [
    "SELECT IF(COUNT(*)=1 AND MAX(q.contact_name_snapshot='PAGE-U05 联系人')=1 AND MAX(CONVERT(q.contact_phone_snapshot USING ascii)='+8613800000000')=1 AND MAX(q.identity_kind='STAFF')=1 AND MAX(q.discount_cents>0)=1 AND MAX(q.order_note='PAGE-U05 整单备注')=1 AND MAX(qi.quantity=2)=1 AND MAX(JSON_LENGTH(qi.flavors_json)=2)=1 AND MAX(qi.line_note='菜品备注保留')=1 AND MAX(qi.image_object_key_snapshot IS NOT NULL)=1,1,0) FROM quotes q JOIN quote_items qi ON qi.quote_id=q.id WHERE q.order_note='PAGE-U05 整单备注'",
    "SELECT COUNT(*) FROM orders WHERE order_note LIKE 'PAGE-U08 pagination %'",
    "SELECT IF((SELECT COUNT(*) FROM merchant_accounts WHERE bound_user_id IS NOT NULL AND enabled=1)>=1 AND (SELECT COUNT(*) FROM miniprogram_users WHERE extra_phone IS NOT NULL AND extra_name IS NOT NULL)>=1,1,0)",
  ].join(';');
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--default-character-set=utf8mb4', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser,
    `--database=${mysqlDatabase}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim().split('\n');
  if (output.length !== 3) throw new Error(`MySQL evidence shape was ${JSON.stringify(output)}`);
  return {
    quote_snapshot: output[0] === '1',
    pagination_orders: Number(output[1]),
    identity_shared_facts: output[2] === '1',
  };
}

async function acquirePCSession() {
  const session = await requestJSON('POST', '/api/v1/auth/miniprogram/session', {
    expected: 201, body: { code: 'ui1-composed-login-code' },
  });
  const miniToken = exactToken(session.access_token, 'Mini session');
  await requestJSON('POST', '/api/v1/me/bind-phone', {
    bearer: miniToken, key: key('bind-owner'), body: { code: 'ui1-user-pages-owner-phone' },
  });
  const login = await requestJSON('POST', '/api/v1/admin/auth/qrcode', { expected: 201, body: {} });
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  await requestJSON('POST', '/api/v1/me/admin-login/approve', {
    bearer: miniToken,
    body: {
      login_id: exactString(login.login_id, 'login_id'),
      approval_secret: exactString(payload.searchParams.get('approval_secret'), 'approval_secret'),
      code: 'ui1-user-pages-owner-approve',
    },
  });
  const poll = await requestJSON('POST', '/api/v1/admin/auth/poll', {
    body: { login_id: login.login_id, poll_secret: exactString(login.poll_secret, 'poll_secret') },
  });
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC session was not approved');
  return { miniToken, pcToken: exactToken(poll.session.token, 'PC session') };
}

async function uploadImage(pcToken, bytes) {
  const form = new FormData();
  form.append('file', new Blob([bytes], { type: 'image/png' }), `${randomUUID()}.png`);
  const response = await fetch(`${apiOrigin}/api/v1/upload`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${pcToken}`, 'Idempotency-Key': key('upload') },
    body: form,
  });
  const body = await response.json();
  if (response.status !== 201 || !body.image || !body.image.object_key || !body.image.url) {
    throw new Error(`image upload returned ${response.status}`);
  }
  return body.image;
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
  if (raw) {
    try { body = JSON.parse(raw); } catch (error) { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  }
  const expected = options.expected === undefined ? [200] : Array.isArray(options.expected) ? options.expected : [options.expected];
  if (!expected.includes(response.status)) {
    throw new Error(`${method} ${pathname} returned ${response.status}/${body && body.error && body.error.code || 'UNKNOWN'}`);
  }
  return body;
}

async function startTransparentProxy(origin, log) {
  const target = new URL(origin);
  const server = http.createServer((request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, {
        'access-control-allow-origin': '*',
        'access-control-allow-methods': 'GET, POST, PUT, DELETE, OPTIONS',
        'access-control-allow-headers': 'authorization, content-type, idempotency-key',
      });
      response.end();
      return;
    }
    const headers = { ...request.headers, host: target.host };
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname, port: target.port, path: request.url,
      method: request.method, headers,
    }, incoming => {
      const chunks = [];
      incoming.on('data', chunk => chunks.push(chunk));
      incoming.on('end', () => {
        const body = Buffer.concat(chunks);
        log.push({ method: request.method, path: new URL(request.url, origin).pathname, status: incoming.statusCode });
        response.writeHead(incoming.statusCode, {
          ...incoming.headers,
          'access-control-allow-origin': '*',
          'access-control-allow-methods': 'GET, POST, PUT, DELETE, OPTIONS',
          'access-control-allow-headers': 'authorization, content-type, idempotency-key',
        });
        response.end(body);
      });
    });
    upstream.on('error', () => { response.writeHead(502); response.end('proxy unavailable'); });
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

function payloadOf(body, keyName) {
  const payload = body && body[keyName];
  if (!payload || typeof payload !== 'object') throw new Error(`${keyName} response was malformed`);
  return payload;
}
function key(scope) { return `user-pages-${scope}-${randomUUID()}`; }
function shanghaiDate(dayOffset) {
  const shifted = new Date(Date.now() + 8 * 60 * 60 * 1000);
  const midnight = Date.UTC(shifted.getUTCFullYear(), shifted.getUTCMonth(), shifted.getUTCDate() + dayOffset);
  return new Date(midnight).toISOString().slice(0, 10);
}
function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || !parsed.port
    || parsed.pathname !== '/' || parsed.search || parsed.hash || parsed.username || parsed.password) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}
function exactName(value, label) {
  if (typeof value !== 'string' || !/^[A-Za-z0-9_.-]+$/.test(value)) throw new Error(`${label} is invalid`);
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
function safeMessage(error) {
  return String(error && error.message || 'unknown error')
    .replace(/Bearer\s+\S+/gi, 'Bearer [REDACTED]')
    .replace(/MYSQL_PWD=\S+/gi, 'MYSQL_PWD=[REDACTED]');
}
