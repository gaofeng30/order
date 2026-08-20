const { nav, itemsSummary, advanceOrder, advanceMeta } = require('../../utils/util.js');

const BIZ = ['营业中', '休息中', '已截单'];
const LANES = ['已预约', '制作中', '待取餐', '已完成', '已退款', '全部'];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    lanes: LANES,
    lane: '已预约',
    biz: BIZ,
    storeStatus: '',
    counts: {},
    list: [],
  },
  // 现场临时休息需要手机可操作，营业状态切换保留在小程序商户端（0818 PRD §6.10、§16.3 P1）
  setBiz(e) {
    const b = e.currentTarget.dataset.b;
    getApp().globalData.store.status = b;
    this.setData({ storeStatus: b });
  },
  reset() { nav.reset(); },
  onLoad(opts) {
    if (opts.lane && LANES.includes(opts.lane)) this.setData({ lane: opts.lane });
  },
  onShow() {
    this.setData({ storeStatus: getApp().globalData.store.status }); this.build(); },
  build() {
    const orders = getApp().globalData.aOrders;
    const counts = {};
    LANES.forEach(l => { counts[l] = l === '全部' ? orders.length : orders.filter(o => o.status === l).length; });
    const lane = this.data.lane;
    const src = lane === '全部' ? orders : orders.filter(o => o.status === lane);
    const list = src.map(o => {
      const hasFlavor = o.flavor && o.flavor !== '—';
      let band = '';
      if (hasFlavor) band += o.flavor;
      if (o.note) band += (hasFlavor ? ' · ' : '') + o.note;
      return { ...o, summary: itemsSummary(o.items), meta: advanceMeta(o.status), band, showBand: !!(hasFlavor || o.note) };
    });
    this.setData({ counts, list });
  },
  switchLane(e) {
    this.setData({ lane: e.currentTarget.dataset.l }, () => this.build());
  },
  cardTap(e) {
    const id = e.currentTarget.dataset.id;
    getApp().globalData._aSel = getApp().globalData.aOrders.find(o => o.id === id);
    nav.go('admin-order-detail', { id });
  },
  advance(e) {
    advanceOrder(e.currentTarget.dataset.id, this.selectComponent('#toast'), () => this.build());
  },
  viewOrder(e) {
    const id = e.currentTarget.dataset.id;
    getApp().globalData._aSel = getApp().globalData.aOrders.find(o => o.id === id);
    nav.go('admin-order-detail', { id });
  },
});
