const data = require('../../utils/data.js');
const { nav, cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    cats: data.CATS,
    groups: [],
    cartMap: {},
    count: 0,
    total: 0,
    active: data.CATS[0],
    intoView: '',
    sheet: false,
    cartItems: [],
    _offsets: [],
  },
  onLoad() {
    const groups = data.CATS.map(c => ({ cat: c, items: data.MENU.filter(m => m.cat === c) }));
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
    this.setData({
      cartMap: Object.assign({}, getApp().globalData.cart),
      count: cart.count(),
      total: cart.total(),
      cartItems: cart.list(),
    });
  },
  add(e) { cart.add(e.currentTarget.dataset.id); this.refresh(); },
  sub(e) { cart.sub(e.currentTarget.dataset.id); this.refresh(); },
  clear() { cart.clear(); this.refresh(); this.setData({ sheet: false }); },
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
