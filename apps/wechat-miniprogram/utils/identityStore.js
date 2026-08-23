const api = require('./apiClient.js');

function parse(body) {
  const identity = body && body.identity;
  if (!identity || !identity.primary_phone || typeof identity.primary_phone.bound !== 'boolean'
    || !identity.extra_phone || typeof identity.extra_phone.set !== 'boolean'
    || !identity.pricing_identity || typeof identity.pricing_identity.kind !== 'string'
    || !identity.merchant || typeof identity.merchant.bound !== 'boolean') {
    throw new api.APIError('IDENTITY_UNAVAILABLE');
  }
  return identity;
}
async function load() { return parse(await api.get('/api/v1/me/identity', true)); }

module.exports = { load, parse };
