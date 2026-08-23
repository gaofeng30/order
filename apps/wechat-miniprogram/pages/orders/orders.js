const orderStore = require('../../utils/orderStore.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    tabs: ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款'],
    tab: '全部', listState: 'loading', list: [], nextAfterID: '', loadingMore: false,
  },
  onShow() { return this.load(true); },
  async load(reset) {
    if (reset) this.setData({ listState: 'loading', list: [], nextAfterID: '' });
    try {
      const result = await orderStore.list(this.data.tab, reset ? '' : this.data.nextAfterID);
      const list = (reset ? [] : this.data.list).concat(result.orders.map(order => Object.assign({}, order, {
        summary: '', timeText: `预约 ${order.pickupDate} ${order.pickupTime} 取餐`, placeText: order.pickupPoint,
        canCancel: order.available_actions.includes('CANCEL'),
      })));
      this.setData({ listState: list.length ? 'ready' : 'empty', list, nextAfterID: result.nextAfterID, loadingMore: false });
      return true;
    } catch (error) {
      this.setData({ listState: 'error', list: [], nextAfterID: '', loadingMore: false });
      return false;
    }
  },
  async loadMore() {
    if (!this.data.nextAfterID || this.data.loadingMore) return false;
    this.setData({ loadingMore: true });
    return this.load(false);
  },
  switchTab(e) { this.setData({ tab: e.currentTarget.dataset.t }); return this.load(true); },
  goDetail(e) { nav.go('order-detail', { id: e.currentTarget.dataset.id }); },
  goMenu() { nav.tabTo('menu'); },
});
