const api = require('./apiClient.js');

async function login(code) {
  const normalized = typeof code === 'string' ? code.trim() : '';
  if (!normalized) throw new api.APIError('MERCHANT_PHONE_REQUIRED');
  const body = await api.intrinsic('/api/v1/me/merchant-login', { code: normalized });
  const merchant = body && body.merchant;
  if (!merchant || merchant.bound !== true || !['OWNER', 'SUBACCOUNT'].includes(merchant.role)) {
    throw new api.APIError('MERCHANT_IDENTITY_UNAVAILABLE');
  }
  return merchant;
}

module.exports = { login };
