const api = require('../../utils/apiClient.js');
const orderStore = require('../../utils/orderStore.js');
const subscriptionStore = require('../../utils/subscriptionStore.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    detailState: 'loading', o: null, rows: [], flavorsStr: '', showQr: false, qrToken: '',
    canCancel: false, canSubscribeReady: false, cancelSheet: false, canceling: false, pickupText: '', navTitle: '订单详情',
  },
  async onLoad(opts) { this._id = String(opts.id || ''); return this.load(); },
  async load() {
    this.setData({ detailState: 'loading', o: null, rows: [], showQr: false, qrToken: '' });
    try {
      const order = await orderStore.detail(this._id);
      this.build(order);
      return true;
    } catch (error) {
      this.setData({ detailState: error && error.statusCode === 404 ? 'not_found' : 'error' });
      return false;
    }
  },
  build(order) {
    const showQr = order.state === 'READY_FOR_PICKUP' && !!order.redemptionToken;
    this.setData({
      detailState: 'ready', o: order, rows: order.rows,
      flavorsStr: [...new Set(order.rows.flatMap(row => row.flavors))].join(' / '),
      showQr, qrToken: showQr ? order.redemptionToken : '',
      canCancel: order.available_actions.includes('CANCEL'),
      canSubscribeReady: order.notificationOptions.includes('READY'),
      pickupText: `${order.pickupDate} ${order.pickupTime}`,
      navTitle: showQr ? '取餐码' : '订单详情',
    });
  },
  copy() {
    if (!this.data.o) return;
    wx.setClipboardData({ data: this.data.o.code,
      success: () => this.selectComponent('#toast').show(`取餐号 ${this.data.o.code} 已复制`, { icon: 'copy' }) });
  },
  openCancel() { if (this.data.canCancel) this.setData({ cancelSheet: true }); },
  closeCancel() { this.setData({ cancelSheet: false }); },
  async doCancel() {
    if (!this.data.canCancel || !this.data.o || this.data.canceling) return false;
    this.setData({ canceling: true });
    try {
      await subscriptionStore.requestAndRecord(this.data.o.id, 'REFUND_RESULT');
      const result = await orderStore.cancel(this.data.o.id, api.newIdempotencyKey('cancel'));
      this.setData({ cancelSheet: false, canceling: false });
      this.build(result.order);
      return true;
    } catch (error) {
      this.setData({ cancelSheet: false, canceling: false });
      return false;
    }
  },
  subscribeReady() { return this.subscribe('READY'); },
  subscribeRefund() { return this.subscribe('REFUND_RESULT'); },
  async subscribe(kind) {
    if (!this.data.o) return false;
    return (await subscriptionStore.requestAndRecord(this.data.o.id, kind)) === 'ACCEPTED';
  },
});
