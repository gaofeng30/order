import { execFileSync, spawnSync } from 'node:child_process';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import automator from 'miniprogram-automator';

import { startCatalogFixture } from './fixture-server.mjs';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(toolRoot, '../..');
const cliPath = process.env.WECHAT_DEVTOOLS_CLI || '/Applications/wechatwebdevtools.app/Contents/MacOS/cli';
const automationPort = 19420;
const receiptPath = path.resolve(
  process.env.MINIPROGRAM_UI_RECEIPT || path.join(toolRoot, 'receipts/ui2-latest.json'),
);

function requireElement(element, label) {
  if (!element) throw new Error(`${label} was not rendered`);
  return element;
}

function requireIncludes(value, expected, label) {
  if (!String(value).includes(expected)) throw new Error(`${label} did not include the expected visible state`);
}

function sanitizedFailure(error) {
  const message = error instanceof Error ? error.message : String(error);
  return message
    .replaceAll(projectRoot, '<project>')
    .replace(/(token|ticket|session|authorization|cookie)=[^\s&]+/gi, '$1=<redacted>')
    .slice(0, 300);
}

function assertProjectPermission() {
  const result = spawnSync(
    cliPath,
    ['auto', '--project', projectRoot, '--auto-port', String(automationPort), '--lang', 'zh'],
    { encoding: 'utf8' },
  );
  const output = `${result.stdout || ''}\n${result.stderr || ''}`;
  if (output.includes('登录用户不是该小程序的开发者')) {
    const error = new Error('logged-in WeChat Developer Tools account lacks developer permission for the configured mini-program project');
    error.code = 'BLOCKED_EXTERNAL';
    throw error;
  }
  if (result.status !== 0) throw new Error('WeChat Developer Tools project permission probe failed');
}

async function writeReceipt(receipt) {
  await mkdir(path.dirname(receiptPath), { recursive: true });
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { encoding: 'utf8', mode: 0o600 });
}

const projectConfig = JSON.parse(await readFile(path.join(projectRoot, 'project.config.json'), 'utf8'));
const developerToolsVersion = execFileSync(
  '/usr/libexec/PlistBuddy',
  ['-c', 'Print :CFBundleShortVersionString', '/Applications/wechatwebdevtools.app/Contents/Info.plist'],
  { encoding: 'utf8' },
).trim();
const sourceHead = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: projectRoot, encoding: 'utf8' }).trim();
const receipt = {
  schema_version: 1,
  gate: 'UI2',
  status: 'FAIL',
  generated_at: new Date().toISOString(),
  source: { head_sha: sourceHead, project: 'order-miniprogram' },
  environment: {
    runner: 'order-miniprogram-ui-gates@1.0.0',
    automator: 'miniprogram-automator@0.12.1',
    developer_tools: developerToolsVersion,
    base_library: projectConfig.libVersion,
    runtime_class: 'WeChat Developer Tools local simulator',
  },
  scenarios: [],
  unverified_boundary: [
    'experience build and physical device',
    'real identity, order, payment, refund, fulfillment, production, and UI3',
  ],
};

let miniProgram;
let fixture;
try {
  assertProjectPermission();
  fixture = await startCatalogFixture();
  miniProgram = await automator.connect({
    wsEndpoint: `ws://127.0.0.1:${automationPort}`,
  });

  let page = await miniProgram.reLaunch('/pages/home/home');
  if (!page) throw new Error('home page did not launch');
  await page.waitFor(500);
  const greeting = requireElement(await page.$('.greet'), 'home greeting');
  requireIncludes(await greeting.text(), '你好，欢迎光临', 'home greeting');
  const phonePrompt = await page.$('button[open-type="getPhoneNumber"]');
  if (phonePrompt) throw new Error('cold start rendered a phone authorization control');
  const menuEntry = requireElement(await page.$('.search'), 'menu entry');
  await menuEntry.tap();
  await page.waitFor(500);
  page = await miniProgram.currentPage();
  if (!page || page.path !== 'pages/menu/menu') throw new Error('menu entry did not navigate to the menu page');
  receipt.scenarios.push({ id: 'cold-start-anonymous-browse', status: 'PASS' });

  const catalogState = requireElement(await page.$('.catalog-state'), 'catalog error state');
  requireIncludes(await catalogState.text(), '目录加载失败', 'catalog error state');
  const retry = requireElement(await page.$('.catalog-state .btn'), 'catalog retry');
  await retry.tap();
  await page.waitFor(700);
  const recoveredProduct = requireElement(await page.$('.dish-name'), 'recovered catalog product');
  requireIncludes(await recoveredProduct.text(), '恢复后的热菜', 'recovered catalog product');
  receipt.scenarios.push({ id: 'network-error-retry-recovery', status: 'PASS' });

  const categories = await page.$$('.seg');
  if (categories.length !== 2) throw new Error(`rendered category count was ${categories.length}`);
  await categories[1].tap();
  await page.waitFor(200);
  requireIncludes(await categories[1].attribute('class'), 'on', 'second category class');
  const choices = await page.$$('.act-btn');
  if (choices.length !== 2) throw new Error(`rendered product choice count was ${choices.length}`);
  await choices[1].tap();
  await page.waitFor(300);
  const selectionSheet = requireElement(await page.$('.cz-sheet'), 'product-selection sheet');
  requireIncludes(await selectionSheet.text(), '口味偏好', 'product-selection sheet');
  receipt.scenarios.push({ id: 'menu-category-and-product-selection', status: 'PASS' });

  receipt.status = 'PASS';
} catch (error) {
  if (error && error.code === 'BLOCKED_EXTERNAL') {
    receipt.status = 'BLOCKED_EXTERNAL';
    receipt.external_asset = {
      owner: 'mini-program AppID administrator',
      missing: 'developer permission for the currently logged-in WeChat Developer Tools account',
      recovery: 'grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2',
    };
  }
  receipt.failure = sanitizedFailure(error);
  process.exitCode = 1;
} finally {
  if (miniProgram) await miniProgram.close().catch(() => {});
  if (fixture) await fixture.close().catch(() => {});
  await writeReceipt(receipt);
  console.log('UI2_RECEIPT', receiptPath);
  console.log('UI2_RESULT', JSON.stringify({ status: receipt.status, scenarios: receipt.scenarios }));
}
