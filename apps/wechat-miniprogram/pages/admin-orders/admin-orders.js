const api = require('../../utils/apiClient.js');
const merchantStore = require('../../utils/merchantStore.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const dateLabel = require('../../utils/dateLabel.js');
const { nav } = require('../../utils/util.js');

const LANES = ['已预约', '制作中', '待取餐', '已完成', '已退款'];
const BIZ = [{ code: 'open', label: '营业中' }, { code: 'closed', label: '休息中' }, { code: 'cutoff', label: '已截单' }];
const BIZ_CONFIRM = {
  closed: '用户将无法下单，可随时切回营业中',
  cutoff: '用户将无法继续下单，可随时切回营业中',
};
const SEARCH_DELAY = 300;
const AUTO_REFRESH = 30 * 1000;

function card(order) {
  const canReady = order.available_actions.includes('READY');
  return Object.assign({}, order, {
    pickupLabel: `${dateLabel.dayLabel(order.pickupDate)} ${order.pickupTime}`,
    placedLabel: dateLabel.instantLabel(order.materialized_at || ''),
    canReady,
    actionLabel: canReady ? '查看并备好' : '查看',
  });
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    lanes: LANES, lane: '已预约', biz: BIZ, storeStatus: '', statusSaving: false,
    listState: 'loading', list: [], kw: '', refreshing: false,
  },
  onShow() {
    // 首次进入显示加载；从详情返回时静默刷新，保留列表与滚动位置
    if (this._shown) {
      this.loadStatus();
      this.load('silent');
    } else {
      this._shown = true;
      this.loadStatus();
      this.load('reset');
    }
    this.startAutoRefresh();
  },
  onHide() {
    this.stopTimers();
    this._generation = (this._generation || 0) + 1;
    this._statusGeneration = (this._statusGeneration || 0) + 1;
  },
  onUnload() {
    this.stopTimers();
    this._generation = (this._generation || 0) + 1;
    this._statusGeneration = (this._statusGeneration || 0) + 1;
  },
  stopTimers() {
    clearInterval(this._refreshTimer);
    clearTimeout(this._searchTimer);
    this._refreshTimer = null;
    this._searchTimer = null;
  },
  // 页面可见时每 30 秒静默刷新；有搜索词时暂停（A6）
  startAutoRefresh() {
    clearInterval(this._refreshTimer);
    this._refreshTimer = setInterval(() => {
      if (this.data.kw) return;
      if (!this.data.statusSaving) this.loadStatus();
      this.load('silent');
    }, AUTO_REFRESH);
  },
  loadStatus() {
    if (this._statusTask && this._statusTask.generation === this._statusGeneration) return this._statusTask.promise;
    const task = {};
    task.promise = this.fetchStatus().finally(() => {
      if (this._statusTask === task) this._statusTask = null;
    });
    task.generation = this._statusGeneration;
    this._statusTask = task;
    return task.promise;
  },
  async fetchStatus() {
    const generation = (this._statusGeneration || 0) + 1;
    this._statusGeneration = generation;
    try {
      const storefront = await storefrontStore.load();
      if (generation !== this._statusGeneration || this.data.statusSaving) return;
      if (storefront.businessStatus !== this.data.storeStatus) {
        this.setData({ storeStatus: storefront.businessStatus });
      }
    } catch (error) {
      // 读取失败保留当前显示；首次就失败时告诉商户怎么恢复
      if (generation === this._statusGeneration && !this.data.storeStatus && !this._statusFailToast) {
        this._statusFailToast = true;
        this.selectComponent('#toast').show('营业状态没加载出来，下拉刷新可重试', { icon: 'warn' });
      }
    }
  },
  /* mode: reset = 显示加载中；search = 保留旧列表直到结果返回，过期响应丢弃；
     silent = 定时/返回时后台刷新，失败不打扰；refresh = 下拉刷新，失败 toast 并保留列表 */
  load(mode) {
    const key = JSON.stringify([this.data.lane, this.data.kw]);
    if (this._loadTask && this._loadTask.key === key && this._loadTask.generation === this._generation) {
      if (mode === 'refresh') this._loadTask.refresh = true;
      return this._loadTask.promise;
    }
    const task = { key, refresh: mode === 'refresh' };
    task.promise = this.fetchOrders(mode).finally(() => {
      if (this._loadTask === task) this._loadTask = null;
    });
    task.generation = this._generation;
    this._loadTask = task;
    return task.promise;
  },
  async fetchOrders(mode) {
    const generation = (this._generation || 0) + 1;
    this._generation = generation;
    if (mode === 'reset') this.setData({ listState: 'loading', list: [] });
    try {
      const orders = await merchantStore.orders(this.data.lane, this.data.kw);
      if (generation !== this._generation) return false;
      const list = orders.map(card);
      const listState = list.length ? 'ready' : 'empty';
      if (listState !== this.data.listState || JSON.stringify(list) !== JSON.stringify(this.data.list)) {
        this.setData({ listState, list });
      }
      return true;
    } catch (error) {
      if (generation !== this._generation) return false;
      if (this._loadTask && this._loadTask.generation === generation && this._loadTask.refresh) mode = 'refresh';
      if (mode === 'silent' && this.data.listState !== 'loading') return false;
      if (mode === 'refresh' && (this.data.listState === 'ready' || this.data.listState === 'empty')) {
        this.selectComponent('#toast').show('刷新失败，请稍后重试', { icon: 'warn' });
        return false;
      }
      this.setData({ listState: 'error', list: [] });
      return false;
    }
  },
  retry() { return this.load('reset'); },
  async onRefresh() {
    this.setData({ refreshing: true });
    await Promise.all([this.load('refresh'), this.loadStatus()]);
    this.setData({ refreshing: false });
  },
  // 停止输入 0.3 秒后搜索；满 4 位取餐号立即搜索（A5）
  onKw(e) {
    const kw = (e.detail.value || '').trim();
    if (kw === this.data.kw) return;
    this._generation = (this._generation || 0) + 1;
    this.setData({ kw });
    clearTimeout(this._searchTimer);
    if (/^\d{4}$/.test(kw)) {
      this.load('search');
      return;
    }
    this._searchTimer = setTimeout(() => this.load('search'), SEARCH_DELAY);
  },
  clearKw() {
    clearTimeout(this._searchTimer);
    this.setData({ kw: '' });
    return this.load('search');
  },
  switchLane(e) {
    const lane = e.currentTarget.dataset.l;
    if (lane === this.data.lane && !this.data.kw) return false;
    clearTimeout(this._searchTimer);
    this.setData({ lane, kw: '' });
    return this.load('reset');
  },
  // 切到休息中 / 已截单前确认；切回营业中直接生效（A4）
  onBiz(e) {
    const code = e.currentTarget.dataset.b;
    const target = BIZ.find(item => item.code === code);
    if (!target || this.data.statusSaving || code === this.data.storeStatus) return;
    if (code === 'open') {
      this.saveStatus(target);
      return;
    }
    wx.showModal({
      title: `切换为${target.label}？`,
      content: BIZ_CONFIRM[code],
      confirmText: '确认切换',
      confirmColor: '#3d6b2f',
      success: result => { if (result.confirm) this.saveStatus(target); },
    });
  },
  async saveStatus(target) {
    if (this.data.statusSaving) return false;
    this._statusGeneration = (this._statusGeneration || 0) + 1;
    this.setData({ statusSaving: true });
    const toast = this.selectComponent('#toast');
    try {
      await merchantStore.setStoreStatus(target.code, api.newIdempotencyKey('store-status'));
      this.setData({ storeStatus: target.code, statusSaving: false });
      toast.show(`已切换为${target.label}`);
      return true;
    } catch (error) {
      this.setData({ statusSaving: false });
      toast.show('营业状态切换失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  openOrder(e) { nav.go('admin-order-detail', { id: e.currentTarget.dataset.id }); },
});
