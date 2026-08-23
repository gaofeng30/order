const orderStore = require('../../utils/orderStore.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { state: 'loading', o: null, navTitle: '支付结果', mainTitle: '订单已创建', sub: '服务端已确认支付并创建订单' },
  async onLoad(opts) {
    this._id = String(opts.id || '');
    try {
      const order = await orderStore.detail(this._id);
      this.setData({
        state: 'ready', o: order, codeLbl: '取餐号', footText: `${order.pickupDate} ${order.pickupTime} 取 · ${order.pickupPoint}`,
        viewBtn: order.state === 'READY_FOR_PICKUP' ? '查看取餐码' : '查看订单', ringIcon: 'calendar', ringSize: 38,
      });
      return true;
    } catch (error) { this.setData({ state: 'error', o: null }); return false; }
  },
  goHome() { nav.tabTo('home'); },
  viewCode() { if (this.data.o) nav.replace('order-detail', { id: this.data.o.id }); },
});
