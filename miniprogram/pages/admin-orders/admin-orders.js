const store = require("../../utils/store");

Page({
  data: {
    orders: []
  },

  onShow() {
    this.setData({ orders: store.getOrders("all") });
  },

  openOrder(event) {
    wx.navigateTo({ url: `/pages/admin-order-detail/admin-order-detail?id=${event.currentTarget.dataset.id}` });
  },

  goVerify(event) {
    wx.navigateTo({ url: `/pages/admin-verify/admin-verify?id=${event.currentTarget.dataset.id}` });
  }
});
