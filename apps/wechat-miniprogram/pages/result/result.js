const orderStore = require('../../utils/orderStore.js');
const subscriptionStore = require('../../utils/subscriptionStore.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    state: 'loading', o: null, navTitle: '支付结果', mainTitle: '订单已创建', sub: '服务端已确认支付并创建订单',
    canSubscribeReady: false, readySubscriptionState: 'idle',
  },
  async onLoad(opts) {
    this._id = String(opts.id || '');
    try {
      const order = await orderStore.detail(this._id);
      this.setData({
        state: 'ready', o: order, codeLbl: '取餐号', footText: `${order.pickupDate} ${order.pickupTime} 取 · ${order.pickupPoint}`,
        viewBtn: order.state === 'READY_FOR_PICKUP' ? '查看取餐码' : '查看订单', ringIcon: 'calendar', ringSize: 38,
        canSubscribeReady: order.state === 'RESERVED' || order.state === 'PREPARING',
      });
      return true;
    } catch (error) { this.setData({ state: 'error', o: null }); return false; }
  },
  requestReadySubscription() {
    if (!this.data.canSubscribeReady || !this.data.o
      || this.data.readySubscriptionState === 'requesting' || this.data.readySubscriptionState === 'accepted') return Promise.resolve(false);
    this.setData({ readySubscriptionState: 'requesting' });
    return subscriptionStore.requestAndRecord(this.data.o.id, 'READY').then(decision => {
      this.setData({ readySubscriptionState: decision === 'ACCEPTED' ? 'accepted' : decision === 'REJECTED' ? 'rejected' : 'error' });
      return decision === 'ACCEPTED';
    });
  },
  goHome() { nav.tabTo('home'); },
  viewCode() { if (this.data.o) nav.replace('order-detail', { id: this.data.o.id }); },
});
