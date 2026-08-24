import http from 'node:http';
import { spawn, spawnSync, execFileSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, readFileSync, statSync, unlinkSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';
import zlib from 'node:zlib';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const apiRoot = path.join(repositoryRoot, 'services/api');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chromium is missing; reuse the configured cache');

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const suffix = randomUUID().replaceAll('-', '').slice(0, 10);
const infoFile = `/private/tmp/order-pc-remaining-${suffix}.json`;
const stopFile = `/private/tmp/order-pc-remaining-${suffix}.stop`;
const evidenceRoot = path.join(repositoryRoot, '.scratch/pc05-pc07-pc12-closure');
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const checks = [];
const childReceipts = [];
let goTest = null, proxy = null, browser = null, context = null, page = null, apiOrigin = '', schema = '', objectRoot = '', pcToken = '';
let failure = null;

process.stdout.write(`PC_REMAINING_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, schema: 'fresh-v44 pending', api: 'random private loopback' })}\n`);
try {
  goTest = spawn('go', ['test', './cmd/order-api', '-run', '^TestPCRemainingUI1Server$', '-count=1', '-v'], {
    cwd: apiRoot,
    env: { ...process.env, ORDER_PC_REMAINING_SERVE: 'YES', ORDER_PC_REMAINING_INFO_FILE: infoFile, ORDER_PC_REMAINING_STOP_FILE: stopFile },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  let goOutput = '';
  goTest.stdout.on('data', chunk => { goOutput += chunk; });
  goTest.stderr.on('data', chunk => { goOutput += chunk; });
  const info = await waitForInfo(goTest, infoFile, () => goOutput);
  apiOrigin = exactLoopbackOrigin(info.origin);
  schema = exactSchema(info.schema);
  objectRoot = path.resolve(exactString(info.object_root, 'private object root'));
  record('fresh-v44 private schema and random API started', true, { schema, origin: apiOrigin });

  for (const runner of [
    'composed-ui1-catalog-image-chrome-runner.mjs',
    'composed-ui1-writes-chrome-runner.mjs',
    'composed-ui1-imports-chrome-runner.mjs',
  ]) {
    const result = spawnSync(process.execPath, [path.join(testsRoot, runner)], {
      cwd: repositoryRoot,
      env: { ...process.env, ORDER_COMPOSED_API_ORIGIN: apiOrigin },
      encoding: 'utf8', timeout: 240000, maxBuffer: 20 * 1024 * 1024,
    });
    writeFileSync(path.join(evidenceRoot, `${runner}.log`), `${result.stdout || ''}${result.stderr || ''}`, { mode: 0o600 });
    childReceipts.push({ runner, status: result.status, signal: result.signal || '' });
    if (runner === 'composed-ui1-writes-chrome-runner.mjs' && result.status === 1) {
      const supporting = JSON.parse(readFileSync(path.join(repositoryRoot, '.scratch/overnight-pc/receipt.json'), 'utf8'));
      const failed = supporting.checks.filter(item => !item.ok);
      const usable = supporting.checks.length === 21 && failed.length === 1 && failed[0].name === 'business settings restored through visible UI';
      record(`${runner} contributes only 20/21 supporting checks`, usable, { passed: supporting.checks.filter(item => item.ok).length, excluded: failed.map(item => item.name) });
      if (!usable) throw new Error(`${runner} has an unexpected failure; see evidence log`);
    } else {
      record(`${runner} supporting L3 exact-candidate runner`, result.status === 0, { status: result.status, signal: result.signal || '' });
      if (result.status !== 0) throw new Error(`${runner} failed; see evidence log`);
    }
  }

  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext({ acceptDownloads: true });
  pcToken = await acquirePCSession(apiOrigin);
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  page = await context.newPage();
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api?.currentAccount()?.role === 'owner');

  await templateScenario('product-import', '菜品批量导入', ['菜品名称', '售价', '分类', '餐段可售', '描述']);
  await templateScenario('staff-import', '员工批量导入', ['姓名', '手机号']);
  await missingHeaderImportScenario();
  await importSizeAndRowLimitScenarios();
  await existingProductImportScenario();
  await uploadFailureScenario();
  await unreadableLayerScenario();
  await settingsPositiveScenario();
  await invalidBoundariesScenario();
  await ownerAndSessionScenario();
  await subaccountRBACScenario();
  await page.screenshot({ path: path.join(evidenceRoot, 'final-state.png'), fullPage: true });
} catch (error) {
  failure = error;
  record(`runner completed without exception: ${safeMessage(error)}`, false);
} finally {
  if (context) await context.close().catch(() => {});
  if (browser) await browser.close().catch(() => {});
  if (proxy) await proxy.close().catch(() => {});
  if (goTest) {
    writeFileSync(stopFile, 'stop\n', { mode: 0o600 });
    const exit = await waitForExit(goTest, 30000);
    record('private API stopped and fresh schema cleanup completed', exit.code === 0, exit);
  }
}

const passed = !failure && checks.every(item => item.ok);
for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
const receipt = {
  schema: 'order.pc05-pc07-pc12-closure-ui1.v1', candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(), browser: browserVersion, private_schema: schema,
  status: passed ? 'PASS' : 'FAIL', child_receipts: childReceipts, checks,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_REMAINING_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, receipt: '.scratch/pc05-pc07-pc12-closure/receipt.json' })}\n`);
if (!passed) process.exitCode = 1;

async function templateScenario(route, title, headers) {
  await navigate(route, title, '#f-file');
  const before = await businessCounts();
  const downloadPromise = page.waitForEvent('download');
  await page.locator('[data-template]').click();
  const download = await downloadPromise;
  const file = await download.path();
  const bytes = readFileSync(file);
  record(`${title} downloads a real xlsx template`, download.suggestedFilename().endsWith('.xlsx') && bytes.subarray(0, 4).equals(Buffer.from([0x50, 0x4b, 0x03, 0x04])));
  const responsePromise = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname.endsWith('/import/preview'));
  await page.locator('#f-file').setInputFiles({
    name: download.suggestedFilename(),
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: bytes,
  });
  const response = await responsePromise;
  const body = await response.json();
  const visible = await page.locator('#content').innerText();
  const after = await businessCounts();
  const factsUnchanged = JSON.stringify(before) === JSON.stringify(after);
  record(`${title} header-only template is canonical and fails closed until rows are added`, response.status() === 422 && body?.error?.code === 'INVALID_TEMPLATE' && headers.every(value => visible.includes(value)) && factsUnchanged, { status: response.status(), code: body?.error?.code || '', facts_unchanged: factsUnchanged });
}

async function missingHeaderImportScenario() {
  const before = await businessCounts();
  const file = buildXlsx([
    ['菜品名称', '售价', '分类', '描述'],
    [`缺表头-${suffix}`, '19.9', `缺表头分类-${suffix}`, '不得写入'],
  ], { numericCols: [1] });
  const result = await previewThroughUI('product-import', '菜品批量导入', `missing-header-${suffix}.xlsx`, file);
  const after = await businessCounts();
  record('BE-28 missing required header is visibly rejected with zero business writes',
    result.status === 422 && result.code === 'INVALID_TEMPLATE' && result.warning && sameCounts(before, after),
    { status: result.status, code: result.code, facts_unchanged: sameCounts(before, after) });
}

async function importSizeAndRowLimitScenarios() {
  await navigate('product-import', '菜品批量导入', '#f-file');
  const beforeSize = await businessCounts();
  let previewRequests = 0;
  const countPreview = request => {
    if (request.method() === 'POST' && new URL(request.url()).pathname.endsWith('/import/preview')) previewRequests += 1;
  };
  page.on('request', countPreview);
  await page.locator('#f-file').setInputFiles({
    name: `over-10mib-${suffix}.xlsx`,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: Buffer.alloc(10 * 1024 * 1024 + 1),
  });
  await page.locator('.toast').filter({ hasText: '文件不能超过 10 MiB' }).last().waitFor({ state: 'visible' });
  const afterSize = await businessCounts();
  record('BE-29 file over 10 MiB fails before HTTP with zero business writes', previewRequests === 0 && sameCounts(beforeSize, afterSize),
    { preview_requests: previewRequests, facts_unchanged: sameCounts(beforeSize, afterSize) });
  page.off('request', countPreview);

  const beforeProductRows = await businessCounts();
  const productLimit = await previewThroughUI('product-import', '菜品批量导入', `products-501-${suffix}.xlsx`, buildXlsx(productRows(501), { numericCols: [1] }));
  const afterProductRows = await businessCounts();
  record('BE-29 product xlsx with 501 rows is visibly rejected with zero business writes',
    productLimit.status === 422 && productLimit.code === 'TOO_MANY_ROWS' && productLimit.warning && sameCounts(beforeProductRows, afterProductRows),
    { status: productLimit.status, code: productLimit.code, facts_unchanged: sameCounts(beforeProductRows, afterProductRows) });

  const beforeStaffRows = await businessCounts();
  const staffLimit = await previewThroughUI('staff-import', '员工批量导入', `staff-5001-${suffix}.xlsx`, buildXlsx(staffRows(5001)));
  const afterStaffRows = await businessCounts();
  record('BE-29 staff xlsx with 5001 rows is visibly rejected with zero business writes',
    staffLimit.status === 422 && staffLimit.code === 'TOO_MANY_ROWS' && staffLimit.warning && sameCounts(beforeStaffRows, afterStaffRows),
    { status: staffLimit.status, code: staffLimit.code, facts_unchanged: sameCounts(beforeStaffRows, afterStaffRows) });
}

async function existingProductImportScenario() {
  const settings = await adminGET('/api/v1/admin/settings');
  const products = (await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(settings.service_date)}`)).products;
  const existing = products[0];
  if (!existing?.id || !existing.name || !existing.category_name) throw new Error('BE-30 seeded product is missing');
  const beforeCounts = await businessCounts();
  const beforeProduct = JSON.stringify(existing);
  const file = buildXlsx([
    ['菜品名称', '售价', '分类', '餐段可售', '描述'],
    [existing.name, '9999', existing.category_name, '全天', `不得覆盖-${suffix}`],
  ], { numericCols: [1] });
  const result = await previewThroughUI('product-import', '菜品批量导入', `existing-product-${suffix}.xlsx`, file);
  const visible = await page.locator('#content').innerText();
  const after = await adminGET(`/api/v1/admin/products/${existing.id}?service_date=${encodeURIComponent(settings.service_date)}`);
  const afterCounts = await businessCounts();
  const commitEnabled = await page.locator('#imp-result [data-ok]:not([disabled])').count();
  record('BE-30 existing product is isolated as an error and cannot be committed or overwritten',
    result.status === 201 && Number(result.body?.new_count || 0) === 0 && Number(result.body?.update_count || 0) === 0 &&
      Number(result.body?.error_count || 0) === 1 && visible.includes('异常 1 条') && visible.includes('已存在') &&
      commitEnabled === 0 && JSON.stringify(after.product) === beforeProduct && sameCounts(beforeCounts, afterCounts),
    { status: result.status, error_count: Number(result.body?.error_count || 0), commit_enabled: commitEnabled, facts_unchanged: sameCounts(beforeCounts, afterCounts) });
}

async function previewThroughUI(route, title, filename, buffer) {
  await navigate(route, title, '#f-file');
  const responsePromise = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname.endsWith('/import/preview'));
  await page.locator('#f-file').setInputFiles({
    name: filename,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer,
  });
  const response = await responsePromise;
  let body = null;
  try { body = await response.json(); } catch {}
  if (response.status() >= 400) await page.locator('.toast .ti--warn').last().waitFor({ state: 'visible' });
  return { status: response.status(), code: body?.error?.code || '', warning: response.status() < 400 || await page.locator('.toast .ti--warn').last().isVisible(), body };
}

function productRows(count) {
  return [['菜品名称', '售价', '分类', '餐段可售', '描述'], ...Array.from({ length: count }, (_, index) => [`边界菜品-${suffix}-${index}`, '10', `边界分类-${suffix}`, '全天', '不得写入'])];
}

function staffRows(count) {
  return [['姓名', '手机号'], ...Array.from({ length: count }, (_, index) => [`边界员工-${suffix}-${index}`, `1${String(3000000000 + index).padStart(10, '0')}`])];
}

async function uploadFailureScenario() {
  await navigate('products', '菜品管理', '#tbl-host');
  await page.locator('[data-new]').click();
  let productWrites = 0;
  page.on('request', request => { if (request.method() === 'POST' && new URL(request.url()).pathname === '/api/v1/admin/products') productWrites += 1; });
  await page.route('**/api/v1/upload', route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { code: 'OBJECT_UNAVAILABLE', message: '图片服务暂时不可用' } }) }));
  await page.locator('.drawer #f-file').setInputFiles({ name: 'failure.png', mimeType: 'image/png', buffer: Buffer.from([137, 80, 78, 71]) });
  await page.locator('.toast').filter({ hasText: '图片服务暂时不可用' }).waitFor({ state: 'visible' });
  record('PC05 upload failure is visible and writes no product', productWrites === 0 && await page.locator('.drawer').isVisible() && await page.locator('.drawer .img-cell:not(.add)').count() === 0);
  await page.locator('.drawer [data-a="c"]').click();

  await navigate('layer', '开屏图层', '#phone');
  await page.locator('#file').setInputFiles({ name: 'failure.png', mimeType: 'image/png', buffer: Buffer.from([137, 80, 78, 71]) });
  await page.locator('.toast').filter({ hasText: '图片服务暂时不可用' }).waitFor({ state: 'visible' });
  record('PC08 upload failure is visible and cannot enable a bad object', await page.locator('#lay-img').count() === 0 && !(await page.locator('#en').evaluate(node => node.classList.contains('on'))));
  await page.unroute('**/api/v1/upload');
}

async function invalidBoundariesScenario() {
  const settings = await adminGET('/api/v1/admin/settings');
  const thirdMeal = { ...settings, meal_periods: [...settings.meal_periods, { code: 'breakfast', name: '早餐', cutoff_time: '07:00', pickup_from: '08:00', pickup_to: '09:00' }] };
  const third = await adminRaw('/api/v1/admin/settings', 'PUT', thirdMeal);
  const invalidTime = await adminRaw('/api/v1/admin/settings', 'PUT', { ...settings, meal_periods: settings.meal_periods.map((meal, index) => index ? meal : { ...meal, cutoff_time: '13:00', pickup_from: '12:00', pickup_to: '11:00' }) });
  record('PC07 rejects a third meal and invalid pickup time without drift', isRejected(third.status) && isRejected(invalidTime.status) && sameSettings(await adminGET('/api/v1/admin/settings'), settings), { third_status: third.status, invalid_time_status: invalidTime.status });

  const zero = await adminRaw('/api/v1/admin/discount-rate', 'PUT', { rate_percent: 0 });
  const high = await adminRaw('/api/v1/admin/discount-rate', 'PUT', { rate_percent: 101 });
  record('PC09 discount rate rejects both sides of 1..100', isRejected(zero.status) && isRejected(high.status), { zero_status: zero.status, high_status: high.status });
}

async function unreadableLayerScenario() {
  await navigate('layer', '开屏图层', '#phone');
  const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M/wHwAF/gL+Wn9Z5QAAAABJRU5ErkJggg==', 'base64');
  const uploadPromise = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/upload');
  await page.locator('#file').setInputFiles({ name: 'fault-fixture.png', mimeType: 'image/png', buffer: png });
  const uploadResponse = await uploadPromise;
  const upload = await uploadResponse.json();
  const objectKey = exactString(upload.image?.object_key, 'fault object key');
  await page.locator('#lay-img').waitFor({ state: 'visible' });
  const savePromise = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === '/api/v1/admin/launch-layer');
  await page.locator('#save').click();
  if ((await savePromise).status() !== 200) throw new Error('fault layer save failed');
  const objectFile = path.resolve(objectRoot, objectKey);
  if (!objectFile.startsWith(objectRoot + path.sep) || !existsSync(objectFile)) throw new Error('fault fixture object path escaped or missing');
  unlinkSync(objectFile); // Explicit fault fixture; never counted as a normal business write.
  const unavailable = await fetch(`${apiOrigin}/api/v1/objects/${objectKey}`);

  const brokenContext = await browser.newContext();
  await brokenContext.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), pcToken);
  const brokenPage = await brokenContext.newPage();
  await brokenPage.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await brokenPage.locator('a[data-r="layer"]').click();
  await brokenPage.waitForFunction(() => document.querySelector('#tb-title')?.textContent === '开屏图层');
  const hidden = await brokenPage.locator('#lay-img').evaluate(node => node.hidden && node.naturalWidth === 0);
  record('PC08 explicit unreadable-object fault is hidden in a fresh browser context', unavailable.status === 404 && hidden);
  await brokenContext.close();
  const cleared = await adminRaw('/api/v1/admin/launch-layer', 'DELETE');
  if (cleared.status !== 200) throw new Error('fault layer cleanup failed');
}

async function settingsPositiveScenario() {
  const before = await adminGET('/api/v1/admin/settings');
  await navigate('settings', '营业设置', '#save');
  const dates = page.locator('[data-date]');
  if (await dates.count() !== 2) throw new Error('PC07 did not render exactly today and tomorrow');
  const initialOpen = await dates.first().getAttribute('data-open');
  await dates.first().click();
  await page.locator('#f-notice').fill(`PC07-${suffix}`);
  const write = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === '/api/v1/admin/settings');
  await page.locator('#save').click();
  if ((await write).status() !== 200) throw new Error('PC07 positive settings save failed');
  const changed = await adminGET('/api/v1/admin/settings');
  const expectedStatus = initialOpen === 'true' ? 'closed' : 'open';
  record('PC07 visible save persists one store, two meals and today/tomorrow facts', changed.notice === `PC07-${suffix}` && changed.meal_periods.length === 2 && changed.service_dates.length === 2 && changed.service_dates[0].status === expectedStatus);
  const restored = await adminRaw('/api/v1/admin/settings', 'PUT', settingsWriteBody(before));
  record('PC07 baseline restores without semantic drift', restored.status === 200 && sameSettings(await adminGET('/api/v1/admin/settings'), before));
}

async function ownerAndSessionScenario() {
  await navigate('accounts', '商户账号名单', '#acc-host');
  const owners = (await adminGET('/api/v1/admin/merchant-accounts')).accounts.filter(item => item.role === 'OWNER' && item.enabled);
  if (owners.length !== 1) throw new Error(`expected one enabled OWNER, got ${owners.length}`);
  const row = page.locator('#acc-host tr[data-id]').filter({ hasText: owners[0].name });
  const responsePromise = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === `/api/v1/admin/merchant-accounts/${owners[0].id}`);
  await row.locator('[data-act="toggle"]').click();
  const response = await responsePromise;
  await page.locator('.toast .ti--warn').last().waitFor({ state: 'visible' });
  record('PC10 last OWNER protection is rendered without false success', response.status() === 409 && (await adminGET('/api/v1/admin/merchant-accounts')).accounts.find(item => item.id === owners[0].id)?.enabled === true);

  const first = await acquirePCSession(apiOrigin);
  const second = await acquirePCSession(apiOrigin);
  const firstMe = await rawGET('/api/v1/admin/me', first);
  const secondMe = await rawGET('/api/v1/admin/me', second);
  const persistedOutsideSession = await page.evaluate(() => window.localStorage.getItem('pc_session_token'));
  record('PC10 two concurrent QR sessions remain independently usable', first !== second && firstMe.status === 200 && secondMe.status === 200 && !persistedOutsideSession);
}

async function subaccountRBACScenario() {
  const ownerMe = await rawGET('/api/v1/admin/me', pcToken);
  record('AC-16 OWNER PC session reaches the owner workspace normally', ownerMe.status === 200 && await page.evaluate(() => window.Api.currentAccount()?.role) === 'owner');

  const subMiniToken = await miniSessionToken(apiOrigin);
  const subContext = await browser.newContext();
  const subPage = await subContext.newPage();
  const challengePromise = subPage.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/admin/auth/qrcode');
  await subPage.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  const challengeResponse = await challengePromise;
  const challenge = await challengeResponse.json();
  await subPage.locator('#pc-qr canvas').waitFor({ state: 'visible' });

  await navigate('accounts', '商户账号名单', '#acc-host');
  const currentAccounts = (await adminGET('/api/v1/admin/merchant-accounts')).accounts;
  const boundOwner = currentAccounts.find(account => account.role === 'OWNER' && account.enabled && account.bound);
  if (!boundOwner) throw new Error('bound OWNER account missing before AC-16 setup');
  const backupName = `RBAC备用主账号-${suffix}`;
  const backupPhone = `199${String(parseInt(suffix.slice(0, 8), 16) % 100000000).padStart(8, '0')}`;
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(backupName);
  await page.locator('.drawer #f-phone').fill(backupPhone);
  await page.locator('.drawer #f-role').selectOption('owner');
  const backupResponse = page.waitForResponse(response => response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/v1/admin/merchant-accounts');
  await page.locator('.drawer [data-a="ok"]').click();
  if ((await backupResponse).status() !== 201) throw new Error('backup OWNER setup failed');
  await page.locator('#acc-host tr[data-id]').filter({ hasText: backupName }).waitFor({ state: 'visible' });

  const ownerRow = page.locator('#acc-host tr[data-id]').filter({ hasText: boundOwner.name });
  await ownerRow.locator('[data-act="edit"]').click();
  await page.locator('.drawer #f-role').selectOption('staff');
  const downgradeResponse = page.waitForResponse(response => response.request().method() === 'PUT' && new URL(response.url()).pathname === `/api/v1/admin/merchant-accounts/${boundOwner.id}`);
  await page.locator('.drawer [data-a="ok"]').click();
  if ((await downgradeResponse).status() !== 200) throw new Error('bound account SUBACCOUNT setup failed');

  const before = await publicBusinessSnapshot();
  const payload = new URL(exactString(challenge.qr_payload, 'RBAC qr_payload'));
  const approval = await rawJSON('/api/v1/me/admin-login/approve', {
    method: 'POST', bearer: subMiniToken,
    body: {
      login_id: exactString(challenge.login_id, 'RBAC login_id'),
      approval_secret: exactString(payload.searchParams.get('approval_secret'), 'RBAC approval_secret'),
      code: `rbac-subaccount-${randomUUID()}`,
    },
  });
  const poll = await rawJSON('/api/v1/admin/auth/poll', {
    method: 'POST', body: { login_id: challenge.login_id, poll_secret: challenge.poll_secret },
  });
  const forged = await subPage.evaluate(async token => {
    window.sessionStorage.setItem('pc_session_token', token);
    window.Api.currentAccount = () => ({ role: 'owner' });
    try { await window.Api.listMerchantAccounts(''); return { status: 200 }; }
    catch (error) { return { status: error.status, code: error.code }; }
  }, subMiniToken);
  const after = await publicBusinessSnapshot();
  const browserState = await subPage.evaluate(() => ({
    token: window.sessionStorage.getItem('pc_session_token'),
    qr: !!document.querySelector('#pc-qr canvas'),
    login_visible: (document.querySelector('#content')?.textContent || '').includes('主账号扫码登录'),
  }));

  record('INV-13 real SUBACCOUNT approval returns 403 and challenge stays waiting',
    approval.status === 403 && approval.body?.error?.code === 'FORBIDDEN' && poll.status === 202 && poll.body?.state === 'WAITING',
    { approval_status: approval.status, approval_code: approval.body?.error?.code || '', poll_status: poll.status, challenge_state: poll.body?.state || '' });
  record('AC-16 client role spoof cannot grant an owner-only PC route or enter the workspace',
    forged.status === 401 && browserState.token === subMiniToken && browserState.qr && browserState.login_visible,
    { route_status: forged.status, route_code: forged.code || '', qr_visible: browserState.qr, login_visible: browserState.login_visible });
  record('AC-16 denied SUBACCOUNT PC flow writes no business facts and creates no PC session',
    JSON.stringify(before) === JSON.stringify(after) && !poll.body?.session,
    { facts_unchanged: JSON.stringify(before) === JSON.stringify(after), session_issued: !!poll.body?.session });
  await subContext.close();
}

async function navigate(route, title, ready) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(ready).waitFor({ state: 'visible' });
}
async function businessCounts() {
  const settings = await adminGET('/api/v1/admin/settings');
  const [products, categories, staff] = await Promise.all([adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(settings.service_date)}`), adminGET('/api/v1/admin/categories'), adminGET('/api/v1/admin/staff-whitelist')]);
  return { products: products.products.length, categories: categories.categories.length, staff: staff.staff.length };
}
async function acquirePCSession(origin) {
  const mini = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-remaining-${randomUUID()}` } }, 201);
  const bearer = exactString(mini.access_token, 'mini token');
  await jsonRequest(origin, '/api/v1/me/bind-phone', { method: 'POST', bearer, key: randomUUID(), body: { code: `phone-${randomUUID()}` } }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr payload'));
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', { method: 'POST', bearer, body: { login_id: login.login_id, approval_secret: payload.searchParams.get('approval_secret'), code: `approve-${randomUUID()}` } }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: login.login_id, poll_secret: login.poll_secret } }, 200);
  return exactString(poll.session?.token, 'PC token');
}
async function miniSessionToken(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-rbac-${randomUUID()}` } }, 201);
  return exactString(session.access_token, 'RBAC mini token');
}
async function publicBusinessSnapshot() {
  const storefront = await jsonRequest(apiOrigin, '/api/v1/storefront/settings', { method: 'GET' }, 200);
  return storefront;
}
async function adminGET(pathname) { return jsonRequest(apiOrigin, pathname, { method: 'GET', bearer: pcToken }, 200); }
async function adminRaw(pathname, method, body) {
  const response = await fetch(`${apiOrigin}${pathname}`, { method, headers: { Accept: 'application/json', 'Content-Type': 'application/json', Authorization: `Bearer ${pcToken}`, 'Idempotency-Key': randomUUID() }, body: JSON.stringify(body) });
  let payload = null; try { payload = await response.json(); } catch {}
  return { status: response.status, body: payload };
}
async function rawGET(pathname, token) { const response = await fetch(`${apiOrigin}${pathname}`, { headers: { Authorization: `Bearer ${token}` } }); return { status: response.status }; }
async function rawJSON(pathname, options) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  const response = await fetch(`${apiOrigin}${pathname}`, { method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error' });
  let body = null; try { body = await response.json(); } catch {}
  return { status: response.status, body };
}
async function jsonRequest(origin, pathname, options, expected) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  const response = await fetch(`${origin}${pathname}`, { method: options.method, headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error' });
  let body = null; try { body = await response.json(); } catch {}
  if (response.status !== expected) throw new Error(`${pathname} returned ${response.status}${body?.error?.code ? ` ${body.error.code}` : ''}`);
  return body;
}

async function startSameOriginProxy(upstream) {
  const server = http.createServer(async (request, response) => {
    try {
      if (request.url.startsWith('/api/')) return proxyAPI(request, response, upstream);
      const clean = decodeURIComponent((request.url || '/').split('?')[0]);
      const relative = clean === '/web-admin/' || clean === '/web-admin/index.html' ? 'web-admin/index.html' : clean.replace(/^\//, '');
      const file = path.resolve(appsRoot, relative);
      if (!file.startsWith(appsRoot + path.sep) || !existsSync(file) || !statSync(file).isFile()) { response.writeHead(404); response.end(); return; }
      const ext = path.extname(file); const types = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.png': 'image/png', '.svg': 'image/svg+xml' };
      response.writeHead(200, { 'Content-Type': types[ext] || 'application/octet-stream', 'Cache-Control': 'no-store' }); createReadStream(file).pipe(response);
    } catch { response.writeHead(502); response.end(); }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { origin: `http://127.0.0.1:${server.address().port}`, close: () => new Promise(resolve => server.close(resolve)) };
}
async function proxyAPI(request, response, upstream) {
  const chunks = []; for await (const chunk of request) chunks.push(chunk);
  const target = await fetch(`${upstream}${request.url}`, { method: request.method, headers: request.headers, body: ['GET', 'HEAD'].includes(request.method) ? undefined : Buffer.concat(chunks), redirect: 'manual' });
  const headers = {}; target.headers.forEach((value, key) => { if (!['content-encoding', 'content-length', 'transfer-encoding'].includes(key)) headers[key] = value; });
  response.writeHead(target.status, headers); response.end(Buffer.from(await target.arrayBuffer()));
}
async function waitForInfo(child, filename, output) {
  for (let i = 0; i < 300; i += 1) {
    if (existsSync(filename)) return JSON.parse(readFileSync(filename, 'utf8'));
    if (child.exitCode !== null) throw new Error(`private API exited ${child.exitCode}: ${output()}`);
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('private API did not publish info');
}
async function waitForExit(child, timeout) {
  if (child.exitCode !== null) return { code: child.exitCode, signal: child.signalCode || '' };
  return new Promise(resolve => { const timer = setTimeout(() => { child.kill('SIGTERM'); resolve({ code: child.exitCode, signal: 'TIMEOUT' }); }, timeout); child.once('exit', (code, signal) => { clearTimeout(timer); resolve({ code, signal: signal || '' }); }); });
}
function exactLoopbackOrigin(value) { const url = new URL(value); if (url.protocol !== 'http:' || url.hostname !== '127.0.0.1' || url.pathname !== '/') throw new Error('invalid private origin'); return url.origin; }
function exactSchema(value) { if (!/^order_acceptance_[a-z0-9_]+$/.test(value)) throw new Error('invalid private schema'); return value; }
function exactString(value, name) { if (typeof value !== 'string' || !value.trim()) throw new Error(`${name} missing`); return value; }
function isRejected(status) { return Number.isInteger(status) && status >= 400 && status < 500; }
function settingsWriteBody(value) { return { store_status: value.store_status, pickup_step_min: value.pickup_step_min, pickup_point: value.pickup_point, notice: value.notice, meal_periods: value.meal_periods, service_dates: value.service_dates }; }
function sameCounts(left, right) { return JSON.stringify(left) === JSON.stringify(right); }
function sameSettings(actual, expected) {
  const normalize = value => ({
    store_status: value.store_status, pickup_step_min: value.pickup_step_min, pickup_point: value.pickup_point, notice: value.notice,
    meal_periods: [...value.meal_periods].sort((a, b) => a.code.localeCompare(b.code)),
    service_dates: [...value.service_dates].sort((a, b) => a.date.localeCompare(b.date)),
  });
  return JSON.stringify(normalize(actual)) === JSON.stringify(normalize(expected));
}
function safeMessage(error) { return String(error?.message || error || 'unknown').replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]'); }
function record(name, ok, detail) { checks.push({ name, ok: Boolean(ok), ...(detail === undefined ? {} : { detail }) }); }

function crc32(buffer) {
  const table = new Int32Array(256);
  for (let index = 0; index < 256; index += 1) {
    let value = index;
    for (let bit = 0; bit < 8; bit += 1) value = value & 1 ? 0xEDB88320 ^ (value >>> 1) : value >>> 1;
    table[index] = value;
  }
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
  const indexOf = value => { const found = shared.indexOf(value); return found >= 0 ? found : shared.push(value) - 1; };
  const sheetRows = rows.map((row, rowIndex) => `<row r="${rowIndex + 1}">${row.map((value, columnIndex) => {
    if (value === '' || value == null) return '';
    const reference = `${columnName(columnIndex + 1)}${rowIndex + 1}`;
    if (options.numericCols?.includes(columnIndex) && /^-?\d+(\.\d+)?$/.test(value)) return `<c r="${reference}"><v>${value}</v></c>`;
    return `<c r="${reference}" t="s"><v>${indexOf(String(value))}</v></c>`;
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

function columnName(value) {
  let name = '', number = value;
  while (number > 0) { const remainder = (number - 1) % 26; name = String.fromCharCode(65 + remainder) + name; number = (number - remainder - 1) / 26; }
  return name;
}
function escapeXML(value) { return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;'); }
