const data = require('../../utils/data.js');
const { cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { m: null, qty: 0, off: false, czVisible: false, czInit: null, imgs: [], gi: 0 },
  onLoad(opts) {
    this._id = opts.id;
    this.reload();
  },
  onShow() { this.reload(); },
  // 商户端可能刚改过菜品或换过图，每次进入都重新取
  reload() {
    const m = data.itemById(this._id) || data.menuList()[0];
    if (!m) return;
    const imgs = (m.imgs && m.imgs.length) ? m.imgs : (m.img ? [m.img] : []);
    const gi = Math.min(this.data.gi, Math.max(0, imgs.length - 1));
    this.setData({ m, off: m.status !== 'on', imgs, gi });
    this.refresh();
  },
  refresh() {
    if (!this.data.m) return;
    this.setData({ qty: cart.qty(this.data.m.id) });
  },
  onGal(e) { this.setData({ gi: e.detail.current }); },
  preview(e) {
    if (!this.data.imgs.length) return;
    wx.previewImage({ current: this.data.imgs[+e.currentTarget.dataset.i], urls: this.data.imgs });
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
