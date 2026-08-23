const SESSION_PATH = '/api/v1/auth/miniprogram/session';
const SESSION_FIELDS = ['access_token', 'expires_at', 'token_type'];

function unavailable() {
  const error = new Error('session unavailable');
  error.code = 'SESSION_UNAVAILABLE';
  return error;
}

function isFutureUTCDateTime(value) {
  if (typeof value !== 'string') return false;
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?Z$/.exec(value);
  if (!match) return false;
  const parts = match.slice(1, 7).map(Number);
  const date = new Date(Date.UTC(parts[0], parts[1] - 1, parts[2], parts[3], parts[4], parts[5]));
  return date.getUTCFullYear() === parts[0]
    && date.getUTCMonth() === parts[1] - 1
    && date.getUTCDate() === parts[2]
    && date.getUTCHours() === parts[3]
    && date.getUTCMinutes() === parts[4]
    && date.getUTCSeconds() === parts[5]
    && Date.parse(value) > Date.now();
}

function isExactSession(data) {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return false;
  const fields = Object.keys(data).sort();
  if (fields.length !== SESSION_FIELDS.length) return false;
  if (!fields.every((field, index) => field === SESSION_FIELDS[index])) return false;
  return typeof data.access_token === 'string'
    && data.access_token.length > 0
    && data.access_token === data.access_token.trim()
    && data.token_type === 'Bearer'
    && isFutureUTCDateTime(data.expires_at);
}

function getLoginCode() {
  return new Promise((resolve, reject) => {
    wx.login({
      success(login) {
        if (!login || typeof login.code !== 'string' || login.code.trim() === '') {
          reject(unavailable());
          return;
        }
        resolve(login.code);
      },
      fail() { reject(unavailable()); },
    });
  });
}

function exchangeCode(apiBaseUrl, code) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${apiBaseUrl}${SESSION_PATH}`,
      method: 'POST',
      header: { 'content-type': 'application/json' },
      data: { code },
      success(response) {
        if (!response || response.statusCode !== 201 || !isExactSession(response.data)) {
          reject(unavailable());
          return;
        }
        resolve({
          accessToken: response.data.access_token,
          expiresAt: response.data.expires_at,
        });
      },
      fail() { reject(unavailable()); },
    });
  });
}

function createSession(apiBaseUrl) {
  if (typeof apiBaseUrl !== 'string' || apiBaseUrl === '') return Promise.reject(unavailable());
  return getLoginCode().then(code => exchangeCode(apiBaseUrl, code));
}

module.exports = { createSession };
