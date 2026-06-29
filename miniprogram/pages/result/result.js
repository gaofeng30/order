const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { o: null, store: data.STORE, reserve: false },
  onLoad() {
    const o = getApp().globalData.lastOrder || data.USER_ORDERS[0];
    const reserve = o.type === 'reserve';
    this.setData({
      o, reserve,
      navTitle: reserve ? '预约结果' : '支付结果',
      ringIcon: reserve ? 'calendar' : 'check',
      ringSize: reserve ? 38 : 44,
      mainTitle: reserve ? '预约成功' : '支付成功',
      sub: reserve ? '请准时前往取餐点，凭取餐码领取' : '已通知商户备餐，请凭取餐码到店领取',
      codeLbl: reserve ? '预约取餐号' : '取餐号',
      footIcon: reserve ? 'calendarClock' : 'clock',
      footText: reserve
        ? `${o.pickupLabel} 取 · ${o.pickupPoint}`
        : `${data.STORE.pickup} · ${o.pickupPoint || data.STORE.branch}`,
      viewBtn: reserve ? '查看预约' : '查看取餐码',
    });
    // 支付/预约成功后停留 5s，自动跳转到取餐码页（立刻、预约一致）
    this._autoTimer = setTimeout(() => this.viewCode(), 5000);
  },
  onUnload() { clearTimeout(this._autoTimer); },
  goHome() { clearTimeout(this._autoTimer); nav.tabTo('home'); },
  viewCode() {
    if (this._navigated) return;        // 防止自动跳转与手动点击重复触发
    this._navigated = true;
    clearTimeout(this._autoTimer);
    nav.replace('order-detail', { id: this.data.o.id });
  },
});
