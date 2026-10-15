const productStore = require('../../utils/productStore.js');
const identityStore = require('../../utils/identityStore.js');
const menuStore = require('../../utils/reservationMenuStore.js');
const { nav, cart, pickup } = require('../../utils/util.js');

async function canShowStaffPrice() {
  const session = getApp().globalData.session;
  if (!session || session.state !== 'ready' || !session.accessToken) return false;
  try {
    const identity = await identityStore.load();
    const pricing = identity && identity.pricing_identity;
    return pricing && pricing.kind === 'STAFF' && Number.isSafeInteger(pricing.rate_percent)
      && pricing.rate_percent >= 1 && pricing.rate_percent <= 100;
  } catch (error) {
    return false;
  }
}

function maskStaffPrice(product) {
  if (!product || !product.isStaffPrice) return product;
  return Object.assign({}, product, {
    staff_unit_price_cents: undefined,
    staffUnitPriceCents: undefined,
    isStaffPrice: false,
    price_cents: product.original_unit_price_cents,
    price_text: product.original_price_text,
    staff_price_text: '',
  });
}

// 所选时段整体能否下单（营业状态、截单、餐段），与菜单一致（D7）：
// 菜单页写下的状态与当前取餐时间一致时直接用，否则自己查菜单
async function selectionState(selection) {
  const shared = getApp().globalData.menuState;
  if (shared && shared.date === selection.date && shared.time === selection.time
    && shared.mealPeriod === selection.mealPeriod) {
    return { orderable: shared.orderable, reason: shared.reason };
  }
  try {
    const menu = await menuStore.loadMenu(selection);
    return { orderable: menu.orderable, reason: menuStore.unavailableReason(menu.storeStatus, selection.mealPeriod) };
  } catch (error) {
    return { orderable: false, reason: menuStore.unavailableReason(null, selection.mealPeriod) };
  }
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    detailState: 'loading', m: null, phItem: null, qty: 0, imageIndex: 0, imgFailed: [],
    orderable: false, reason: '', periodLabel: '', czVisible: false, czInit: null, flavors: [],
  },
  onLoad(opts) {
    this._id = String(opts.id || '');
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    return this.loadProduct();
  },
  onShow() { this.refresh(); },
  retryProduct() { return this.loadProduct(); },
  backToMenu() { nav.tabTo('menu'); },
  async loadProduct() {
    const selection = pickup.get();
    if (!selection) {
      this.setData({ detailState: 'selection_required', m: null, qty: 0 });
      return false;
    }
    this.setData({ detailState: 'loading', m: null, qty: 0, imageIndex: 0, imgFailed: [] });
    try {
      const [showStaffPrice, product, menuState] = await Promise.all([
        canShowStaffPrice(), productStore.load(this._id, selection), selectionState(selection),
      ]);
      const visibleProduct = showStaffPrice ? product : maskStaffPrice(product);
      // 购物车里已有这道菜时，用详情读到的最新价格刷新（E3）
      if (cart.entry(visibleProduct.id)) cart.reprice({ [visibleProduct.id]: visibleProduct });
      let reason = '';
      if (visibleProduct.soldOut) reason = '该菜品已售罄';
      else if (!visibleProduct.orderable) reason = '该菜品已下架';
      else if (!menuState.orderable) reason = menuState.reason || menuStore.unavailableReason(null, selection.mealPeriod);
      this.setData({
        detailState: 'ready', m: visibleProduct, phItem: { name: visibleProduct.name },
        qty: cart.qty(visibleProduct.id), imageIndex: 0,
        orderable: !reason, reason, periodLabel: pickup.periodLabel(),
      });
      return true;
    } catch (error) {
      this.setData({
        detailState: error && error.code === 'PRODUCT_NOT_FOUND' ? 'not_found' : 'error',
        m: null, qty: 0,
      });
      return false;
    }
  },
  toast(message, icon) { this.selectComponent('#toast').show(message, { icon: icon || 'check' }); },
  refresh() { if (this.data.m) this.setData({ qty: cart.qty(this.data.m.id) }); },
  add() {
    if (cart.isLocked(pickup.get())) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    if (!this.data.m) return false;
    if (!this.data.orderable) { this.toast(this.data.reason, 'warn'); return false; }
    cart.add(this.data.m);
    this.refresh();
    return true;
  },
  sub() {
    if (cart.isLocked(pickup.get())) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    if (this.data.m) { cart.sub(this.data.m.id); this.refresh(); }
  },
  openCustomize() {
    if (cart.isLocked(pickup.get())) return this.toast('该餐段有待确认支付，请先处理', 'warn');
    if (!this.data.m) return false;
    if (!this.data.orderable) { this.toast(this.data.reason, 'warn'); return false; }
    const entry = cart.entry(this.data.m.id);
    this.setData({ czVisible: true, czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null });
    return true;
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    if (!this.data.m || !this.data.orderable) return false;
    cart.setPrefs(this.data.m, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
    this.toast('已加入购物车');
    return true;
  },
  onImageChange(e) {
    const current = e && e.detail && e.detail.current;
    if (!this.data.m || this.data.m.images.length < 2 || !Number.isSafeInteger(current)
      || current < 0 || current >= this.data.m.images.length) return false;
    this.setData({ imageIndex: current });
    return true;
  },
  // 大图加载失败退回首字占位（D12）
  onImageError(e) {
    const idx = e.currentTarget.dataset.idx;
    if (Number.isSafeInteger(idx)) this.setData({ [`imgFailed[${idx}]`]: true });
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
