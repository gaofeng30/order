const api = require('../../utils/apiClient.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const subscriptionStore = require('../../utils/subscriptionStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav } = require('../../utils/util.js');

const ACTIVE = new Set(['RESERVED', 'PREPARING', 'READY_FOR_PICKUP']);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    storeName: '', storeAddress: '', pickupPoint: '', notice: '', bizStatus: '',
    settingsState: 'loading', ordersState: 'loading', ongoing: null, launchLayer: null, subscribing: false,
    grid: [
      { k: 'reserve', icon: 'calendarClock', cn: '预约点餐' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
    ],
  },
  async onShow() {
    // 门店设置已显示过时静默刷新；进行中订单提示条不先清空，避免闪烁（E5）
    await Promise.all([this.loadSettings(this.data.settingsState !== 'ready'), this.loadOrders()]);
  },
  onHide() { this.dismissLaunchLayer(); },
  onUnload() { this.clearLaunchTimer(); },
  loadSettings(showLoading) {
    if (showLoading) this.setData({ settingsState: 'loading' });
    return storefrontStore.load().then(settings => {
      const g = getApp().globalData;
      g.storefrontFlavors = settings.flavors;
      const patch = {
        settingsState: 'ready', storeName: settings.name, storeAddress: settings.address,
        pickupPoint: settings.pickupPoint, notice: settings.announcement, bizStatus: settings.businessStatusLabel,
      };
      // 开屏装饰图每次打开小程序只弹一次，首页与启动页共用标记（E5）
      if (settings.launchLayer && !g.launchLayerShown) {
        g.launchLayerShown = true;
        patch.launchLayer = settings.launchLayer;
      }
      this.setData(patch);
    }).catch(() => {
      if (this.data.settingsState !== 'ready') this.setData({ settingsState: 'error' });
    });
  },
  loadOrders() {
    return api.get('/api/v1/orders?active=true', true).then(body => {
      if (!body || !Array.isArray(body.orders)) throw new api.APIError('ORDERS_UNAVAILABLE');
      this.setData({ ordersState: 'ready', ongoing: this.buildOngoing(body.orders) });
    }).catch(() => this.setData({ ordersState: 'error', ongoing: null }));
  },
  clearLaunchTimer() {
    if (this._launchTimer) clearTimeout(this._launchTimer);
    this._launchTimer = null;
  },
  // 图片加载完成后才开始 1.5 秒倒计时（D11）
  onLaunchImageLoad() {
    this.clearLaunchTimer();
    if (!this.data.launchLayer) return;
    this._launchTimer = setTimeout(() => this.dismissLaunchLayer(), 1500);
  },
  // 加载失败直接关闭，不留下透明层吞点击（D11）
  onLaunchImageError() { this.dismissLaunchLayer(); },
  dismissLaunchLayer() {
    this.clearLaunchTimer();
    if (this.data.launchLayer) this.setData({ launchLayer: null });
    return true;
  },
  retrySettings() { return this.loadSettings(true); },
  buildOngoing(orders) {
    const live = orders.filter(order => order && ACTIVE.has(order.state));
    if (!live.length) return null;
    const sorted = live.slice().sort((a, b) =>
      `${a.pickup_date} ${a.pickup_time}`.localeCompare(`${b.pickup_date} ${b.pickup_time}`));
    const next = sorted.find(order => order.state === 'READY_FOR_PICKUP') || sorted[0];
    return {
      ready: next.state === 'READY_FOR_PICKUP', count: live.length, orderId: next.id,
      text: next.state === 'READY_FOR_PICKUP'
        ? `已备好，可取餐 · 取餐号 ${next.pickup_number}`
        : `你有 ${live.length} 单进行中 · ${dayLabel(next.pickup_date)} ${next.pickup_time} 取餐`,
    };
  },
  tapOngoing() {
    if (this.data.settingsState !== 'ready' || !this.data.ongoing) return false;
    nav.go('order-detail', { id: this.data.ongoing.orderId });
    return true;
  },
  async supplementReady() {
    if (!this.data.ongoing || !this.data.ongoing.ready || this.data.subscribing) return false;
    this.setData({ subscribing: true });
    const decision = await subscriptionStore.requestAndRecord(this.data.ongoing.orderId, 'READY');
    this.setData({ subscribing: false });
    if (decision === 'ACCEPTED') this.toast('已开启取餐提醒');
    else if (decision === 'REJECTED') this.toast('未开启提醒', 'warn');
    else this.toast('提醒开启失败，请稍后重试', 'warn');
    return decision === 'ACCEPTED';
  },
  toMenu() {
    if (this.data.settingsState !== 'ready') return false;
    nav.tabTo('menu');
    return true;
  },
  toast(message, icon) { this.selectComponent('#toast').show(message, { icon: icon || 'check' }); },
  gridTap(e) {
    if (this.data.settingsState !== 'ready') return false;
    const key = e.currentTarget.dataset.k;
    if (key === 'reserve') nav.tabTo('menu');
    else if (key === 'orders') nav.tabTo('orders');
    else if (key === 'pickup' && this.data.ongoing && this.data.ongoing.ready) nav.go('order-detail', { id: this.data.ongoing.orderId });
    else if (key === 'pickup') this.toast('暂无可取餐订单', 'warn');
    else return false;
    return true;
  },
});
