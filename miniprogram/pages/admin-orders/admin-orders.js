const { nav, itemsSummary, advanceOrder, advanceMeta } = require('../../utils/util.js');

const LANES = ['待接单', '制作中', '待取餐', '已完成', '已取消'];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    lanes: LANES,
    lane: '待接单',
    counts: {},
    list: [],
    batch: false,
    selMap: {},
    selCount: 0,
  },
  onLoad(opts) {
    if (opts.lane && LANES.includes(opts.lane)) this.setData({ lane: opts.lane });
  },
  onShow() { this.build(); },
  build() {
    const orders = getApp().globalData.aOrders;
    const counts = {};
    LANES.forEach(l => { counts[l] = orders.filter(o => o.status === l).length; });
    const list = orders.filter(o => o.status === this.data.lane).map(o => {
      const hasFlavor = o.flavor && o.flavor !== '—';
      let band = '';
      if (hasFlavor) band += o.flavor;
      if (o.note) band += (hasFlavor ? ' · ' : '') + o.note;
      return { ...o, summary: itemsSummary(o.items), meta: advanceMeta(o.status), band, showBand: !!(hasFlavor || o.note) };
    });
    this.setData({ counts, list });
  },
  switchLane(e) {
    this.setData({ lane: e.currentTarget.dataset.l, batch: false, selMap: {}, selCount: 0 }, () => this.build());
  },
  toggleBatch() { this.setData({ batch: !this.data.batch, selMap: {}, selCount: 0 }); },
  cardTap(e) {
    const id = e.currentTarget.dataset.id;
    if (this.data.batch && this.data.lane === '待接单') {
      this.toggleSel(id);
    } else {
      getApp().globalData._aSel = getApp().globalData.aOrders.find(o => o.id === id);
      nav.go('admin-order-detail', { id });
    }
  },
  toggleSel(id) {
    const map = Object.assign({}, this.data.selMap);
    if (map[id]) delete map[id]; else map[id] = true;
    this.setData({ selMap: map, selCount: Object.keys(map).length });
  },
  advance(e) {
    advanceOrder(e.currentTarget.dataset.id, this.selectComponent('#toast'), () => this.build());
  },
  viewOrder(e) {
    const id = e.currentTarget.dataset.id;
    getApp().globalData._aSel = getApp().globalData.aOrders.find(o => o.id === id);
    nav.go('admin-order-detail', { id });
  },
  doBatch() {
    const ids = Object.keys(this.data.selMap);
    if (!ids.length) return;
    const toast = this.selectComponent('#toast');
    const g = getApp().globalData;
    ids.forEach(id => { const o = g.aOrders.find(x => x.id === id); if (o && o.status === '待接单') o.status = '制作中'; });
    const n = ids.length;
    this.setData({ selMap: {}, selCount: 0, batch: false }, () => this.build());
    toast.show('已批量接单 ' + n + ' 单', { icon: 'check' });
  },
});
