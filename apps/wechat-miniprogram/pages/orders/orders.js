const orderStore = require('../../utils/orderStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav } = require('../../utils/util.js');

function card(order) {
  return Object.assign({}, order, {
    timeText: `${dayLabel(order.pickupDate)} ${order.pickupTime} 取餐`, placeText: order.pickupPoint,
    canCancel: order.available_actions.includes('CANCEL'),
  });
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    tabs: ['全部', '已预约', '制作中', '待取餐', '已完成', '已退款'],
    tab: '全部', listState: 'loading', list: [], nextAfterID: '', loadingMore: false, refreshing: false,
  },
  // 回到本页时已有列表则静默刷新，不闪加载态
  onShow() { return this.reload(this.data.listState === 'ready'); },
  // 重新加载当前分类第一页；silent 时保留旧列表，失败只 toast
  async reload(silent) {
    const seq = this._seq = (this._seq || 0) + 1;
    if (!silent) this.setData({ listState: 'loading', list: [], nextAfterID: '', loadingMore: false });
    try {
      const result = await orderStore.list(this.data.tab, '');
      if (seq !== this._seq) return false;
      const list = result.orders.map(card);
      this.setData({ listState: list.length ? 'ready' : 'empty', list, nextAfterID: result.nextAfterID, loadingMore: false });
      return true;
    } catch (error) {
      if (seq !== this._seq) return false;
      if (silent && this.data.listState === 'ready') this.selectComponent('#toast').show('刷新失败，请重试', { icon: 'warn' });
      else this.setData({ listState: 'error', list: [], nextAfterID: '', loadingMore: false });
      return false;
    }
  },
  retry() { return this.reload(false); },
  async onRefresh() {
    this.setData({ refreshing: true });
    try { await this.reload(this.data.listState === 'ready'); }
    finally { this.setData({ refreshing: false }); }
  },
  async loadMore() {
    if (!this.data.nextAfterID || this.data.loadingMore) return false;
    const seq = this._seq;
    this.setData({ loadingMore: true });
    try {
      const result = await orderStore.list(this.data.tab, this.data.nextAfterID);
      if (seq !== this._seq) return false;
      this.setData({ list: this.data.list.concat(result.orders.map(card)), nextAfterID: result.nextAfterID, loadingMore: false });
      return true;
    } catch (error) {
      if (seq !== this._seq) return false;
      this.setData({ loadingMore: false });
      this.selectComponent('#toast').show('加载失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  switchTab(e) {
    const tab = e.currentTarget.dataset.t;
    if (tab === this.data.tab) return false;
    this.setData({ tab });
    return this.reload(false);
  },
  goDetail(e) { nav.go('order-detail', { id: e.currentTarget.dataset.id }); },
  // 卡片「取消预约」：进入详情页并直接弹出取消确认（U43）
  goCancel(e) { nav.go('order-detail', { id: e.currentTarget.dataset.id, cancel: 1 }); },
  goMenu() { nav.tabTo('menu'); },
});
