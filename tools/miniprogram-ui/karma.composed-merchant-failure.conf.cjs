const { createRequire } = require('node:module');
const path = require('node:path');

module.exports = function configure(config) {
  const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || __dirname;
  const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
  const webpack = dependencyRequire('webpack');
  const fixture = process.env.ORDER_MERCHANT_FAILURE_FIXTURE;
  const origin = process.env.ORDER_MERCHANT_FAILURE_PROXY_ORIGIN;
  if (!fixture || !origin) throw new Error('merchant failure fixture and proxy origin are required');
  config.set({
    basePath: path.resolve(__dirname, '../..'),
    frameworks: ['mocha', 'webpack'],
    files: [
      path.join(dependencyRoot, 'node_modules/miniprogram-simulate/build.js'),
      'tools/miniprogram-ui/test/browser/ui1-composed-merchant-failure.spec.cjs',
    ],
    preprocessors: {
      'tools/miniprogram-ui/test/browser/ui1-composed-merchant-failure.spec.cjs': ['webpack'],
    },
    webpack: {
      mode: 'development', devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_MERCHANT_FAILURE_FIXTURE: fixture,
          ORDER_MERCHANT_FAILURE_ORIGIN: JSON.stringify(origin),
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.join(__dirname, 'test/browser/composed-merchant-failure-runtime-endpoint-config.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderMerchantFailureChromiumHeadless'],
    customLaunchers: {
      OrderMerchantFailureChromiumHeadless: {
        base: 'ChromeHeadless',
        flags: ['--disable-gpu', '--no-first-run'],
      },
    },
    client: { mocha: { timeout: 60000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    browserNoActivityTimeout: 30000,
    singleRun: true,
    concurrency: 1,
  });
};
