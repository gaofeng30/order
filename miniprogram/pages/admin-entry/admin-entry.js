Page({
  goDashboard() {
    wx.navigateTo({ url: "/pages/admin-dashboard/admin-dashboard" });
  },
  goOrders() {
    wx.navigateTo({ url: "/pages/admin-orders/admin-orders" });
  },
  goVerify() {
    wx.navigateTo({ url: "/pages/admin-verify/admin-verify" });
  }
});
