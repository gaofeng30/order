const store = require("../../utils/store");

Page({
  data: {
    order: {}
  },

  onLoad(query) {
    this.setData({ order: store.getOrder(query.id) });
  },

  goVerify() {
    wx.navigateTo({ url: `/pages/admin-verify/admin-verify?id=${this.data.order.id}` });
  }
});
