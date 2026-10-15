const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');

// 核销失败按错误码分原因提示（A17）
function failureText(error, source) {
  const code = error && error.code;
  // 无效或过期的二维码后端同样返回 ORDER_NOT_FOUND，扫码场景不说「取餐号」
  if (code === 'ORDER_NOT_FOUND') return source === 'scan' ? '没有找到今天的这个订单，请核对二维码或改用取餐号' : '没有找到今天的这个取餐号';
  if (code === 'TRANSITION_NOT_ALLOWED' || code === 'REDEEM_UNAVAILABLE') return '当前不能核销（可能已核销、未备好或已退款）';
  if (code === 'FORBIDDEN') return '没有核销权限';
  return '网络异常，请重试';
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { code: '', scanning: false, redeeming: false, result: null },
  // 进入核销 tab 自动打开扫码（A15）；只在进入时打开一次，用户取消不会反复弹出
  onReady() { this.scan(); },
  onUnload() { this._disposed = true; },
  toast(text, warn) {
    const toast = this.selectComponent('#toast');
    if (toast) toast.show(text, warn ? { icon: 'warn' } : {});
  },
  scan() {
    if (this.data.scanning || this.data.redeeming || typeof wx.scanCode !== 'function') return false;
    this.setData({ scanning: true });
    wx.scanCode({
      onlyFromCamera: true,
      scanType: ['qrCode'],
      success: result => {
        if (this._disposed) return;
        this.setData({ scanning: false });
        const token = result && result.result;
        if (typeof token !== 'string' || !token) {
          this.toast('没有识别到取餐二维码，请重试', true);
          return;
        }
        // 扫码后直接核销，无二次确认
        this.redeem(() => merchantStore.verifyScan(token), 'scan');
      },
      fail: result => {
        if (this._disposed) return;
        this.setData({ scanning: false });
        const cancelled = result && typeof result.errMsg === 'string' && /cancel/.test(result.errMsg);
        if (!cancelled) this.toast('扫码未完成，请重试', true);
      },
    });
    return true;
  },
  // 只保留数字，最多 4 位（A18）
  onCode(e) {
    const code = String(e.detail.value || '').replace(/\D/g, '').slice(0, 4);
    if (code !== this.data.code) this.setData({ code });
    return code;
  },
  manual() {
    if (this.data.redeeming || this.data.scanning) return false;
    if (!/^\d{4}$/.test(this.data.code)) {
      this.toast('请输入 4 位取餐号', true);
      return false;
    }
    const code = this.data.code;
    return this.redeem(() => merchantStore.verifyCode(code), 'code');
  },
  async redeem(loader, source) {
    if (this.data.redeeming) return false;
    this.setData({ redeeming: true });
    try {
      const order = await loader();
      if (order.state !== 'COMPLETED') throw new api.APIError('REDEEM_UNAVAILABLE');
      if (this._disposed) return true;
      this.setData({
        redeeming: false, code: '',
        result: { code: order.code, contactName: order.contactName, rows: order.rows },
      });
      return true;
    } catch (error) {
      if (this._disposed) return false;
      // 失败保留手输的取餐号（G5）
      this.setData({ redeeming: false });
      this.toast(failureText(error, source), true);
      return false;
    }
  },
  // 成功弹层：继续扫码（主）/ 完成（次）；点遮罩等同完成（A16）
  continueScan() {
    this.setData({ result: null });
    this.scan();
  },
  done() { this.setData({ result: null }); },
});
