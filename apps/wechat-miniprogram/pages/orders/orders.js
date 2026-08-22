const data = require('../../utils/data.js');
const { nav, itemsSummary } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    tabs: ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款'],
    tab: '全部',
    counts: {},
    list: [],
  },
  onShow() { this.build(); },
  build() {
    const orders = getApp().globalData.orders;
    const counts = { 全部: orders.length };
    this.data.tabs.forEach(t => { if (t !== '全部') counts[t] = orders.filter(o => o.status === t).length; });
    const filtered = this.data.tab === '全部' ? orders : orders.filter(o => o.status === this.data.tab);
    const list = filtered.map(o => ({
      ...o,
      summary: itemsSummary(o.items),
      /* 取餐时间与取餐地点是两条独立事实，各占一行。拼成一串后，
         渲染宽度一变两条信息一起塌陷，而宽度取决于客户填了多长的地址。
         取餐文案现算：§15.6.2 删除了 pickupLabel 字段。 */
      timeText: '预约 ' + data.orderPickupLabel(o) + ' 取餐',
      placeText: o.pickupPoint,
      canCancel: data.canCancelReserve(o),
    }));
    this.setData({ counts, list });
  },
  switchTab(e) { this.setData({ tab: e.currentTarget.dataset.t }, () => this.build()); },
  goDetail(e) { nav.go('order-detail', { id: e.currentTarget.dataset.id }); },
  goMenu() { nav.tabTo('menu'); },
});
