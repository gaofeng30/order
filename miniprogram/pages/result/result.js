const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { o: null, store: data.STORE },
  onLoad() {
    this.setData({ o: getApp().globalData.lastOrder || data.USER_ORDERS[0] });
  },
  goHome() { nav.tabTo('home'); },
  viewCode() { nav.replace('order-detail', { id: this.data.o.id }); },
});
