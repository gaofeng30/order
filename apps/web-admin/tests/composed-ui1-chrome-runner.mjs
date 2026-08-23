import http from 'node:http';
import { execFileSync } from 'node:child_process';
import { createReadStream, existsSync, statSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || 'http://127.0.0.1:8080');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chromium is missing; reuse the configured Playwright cache');
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();

const sessionToken = await acquirePCSession(apiOrigin);
const proxy = await startSameOriginProxy(apiOrigin);
const browser = await chromium.launch({ executablePath: browserPath, headless: true });
const checks = [];

process.stdout.write(`PC_COMPOSED_UI1_ENV ${JSON.stringify({ browser: browserVersion, upstream: apiOrigin, proxy: 'random loopback' })}\n`);

try {
  const context = await browser.newContext();
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), sessionToken);
  const page = await context.newPage();
  const apiRequests = [];
  page.on('request', request => {
    const pathname = new URL(request.url()).pathname;
    if (pathname.startsWith('/api/v1/')) apiRequests.push(`${request.method()} ${pathname}`);
  });

  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api && window.Api.currentAccount() && document.querySelector('#tb-title')?.textContent === '工作台');

  const store = await page.evaluate(() => window.Api.storeView());
  record(checks, 'authenticated OWNER from composed API', store.account && store.account.role === 'owner');
  record(checks, 'server storefront rendered', store.name === '绥安食品');
  record(checks, 'dashboard reads real server facts', await hasText(page, '#content', '当日营收') && await hasText(page, '#content', '实时订单'));

  const routes = [
    ['orders', '订单管理'],
    ['finance', '财务与对账'],
    ['pending', '支付待处理'],
    ['products', '菜品管理'],
    ['product-import', '菜品批量导入'],
    ['categories', '分类管理'],
    ['settings', '营业设置'],
    ['layer', '开屏图层'],
    ['staff', '员工折扣白名单'],
    ['staff-import', '员工批量导入'],
    ['accounts', '商户账号名单'],
  ];
  for (const [route, title] of routes) {
    await page.locator(`a[data-r="${route}"]`).click();
    await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
    await page.waitForTimeout(100);
    const visible = await page.locator('#content').innerText();
    const failed = ['后台服务不可用', '暂不可用', '无法连接服务端', '登录已失效'].some(message => visible.includes(message));
    record(checks, `${title} composed read`, !failed);
  }

  const requiredRequests = [
    'GET /api/v1/admin/settings',
    'GET /api/v1/admin/me',
    'GET /api/v1/admin/stats',
    'GET /api/v1/admin/orders',
    'GET /api/v1/admin/finance/summary',
    'GET /api/v1/admin/pending-payments',
    'GET /api/v1/admin/products',
    'GET /api/v1/admin/categories',
    'GET /api/v1/admin/launch-layer',
    'GET /api/v1/admin/staff-whitelist',
    'GET /api/v1/admin/merchant-accounts',
  ];
  for (const expected of requiredRequests) {
    record(checks, `${expected} observed`, apiRequests.some(value => value === expected));
  }
  await context.close();
} finally {
  await browser.close();
  await proxy.close();
}

for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
process.stdout.write(`PC_COMPOSED_UI1_RESULT ${JSON.stringify({ status: checks.every(item => item.ok) ? 'PASS' : 'FAIL', checks: checks.length })}\n`);
if (checks.some(item => !item.ok)) process.exitCode = 1;

function record(items, name, ok) {
  items.push({ name, ok: Boolean(ok) });
}

async function hasText(page, selector, value) {
  return (await page.locator(selector).innerText()).includes(value);
}

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}

async function acquirePCSession(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', {
    method: 'POST', body: { code: `pc-ui1-${randomUUID()}` },
  }, 201);
  const bearer = exactToken(session.access_token, 'Mini session');
  await jsonRequest(origin, '/api/v1/me/bind-phone', {
    method: 'POST', bearer, idempotencyKey: randomUUID(), body: { code: `pc-phone-${randomUUID()}` },
  }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  const approvalSecret = exactString(payload.searchParams.get('approval_secret'), 'approval_secret');
  const loginID = exactString(login.login_id, 'login_id');
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', {
    method: 'POST', bearer, body: { login_id: loginID, approval_secret: approvalSecret, code: `pc-approve-${randomUUID()}` },
  }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', {
    method: 'POST', body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll_secret') },
  }, 200);
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session');
}

async function jsonRequest(origin, pathname, options, expectedStatus) {
  const headers = { Accept: 'application/json', 'Content-Type': 'application/json' };
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  const response = await fetch(`${origin}${pathname}`, {
    method: options.method, headers, body: JSON.stringify(options.body), redirect: 'error',
  });
  let body;
  try { body = await response.json(); } catch { throw new Error(`${pathname} returned invalid JSON`); }
  if (response.status !== expectedStatus) {
    const code = body && body.error && body.error.code;
    throw new Error(`${pathname} returned ${response.status}${code ? ` ${code}` : ''}`);
  }
  return body;
}

function exactToken(value, label) {
  const token = exactString(value, label);
  if (token.length < 32) throw new Error(`${label} is malformed`);
  return token;
}

function exactString(value, label) {
  if (typeof value !== 'string' || value.trim() === '') throw new Error(`${label} is missing`);
  return value;
}

async function startSameOriginProxy(upstreamOrigin) {
  const server = http.createServer(async (request, response) => {
    try {
      if (request.url.startsWith('/api/')) {
        await proxyAPI(request, response, upstreamOrigin);
        return;
      }
      serveStatic(request, response);
    } catch {
      response.writeHead(502, { 'Content-Type': 'text/plain; charset=utf-8' });
      response.end('local composed proxy unavailable');
    }
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

async function proxyAPI(request, response, upstreamOrigin) {
  const chunks = [];
  for await (const chunk of request) chunks.push(chunk);
  const headers = {};
  for (const [name, value] of Object.entries(request.headers)) {
    if (value !== undefined && !['host', 'content-length', 'connection'].includes(name)) headers[name] = value;
  }
  const upstream = await fetch(`${upstreamOrigin}${request.url}`, {
    method: request.method,
    headers,
    body: chunks.length ? Buffer.concat(chunks) : undefined,
    redirect: 'error',
  });
  const body = Buffer.from(await upstream.arrayBuffer());
  const contentType = upstream.headers.get('content-type');
  response.writeHead(upstream.status, {
    ...(contentType ? { 'Content-Type': contentType } : {}),
    'Content-Length': body.length,
    'Cache-Control': 'no-store',
  });
  response.end(body);
}

function serveStatic(request, response) {
  const pathname = decodeURIComponent(new URL(request.url, 'http://127.0.0.1').pathname);
  const relative = pathname === '/' ? 'web-admin/index.html' : pathname.replace(/^\/+/, '');
  const target = path.resolve(appsRoot, relative);
  if (!target.startsWith(`${appsRoot}${path.sep}`) || !existsSync(target) || !statSync(target).isFile()) {
    response.writeHead(404);
    response.end('not found');
    return;
  }
  const contentType = target.endsWith('.js') ? 'text/javascript; charset=utf-8' :
    target.endsWith('.css') ? 'text/css; charset=utf-8' :
      target.endsWith('.png') ? 'image/png' : 'text/html; charset=utf-8';
  response.writeHead(200, { 'Content-Type': contentType, 'Cache-Control': 'no-store' });
  createReadStream(target).pipe(response);
}
