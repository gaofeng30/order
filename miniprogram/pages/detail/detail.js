const data = require('../../utils/data.js');
const { cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { m: null, qty: 0, off: false, czVisible: false, czInit: null },
  onLoad(opts) {
    const m = data.itemById(opts.id) || data.MENU[0];
    this.setData({ m, off: m.status !== 'on' });
    this.refresh();
  },
  onShow() { this.refresh(); },
  refresh() {
    if (!this.data.m) return;
    this.setData({ qty: cart.qty(this.data.m.id) });
  },
  add() { cart.add(this.data.m.id); this.refresh(); },
  sub() { cart.sub(this.data.m.id); this.refresh(); },
  openCustomize() {
    if (this.data.off) return;
    const entry = cart.entry(this.data.m.id);
    this.setData({
      czVisible: true,
      czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null,
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.m.id, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
    this.selectComponent('#toast').show('已加入购物车', { icon: 'cart' });
  },
});
