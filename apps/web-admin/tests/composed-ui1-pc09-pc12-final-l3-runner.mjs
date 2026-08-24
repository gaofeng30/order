import { randomInt, randomUUID } from 'node:crypto';
import { existsSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import {
  acquireOwnerSessions, browserPath, browserVersion, candidateSHA, chromium, evidenceRoot,
  json, provisionFreshRuntime, schema, startPrivateAPI, startProxy, stopAndCleanup, suffix,
} from './composed-ui1-pc09-pc12-final-l3-helper.mjs';

// PAGE-PC09 and PAGE-PC12 dedicated L3 gate. Canonical bootstrap is order-bootstrap;
// all ordinary facts enter through rendered PC UI or the private fresh-v44 root API.
const checks = [];
const names = {
  staff: `PC09员工-${suffix}`,
  imported: `PC12覆盖员工-${suffix}`,
  duplicate: `PC12重复员工-${suffix}`,
  category: `PC09分类-${suffix}`,
  product: `PC09菜品-${suffix}`,
};
const staffPhone = `166${String(randomInt(0, 100000000)).padStart(8, '0')}`;
let browser = null, context = null, page = null, proxy = null;
let miniToken = '', pcToken = '', staffID = '', productID = '', serviceDate = '';
let failure = null;

process.stdout.write(`PC09_PC12_FINAL_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, database: 'fresh-v44', api: 'random private loopback' })}\n`);

try {
  await provisionFreshRuntime();
  record('order-bootstrap creates canonical root facts in fresh-v44', true);
  await startPrivateAPI();
  ({ miniToken, pcToken } = await acquireOwnerSessions());
  proxy = await startProxy();
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext({ acceptDownloads: true });
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  page = await context.newPage();
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner');
  record('rendered PC authenticates the real fresh-schema OWNER session', true);

  await configureOrderFixture();
  await pagePC09();
  await pagePC12();
  record('required staff and import actions have unified audit receipts',
    await auditContains('SET_STAFF_ENABLED', staffID) && await auditContains('COMMIT_IMPORT'));
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safe(error)}`, false);
} finally {
  if (page) await page.screenshot({ path: path.join(evidenceRoot, 'final-state.png'), fullPage: true }).catch(() => {});
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
  const cleanup = await stopAndCleanup();
  record('private API stopped and fresh schema cleanup completed', cleanup.stopped && cleanup.dropped, cleanup);
}

const passed = !failure && checks.every(item => item.ok);
for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
const receipt = {
  schema: 'order.pc09-pc12-final-l3.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED', database_schema_version: 44,
  private_schema: schema.replace(/[a-f0-9]{10}$/, '[redacted]'),
  status: passed ? 'PASS' : 'FAIL', checks,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC09_PC12_FINAL_RESULT ${JSON.stringify({ status: receipt.status, candidate_sha: candidateSHA, checks: checks.length, receipt: path.join(evidenceRoot, 'receipt.json') })}\n`);
if (!passed) process.exitCode = 1;

async function configureOrderFixture() {
  const dates = serviceDates();
  serviceDate = dates.tomorrow;
  await json('/api/v1/admin/settings', {
    method: 'PUT', bearer: pcToken, key: randomUUID(), body: {
      store_status: 'open', pickup_point: 'PC09取餐点', notice: 'PC09累计统计验收', pickup_step_min: 30,
      meal_periods: [
        { code: 'lunch', name: '午餐', cutoff_time: '10:30', pickup_from: '11:30', pickup_to: '12:30' },
        { code: 'dinner', name: '晚餐', cutoff_time: '17:00', pickup_from: '17:30', pickup_to: '18:30' },
      ],
      service_dates: [{ date: dates.today, status: 'open' }, { date: dates.tomorrow, status: 'open' }],
    },
  }, 200);
  const category = payload(await json('/api/v1/admin/categories', { method: 'POST', bearer: pcToken, key: randomUUID(), body: { name: names.category } }, 201), 'category');
  const product = payload(await json('/api/v1/admin/products', {
    method: 'POST', bearer: pcToken, key: randomUUID(), body: {
      name: names.product, price_cents: 1880, category_id: String(category.id), meal_period: 'all', description: 'PC09统计事实', images: [],
    },
  }, 201), 'product');
  productID = String(product.id);
}

async function pagePC09() {
  await navigate('staff', '员工折扣白名单', '#staff-host');
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(names.staff);
  await page.locator('.drawer #f-phone').fill(staffPhone);
  const created = await mutate('POST', '/api/v1/admin/staff-whitelist', () => page.locator('.drawer [data-a="ok"]').click(), 201);
  staffID = String(created.body.id);
  await waitForText('#staff-host tr[data-id]', names.staff, true);

  let staff = findStaff(await staffList(), names.staff);
  const row = () => page.locator('#staff-host tr[data-id]').filter({ hasText: staffID }).filter({ hasText: names.staff });
  record('PAGE-PC09 phone/name dual-factor staff is visibly stored with masked PII',
    staff?.enabled === true && staff?.phone_masked === mask(staffPhone) && !(await row().innerText()).includes(staffPhone));

  await json('/api/v1/me/extra-phone', { method: 'POST', bearer: miniToken, key: randomUUID(), body: { phone: staffPhone, name: names.staff } }, 200);
  await page.locator('#f-rate').fill('80');
  await mutate('PUT', '/api/v1/admin/discount-rate', () => page.locator('[data-save-rate]').click(), 200);

  const priced = await quote('pc09-priced');
  record('PAGE-PC09 server pricing recognizes only the exact dual-factor enabled current fact',
    priced.identity?.kind === 'STAFF' && priced.discount?.rate_percent === 80 && priced.original_subtotal_cents === 1880
      && priced.discount_cents === 376 && priced.payable_cents === 1504, priced);
  await materialize(priced.id);

  await navigate('dashboard', '工作台', '#content');
  await navigate('staff', '员工折扣白名单', '#staff-host');
  staff = findStaff(await staffList(), names.staff);
  const statsRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staff });
  const visibleStats = await statsRow.innerText();
  await statsRow.locator('[data-act="edit"]').click();
  const drawerStats = await page.locator('.drawer').innerText();
  await page.locator('.drawer [data-a="cancel"]').click();
  record('PAGE-PC09 cumulative staff spend/order count is rendered from the same completed payment facts',
    staff?.spend_cents === 1504 && staff?.order_count === 1 && visibleStats.includes('¥15.04') && visibleStats.includes('1') && drawerStats.includes('15.04 元 / 1 单'));

  const toggleRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staff });
  await mutate('PUT', `/api/v1/admin/staff-whitelist/${staffID}`, () => toggleRow.locator('[data-act="toggle"]').click(), 200);
  await waitFor(async () => findStaff(await staffList(), names.staff)?.enabled === false);
  const disabledRow = page.locator('#staff-host tr[data-id]').filter({ hasText: names.staff });
  const afterDisable = await quote('pc09-disabled');
  staff = findStaff(await staffList(), names.staff);
  record('PAGE-PC09 disabled staff current-fact revalidation removes new discount without rewriting history',
    (await disabledRow.innerText()).includes('停用') && afterDisable.identity?.kind === 'VISITOR' && afterDisable.discount?.rate_percent === 100
      && afterDisable.discount_cents === 0 && afterDisable.payable_cents === 1880 && staff?.spend_cents === 1504 && staff?.order_count === 1);
}

async function pagePC12() {
  await navigate('staff-import', '员工批量导入', '#f-file');
  const downloadPromise = page.waitForEvent('download');
  await page.locator('[data-template]').click();
  const download = await downloadPromise;
  const suggested = download.suggestedFilename();
  const templatePath = await download.path();
  record('PAGE-PC12 rendered page downloads the canonical two-column xlsx template', suggested === '员工白名单批量导入模板.xlsx' && !!templatePath && existsSync(templatePath));

  const before = findStaff(await staffList(), names.staff);
  const importFile = xlsx([['姓名', '手机号'], [names.imported, staffPhone]]);
  const preview = await mutate('POST', '/api/v1/admin/staff-whitelist/import/preview', () => page.locator('#f-file').setInputFiles({
    name: `staff-overwrite-${suffix}.xlsx`, mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', buffer: importFile,
  }), 201);
  await page.locator('#imp-result [data-ok]').waitFor({ state: 'visible' });
  const previewText = await page.locator('#imp-result').innerText();
  record('PAGE-PC12 same-phone preview visibly declares one overwrite and no new staff row',
    preview.body.new_count === 0 && preview.body.update_count === 1 && preview.body.error_count === 0 && previewText.includes('更新 1 条'));
  await mutate('POST', '/api/v1/import/commit', () => page.locator('#imp-result [data-ok]').click(), 200);
  await page.waitForFunction(() => document.querySelector('#tb-title')?.textContent === '员工折扣白名单');
  await waitForText('#staff-host tr[data-id]', names.imported, true);

  const after = findStaff(await staffList(), names.imported);
  const rendered = await page.locator('#staff-host tr[data-id]').filter({ hasText: names.imported }).innerText();
  record('PAGE-PC12 same-phone import preserves disabled state and immutable cumulative facts',
    after?.id === before?.id && after?.enabled === false && after?.created_at === before?.created_at && after?.bound === before?.bound
      && after?.spend_cents === before?.spend_cents && after?.order_count === before?.order_count
      && rendered.includes('停用') && rendered.includes('¥15.04') && !rendered.includes(staffPhone));

  await navigate('staff-import', '员工批量导入', '#f-file');
  const duplicateFile = xlsx([['姓名', '手机号'], [names.duplicate, staffPhone], [`${names.duplicate}二`, staffPhone]]);
  const duplicate = await mutate('POST', '/api/v1/admin/staff-whitelist/import/preview', () => page.locator('#f-file').setInputFiles({
    name: `staff-duplicate-${suffix}.xlsx`, mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', buffer: duplicateFile,
  }), 201);
  await page.locator('#imp-result [data-ok]').waitFor({ state: 'visible' });
  const duplicateText = await page.locator('#imp-result').innerText();
  const errorRow = (duplicate.body.rows || []).find(item => item.outcome === 'ERROR');
  const unchanged = findStaff(await staffList(), names.imported);
  record('PAGE-PC12 duplicate phone is visibly abnormal before confirmation with zero staff write',
    duplicate.body.update_count === 1 && duplicate.body.error_count === 1 && errorRow?.reason?.includes('重复')
      && duplicateText.includes('异常 1 条') && duplicateText.includes('重复') && unchanged?.name === names.imported && unchanged?.enabled === false);
  record('PAGE-PC12 same-phone import preserves disabled state', after?.enabled === false && unchanged?.enabled === false);
}

async function quote(key) {
  const body = await json('/api/v1/quotes', { method: 'POST', bearer: miniToken, key: `${key}-${suffix}`, body: {
    contact_name: 'PC09验收用户', pickup_date: serviceDate, pickup_time: '11:30', order_note: '',
    items: [{ product_id: productID, quantity: 1, flavors: [], note: '' }],
  } }, 201);
  return payload(body, 'quote');
}

async function materialize(quoteID) {
  const prepared = await json('/api/v1/orders/prepay', { method: 'POST', bearer: miniToken, key: `pc09-prepay-${suffix}`, body: { quote_id: String(quoteID) } }, 201);
  const prepayment = payload(prepared, 'prepayment');
  const confirmed = await json('/api/v1/orders/confirm', { method: 'POST', bearer: miniToken, key: `pc09-confirm-${suffix}`, body: { prepayment_id: String(prepayment.id) } }, 200);
  if (confirmed.state !== 'ORDER_CREATED') throw new Error(`order materialization returned ${confirmed.state}`);
}

async function staffList() { return (await json('/api/v1/admin/staff-whitelist', { method: 'GET', bearer: pcToken }, 200)).staff || []; }

async function auditContains(action, targetID = '') {
  let afterID = '';
  for (let turn = 0; turn < 30; turn += 1) {
    const query = new URLSearchParams({ action, limit: '100', ...(afterID ? { after_id: afterID } : {}) });
    const body = await json(`/api/v1/admin/audits?${query}`, { method: 'GET', bearer: pcToken }, 200);
    if ((body.audits || []).some(item => item.action === action && (!targetID || String(item.target_id) === String(targetID)))) return true;
    if (!body.next_after_id) return false;
    afterID = String(body.next_after_id);
  }
  return false;
}

async function navigate(route, title, ready) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(ready).waitFor({ state: 'visible' });
}

async function mutate(method, pathname, action, expected) {
  const pending = page.waitForResponse(response => response.request().method() === method && new URL(response.url()).pathname === pathname);
  await action();
  const response = await pending;
  let body = null;
  try { body = await response.json(); } catch {}
  if (response.status() !== expected) throw new Error(`${method} ${pathname} returned ${response.status()}/${body?.error?.code || 'UNKNOWN'}`);
  return { body, key: response.request().headers()['idempotency-key'] || '' };
}

async function waitForText(selector, text, present) {
  await page.waitForFunction(({ selector, text, present }) => Array.from(document.querySelectorAll(selector)).some(node => (node.textContent || '').includes(text)) === present, { selector, text, present });
}

async function waitFor(predicate) {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    if (await predicate()) return;
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('server fact did not converge');
}

function xlsx(rows) {
  const xml = rows.map((row, y) => `<row r="${y + 1}">${row.map((value, x) => `<c r="${column(x)}${y + 1}" t="inlineStr"><is><t>${escapeXML(value)}</t></is></c>`).join('')}</row>`).join('');
  return zip([
    ['[Content_Types].xml', '<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/></Types>'],
    ['xl/workbook.xml', '<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheets><sheet name="Sheet1" sheetId="1"/></sheets></workbook>'],
    ['xl/worksheets/sheet1.xml', `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${xml}</sheetData></worksheet>`],
  ]);
}

function zip(files) {
  const locals = [], centrals = [];
  let offset = 0;
  for (const [filename, value] of files) {
    const filenameBytes = Buffer.from(filename), data = Buffer.from(value), checksum = crc32(data);
    const local = Buffer.alloc(30 + filenameBytes.length + data.length);
    local.writeUInt32LE(0x04034b50, 0); local.writeUInt16LE(20, 4); local.writeUInt32LE(checksum, 14);
    local.writeUInt32LE(data.length, 18); local.writeUInt32LE(data.length, 22); local.writeUInt16LE(filenameBytes.length, 26);
    filenameBytes.copy(local, 30); data.copy(local, 30 + filenameBytes.length); locals.push(local);
    const central = Buffer.alloc(46 + filenameBytes.length);
    central.writeUInt32LE(0x02014b50, 0); central.writeUInt16LE(20, 4); central.writeUInt16LE(20, 6);
    central.writeUInt32LE(checksum, 16); central.writeUInt32LE(data.length, 20); central.writeUInt32LE(data.length, 24);
    central.writeUInt16LE(filenameBytes.length, 28); central.writeUInt32LE(offset, 42); filenameBytes.copy(central, 46);
    centrals.push(central); offset += local.length;
  }
  const end = Buffer.alloc(22), centralSize = centrals.reduce((sum, item) => sum + item.length, 0);
  end.writeUInt32LE(0x06054b50, 0); end.writeUInt16LE(files.length, 8); end.writeUInt16LE(files.length, 10);
  end.writeUInt32LE(centralSize, 12); end.writeUInt32LE(offset, 16);
  return Buffer.concat([...locals, ...centrals, end]);
}

function crc32(data) { let value = 0xffffffff; for (const byte of data) { value ^= byte; for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ (value & 1 ? 0xedb88320 : 0); } return (value ^ 0xffffffff) >>> 0; }
function column(index) { let value = index + 1, out = ''; while (value) { value -= 1; out = String.fromCharCode(65 + value % 26) + out; value = Math.floor(value / 26); } return out; }
function escapeXML(value) { return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;'); }
function payload(body, key) { if (!body?.[key] || typeof body[key] !== 'object') throw new Error(`${key} missing`); return body[key]; }
function findStaff(items, name) { return items.find(item => item.name === name); }
function mask(phone) { return `${phone.slice(0, 3)}****${phone.slice(-4)}`; }
function serviceDates() { const today = shanghaiDate(new Date()); const tomorrow = shanghaiDate(new Date(Date.now() + 24 * 60 * 60 * 1000)); return { today, tomorrow }; }
function shanghaiDate(date) { const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(date); const get = type => parts.find(part => part.type === type).value; return `${get('year')}-${get('month')}-${get('day')}`; }
function record(name, ok, detail) { checks.push({ name, ok: Boolean(ok), ...(detail === undefined ? {} : { detail }) }); }
function safe(error) { return String(error?.message || error || 'unknown').replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]').slice(0, 2000); }
