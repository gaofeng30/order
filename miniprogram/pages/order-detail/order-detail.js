const store = require("../../utils/store");

Page({
  data: {
    order: {},
    qrCells: []
  },

  onLoad(query) {
    const order = store.getOrder(query.id);
    this.setData({
      order,
      qrCells: Array.from({ length: 49 }).map((_, index) => ({
        id: index,
        active: [0, 1, 2, 6, 7, 8, 12, 14, 18, 20, 23, 24, 28, 30, 34, 36, 40, 41, 42, 46, 47, 48].includes(index)
      }))
    });
  },

  goOrders() {
    wx.switchTab({ url: "/pages/orders/orders" });
  },

  copyCode() {
    wx.setClipboardData({ data: this.data.order.pickupNo });
  }
});
