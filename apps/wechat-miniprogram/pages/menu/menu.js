const data = require('../../utils/data.js');
const catalogStore = require('../../utils/catalogStore.js');
const { nav, cart, pickup } = require('../../utils/util.js');

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
    // 取餐时间（生效 spec: 离散时间点，按餐段分组，已截餐段整组折叠）
    pickup: {},
    pickerVisible: false,
    pickerDates: [],
    pickerGroups: [],
    pickerOff: 0,
  },
  onLoad() { this.refresh(); this.syncPickup(); },
  onShow() { this.refresh(); this.syncPickup(); return this.loadCatalog(); },

  // ---- 取餐时间 ----
  syncPickup() {
    const pk = pickup.get();
    this.setData({ pickup: Object.assign({}, pk, { label: data.pickupLabel(pk) }) });
  },
  buildPicker(off) {
    const dates = data.RESERVE_DATES.map(d => ({
      k: d.k, off: d.off, allCutOff: data.isDateCutOff(d.off),
    }));
    const groups = data.MEAL_PERIODS.map(p => {
      const cutOff = data.isPeriodCutOff(off, p.key);
      return {
        key: p.key,
        name: p.name,
        cutOff,
        // 已截餐段整组折叠：只标注截止时刻，不逐条渲染灰项
        cutoffLabel: cutOff ? `已截单 · ${p.cutoff} 截止` : '',
        times: cutOff ? [] : data.pickupTimes(p.key),
      };
    });
    this.setData({ pickerOff: off, pickerDates: dates, pickerGroups: groups });
  },
  openPicker() {
    this.buildPicker(this.data.pickup.off);
    this.setData({ pickerVisible: true });
  },
  closePicker() { this.setData({ pickerVisible: false }); },
  pickPickerDate(e) {
    const off = +e.currentTarget.dataset.off;
    if (data.isDateCutOff(off)) return this.toast('该日期已截单', 'warn');
    this.buildPicker(off);
  },
  pickPickerTime(e) {
    const { period, t } = e.currentTarget.dataset;
    pickup.set({ off: this.data.pickerOff, period, time: t });
    this.setData({ pickerVisible: false });
    this.syncPickup();
  },
  retryCatalog() { return this.loadCatalog(); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
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
