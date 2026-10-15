const api = require('./apiClient.js');

function parseStatus(body) {
  const value = body && (body.primary_phone || body);
  const bound = value && (value.bound === undefined ? value.primary_phone_bound : value.bound);
  const phone = value && value.phone;
  if (typeof bound !== 'boolean' || (bound && (typeof phone !== 'string' || !/^\+[1-9]\d{1,14}$/.test(phone)))) {
    throw new api.APIError('PHONE_BINDING_UNAVAILABLE');
  }
  return { bound, phone: bound ? displayPhone(phone) : '' };
}

async function status() { return parseStatus(await api.get('/api/v1/me/primary-phone', true)); }

async function bind(code) {
  if (typeof code !== 'string' || !code.trim()) throw new api.APIError('PHONE_CODE_REJECTED');
  const result = parseStatus(await api.intrinsic('/api/v1/me/bind-phone', { code: code.trim() }));
  if (!result.bound) throw new api.APIError('PHONE_CODE_REJECTED');
  return result;
}

async function setExtra(phone, key) {
  const body = await api.post('/api/v1/me/extra-phone', { phone }, key);
  if (!body || !body.extra_phone || body.extra_phone.set !== true || typeof body.extra_phone.phone !== 'string') {
    throw new api.APIError('EXTRA_PHONE_UNAVAILABLE');
  }
  return { extraPhone: body.extra_phone };
}

function displayPhone(phone) { return /^\+861\d{10}$/.test(phone || '') ? phone.slice(3) : phone || ''; }
module.exports = { bind, parseStatus, setExtra, status, displayPhone };
