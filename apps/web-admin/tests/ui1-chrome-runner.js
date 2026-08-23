const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawn } = require('node:child_process');

const appsRoot = path.resolve(__dirname, '../..');
const chrome = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

function json(res, status, body) {
  const data = Buffer.from(JSON.stringify(body));
  res.writeHead(status, { 'Content-Type': 'application/json', 'Content-Length': data.length });
  res.end(data);
}

const server = http.createServer((req, res) => {
  if (req.url === '/api/v1/admin/auth/qrcode' && req.method === 'POST') {
    return json(res, 201, {
      login_id: '123456789', poll_secret: 'poll-secret', expires_at: '2030-08-25T00:00:00Z',
      qr_payload: 'order-admin-login://approve?login_id=123456789&approval_secret=abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG',
    });
  }
  if (req.url === '/api/v1/admin/auth/poll' && req.method === 'POST') return json(res, 410, { error: { code: 'PC_LOGIN_EXPIRED', message: 'test poll stopped' } });
  if (req.url.startsWith('/api/')) return json(res, 401, { error: { code: 'UNAUTHENTICATED', message: 'authentication required' } });

  const pathname = new URL(req.url, 'http://127.0.0.1').pathname;
  const relative = pathname === '/' ? 'web-admin/index.html' : pathname.replace(/^\//, '');
  const target = path.resolve(appsRoot, relative);
  if (!target.startsWith(appsRoot + path.sep) || !fs.existsSync(target) || !fs.statSync(target).isFile()) {
    res.writeHead(404); res.end('not found'); return;
  }
  const type = target.endsWith('.js') ? 'text/javascript' : target.endsWith('.css') ? 'text/css' : target.endsWith('.png') ? 'image/png' : 'text/html';
  res.writeHead(200, { 'Content-Type': type });
  fs.createReadStream(target).pipe(res);
});

server.listen(0, '127.0.0.1', () => {
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'order-pc-ui1-'));
  const url = `http://127.0.0.1:${server.address().port}/web-admin/index.html`;
  const child = spawn(chrome, [
    '--headless=new', '--disable-gpu', '--disable-background-networking', '--no-first-run', '--no-default-browser-check',
    `--user-data-dir=${profile}`, '--virtual-time-budget=3000', '--dump-dom', url,
  ], { stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '', errors = '';
  let domComplete = false;
  child.stdout.on('data', chunk => {
    output += chunk;
    if (!domComplete && output.includes('</html>')) {
      domComplete = true;
      child.kill('SIGTERM');
    }
  });
  child.stderr.on('data', chunk => { errors += chunk; });
  const timer = setTimeout(() => child.kill('SIGKILL'), 15000);
  child.on('close', code => {
    clearTimeout(timer);
    server.close();
    fs.rmSync(profile, { recursive: true, force: true });
    const checks = [
      ['Chrome DOM complete', domComplete && (code === 0 || code === null)],
      ['PC login page', output.includes('主账号扫码登录')],
      ['local QR canvas', output.includes('aria-label="PC 登录二维码"')],
      ['payload is not rendered as text', !output.includes('order-admin-login://approve?')],
      ['twelve-page navigation', ['工作台', '订单管理', '财务与对账', '支付待处理', '菜品管理', '菜品批量导入', '分类管理', '营业设置', '开屏图层', '员工折扣白名单', '员工批量导入', '商户账号名单'].every(label => output.includes(label))],
    ];
    checks.forEach(([name, ok]) => process.stdout.write(`${ok ? 'ok' : 'not ok'} - ${name}\n`));
    if (checks.some(([, ok]) => !ok)) {
      if (errors) process.stderr.write(errors.slice(-4000));
      process.exitCode = 1;
    }
  });
});
