const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    o: null, store: data.STORE, rows: [], flavorsStr: '', showQr: false,
    canCancel: false, ptAddr: '', cancelMin: data.CANCEL_LIMIT_MIN, cancelSheet: false,
  },
  onLoad(opts) {
    const orders = getApp().globalData.orders;
    const o = orders.find(x => x.id === opts.id) || orders[0] || data.USER_ORDERS[0];
    this.build(o);
  },
  build(o) {
    /* 名称取订单自身的快照；m 只用于图片，订单没有固化图片，
       商品不在本地目录时回落占位图（imageph 接受空 item）。 */
    const rows = o.items.map(([id, name, q, p, dp, flavors, note]) => {
      const m = data.itemById(id) || null;
      return { id, m, name, q, p: dp, sub: dp * q, flavors: flavors || [], note: note || '' };
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
    // 一期没有 已取消：已支付订单取消后直接进 退款中，微信确认退款成功才是 已退款
    g.orders = g.orders.map(o => (o.id === id ? Object.assign({}, o, { status: '退款中' }) : o));
    this.setData({ cancelSheet: false });
    this.build(Object.assign({}, this.data.o, { status: '退款中' }));
    this.selectComponent('#toast').show('已发起退款，微信确认后到账', { icon: 'check' });
  },
});
