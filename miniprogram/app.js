// 绥安食品 · 点餐小程序 — App 入口
const data = require('./utils/data.js');

App({
  globalData: {
    // ---- 屏幕适配信息（各机型自适应核心）----
    statusBarHeight: 20, // 状态栏高度 px
    navBarHeight: 44,    // 标题栏高度 px（与右上胶囊对齐）
    navTotalHeight: 64,  // 状态栏 + 标题栏
    capsuleRightGap: 95, // 右侧操作区需避让胶囊的宽度 px
    safeBottom: 0,       // 底部安全区 px（全面屏 home 条）
    screenWidth: 375,

    // ---- 业务状态（跨页共享，模拟后端）----
    store: { status: '营业中' },
    cart: {},               // { [id]: { qty, flavors:[], note:'' } } 口味/备注绑定到菜品
    orderMode: 'now',       // 下单模式 'now'尽快 | 'reserve'预约
    orders: [],             // 用户端订单
    lastOrder: null,        // 最近一笔下单
    aOrders: [],            // 商户端订单
    products: {},           // { [id]: 'on'|'soldout'|'off' }
  },

  onLaunch() {
    this.initSystemInfo();
    const g = this.globalData;
    // 深拷贝种子数据，避免污染原始 mock
    g.orders = JSON.parse(JSON.stringify(data.USER_ORDERS));
    g.aOrders = JSON.parse(JSON.stringify(data.ADMIN_ORDERS));
    g.products = {};
    data.MENU.forEach(m => { g.products[m.id] = m.status; });
  },

  initSystemInfo() {
    let win, menu;
    try {
      win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
    } catch (e) {
      win = wx.getSystemInfoSync();
    }
    try {
      menu = wx.getMenuButtonBoundingClientRect();
    } catch (e) {
      menu = null;
    }
    const g = this.globalData;
    const statusBarHeight = win.statusBarHeight || 20;
    g.statusBarHeight = statusBarHeight;
    g.screenWidth = win.screenWidth || 375;

    if (menu && menu.height) {
      // 标题栏高度 = (胶囊上边距 - 状态栏)*2 + 胶囊高度，使标题与胶囊垂直居中
      g.navBarHeight = (menu.top - statusBarHeight) * 2 + menu.height;
      // 右侧操作区避让胶囊：从屏幕右边到胶囊左边的宽度
      g.capsuleRightGap = (g.screenWidth - menu.left) + 8;
    } else {
      g.navBarHeight = 44;
      g.capsuleRightGap = 95;
    }
    g.navTotalHeight = g.statusBarHeight + g.navBarHeight;

    // 底部安全区
    const safe = win.safeArea;
    if (safe && win.screenHeight) {
      g.safeBottom = Math.max(0, win.screenHeight - safe.bottom);
    } else {
      g.safeBottom = 0;
    }
  },
});
