import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || toolRoot;
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();
const upstreamOrigin = process.env.ORDER_COMPOSED_API_ORIGIN;

if (!existsSync(browserPath)) {
  throw new Error('locked Chromium is missing; reuse the configured MINIPROGRAM_UI_DEPS cache');
}
if (!/^http:\/\/127\.0\.0\.1:\d{1,5}$/.test(upstreamOrigin || '')) {
  throw new Error('ORDER_COMPOSED_API_ORIGIN must be an explicit http://127.0.0.1:<port> origin');
}

process.env.CHROME_BIN = browserPath;
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET, POST, PUT, OPTIONS',
    'access-control-allow-headers': 'authorization, content-type, idempotency-key',
    'access-control-expose-headers': 'x-request-id',
  };
}

async function startTransparentProxy(origin) {
  const target = new URL(origin);
  const requests = [];
  const server = http.createServer((request, response) => {
    if (request.method === 'OPTIONS') {
      response.writeHead(204, corsHeaders());
      response.end();
      return;
    }
    const headers = Object.assign({}, request.headers, { host: target.host });
    delete headers.connection;
    const upstream = http.request({
      hostname: target.hostname,
      port: target.port,
      method: request.method,
      path: request.url,
      headers,
    }, upstreamResponse => {
      const responseHeaders = Object.assign({}, upstreamResponse.headers, corsHeaders());
      delete responseHeaders.connection;
      requests.push({
        method: request.method,
        path: request.url,
        status: upstreamResponse.statusCode,
        request_id: upstreamResponse.headers['x-request-id'] || '',
      });
      response.writeHead(upstreamResponse.statusCode || 502, responseHeaders);
      upstreamResponse.pipe(response);
    });
    upstream.on('error', error => {
      requests.push({ method: request.method, path: request.url, status: 0, error: error.code || 'UPSTREAM_ERROR' });
      if (!response.headersSent) response.writeHead(502, Object.assign({ 'content-type': 'application/json' }, corsHeaders()));
      response.end(JSON.stringify({ error: { code: 'COMPOSED_UPSTREAM_UNAVAILABLE' } }));
    });
    request.pipe(upstream);
  });
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  return {
    origin: `http://127.0.0.1:${address.port}`,
    requests,
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  };
}

const proxy = await startTransparentProxy(upstreamOrigin);
process.env.ORDER_COMPOSED_PROXY_ORIGIN = proxy.origin;
console.log('UI1_COMPOSED_ENV', JSON.stringify({
  runner: 'order-miniprogram-ui-gates@1.0.0',
  simulator: 'miniprogram-simulate@1.6.2',
  browser: browserVersion,
  upstream: upstreamOrigin,
  proxy: `${proxy.origin} (random loopback)`,
}));

let exitCode;
try {
  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.composed.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
} finally {
  await proxy.close();
}

console.log('UI1_COMPOSED_RESULT', JSON.stringify({
  status: exitCode === 0 ? 'PASS' : 'FAIL',
  scenarios: 4,
  evidence_level: 'L3_LOCAL_COMPOSED',
  upstream_requests: proxy.requests,
}));
if (exitCode !== 0) process.exitCode = exitCode;
