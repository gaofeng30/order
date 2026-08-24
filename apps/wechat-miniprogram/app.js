// 绥安食品 · 点餐小程序 — App 入口
const runtimeEndpointConfig = require('./utils/runtimeEndpointConfig.js');
const { isRuntimeOrigin, resolveRuntimeEndpoint } = require('./utils/runtimeEndpoint.js');
const sessionApi = require('./utils/sessionApi.js');
const identityStore = require('./utils/identityStore.js');
const subscriptionTemplateConfig = require('./utils/subscriptionTemplateConfig.js');
const { resolveSubscriptionTemplateIds } = require('./utils/subscriptionTemplate.js');

App({
  globalData: {
    apiBaseUrl: '',
    runtimeEndpoint: { state: 'idle', envVersion: '', origin: '', errorCode: '' },
    session: { state: 'idle', accessToken: '', expiresAt: '' },
    entryRouting: { state: 'loading' },
    subscriptionTemplateIds: {},

    // ---- 屏幕适配信息（各机型自适应核心）----
    statusBarHeight: 20, // 状态栏高度 px
    navBarHeight: 44,    // 标题栏高度 px（与右上胶囊对齐）
    navTotalHeight: 64,  // 状态栏 + 标题栏
    capsuleRightGap: 95, // 右侧操作区需避让胶囊的宽度 px
    safeBottom: 0,       // 底部安全区 px（全面屏 home 条）
    screenWidth: 375,

    // 购物车和当前选择只负责跨页交互；目录、价格、订单、营业与售罄均由服务端持有。
    cart: {},
    pickup: null,

  },

  onLaunch() {
    this.initSystemInfo();
    const g = this.globalData;
    g.runtimeEndpoint = resolveRuntimeEndpoint(wx, runtimeEndpointConfig);
    g.apiBaseUrl = g.runtimeEndpoint.state === 'ready' ? g.runtimeEndpoint.origin : '';
    g.subscriptionTemplateIds = resolveSubscriptionTemplateIds(g.runtimeEndpoint.envVersion, subscriptionTemplateConfig);
    if (this.isSessionEndpointReady()) {
      // 首屏 launch 的 onShow 在会话就绪后读取服务端身份并完成分流。
      this.startSession();
    } else {
      g.session = { state: 'error', accessToken: '', expiresAt: '' };
      g.entryRouting = { state: 'error' };
    }
  },

  resolveEntryRoute() {
    const g = this.globalData;
    if (g.entryRouting.state === 'user' || g.entryRouting.state === 'merchant') {
      return Promise.resolve(g.entryRouting);
    }
    if (this.entryRoutingPromise) return this.entryRoutingPromise;

    g.entryRouting = { state: 'loading' };
    const pending = this.startSession().then(session => {
      if (!session || session.state !== 'ready') throw new Error('SESSION_UNAVAILABLE');
      return identityStore.load();
    }).then(identity => {
      if (identity.merchant.bound) {
        g.entryRouting = { state: 'merchant', role: identity.merchant.role || '' };
        return g.entryRouting;
      }
      g.entryRouting = { state: 'user' };
      wx.reLaunch({ url: '/pages/home/home' });
      return g.entryRouting;
    }).catch(() => {
      // 身份事实不可读时不猜测商户身份；保留入口页的用户通道和显式重试。
      g.entryRouting = { state: 'error' };
      return g.entryRouting;
    });
    this.entryRoutingPromise = pending;
    return pending.finally(() => {
      if (this.entryRoutingPromise === pending) this.entryRoutingPromise = null;
    });
  },

  isSessionEndpointReady() {
    const g = this.globalData;
    const endpoint = g.runtimeEndpoint;
    return endpoint
      && endpoint.state === 'ready'
      && endpoint.origin === g.apiBaseUrl
      && isRuntimeOrigin(endpoint.envVersion, endpoint.origin);
  },

  startSession() {
    const g = this.globalData;
    if (!this.isSessionEndpointReady()) {
      g.session = { state: 'error', accessToken: '', expiresAt: '' };
      return Promise.resolve(g.session);
    }
    if (g.session.state === 'ready' && g.session.accessToken) return Promise.resolve(g.session);
    if (g.session.state === 'loading' && this.sessionPromise) return this.sessionPromise;

    g.session = { state: 'loading', accessToken: '', expiresAt: '' };
    const pending = sessionApi.createSession(g.apiBaseUrl).then(session => {
      g.session = {
        state: 'ready',
        accessToken: session.accessToken,
        expiresAt: session.expiresAt,
      };
      return g.session;
    }).catch(() => {
      g.session = { state: 'error', accessToken: '', expiresAt: '' };
      return g.session;
    });
    this.sessionPromise = pending;
    return pending.finally(() => {
      if (this.sessionPromise === pending) this.sessionPromise = null;
    });
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
