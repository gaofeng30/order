const data = require('../../utils/data.js');
const { nav, orderMode } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    grid: [
      { k: 'now', icon: 'list', cn: '到店点单' },
      { k: 'reserve', icon: 'calendarClock', cn: '预约点餐' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
    ],
  },
  toMenu() { nav.tabTo('menu'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  gridTap(e) {
    const k = e.currentTarget.dataset.k;
    switch (k) {
      case 'now': orderMode.set('now'); nav.tabTo('menu'); break;
      case 'reserve': orderMode.set('reserve'); nav.tabTo('menu'); break;
      case 'orders': nav.tabTo('orders'); break;
      case 'pickup': {
        const t = getApp().globalData.orders.find(o => o.status === '待取餐' || o.status === '已预约');
        if (t) nav.go('order-detail', { id: t.id });
        else this.toast('暂无可取餐订单', 'warn');
        break;
      }
      default: break;
    }
  },
});
