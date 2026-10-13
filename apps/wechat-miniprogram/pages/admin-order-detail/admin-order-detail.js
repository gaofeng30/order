const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const dateLabel = require('../../utils/dateLabel.js');

function failureText(error) {
  const code = error && error.code;
  if (code === 'TRANSITION_NOT_ALLOWED') return '订单状态已变化，已为你刷新';
  if (code === 'FORBIDDEN') return '没有操作权限';
  return '网络异常，请重试';
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    detailState: 'loading', busy: false, o: null, rows: [], flavorLines: [],
    action: '', placedLabel: '', pickupLabel: '',
  },
  onLoad(opts) {
    this._id = String((opts && opts.id) || '');
    this.load('reset');
  },
  // 回到前台时静默刷新，不闪加载态
  onShow() {
    if (this._shown && this.data.detailState === 'ready' && !this.data.busy) this.load('silent');
    this._shown = true;
  },
  async load(mode) {
    const generation = (this._generation || 0) + 1;
    this._generation = generation;
    if (mode === 'reset') this.setData({ detailState: 'loading' });
    try {
      const order = await merchantStore.detail(this._id);
      if (generation === this._generation) this.build(order);
      return true;
    } catch (error) {
      if (generation === this._generation && mode !== 'silent') this.setData({ detailState: 'error' });
      return false;
    }
  },
  retry() { return this.load('reset'); },
  build(order) {
    const actions = order.available_actions;
    const rows = order.rows.map((row, index) => Object.assign({}, row, { last: index === order.rows.length - 1 }));
    // 口味按菜品分行（A12）
    const flavorLines = order.rows
      .filter(row => row.flavors.length || row.note)
      .map(row => ({ key: row.key, id: row.id, name: row.name, text: row.flavors.concat(row.note ? [row.note] : []).join('、') }));
    this.setData({
      detailState: 'ready', o: order, rows, flavorLines,
      action: actions.includes('READY') ? 'ready' : actions.includes('REDEEM') ? 'redeem' : '',
      placedLabel: dateLabel.instantLabel(order.paid_at || order.materialized_at || ''),
      pickupLabel: `${dateLabel.dayLabel(order.pickupDate)} ${order.pickupTime}`,
    });
  },
  advance() {
    if (this.data.busy) return;
    if (this.data.action === 'ready') this.markReady();
    else if (this.data.action === 'redeem') this.confirmRedeem();
  },
  // 备好不确认，只防重复点击（A10）
  markReady() {
    return this.run(id => merchantStore.markReady(id, api.newIdempotencyKey('ready')), '已备好，已通知顾客');
  },
  // 核销前确认顾客已取餐（A11）
  confirmRedeem() {
    const o = this.data.o;
    wx.showModal({
      title: '确认顾客已取餐？',
      content: o.contactName ? `取餐号 ${o.code} · ${o.contactName}` : `取餐号 ${o.code}`,
      confirmText: '确认核销',
      confirmColor: '#3d6b2f',
      success: result => {
        if (result.confirm) this.run(id => merchantStore.redeem(id, api.newIdempotencyKey('redeem')), '已核销');
      },
    });
  },
  async run(action, successText) {
    if (this.data.busy || !this.data.o) return false;
    // 作废进行中的静默刷新，避免旧状态覆盖本次操作结果
    this._generation = (this._generation || 0) + 1;
    this.setData({ busy: true });
    const toast = this.selectComponent('#toast');
    try {
      const order = await action(this.data.o.id);
      this.build(order);
      this.setData({ busy: false });
      toast.show(successText);
      return true;
    } catch (error) {
      this.setData({ busy: false });
      toast.show(failureText(error), { icon: 'warn' });
      if (error && error.code === 'TRANSITION_NOT_ALLOWED') this.load('silent');
      return false;
    }
  },
});
