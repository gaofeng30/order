const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const { nav } = require('../../utils/util.js');

const LANES = ['已预约', '制作中', '待取餐', '已完成', '已退款'];
const BIZ = [{ code: 'open', label: '营业中' }, { code: 'closed', label: '休息中' }, { code: 'cutoff', label: '已截单' }];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { lanes: LANES, lane: '已预约', biz: BIZ, storeStatus: '', listState: 'loading', list: [], kw: '', hint: '' },
  reset() { nav.reset(); },
  onLoad(opts) { if (opts && LANES.includes(opts.lane)) this.setData({ lane: opts.lane }); },
  onShow() { return this.load(); },
  async load() {
    this.setData({ listState: 'loading', list: [] });
    try {
      const orders = await merchantStore.orders(this.data.lane, this.data.kw);
      this.setData({ listState: orders.length ? 'ready' : 'empty', list: orders.map(order => Object.assign({}, order, {
        itemCount: '', paidTime: String(order.materialized_at || '').slice(11, 16), summary: '',
        meta: { isView: !order.available_actions.includes('READY'), label: order.available_actions.includes('READY') ? '备好' : '查看' },
      })) });
      return true;
    } catch (error) { this.setData({ listState: 'error', list: [] }); return false; }
  },
  onKw(e) { this.setData({ kw: (e.detail.value || '').trim() }); return this.load(); },
  switchLane(e) { this.setData({ lane: e.currentTarget.dataset.l, kw: '', hint: '' }); return this.load(); },
  async setBiz(e) {
    const status = e.currentTarget.dataset.b;
    if (!BIZ.some(item => item.code === status)) return false;
    try {
      await merchantStore.setStoreStatus(status, api.newIdempotencyKey('store-status'));
      this.setData({ storeStatus: status });
      return true;
    } catch (error) { return false; }
  },
  cardTap(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
  viewOrder(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
  advance(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
});
