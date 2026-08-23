const api = require('./apiClient.js');

function object(value, code) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new api.APIError(code);
  return value;
}
function decimalID(value, code) {
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) throw new api.APIError(code);
  return value;
}
function cents(value, code) {
  if (!Number.isSafeInteger(value) || value < 0) throw new api.APIError(code);
  return value;
}

function parseQuote(body) {
  const quote = object(body && body.quote, 'QUOTE_UNAVAILABLE');
  decimalID(quote.id, 'QUOTE_UNAVAILABLE');
  object(quote.contact, 'QUOTE_UNAVAILABLE');
  object(quote.pickup, 'QUOTE_UNAVAILABLE');
  if (!Array.isArray(quote.items) || typeof quote.expires_at !== 'string') throw new api.APIError('QUOTE_UNAVAILABLE');
  cents(quote.original_subtotal_cents, 'QUOTE_UNAVAILABLE');
  cents(quote.discount_cents, 'QUOTE_UNAVAILABLE');
  cents(quote.payable_cents, 'QUOTE_UNAVAILABLE');
  return quote;
}

function parsePrepayment(body) {
  const prepayment = object(body && body.prepayment, 'PREPAYMENT_UNAVAILABLE');
  decimalID(prepayment.id, 'PREPAYMENT_UNAVAILABLE');
  if (typeof prepayment.state !== 'string' || typeof prepayment.expires_at !== 'string') throw new api.APIError('PREPAYMENT_UNAVAILABLE');
  const payment = object(prepayment.wx_request_payment, 'PREPAYMENT_UNAVAILABLE');
  const required = ['timeStamp', 'nonceStr', 'package', 'signType', 'paySign'];
  if (Object.keys(payment).sort().join(',') !== required.slice().sort().join(',')
    || required.some(field => typeof payment[field] !== 'string' || !payment[field])) {
    throw new api.APIError('PREPAYMENT_UNAVAILABLE');
  }
  return prepayment;
}

function createQuote(data, key) {
  return api.post('/api/v1/quotes', data, key).then(parseQuote);
}
function createPrepayment(quoteID, key) {
  return api.post('/api/v1/orders/prepay', { quote_id: quoteID }, key).then(parsePrepayment);
}
function confirm(prepaymentID, key) {
  return api.post('/api/v1/orders/confirm', { prepayment_id: prepaymentID }, key).then(body => {
    if (body && body.state === 'PENDING') return { state: 'PENDING', orderID: '' };
    if (!body || body.state !== 'ORDER_CREATED') throw new api.APIError('PAYMENT_CONFIRM_UNAVAILABLE');
    return { state: body.state, orderID: decimalID(body.order_id, 'PAYMENT_CONFIRM_UNAVAILABLE') };
  });
}
function requestPayment(payment) {
  return new Promise(resolve => {
    wx.requestPayment(Object.assign({}, payment, {
      success() { resolve(true); },
      fail() { resolve(false); },
    }));
  });
}

module.exports = { confirm, createPrepayment, createQuote, parsePrepayment, parseQuote, requestPayment };
