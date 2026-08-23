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
      'tools/miniprogram-ui/test/browser/ui1-composed-boundaries.spec.cjs',
    ],
    preprocessors: {
      'tools/miniprogram-ui/test/browser/ui1-composed-boundaries.spec.cjs': ['webpack'],
    },
    webpack: {
      mode: 'development',
      devtool: false,
      module: { rules: [{ test: /\.wxml$/, type: 'asset/source' }] },
      plugins: [
        new webpack.DefinePlugin({
          ORDER_BOUNDARIES_PROXY_ORIGIN: JSON.stringify(process.env.ORDER_BOUNDARIES_PROXY_ORIGIN),
          ORDER_BOUNDARIES_RUN_ID: JSON.stringify(process.env.ORDER_BOUNDARIES_RUN_ID),
          ORDER_BOUNDARIES_SETUP: process.env.ORDER_BOUNDARIES_SETUP,
        }),
        new webpack.NormalModuleReplacementPlugin(
          /runtimeEndpointConfig\.js$/,
          path.join(__dirname, 'test/browser/composed-boundaries-runtime-endpoint-config.cjs'),
        ),
      ],
    },
    reporters: ['progress'],
    browsers: ['OrderBoundariesChromiumHeadless'],
    customLaunchers: {
      OrderBoundariesChromiumHeadless: {
        base: 'ChromeHeadless',
        flags: ['--disable-gpu', '--no-first-run'],
      },
    },
    client: { mocha: { timeout: 60000 } },
    browserConsoleLogOptions: { level: 'error', format: '%b %T: %m', terminal: true },
    logLevel: config.LOG_WARN,
    colors: false,
    singleRun: true,
    concurrency: 1,
  });
};
