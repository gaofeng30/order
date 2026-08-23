const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');

test('PC data Interface reads catalog through authenticated HTTP and fails closed', async () => {
  const calls = [];
  const window = {
    crypto: { randomUUID: () => 'op-1' },
    sessionStorage: { getItem: () => 'session-token', setItem() {}, removeItem() {} },
    fetch: async (url, init) => {
      calls.push({ url, init });
      return {
        ok: true,
        status: 200,
        headers: { get: () => 'application/json' },
        json: async () => ({ products: [{ id: '7', name: '面', price_cents: 1234 }] }),
      };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });

  const products = await window.Api.listProducts();
  assert.equal(products[0].id, '7');
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, '/api/v1/admin/products');
  assert.equal(calls[0].init.headers.Authorization, 'Bearer session-token');
});

test('PC app contains no seed, local xlsx, or window store production truth', () => {
  const files = [];
  const walk = dir => fs.readdirSync(dir, { withFileTypes: true }).forEach(entry => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name !== 'tests') walk(full);
    } else if (/\.(js|html)$/.test(entry.name)) files.push(full);
  });
  walk(root);
  const source = files.map(file => fs.readFileSync(file, 'utf8')).join('\n');
  assert.doesNotMatch(source, /window\.__store|window\.Seed|window\.Xlsx/);
  assert.doesNotMatch(fs.readFileSync(path.join(root, 'index.html'), 'utf8'), /data\/(?:seed|xlsx)\.js/);
});

test('PC navigation keeps all twelve frozen pages', () => {
  const source = fs.readFileSync(path.join(root, 'app.js'), 'utf8');
  const routes = [...source.matchAll(/\{ r: '([^']+)', t:/g)].map(match => match[1]);
  assert.deepEqual(routes, [
    'dashboard', 'orders', 'finance', 'pending', 'products', 'product-import',
    'categories', 'staff', 'staff-import', 'accounts', 'settings', 'layer',
  ]);
});

test('business mutation carries one idempotency key and integer-cent DTO', async () => {
  const calls = [];
  const window = {
    crypto: { randomUUID: () => 'write-op-1' },
    sessionStorage: { getItem: () => 'session-token' },
    fetch: async (url, init) => {
      calls.push({ url, init });
      return { ok: true, status: 201, headers: { get: () => 'application/json' }, json: async () => ({ id: '8', name: '面', price_cents: 1234, category_id: '2', images: [], status: 'ON' }) };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });
  await window.Api.saveProduct({ name: '面', price: 12.34, categoryId: '2', meal: 'all', imgs: [] });
  assert.equal(calls[0].url, '/api/v1/admin/products');
  assert.equal(calls[0].init.headers['Idempotency-Key'], 'write-op-1');
  assert.equal(JSON.parse(calls[0].init.body).price_cents, 1234);
});

test('network failure is surfaced and never replaced with browser data', async () => {
  const window = {
    sessionStorage: { getItem: () => 'session-token' },
    fetch: async () => { throw new Error('offline'); },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });
  await assert.rejects(window.Api.listMerchantAccounts(''), /无法连接服务端/);
});

test('PC12 adapters reference all frozen server fact groups', () => {
  const source = fs.readFileSync(path.join(root, 'data/api.js'), 'utf8');
  [
    '/admin/stats', '/admin/orders', '/admin/finance/summary', '/admin/pending-payments',
    '/admin/products', '/admin/products/import/preview', '/admin/categories', '/admin/settings',
    '/admin/launch-layer', '/admin/staff-whitelist', '/admin/staff-whitelist/import/preview',
    '/admin/merchant-accounts', '/import/commit', '/upload',
  ].forEach(endpoint => assert.match(source, new RegExp(endpoint.replaceAll('/', '\\/'))));
});

test('PC QR exchanges use intrinsic dedupe and never send client idempotency keys', async () => {
  const calls = [];
  const window = {
    sessionStorage: { getItem: () => null, setItem() {} },
    fetch: async (url, init) => {
      calls.push({ url, init });
      return { ok: true, status: 201, headers: { get: () => 'application/json' }, json: async () => ({ login_id: '1', poll_secret: 'p', qr_payload: 'q', expires_at: '2026-08-25T00:00:00Z' }) };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'data/api.js'), 'utf8'), { window, URL, Blob, FormData });
  await window.Api.beginPCLogin();
  assert.equal(calls[0].url, '/api/v1/admin/auth/qrcode');
  assert.equal(calls[0].init.headers['Idempotency-Key'], undefined);
});

test('PC login renders a local QR matrix instead of exposing payload text', () => {
  const window = {};
  vm.runInNewContext(fs.readFileSync(path.join(root, 'ui/qrcode.js'), 'utf8'), { window, TextEncoder, Uint8Array, Error });
  const payload = 'order-admin-login://approve?login_id=123456789&approval_secret=abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG';
  const matrix = window.PCQRCode.matrix(payload);
  assert.equal(matrix.length, 49);
  assert.ok(matrix.every(row => row.length === 49 && row.every(value => typeof value === 'boolean')));
  assert.deepEqual(Array.from(matrix[0].slice(0, 7)), [true, true, true, true, true, true, true]);
  const app = fs.readFileSync(path.join(root, 'app.js'), 'utf8');
  assert.match(app, /PCQRCode\.render\(canvas, login\.qr_payload/);
  assert.doesNotMatch(app, /code\.textContent\s*=\s*login\.qr_payload/);
});
