const productStore = require('../../utils/productStore.js');
const { cart, pickup } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { detailState: 'loading', m: null, qty: 0, imageIndex: 0, czVisible: false, czInit: null, flavors: [] },
  onLoad(opts) {
    this._id = String(opts.id || '');
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    return this.loadProduct();
  },
  onShow() { this.refresh(); },
  retryProduct() { return this.loadProduct(); },
  async loadProduct() {
    const selection = pickup.get();
    if (!selection) {
      this.setData({ detailState: 'selection_required', m: null, qty: 0 });
      return false;
    }
    this.setData({ detailState: 'loading', m: null, qty: 0, imageIndex: 0 });
    try {
      const product = await productStore.load(this._id, selection);
      this.setData({ detailState: 'ready', m: product, qty: cart.qty(product.id), imageIndex: 0 });
      return true;
    } catch (error) {
      this.setData({
        detailState: error && error.code === 'PRODUCT_NOT_FOUND' ? 'not_found' : 'error',
        m: null, qty: 0,
      });
      return false;
    }
  },
  refresh() { if (this.data.m) this.setData({ qty: cart.qty(this.data.m.id) }); },
  add() { if (this.data.m && this.data.m.orderable) { cart.add(this.data.m); this.refresh(); return true; } return false; },
  sub() { if (this.data.m) { cart.sub(this.data.m.id); this.refresh(); } },
  openCustomize() {
    if (!this.data.m || !this.data.m.orderable) {
      this.selectComponent('#toast').show('当前不可下单', { icon: 'warn' });
      return false;
    }
    const entry = cart.entry(this.data.m.id);
    this.setData({ czVisible: true, czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null });
    return true;
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    if (!this.data.m || !this.data.m.orderable) return false;
    cart.setPrefs(this.data.m, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
    this.selectComponent('#toast').show('已加入购物车', { icon: 'cart' });
    return true;
  },
  onImageChange(e) {
    const current = e && e.detail && e.detail.current;
    if (!this.data.m || this.data.m.images.length < 2 || !Number.isSafeInteger(current)
      || current < 0 || current >= this.data.m.images.length) return false;
    this.setData({ imageIndex: current });
    return true;
  },
  previewImage(e) {
    if (!this.data.m || !this.data.m.images.length) return false;
    const urls = this.data.m.images.map(image => image.url);
    const current = e.currentTarget.dataset.url;
    if (!urls.includes(current)) return false;
    wx.previewImage({ current, urls });
    return true;
  },
});
