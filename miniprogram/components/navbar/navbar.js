Component({
  properties: {
    title: { type: String, value: '' },
    tone: { type: String, value: 'light' },       // light / dark
    back: { type: Boolean, value: false },
    overlay: { type: Boolean, value: false },       // true: 透明浮层（沉浸式深色头使用）
    leftLabel: { type: String, value: '' },
  },
  data: { statusBarH: 20, navH: 44, capsuleGap: 95 },
  lifetimes: {
    attached() {
      const g = getApp().globalData;
      this.setData({
        statusBarH: g.statusBarHeight,
        navH: g.navBarHeight,
        capsuleGap: g.capsuleRightGap,
      });
    },
  },
  methods: {
    onBack() {
      wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/home/home' }) });
    },
  },
});
