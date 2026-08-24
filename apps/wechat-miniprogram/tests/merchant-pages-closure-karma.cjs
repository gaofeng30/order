const { createRequire } = require('node:module');
const path = require('node:path');

module.exports = function configure(config) {
  const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS;
  if (!dependencyRoot) throw new Error('MINIPROGRAM_UI_DEPS is required');
  const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
  const webpack = dependencyRequire('webpack');
  config.set({
    basePath: path.resolve(__dirname, '../../..'),
    frameworks: ['mocha', 'webpack'],
    files: [
      path.join(dependencyRoot, 'node_modules/miniprogram-simulate/build.js'),
      'apps/wechat-miniprogram/tests/merchant-pages-closure-ui1.spec.cjs',
    ],
    preprocessors: { 'apps/wechat-miniprogram/tests/merchant-pages-closure-ui1.spec.cjs': ['webpack'] },
    webpack: {
      mode: 'development',
      devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_MERCHANT_CLOSURE_ORIGIN: JSON.stringify(process.env.ORDER_MERCHANT_CLOSURE_PROXY_ORIGIN),
          ORDER_MERCHANT_CLOSURE_FIXTURE: process.env.ORDER_MERCHANT_CLOSURE_FIXTURE,
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.join(__dirname, 'merchant-pages-closure-runtime.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderChromiumHeadless'],
    customLaunchers: { OrderChromiumHeadless: { base: 'ChromeHeadless', flags: ['--disable-gpu', '--no-first-run'] } },
    client: { mocha: { timeout: 40000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
