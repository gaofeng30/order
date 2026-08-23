const SUCCESS = new Set([200, 201, 202, 204]);
const { isRuntimeOrigin } = require('./runtimeEndpoint.js');
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
  if (!state.runtimeEndpoint || state.runtimeEndpoint.state !== 'ready' || !state.apiBaseUrl
    || state.runtimeEndpoint.origin !== state.apiBaseUrl
    || !isRuntimeOrigin(state.runtimeEndpoint.envVersion, state.apiBaseUrl)) {
    throw new APIError('RUNTIME_NOT_READY');
  }
  return state;
}

function validObjectKey(value) {
  if (typeof value !== 'string' || !value || value !== value.trim() || value.length > 1024
    || value.includes('\\') || value.startsWith('/')) return false;
  return value.split('/').every(segment => segment && segment !== '.' && segment !== '..');
}

function resolvePublicURL(value, objectKey) {
  const state = runtime();
  if (typeof value !== 'string' || !value || /[\u0000-\u0020\u007f\\]/.test(value)
    || value.includes('#') || !validObjectKey(objectKey)) {
    throw new APIError('OBJECT_URL_INVALID');
  }

  const localPrefix = '/api/v1/objects/';
  if (value.startsWith('/')) {
    if (!value.startsWith(localPrefix) || value.includes('?')) throw new APIError('OBJECT_URL_INVALID');
    const encoded = value.slice(localPrefix.length);
    if (!encoded) throw new APIError('OBJECT_URL_INVALID');
    let decoded;
    try {
      decoded = encoded.split('/').map(segment => {
        if (!segment) throw new Error('empty object path segment');
        const part = decodeURIComponent(segment);
        if (!part || part === '.' || part === '..' || part.includes('/') || part.includes('\\')) {
          throw new Error('invalid object path segment');
        }
        return part;
      }).join('/');
    } catch (error) {
      throw new APIError('OBJECT_URL_INVALID');
    }
    if (decoded !== objectKey) throw new APIError('OBJECT_URL_INVALID');
    return `${state.apiBaseUrl}${value}`;
  }

  const match = /^https:\/\/([^/?#]+)(?:\/[^#]*)?$/.exec(value);
  if (!match || match[1].includes('@') || match[1].startsWith(':') || match[1].endsWith(':')) {
    throw new APIError('OBJECT_URL_INVALID');
  }
  return value;
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
  resolvePublicURL,
};
