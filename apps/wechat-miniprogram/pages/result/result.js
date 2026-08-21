const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { o: null, store: data.STORE },
  onLoad() {
    const o = getApp().globalData.lastOrder || data.USER_ORDERS[0];
    // 一期只有预约单，不再按订单类型分支
    this.setData({
      o,
      navTitle: '预约结果',
      ringIcon: 'calendar',
      ringSize: 38,
      mainTitle: '预约成功',
      sub: '备好后会推送提醒，凭取餐码到窗口领取',
      codeLbl: '取餐号',
      footIcon: 'calendarClock',
      // 取餐文案现算：§15.6.2 删除了 pickupLabel 字段
      footText: `${data.orderPickupLabel(o)} 取 · ${o.pickupPoint}`,
      viewBtn: '查看取餐码',
    });
    // 预约成功后停留 5s，自动跳转到取餐码页
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
