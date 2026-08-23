const SUCCESS = new Set([200, 201, 202, 204]);
let keySequence = 0;

class APIError extends Error {
  constructor(code, statusCode) {
    super(code || 'API_UNAVAILABLE');
    this.name = 'APIError';
    this.code = code || 'API_UNAVAILABLE';
    this.statusCode = statusCode || 0;
  }
}

function runtime() {
  const state = getApp().globalData;
  if (!state.runtimeEndpoint || state.runtimeEndpoint.state !== 'ready' || !state.apiBaseUrl) {
    throw new APIError('RUNTIME_NOT_READY');
  }
  return state;
}

function sessionToken(required) {
  const state = runtime();
  const session = state.session || {};
  if (session.state === 'ready' && session.accessToken) return session.accessToken;
  if (required) throw new APIError('UNAUTHENTICATED', 401);
  return '';
}

function responseError(response) {
  const body = response && response.data;
  const code = body && body.error && typeof body.error.code === 'string'
    ? body.error.code : 'API_UNAVAILABLE';
  return new APIError(code, response && response.statusCode);
}

function request(path, options) {
  const config = options || {};
  const app = getApp();
  if (config.auth === true && app.globalData.session && app.globalData.session.state === 'loading' && app.sessionPromise) {
    return app.sessionPromise.then(() => request(path, config));
  }
  let state;
  let token;
  try {
    state = runtime();
    token = sessionToken(config.auth === true);
  } catch (error) {
    return Promise.reject(error);
  }
  const header = {};
  if (token && config.auth !== false) header.Authorization = `Bearer ${token}`;
  if (Object.hasOwn(config, 'data')) header['content-type'] = 'application/json';
  if (config.idempotencyKey) header['Idempotency-Key'] = config.idempotencyKey;
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${state.apiBaseUrl}${path}`,
      method: config.method || 'GET',
      header,
      data: config.data,
      success(response) {
        if (!response || !SUCCESS.has(response.statusCode)) {
          reject(responseError(response));
          return;
        }
        resolve(response.data || {});
      },
      fail() { reject(new APIError('API_UNAVAILABLE')); },
    });
  });
}

function newIdempotencyKey(scope) {
  if (!wx.getRandomValues) throw new APIError('IDEMPOTENCY_UNAVAILABLE');
  const bytes = new Uint8Array(16);
  wx.getRandomValues(bytes);
  keySequence += 1;
  return `${scope}-${Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')}-${keySequence}`;
}

function write(path, method, data, key) {
  return request(path, { method, data, auth: true, idempotencyKey: key });
}

module.exports = {
  APIError,
  get(path, auth) { return request(path, { auth: auth === true }); },
  getOptional(path) { return request(path, {}); },
  intrinsic(path, data) { return request(path, { method: 'POST', data, auth: true }); },
  newIdempotencyKey,
  post(path, data, key) { return write(path, 'POST', data, key); },
  put(path, data, key) { return write(path, 'PUT', data, key); },
};
