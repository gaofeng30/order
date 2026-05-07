const store = require("../../utils/store");

Page({
  data: {
    order: {},
    code: "",
    verified: false
  },

  onLoad(query) {
    const order = query.id ? store.getOrder(query.id) : store.getOrders("ready")[0];
    this.setData({
      order,
      code: order ? order.pickupNo : ""
    });
  },

  updateCode(event) {
    this.setData({ code: event.detail.value.toUpperCase() });
  },

  verify() {
    const orders = store.getOrders("all");
    const order = orders.find((item) => item.pickupNo === this.data.code || item.id === this.data.order.id);
    if (!order) {
      wx.showToast({ title: "未找到取餐号", icon: "none" });
      return;
    }
    const verified = store.verifyOrder(order.id);
    this.setData({ order: verified, verified: true });
    wx.showToast({ title: "核销成功", icon: "success" });
  },

  goDashboard() {
    wx.navigateTo({ url: "/pages/admin-dashboard/admin-dashboard" });
  }
});
