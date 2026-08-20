const { nav } = require('../../utils/util.js');
const data = require('../../utils/data.js');

const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    pend: 0,
    nick: data.ME.nick,
    phoneMask: maskPhone(data.ME.phone),
  },
  onShow() {
    this.setData({ pend: getApp().globalData.orders.filter(o => o.status === '待取餐' || o.status === '已预约').length });
  },
  toOrders() { nav.tabTo('orders'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  service() { this.toast('正在拨打 0596-388 1688', 'phone'); },
  settings() { this.toast('设置建设中', 'settings'); },
  reset() { nav.reset(); },
});
