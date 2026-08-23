const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { detailState: 'loading', o: null, rows: [], meta: {}, flavorShow: false, flavorText: '' },
  async onLoad(opts) { this._id = String(opts.id || ''); return this.load(); },
  onShow() { if (this._id && this.data.detailState !== 'loading') return this.load(); },
  async load() {
    this.setData({ detailState: 'loading', o: null, rows: [] });
    try { this.build(await merchantStore.detail(this._id)); return true; }
    catch (error) { this.setData({ detailState: 'error' }); return false; }
  },
  build(order) {
    const flavorText = [...new Set(order.rows.flatMap(row => row.flavors.concat(row.note ? [row.note] : [])))].join('；');
    const canReady = order.available_actions.includes('READY');
    const canRedeem = order.available_actions.includes('REDEEM');
    this.setData({
      detailState: 'ready', o: order, rows: order.rows, flavorText, flavorShow: !!flavorText,
      meta: { isView: !canReady && !canRedeem, label: canReady ? '备好' : canRedeem ? '核销' : order.status },
      paidTime: String(order.paid_at || order.materialized_at || '').slice(11, 16),
    });
  },
  async markReady() {
    if (!this.data.o || this.data.o.state !== 'PREPARING' || !this.data.o.available_actions.includes('READY')) return false;
    try { this.build(await merchantStore.markReady(this.data.o.id, api.newIdempotencyKey('ready'))); return true; }
    catch (error) { return false; }
  },
  advance() {
    if (this.data.o && this.data.o.available_actions.includes('READY')) return this.markReady();
    if (this.data.o && this.data.o.available_actions.includes('REDEEM')) return this.redeem();
    return false;
  },
  async redeem() {
    if (!this.data.o || !this.data.o.available_actions.includes('REDEEM')) return false;
    try { this.build(await merchantStore.redeem(this.data.o.id, api.newIdempotencyKey('redeem'))); return true; }
    catch (error) { return false; }
  },
});
