Component({
  properties: {
    title: { type: String, value: '' },
    tone: { type: String, value: 'light' },       // light / dark
    back: { type: Boolean, value: false },
    exit: { type: Boolean, value: false },          // true: 左上角显示退出键，点击回身份选择页
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
      if (this.data.exit) {
        wx.reLaunch({ url: '/pages/launch/launch' });   // 退出当前身份，回到启动选择页
        return;
      }
      wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/home/home' }) });
    },
  },
});
