const orderStore = require('../../utils/orderStore.js');
const subscriptionStore = require('../../utils/subscriptionStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav, cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    state: 'loading', orderId: '', o: null, footText: '', viewBtn: '查看订单',
    canSubscribeReady: false, readySubscriptionState: 'idle',
    remainingGroups: 0,
  },
  onLoad(opts) {
    this._id = String((opts && opts.id) || '');
    this.setData({ orderId: this._id, remainingGroups: cart.groups().length });
    return this.load();
  },
  async load() {
    if (this.data.state !== 'loading') this.setData({ state: 'loading' });
    try {
      const order = await orderStore.detail(this._id);
      this.setData({
        state: 'ready', o: order,
        footText: `${dayLabel(order.pickupDate)} ${order.pickupTime} 取 · ${order.pickupPoint}`,
        viewBtn: order.state === 'READY_FOR_PICKUP' ? '查看取餐码' : '查看订单',
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
      const toast = this.selectComponent('#toast');
      if (decision === 'ACCEPTED') toast.show('已开启取餐提醒');
      else if (decision === 'REJECTED') toast.show('未开启提醒，可在订单中查看进度', { icon: 'warn' });
      else toast.show('提醒暂未开启，请重试', { icon: 'warn' });
      this.setData({ readySubscriptionState: decision === 'ACCEPTED' ? 'accepted' : 'idle' });
      return decision === 'ACCEPTED';
    });
  },
  goHome() { nav.tabTo('home'); },
  continueCheckout() {
    const groups = cart.groups();
    if (groups.length === 1) nav.replace('confirm', { group: groups[0].key });
    else nav.tabTo('cart-confirm');
  },
  viewCode() { if (this._id) nav.replace('order-detail', { id: this._id }); },
});
