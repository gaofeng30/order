const path = require('node:path');

const miniprogramRoot = path.resolve(__dirname, '..');

function clone(value) {
  if (value === undefined) return undefined;
  return JSON.parse(JSON.stringify(value));
}

function setPath(target, key, value) {
  const parts = key.split('.');
  let cursor = target;
  for (let index = 0; index < parts.length - 1; index += 1) {
    const part = parts[index];
    if (!cursor[part] || typeof cursor[part] !== 'object') cursor[part] = {};
    cursor = cursor[part];
  }
  cursor[parts[parts.length - 1]] = clone(value);
}

function clearMiniprogramModules() {
  for (const modulePath of Object.keys(require.cache)) {
    if (modulePath.startsWith(miniprogramRoot + path.sep) && modulePath !== __filename) {
      delete require.cache[modulePath];
    }
  }
}

function createHarness(options) {
  clearMiniprogramModules();

  const requestQueue = ((options && options.requests) || []).slice();
  const requestCalls = [];
  const navigationCalls = [];
  const toastCalls = [];
  let appDefinition = null;
  let appInstance = null;
  let pageDefinition = null;

  function completeRequest(call, response) {
    queueMicrotask(() => {
      if (response && response.networkError) {
        call.fail({ errMsg: 'request failed' });
        return;
      }
      call.success(response || { statusCode: 500, data: {} });
    });
  }

  function navigate(type, request) {
    navigationCalls.push({ type, url: request && request.url, delta: request && request.delta });
    if (request && request.success) request.success();
  }

  global.Behavior = definition => definition;
  global.App = definition => { appDefinition = definition; };
  global.Page = definition => { pageDefinition = definition; };
  global.Component = definition => definition;
  global.getApp = () => appInstance;
  global.getCurrentPages = () => [];
  global.wx = {
    request(request) {
      requestCalls.push(request);
      const response = requestQueue.length ? requestQueue.shift() : { networkError: true };
      completeRequest(request, response);
      return { abort() {} };
    },
    getWindowInfo() {
      return {
        statusBarHeight: 20,
        screenWidth: 375,
        screenHeight: 812,
        safeArea: { bottom: 778 },
      };
    },
    getSystemInfoSync() { return this.getWindowInfo(); },
    getMenuButtonBoundingClientRect() { return { top: 26, left: 278, height: 32 }; },
    navigateTo(request) { navigate('navigateTo', request); },
    redirectTo(request) { navigate('redirectTo', request); },
    reLaunch(request) { navigate('reLaunch', request); },
    navigateBack(request) { navigate('navigateBack', request || {}); },
    previewImage() {},
  };

  function loadApp() {
    const appPath = path.join(miniprogramRoot, 'app.js');
    delete require.cache[require.resolve(appPath)];
    require(appPath);
    if (!appDefinition) throw new Error('App was not registered');
    appInstance = { globalData: clone(appDefinition.globalData) };
    for (const [key, value] of Object.entries(appDefinition)) {
      if (key !== 'globalData') appInstance[key] = value;
    }
    if (typeof appInstance.onLaunch === 'function') appInstance.onLaunch();
    return appInstance;
  }

  function loadPage(relativePath) {
    pageDefinition = null;
    const pagePath = path.join(miniprogramRoot, relativePath);
    delete require.cache[require.resolve(pagePath)];
    require(pagePath);
    if (!pageDefinition) throw new Error(`Page was not registered: ${relativePath}`);

    const behaviors = (pageDefinition.behaviors || []).filter(Boolean);
    const behaviorData = Object.assign({}, ...behaviors.map(behavior => clone(behavior.data || {})));
    const page = {
      data: Object.assign({}, behaviorData, clone(pageDefinition.data || {})),
      __definition: pageDefinition,
      __behaviors: behaviors,
      __stateHistory: [],
      setData(patch, callback) {
        for (const [key, value] of Object.entries(patch || {})) setPath(this.data, key, value);
        this.__stateHistory.push(clone(this.data));
        if (callback) callback.call(this);
      },
      createSelectorQuery() {
        const query = {
          select() { return query; },
          selectAll() { return query; },
          boundingClientRect() { return query; },
          exec(callback) { callback([]); },
        };
        return query;
      },
      selectComponent() {
        return { show(message, config) { toastCalls.push({ message, config }); } };
      },
    };

    for (const behavior of behaviors) Object.assign(page, behavior.methods || {});
    for (const [key, value] of Object.entries(pageDefinition)) {
      if (key !== 'data' && key !== 'behaviors') page[key] = value;
    }
    return page;
  }

  function invoke(page, lifecycle, argument) {
    for (const behavior of page.__behaviors) {
      if (typeof behavior[lifecycle] === 'function') behavior[lifecycle].call(page, argument);
    }
    if (typeof page.__definition[lifecycle] === 'function') {
      return page.__definition[lifecycle].call(page, argument);
    }
    return undefined;
  }

  return {
    miniprogramRoot,
    requestCalls,
    navigationCalls,
    toastCalls,
    enqueueRequest(response) { requestQueue.push(response); },
    loadApp,
    loadPage,
    invoke,
    flush(milliseconds) {
      return new Promise(resolve => setTimeout(resolve, milliseconds || 0));
    },
  };
}

module.exports = { createHarness, miniprogramRoot };
