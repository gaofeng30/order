import http from 'node:http';
import net from 'node:net';
import { execFileSync, spawn, spawnSync } from 'node:child_process';
import { createReadStream, existsSync, mkdirSync, mkdtempSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';

const testsRoot = path.dirname(fileURLToPath(import.meta.url));
const appsRoot = path.resolve(testsRoot, '../..');
const repositoryRoot = path.resolve(appsRoot, '..');
const dependencyRoot = required(process.env.MINIPROGRAM_UI_DEPS, 'MINIPROGRAM_UI_DEPS');
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
export const { chromium } = dependencyRequire('playwright');
export const browserPath = chromium.executablePath();
if (!existsSync(browserPath)) throw new Error('locked Chromium is missing; reuse the configured cache');
if (process.env.ORDER_TEST_MYSQL_ISOLATED !== 'YES') throw new Error('ORDER_TEST_MYSQL_ISOLATED=YES is required');
if (process.env.PLAYWRIGHT_BROWSERS_PATH !== '0') throw new Error('PLAYWRIGHT_BROWSERS_PATH=0 is required');

const docker = '/opt/homebrew/bin/docker';
const mysql = {
  host: required(process.env.ORDER_TEST_MYSQL_HOST, 'ORDER_TEST_MYSQL_HOST'),
  port: port(process.env.ORDER_TEST_MYSQL_PORT),
  user: name(process.env.ORDER_TEST_MYSQL_USER, 'ORDER_TEST_MYSQL_USER'),
  password: required(process.env.ORDER_TEST_MYSQL_PASSWORD, 'ORDER_TEST_MYSQL_PASSWORD'),
  tls: required(process.env.ORDER_TEST_MYSQL_TLS_MODE, 'ORDER_TEST_MYSQL_TLS_MODE'),
  container: name(process.env.ORDER_TEST_MYSQL_INSTANCE, 'ORDER_TEST_MYSQL_INSTANCE'),
};

export const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
export const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();
export const suffix = randomUUID().replaceAll('-', '').slice(0, 10);
export const schema = `order_pc0912_${Date.now()}_${suffix}`;
export const evidenceRoot = `/private/tmp/order-pc09-pc12-final-${candidateSHA}`;
mkdirSync(evidenceRoot, { recursive: true, mode: 0o700 });

const buildRoot = mkdtempSync('/private/tmp/order-pc09-pc12-build.');
const binaries = {
  api: path.join(buildRoot, 'order-api'),
  migrate: path.join(buildRoot, 'order-migrate'),
  bootstrap: path.join(buildRoot, 'order-bootstrap'),
};
let apiProcess = null;
let apiOrigin = '';
let schemaCreated = false;

export async function provisionFreshRuntime() {
  for (const [kind, target] of Object.entries(binaries)) run('go', ['build', '-o', target, `./services/api/cmd/order-${kind}`]);
  mysqlExec(`CREATE DATABASE \`${schema}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci`);
  schemaCreated = true;
  run(binaries.migrate, [], runtimeEnv());
  const cleanV44 = mysqlExec(`SELECT IF(COUNT(*)=44 AND MAX(version)=44 AND SUM(dirty)=0,1,0) FROM \`${schema}\`.schema_migrations`).trim();
  if (cleanV44 !== '1') throw new Error(`fresh schema is not clean v44: ${cleanV44}`);
  run(binaries.bootstrap, [], {
    ...runtimeEnv(),
    ORDER_BOOTSTRAP_OWNER_PHONE: '+8613800000000',
    ORDER_BOOTSTRAP_OWNER_NAME: 'PC09-PC12主账号',
    ORDER_BOOTSTRAP_STORE_NAME: '绥安食品',
    ORDER_BOOTSTRAP_STORE_ADDRESS: '本地验收地址',
    ORDER_BOOTSTRAP_PICKUP_POINT: '本地验收取餐点',
  });
}

export async function startPrivateAPI() {
  apiOrigin = `http://127.0.0.1:${await freePort()}`;
  apiProcess = spawn(binaries.api, [], {
    cwd: repositoryRoot,
    env: {
      ...process.env, ...runtimeEnv(),
      ORDER_API_HTTP_ADDR: new URL(apiOrigin).host,
      ORDER_API_SHUTDOWN_TIMEOUT: '2s',
      ORDER_LOCAL_PAYMENT_AUTO_PAY: 'true',
    },
    stdio: ['ignore', 'ignore', 'pipe'],
  });
  let output = '';
  apiProcess.stderr.on('data', chunk => {
    output += chunk;
    writeFileSync(path.join(evidenceRoot, 'order-api.log'), output.slice(-1024 * 1024), { mode: 0o600 });
  });
  for (let attempt = 0; attempt < 100; attempt += 1) {
    if (apiProcess.exitCode !== null) throw new Error(`order-api exited ${apiProcess.exitCode}`);
    try { if ((await fetch(`${apiOrigin}/health/ready`)).status === 200) return; } catch {}
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('order-api readiness timed out');
}

export async function acquireOwnerSessions() {
  const login = await json('/api/v1/auth/miniprogram/session', { method: 'POST', body: { code: `pc0912-login-${suffix}` } }, 201);
  const miniToken = required(login.access_token, 'Mini token');
  await json('/api/v1/me/bind-phone', { method: 'POST', bearer: miniToken, key: randomUUID(), body: { code: `pc0912-phone-${suffix}` } }, 200);
  const qr = await json('/api/v1/admin/auth/qrcode', { method: 'POST', body: {} }, 201);
  const payload = new URL(required(qr.qr_payload, 'qr payload'));
  await json('/api/v1/me/admin-login/approve', {
    method: 'POST', bearer: miniToken,
    body: { login_id: qr.login_id, approval_secret: required(payload.searchParams.get('approval_secret'), 'approval secret'), code: `pc0912-approve-${suffix}` },
  }, 200);
  const poll = await json('/api/v1/admin/auth/poll', { method: 'POST', body: { login_id: qr.login_id, poll_secret: qr.poll_secret } }, 200);
  return { miniToken, pcToken: required(poll.session?.token, 'PC token') };
}

export async function json(pathname, options, expected) {
  const headers = { Accept: 'application/json' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.bearer) headers.Authorization = `Bearer ${options.bearer}`;
  if (options.key) headers['Idempotency-Key'] = options.key;
  const response = await fetch(`${apiOrigin}${pathname}`, {
    method: options.method, headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body), redirect: 'error',
  });
  let body = null;
  try { body = await response.json(); } catch {}
  if (response.status !== expected) throw new Error(`${options.method} ${pathname} returned ${response.status}/${body?.error?.code || 'UNKNOWN'}`);
  return body;
}

export async function startProxy() {
  const server = http.createServer(async (request, response) => {
    try {
      if ((request.url || '').startsWith('/api/')) return await proxyAPI(request, response);
      const pathname = decodeURIComponent(new URL(request.url || '/', 'http://127.0.0.1').pathname);
      const relative = pathname === '/' || pathname === '/web-admin/' || pathname === '/web-admin/index.html' ? 'web-admin/index.html' : pathname.replace(/^\/+/, '');
      const target = path.resolve(appsRoot, relative);
      if (!target.startsWith(`${appsRoot}${path.sep}`) || !existsSync(target) || !statSync(target).isFile()) { response.writeHead(404); response.end('not found'); return; }
      const type = target.endsWith('.js') ? 'text/javascript; charset=utf-8' : target.endsWith('.css') ? 'text/css; charset=utf-8' : 'text/html; charset=utf-8';
      response.writeHead(200, { 'Content-Type': type, 'Cache-Control': 'no-store' });
      createReadStream(target).pipe(response);
    } catch { response.writeHead(502); response.end('private proxy unavailable'); }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return { origin: `http://127.0.0.1:${server.address().port}`, close: () => new Promise(resolve => server.close(resolve)) };
}

async function proxyAPI(request, response) {
  const chunks = [];
  for await (const chunk of request) chunks.push(chunk);
  const target = await fetch(`${apiOrigin}${request.url}`, {
    method: request.method, headers: request.headers,
    body: ['GET', 'HEAD'].includes(request.method) ? undefined : Buffer.concat(chunks), redirect: 'manual',
  });
  const headers = {};
  target.headers.forEach((value, key) => { if (!['content-encoding', 'content-length', 'transfer-encoding'].includes(key)) headers[key] = value; });
  response.writeHead(target.status, headers);
  response.end(Buffer.from(await target.arrayBuffer()));
}

export async function stopAndCleanup() {
  let stopped = true;
  if (apiProcess && apiProcess.exitCode === null) {
    apiProcess.kill('SIGTERM');
    stopped = await new Promise(resolve => {
      const timer = setTimeout(() => { apiProcess.kill('SIGKILL'); resolve(false); }, 10000);
      apiProcess.once('exit', code => { clearTimeout(timer); resolve(code === 0); });
    });
  }
  let dropped = true;
  if (schemaCreated) {
    try {
      mysqlExec(`DROP DATABASE IF EXISTS \`${schema}\``);
      dropped = mysqlExec(`SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name='${schema}'`).trim() === '0';
      schemaCreated = false;
    } catch { dropped = false; }
  }
  try { rmSync(buildRoot, { recursive: true, force: true }); } catch {}
  return { stopped, dropped };
}

function runtimeEnv() {
  return {
    ORDER_ENV: 'development', ORDER_DB_HOST: mysql.host, ORDER_DB_PORT: mysql.port,
    ORDER_DB_NAME: schema, ORDER_DB_USER: mysql.user, ORDER_DB_PASSWORD: mysql.password,
    ORDER_DB_TLS_MODE: mysql.tls, ORDER_WECHAT_MINIPROGRAM_APP_ID: 'wx-local-pc0912',
    ORDER_WECHAT_MINIPROGRAM_APP_SECRET: 'pc0912-local-secret',
  };
}

function mysqlExec(sql) {
  const result = spawnSync(docker, ['exec', '-e', `MYSQL_PWD=${mysql.password}`, mysql.container, 'mysql', '--batch', '--raw', '--skip-column-names', '-u', mysql.user, '--execute', sql], { encoding: 'utf8', maxBuffer: 5 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(`MySQL command failed: ${safe(result.stderr)}`);
  return result.stdout;
}

function run(command, args, extraEnv = {}) {
  const result = spawnSync(command, args, { cwd: repositoryRoot, env: { ...process.env, ...extraEnv }, encoding: 'utf8', maxBuffer: 20 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(`${path.basename(command)} failed: ${safe(result.stderr || result.stdout)}`);
}

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const value = server.address().port;
      server.close(error => error ? reject(error) : resolve(value));
    });
  });
}

function required(value, label) { if (typeof value !== 'string' || !value.trim()) throw new Error(`${label} missing`); return value; }
function name(value, label) { const item = required(value, label); if (!/^[A-Za-z0-9_.-]+$/.test(item)) throw new Error(`${label} invalid`); return item; }
function port(value) { const item = Number(value); if (!Number.isInteger(item) || item < 1 || item > 65535) throw new Error('ORDER_TEST_MYSQL_PORT invalid'); return String(item); }
function safe(value) { return String(value || 'unknown').replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]').slice(0, 2000); }
