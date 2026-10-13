const api = require('../../utils/apiClient.js');
const orderStore = require('../../utils/orderStore.js');
const subscriptionStore = require('../../utils/subscriptionStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    detailState: 'loading', o: null, rows: [], flavorsStr: '', showQr: false, qrToken: '',
    canCancel: false, canSubscribeReady: false, readySubscribing: false, readySubscribed: false,
    cancelSheet: false, canceling: false, pickupText: '', navTitle: '订单详情',
  },
  onLoad(opts) {
    this._id = String((opts && opts.id) || '');
    // 订单列表「取消预约」带 cancel=1 进入：加载完且可取消时直接弹出取消确认（U43）
    this._autoCancel = !!(opts && opts.cancel === '1');
    return this.load();
  },
  async load() {
    this.setData({ detailState: 'loading', o: null, rows: [], showQr: false, qrToken: '' });
    try {
      const order = await orderStore.detail(this._id);
      this.build(order);
      if (this._autoCancel) {
        this._autoCancel = false;
        if (this.data.canCancel) this.setData({ cancelSheet: true });
      }
      return true;
    } catch (error) {
      this.setData({ detailState: error && error.statusCode === 404 ? 'not_found' : 'error' });
      return false;
    }
  },
  build(order) {
    const showQr = order.state === 'READY_FOR_PICKUP' && !!order.redemptionToken;
    this.setData({
      detailState: 'ready', o: order, rows: order.rows,
      flavorsStr: [...new Set(order.rows.flatMap(row => row.flavors))].join(' / '),
      showQr, qrToken: showQr ? order.redemptionToken : '',
      canCancel: order.available_actions.includes('CANCEL'),
      canSubscribeReady: order.notificationOptions.includes('READY'),
      pickupText: `${dayLabel(order.pickupDate)} ${order.pickupTime}`,
      navTitle: showQr ? '取餐码' : '订单详情',
    });
  },
  copy() {
    if (!this.data.o) return;
    // 系统会自动提示「内容已复制」，不再叠加自定义提示
    wx.setClipboardData({ data: this.data.o.code });
  },
  goOrders() { nav.tabTo('orders'); },
  openCancel() { if (this.data.canCancel) this.setData({ cancelSheet: true }); },
  closeCancel() { if (!this.data.canceling) this.setData({ cancelSheet: false }); },
  async doCancel() {
    if (!this.data.canCancel || !this.data.o || this.data.canceling) return false;
    this.setData({ canceling: true });
    try {
      await subscriptionStore.requestAndRecord(this.data.o.id, 'REFUND_RESULT');
      const result = await orderStore.cancel(this.data.o.id, api.newIdempotencyKey('cancel'));
      this.setData({ cancelSheet: false, canceling: false });
      this.build(result.order);
      this.selectComponent('#toast').show('已取消预约，款项将原路退回');
      return true;
    } catch (error) {
      // 失败时保留弹层，便于直接重试（U34）
      this.setData({ canceling: false });
      this.selectComponent('#toast').show('取消失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  async subscribeReady() {
    if (!this.data.o || this.data.readySubscribing || this.data.readySubscribed) return false;
    this.setData({ readySubscribing: true });
    const decision = await subscriptionStore.requestAndRecord(this.data.o.id, 'READY');
    const toast = this.selectComponent('#toast');
    if (decision === 'ACCEPTED') toast.show('已开启取餐提醒');
    else if (decision === 'REJECTED') toast.show('未开启提醒', { icon: 'warn' });
    else toast.show('提醒暂未开启，请重试', { icon: 'warn' });
    this.setData({ readySubscribing: false, readySubscribed: decision === 'ACCEPTED' });
    return decision === 'ACCEPTED';
  },
});
