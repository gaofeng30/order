const { createRequire } = require('node:module');
const path = require('node:path');

module.exports = function configure(config) {
  const dependencyRoot = process.env.MINIPROGRAM_UI_DEPS || __dirname;
  const dependencyRequire = createRequire(path.join(dependencyRoot, 'package.json'));
  const webpack = dependencyRequire('webpack');
  const proxyOrigin = process.env.ORDER_COMPOSED_PROXY_ORIGIN;
  const paymentExpectation = process.env.ORDER_COMPOSED_PAYMENT_EXPECTATION;
  config.set({
    basePath: path.resolve(__dirname, '../..'),
    frameworks: ['mocha', 'webpack'],
    files: [
      path.join(dependencyRoot, 'node_modules/miniprogram-simulate/build.js'),
      'tools/miniprogram-ui/test/browser/ui1-composed.spec.cjs',
    ],
    preprocessors: {
      'tools/miniprogram-ui/test/browser/ui1-composed.spec.cjs': ['webpack'],
    },
    webpack: {
      mode: 'development',
      devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_COMPOSED_PROXY_ORIGIN: JSON.stringify(proxyOrigin),
          ORDER_COMPOSED_PAYMENT_EXPECTATION: JSON.stringify(paymentExpectation),
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.join(__dirname, 'test/browser/composed-runtime-endpoint-config.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderChromiumHeadless'],
    customLaunchers: {
      OrderChromiumHeadless: {
        base: 'ChromeHeadless',
        flags: ['--disable-gpu', '--no-first-run'],
      },
    },
    client: { mocha: { timeout: 30000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
