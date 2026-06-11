const data = require('../../utils/data.js');
const { nav, itemsSummary } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    tabs: ['全部', '待支付', '待取餐', '已完成', '已取消'],
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
      itemCount: o.items.reduce((a, b) => a + b[1], 0),
    }));
    this.setData({ counts, list });
  },
  switchTab(e) { this.setData({ tab: e.currentTarget.dataset.t }, () => this.build()); },
  goDetail(e) { nav.go('order-detail', { id: e.currentTarget.dataset.id }); },
  goPay() { nav.go('confirm'); },
  goMenu() { nav.tabTo('menu'); },
});
