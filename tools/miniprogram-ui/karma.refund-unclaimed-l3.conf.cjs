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
      'tools/miniprogram-ui/test/browser/ui1-refund-unclaimed-l3.spec.cjs',
    ],
    preprocessors: {
      'tools/miniprogram-ui/test/browser/ui1-refund-unclaimed-l3.spec.cjs': ['webpack'],
    },
    webpack: {
      mode: 'development',
      devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_REFUND_UNCLAIMED_L3_PROXY_ORIGIN: JSON.stringify(process.env.ORDER_REFUND_UNCLAIMED_L3_PROXY_ORIGIN),
          ORDER_REFUND_UNCLAIMED_L3_FIXTURE: process.env.ORDER_REFUND_UNCLAIMED_L3_FIXTURE,
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.join(__dirname, 'test/browser/refund-unclaimed-l3-runtime-endpoint-config.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderRefundUnclaimedL3ChromiumHeadless'],
    customLaunchers: {
      OrderRefundUnclaimedL3ChromiumHeadless: {
        base: 'ChromeHeadless',
        flags: ['--disable-gpu', '--no-first-run'],
      },
    },
    client: { mocha: { timeout: 120000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
