const store = require("../../utils/store");

Page({
  data: {
    tabs: [
      { id: "all", name: "全部" },
      { id: "ready", name: "待取餐" },
      { id: "completed", name: "已完成" },
      { id: "cancelled", name: "已取消" }
    ],
    active: "all",
    orders: []
  },

  onShow() {
    this.refresh();
  },

  refresh() {
    this.setData({ orders: store.getOrders(this.data.active) });
  },

  switchTab(event) {
    this.setData({ active: event.currentTarget.dataset.id });
    this.refresh();
  },

  openOrder(event) {
    wx.navigateTo({ url: `/pages/order-detail/order-detail?id=${event.currentTarget.dataset.id}` });
  },

  goMenu() {
    wx.switchTab({ url: "/pages/menu/menu" });
  }
});
