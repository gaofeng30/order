const data = require('../../utils/data.js');
const catalogStore = require('../../utils/catalogStore.js');
const { nav, orderMode } = require('../../utils/util.js');

const CAMPAIGNS = [
  { pill: '当日目录', tone: 'info', title: '查看菜单', sub: ['商品信息由门店目录提供，', '选择后到店自提。'], cta: '进入菜单',
    bg: 'linear-gradient(150deg,#123f7e,#0b2f63)', go: 'menu' },
];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    campaigns: CAMPAIGNS,
    bannerIdx: 0,
    listState: 'loading',
    signature: [],
    grid: [
      { k: 'now', icon: 'list', cn: '到店点单' },
      { k: 'reserve', icon: 'calendarClock', cn: '预约点餐' },
      { k: 'orders', icon: 'receipt', cn: '我的订单' },
      { k: 'pickup', icon: 'ticket', cn: '取餐码' },
      { k: 'member', icon: 'gift', cn: '会员中心' },
      { k: 'service', icon: 'headset', cn: '联系客服' },
    ],
  },
  onShow() { return this.loadCatalog(); },
  retryCatalog() { return this.loadCatalog(); },
  async loadCatalog() {
    this.setData({ listState: 'loading', signature: [] });
    try {
      const catalog = await catalogStore.loadCatalog();
      const signature = catalogStore.flattenProducts(catalog.categories).slice(0, 4).map(catalogStore.withPrice);
      this.setData({ listState: catalog.categories.length ? 'ready' : 'empty', signature });
    } catch (error) {
      this.setData({ listState: 'error', signature: [] });
    }
  },
  onBanner(e) { this.setData({ bannerIdx: e.detail.current }); },
  dotTap(e) { this.setData({ bannerIdx: e.currentTarget.dataset.i }); },
  toMenu() { nav.tabTo('menu'); },
  toDetail(e) { nav.go('detail', { id: e.currentTarget.dataset.id }); },
  openBanner(e) {
    const c = CAMPAIGNS[e.currentTarget.dataset.i];
    nav.tabTo(c.go);
  },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  gridTap(e) {
    const k = e.currentTarget.dataset.k;
    switch (k) {
      case 'now': orderMode.set('now'); nav.tabTo('menu'); break;
      case 'reserve': orderMode.set('reserve'); nav.tabTo('menu'); break;
      case 'orders': nav.tabTo('orders'); break;
      case 'pickup': {
        const t = getApp().globalData.orders.find(o => o.status === '待取餐' || o.status === '已预约');
        if (t) nav.go('order-detail', { id: t.id });
        else this.toast('暂无可取餐订单', 'warn');
        break;
      }
      case 'member': this.toast('会员中心建设中', 'warn'); break;
      case 'service': this.toast('客服电话 0596-388 1688', 'phone'); break;
      default: break;
    }
  },
});
