const catalogStore = require('../../utils/catalogStore.js');
const { cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { detailState: 'loading', m: null, qty: 0, czVisible: false, czInit: null },
  onLoad(opts) {
    this._id = String(opts.id);
    return this.loadProduct();
  },
  onShow() { this.refresh(); },
  retryProduct() { return this.loadProduct(); },
  async loadProduct() {
    this.setData({ detailState: 'loading', m: null, qty: 0 });
    try {
      const m = catalogStore.withPrice(await catalogStore.loadProduct(this._id));
      this.setData({ detailState: 'ready', m, qty: cart.qty(m.id) });
    } catch (error) {
      this.setData({
        detailState: error && error.code === 'PRODUCT_NOT_FOUND' ? 'not_found' : 'error',
        m: null,
        qty: 0,
      });
    }
  },
  refresh() {
    if (this.data.m) this.setData({ qty: cart.qty(this.data.m.id) });
  },
  add() { cart.add(this.data.m); this.refresh(); },
  sub() { cart.sub(this.data.m.id); this.refresh(); },
  openCustomize() {
    const entry = cart.entry(this.data.m.id);
    this.setData({
      czVisible: true,
      czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null,
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.m, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
    this.selectComponent('#toast').show('已加入购物车', { icon: 'cart' });
  },
});
