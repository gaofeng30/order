const store = require("../../utils/store");

Page({
  data: {
    cart: {},
    settings: {},
    tasteOptions: [
      { label: "免葱", selected: false },
      { label: "免葱花", selected: false },
      { label: "免辣", selected: false },
      { label: "少辣", selected: false },
      { label: "少盐", selected: false },
      { label: "不要香菜", selected: false }
    ],
    form: {
      contact: "",
      phone: "",
      remark: ""
    }
  },

  onLoad() {
    const app = getApp();
    const cart = store.getCartSummary();
    this.setData({
      cart,
      settings: app.globalData.settings,
      form: {
        contact: app.globalData.user.name,
        phone: app.globalData.user.phone,
        remark: ""
      }
    });
  },

  updateField(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({
      [`form.${field}`]: event.detail.value
    });
  },

  toggleTaste(event) {
    const value = event.currentTarget.dataset.value;
    const tasteOptions = this.data.tasteOptions.map((item) => ({
      ...item,
      selected: item.label === value ? !item.selected : item.selected
    }));
    this.setData({ tasteOptions });
  },

  submitOrder() {
    if (!this.data.cart.totalQuantity) {
      wx.showToast({ title: "购物车为空", icon: "none" });
      return;
    }
    const order = store.createOrder({
      ...this.data.form,
      tastes: this.data.tasteOptions.filter((item) => item.selected).map((item) => item.label)
    });
    wx.redirectTo({ url: `/pages/result/result?id=${order.id}` });
  }
});
