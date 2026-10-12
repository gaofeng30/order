const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const identityStore = require('../../utils/identityStore.js');
const { nav } = require('../../utils/util.js');

const LANES = ['已预约', '制作中', '待取餐', '已完成', '已退款'];
const BIZ = [{ code: 'open', label: '营业中' }, { code: 'closed', label: '休息中' }, { code: 'cutoff', label: '已截单' }];

function beijingTime(value) {
  const instant = Date.parse(value);
  if (!Number.isFinite(instant)) return '';
  return new Date(instant + 8 * 60 * 60 * 1000).toISOString().slice(11, 16);
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    lanes: LANES, lane: '已预约', biz: BIZ, storeStatus: '', statusWriteState: 'idle',
    listState: 'loading', list: [], kw: '', hint: '', merchantRole: '',
  },
  reset() { nav.reset(); },
  async goPCLogin() {
    try {
      const identity = await identityStore.load();
      const owner = identity.merchant.bound && identity.merchant.role === 'OWNER';
      this.setData({ merchantRole: identity.merchant.bound ? identity.merchant.role : '' });
      if (!owner) return false;
      nav.go('admin-login');
      return true;
    } catch (error) {
      this.setData({ merchantRole: '' });
      return false;
    }
  },
  onLoad(opts) { if (opts && LANES.includes(opts.lane)) this.setData({ lane: opts.lane }); },
  onShow() { return this.load(); },
  onUnload() { this._loadGeneration = (this._loadGeneration || 0) + 1; },
  async load() {
    const generation = (this._loadGeneration || 0) + 1;
    this._loadGeneration = generation;
    this.setData({ listState: 'loading', list: [] });
    try {
      const [orders, storefront, identity] = await Promise.all([
        merchantStore.orders(this.data.lane, this.data.kw),
        storefrontStore.load(),
        identityStore.load(),
      ]);
      if (generation !== this._loadGeneration) return false;
      this.setData({
        storeStatus: storefront.businessStatus,
        merchantRole: identity.merchant.bound ? identity.merchant.role : '',
        listState: orders.length ? 'ready' : 'empty', list: orders.map(order => Object.assign({}, order, {
        paidTime: beijingTime(order.materialized_at || ''),
        meta: { isView: !order.available_actions.includes('READY'), label: order.available_actions.includes('READY') ? '查看并备好' : '查看' },
        })),
      });
      return true;
    } catch (error) {
      if (generation === this._loadGeneration) this.setData({ listState: 'error', list: [], merchantRole: '' });
      return false;
    }
  },
  onKw(e) { this.setData({ kw: (e.detail.value || '').trim() }); return this.load(); },
  switchLane(e) { this.setData({ lane: e.currentTarget.dataset.l, kw: '', hint: '' }); return this.load(); },
  async setBiz(e) {
    const status = e.currentTarget.dataset.b;
    if (!BIZ.some(item => item.code === status)) return false;
    this.setData({ statusWriteState: 'loading' });
    try {
      await merchantStore.setStoreStatus(status, api.newIdempotencyKey('store-status'));
      this.setData({ storeStatus: status, statusWriteState: 'ready' });
      return true;
    } catch (error) { this.setData({ statusWriteState: 'error' }); return false; }
  },
  cardTap(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
  viewOrder(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
  advance(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
});
