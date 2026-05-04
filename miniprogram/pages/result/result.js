const store = require("../../utils/store");

Page({
  data: {
    order: {}
  },

  onLoad(query) {
    this.setData({ order: store.getOrder(query.id) });
  },

  viewOrder() {
    wx.redirectTo({ url: `/pages/order-detail/order-detail?id=${this.data.order.id}` });
  },

  goHome() {
    wx.switchTab({ url: "/pages/home/home" });
  }
});
