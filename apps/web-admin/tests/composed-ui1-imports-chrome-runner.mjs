import http from 'node:http';
import zlib from 'node:zlib';
import { execFileSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, statSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomInt, randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || 'http://127.0.0.1:8080');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chromium is missing; set PLAYWRIGHT_BROWSERS_PATH=0 and reuse the configured cache');

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const evidenceRoot = path.join(repositoryRoot, '.scratch/overnight-pc-import');
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const suffix = randomUUID().replaceAll('-', '').slice(0, 8);
const names = {
  category: `UI1导入分类-${suffix}`,
  product: `UI1导入菜品-${suffix}`,
  staff: `UI1导入员工-${suffix}`,
  duplicateStaff: `UI1重复员工-${suffix}`,
};
const staffPhone = `166${String(randomInt(0, 100000000)).padStart(8, '0')}`;
const checks = [];
const cleanup = [];
const batchIDs = [];
const resources = { productID: '', categoryID: '', staffID: '' };
let sessionToken = '';
let serviceDate = '';
let browser = null;
let context = null;
let page = null;
let proxy = null;
let failure = null;
let previewRequests = 0;

process.stdout.write(`PC_COMPOSED_IMPORT_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin, proxy: 'random loopback', test_prefix: `UI1导入-${suffix}` })}\n`);

try {
  sessionToken = await acquirePCSession(apiOrigin);
  serviceDate = exactString((await adminGET('/api/v1/admin/settings')).service_date, 'service_date');
  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext();
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), sessionToken);
  page = await context.newPage();
  page.on('request', request => {
    if (request.method() === 'POST' && new URL(request.url()).pathname.endsWith('/import/preview')) previewRequests += 1;
  });

  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api && window.Api.currentAccount() && document.querySelector('#tb-title')?.textContent === '工作台');
  record('authenticated real OWNER PC session', (await page.evaluate(() => window.Api.currentAccount()))?.role === 'owner');

  await nonXlsxScenario();
  await productImportScenario();
  await staffImportScenario();
  record('committed import audits survive business-row cleanup', batchIDs.length === 2 && await auditsContain('COMMIT_IMPORT', batchIDs));
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
} finally {
  if (page) {
    try { await page.screenshot({ path: path.join(evidenceRoot, 'final-state.png'), fullPage: true }); }
    catch (error) { cleanup.push({ name: 'final screenshot', ok: false, error: safeMessage(error) }); }
  }
  await cleanupResiduals();
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
}

const passed = !failure && checks.every(item => item.ok) && cleanup.every(item => item.ok);
for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
for (const item of cleanup) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - cleanup ${item.name}${item.error ? `: ${item.error}` : ''}\n`);
const receipt = {
  schema: 'order.pc-composed-import-ui1.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), browser: browserVersion, upstream: apiOrigin,
  test_prefix: `UI1导入-${suffix}`, status: passed ? 'PASS' : 'FAIL', checks, cleanup,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_COMPOSED_IMPORT_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, cleanup: cleanup.length, receipt: '.scratch/overnight-pc-import/receipt.json' })}\n`);
if (!passed) process.exitCode = 1;

async function nonXlsxScenario() {
  await navigate('product-import', '菜品批量导入', '#f-file');
  const before = previewRequests;
  await page.locator('#f-file').setInputFiles({ name: `not-xlsx-${suffix}.csv`, mimeType: 'text/csv', buffer: Buffer.from('not an xlsx') });
  await page.locator('.toast').filter({ hasText: '只接受 .xlsx 文件' }).waitFor({ state: 'visible' });
  record('non-xlsx selection fails visibly before any preview request', previewRequests === before && !(await page.locator('#imp-result [data-ok]').count()));
}

async function productImportScenario() {
  await navigate('product-import', '菜品批量导入', '#f-file');
  const file = buildXlsx([
    ['菜品名称', '售价', '分类', '餐段可售', '描述'],
    [names.product, '18.88', names.category, '午餐', `真实 UI 导入 ${suffix}`],
  ], { numericCols: [1] });
  const preview = await mutate('POST', '/api/v1/admin/products/import/preview', () => page.locator('#f-file').setInputFiles({
    name: `products-${suffix}.xlsx`,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: file,
  }));
  const previewToken = exactToken(preview.body.preview_token, 'product preview token');
  await page.locator('#imp-result [data-ok]').waitFor({ state: 'visible' });
  const visible = await page.locator('#content').innerText();
  record('product xlsx preview is visible with one new category', preview.body.new_count === 1 && preview.body.error_count === 0 && preview.body.new_categories.includes(names.category) && visible.includes('新增 1 条') && visible.includes(`本次将新建分类：${names.category}`) && visible.includes('确认导入 1 条'));
  await screenshot('product-preview.png');

  const commit = await mutate('POST', '/api/v1/import/commit', () => page.locator('#imp-result [data-ok]').click());
  const batchID = exactID(commit.body.batch_id, 'product import batch');
  const commitKey = exactToken(commit.idempotencyKey, 'product commit idempotency key');
  batchIDs.push(batchID);
  await page.waitForFunction(() => document.querySelector('#tb-title')?.textContent === '菜品管理');
  await waitForText('#tbl-host tr[data-id]', names.product, true);
  const product = findNamed((await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(serviceDate)}`)).products, names.product);
  const category = findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category);
  resources.productID = exactID(product?.id, 'imported product');
  resources.categoryID = exactID(category?.id, 'imported category');
  record('product commit creates server product and category from visible confirmation', commit.body.new_count === 1 && product.price_cents === 1888 && product.meal_period === 'lunch' && product.category_id === resources.categoryID && product.images.length === 0 && category.enabled === true);
  await screenshot('product-committed.png');

  const replay = await directCommit(previewToken, commitKey);
  const conflict = await directCommit(previewToken, randomUUID());
  const matchingProducts = (await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(serviceDate)}`)).products.filter(item => item.name === names.product);
  record('same preview token exact replay is idempotent', replay.status === 200 && replay.body.duplicate === true && replay.body.batch_id === batchID && matchingProducts.length === 1);
  record('same preview token with a different operation key fails closed', conflict.status === 409 && conflict.body?.error?.code === 'IDEMPOTENCY_CONFLICT' && matchingProducts.length === 1);

  let row = page.locator('#tbl-host tr[data-id]').filter({ hasText: names.product });
  await row.locator('[data-act="edit"]').click();
  await page.locator('.drawer [data-a="del"]').click();
  await mutate('DELETE', `/api/v1/admin/products/${resources.productID}`, () => page.locator('.modal [data-a="ok"]').click());
  resources.productID = '';
  await waitForText('#tbl-host tr[data-id]', names.product, false);

  await navigate('categories', '分类管理', '#cat-host');
  row = page.locator('.cat-row').filter({ hasText: names.category });
  await row.locator('[data-del]').click();
  await mutate('DELETE', `/api/v1/admin/categories/${resources.categoryID}`, () => page.locator('.modal [data-a="ok"]').click());
  resources.categoryID = '';
  await waitForText('.cat-row', names.category, false);
  record('imported product and generated category are removed through visible UI', !findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category) && !findNamed((await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(serviceDate)}`)).products, names.product));
}

async function staffImportScenario() {
  await navigate('staff-import', '员工批量导入', '#f-file');
  const file = buildXlsx([
    ['姓名', '手机号'],
    [names.staff, staffPhone],
    [names.duplicateStaff, staffPhone],
  ]);
  const preview = await mutate('POST', '/api/v1/admin/staff-whitelist/import/preview', () => page.locator('#f-file').setInputFiles({
    name: `staff-${suffix}.xlsx`,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: file,
  }));
  exactToken(preview.body.preview_token, 'staff preview token');
  await page.locator('#imp-result [data-ok]').waitFor({ state: 'visible' });
  const visible = await page.locator('#content').innerText();
  const duplicate = (preview.body.rows || []).find(item => item.row === 3);
  record('duplicate phone in one xlsx is visibly isolated from the valid row', preview.body.new_count === 1 && preview.body.error_count === 1 && duplicate?.outcome === 'ERROR' && duplicate.reason.includes('重复') && visible.includes('新增 1 条') && visible.includes('异常 1 条') && visible.includes('跳过异常行，导入 1 条'));
  await screenshot('staff-preview-duplicate.png');

  const commit = await mutate('POST', '/api/v1/import/commit', () => page.locator('#imp-result [data-ok]').click());
  const batchID = exactID(commit.body.batch_id, 'staff import batch');
  batchIDs.push(batchID);
  await page.waitForFunction(() => document.querySelector('#tb-title')?.textContent === '员工折扣白名单');
  await waitForText('#staff-host tr[data-id]', names.staff, true);
  const items = (await adminGET('/api/v1/admin/staff-whitelist')).staff;
  const staff = findNamed(items, names.staff);
  resources.staffID = exactID(staff?.id, 'imported staff');
  record('staff commit writes only the valid row and keeps PII masked', commit.body.new_count === 1 && commit.body.skipped_count === 1 && !findNamed(items, names.duplicateStaff) && !staff.phone_masked.includes(staffPhone));
  await screenshot('staff-committed.png');

  const row = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staff });
  await row.locator('[data-act="del"]').click();
  await mutate('DELETE', `/api/v1/admin/staff-whitelist/${resources.staffID}`, () => page.locator('.modal [data-a="ok"]').click());
  resources.staffID = '';
  await waitForText('#staff-host tr[data-id]', names.staff, false);
  const after = (await adminGET('/api/v1/admin/staff-whitelist')).staff;
  record('imported staff row is removed through visible UI', !findNamed(after, names.staff) && !findNamed(after, names.duplicateStaff));
}

async function directCommit(previewToken, idempotencyKey) {
  return page.evaluate(async ({ previewToken, idempotencyKey }) => {
    const response = await fetch('/api/v1/import/commit', {
      method: 'POST',
      headers: {
        Accept: 'application/json', 'Content-Type': 'application/json',
        Authorization: `Bearer ${window.sessionStorage.getItem('pc_session_token')}`,
        'Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify({ preview_token: previewToken, skip_invalid: true }),
    });
    let body = null;
    try { body = await response.json(); } catch {}
    return { status: response.status, body };
  }, { previewToken, idempotencyKey });
}

async function navigate(route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).waitFor({ state: 'visible' });
}

async function mutate(method, pathname, action) {
  const responsePromise = page.waitForResponse(response => response.request().method() === method && new URL(response.url()).pathname === pathname);
  await action();
  const response = await responsePromise;
  let body = null;
  try { body = await response.json(); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  if (!response.ok()) throw new Error(`${method} ${pathname} returned ${response.status()}${body?.error?.code ? ` ${body.error.code}` : ''}`);
  return { body, idempotencyKey: response.request().headers()['idempotency-key'] || '' };
}

async function waitForText(selector, value, present) {
  await page.waitForFunction(({ selector, value, present }) => Array.from(document.querySelectorAll(selector)).some(node => (node.textContent || '').includes(value)) === present, { selector, value, present });
}

async function auditsContain(action, targetIDs) {
  const missing = new Set(targetIDs);
  let afterID = '';
  for (let pageNumber = 0; pageNumber < 30 && missing.size; pageNumber += 1) {
    const params = new URLSearchParams({ action, limit: '100', ...(afterID ? { after_id: afterID } : {}) });
    const body = await adminGET(`/api/v1/admin/audits?${params}`);
    for (const item of body.audits || []) {
      if (item.result_code === 'SUCCEEDED:OK') missing.delete(item.target_id);
    }
    if (!body.next_after_id) break;
    afterID = body.next_after_id;
  }
  return missing.size === 0;
}

async function cleanupResiduals() {
  if (!sessionToken) return;
  const discover = async () => {
    const products = (await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(serviceDate)}`)).products || [];
    const staff = (await adminGET('/api/v1/admin/staff-whitelist')).staff || [];
    const categories = (await adminGET('/api/v1/admin/categories')).categories || [];
    resources.productID ||= findNamed(products, names.product)?.id || '';
    resources.staffID ||= findNamed(staff, names.staff)?.id || '';
    resources.categoryID ||= findNamed(categories, names.category)?.id || '';
  };
  try { await discover(); }
  catch (error) { cleanup.push({ name: 'discover residual rows', ok: false, error: safeMessage(error) }); }
  for (const [name, id, pathname] of [
    ['product row', resources.productID, `/api/v1/admin/products/${resources.productID}`],
    ['staff row', resources.staffID, `/api/v1/admin/staff-whitelist/${resources.staffID}`],
    ['category row', resources.categoryID, `/api/v1/admin/categories/${resources.categoryID}`],
  ]) {
    if (!id) continue;
    try { await cleanupDELETE(pathname); cleanup.push({ name, ok: true }); }
    catch (error) { cleanup.push({ name, ok: false, error: safeMessage(error) }); }
  }
}

function record(name, ok) { checks.push({ name, ok: Boolean(ok) }); }
function findNamed(items, name) { return (Array.isArray(items) ? items : []).find(item => item.name === name); }
function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) throw new Error(`${label} id is malformed`);
  return value;
}
function exactToken(value, label) {
  const token = exactString(value, label);
  if (token.length < 16) throw new Error(`${label} is malformed`);
  return token;
}
function exactString(value, label) {
  if (typeof value !== 'string' || value.trim() === '') throw new Error(`${label} is missing`);
  return value;
}
function safeMessage(error) { return error?.message ? String(error.message).replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : 'unknown error'; }
async function screenshot(name) { await page.screenshot({ path: path.join(evidenceRoot, name), fullPage: true }); }

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  return parsed.origin;
}

async function acquirePCSession(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-import-${randomUUID()}` } }, 201);
  const bearer = exactToken(session.access_token, 'Mini session');
  await jsonRequest(origin, '/api/v1/me/bind-phone', { method: 'POST', bearer, idempotencyKey: randomUUID(), body: { code: `pc-import-phone-${randomUUID()}` } }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  const loginID = exactString(login.login_id, 'login_id');
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', { method: 'POST', bearer, body: { login_id: loginID, approval_secret: exactString(payload.searchParams.get('approval_secret'), 'approval_secret'), code: `pc-import-approve-${randomUUID()}` } }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll_secret') } }, 200);
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session');
}

async function adminGET(pathname) { return jsonRequest(apiOrigin, pathname, { method: 'GET', bearer: sessionToken }, 200); }
async function cleanupDELETE(pathname) {
  const response = await fetch(`${apiOrigin}${pathname}`, { method: 'DELETE', headers: { Accept: 'application/json', Authorization: `Bearer ${sessionToken}`, 'Idempotency-Key': randomUUID() }, redirect: 'error' });
  if (response.status !== 200 && response.status !== 404) throw new Error(`${pathname} cleanup returned ${response.status}`);
}
async function jsonRequest(origin, pathname, options, expectedStatus) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  const response = await fetch(`${origin}${pathname}`, { method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error' });
  let body;
  try { body = await response.json(); } catch { throw new Error(`${pathname} returned invalid JSON`); }
  if (response.status !== expectedStatus) throw new Error(`${pathname} returned ${response.status}${body?.error?.code ? ` ${body.error.code}` : ''}`);
  return body;
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

function crc32(buffer) {
  const table = new Int32Array(256);
  for (let i = 0; i < 256; i += 1) { let value = i; for (let k = 0; k < 8; k += 1) value = value & 1 ? 0xEDB88320 ^ (value >>> 1) : value >>> 1; table[i] = value; }
  let value = -1;
  for (const byte of buffer) value = table[(value ^ byte) & 0xff] ^ (value >>> 8);
  return (value ^ -1) >>> 0;
}
function zip(entries) {
  const locals = [], central = [];
  let offset = 0;
  for (const { name, data, deflate } of entries) {
    const raw = Buffer.from(data, 'utf8');
    const body = deflate ? zlib.deflateRawSync(raw) : raw;
    const nameBuffer = Buffer.from(name, 'utf8');
    const crc = crc32(raw);
    const local = Buffer.alloc(30);
    local.writeUInt32LE(0x04034b50, 0); local.writeUInt16LE(20, 4); local.writeUInt16LE(deflate ? 8 : 0, 8);
    local.writeUInt32LE(crc, 14); local.writeUInt32LE(body.length, 18); local.writeUInt32LE(raw.length, 22); local.writeUInt16LE(nameBuffer.length, 26);
    locals.push(local, nameBuffer, body);
    const directory = Buffer.alloc(46);
    directory.writeUInt32LE(0x02014b50, 0); directory.writeUInt16LE(20, 4); directory.writeUInt16LE(20, 6); directory.writeUInt16LE(deflate ? 8 : 0, 10);
    directory.writeUInt32LE(crc, 16); directory.writeUInt32LE(body.length, 20); directory.writeUInt32LE(raw.length, 24); directory.writeUInt16LE(nameBuffer.length, 28); directory.writeUInt32LE(offset, 42);
    central.push(directory, nameBuffer);
    offset += local.length + nameBuffer.length + body.length;
  }
  const directory = Buffer.concat(central);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0); end.writeUInt16LE(entries.length, 8); end.writeUInt16LE(entries.length, 10); end.writeUInt32LE(directory.length, 12); end.writeUInt32LE(offset, 16);
  return Buffer.concat([Buffer.concat(locals), directory, end]);
}
function buildXlsx(rows, options = {}) {
  const shared = [];
  const index = value => { const found = shared.indexOf(value); return found >= 0 ? found : shared.push(value) - 1; };
  const sheetRows = rows.map((row, rowIndex) => `<row r="${rowIndex + 1}">${row.map((value, columnIndex) => {
    if (value === '' || value == null) return '';
    const reference = `${columnName(columnIndex + 1)}${rowIndex + 1}`;
    if (options.numericCols?.includes(columnIndex) && /^-?\d+(\.\d+)?$/.test(value)) return `<c r="${reference}"><v>${value}</v></c>`;
    return `<c r="${reference}" t="s"><v>${index(String(value))}</v></c>`;
  }).join('')}</row>`).join('');
  const sheet = `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${sheetRows}</sheetData></worksheet>`;
  const strings = `<?xml version="1.0"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="${shared.length}" uniqueCount="${shared.length}">${shared.map(value => `<si><t>${escapeXML(value)}</t></si>`).join('')}</sst>`;
  return zip([
    { name: '[Content_Types].xml', data: '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>', deflate: false },
    { name: 'xl/workbook.xml', data: '<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>', deflate: true },
    { name: 'xl/sharedStrings.xml', data: strings, deflate: true },
    { name: 'xl/worksheets/sheet1.xml', data: sheet, deflate: true },
  ]);
}
function columnName(value) { let name = '', number = value; while (number > 0) { const remainder = (number - 1) % 26; name = String.fromCharCode(65 + remainder) + name; number = (number - remainder - 1) / 26; } return name; }
function escapeXML(value) { return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;'); }
