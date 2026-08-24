const { createRequire } = require('node:module');
const path = require('node:path');

module.exports = function configure(config) {
  const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || __dirname;
  const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
  const webpack = dependencyRequire('webpack');
  config.set({
    basePath: path.resolve(__dirname, '../..'),
    frameworks: ['mocha', 'webpack'],
    files: [
      path.join(dependencyRoot, 'node_modules/miniprogram-simulate/build.js'),
      'tools/miniprogram-ui/test/browser/ui1-staff-profile-static.spec.cjs',
    ],
    preprocessors: {
      'tools/miniprogram-ui/test/browser/ui1-staff-profile-static.spec.cjs': ['webpack'],
    },
    webpack: { mode: 'development', devtool: false, module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] } },
    reporters: ['progress'],
    browsers: ['OrderStaffProfileChromiumHeadless'],
    customLaunchers: {
      OrderStaffProfileChromiumHeadless: { base: 'ChromeHeadless', flags: ['--disable-gpu', '--no-first-run'] },
    },
    client: { mocha: { timeout: 10000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
