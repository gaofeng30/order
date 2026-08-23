const api = require('./apiClient.js');

const STATE_LABELS = {
  RESERVED: '已预约', PREPARING: '制作中', READY_FOR_PICKUP: '待取餐',
  COMPLETED: '已完成', REFUNDING: '退款中', REFUNDED: '已退款',
};
const LABEL_STATES = { 已预约: 'RESERVED', 制作中: 'PREPARING', 待取餐: 'READY_FOR_PICKUP', 已完成: 'COMPLETED', 已退款: 'REFUNDED' };

function unavailable() { return new api.APIError('ORDERS_UNAVAILABLE'); }
function parseSummary(value) {
  if (!value || typeof value !== 'object' || typeof value.id !== 'string' || !/^[1-9]\d*$/.test(value.id)
    || typeof value.order_no !== 'string' || !Object.hasOwn(STATE_LABELS, value.state)
    || typeof value.pickup_date !== 'string' || typeof value.pickup_time !== 'string'
    || typeof value.pickup_point !== 'string' || typeof value.pickup_number !== 'string'
    || !Number.isSafeInteger(value.payable_cents) || !Array.isArray(value.available_actions)) throw unavailable();
  return Object.assign({}, value, {
    no: value.order_no, status: STATE_LABELS[value.state], displayState: STATE_LABELS[value.state],
    pickupDate: value.pickup_date, pickupTime: value.pickup_time, pickupPoint: value.pickup_point,
    code: value.pickup_number, total: value.payable_cents,
  });
}
function parseDetail(value) {
  const order = parseSummary(value);
  if (!Array.isArray(value.items)) throw unavailable();
  const rows = value.items.map(item => {
    if (!item || typeof item.name !== 'string' || typeof item.product_id !== 'string'
      || !Number.isSafeInteger(item.quantity) || item.quantity <= 0
      || !Number.isSafeInteger(item.unit_price_cents) || !Number.isSafeInteger(item.line_total_cents)
      || !Array.isArray(item.flavors) || typeof item.note !== 'string') throw unavailable();
    return {
      id: item.product_id, name: item.name, q: item.quantity, p: item.unit_price_cents,
      sub: item.line_total_cents, flavors: item.flavors.slice(), note: item.note,
    };
  });
  return Object.assign(order, {
    rows,
    contactName: value.contact && value.contact.name || '',
    maskedPhone: value.contact && value.contact.masked_phone || '',
    orderNote: value.order_note || '',
    redemptionToken: value.redemption_token || '',
    notificationOptions: Array.isArray(value.notification_options) ? value.notification_options.slice() : [],
  });
}
async function list(label, afterID) {
  const params = ['limit=20'];
  if (afterID) params.push(`after_id=${encodeURIComponent(afterID)}`);
  if (label && label !== '全部') params.push(`state=${LABEL_STATES[label]}`);
  const body = await api.get(`/api/v1/orders?${params.join('&')}`, true);
  if (!body || !Array.isArray(body.orders)) throw unavailable();
  return { orders: body.orders.map(parseSummary), nextAfterID: body.next_after_id || '' };
}
async function detail(id) {
  const body = await api.get(`/api/v1/orders/${id}`, true);
  return parseDetail(body && body.order);
}
async function cancel(id, key) {
  const body = await api.post(`/api/v1/orders/${id}/cancel`, { reason: 'USER_REQUEST' }, key);
  return { order: parseDetail(body && body.order), refund: body && body.refund };
}
async function subscription(id, kind, decision, key) {
  const body = await api.post(`/api/v1/orders/${id}/subscriptions`, { kind, decision }, key);
  if (!body || !body.subscription || body.subscription.kind !== kind || body.subscription.decision !== decision) throw unavailable();
  return body.subscription;
}

module.exports = { LABEL_STATES, STATE_LABELS, cancel, detail, list, parseDetail, parseSummary, subscription };
