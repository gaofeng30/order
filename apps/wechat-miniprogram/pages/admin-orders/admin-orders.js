const { nav, itemsSummary, advanceOrder, advanceMeta, searchOrders, codeHint } = require('../../utils/util.js');

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
    kw: '',            // 搜索关键词：页面局部查询，不写入 globalData、不进订单模型
    hint: '',          // 跨营业日取餐号提示
  },
  // 现场临时休息需要手机可操作，营业状态切换保留在小程序商户端（0818 PRD §6.10、§16.3 P1）
  setBiz(e) {
    const b = e.currentTarget.dataset.b;
    getApp().globalData.store.status = b;
    this.setData({ storeStatus: b });
  },
  /* 搜索按取餐号 / 订单号 / 手机号定位，跨全部泳道（§6.6）。
     不做「在当前泳道内搜索」：商户用搜索时并不知道单在哪个状态，
     把结果限制在当前泳道等于要求使用者先猜对答案。 */
  onKw(e) {
    this.setData({ kw: (e.detail.value || '').trim() }, () => this.build());
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
    const kw = this.data.kw;
    const lane = this.data.lane;
    const src = kw
      ? searchOrders(kw)
      : (lane === '全部' ? orders : orders.filter(o => o.status === lane));
    const list = src.map(o => {
      /* 整单级口味已删除（§15.6.2）：口味本就绑定在具体菜品上，
         一张单里两个菜要不同口味时整单级字段根本表达不了。聚合行内展示。 */
      const inline = [...new Set(o.items.flatMap(it => [it[5], it[6]]).filter(Boolean).map(String))];
      const band = [...inline, o.orderNote].filter(Boolean).join(' · ');
      return {
        ...o,
        summary: itemsSummary(o.items),
        meta: advanceMeta(o.status),
        band,
        showBand: !!band,
        itemCount: o.items.reduce((a, it) => a + it[2], 0),
        paidTime: String(o.paidAt).slice(11, 16),
      };
    });
    this.setData({ counts, list, hint: kw ? codeHint(kw) : '' });
  },
  // 选择泳道即退出搜索态：两者是互斥的取数方式。
  switchLane(e) {
    this.setData({ lane: e.currentTarget.dataset.l, kw: '', hint: '' }, () => this.build());
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
