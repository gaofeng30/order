const api = require('./apiClient.js');
const orderStore = require('./orderStore.js');

const LABEL_STATES = orderStore.LABEL_STATES;

async function orders(label, query) {
  const params = [];
  if (query) params.push(`q=${encodeURIComponent(query)}`);
  else if (label) params.push(`state=${LABEL_STATES[label]}`);
  params.push('limit=20');
  const body = await api.get(`/api/v1/merchant/orders?${params.join('&')}`, true);
  if (!body || !Array.isArray(body.orders)) throw new api.APIError('MERCHANT_ORDERS_UNAVAILABLE');
  return body.orders.map(orderStore.parseSummary);
}
async function detail(id) {
  const body = await api.get(`/api/v1/merchant/orders/${id}`, true);
  return orderStore.parseDetail(body && body.order);
}
async function setStoreStatus(status, key) {
  const body = await api.put('/api/v1/merchant/store-status', { status }, key);
  if (!body || body.store_status !== status) throw new api.APIError('STORE_STATUS_UNAVAILABLE');
  return status;
}
async function markReady(id, key) {
  const body = await api.post(`/api/v1/merchant/orders/${id}/ready`, {}, key);
  return orderStore.parseDetail(body && body.order);
}
async function redeem(id, key) {
  const body = await api.post(`/api/v1/merchant/orders/${id}/redeem`, {}, key);
  return orderStore.parseDetail(body && body.order);
}
async function verify(path, data, key) {
  const body = await api.post(path, data, key);
  return orderStore.parseDetail(body && body.order);
}
function verifyScan(token) { return verify('/api/v1/verify/scan', { token }, api.newIdempotencyKey('redeem-scan')); }
function verifyCode(pickupNumber) { return verify('/api/v1/verify/code', { pickup_number: pickupNumber }, api.newIdempotencyKey('redeem-code')); }
async function setSoldOut(id, date, soldOut, key) {
  const body = await api.put(`/api/v1/merchant/products/${id}/soldout`, { service_date: date, sold_out: soldOut }, key);
  if (!body || body.product_id !== id || body.service_date !== date || body.sold_out !== soldOut) {
    throw new api.APIError('SOLDOUT_UNAVAILABLE');
  }
  return body;
}

module.exports = { detail, markReady, orders, redeem, setSoldOut, setStoreStatus, verifyCode, verifyScan };
