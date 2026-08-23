const api = require('../../utils/apiClient.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const { nav } = require('../../utils/util.js');

const ACTIVE = new Set(['RESERVED', 'PREPARING', 'READY_FOR_PICKUP']);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    storeName: '', storeAddress: '', pickupPoint: '', notice: '', bizStatus: '',
    settingsState: 'loading', ordersState: 'loading', ongoing: null, launchLayer: null,
    grid: [
      { k: 'reserve', icon: 'calendarClock', cn: '预约点餐' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
    ],
  },
  async onShow() {
    this.clearLaunchTimer();
    this.setData({ settingsState: 'loading', ordersState: 'loading', ongoing: null, launchLayer: null });
    const settingsTask = storefrontStore.load().then(settings => {
      getApp().globalData.storefrontFlavors = settings.flavors;
      this.setData({
        settingsState: 'ready', storeName: settings.name, storeAddress: settings.address,
        pickupPoint: settings.pickupPoint, notice: settings.announcement,
        bizStatus: settings.businessStatusLabel, launchLayer: settings.launchLayer,
      });
      this.scheduleLaunchLayer();
    }).catch(() => this.setData({ settingsState: 'error' }));
    const ordersTask = api.get('/api/v1/orders?active=true', true).then(body => {
      if (!body || !Array.isArray(body.orders)) throw new api.APIError('ORDERS_UNAVAILABLE');
      this.setData({ ordersState: 'ready', ongoing: this.buildOngoing(body.orders) });
    }).catch(() => this.setData({ ordersState: 'error', ongoing: null }));
    await Promise.all([settingsTask, ordersTask]);
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
  retrySettings() { return this.onShow(); },
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
        : `你有 ${live.length} 单进行中 · ${next.pickup_date} ${next.pickup_time} 取餐`,
    };
  },
  tapOngoing() {
    if (this.data.settingsState !== 'ready' || !this.data.ongoing) return false;
    nav.go('order-detail', { id: this.data.ongoing.orderId });
    return true;
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
    else if (key === 'pickup' && this.data.ongoing) nav.go('order-detail', { id: this.data.ongoing.orderId });
    else if (key === 'pickup') this.toast('暂无可取餐订单', 'warn');
    else return false;
    return true;
  },
});
