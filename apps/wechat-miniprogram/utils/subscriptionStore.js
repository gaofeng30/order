const api = require('./apiClient.js');
const orderStore = require('./orderStore.js');

const KINDS = new Set(['READY', 'REFUND_RESULT']);

async function requestAndRecord(orderID, kind) {
  try {
    if (typeof orderID !== 'string' || !/^[1-9]\d*$/.test(orderID) || !KINDS.has(kind)) return false;
    const ids = getApp().globalData.subscriptionTemplateIds || {};
    const templateID = ids[kind];
    if (typeof templateID !== 'string' || !templateID || typeof wx.requestSubscribeMessage !== 'function') return false;
    const decision = await new Promise(resolve => {
      wx.requestSubscribeMessage({
        tmplIds: [templateID],
        success(result) { resolve(result && result[templateID] === 'accept' ? 'ACCEPTED' : 'REJECTED'); },
        fail() { resolve(''); },
      });
    });
    if (!decision) return false;
    await orderStore.subscription(orderID, kind, decision, api.newIdempotencyKey('subscription'));
    return decision;
  } catch (error) {
    return false;
  }
}

module.exports = { requestAndRecord };
