const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { pend: 0 },
  onShow() {
    this.setData({ pend: getApp().globalData.orders.filter(o => o.status === '待取餐').length });
  },
  toOrders() { nav.tabTo('orders'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  service() { this.toast('正在拨打 0596-388 1688', 'phone'); },
  settings() { this.toast('设置建设中', 'settings'); },
  reset() { nav.reset(); },
});
