import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { randomUUID } from 'node:crypto';
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
const receiptPath = path.resolve(process.env.ORDER_STAFF_PROFILE_RECEIPT_PATH
  || '/private/tmp/order-mini-staff-profile/receipt.json');
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = existsSync(browserPath)
  ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const runID = `staff-profile-${Date.now()}-${randomUUID().slice(0, 8)}`;
const visitorNote = `${runID}-visitor`;
const staffNote = `${runID}-staff`;
const cases = ['PAGE-U03', 'PAGE-U09', 'AC-04', 'AC-05'];
const requests = [];
let proxy;
let karmaExit = 1;
let failure = '';
let mysqlEvidence = {};

if (process.env.ORDER_STAFF_PROFILE_FRESH_DB !== 'YES') throw new Error('ORDER_STAFF_PROFILE_FRESH_DB=YES is required');
if (!/^order_staff_profile_[a-z0-9_]+$/.test(mysqlDatabase)) {
  throw new Error('ORDER_COMPOSED_MYSQL_DATABASE must be a dedicated order_staff_profile_* schema');
}
if (!browserVersion) throw new Error('locked Chromium is missing; reuse MINIPROGRAM_UI_DEPS and PLAYWRIGHT_BROWSERS_PATH=0');

try {
  const setup = await prepareFacts();
  proxy = await startProxy(apiOrigin, requests);
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_STAFF_PROFILE_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_STAFF_PROFILE_SETUP = JSON.stringify(Object.assign(setup, { api_origin: proxy.origin }));
  process.stdout.write(`MINI_STAFF_PROFILE_UI1_ENV ${JSON.stringify({
    candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin,
    database: 'order_staff_profile_*', proxy: `${proxy.origin} (random loopback)`, cases,
  })}\n`);
  const processed = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-staff-profile.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  karmaExit = await new Promise((resolve, reject) => {
    const server = new karma.Server(processed, resolve);
    server.start().catch(reject);
  });
  if (karmaExit !== 0) throw new Error(`staff/profile Chrome Gate exited ${karmaExit}`);
  mysqlEvidence = verifyMySQL(setup.product);
} catch (error) {
  failure = safeMessage(error);
} finally {
  if (proxy) await proxy.close().catch(error => { failure ||= `proxy cleanup failed: ${safeMessage(error)}`; });
}

const passed = karmaExit === 0 && !failure
  && mysqlEvidence.exact_quote_rows === true && mysqlEvidence.tamper_created_zero_quotes === true
  && mysqlEvidence.product_staff_source === true && mysqlEvidence.quote_receipts === 2
  && mysqlEvidence.cosmetic_columns === 0
  && requests.some(item => item.path === '/api/v1/me/identity' && item.status === 503 && item.injected === 'identity-503')
  && requests.some(item => item.path === '/api/v1/me/identity' && item.status === 200 && item.injected === 'identity-malformed');
const receipt = {
  schema: 'order.mini-staff-profile.ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED',
  database_schema_version: 44,
  database_name_redacted: 'order_staff_profile_*',
  cases,
  status: passed ? 'PASS' : 'FAIL',
  case_evidence: {
    'PAGE-U03': 'same root product rendered through anonymous, identity-unknown and trusted STAFF menu/detail states; server pickup/menu facts remain authoritative',
    'PAGE-U09': 'rendered neutral nickname/avatar plus native contact; profile/contact rejection performs zero business HTTP; identity 503/malformed never projects a false staff identity; existing exact integrated extra-phone/merchant-switch evidence remains supporting only',
    'AC-04': 'anonymous/unbound and unknown identity show original only; trusted bound whitelist identity shows exact original plus staff price from root MySQL facts',
    'AC-05': 'menu, detail, visitor Quote and STAFF Quote use the same product and exact integer cents; price/phone/name client fields are rejected and create zero extra quotes',
  },
  root_requests: requests,
  mysql_evidence: mysqlEvidence,
  cleanup_contract: 'verify-writer.sh stops private API and drops the dedicated fresh schema; no shared database is mutated',
  external_only: ['real WeChat getUserProfile UI', 'real WeChat contact session', 'WeChat Developer Tools UI2', 'physical-device UI3'],
  forbidden_claims: ['mocked wx rejection is not real platform UI', 'Chrome/miniprogram-simulate is not WeChat Developer Tools UI2'],
  error: failure || undefined,
};
mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`MINI_STAFF_PROFILE_UI1_RESULT ${JSON.stringify({
  status: receipt.status, candidate_sha: candidateSHA, cases: cases.length,
  browser: browserVersion, requests: requests.length, mysql_evidence: mysqlEvidence,
  receipt: receiptPath, error: failure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

async function prepareFacts() {
  const session = await requestJSON('POST', '/api/v1/auth/miniprogram/session', {
    expected: 201, body: { code: `${runID}-session` },
  });
  const staffToken = exactToken(session.access_token, 'Mini session');
  await requestJSON('POST', '/api/v1/me/bind-phone', {
    bearer: staffToken, body: { code: `${runID}-phone` },
  });
  const pcToken = await acquirePCSession(staffToken);
  const baseline = await requestJSON('GET', '/api/v1/admin/settings', { bearer: pcToken });
  const tomorrow = shanghaiDate(1);
  await requestJSON('PUT', '/api/v1/admin/settings', {
    bearer: pcToken, key: key('settings'),
    body: {
      store_status: 'open', pickup_point: baseline.pickup_point, notice: baseline.notice, pickup_step_min: 30,
      meal_periods: [
        { code: 'lunch', name: '午餐', cutoff_time: '11:00', pickup_from: '11:30', pickup_to: '11:30' },
        { code: 'dinner', name: '晚餐', cutoff_time: '17:00', pickup_from: '17:30', pickup_to: '17:30' },
      ],
      service_dates: [{ date: shanghaiDate(0), status: 'closed' }, { date: tomorrow, status: 'open' }],
    },
  });
  const category = payloadOf(await requestJSON('POST', '/api/v1/admin/categories', {
    bearer: pcToken, key: key('category'), expected: 201, body: { name: `员工价-${runID}` },
  }), 'category');
  const product = payloadOf(await requestJSON('POST', '/api/v1/admin/products', {
    bearer: pcToken, key: key('product'), expected: 201,
    body: {
      name: `同源工作餐-${runID}`, price_cents: 1255, category_id: exactID(category.id, 'category'),
      meal_period: 'all', description: '员工与访客同一商品', images: [],
    },
  }), 'product');
  const productID = exactID(product.id, 'product');
  const pickup = { date: tomorrow, meal_period: 'lunch', time: '11:30' };
  const anonymousMenu = await requestJSON('GET', `/api/v1/menu?date=${tomorrow}&time=11%3A30`);
  const anonymousMenuProduct = findMenuProduct(anonymousMenu, productID);
  const anonymousDetail = payloadOf(await requestJSON('GET', `/api/v1/catalog/products/${productID}?date=${tomorrow}&time=11%3A30`), 'product');
  assertOriginalOnly(anonymousMenuProduct, 'anonymous menu');
  assertOriginalOnly(anonymousDetail, 'anonymous detail');

  const visitorQuoteBody = quoteBody(productID, visitorNote);
  const visitorQuote = payloadOf(await requestJSON('POST', '/api/v1/quotes', {
    bearer: staffToken, key: key('visitor-quote'), expected: 201, body: visitorQuoteBody,
  }), 'quote');
  const visitorLine = visitorQuote.items && visitorQuote.items[0];
  if (visitorQuote.identity.kind !== 'VISITOR' || !visitorLine || visitorLine.product_id !== productID
    || visitorLine.original_unit_price_cents !== 1255 || visitorLine.discounted_unit_price_cents !== 1255) {
    throw new Error(`pre-whitelist visitor Quote was ${JSON.stringify(visitorQuote)}`);
  }

  const discount = await requestJSON('PUT', '/api/v1/admin/discount-rate', {
    bearer: pcToken, key: key('discount'), body: { rate_percent: 88 },
  });
  if (discount.rate_percent !== 88) throw new Error('discount rate did not persist 88 percent');
  await requestJSON('POST', '/api/v1/admin/staff-whitelist', {
    bearer: pcToken, key: key('staff'), expected: 201,
    body: { phone: '13800000000', name: `员工-${runID}` },
  });
  const identity = payloadOf(await requestJSON('GET', '/api/v1/me/identity', { bearer: staffToken }), 'identity');
  if (!identity.pricing_identity || identity.pricing_identity.kind !== 'STAFF' || identity.pricing_identity.rate_percent !== 88) {
    throw new Error(`staff identity was ${JSON.stringify(identity.pricing_identity)}`);
  }
  const staffMenu = await requestJSON('GET', `/api/v1/menu?date=${tomorrow}&time=11%3A30`, { bearer: staffToken });
  const staffMenuProduct = findMenuProduct(staffMenu, productID);
  const staffDetail = payloadOf(await requestJSON('GET', `/api/v1/catalog/products/${productID}?date=${tomorrow}&time=11%3A30`, {
    bearer: staffToken,
  }), 'product');
  if (staffMenuProduct.staff_unit_price_cents !== 1104 || staffDetail.staff_unit_price_cents !== 1104) {
    throw new Error(`root staff price drifted ${JSON.stringify([staffMenuProduct.staff_unit_price_cents, staffDetail.staff_unit_price_cents])}`);
  }
  const staffQuoteBody = quoteBody(productID, staffNote);
  return {
    staff_token: staffToken,
    pickup,
    flavors: baseline.flavor_options || [],
    product: {
      id: productID, name: product.name,
      original_unit_price_cents: 1255, staff_unit_price_cents: 1104,
    },
    visitor_quote: {
      id: visitorQuote.id, product_id: visitorLine.product_id, identity_kind: visitorQuote.identity.kind,
      original_unit_price_cents: visitorLine.original_unit_price_cents,
      discounted_unit_price_cents: visitorLine.discounted_unit_price_cents,
    },
    staff_quote_body: staffQuoteBody,
    staff_quote_key: key('staff-quote'),
    tamper_bodies: [
      { kind: 'price', key: key('tamper-price'), body: tamperedQuote(staffQuoteBody, 'price') },
      { kind: 'phone', key: key('tamper-phone'), body: tamperedQuote(staffQuoteBody, 'phone') },
      { kind: 'name', key: key('tamper-name'), body: tamperedQuote(staffQuoteBody, 'name') },
    ],
  };
}

async function acquirePCSession(miniToken) {
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
  return exactToken(poll.session.token, 'PC session');
}

function verifyMySQL(product) {
  const sql = [
    `SELECT q.order_note,q.identity_kind,q.discount_rate_percent,q.original_subtotal_cents,q.discount_cents,q.payable_cents,qi.original_unit_price_cents,qi.discounted_unit_price_cents,qi.quantity,CONVERT(q.contact_phone_snapshot USING ascii) FROM quotes q JOIN quote_items qi ON qi.quote_id=q.id WHERE q.order_note IN ('${visitorNote}','${staffNote}') ORDER BY q.order_note`,
    `SELECT COUNT(*) FROM quotes`,
    `SELECT IF(p.price_cents=1255 AND d.rate_percent=88 AND s.enabled=1,1,0) FROM products p JOIN discount_settings d ON d.id=1 JOIN staff_whitelist s ON CONVERT(s.phone USING ascii)='+8613800000000' WHERE p.id=${product.id}`,
    `SELECT COUNT(*) FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND action='quote.create' AND reason_code='QUOTE_CREATED'`,
    `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='${mysqlDatabase}' AND table_name='miniprogram_users' AND column_name IN ('nickname','nick_name','avatar','avatar_url')`,
  ].join(';');
  const output = execFileSync('/opt/homebrew/bin/docker', [
    'exec', '-e', `MYSQL_PWD=${mysqlPassword}`, mysqlContainer,
    'mysql', '--default-character-set=utf8mb4', '--batch', '--raw', '--skip-column-names', '-u', mysqlUser,
    `--database=${mysqlDatabase}`, '--execute', sql,
  ], { encoding: 'utf8' }).trim().split('\n');
  if (output.length !== 6) throw new Error(`MySQL evidence shape was ${JSON.stringify(output)}`);
  const quoteRows = output.slice(0, 2).map(line => line.split('\t'));
  const visitor = quoteRows.find(row => row[0] === visitorNote);
  const staff = quoteRows.find(row => row[0] === staffNote);
  const exactRows = !!visitor && !!staff
    && visitor[1] === 'VISITOR' && visitor[2] === '100' && visitor[3] === '2510'
    && visitor[4] === '0' && visitor[5] === '2510' && visitor[6] === '1255'
    && visitor[7] === '1255' && visitor[8] === '2' && visitor[9] === '+8613800000000'
    && staff[1] === 'STAFF' && staff[2] === '88' && staff[3] === '2510'
    && staff[4] === '302' && staff[5] === '2208' && staff[6] === '1255'
    && staff[7] === '1104' && staff[8] === '2' && staff[9] === '+8613800000000';
  return {
    exact_quote_rows: exactRows,
    tamper_created_zero_quotes: output[2] === '2',
    product_staff_source: output[3] === '1',
    quote_receipts: Number(output[4]),
    cosmetic_columns: Number(output[5]),
    quote_row_count: quoteRows.length,
  };
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
  requests.push({ phase: 'setup', method, path: new URL(pathname, apiOrigin).pathname, status: response.status });
  if (!expected.includes(response.status)) {
    throw new Error(`${method} ${pathname} returned ${response.status}/${body && body.error && body.error.code || 'UNKNOWN'}`);
  }
  return body;
}

async function startProxy(origin, observations) {
  const target = new URL(origin);
  const server = http.createServer((request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, corsHeaders());
      response.end();
      return;
    }
    const requestURL = new URL(request.url, origin);
    const identityMode = request.headers['x-staff-profile-identity-mode'];
    if (requestURL.pathname === '/api/v1/me/identity' && identityMode === '503') {
      observations.push({ phase: 'browser', method: request.method, path: requestURL.pathname, status: 503, injected: 'identity-503' });
      response.writeHead(503, { ...corsHeaders(), 'content-type': 'application/json' });
      response.end(JSON.stringify({ error: { code: 'IDENTITY_UNAVAILABLE' } }));
      return;
    }
    if (requestURL.pathname === '/api/v1/me/identity' && identityMode === 'malformed') {
      observations.push({ phase: 'browser', method: request.method, path: requestURL.pathname, status: 200, injected: 'identity-malformed' });
      response.writeHead(200, { ...corsHeaders(), 'content-type': 'application/json' });
      response.end(JSON.stringify({ identity: { malformed: true } }));
      return;
    }
    const headers = { ...request.headers, host: target.host };
    delete headers.connection;
    delete headers['x-staff-profile-identity-mode'];
    const upstream = http.request({
      hostname: target.hostname, port: target.port, method: request.method, path: request.url, headers,
    }, incoming => {
      observations.push({
        phase: 'browser', method: request.method, path: requestURL.pathname, query: requestURL.search,
        status: incoming.statusCode, request_id: incoming.headers['x-request-id'] || '',
      });
      const responseHeaders = { ...incoming.headers, ...corsHeaders() };
      delete responseHeaders.connection;
      response.writeHead(incoming.statusCode || 502, responseHeaders);
      incoming.pipe(response);
    });
    upstream.on('error', error => {
      observations.push({ phase: 'browser', method: request.method, path: requestURL.pathname, status: 0, error: error.code || 'UPSTREAM_ERROR' });
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

function quoteBody(productID, note) {
  return {
    contact_name: '本地验收用户', pickup_date: shanghaiDate(1), pickup_time: '11:30', order_note: note,
    items: [{ product_id: productID, quantity: 2, flavors: [], note: '' }],
  };
}
function tamperedQuote(base, kind) {
  const value = JSON.parse(JSON.stringify(base));
  if (kind === 'price') value.items[0].price_cents = 1;
  if (kind === 'phone') value.phone = '+8613999999999';
  if (kind === 'name') value.name = '伪造员工身份';
  return value;
}
function findMenuProduct(body, id) {
  const products = body && Array.isArray(body.categories)
    ? body.categories.flatMap(category => Array.isArray(category.products) ? category.products : []) : [];
  const product = products.find(item => item.id === id);
  if (!product) throw new Error(`menu omitted product ${id}`);
  return product;
}
function assertOriginalOnly(product, label) {
  if (product.original_unit_price_cents !== 1255 || Object.hasOwn(product, 'staff_unit_price_cents')) {
    throw new Error(`${label} did not remain original-only: ${JSON.stringify(product)}`);
  }
}
function payloadOf(body, name) {
  const value = body && body[name];
  if (!value || typeof value !== 'object') throw new Error(`${name} response was malformed`);
  return value;
}
function shanghaiDate(dayOffset) {
  const shifted = new Date(Date.now() + 8 * 60 * 60 * 1000);
  const midnight = Date.UTC(shifted.getUTCFullYear(), shifted.getUTCMonth(), shifted.getUTCDate() + dayOffset);
  return new Date(midnight).toISOString().slice(0, 10);
}
function key(scope) { return `staff-profile-${scope}-${randomUUID()}`; }
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
function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) throw new Error(`${label} id is malformed`);
  return value;
}
function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key, x-staff-profile-identity-mode',
    'access-control-expose-headers': 'x-request-id',
  };
}
function safeMessage(error) {
  return String(error && error.message || 'unknown error')
    .replace(/Bearer\s+\S+/gi, 'Bearer [REDACTED]')
    .replace(/MYSQL_PWD=\S+/gi, 'MYSQL_PWD=[REDACTED]');
}
