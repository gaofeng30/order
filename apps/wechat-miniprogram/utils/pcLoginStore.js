const api = require('./apiClient.js');

function invalidQR() { return new api.APIError('PC_LOGIN_QR_INVALID'); }

function parseQR(payload) {
  if (typeof payload !== 'string' || payload.length > 256) throw invalidQR();
  const match = /^order-admin-login:\/\/approve\?([^#]+)$/.exec(payload);
  if (!match) throw invalidQR();
  const pairs = match[1].split('&');
  if (pairs.length !== 2) throw invalidQR();
  const values = {};
  pairs.forEach(pair => {
    const parts = pair.split('=');
    if (parts.length !== 2) throw invalidQR();
    let key;
    let value;
    try { key = decodeURIComponent(parts[0]); value = decodeURIComponent(parts[1]); }
    catch (error) { throw invalidQR(); }
    if ((key !== 'login_id' && key !== 'approval_secret')
      || Object.prototype.hasOwnProperty.call(values, key)) throw invalidQR();
    values[key] = value;
  });
  const loginID = values.login_id;
  if (!/^[1-9]\d{0,19}$/.test(loginID || '')
    || (loginID.length === 20 && loginID > '18446744073709551615')
    || !/^[A-Za-z0-9_-]{43}$/.test(values.approval_secret || '')) throw invalidQR();
  return { loginID, approvalSecret: values.approval_secret };
}

async function approve(challenge, code) {
  if (!challenge || typeof code !== 'string' || !code) throw new api.APIError('MERCHANT_PHONE_REQUIRED');
  const body = await api.intrinsic('/api/v1/me/admin-login/approve', {
    login_id: challenge.loginID,
    approval_secret: challenge.approvalSecret,
    code,
  });
  if (!body || body.approved !== true) throw new api.APIError('PC_LOGIN_UNAVAILABLE');
  return true;
}

module.exports = { parseQR, approve };
