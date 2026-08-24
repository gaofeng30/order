import http from 'node:http';
import net from 'node:net';
import { spawn, spawnSync, execFileSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, mkdtempSync, rmSync, statSync, unlinkSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash, randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS;
if (!dependencyRoot) throw new Error('MINIPROGRAM_UI_DEPS is required');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const karma = dependencyRequire('karma');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chromium is missing; reuse the configured cache');

const docker = '/opt/homebrew/bin/docker';
const mysql = {
  host: exactString(process.env.ORDER_TEST_MYSQL_HOST, 'ORDER_TEST_MYSQL_HOST'),
  port: exactPort(process.env.ORDER_TEST_MYSQL_PORT),
  user: exactName(process.env.ORDER_TEST_MYSQL_USER, 'ORDER_TEST_MYSQL_USER'),
  password: exactString(process.env.ORDER_TEST_MYSQL_PASSWORD, 'ORDER_TEST_MYSQL_PASSWORD'),
  tls: exactString(process.env.ORDER_TEST_MYSQL_TLS_MODE, 'ORDER_TEST_MYSQL_TLS_MODE'),
  container: exactName(process.env.ORDER_TEST_MYSQL_INSTANCE, 'ORDER_TEST_MYSQL_INSTANCE'),
};
if (process.env.ORDER_TEST_MYSQL_ISOLATED !== 'YES') throw new Error('ORDER_TEST_MYSQL_ISOLATED=YES is required');
if (process.env.PLAYWRIGHT_BROWSERS_PATH !== '0') throw new Error('PLAYWRIGHT_BROWSERS_PATH=0 is required');

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const suffix = randomUUID().replaceAll('-', '').slice(0, 10);
const schema = `order_ac17_${Date.now()}_${suffix}`;
const buildRoot = mkdtempSync('/private/tmp/order-ac17-build.');
const evidenceRoot = `/private/tmp/order-ac17-three-client-${candidateSHA}`;
const apiLog = path.join(evidenceRoot, 'order-api.log');
const binaries = {
  api: path.join(buildRoot, 'order-api'),
  migrate: path.join(buildRoot, 'order-migrate'),
  bootstrap: path.join(buildRoot, 'order-bootstrap'),
};
const facts = {
  notice: `AC17三端同源-${suffix}`,
  category: `AC17分类-${suffix}`,
  product: `AC17菜品-${suffix}`,
  description: `PC可见写入-${suffix}`,
};
const checks = [];
const stages = [];
let apiProcess = null, proxy = null, browser = null, context = null, page = null;
let apiOrigin = '', pcToken = '', miniToken = '', objectKey = '', serviceDate = '', category = null, product = null;
let schemaCreated = false, failure = null;

mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });
process.stdout.write(`AC17_THREE_CLIENT_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, schema: 'fresh-v44 private', api: 'random loopback' })}\n`);

try {
  provisionFreshRuntime();
  apiOrigin = `http://127.0.0.1:${await freePort()}`;
  startAPI();
  await waitReady();
  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext();
  const sessions = await acquirePCSession(apiOrigin);
  pcToken = sessions.pcToken;
  miniToken = sessions.miniToken;
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  page = await context.newPage();
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner');
  record('pc-rendered-write authenticates an OWNER in rendered PC UI', true);

  await pcRenderedWrite();
  const baseSnapshot = await sharedSnapshot();
  record('PC rendered settings/category/product/launch are one server snapshot',
    baseSnapshot.notice === facts.notice && baseSnapshot.category_name === facts.category
      && baseSnapshot.product_name === facts.product && baseSnapshot.sold_out === false
      && baseSnapshot.launch_object_key === objectKey, baseSnapshot);

  await runMini('customer-before');
  record('new rendered customer context reads root storefront/menu/detail before sold-out', true);

  await runMini('merchant-soldout');
  const soldSnapshot = await sharedSnapshot();
  record('rendered Merchant Mini writes one dated sold-out fact', soldSnapshot.sold_out === true && soldSnapshot.service_date === serviceDate, soldSnapshot);

  await runMini('customer-after');
  record('new rendered customer context reads the Merchant Mini dated fact', true);

  await pcRenderedReadback();
  record('rendered PC reads the same sold-out product fact', true);

  const beforeFaults = await sharedSnapshot();
  await runMini('merchant-http-failure', `/api/v1/merchant/products/${product.id}/soldout`);
  await runMini('customer-http-failure', '/api/v1/menu/pickup-options');
  await runMini('customer-bad-object');
  const afterFaults = await sharedSnapshot();
  record('bad object and HTTP failures render zero false success and write zero facts',
    JSON.stringify(beforeFaults) === JSON.stringify(afterFaults), { unchanged: JSON.stringify(beforeFaults) === JSON.stringify(afterFaults) });
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
} finally {
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
  const stopped = await stopAPI();
  const objectClean = cleanupObject();
  const databaseClean = cleanupDatabase();
  try { rmSync(buildRoot, { recursive: true, force: true }); } catch {}
  record('private API stopped and fresh schema cleanup completed', stopped && databaseClean && objectClean,
    { api_stopped: stopped, database_dropped: databaseClean, object_removed: objectClean });
}

const passed = !failure && checks.every(item => item.ok);
for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
const receipt = {
  schema: 'order.ac17-three-client-source.ui1.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED', database_schema_version: 44,
  private_schema: schema.replace(/[a-f0-9]{10}$/, '[redacted]'),
  status: passed ? 'PASS' : 'FAIL', stages, checks,
  external_only: ['real WeChat Developer Tools UI2/UI3', 'real Tencent object storage'],
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`AC17_THREE_CLIENT_RESULT ${JSON.stringify({ status: receipt.status, candidate_sha: candidateSHA, checks: checks.length, stages: stages.length, receipt: path.join(evidenceRoot, 'receipt.json') })}\n`);
if (!passed) process.exitCode = 1;

function provisionFreshRuntime() {
  for (const [name, target] of Object.entries(binaries)) {
    runChecked('go', ['build', '-o', target, `./services/api/cmd/order-${name}`], repositoryRoot);
  }
  mysqlExec(`CREATE DATABASE \`${schema}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci`);
  schemaCreated = true;
  const env = runtimeEnv();
  runChecked(binaries.migrate, [], repositoryRoot, env);
  const migrationState = mysqlExec(`SELECT IF(COUNT(*)=44 AND MAX(version)=44 AND SUM(dirty)=0,1,0) FROM \`${schema}\`.schema_migrations`).trim();
  if (migrationState !== '1') throw new Error(`fresh schema is not clean v44: ${migrationState}`);
  runChecked(binaries.bootstrap, [], repositoryRoot, {
    ...env,
    ORDER_BOOTSTRAP_OWNER_PHONE: '+8613800000000',
    ORDER_BOOTSTRAP_OWNER_NAME: 'AC17本地主账号',
    ORDER_BOOTSTRAP_STORE_NAME: '绥安食品',
    ORDER_BOOTSTRAP_STORE_ADDRESS: 'AC17本地地址',
    ORDER_BOOTSTRAP_PICKUP_POINT: 'AC17本地取餐点',
  });
  record('order-bootstrap created only canonical root facts in fresh v44', true);
}

function startAPI() {
  apiProcess = spawn(binaries.api, [], {
    cwd: repositoryRoot,
    env: { ...process.env, ...runtimeEnv(), ORDER_API_HTTP_ADDR: new URL(apiOrigin).host, ORDER_API_SHUTDOWN_TIMEOUT: '2s', ORDER_LOCAL_PAYMENT_AUTO_PAY: 'true' },
    stdio: ['ignore', 'ignore', 'pipe'],
  });
  let output = '';
  apiProcess.stderr.on('data', chunk => { output += chunk; writeFileSync(apiLog, output.slice(-1024 * 1024), { mode: 0o600 }); });
}

async function waitReady() {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    if (apiProcess.exitCode !== null) throw new Error(`order-api exited ${apiProcess.exitCode}`);
    try { if ((await fetch(`${apiOrigin}/health/ready`)).status === 200) return; } catch {}
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('order-api readiness timed out');
}

async function pcRenderedWrite() {
  await navigate('settings', '营业设置', '#save');
  await page.locator('[data-b="营业中"]').click();
  for (const date of await page.locator('[data-date]').all()) {
    if (await date.getAttribute('data-open') !== 'true') await date.click();
  }
  for (const meal of ['lunch', 'dinner']) {
    await page.locator(`[data-mp="${meal}"][data-k="cutoff"]`).fill(meal === 'lunch' ? '22:56' : '23:58');
    await page.locator(`[data-mp="${meal}"][data-k="from"]`).fill(meal === 'lunch' ? '22:57' : '23:59');
    await page.locator(`[data-mp="${meal}"][data-k="to"]`).fill(meal === 'lunch' ? '22:57' : '23:59');
  }
  await page.locator('#f-notice').fill(facts.notice);
  const settingsWrite = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === '/api/v1/admin/settings');
  await page.locator('#save').click();
  if ((await settingsWrite).status() !== 200) throw new Error('PC settings write failed');
  const settings = await adminGET('/api/v1/admin/settings');
  serviceDate = exactDate(settings.service_date);

  await navigate('categories', '分类管理', '#cat-host');
  await page.locator('[data-new]').click();
  await page.locator('.modal #c-name').fill(facts.category);
  const categoryWrite = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/admin/categories');
  await page.locator('.modal [data-a="ok"]').click();
  category = payloadOf(await (await categoryWrite).json(), 'category');
  await page.locator('.cat-row').filter({ hasText: facts.category }).waitFor({ state: 'visible' });

  await navigate('products', '菜品管理', '#tbl-host');
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(facts.product);
  await page.locator('.drawer #f-price').fill('18.88');
  await page.locator('.drawer #f-meal').selectOption('all');
  await page.locator('.drawer #f-cat').selectOption({ label: facts.category });
  await page.locator('.drawer #f-desc').fill(facts.description);
  const productWrite = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/admin/products');
  await page.locator('.drawer [data-a="ok"]').click();
  product = payloadOf(await (await productWrite).json(), 'product');
  await page.locator('#tbl-host tr[data-id]').filter({ hasText: facts.product }).waitFor({ state: 'visible' });

  await navigate('layer', '开屏图层', '#phone');
  const png = uniquePNG(suffix);
  const expectedObjectKey = `images/${createHash('sha256').update(png).digest('hex')}.png`;
  if (existsSync(path.join('/private/tmp/order-local-objects', expectedObjectKey))) throw new Error('AC-17 object fixture is not unique');
  const upload = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/upload');
  await page.locator('#file').setInputFiles({ name: `ac17-${suffix}.png`, mimeType: 'image/png', buffer: png });
  objectKey = exactString(payloadOf(await (await upload).json(), 'image').object_key, 'launch object key');
  if (objectKey !== expectedObjectKey) throw new Error('uploaded launch object digest drifted');
  await page.locator('#lay-img').waitFor({ state: 'visible' });
  if (!(await page.locator('#en').evaluate(node => node.classList.contains('on')))) await page.locator('#en').click();
  const layerWrite = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === '/api/v1/admin/launch-layer');
  await page.locator('#save').click();
  if ((await layerWrite).status() !== 200) throw new Error('PC launch layer write failed');
  record('pc-rendered-write visibly saved settings/category/product/launch', true);
}

async function runMini(mode, failPath = '') {
  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_AC17_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_AC17_SETUP = JSON.stringify({
    mode, fail_path: failPath, mini_token: miniToken, expires_at: new Date(Date.now() + 3600000).toISOString(),
    store_name: '绥安食品', launch_object_key: objectKey,
    category: { id: String(category.id), name: facts.category },
    product: { id: String(product.id), name: facts.product },
  });
  const config = await karma.config.parseConfig(path.join(testsRoot, 'ac17-three-client-source-karma.cjs'), { singleRun: true }, { promiseConfig: true, throwErrors: true });
  const exit = await new Promise((resolve, reject) => {
    const server = new karma.Server(config, resolve);
    server.start().catch(reject);
  });
  stages.push({ name: mode, browser: browserVersion, status: exit === 0 ? 'PASS' : 'FAIL' });
  if (exit !== 0) throw new Error(`${mode} rendered Chrome Gate exited ${exit}`);
}

async function pcRenderedReadback() {
  await navigate('products', '菜品管理', '#tbl-host');
  const row = page.locator('#tbl-host tr[data-id]').filter({ hasText: facts.product });
  await row.waitFor({ state: 'visible' });
  if (!(await row.innerText()).includes('恢复售卖')) throw new Error('PC did not render the Merchant Mini sold-out fact');
  const settings = await adminGET('/api/v1/admin/settings');
  if (settings.notice !== facts.notice) throw new Error('PC settings drifted after Mini write');
}

async function sharedSnapshot() {
  const [settings, categories, productView, layer] = await Promise.all([
    adminGET('/api/v1/admin/settings'), adminGET('/api/v1/admin/categories'),
    adminGET(`/api/v1/admin/products/${product.id}?service_date=${encodeURIComponent(serviceDate)}`),
    adminGET('/api/v1/admin/launch-layer'),
  ]);
  const stored = payloadOf(productView, 'product');
  return {
    service_date: serviceDate, notice: settings.notice,
    category_name: categories.categories.find(item => String(item.id) === String(category.id))?.name || '',
    product_name: stored.name, sold_out: stored.sold_out,
    launch_object_key: layer.image_object_key || '',
  };
}

async function navigate(route, title, ready) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(ready).waitFor({ state: 'visible' });
}

async function acquirePCSession(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `ac17-login-${suffix}` } }, 201);
  const token = exactString(session.access_token, 'Mini token');
  await jsonRequest(origin, '/api/v1/me/bind-phone', { method: 'POST', bearer: token, key: randomUUID(), body: { code: `ac17-phone-${suffix}` } }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr payload'));
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', { method: 'POST', bearer: token, body: { login_id: login.login_id, approval_secret: payload.searchParams.get('approval_secret'), code: `ac17-approve-${suffix}` } }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: login.login_id, poll_secret: login.poll_secret } }, 200);
  return { miniToken: token, pcToken: exactString(poll.session?.token, 'PC token') };
}

async function adminGET(pathname) { return jsonRequest(apiOrigin, pathname, { method: 'GET', bearer: pcToken }, 200); }
async function jsonRequest(origin, pathname, options, expected) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  const response = await fetch(`${origin}${pathname}`, { method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error' });
  let body = null; try { body = await response.json(); } catch {}
  if (response.status !== expected) throw new Error(`${pathname} returned ${response.status}/${body?.error?.code || 'UNKNOWN'}`);
  return body;
}

async function startSameOriginProxy(upstream) {
  const server = http.createServer(async (request, response) => {
    try {
      if (request.method === 'OPTIONS') {
        response.writeHead(204, {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Headers': 'authorization,content-type,idempotency-key',
          'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
        });
        response.end();
        return;
      }
      if (request.url.startsWith('/api/')) return proxyAPI(request, response, upstream);
      const clean = decodeURIComponent((request.url || '/').split('?')[0]);
      const relative = clean === '/web-admin/' || clean === '/web-admin/index.html' ? 'web-admin/index.html' : clean.replace(/^\//, '');
      const file = path.resolve(appsRoot, relative);
      if (!file.startsWith(appsRoot + path.sep) || !existsSync(file) || !statSync(file).isFile()) { response.writeHead(404); response.end(); return; }
      const types = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.png': 'image/png', '.svg': 'image/svg+xml' };
      response.writeHead(200, { 'Content-Type': types[path.extname(file)] || 'application/octet-stream', 'Cache-Control': 'no-store' });
      createReadStream(file).pipe(response);
    } catch { response.writeHead(502); response.end(); }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { origin: `http://127.0.0.1:${server.address().port}`, close: () => new Promise(resolve => server.close(resolve)) };
}

async function proxyAPI(request, response, upstream) {
  const chunks = []; for await (const chunk of request) chunks.push(chunk);
  const target = await fetch(`${upstream}${request.url}`, { method: request.method, headers: request.headers, body: ['GET', 'HEAD'].includes(request.method) ? undefined : Buffer.concat(chunks), redirect: 'manual' });
  const headers = { 'Access-Control-Allow-Origin': '*' }; target.headers.forEach((value, key) => { if (!['content-encoding', 'content-length', 'transfer-encoding'].includes(key)) headers[key] = value; });
  response.writeHead(target.status, headers); response.end(Buffer.from(await target.arrayBuffer()));
}

function runtimeEnv() {
  return {
    ...process.env,
    ORDER_ENV: 'development', ORDER_DB_HOST: mysql.host, ORDER_DB_PORT: mysql.port,
    ORDER_DB_NAME: schema, ORDER_DB_USER: mysql.user, ORDER_DB_PASSWORD: mysql.password, ORDER_DB_TLS_MODE: mysql.tls,
    ORDER_WECHAT_MINIPROGRAM_APP_ID: 'wx-local-ac17', ORDER_WECHAT_MINIPROGRAM_APP_SECRET: 'ac17-local-secret',
  };
}

function mysqlExec(sql) {
  const result = spawnSync(docker, ['exec', '-e', `MYSQL_PWD=${mysql.password}`, mysql.container, 'mysql', '--batch', '--raw', '--skip-column-names', '-u', mysql.user, '--execute', sql], { encoding: 'utf8', maxBuffer: 5 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(`MySQL command failed: ${safeMessage(result.stderr)}`);
  return result.stdout;
}

function runChecked(command, args, cwd, extraEnv = {}) {
  const result = spawnSync(command, args, { cwd, env: { ...process.env, ...extraEnv }, encoding: 'utf8', maxBuffer: 20 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(`${path.basename(command)} failed: ${safeMessage(result.stderr || result.stdout)}`);
}

async function stopAPI() {
  if (!apiProcess) return true;
  if (apiProcess.exitCode !== null) return apiProcess.exitCode === 0;
  apiProcess.kill('SIGTERM');
  return new Promise(resolve => {
    const timer = setTimeout(() => { apiProcess.kill('SIGKILL'); resolve(false); }, 10000);
    apiProcess.once('exit', code => { clearTimeout(timer); resolve(code === 0); });
  });
}

function cleanupObject() {
  if (!objectKey) return true;
  const root = '/private/tmp/order-local-objects';
  const target = path.resolve(root, objectKey);
  if (!target.startsWith(root + path.sep)) return false;
  try { if (existsSync(target)) unlinkSync(target); return !existsSync(target); } catch { return false; }
}

function cleanupDatabase() {
  if (!schemaCreated) return true;
  try {
    mysqlExec(`DROP DATABASE IF EXISTS \`${schema}\``);
    schemaCreated = false;
    return mysqlExec(`SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='${schema}'`).trim() === '0';
  } catch { return false; }
}

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => { const port = server.address().port; server.close(error => error ? reject(error) : resolve(port)); });
  });
}

function uniquePNG(marker) {
  const base = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M/wHwAF/gL+Wn9Z5QAAAABJRU5ErkJggg==', 'base64');
  const iend = base.subarray(base.length - 12);
  const text = Buffer.from(`ac17\0${marker}`, 'ascii');
  const type = Buffer.from('tEXt', 'ascii');
  const crcInput = Buffer.concat([type, text]);
  const chunk = Buffer.alloc(12 + text.length);
  chunk.writeUInt32BE(text.length, 0);
  type.copy(chunk, 4);
  text.copy(chunk, 8);
  chunk.writeUInt32BE(crc32(crcInput), 8 + text.length);
  return Buffer.concat([base.subarray(0, base.length - 12), chunk, iend]);
}

function crc32(buffer) {
  let value = 0xffffffff;
  for (const byte of buffer) {
    value ^= byte;
    for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ (value & 1 ? 0xedb88320 : 0);
  }
  return (value ^ 0xffffffff) >>> 0;
}

function payloadOf(body, name) { const value = body && body[name]; if (!value || typeof value !== 'object') throw new Error(`${name} missing`); return value; }
function exactString(value, name) { if (typeof value !== 'string' || !value.trim()) throw new Error(`${name} missing`); return value; }
function exactName(value, name) { const result = exactString(value, name); if (!/^[A-Za-z0-9_.-]+$/.test(result)) throw new Error(`${name} invalid`); return result; }
function exactPort(value) { const port = Number(value); if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error('ORDER_TEST_MYSQL_PORT invalid'); return String(port); }
function exactDate(value) { if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) throw new Error('service date invalid'); return value; }
function safeMessage(error) { return String(error?.message || error || 'unknown').replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]').slice(0, 2000); }
function record(name, ok, detail) { checks.push({ name, ok: Boolean(ok), ...(detail === undefined ? {} : { detail }) }); }
