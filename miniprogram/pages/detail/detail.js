const data = require('../../utils/data.js');
const { cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { m: null, qty: 0, off: false },
  onLoad(opts) {
    const m = data.itemById(opts.id) || data.MENU[0];
    this.setData({ m, off: m.status !== 'on' });
    this.refresh();
  },
  onShow() { this.refresh(); },
  refresh() {
    if (!this.data.m) return;
    this.setData({ qty: getApp().globalData.cart[this.data.m.id] || 0 });
  },
  add() { cart.add(this.data.m.id); this.refresh(); },
  sub() { cart.sub(this.data.m.id); this.refresh(); },
  addToCart() {
    if (this.data.off) return;
    cart.add(this.data.m.id);
    this.refresh();
    this.selectComponent('#toast').show('已加入购物车', { icon: 'cart' });
  },
});
