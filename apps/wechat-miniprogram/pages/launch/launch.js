const { nav } = require('../../utils/util.js');
const api = require('../../utils/apiClient.js');
const storefrontStore = require('../../utils/storefrontStore.js');

/* 身份选择页 —— 入口页（项目方 2026-08-22 决策，已回写 §4.4）。

   用户端一侧零索取：§14 要求用户能免手机号浏览、启动时不弹手机号授权。
   商户端一侧触发微信真实的手机号授权面板（§4.4 的绑定链路）。

   前端拿到的只是加密数据，明文手机号与商户账号名单的比对必须由服务端完成，
   所以这里不做也不声称做任何校验。§4.4 末条：客户端隐藏入口不能代替鉴权，
   商户端四屏的访问控制在服务端，本页不给它任何前端替身。 */
Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    storefrontState: 'loading',
    storeName: '',
    launchLayer: null,
    hint: '',
  },
  async onShow() {
    this.clearLaunchTimer();
    this.setData({ storefrontState: 'loading', storeName: '', launchLayer: null });
    try {
      const settings = await storefrontStore.load();
      getApp().globalData.storefrontFlavors = settings.flavors;
      this.setData({ storefrontState: 'ready', storeName: settings.name, launchLayer: settings.launchLayer });
      this.scheduleLaunchLayer();
    } catch (error) {
      this.setData({ storefrontState: 'error' });
    }
  },
  onHide() { this.clearLaunchTimer(); },
  onUnload() { this.clearLaunchTimer(); },
  clearLaunchTimer() {
    if (this._launchTimer) clearTimeout(this._launchTimer);
    this._launchTimer = null;
  },
  scheduleLaunchLayer() {
    this.clearLaunchTimer();
    if (!this.data.launchLayer) return;
    this._launchTimer = setTimeout(() => this.dismissLaunchLayer(), 1500);
  },
  dismissLaunchLayer() {
    this.clearLaunchTimer();
    if (this.data.launchLayer) this.setData({ launchLayer: null });
    return true;
  },
  retryStorefront() { return this.onShow(); },
  back() { nav.back(); },
  go(e) {
    if (this.data.storefrontState !== 'ready') return false;
    nav.go(e.currentTarget.dataset.to);
    return true;
  },

  /* 微信手机号授权回调。允许时 detail 带 code / encryptedData，拒绝时只有 errMsg。 */
  async onMerchantPhone(e) {
    if (this.data.storefrontState !== 'ready') return false;
    const d = (e && e.detail) || {};
    if (typeof d.code !== 'string' || !d.code.trim()) {
      // 拒绝是合法选择，不渲染成失败，也不拦路
      this.setData({ hint: '商户端需要验证手机号身份。未授权时仍可从上方进入用户端浏览。' });
      return false;
    }
    this.setData({ hint: '正在核验商户身份…' });
    try {
      const result = await api.intrinsic('/api/v1/me/merchant-login', { code: d.code.trim() });
      if (!result || !result.merchant || !result.merchant.role) throw new api.APIError('MERCHANT_IDENTITY_UNAVAILABLE');
      this.setData({ hint: '' });
      nav.go('admin-orders');
      return true;
    } catch (error) {
      this.setData({ hint: '商户身份未核验，请重试；用户端入口仍可使用。' });
      return false;
    }
  },
});
