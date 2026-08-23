import http from 'node:http';
import { execFileSync } from 'node:child_process';
import { mkdirSync, createReadStream, existsSync, statSync, writeFileSync } from 'node:fs';
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
const evidenceRoot = path.join(repositoryRoot, '.scratch/overnight-pc');
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const suffix = randomUUID().replaceAll('-', '').slice(0, 8);
const names = {
  category: `UI1-${suffix}`,
  product: `UI1菜品-${suffix}`,
  productUpdated: `UI1菜品改-${suffix}`,
  staff: `UI1员工-${suffix}`,
  staffUpdated: `UI1员工改-${suffix}`,
  account: `UI1商户-${suffix}`,
  accountUpdated: `UI1商户改-${suffix}`,
  noticeMarker: `[UI1-${suffix}]`,
};
const phones = {
  staff: syntheticPhone('188'),
  account: syntheticPhone('177'),
};

const checks = [];
const cleanup = [];
const resources = { categoryID: '', productID: '', staffID: '', accountID: '' };
let initialSettings = null;
let settingsDirty = false;
let initialDiscount = null;
let discountDirty = false;
let sessionToken = '';
let proxy = null;
let browser = null;
let context = null;
let page = null;
let failure = null;

process.stdout.write(`PC_COMPOSED_WRITES_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin, proxy: 'random loopback', test_prefix: names.category })}\n`);

try {
  sessionToken = await acquirePCSession(apiOrigin);
  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext();
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), sessionToken);
  page = await context.newPage();

  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api && window.Api.currentAccount() && document.querySelector('#tb-title')?.textContent === '工作台');
  const owner = await page.evaluate(() => window.Api.currentAccount());
  record('authenticated real OWNER PC session', owner && owner.role === 'owner');

  initialSettings = await adminGET('/api/v1/admin/settings');
  initialDiscount = Number((await adminGET('/api/v1/admin/discount-rate')).rate_percent);
  record('settings and discount baseline read from composed API', validSettings(initialSettings) && Number.isInteger(initialDiscount));

  await categoryAndProductScenario();
  await staffAndDiscountScenario();
  await settingsScenario();
  await merchantAccountScenario();
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
  schema: 'order.pc-composed-writes-ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  browser: browserVersion,
  upstream: apiOrigin,
  test_prefix: names.category,
  status: passed ? 'PASS' : 'FAIL',
  checks,
  cleanup,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_COMPOSED_WRITES_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, cleanup: cleanup.length, receipt: '.scratch/overnight-pc/receipt.json' })}\n`);
if (!passed) process.exitCode = 1;

async function categoryAndProductScenario() {
  await navigate('categories', '分类管理', '#cat-host');
  await page.locator('[data-new]').click();
  await page.locator('#c-name').fill(names.category);
  const createdCategory = payloadOf(await mutate('POST', '/api/v1/admin/categories', () => page.locator('.modal [data-a="ok"]').click()), 'category');
  resources.categoryID = exactID(createdCategory.id, 'created category');
  await waitForText('.cat-row', names.category, true);
  const serverCategory = findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category);
  record('category create is visible and read back from server', serverCategory && serverCategory.id === resources.categoryID && serverCategory.enabled === true);
  await screenshot('category-created.png');

  let categoryRow = page.locator('.cat-row').filter({ hasText: names.category });
  await mutate('PUT', `/api/v1/admin/categories/${resources.categoryID}`, () => categoryRow.locator('[data-on]').click());
  await waitForServer(async () => findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category)?.enabled === false);
  categoryRow = page.locator('.cat-row').filter({ hasText: names.category });
  record('category update is visible and persisted', !(await categoryRow.locator('[data-on]').evaluate(node => node.classList.contains('on'))));

  categoryRow = page.locator('.cat-row').filter({ hasText: names.category });
  await mutate('PUT', `/api/v1/admin/categories/${resources.categoryID}`, () => categoryRow.locator('[data-on]').click());
  await waitForServer(async () => findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category)?.enabled === true);
  record('category enabled state restored before product flow', true);

  await navigate('products', '菜品管理', '#tbl-host');
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(names.product);
  await page.locator('.drawer #f-price').fill('12.34');
  await page.locator('.drawer #f-meal').selectOption('all');
  await page.locator('.drawer #f-cat').selectOption({ label: names.category });
  await page.locator('.drawer #f-desc').fill(`真实组合写链路 ${suffix}`);
  const createdProduct = payloadOf(await mutate('POST', '/api/v1/admin/products', () => page.locator('.drawer [data-a="ok"]').click()), 'product');
  resources.productID = exactID(createdProduct.id, 'created product');
  await waitForText('#tbl-host tr[data-id]', names.product, true);
  let serverProduct = await readProduct(resources.productID);
  record('product create is visible and read back from server', serverProduct.name === names.product && serverProduct.price_cents === 1234 && serverProduct.category_id === resources.categoryID);
  await screenshot('product-created.png');

  let productRow = page.locator('#tbl-host tr[data-id]').filter({ hasText: names.product });
  await productRow.locator('[data-act="edit"]').click();
  await page.locator('.drawer #f-name').fill(names.productUpdated);
  await page.locator('.drawer #f-price').fill('13.45');
  await page.locator('.drawer #f-desc').fill(`真实组合写链路已更新 ${suffix}`);
  await mutate('PUT', `/api/v1/admin/products/${resources.productID}`, () => page.locator('.drawer [data-a="ok"]').click());
  await waitForText('#tbl-host tr[data-id]', names.productUpdated, true);
  serverProduct = await readProduct(resources.productID);
  record('product update is visible and read back from server', serverProduct.name === names.productUpdated && serverProduct.price_cents === 1345 && serverProduct.description.endsWith(suffix));

  productRow = page.locator('#tbl-host tr[data-id]').filter({ hasText: names.productUpdated });
  await productRow.locator('[data-act="edit"]').click();
  await page.locator('.drawer [data-a="del"]').click();
  await mutate('DELETE', `/api/v1/admin/products/${resources.productID}`, () => page.locator('.modal [data-a="ok"]').click());
  const deletedProductID = resources.productID;
  resources.productID = '';
  await waitForText('#tbl-host tr[data-id]', names.productUpdated, false);
  record('product delete is visible and removed from server', !(await productExists(deletedProductID)));

  await navigate('categories', '分类管理', '#cat-host');
  categoryRow = page.locator('.cat-row').filter({ hasText: names.category });
  await categoryRow.locator('[data-del]').click();
  await mutate('DELETE', `/api/v1/admin/categories/${resources.categoryID}`, () => page.locator('.modal [data-a="ok"]').click());
  const deletedCategoryID = resources.categoryID;
  resources.categoryID = '';
  await waitForText('.cat-row', names.category, false);
  record('empty category delete recycles the business row', !findNamed((await adminGET('/api/v1/admin/categories')).categories, names.category));
  record('deleted category keeps a redacted audit receipt', await auditContains('DELETE_CATEGORY', deletedCategoryID));
}

async function staffAndDiscountScenario() {
  await navigate('staff', '员工折扣白名单', '#staff-host');
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(names.staff);
  await page.locator('.drawer #f-phone').fill(phones.staff);
  const createdStaff = await mutate('POST', '/api/v1/admin/staff-whitelist', () => page.locator('.drawer [data-a="ok"]').click());
  resources.staffID = exactID(createdStaff.id, 'created staff');
  await waitForText('#staff-host tr[data-id]', names.staff, true);
  let serverStaff = findNamed((await adminGET('/api/v1/admin/staff-whitelist')).staff, names.staff);
  record('staff create is visible with masked PII and read back from server', serverStaff && serverStaff.id === resources.staffID && serverStaff.enabled === true && !serverStaff.phone_masked.includes(phones.staff));
  await screenshot('staff-created.png');

  let staffRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staff });
  await staffRow.locator('[data-act="edit"]').click();
  await page.locator('.drawer #f-name').fill(names.staffUpdated);
  await mutate('PUT', `/api/v1/admin/staff-whitelist/${resources.staffID}`, () => page.locator('.drawer [data-a="ok"]').click());
  await waitForText('#staff-host tr[data-id]', names.staffUpdated, true);
  serverStaff = findNamed((await adminGET('/api/v1/admin/staff-whitelist')).staff, names.staffUpdated);
  record('staff update is visible and read back from server', serverStaff && serverStaff.id === resources.staffID);

  staffRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staffUpdated });
  await mutate('PUT', `/api/v1/admin/staff-whitelist/${resources.staffID}`, () => staffRow.locator('[data-act="toggle"]').click());
  await waitForServer(async () => findNamed((await adminGET('/api/v1/admin/staff-whitelist')).staff, names.staffUpdated)?.enabled === false);
  staffRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staffUpdated });
  record('staff enabled state update is visible and persisted', (await staffRow.innerText()).includes('停用'));

  const changedRate = initialDiscount === 100 ? 99 : initialDiscount + 1;
  await page.locator('#f-rate').fill(String(changedRate));
  await mutate('PUT', '/api/v1/admin/discount-rate', () => page.locator('[data-save-rate]').click());
  discountDirty = true;
  record('discount update is visible and read back from server', Number(await page.locator('#f-rate').inputValue()) === changedRate && Number((await adminGET('/api/v1/admin/discount-rate')).rate_percent) === changedRate);

  await page.locator('#f-rate').fill(String(initialDiscount));
  await mutate('PUT', '/api/v1/admin/discount-rate', () => page.locator('[data-save-rate]').click());
  await waitForServer(async () => Number((await adminGET('/api/v1/admin/discount-rate')).rate_percent) === initialDiscount);
  discountDirty = false;
  record('discount rate restored through visible UI', Number(await page.locator('#f-rate').inputValue()) === initialDiscount);

  staffRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staffUpdated });
  await staffRow.locator('[data-act="del"]').click();
  await mutate('DELETE', `/api/v1/admin/staff-whitelist/${resources.staffID}`, () => page.locator('.modal [data-a="ok"]').click());
  resources.staffID = '';
  await waitForText('#staff-host tr[data-id]', names.staffUpdated, false);
  record('staff delete is visible and removes the business row', !findNamed((await adminGET('/api/v1/admin/staff-whitelist')).staff, names.staffUpdated));
}

async function settingsScenario() {
  await navigate('settings', '营业设置', '#save');
  const changedNotice = `${initialSettings.notice || ''}${initialSettings.notice ? ' ' : ''}${names.noticeMarker}`;
  await page.locator('#f-notice').fill(changedNotice);
  await mutate('PUT', '/api/v1/admin/settings', () => page.locator('#save').click());
  settingsDirty = true;
  let serverSettings = await adminGET('/api/v1/admin/settings');
  record('business settings save is visible and read back from server', await page.locator('#f-notice').inputValue() === changedNotice && serverSettings.notice === changedNotice);
  await screenshot('settings-changed.png');

  await page.locator('#f-notice').fill(initialSettings.notice || '');
  await mutate('PUT', '/api/v1/admin/settings', () => page.locator('#save').click());
  await waitForServer(async () => (await adminGET('/api/v1/admin/settings')).notice === (initialSettings.notice || ''));
  settingsDirty = false;
  serverSettings = await adminGET('/api/v1/admin/settings');
  record('business settings restored through visible UI', sameSettings(serverSettings, initialSettings));
}

async function merchantAccountScenario() {
  await navigate('accounts', '商户账号名单', '#acc-host');
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(names.account);
  await page.locator('.drawer #f-phone').fill(phones.account);
  await page.locator('.drawer #f-role').selectOption('staff');
  const createdAccount = await mutate('POST', '/api/v1/admin/merchant-accounts', () => page.locator('.drawer [data-a="ok"]').click());
  resources.accountID = exactID(createdAccount.id, 'created merchant account');
  await waitForText('#acc-host tr[data-id]', names.account, true);
  let serverAccount = findNamed((await adminGET('/api/v1/admin/merchant-accounts')).accounts, names.account);
  record('merchant subaccount create is visible and read back from server', serverAccount && serverAccount.id === resources.accountID && serverAccount.role === 'SUBACCOUNT' && serverAccount.enabled === true);
  await screenshot('account-created.png');

  let accountRow = page.locator('#acc-host tr[data-id]').filter({ hasText: names.account });
  await accountRow.locator('[data-act="edit"]').click();
  await page.locator('.drawer #f-name').fill(names.accountUpdated);
  await mutate('PUT', `/api/v1/admin/merchant-accounts/${resources.accountID}`, () => page.locator('.drawer [data-a="ok"]').click());
  await waitForText('#acc-host tr[data-id]', names.accountUpdated, true);
  serverAccount = findNamed((await adminGET('/api/v1/admin/merchant-accounts')).accounts, names.accountUpdated);
  record('merchant account update is visible and read back from server', serverAccount && serverAccount.id === resources.accountID && serverAccount.role === 'SUBACCOUNT');

  accountRow = page.locator('#acc-host tr[data-id]').filter({ hasText: names.accountUpdated });
  await accountRow.locator('[data-act="del"]').click();
  await mutate('DELETE', `/api/v1/admin/merchant-accounts/${resources.accountID}`, () => page.locator('.modal [data-a="ok"]').click());
  resources.accountID = '';
  await waitForText('#acc-host tr[data-id]', names.accountUpdated, false);
  record('merchant account delete is visible and removes the business row', !findNamed((await adminGET('/api/v1/admin/merchant-accounts')).accounts, names.accountUpdated));
}

async function navigate(route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).waitFor({ state: 'visible' });
}

async function mutate(method, pathname, action) {
  const responsePromise = page.waitForResponse(response => {
    const url = new URL(response.url());
    return response.request().method() === method && url.pathname === pathname;
  });
  await action();
  const response = await responsePromise;
  let body = null;
  try { body = await response.json(); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  if (!response.ok()) {
    const code = body && body.error && body.error.code;
    throw new Error(`${method} ${pathname} returned ${response.status()}${code ? ` ${code}` : ''}`);
  }
  return body;
}

async function waitForText(selector, value, present) {
  await page.waitForFunction(({ selector, value, present }) => {
    const found = Array.from(document.querySelectorAll(selector)).some(node => (node.textContent || '').includes(value));
    return found === present;
  }, { selector, value, present });
}

async function waitForServer(predicate) {
  const deadline = Date.now() + 5000;
  let lastError = null;
  while (Date.now() < deadline) {
    try { if (await predicate()) return; }
    catch (error) { lastError = error; }
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw lastError || new Error('server readback did not converge');
}

async function readProduct(id) {
  return payloadOf(await adminGET(`/api/v1/admin/products/${id}?service_date=${encodeURIComponent(initialSettings.service_date)}`), 'product');
}

async function productExists(id) {
  const response = await fetch(`${apiOrigin}/api/v1/admin/products/${id}?service_date=${encodeURIComponent(initialSettings.service_date)}`, {
    headers: { Accept: 'application/json', Authorization: `Bearer ${sessionToken}` },
    redirect: 'error',
  });
  if (response.status === 404) return false;
  if (response.status !== 200) throw new Error(`product readback returned ${response.status}`);
  return true;
}

async function auditContains(action, targetID) {
  let afterID = '';
  for (let pageNumber = 0; pageNumber < 20; pageNumber += 1) {
    const suffix = new URLSearchParams({ action, limit: '100', ...(afterID ? { after_id: afterID } : {}) });
    const body = await adminGET(`/api/v1/admin/audits?${suffix}`);
    if ((body.audits || []).some(entry => entry.action === action && entry.target_id === targetID && entry.result_code === 'SUCCEEDED:OK')) return true;
    if (!body.next_after_id) return false;
    afterID = body.next_after_id;
  }
  throw new Error(`audit pagination exceeded for ${action}`);
}

async function screenshot(name) {
  await page.screenshot({ path: path.join(evidenceRoot, name), fullPage: true });
}

async function cleanupResiduals() {
  if (!sessionToken) return;
  const jobs = [
    ['product row', () => cleanupDELETE(`/api/v1/admin/products/${resources.productID}`), () => resources.productID],
    ['staff row', () => cleanupDELETE(`/api/v1/admin/staff-whitelist/${resources.staffID}`), () => resources.staffID],
    ['merchant account row', () => cleanupDELETE(`/api/v1/admin/merchant-accounts/${resources.accountID}`), () => resources.accountID],
    ['category row', () => cleanupDELETE(`/api/v1/admin/categories/${resources.categoryID}`), () => resources.categoryID],
  ];
  for (const [name, action, id] of jobs) {
    if (!id()) continue;
    try { await action(); cleanup.push({ name, ok: true }); }
    catch (error) { cleanup.push({ name, ok: false, error: safeMessage(error) }); }
  }
  if (discountDirty && Number.isInteger(initialDiscount)) {
    try {
      await adminWrite('/api/v1/admin/discount-rate', 'PUT', { rate_percent: initialDiscount });
      cleanup.push({ name: 'discount baseline', ok: true });
    } catch (error) { cleanup.push({ name: 'discount baseline', ok: false, error: safeMessage(error) }); }
  }
  if (settingsDirty && initialSettings) {
    try {
      await adminWrite('/api/v1/admin/settings', 'PUT', settingsWriteBody(initialSettings));
      cleanup.push({ name: 'settings baseline', ok: true });
    } catch (error) { cleanup.push({ name: 'settings baseline', ok: false, error: safeMessage(error) }); }
  }
}

function record(name, ok) { checks.push({ name, ok: Boolean(ok) }); }
function findNamed(items, name) { return (Array.isArray(items) ? items : []).find(item => item.name === name); }
function payloadOf(body, key) {
  const payload = body && body[key];
  if (!payload || typeof payload !== 'object') throw new Error(`${key} response is malformed`);
  return payload;
}
function exactID(value, label) {
  if (typeof value !== 'string' || !/^[1-9][0-9]*$/.test(value)) throw new Error(`${label} id is malformed`);
  return value;
}
function syntheticPhone(prefix) { return prefix + String(randomInt(0, 100000000)).padStart(8, '0'); }
function safeMessage(error) { return error && error.message ? String(error.message).replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : 'unknown error'; }
function validSettings(value) { return value && typeof value.notice === 'string' && typeof value.service_date === 'string' && Array.isArray(value.meal_periods) && Array.isArray(value.service_dates); }
function sameSettings(actual, expected) {
  return actual.store_status === expected.store_status &&
    actual.pickup_step_min === expected.pickup_step_min &&
    actual.pickup_point === expected.pickup_point &&
    actual.notice === expected.notice &&
    JSON.stringify(actual.meal_periods) === JSON.stringify(expected.meal_periods) &&
    JSON.stringify(actual.service_dates) === JSON.stringify(expected.service_dates);
}
function settingsWriteBody(settings) {
  return {
    store_status: settings.store_status,
    pickup_step_min: settings.pickup_step_min,
    pickup_point: settings.pickup_point,
    notice: settings.notice,
    meal_periods: settings.meal_periods,
    service_dates: settings.service_dates,
  };
}

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}

async function acquirePCSession(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-writes-${randomUUID()}` } }, 201);
  const bearer = exactToken(session.access_token, 'Mini session');
  await jsonRequest(origin, '/api/v1/me/bind-phone', { method: 'POST', bearer, idempotencyKey: randomUUID(), body: { code: `pc-writes-phone-${randomUUID()}` } }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  const approvalSecret = exactString(payload.searchParams.get('approval_secret'), 'approval_secret');
  const loginID = exactString(login.login_id, 'login_id');
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', { method: 'POST', bearer, body: { login_id: loginID, approval_secret: approvalSecret, code: `pc-writes-approve-${randomUUID()}` } }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll_secret') } }, 200);
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session');
}

async function adminGET(pathname) {
  return jsonRequest(apiOrigin, pathname, { method: 'GET', bearer: sessionToken }, 200);
}

async function adminWrite(pathname, method, body) {
  return jsonRequest(apiOrigin, pathname, { method, bearer: sessionToken, idempotencyKey: randomUUID(), body }, 200);
}

async function cleanupDELETE(pathname) {
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method: 'DELETE',
    headers: { Accept: 'application/json', Authorization: `Bearer ${sessionToken}`, 'Idempotency-Key': randomUUID() },
    redirect: 'error',
  });
  if (response.status !== 200 && response.status !== 404) throw new Error(`${pathname} cleanup returned ${response.status}`);
}

async function jsonRequest(origin, pathname, options, expectedStatus) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  const response = await fetch(`${origin}${pathname}`, {
    method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
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
      if (request.url.startsWith('/api/')) { await proxyAPI(request, response, upstreamOrigin); return; }
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
    method: request.method, headers, body: chunks.length ? Buffer.concat(chunks) : undefined, redirect: 'error',
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
