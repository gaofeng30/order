Page({
  startUser() {
    wx.switchTab({ url: "/pages/home/home" });
  },
  startAdmin() {
    wx.navigateTo({ url: "/pages/admin-entry/admin-entry" });
  }
});
