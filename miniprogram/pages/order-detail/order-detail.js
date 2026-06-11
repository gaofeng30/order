const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { o: null, store: data.STORE, rows: [], flavorsStr: '' },
  onLoad(opts) {
    const orders = getApp().globalData.orders;
    const o = orders.find(x => x.id === opts.id) || orders[0] || data.USER_ORDERS[0];
    const rows = o.items.map(([id, q, p]) => {
      const m = data.itemById(id);
      return { name: m.name, q, p, sub: p * q };
    });
    this.setData({ o, rows, flavorsStr: (o.flavors || []).join(' / ') });
  },
  copy() {
    wx.setClipboardData({
      data: this.data.o.code,
      success: () => this.selectComponent('#toast').show('取餐号 ' + this.data.o.code + ' 已复制', { icon: 'copy' }),
    });
  },
});
