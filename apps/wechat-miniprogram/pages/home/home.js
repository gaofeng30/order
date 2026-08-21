const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');

// §5.1：存在这三态订单时首页顶部常驻提示条
const ONGOING = ['已预约', '制作中', '待取餐'];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: {},
    notice: '',
    bizStatus: '',
    ongoing: null,
    grid: [
      { k: 'reserve', icon: 'calendarClock', cn: '预约点餐' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
    ],
  },
  onShow() { this.build(); },
  build() {
    /* 公告与营业状态都读配置，首页不持有事实。营业状态不从截单时刻派生：
       §6.9 允许主账号手动覆盖营业时间规则，派生值只是默认值。 */
    const g = getApp().globalData;
    const store = g.store && g.store.name ? g.store : data.STORE;
    this.setData({
      store,
      notice: store.notice || '',
      bizStatus: store.status || '',
      ongoing: this.buildOngoing((g.orders || []).filter(o => ONGOING.includes(o.status))),
    });
  },
  /* 提示条同时是 §5.10 的兜底：订阅消息需用户主动授权且只能一次性订阅，
     拒绝授权的用户只能靠这条提示得知餐已备好。 */
  buildOngoing(live) {
    if (!live.length) return null;
    // 按取餐时刻排序，不按下单时间 —— 用户关心的是下一顿什么时候能拿
    const sorted = live.slice().sort((a, b) =>
      `${a.pickupDate} ${a.pickupTime}`.localeCompare(`${b.pickupDate} ${b.pickupTime}`));
    const ready = sorted.find(o => o.status === '待取餐');
    if (ready) {
      // 此刻用户需要的动作是现在就去窗口，文案不再重复单数以免稀释行动号召
      return { ready: true, count: live.length, orderId: ready.id,
               text: `已备好，可取餐 · 取餐号 ${ready.code}` };
    }
    const next = sorted[0];
    return { ready: false, count: live.length, orderId: next.id,
             text: `你有 ${live.length} 单进行中 · ${data.orderPickupLabel(next)} 取餐` };
  },
  tapOngoing() {
    if (this.data.ongoing) nav.go('order-detail', { id: this.data.ongoing.orderId });
  },
  toMenu() { nav.tabTo('menu'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  gridTap(e) {
    const k = e.currentTarget.dataset.k;
    switch (k) {
      case 'reserve': nav.tabTo('menu'); break;
      case 'orders': nav.tabTo('orders'); break;
      case 'pickup': {
        const t = getApp().globalData.orders.find(o => o.status === '待取餐' || o.status === '已预约');
        if (t) nav.go('order-detail', { id: t.id });
        else this.toast('暂无可取餐订单', 'warn');
        break;
      }
      default: break;
    }
  },
});
