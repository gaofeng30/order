Page({
  data: {
    settings: {}
  },

  onLoad() {
    this.setData({ settings: { ...getApp().globalData.settings } });
  },

  updateField(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({ [`settings.${field}`]: event.detail.value });
  },

  setStatus(event) {
    this.setData({ "settings.status": event.currentTarget.dataset.status });
  },

  save() {
    getApp().globalData.settings = { ...this.data.settings };
    wx.showToast({ title: "已保存营业设置", icon: "success" });
  }
});
