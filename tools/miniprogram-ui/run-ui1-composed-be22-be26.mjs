import { execFileSync, spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { createRequire } from 'node:module';
import { closeSync, existsSync, mkdirSync, mkdtempSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import http from 'node:http';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(toolRoot, '../..');
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || toolRoot;
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const browserVersion = existsSync(browserPath)
  ? execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim() : '';
const candidateSHA = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: repositoryRoot, encoding: 'utf8' }).trim();
const runID = `be22-be26-${Date.now()}-${randomUUID().slice(0, 8)}`;
const databaseName = `order_be22_be26_${randomUUID().replaceAll('-', '').slice(0, 16)}`;
const container = exactToken(process.env.ORDER_TEST_MYSQL_INSTANCE || '', /^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$/, 'ORDER_TEST_MYSQL_INSTANCE');
const mysqlPassword = required(process.env.ORDER_TEST_MYSQL_PASSWORD, 'ORDER_TEST_MYSQL_PASSWORD');
const mysqlHost = required(process.env.ORDER_TEST_MYSQL_HOST, 'ORDER_TEST_MYSQL_HOST');
const mysqlPort = exactToken(process.env.ORDER_TEST_MYSQL_PORT || '', /^[1-9][0-9]{0,4}$/, 'ORDER_TEST_MYSQL_PORT');
const mysqlUser = exactToken(process.env.ORDER_TEST_MYSQL_USER || '', /^[a-zA-Z0-9_.-]+$/, 'ORDER_TEST_MYSQL_USER');
const tlsMode = exactToken(process.env.ORDER_TEST_MYSQL_TLS_MODE || '', /^(disabled|required|verify-ca|verify-identity)$/, 'ORDER_TEST_MYSQL_TLS_MODE');
const isolated = process.env.ORDER_TEST_MYSQL_ISOLATED === 'YES';
const receiptPath = path.resolve(process.env.ORDER_BE22_BE26_RECEIPT_PATH
  || path.join(repositoryRoot, '.scratch/mini-be22-be26-ui1/receipt.json'));
const temporaryRoot = mkdtempSync(path.join(os.tmpdir(), 'order-be22-be26-ui1-'));
const apiBinary = path.join(temporaryRoot, 'order-api');
const migrateBinary = path.join(temporaryRoot, 'order-migrate');
const apiLogPath = path.join(temporaryRoot, 'order-api.log');
const requests = [];
const cleanup = [];
let apiProcess;
let proxy;
let schemaCreated = false;
let apiLogFD;
let mysqlFacts = null;
let executionFailure = '';
let exitCode = 1;

if (!browserVersion) throw new Error('locked Chromium is missing; reuse MINIPROGRAM_UI_DEPS');
if (!isolated || container !== 'order-mysql-w3') throw new Error('the one isolated order-mysql-w3 gate is required');

try {
  buildBinaries();
  mysqlExec(`CREATE DATABASE \`${databaseName}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci`, false);
  schemaCreated = true;
  execFileSync(migrateBinary, [], { cwd: repositoryRoot, env: apiEnvironment('127.0.0.1:1'), stdio: ['ignore', 'ignore', 'pipe'] });

  // BOOTSTRAP_ROOT_OF_TRUST_FIXTURE: the fresh production schema has no
  // authenticated first-owner/storefront creation route. These three fixed
  // singleton facts are infrastructure bootstrap only and never acceptance evidence.
  mysqlExec([
    "INSERT INTO merchant_accounts(id,phone,name,role,enabled,record_version,auth_version,created_at,updated_at) VALUES(1,'+8613800000000','UI1临时主账号','OWNER',TRUE,1,1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))",
    "INSERT INTO storefront_settings(id,store_name,store_address,pickup_point,announcement,business_status,flavor_options_json,record_version) VALUES(1,'UI1临时门店','UI1临时地址','UI1临时取餐点','','open',JSON_ARRAY(),1)",
    "INSERT INTO discount_settings(id,rate_percent,discount_version,whitelist_version,updated_at) VALUES(1,100,1,1,UTC_TIMESTAMP(6))",
  ].join(';'));

  const apiPort = await reservePort();
  const apiOrigin = `http://127.0.0.1:${apiPort}`;
  apiLogFD = openSync(apiLogPath, 'a', 0o600);
  apiProcess = spawn(apiBinary, [], {
    cwd: repositoryRoot,
    env: apiEnvironment(`127.0.0.1:${apiPort}`),
    stdio: ['ignore', 'ignore', apiLogFD],
  });
  await waitReady(apiOrigin, apiProcess);
  proxy = await startProxy(apiOrigin, requests);

  process.env.CHROME_BIN = browserPath;
  process.env.ORDER_BE22_BE26_PROXY_ORIGIN = proxy.origin;
  process.env.ORDER_BE22_BE26_RUN_ID = runID;
  process.stdout.write(`MINI_BE22_BE26_UI1_ENV ${JSON.stringify({
    candidate_sha: candidateSHA,
    browser: browserVersion,
    database: `${databaseName} (ephemeral)`,
    api: `${apiOrigin} (private random loopback)`,
    proxy: `${proxy.origin} (private random loopback)`,
    cases: ['BE-22', 'BE-26'],
  })}\n`);

  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed-be22-be26.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
  mysqlFacts = JSON.parse(mysqlExec([
    "SELECT JSON_OBJECT(",
    "'schema_version',(SELECT COALESCE(MAX(version),0) FROM schema_migrations),",
    "'users',(SELECT COUNT(*) FROM miniprogram_users),",
    "'bound_users',(SELECT COUNT(*) FROM miniprogram_users WHERE primary_phone IS NOT NULL AND primary_phone_bound_at IS NOT NULL),",
    "'quotes',(SELECT COUNT(*) FROM quotes),",
    "'prepayments',(SELECT COUNT(*) FROM prepayments),",
    "'orders',(SELECT COUNT(*) FROM orders),",
    "'ready_orders',(SELECT COUNT(*) FROM orders WHERE state='READY_FOR_PICKUP'),",
    "'completed_orders',(SELECT COUNT(*) FROM orders WHERE state='COMPLETED'),",
    "'redemptions',(SELECT COUNT(*) FROM orders WHERE redeemed_at IS NOT NULL)",
    ")",
  ].join('')).trim());
} catch (error) {
  executionFailure = safeMessage(error);
  exitCode = 1;
} finally {
  if (proxy) await proxy.close().then(() => cleanup.push({ target: 'private proxy', ok: true }))
    .catch(error => cleanup.push({ target: 'private proxy', ok: false, error: safeMessage(error) }));
  if (apiProcess) await stopProcess(apiProcess).then(() => cleanup.push({ target: 'private API', ok: true }))
    .catch(error => cleanup.push({ target: 'private API', ok: false, error: safeMessage(error) }));
  if (apiLogFD !== undefined) closeSync(apiLogFD);
  if (schemaCreated) {
    try {
      mysqlExec(`DROP DATABASE \`${databaseName}\``, false);
      cleanup.push({ target: `ephemeral schema ${databaseName}`, ok: true });
    } catch (error) {
      cleanup.push({ target: `ephemeral schema ${databaseName}`, ok: false, error: safeMessage(error) });
    }
  }
}

const passed = exitCode === 0 && !executionFailure && mysqlFacts
  && mysqlFacts.schema_version === 44 && mysqlFacts.bound_users === 1
  && mysqlFacts.quotes >= 2 && mysqlFacts.prepayments >= 2 && mysqlFacts.orders >= 2
  && mysqlFacts.completed_orders === 1 && mysqlFacts.redemptions === 1
  && cleanup.every(item => item.ok);
const receipt = {
  schema: 'order.mini-be22-be26.ui1.v1',
  candidate_sha: candidateSHA,
  generated_at: new Date().toISOString(),
  browser: browserVersion,
  evidence_level: 'L3_LOCAL_COMPOSED',
  cases: ['BE-22', 'BE-26'],
  status: passed ? 'PASS' : 'FAIL',
  database: { kind: 'fresh_ephemeral_mysql8_schema', name: databaseName, facts: mysqlFacts },
  bootstrap: {
    classification: 'BOOTSTRAP_ROOT_OF_TRUST_FIXTURE',
    evidence: false,
    rows: ['merchant_accounts OWNER singleton', 'storefront_settings singleton', 'discount_settings singleton'],
    todo_code: 'P0: fresh production schema has no first-owner/storefront bootstrap path; do not mistake this fixture for deploy readiness',
  },
  requests,
  cleanup,
  error: executionFailure || undefined,
  api_log_tail: executionFailure && existsSync(apiLogPath)
    ? readFileSync(apiLogPath, 'utf8').split('\n').slice(-10).join('\n').replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : undefined,
};
mkdirSync(path.dirname(receiptPath), { recursive: true, mode: 0o700 });
writeFileSync(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600 });
rmSync(temporaryRoot, { recursive: true, force: true });
process.stdout.write(`MINI_BE22_BE26_UI1_RESULT ${JSON.stringify({
  status: receipt.status,
  candidate_sha: candidateSHA,
  scenarios: 2,
  requests: requests.length,
  mysql_facts: mysqlFacts,
  cleanup,
  receipt: path.relative(repositoryRoot, receiptPath),
  error: executionFailure || undefined,
})}\n`);
if (!passed) process.exitCode = 1;

function buildBinaries() {
  const goEnv = Object.assign({}, process.env, { GOPROXY: process.env.GOPROXY || 'off', GOTOOLCHAIN: process.env.GOTOOLCHAIN || 'go1.26.5' });
  execFileSync('go', ['build', '-o', migrateBinary, './services/api/cmd/order-migrate'], { cwd: repositoryRoot, env: goEnv, stdio: ['ignore', 'ignore', 'pipe'] });
  execFileSync('go', ['build', '-o', apiBinary, './services/api/cmd/order-api'], { cwd: repositoryRoot, env: goEnv, stdio: ['ignore', 'ignore', 'pipe'] });
}

function apiEnvironment(address) {
  return Object.assign({}, process.env, {
    ORDER_ENV: 'development', ORDER_API_HTTP_ADDR: address,
    ORDER_DB_HOST: mysqlHost, ORDER_DB_PORT: mysqlPort, ORDER_DB_NAME: databaseName,
    ORDER_DB_USER: mysqlUser, ORDER_DB_PASSWORD: mysqlPassword, ORDER_DB_TLS_MODE: tlsMode,
    ORDER_WECHAT_MINIPROGRAM_APP_ID: 'wx-local-order-be22-be26',
    ORDER_WECHAT_MINIPROGRAM_APP_SECRET: 'order-local-be22-be26-secret',
    ORDER_LOCAL_PAYMENT_AUTO_PAY: 'true',
  });
}

function mysqlExec(sql, useDatabase = true) {
  return execFileSync('docker', [
    'exec', '--env', `MYSQL_PWD=${mysqlPassword}`, container,
    'mysql', `-u${mysqlUser}`, '--batch', '--skip-column-names', '--raw', useDatabase ? `--database=${databaseName}` : '', '-e', sql,
  ].filter(Boolean), { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

async function reservePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  const address = server.address();
  await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
  return address.port;
}

async function waitReady(origin, child) {
  const deadline = Date.now() + 20000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`private API exited ${child.exitCode}`);
    try {
      const response = await fetch(`${origin}/health/ready`);
      if (response.status === 200) return;
    } catch {}
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error('private API readiness timeout');
}

async function stopProcess(child) {
  if (child.exitCode !== null) return;
  child.kill('SIGTERM');
  const stopped = await Promise.race([
    new Promise(resolve => child.once('exit', () => resolve(true))),
    new Promise(resolve => setTimeout(() => resolve(false), 10000)),
  ]);
  if (!stopped) throw new Error('private API did not stop within 10s');
}

async function startProxy(origin, observations) {
  const target = new URL(origin);
  const server = http.createServer(async (request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, corsHeaders()); response.end(); return;
    }
    const body = await readBody(request);
    if (request.headers['x-be22-be26-force-status'] === '503') {
      observations.push({ phase: 'browser', method: request.method, path: request.url, status: 503, deterministic_transport_failure: true });
      response.writeHead(503, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
      response.end(JSON.stringify({ error: { code: 'LOCAL_PROVIDER_UNAVAILABLE' } }));
      return;
    }
    try {
      const replay = request.headers['x-be22-be26-network-replay'] === 'same-request';
      const first = await upstream(target, request, body, observations, replay ? 1 : 0);
      const result = replay ? await upstream(target, request, body, observations, 2) : first;
      response.writeHead(result.status, Object.assign({}, result.headers, corsHeaders()));
      response.end(result.body);
    } catch (error) {
      observations.push({ phase: 'browser', method: request.method, path: request.url, status: 0, error: safeMessage(error) });
      response.writeHead(502, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
      response.end(JSON.stringify({ error: { code: 'COMPOSED_UPSTREAM_UNAVAILABLE' } }));
    }
  });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolve); });
  return {
    origin: `http://127.0.0.1:${server.address().port}`,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

function upstream(target, original, body, observations, networkAttempt) {
  return new Promise((resolve, reject) => {
    const headers = Object.assign({}, original.headers, { host: target.host });
    delete headers.connection;
    delete headers['x-be22-be26-force-status'];
    delete headers['x-be22-be26-network-replay'];
    const req = http.request({ hostname: target.hostname, port: target.port, method: original.method, path: original.url, headers }, res => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        observations.push({
          phase: 'browser', method: original.method, path: original.url, status: res.statusCode,
          request_id: res.headers['x-request-id'] || '', network_attempt: networkAttempt || undefined,
        });
        resolve({ status: res.statusCode || 502, headers: res.headers, body: Buffer.concat(chunks) });
      });
    });
    req.on('error', reject);
    req.end(body);
  });
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    request.on('data', chunk => chunks.push(chunk));
    request.on('end', () => resolve(Buffer.concat(chunks)));
    request.on('error', reject);
  });
}

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key, x-be22-be26-force-status, x-be22-be26-network-replay',
    'access-control-expose-headers': 'x-request-id',
  };
}

function required(value, name) { if (!value) throw new Error(`${name} is required`); return value; }
function exactToken(value, pattern, name) { if (!pattern.test(value)) throw new Error(`${name} is invalid`); return value; }
function safeMessage(error) { return error && error.message ? String(error.message).replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]') : 'unknown error'; }
