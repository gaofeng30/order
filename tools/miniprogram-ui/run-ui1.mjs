import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { startCatalogFixture } from './fixture-server.mjs';

const toolRoot = path.dirname(fileURLToPath(import.meta.url));
const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || toolRoot;
const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
const karma = dependencyRequire('karma');
const { chromium } = dependencyRequire('playwright');
const browserPath = chromium.executablePath();

if (!existsSync(browserPath)) {
  throw new Error('locked Chromium is missing; run npm --prefix tools/miniprogram-ui run browser:install');
}

process.env.CHROME_BIN = browserPath;
const browserVersion = execFileSync(browserPath, ['--version'], { encoding: 'utf8' }).trim();

console.log('UI1_ENV', JSON.stringify({
  runner: 'order-miniprogram-ui-gates@1.0.0',
  simulator: 'miniprogram-simulate@1.6.2',
  browser: browserVersion,
}));

const fixture = await startCatalogFixture();
let exitCode;
try {
  const processedConfig = await karma.config.parseConfig(
    path.join(toolRoot, 'karma.conf.cjs'),
    { singleRun: true },
    { promiseConfig: true, throwErrors: true },
  );
  exitCode = await new Promise((resolve, reject) => {
    const server = new karma.Server(processedConfig, resolve);
    server.start().catch(reject);
  });
} finally {
  await fixture.close();
}

console.log('UI1_RESULT', JSON.stringify({
  status: exitCode === 0 ? 'PASS' : 'FAIL',
  scenarios: 3,
}));
if (exitCode !== 0) process.exitCode = exitCode;
