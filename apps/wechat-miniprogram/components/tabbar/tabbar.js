const util = require('../../utils/util.js');

Component({
  properties: {
    variant: { type: String, value: 'user' },  // user / admin
    active: { type: String, value: '' },         // 当前页面 id
  },
  data: {
    items: [],
    safeBottom: 0,
    activeColor: '#467a32',
    inactiveColor: '#8f9384',
  },
  lifetimes: {
    attached() { this.build(); },
  },
  // 页面实例被复用（navigateBack 回到 Tab 页）时不会重新 attached，
  // 通过宿主页 show 重算角标，避免「制作中 / 进行中」数字陈旧
  pageLifetimes: {
    show() { this.build(); },
  },
  methods: {
    build() {
      const g = getApp().globalData;
      let items, activeColor;
      if (this.data.variant === 'admin') {
        const pend = g.aOrders.filter(o => o.status === '制作中').length;
        items = [
          { id: 'admin-orders', icon: 'layers', label: '订单', badge: pend },
          { id: 'admin-verify', icon: 'scan', label: '核销' },
          { id: 'admin-products', icon: 'box', label: '菜品' },
        ];
        activeColor = '#2a5fa6';
      } else {
        const pend = g.orders.filter(o => ['已预约', '制作中', '待取餐'].includes(o.status)).length;
        items = [
          { id: 'home', icon: 'home', label: '首页' },
          { id: 'menu', icon: 'list', label: '菜单' },
          { id: 'orders', icon: 'receipt', label: '订单', badge: pend },
          { id: 'profile', icon: 'user', label: '我的' },
        ];
        activeColor = '#467a32';
      }
      this.setData({ items, activeColor, safeBottom: g.safeBottom });
    },
    onTap(e) {
      const id = e.currentTarget.dataset.id;
      if (id === this.data.active) return;
      util.nav.tabTo(id);
    },
  },
});
