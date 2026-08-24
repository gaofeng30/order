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
      'apps/web-admin/tests/ac17-three-client-source-ui1.spec.cjs',
    ],
    preprocessors: { 'apps/web-admin/tests/ac17-three-client-source-ui1.spec.cjs': ['webpack'] },
    webpack: {
      mode: 'development',
      devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_AC17_PROXY_ORIGIN: JSON.stringify(process.env.ORDER_AC17_PROXY_ORIGIN),
          ORDER_AC17_SETUP: process.env.ORDER_AC17_SETUP,
          ORDER_USER_PAGES_PROXY_ORIGIN: JSON.stringify(process.env.ORDER_AC17_PROXY_ORIGIN),
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.resolve(__dirname, '../../../tools/miniprogram-ui/test/browser/composed-user-pages-runtime-endpoint-config.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderAC17ChromiumHeadless'],
    customLaunchers: {
      OrderAC17ChromiumHeadless: {
        base: 'ChromeHeadless',
        flags: ['--disable-gpu', '--no-first-run'],
      },
    },
    client: { mocha: { timeout: 40000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
