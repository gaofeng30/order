Page({
  data: {
    user: {},
    settings: {}
  },

  onShow() {
    const app = getApp();
    this.setData({
      user: app.globalData.user,
      avatarText: app.globalData.user.name.slice(0, 1),
      settings: app.globalData.settings
    });
  },

  goOrders() {
    wx.switchTab({ url: "/pages/orders/orders" });
  },

  disabled() {
    wx.showToast({ title: "外卖暂未开放", icon: "none" });
  }
});
