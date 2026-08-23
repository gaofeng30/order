const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { code: '', match: null, lookupState: 'idle', lastResult: null },
  onCode(e) { this.setData({ code: (e.detail.value || '').trim() }); },
  scan() {
    if (!wx.scanCode) return Promise.resolve(false);
    return new Promise(resolve => wx.scanCode({ onlyFromCamera: true, scanType: ['qrCode'],
      success: async result => {
        const token = result && result.result;
        if (typeof token !== 'string' || !token) { resolve(false); return; }
        resolve(this.lookup(() => merchantStore.verifyScan(token)));
      }, fail: () => resolve(false) })).then(value => value);
  },
  manual() {
    if (!/^\d{4}$/.test(this.data.code)) return Promise.resolve(false);
    return this.lookup(() => merchantStore.verifyCode(this.data.code));
  },
  async lookup(loader) {
    this.setData({ lookupState: 'loading', match: null });
    try {
      const order = await loader();
      const err = order.state === 'COMPLETED' ? '该订单已核销'
        : order.state === 'REFUNDED' || order.state === 'REFUNDING' ? '退款订单不可核销'
          : order.state !== 'READY_FOR_PICKUP' ? '订单尚未备好' : '';
      this.setData({ lookupState: 'ready', match: { o: order, rows: order.rows, err } });
      return true;
    } catch (error) { this.setData({ lookupState: 'error', match: null }); return false; }
  },
  closeSheet() { this.setData({ match: null }); },
  async confirm() {
    const order = this.data.match && this.data.match.o;
    if (!order || this.data.match.err || !order.available_actions.includes('REDEEM')) return false;
    try {
      const completed = await merchantStore.redeem(order.id, api.newIdempotencyKey('redeem'));
      this.setData({ match: null, code: '', lastResult: completed });
      return true;
    } catch (error) { return false; }
  },
});
