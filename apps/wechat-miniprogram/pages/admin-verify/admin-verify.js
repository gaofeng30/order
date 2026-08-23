const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { code: '', lookupState: 'idle', lastResult: null },
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
    this.setData({ lookupState: 'loading', lastResult: null });
    try {
      const order = await loader();
      if (order.state !== 'COMPLETED') throw new api.APIError('REDEEM_UNAVAILABLE');
      this.setData({ lookupState: 'completed', code: '', lastResult: order });
      return true;
    } catch (error) { this.setData({ lookupState: 'error', lastResult: null }); return false; }
  },
  backToOrders() {
    if (!this.data.lastResult || this.data.lastResult.state !== 'COMPLETED') return false;
    nav.replace('admin-orders', { lane: '已完成' });
    return true;
  },
});
