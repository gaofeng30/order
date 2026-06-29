const data = require('../../utils/data.js');
const { nav, cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    cats: data.CATS,
    groups: [],
    qtyMap: {},       // { [id]: qty }
    count: 0,
    total: 0,
    active: data.CATS[0],
    intoView: '',
    sheet: false,
    cartItems: [],
    _offsets: [],
    // 口味/备注弹层
    czVisible: false,
    czItem: null,
    czInit: null,
    czLabel: '加入购物车',
  },
  onLoad() {
    // 用户端隐藏已下架 (off) 菜品
    const groups = data.CATS.map(c => ({ cat: c, items: data.MENU.filter(m => m.cat === c && m.status !== 'off') }));
    this.setData({ groups });
    this.refresh();
  },
  onShow() { this.refresh(); },
  onReady() { this.measure(); },
  measure() {
    const q = this.createSelectorQuery();
    q.select('#menuScroll').boundingClientRect();
    q.selectAll('.sec-anchor').boundingClientRect();
    q.exec((res) => {
      if (!res || !res[0] || !res[1]) return;
      const base = res[0].top;
      const offsets = res[1].map(r => ({ cat: r.dataset.cat, top: r.top - base }));
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
  add(e) { cart.add(e.currentTarget.dataset.id); this.refresh(); },
  sub(e) { cart.sub(e.currentTarget.dataset.id); this.refresh(); },
  clear() { cart.clear(); this.refresh(); this.setData({ sheet: false }); },
  // 选规格 / 加口味备注 弹层
  openCustomize(e) {
    const id = e.currentTarget.dataset.id;
    this.setData({ czVisible: true, czItem: data.itemById(id), czInit: null, czLabel: '加入购物车' });
  },
  editItem(e) {
    const id = e.currentTarget.dataset.id;
    const entry = cart.entry(id);
    this.setData({
      sheet: false,
      czVisible: true,
      czItem: data.itemById(id),
      czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null,
      czLabel: '保存',
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.czItem.id, e.detail);
    this.setData({ czVisible: false });
    this.refresh();
  },
  jump(e) {
    const c = e.currentTarget.dataset.cat;
    const idx = e.currentTarget.dataset.idx;
    this.setData({ active: c, intoView: 'sec-' + idx });
  },
  onScroll(e) {
    const top = e.detail.scrollTop + 70;
    const offs = this.data._offsets;
    if (!offs.length) return;
    let cur = offs[0].cat;
    for (const o of offs) { if (o.top <= top) cur = o.cat; }
    if (cur !== this.data.active) this.setData({ active: cur });
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
