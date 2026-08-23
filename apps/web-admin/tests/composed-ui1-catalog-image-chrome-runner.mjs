import http from 'node:http';
import { execFileSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, statSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash, randomUUID } from 'node:crypto';
import { deflateSync } from 'node:zlib';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const apiOrigin = exactLoopbackOrigin(process.env.ORDER_COMPOSED_API_ORIGIN || 'http://127.0.0.1:8080');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || path.join(repositoryRoot, 'tools/miniprogram-ui');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const { chromium } = dependencyRequire('playwright');
const browserPath = process.env.CHROME_BIN || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
if (!existsSync(browserPath)) throw new Error('Chrome is missing; reuse the configured local browser');

const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
const evidenceRoot = path.join(repositoryRoot, '.scratch/overnight-pc-catalog-image');
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });
const suffix = randomUUID().replaceAll('-', '').slice(0, 8);
const names = {
  categoryA: `UI1分类A-${suffix}`,
  categoryB: `UI1分类B-${suffix}`,
  categoryBRenamed: `UI1分类B改-${suffix}`,
  product0: `UI1零图-${suffix}`,
  product1: `UI1一图-${suffix}`,
  product3: `UI1三图-${suffix}`,
};
const imageFiles = [
  writePNG('red.png', 220, 45, 45, 255),
  writePNG('green.png', 45, 180, 75, 230),
  writePNG('blue.png', 45, 90, 220, 200),
];
const imageKeys = imageFiles.map(file => `images/${createHash('sha256').update(file.bytes).digest('hex')}.png`);
const checks = [];
const cleanup = [];
const resources = { categoryIDs: [], productIDs: [] };
let sessionToken = '';
let proxy;
let browser;
let context;
let page;
let initialLayer;
let layerDirty = false;
let failure;

process.stdout.write(`PC_CATALOG_IMAGE_UI1_ENV ${JSON.stringify({ candidate_sha: candidateSHA, browser: browserVersion, upstream: apiOrigin })}\n`);

try {
  sessionToken = await acquirePCSession(apiOrigin);
  proxy = await startSameOriginProxy(apiOrigin);
  browser = await chromium.launch({ executablePath: browserPath, headless: true });
  context = await browser.newContext();
  await context.addInitScript(token => window.sessionStorage.setItem('pc_session_token', token), sessionToken);
  page = await context.newPage();
  await page.goto(`${proxy.origin}/web-admin/index.html`, { waitUntil: 'networkidle' });
  await page.waitForFunction(() => window.Api && window.Api.currentAccount()?.role === 'owner');
  record('authenticated real OWNER PC session', true);
  initialLayer = await adminGET('/api/v1/admin/launch-layer');
  await catalogScenario();
  await layerScenario();
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

for (const item of checks) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - ${item.name}\n`);
for (const item of cleanup) process.stdout.write(`${item.ok ? 'ok' : 'not ok'} - cleanup ${item.name}${item.error ? `: ${item.error}` : ''}\n`);
const passed = !failure && checks.every(item => item.ok) && cleanup.every(item => item.ok);
const receipt = {
  schema: 'order.pc-catalog-image-ui1.v1', candidate_sha: candidateSHA, generated_at: new Date().toISOString(),
  browser: browserVersion, upstream: apiOrigin, test_prefix: names.categoryA,
  status: passed ? 'PASS' : 'FAIL', checks, cleanup,
};
writeFileSync(path.join(evidenceRoot, 'receipt.json'), `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
process.stdout.write(`PC_CATALOG_IMAGE_UI1_RESULT ${JSON.stringify({ status: receipt.status, checks: checks.length, cleanup: cleanup.length, receipt: '.scratch/overnight-pc-catalog-image/receipt.json' })}\n`);
if (!passed) process.exitCode = 1;

async function catalogScenario() {
  const settings = await adminGET('/api/v1/admin/settings');
  const today = exactString(settings.service_date, 'service_date');
  const tomorrow = addISODate(today, 1);

  await navigate('categories', '分类管理', '#cat-host');
  const categoryA = await createCategory(names.categoryA);
  const categoryB = await createCategory(names.categoryB);
  resources.categoryIDs.push(categoryA.id, categoryB.id);
  record('two categories are created through visible UI and read from server',
    categoryA.name === names.categoryA && categoryB.name === names.categoryB);

  let rowA = categoryRow(names.categoryA);
  record('category row exposes edit and PRD up/down controls',
    await rowA.locator('[data-act="cat-edit"]').count() === 1 &&
    await rowA.locator('[data-act="cat-up"]').count() === 1 &&
    await rowA.locator('[data-act="cat-down"]').count() === 1);

  await page.locator('[data-new]').click();
  await page.locator('.modal #c-name').fill(`  ${names.categoryA}  `);
  const duplicate = await uiMutation('POST', '/api/v1/admin/categories', 409, () => page.locator('.modal [data-a="ok"]').click());
  const duplicateRows = await page.locator('.cat-row').filter({ hasText: names.categoryA }).count();
  record('normalized duplicate category key is rejected without false-success',
    duplicate.body?.error?.code === 'CATALOG_CONFLICT' && duplicateRows === 1 && await page.locator('.modal').isVisible() && await page.locator('.toast .ti--warn').count() > 0);
  await page.locator('.modal [data-a="c"]').click();

  let rowB = categoryRow(names.categoryB);
  await rowB.locator('[data-act="cat-edit"]').click();
  await page.locator('.modal #c-name').fill(names.categoryBRenamed);
  await uiMutation('PUT', `/api/v1/admin/categories/${categoryB.id}`, 200, () => page.locator('.modal [data-a="ok"]').click());
  await waitForText('.cat-row', names.categoryBRenamed, true);
  let categories = (await adminGET('/api/v1/admin/categories')).categories;
  record('category rename is visible and persisted', findNamed(categories, names.categoryBRenamed)?.id === categoryB.id);

  rowB = categoryRow(names.categoryBRenamed);
  await uiMutation('PUT', `/api/v1/admin/categories/${categoryB.id}`, 200, () => rowB.locator('[data-on]').click());
  await waitForServer(async () => findNamed((await adminGET('/api/v1/admin/categories')).categories, names.categoryBRenamed)?.enabled === false);
  rowB = categoryRow(names.categoryBRenamed);
  await uiMutation('PUT', `/api/v1/admin/categories/${categoryB.id}`, 200, () => rowB.locator('[data-on]').click());
  await waitForServer(async () => findNamed((await adminGET('/api/v1/admin/categories')).categories, names.categoryBRenamed)?.enabled === true);
  record('category enabled state toggles off/on through UI', true);

  categories = (await adminGET('/api/v1/admin/categories')).categories;
  const beforeA = categories.findIndex(item => item.id === categoryA.id);
  const beforeB = categories.findIndex(item => item.id === categoryB.id);
  rowB = categoryRow(names.categoryBRenamed);
  await uiMutation('PUT', '/api/v1/admin/categories/order', 200, () => rowB.locator('[data-act="cat-up"]').click());
  categories = (await adminGET('/api/v1/admin/categories')).categories;
  record('complete category order moves through server-owned sequence',
    beforeB === beforeA + 1 && categories.findIndex(item => item.id === categoryB.id) === beforeA && categories.findIndex(item => item.id === categoryA.id) === beforeB);

  await navigate('products', '菜品管理', '#tbl-host');
  const zero = await createProduct(names.product0, names.categoryA, [], -1);
  resources.productIDs.push(zero.id);
  const one = await createProduct(names.product1, names.categoryA, [imageFiles[0]], -1);
  resources.productIDs.push(one.id);
  const three = await createProduct(names.product3, names.categoryA, imageFiles, 2);
  resources.productIDs.push(three.id);

  const zeroStored = await readProduct(zero.id, today);
  const oneStored = await readProduct(one.id, today);
  const threeStored = await readProduct(three.id, today);

  record('products persist zero, one and three image boundaries',
    zeroStored.images.length === 0 && oneStored.images.length === 1 && threeStored.images.length === 3);
  record('three-image product persists explicit cover-first order',
    JSON.stringify(threeStored.images.map(image => image.object_key)) === JSON.stringify([imageKeys[2], imageKeys[0], imageKeys[1]]));
  record('uploaded product object URLs are publicly readable', await allObjectURLsReadable([...oneStored.images, ...threeStored.images]));

  let order = await productOrder(categoryA.id, today);
  let rowThree = productRow(names.product3);
  await uiMutation('PUT', '/api/v1/admin/products/order', 200, () => rowThree.locator('[data-act="up"]').click());
  order = await productOrder(categoryA.id, today);
  const upOK = JSON.stringify(order) === JSON.stringify([zero.id, three.id, one.id]);
  rowThree = productRow(names.product3);
  await uiMutation('PUT', '/api/v1/admin/products/order', 200, () => rowThree.locator('[data-act="down"]').click());
  order = await productOrder(categoryA.id, today);
  record('product up/down writes complete category-local order', upOK && JSON.stringify(order) === JSON.stringify([zero.id, one.id, three.id]));

  let rowZero = productRow(names.product0);
  const shelfOff = await uiMutation('PUT', `/api/v1/admin/products/${zero.id}/status`, 200, () => rowZero.locator('[data-act="shelf"]').click());
  let product = await readProduct(zero.id, today);
  await waitForNamedRowText('#tbl-host tr[data-id]', names.product0, '上架');
  rowZero = productRow(names.product0);
  const shelfOn = await uiMutation('PUT', `/api/v1/admin/products/${zero.id}/status`, 200, () => rowZero.locator('[data-act="shelf"]').click());
  await waitForNamedRowText('#tbl-host tr[data-id]', names.product0, '下架');
  const shelfRestored = await readProduct(zero.id, today);
  record('product shelf status toggles off/on and persists', product.listed === false && shelfRestored.listed === true,
    { after_off: product.listed, after_on: shelfRestored.listed, request_off: shelfOff.requestBody, request_on: shelfOn.requestBody });

  rowZero = productRow(names.product0);
  await uiMutation('PUT', `/api/v1/admin/products/${zero.id}/soldout`, 200, () => rowZero.locator('[data-act="sold"]').click());
  const soldToday = await readProduct(zero.id, today);
  const soldTomorrow = await readProduct(zero.id, tomorrow);
  await waitForNamedRowText('#tbl-host tr[data-id]', names.product0, '恢复售卖');
  rowZero = productRow(names.product0);
  await uiMutation('PUT', `/api/v1/admin/products/${zero.id}/soldout`, 200, () => rowZero.locator('[data-act="sold"]').click());
  await waitForNamedRowText('#tbl-host tr[data-id]', names.product0, '标记售罄');
  record('today sold-out never leaks into tomorrow and can be restored',
    soldToday.sold_out === true && soldTomorrow.sold_out === false && (await readProduct(zero.id, today)).sold_out === false);

  await navigate('categories', '分类管理', '#cat-host');
  rowA = categoryRow(names.categoryA);
  await rowA.locator('[data-del]').click();
  const protectedDelete = await uiMutation('DELETE', `/api/v1/admin/categories/${categoryA.id}`, 409, () => page.locator('.modal [data-a="ok"]').click());
  categories = (await adminGET('/api/v1/admin/categories')).categories;
  record('category containing products returns 409 and UI keeps the row',
    protectedDelete.body?.error?.code === 'CATALOG_CONFLICT' && findNamed(categories, names.categoryA)?.product_count === 3 && await categoryRow(names.categoryA).isVisible());

  await navigate('products', '菜品管理', '#tbl-host');
  for (const item of [zero, one, three]) await deleteProductUI(item.id, item.name);
  resources.productIDs = [];
  record('three temporary products are removed through visible UI', (await productOrder(categoryA.id, today)).length === 0);

  await navigate('categories', '分类管理', '#cat-host');
  for (const item of [{ id: categoryA.id, name: names.categoryA }, { id: categoryB.id, name: names.categoryBRenamed }]) await deleteCategoryUI(item.id, item.name);
  resources.categoryIDs = [];
  categories = (await adminGET('/api/v1/admin/categories')).categories;
  record('temporary categories are removed while audit receipts remain',
    !findNamed(categories, names.categoryA) && !findNamed(categories, names.categoryBRenamed) &&
    await auditContains('CREATE_PRODUCT', zero.id) && await auditContains('DELETE_CATEGORY', categoryA.id));
}

async function createCategory(name) {
  await page.locator('[data-new]').click();
  await page.locator('.modal #c-name').fill(name);
  const response = await uiMutation('POST', '/api/v1/admin/categories', 201, () => page.locator('.modal [data-a="ok"]').click());
  const category = payloadOf(response.body, 'category');
  await waitForText('.cat-row', name, true);
  return category;
}

async function createProduct(name, categoryName, files, coverIndex) {
  await page.locator('[data-new]').click();
  await page.locator('.drawer #f-name').fill(name);
  await page.locator('.drawer #f-price').fill('12.34');
  await page.locator('.drawer #f-meal').selectOption('all');
  await page.locator('.drawer #f-cat').selectOption({ label: categoryName });
  await page.locator('.drawer #f-desc').fill(`组合 UI1 ${suffix}`);
  if (files.length) {
    await page.locator('.drawer #f-file').setInputFiles(files.map(file => file.path));
    await page.waitForFunction(expected => document.querySelectorAll('.drawer .img-cell:not(.add)').length === expected, files.length);
  }
  if (files.length === 3) record('three-image editor presents no fourth-image add control', await page.locator('.drawer [data-add]').count() === 0);
  if (coverIndex > 0) {
    const cover = page.locator(`.drawer [data-cover="${coverIndex}"]`);
    if (await cover.count() !== 1) throw new Error('non-cover product image has no PRD set-cover control');
    await cover.click();
  }
  const response = await uiMutation('POST', '/api/v1/admin/products', 201, () => page.locator('.drawer [data-a="ok"]').click());
  const product = payloadOf(response.body, 'product');
  await waitForText('#tbl-host tr[data-id]', name, true);
  return product;
}

async function deleteProductUI(id, name) {
  const row = productRow(name);
  await row.locator('[data-act="edit"]').click();
  await page.locator('.drawer [data-a="del"]').click();
  await uiMutation('DELETE', `/api/v1/admin/products/${id}`, 200, () => page.locator('.modal [data-a="ok"]').click());
  await waitForText('#tbl-host tr[data-id]', name, false);
}

async function deleteCategoryUI(id, name) {
  const row = categoryRow(name);
  await row.locator('[data-del]').click();
  await uiMutation('DELETE', `/api/v1/admin/categories/${id}`, 200, () => page.locator('.modal [data-a="ok"]').click());
  await waitForText('.cat-row', name, false);
}

async function layerScenario() {
  await navigate('layer', '开屏图层', '#phone');
  const previewText = await page.locator('#content').innerText();
  record('launch-layer preview matches current home/identity scope',
    previewText.includes('用户端首页') && previewText.includes('身份选择页') && !['绥安洗衣', '绥安洗车', '接单', '业务选择页'].some(value => previewText.includes(value)));

  const upload = await uiMutation('POST', '/api/v1/upload', 201, () => page.locator('#file').setInputFiles(imageFiles[2].path));
  layerDirty = true;
  const stored = payloadOf(upload.body, 'image');
  await page.locator('#lay-img').waitFor({ state: 'visible' });
  record('transparent PNG upload uses real object service', stored.object_key === imageKeys[2] && typeof stored.url === 'string' && stored.url !== '');

  const before = await layerGeometryText();
  await page.locator('#size').evaluate(node => {
    node.value = '44';
    node.dispatchEvent(new Event('input', { bubbles: true }));
  });
  const sized = await layerGeometryText();
  const box = await page.locator('#lay-img').boundingBox();
  if (!box) throw new Error('launch layer image has no layout box');
  await page.locator('#lay-img').evaluate(img => {
    window.__layerPointerEvents = [];
    for (const type of ['pointerdown', 'pointermove', 'pointerup']) img.addEventListener(type, () => window.__layerPointerEvents.push(type), true);
  });
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 + 48, box.y + box.height / 2 + 72, { steps: 5 });
  await page.mouse.up();
  const after = await layerGeometryText();
  const pointerEvents = await page.evaluate(() => window.__layerPointerEvents || []);
  record('layer size slider and pointer positioning change visible geometry',
    before.size !== sized.size && sized.size === '44' && (sized.cx !== after.cx || sized.cy !== after.cy), { before, sized, after, pointer_events: pointerEvents });

  await page.locator('#en').click();
  const disabledSave = await uiMutation('PUT', '/api/v1/admin/launch-layer', 200, () => page.locator('#save').click());
  const disabled = disabledSave.body;
  record('disabling layer clears server projection without a stale object key', disabled.enabled === false && disabled.image_object_key === '',
    { enabled: disabled.enabled, image_object_key: disabled.image_object_key, expected_object_key: stored.object_key });
  const hiddenPublic = await publicStorefrontFromNewContext();
  record('disabled layer is omitted from a new unauthenticated device context', !hiddenPublic.body.storefront.launch_layer);
  await hiddenPublic.context.close();

  await page.locator('#en').click();
  await uiMutation('PUT', '/api/v1/admin/launch-layer', 200, () => page.locator('#save').click());
  const adminLayer = await adminGET('/api/v1/admin/launch-layer');
  const publicLayer = await publicStorefrontFromNewContext();
  const projected = publicLayer.body.storefront.launch_layer;
  record('enabled layer geometry persists and is projected by public storefront',
    adminLayer.enabled === true && adminLayer.image_object_key === stored.object_key &&
    approx(adminLayer.size_ratio, 0.44) && projected?.image?.object_key === stored.object_key &&
    approx(projected.width_ratio, adminLayer.size_ratio) && approx(projected.center_x, adminLayer.center_x) && approx(projected.center_y, adminLayer.center_y));
  record('new device can read the projected object URL', projected && await browserURLReadable(publicLayer.context, projected.image.url));
  await publicLayer.context.close();

  await page.locator('#rm').click();
  await page.locator('.modal [data-a="ok"]').click();
  await uiMutation('DELETE', '/api/v1/admin/launch-layer', 200, () => page.locator('#save').click());
  const cleared = await adminGET('/api/v1/admin/launch-layer');
  const clearedPublic = await publicStorefrontFromNewContext();
  record('clear removes old object key and public launch projection',
    cleared.image_object_key === '' && cleared.enabled === false && !clearedPublic.body.storefront.launch_layer);
  await clearedPublic.context.close();
  record('launch configuration writes leave audit receipts', await auditContains('CONFIGURE_LAUNCH_LAYER'));
}

async function navigate(route, title, readySelector) {
  await page.locator(`a[data-r="${route}"]`).click();
  await page.waitForFunction(expected => document.querySelector('#tb-title')?.textContent === expected, title);
  await page.locator(readySelector).waitFor({ state: 'visible' });
}

function categoryRow(name) { return page.locator('.cat-row').filter({ hasText: name }); }
function productRow(name) { return page.locator('#tbl-host tr[data-id]').filter({ hasText: name }); }

async function uiMutation(method, pathname, expectedStatus, action) {
  const responsePromise = page.waitForResponse(response => {
    const url = new URL(response.url());
    return response.request().method() === method && url.pathname === pathname;
  });
  await action();
  const response = await responsePromise;
  let body = null;
  try { body = await response.json(); } catch { throw new Error(`${method} ${pathname} returned invalid JSON`); }
  if (response.status() !== expectedStatus) {
    const code = body?.error?.code;
    throw new Error(`${method} ${pathname} returned ${response.status()}${code ? ` ${code}` : ''}`);
  }
  let requestBody = null;
  try { requestBody = response.request().postDataJSON(); } catch {}
  return { status: response.status(), body, requestBody };
}

async function waitForText(selector, value, present) {
  await page.waitForFunction(({ selector, value, present }) => {
    const found = Array.from(document.querySelectorAll(selector)).some(node => (node.textContent || '').includes(value));
    return found === present;
  }, { selector, value, present });
}

async function waitForNamedRowText(selector, name, value) {
  await page.waitForFunction(({ selector, name, value }) => {
    const row = Array.from(document.querySelectorAll(selector)).find(node => (node.textContent || '').includes(name));
    return !!row && (row.textContent || '').includes(value);
  }, { selector, name, value });
}

async function waitForServer(predicate) {
  const deadline = Date.now() + 5000;
  let lastError;
  while (Date.now() < deadline) {
    try { if (await predicate()) return; } catch (error) { lastError = error; }
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw lastError || new Error('server readback did not converge');
}

async function readProduct(id, serviceDate) {
  return payloadOf(await adminGET(`/api/v1/admin/products/${id}?service_date=${encodeURIComponent(serviceDate)}`), 'product');
}

async function productOrder(categoryID, serviceDate) {
  const body = await adminGET(`/api/v1/admin/products?service_date=${encodeURIComponent(serviceDate)}`);
  return (body.products || []).filter(product => product.category_id === categoryID).map(product => product.id);
}

async function allObjectURLsReadable(images) {
  for (const image of images) {
    const response = await fetch(new URL(image.url, apiOrigin), { redirect: 'error' });
    if (response.status !== 200 || response.headers.get('content-type') !== 'image/png' || (await response.arrayBuffer()).byteLength === 0) return false;
  }
  return true;
}

async function publicStorefrontFromNewContext() {
  const deviceContext = await browser.newContext();
  const devicePage = await deviceContext.newPage();
  const response = await devicePage.goto(`${proxy.origin}/api/v1/storefront/settings`, { waitUntil: 'networkidle' });
  if (!response || response.status() !== 200) {
    await deviceContext.close();
    throw new Error('public storefront did not return 200 in new context');
  }
  return { context: deviceContext, body: await response.json() };
}

async function browserURLReadable(deviceContext, value) {
  const objectPage = await deviceContext.newPage();
  const response = await objectPage.goto(new URL(value, proxy.origin).toString(), { waitUntil: 'load' });
  return !!response && response.status() === 200 && response.headers()['content-type'] === 'image/png';
}

async function layerGeometryText() {
  return {
    size: await page.locator('#size').inputValue(),
    cx: await page.locator('#vx').innerText(),
    cy: await page.locator('#vy').innerText(),
  };
}

async function auditContains(action, targetID) {
  let afterID = '';
  for (let pageNumber = 0; pageNumber < 20; pageNumber += 1) {
    const query = new URLSearchParams({ action, limit: '100', ...(afterID ? { after_id: afterID } : {}) });
    const body = await adminGET(`/api/v1/admin/audits?${query}`);
    if ((body.audits || []).some(entry => entry.action === action && (targetID === undefined || entry.target_id === targetID) && entry.result_code === 'SUCCEEDED:OK')) return true;
    if (!body.next_after_id) return false;
    afterID = body.next_after_id;
  }
  throw new Error(`audit pagination exceeded for ${action}`);
}

async function cleanupResiduals() {
  if (!sessionToken) return;
  for (const id of resources.productIDs.slice().reverse()) {
    try { await cleanupDELETE(`/api/v1/admin/products/${id}`); cleanup.push({ name: `product ${id}`, ok: true }); }
    catch (error) { cleanup.push({ name: `product ${id}`, ok: false, error: safeMessage(error) }); }
  }
  for (const id of resources.categoryIDs.slice().reverse()) {
    try { await cleanupDELETE(`/api/v1/admin/categories/${id}`); cleanup.push({ name: `category ${id}`, ok: true }); }
    catch (error) { cleanup.push({ name: `category ${id}`, ok: false, error: safeMessage(error) }); }
  }
  if (layerDirty && initialLayer) {
    try {
      if (initialLayer.image_object_key) await adminWrite('/api/v1/admin/launch-layer', 'PUT', initialLayer, 200);
      else await adminWrite('/api/v1/admin/launch-layer', 'DELETE', undefined, 200);
      const restored = await adminGET('/api/v1/admin/launch-layer');
      cleanup.push({ name: 'launch-layer baseline', ok: sameLayer(restored, initialLayer) });
    } catch (error) { cleanup.push({ name: 'launch-layer baseline', ok: false, error: safeMessage(error) }); }
  }
}

function record(name, ok, detail) { checks.push({ name, ok: Boolean(ok), ...(detail === undefined ? {} : { detail }) }); }
function findNamed(items, name) { return (Array.isArray(items) ? items : []).find(item => item.name === name); }
function payloadOf(body, key) {
  const payload = body && body[key];
  if (!payload || typeof payload !== 'object') throw new Error(`${key} response is malformed`);
  return payload;
}
function safeMessage(error) {
  return error?.message ? String(error.message).replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : 'unknown error';
}
function addISODate(value, days) {
  const date = new Date(`${value}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}
function approx(left, right) { return Number.isFinite(Number(left)) && Math.abs(Number(left) - Number(right)) < 0.000001; }
function sameLayer(actual, expected) {
  return actual.image_object_key === expected.image_object_key && actual.enabled === expected.enabled &&
    approx(actual.size_ratio, expected.size_ratio) && approx(actual.center_x, expected.center_x) &&
    approx(actual.center_y, expected.center_y) && approx(actual.aspect_ratio, expected.aspect_ratio);
}

function writePNG(filename, red, green, blue, alpha) {
  const filePath = path.join(evidenceRoot, filename);
  const bytes = solidPNG(red, green, blue, alpha);
  writeFileSync(filePath, bytes, { mode: 0o600 });
  return { path: filePath, bytes };
}

function solidPNG(red, green, blue, alpha) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  const header = Buffer.alloc(13);
  header.writeUInt32BE(2, 0); header.writeUInt32BE(2, 4);
  header[8] = 8; header[9] = 6;
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

function exactLoopbackOrigin(value) {
  const parsed = new URL(value);
  if (parsed.protocol !== 'http:' || parsed.hostname !== '127.0.0.1' || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) {
    throw new Error('ORDER_COMPOSED_API_ORIGIN must be an exact http://127.0.0.1:<port> origin');
  }
  return parsed.origin;
}

async function acquirePCSession(origin) {
  const session = await jsonRequest(origin, '/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc-catalog-ui1-${randomUUID()}` } }, 201);
  const bearer = exactToken(session.access_token, 'Mini session');
  await jsonRequest(origin, '/api/v1/me/bind-phone', { method: 'POST', bearer, idempotencyKey: randomUUID(), body: { code: `pc-catalog-phone-${randomUUID()}` } }, 200);
  const login = await jsonRequest(origin, '/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(exactString(login.qr_payload, 'qr_payload'));
  const loginID = exactString(login.login_id, 'login_id');
  await jsonRequest(origin, '/api/v1/me/admin-login/approve', {
    method: 'POST', bearer,
    body: { login_id: loginID, approval_secret: exactString(payload.searchParams.get('approval_secret'), 'approval_secret'), code: `pc-catalog-approve-${randomUUID()}` },
  }, 200);
  const poll = await jsonRequest(origin, '/api/v1/admin/auth/poll', {
    method: 'POST', body: { login_id: loginID, poll_secret: exactString(login.poll_secret, 'poll_secret') },
  }, 200);
  if (poll.state !== 'APPROVED' || !poll.session) throw new Error('PC login did not become APPROVED');
  return exactToken(poll.session.token, 'PC session');
}

async function jsonRequest(origin, pathname, options, expectedStatus) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.idempotencyKey) headers['Idempotency-Key'] = options.idempotencyKey;
  const response = await fetch(`${origin}${pathname}`, {
    method: options.method, headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    redirect: 'error',
  });
  let body;
  try { body = await response.json(); } catch { throw new Error(`${pathname} returned invalid JSON`); }
  if (response.status !== expectedStatus) throw new Error(`${pathname} returned ${response.status}`);
  return body;
}

async function adminGET(pathname) {
  return jsonRequest(apiOrigin, pathname, { method: 'GET', bearer: sessionToken }, 200);
}

async function adminWrite(pathname, method, body, expectedStatus) {
  return jsonRequest(apiOrigin, pathname, { method, bearer: sessionToken, idempotencyKey: randomUUID(), body }, expectedStatus);
}

async function cleanupDELETE(pathname) {
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method: 'DELETE',
    headers: { Accept: 'application/json', Authorization: `Bearer ${sessionToken}`, 'Idempotency-Key': randomUUID() },
    redirect: 'error',
  });
  if (response.status !== 200 && response.status !== 404) throw new Error(`${pathname} cleanup returned ${response.status}`);
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
    method: request.method, headers,
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
    response.writeHead(404); response.end('not found'); return;
  }
  const contentType = target.endsWith('.js') ? 'text/javascript; charset=utf-8' : target.endsWith('.css') ? 'text/css; charset=utf-8' : target.endsWith('.png') ? 'image/png' : 'text/html; charset=utf-8';
  response.writeHead(200, { 'Content-Type': contentType, 'Cache-Control': 'no-store' });
  createReadStream(target).pipe(response);
}
