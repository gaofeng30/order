const api = require('./apiClient.js');

function parseStatus(body) {
  const value = body && (body.primary_phone || body);
  const bound = value && (value.bound === undefined ? value.primary_phone_bound : value.bound);
  const masked = value && (value.masked_phone === null ? '' : value.masked_phone);
  if (typeof bound !== 'boolean' || (masked !== undefined && typeof masked !== 'string')) {
    throw new api.APIError('PHONE_BINDING_UNAVAILABLE');
  }
  return { bound, maskedPhone: masked || '' };
}

async function status() { return parseStatus(await api.get('/api/v1/me/primary-phone', true)); }

async function bind(code) {
  if (typeof code !== 'string' || !code.trim()) throw new api.APIError('PHONE_CODE_REJECTED');
  const result = parseStatus(await api.intrinsic('/api/v1/me/bind-phone', { code: code.trim() }));
  if (!result.bound) throw new api.APIError('PHONE_CODE_REJECTED');
  return result;
}

async function setExtra(phone, name, key) {
  const body = await api.post('/api/v1/me/extra-phone', { phone, name }, key);
  if (!body || !body.extra_phone || body.extra_phone.set !== true || typeof body.extra_phone.masked_phone !== 'string') {
    throw new api.APIError('EXTRA_PHONE_UNAVAILABLE');
  }
  return { extraPhone: body.extra_phone, pricingIdentity: body.pricing_identity || null };
}

module.exports = { bind, parseStatus, setExtra, status };
