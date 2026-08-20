const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    o: null, store: data.STORE, reserve: false, rows: [], flavorsStr: '',
    canCancel: false, ptAddr: '', cancelMin: data.CANCEL_LIMIT_MIN, cancelSheet: false,
  },
  onLoad(opts) {
    const orders = getApp().globalData.orders;
    const o = orders.find(x => x.id === opts.id) || orders[0] || data.USER_ORDERS[0];
    this.build(o);
  },
  build(o) {
    const rows = o.items.map(([id, q, p, flavors, note]) => {
      const m = data.itemById(id);
      return { id, m, name: m.name, q, p, sub: p * q, flavors: flavors || [], note: note || '' };
    });
    const pt = data.PICKUP_POINTS.find(x => x.name === o.pickupPoint);
    this.setData({
      o, rows,
      showQr: o.status === '待取餐',
      flavorsStr: (o.flavors || []).join(' / '),
      canCancel: data.canCancelReserve(o),
      ptAddr: pt ? pt.addr : data.STORE.addr,
      navTitle: o.status === '待取餐' ? '取餐码' : '预约详情',
    });
  },
  copy() {
    wx.setClipboardData({
      data: this.data.o.code,
      success: () => this.selectComponent('#toast').show('取餐号 ' + this.data.o.code + ' 已复制', { icon: 'copy' }),
    });
  },
  openCancel() { if (this.data.canCancel) this.setData({ cancelSheet: true }); },
  closeCancel() { this.setData({ cancelSheet: false }); },
  doCancel() {
    const g = getApp().globalData;
    const id = this.data.o.id;
    g.orders = g.orders.map(o => (o.id === id ? Object.assign({}, o, { status: '已取消' }) : o));
    this.setData({ cancelSheet: false });
    this.build(Object.assign({}, this.data.o, { status: '已取消' }));
    this.selectComponent('#toast').show('预约已取消，款项原路退回', { icon: 'check' });
  },
});
