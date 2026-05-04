const store = require("../../utils/store");

Page({
  data: {
    settings: {},
    products: [],
    cartQuantity: 0
  },

  onShow() {
    const app = getApp();
    const summary = store.getCartSummary();
    this.setData({
      settings: app.globalData.settings,
      products: app.globalData.products.filter((item) => item.status === "available").slice(0, 3),
      cartQuantity: summary.totalQuantity
    });
  },

  goMenu() {
    wx.switchTab({ url: "/pages/menu/menu" });
  },

  goProfile() {
    wx.switchTab({ url: "/pages/profile/profile" });
  },

  openDetail(event) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${event.detail.id}` });
  },

  addProduct(event) {
    store.addToCart(event.detail.id, 1);
    wx.showToast({ title: "已加入购物车", icon: "success" });
    this.onShow();
  }
});
