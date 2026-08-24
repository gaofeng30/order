const { nav } = require('../../utils/util.js');
const storefrontStore = require('../../utils/storefrontStore.js');

/* 身份选择页 —— 仅已绑定商户的冷启动落点。

   未绑定用户由 App 在静默会话与服务端身份判定后直接送往首页；本页仍保留
   用户端入口，供已绑定商户切回普通用户浏览。商户首次绑定只从个人中心触发。
   商户端四屏仍由服务端逐请求执行 RBAC，本页只负责导航。 */
Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    entryState: 'loading',
    storefrontState: 'loading',
    storeName: '',
    launchLayer: null,
    hint: '',
  },
  async onShow() {
    this.clearLaunchTimer();
    const routing = await getApp().resolveEntryRoute();
    if (routing.state === 'user') return;
    this.setData({
      entryState: routing.state,
      hint: routing.state === 'error' ? '身份状态暂不可用，仍可进入用户端浏览或重试商户验证。' : '',
    });
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
  goMerchant() {
    if (this.data.entryState !== 'merchant' || this.data.storefrontState !== 'ready') return false;
    nav.go('admin-orders');
    return true;
  },
});
