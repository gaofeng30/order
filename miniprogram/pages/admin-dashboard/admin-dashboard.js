const store = require("../../utils/store");

Page({
  data: {
    dashboard: {},
    settings: {}
  },

  onShow() {
    this.setData({
      dashboard: store.getDashboard(),
      settings: getApp().globalData.settings
    });
  },

  goProducts() {
    wx.navigateTo({ url: "/pages/admin-products/admin-products" });
  },

  goOrders() {
    wx.navigateTo({ url: "/pages/admin-orders/admin-orders" });
  },

  goVerify() {
    wx.navigateTo({ url: "/pages/admin-verify/admin-verify" });
  },

  goSettings() {
    wx.navigateTo({ url: "/pages/admin-settings/admin-settings" });
  }
});
