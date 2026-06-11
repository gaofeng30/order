const data = require('../../utils/data.js');
const { nav } = require('../../utils/util.js');

const CAMPAIGNS = [
  { pill: '新店首发', tone: 'warn', title: '首单立减 ¥8', sub: ['新人入群即领，到店自提，', '下单自动抵扣。'], cta: '立即点单',
    bg: 'linear-gradient(150deg,#123f7e,#0b2f63)', emblem: true, go: 'menu' },
  { pill: '今日套餐', tone: 'info', title: '商务双拼饭 ¥32', sub: ['黑椒牛肉 + 低温鸡胸，', '午间快速取餐。'], cta: '去看套餐',
    bg: 'linear-gradient(150deg,#3f6e8c,#264a63)', img: '/assets/dishes/p001.jpg', go: 'detail', id: 'p001' },
  { pill: '轻食低脂', tone: 'ok', title: '能量碗 ¥30', sub: ['高蛋白低脂，配油醋汁，', '轻负担也好吃。'], cta: '立即点单',
    bg: 'linear-gradient(150deg,#5b8f3f,#3d6b2f)', img: '/assets/dishes/p005.jpg', go: 'detail', id: 'p005' },
];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    campaigns: CAMPAIGNS,
    bannerIdx: 0,
    signature: [],
    grid: [
      { k: 'menu', icon: 'list', cn: '到店点单' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
      { k: 'member', icon: 'gift', cn: '会员中心' },
      { k: 'delivery', icon: 'truck', cn: '外卖配送', off: true },
      { k: 'service', icon: 'headset', cn: '联系客服' },
    ],
  },
  onLoad() {
    this.setData({
      signature: data.MENU.filter(m => m.tags.includes('今日推荐') || m.tags.includes('热销')).slice(0, 4),
    });
  },
  onBanner(e) { this.setData({ bannerIdx: e.detail.current }); },
  dotTap(e) { this.setData({ bannerIdx: e.currentTarget.dataset.i }); },
  toMenu() { nav.tabTo('menu'); },
  toDetail(e) { nav.go('detail', { id: e.currentTarget.dataset.id }); },
  openBanner(e) {
    const c = CAMPAIGNS[e.currentTarget.dataset.i];
    if (c.id) nav.go('detail', { id: c.id }); else nav.tabTo(c.go);
  },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  login() { this.toast('已发送登录邀请', 'check'); },
  gridTap(e) {
    const k = e.currentTarget.dataset.k;
    switch (k) {
      case 'menu': nav.tabTo('menu'); break;
      case 'orders': nav.tabTo('orders'); break;
      case 'pickup': {
        const t = getApp().globalData.orders.find(o => o.status === '待取餐');
        if (t) nav.go('order-detail', { id: t.id });
        else this.toast('暂无待取餐订单', 'warn');
        break;
      }
      case 'member': this.toast('会员中心建设中', 'warn'); break;
      case 'service': this.toast('客服电话 0596-388 1688', 'phone'); break;
      default: break;
    }
  },
});
