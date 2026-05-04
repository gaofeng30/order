const store = require("../../utils/store");

Page({
  data: {
    product: {}
  },

  onLoad(query) {
    this.setData({ product: store.getProduct(query.id) });
  },

  updateField(event) {
    const field = event.currentTarget.dataset.field;
    this.setData({ [`product.${field}`]: event.detail.value });
  },

  save() {
    const products = getApp().globalData.products;
    const index = products.findIndex((item) => item.id === this.data.product.id);
    if (index >= 0) {
      products[index] = {
        ...products[index],
        ...this.data.product,
        price: Number(this.data.product.price),
        stock: Number(this.data.product.stock)
      };
    }
    wx.showToast({ title: "已保存商品", icon: "success" });
    setTimeout(() => wx.navigateBack(), 500);
  }
});
