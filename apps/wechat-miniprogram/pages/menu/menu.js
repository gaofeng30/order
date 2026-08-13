const data = require('../../utils/data.js');
const catalogStore = require('../../utils/catalogStore.js');
const { nav, cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    listState: 'loading',
    cats: [],
    groups: [],
    qtyMap: {},
    count: 0,
    total: 0,
    active: '',
    intoView: '',
    sheet: false,
    cartItems: [],
    _offsets: [],
    czVisible: false,
    czItem: null,
    czInit: null,
    czLabel: '加入购物车',
  },
  onLoad() { this.refresh(); },
  onShow() { this.refresh(); return this.loadCatalog(); },
  retryCatalog() { return this.loadCatalog(); },
  async loadCatalog() {
    this.setData({ listState: 'loading', cats: [], groups: [], active: '' });
    try {
      const catalog = await catalogStore.loadCatalog();
      const groups = catalog.categories.map(category => ({
        id: category.id,
        name: category.name,
        products: category.products.map(catalogStore.withPrice),
      }));
      this._productsById = {};
      groups.forEach(group => group.products.forEach(product => { this._productsById[product.id] = product; }));
      this.setData({
        listState: groups.length ? 'ready' : 'empty',
        cats: groups.map(group => ({ id: group.id, name: group.name })),
        groups,
        active: groups.length ? groups[0].id : '',
      }, () => this.measure());
    } catch (error) {
      this._productsById = {};
      this.setData({ listState: 'error', cats: [], groups: [], active: '' });
    }
  },
  onReady() { this.measure(); },
  measure() {
    const query = this.createSelectorQuery();
    query.select('#menuScroll').boundingClientRect();
    query.selectAll('.sec-anchor').boundingClientRect();
    query.exec((result) => {
      if (!result || !result[0] || !result[1]) return;
      const base = result[0].top;
      const offsets = result[1].map(row => ({ id: row.dataset.id, top: row.top - base }));
      this.setData({ _offsets: offsets });
    });
  },
  refresh() {
    const raw = getApp().globalData.cart;
    const qtyMap = {};
    Object.keys(raw).forEach(id => { qtyMap[id] = raw[id].qty; });
    this.setData({
      qtyMap,
      count: cart.count(),
      total: cart.total(),
      cartItems: cart.list(),
    });
  },
  add(e) {
    const id = e.currentTarget.dataset.id;
    const entry = cart.entry(id);
    const product = entry ? entry.product : this._productsById[id];
    if (!product) return;
    cart.add(product);
    this.refresh();
  },
  sub(e) { cart.sub(e.currentTarget.dataset.id); this.refresh(); },
  clear() { cart.clear(); this.refresh(); this.setData({ sheet: false }); },
  openCustomize(e) {
    const id = e.currentTarget.dataset.id;
    this.setData({ czVisible: true, czItem: this._productsById[id], czInit: null, czLabel: '加入购物车' });
  },
  editItem(e) {
    const id = e.currentTarget.dataset.id;
    const entry = cart.entry(id);
    this.setData({
      sheet: false,
      czVisible: true,
      czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note },
      czLabel: '保存',
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.czItem, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
  },
  jump(e) {
    const id = e.currentTarget.dataset.id;
    this.setData({ active: id, intoView: `sec-${e.currentTarget.dataset.idx}` });
  },
  onScroll(e) {
    const top = e.detail.scrollTop + 70;
    const offsets = this.data._offsets;
    if (!offsets.length) return;
    let current = offsets[0].id;
    for (const offset of offsets) { if (offset.top <= top) current = offset.id; }
    if (current !== this.data.active) this.setData({ active: current });
  },
  goDetail(e) { nav.go('detail', { id: e.currentTarget.dataset.id }); },
  openSheet() { if (this.data.count) this.setData({ sheet: true }); },
  closeSheet() { this.setData({ sheet: false }); },
  goConfirm() {
    if (!this.data.count) return;
    this.setData({ sheet: false });
    nav.go('confirm');
  },
  noop() {},
});
