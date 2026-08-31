import { execFileSync, spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
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

async function connectWithRetry(attempts = 30, intervalMs = 1000) {
  let lastError;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      return await automator.connect({ wsEndpoint: `ws://127.0.0.1:${automationPort}` });
    } catch (error) {
      lastError = error;
      await new Promise(resolve => setTimeout(resolve, intervalMs));
    }
  }
  throw blockedExternal(
    'UI2 runner machine operator',
    `WeChat Developer Tools automation port ${automationPort} never accepted a connection: ${sanitizedFailure(lastError)}`,
    'open the project in WeChat Developer Tools with 设置 > 安全设置 > CLI/HTTP 调用 enabled, then rerun npm --prefix tools/miniprogram-ui run ui2',
  );
}

// 真实布局引擎下的盒模型。UI0 无 DOM、UI1 不加载 wxss，几何只能在这里量。
async function measure(element) {
  const [size, offset] = await Promise.all([element.size(), element.offset()]);
  return { width: size.width, height: size.height, left: offset.left, top: offset.top };
}

function sanitizedFailure(error) {
  const message = error instanceof Error ? error.message : String(error);
  return message
    .replaceAll(projectRoot, '<project>')
    .replace(/(token|ticket|session|authorization|cookie)=[^\s&]+/gi, '$1=<redacted>')
    .slice(0, 300);
}

function blockedExternal(owner, missing, recovery) {
  const error = new Error(missing);
  error.code = 'BLOCKED_EXTERNAL';
  error.externalAsset = { owner, missing, recovery };
  return error;
}

function developerToolsVersion() {
  const infoPlist = path.resolve(path.dirname(cliPath), '../Info.plist');
  if (!existsSync(infoPlist)) return 'unknown (custom CLI path)';
  try {
    return execFileSync('/usr/libexec/PlistBuddy', ['-c', 'Print :CFBundleShortVersionString', infoPlist], {
      encoding: 'utf8',
    }).trim();
  } catch {
    return 'unknown (version metadata unavailable)';
  }
}

function assertProjectPermission() {
  if (!existsSync(cliPath)) {
    throw blockedExternal(
      'UI2 runner machine operator',
      'WeChat Developer Tools CLI is not installed at the configured path',
      'install WeChat Developer Tools or set WECHAT_DEVTOOLS_CLI to its CLI, then rerun npm --prefix tools/miniprogram-ui run ui2',
    );
  }
  const login = spawnSync(cliPath, ['islogin'], { encoding: 'utf8' });
  const loginOutput = `${login.stdout || ''}\n${login.stderr || ''}`;
  if (login.status !== 0 || !/"login"\s*:\s*true/.test(loginOutput)) {
    throw blockedExternal(
      'UI2 runner machine operator',
      'WeChat Developer Tools CLI login is not active',
      'log in through WeChat Developer Tools, confirm cli islogin reports true, then rerun npm --prefix tools/miniprogram-ui run ui2',
    );
  }
  const result = spawnSync(
    cliPath,
    ['auto', '--project', projectRoot, '--auto-port', String(automationPort), '--lang', 'zh'],
    { encoding: 'utf8' },
  );
  const output = `${result.stdout || ''}\n${result.stderr || ''}`;
  if (output.includes('登录用户不是该小程序的开发者')) {
    throw blockedExternal(
      'mini-program AppID administrator',
      'developer permission for the currently logged-in WeChat Developer Tools account',
      'grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2',
    );
  }
  if (result.status !== 0) throw new Error('WeChat Developer Tools project permission probe failed');
}

async function writeReceipt(receipt) {
  await mkdir(path.dirname(receiptPath), { recursive: true });
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { encoding: 'utf8', mode: 0o600 });
}

const projectConfig = JSON.parse(await readFile(path.join(projectRoot, 'project.config.json'), 'utf8'));
const appConfig = JSON.parse(await readFile(path.join(projectRoot, 'apps/wechat-miniprogram/app.json'), 'utf8'));
const entryRoute = appConfig.pages[0];
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
    developer_tools: developerToolsVersion(),
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
  // `cli auto` 只是请求开发者工具打开项目窗口，自动化端口要等窗口就绪才监听；
  // 立即连接会必然抢跑。有界重试，耗尽后仍失败才算真正的外部阻断。
  miniProgram = await connectWithRetry();

  if (entryRoute !== 'pages/launch/launch') throw new Error(`configured first route was ${entryRoute || 'missing'}`);
  let page = await miniProgram.reLaunch(`/${entryRoute}`);
  if (!page) throw new Error('launch page did not launch');
  // 冷启动要串行走 session POST、identity GET 再 reLaunch；2 秒窗口在真实
  // 开发者工具里会抢跑，尤其首次编译之后。
  for (let attempt = 0; attempt < 60; attempt += 1) {
    await page.waitFor(250);
    const current = await miniProgram.currentPage();
    if (current && current.path === 'pages/home/home') {
      page = current;
      break;
    }
  }
  if (!page || page.path !== 'pages/home/home') {
    // 停在哪一页、入口状态机停在哪一档，是诊断冷启动分流的最短线索。
    const stuck = await miniProgram.currentPage();
    let entry = 'unavailable';
    try { entry = JSON.stringify((await stuck.data()).entryState || null); } catch { /* 非 launch 页无此字段 */ }
    throw new Error(`unbound cold start stopped at ${stuck ? stuck.path : 'no page'} with entryState=${entry}`);
  }
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

  // P0-6：个人中心布局。只有这一层有真实 wxss 与真实布局引擎，
  // UI0 与 UI1 都测不出「标题逐字换行」和「行未占满卡片」。
  page = await miniProgram.reLaunch('/pages/profile/profile');
  if (!page) throw new Error('profile page did not launch');
  await page.waitFor(600);

  // 我的订单、绑定主手机号、附加手机号、商户登录、联系客服。
  // fixture 的身份是「未绑主手机号、非商户」，五行全部渲染。
  const rows = await page.$$('.prow');
  if (rows.length !== 5) throw new Error(`profile rendered ${rows.length} rows, want 5`);
  const rowBoxes = [];
  for (const row of rows) rowBoxes.push(await measure(row));
  const rowWidths = rowBoxes.map(box => box.width);
  const widestRow = Math.max(...rowWidths);
  for (const width of rowWidths) {
    // 原生 button 的 margin:auto 会让行缩在卡片中部；四行必须等宽。
    if (widestRow - width > 1) throw new Error(`a profile row is ${width}px wide against ${widestRow}px`);
  }

  const collapsedInputs = await page.$$('.extra-in');
  if (collapsedInputs.length !== 0) throw new Error('collapsed extra-phone row rendered inputs');

  const label = requireElement(await page.$('.prow--extra .prow-label'), 'extra phone label');
  const labelBox = await measure(label);
  // 逐字换行的直接判据：标题被压到近零宽，于是高度堆成多行。
  const singleLine = labelBox.height;
  if (labelBox.width < widestRow * 0.5) {
    throw new Error(`extra phone label is ${labelBox.width}px wide inside a ${widestRow}px row`);
  }
  if (singleLine > 30) throw new Error(`extra phone label wrapped to ${singleLine}px tall`);

  const head = requireElement(await page.$('.prow--extra .prow-head'), 'extra phone row head');
  await head.tap();
  await page.waitFor(300);

  const phoneBox = await measure(requireElement(await page.$('.extra-in--phone'), 'extra phone input'));
  const nameBox = await measure(requireElement(await page.$('.extra-in--name'), 'extra name input'));
  const saveBox = await measure(requireElement(await page.$('.extra-save'), 'extra save button'));
  for (const [box, name] of [[phoneBox, 'phone input'], [nameBox, 'name input'], [saveBox, 'save button']]) {
    if (box.width <= 0 || box.height <= 0) throw new Error(`${name} rendered with a zero box`);
    if (box.height < 36) throw new Error(`${name} is only ${box.height}px tall, below the tap target floor`);
  }
  // 5:3 分栏：手机号必须比姓名宽，且两者不得重叠。
  if (phoneBox.width <= nameBox.width) {
    throw new Error(`phone input ${phoneBox.width}px is not wider than name input ${nameBox.width}px`);
  }
  if (phoneBox.left + phoneBox.width > nameBox.left + 1) throw new Error('extra phone inputs overlap');
  // 保存独占全宽行。
  if (widestRow - saveBox.width > 1) throw new Error(`save button is ${saveBox.width}px inside a ${widestRow}px row`);
  if (saveBox.top < phoneBox.top + phoneBox.height) throw new Error('save button shares the input row');

  receipt.scenarios.push({
    id: 'profile-layout-geometry',
    status: 'PASS',
    measurements: {
      row_widths: rowWidths,
      extra_label: labelBox,
      extra_phone_input: phoneBox,
      extra_name_input: nameBox,
      extra_save: saveBox,
    },
  });

  receipt.status = 'PASS';
} catch (error) {
  if (error && error.code === 'BLOCKED_EXTERNAL') {
    receipt.status = 'BLOCKED_EXTERNAL';
    receipt.external_asset = error.externalAsset;
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
