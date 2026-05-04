const store = require("../../utils/store");

Page({
  data: {
    product: {},
    quantity: 1,
    inCart: 0
  },

  onLoad(query) {
    const product = store.getProduct(query.id);
    const summary = store.getCartSummary();
    const cartItem = summary.items.find((item) => item.productId === query.id);
    this.setData({
      product: {
        ...product,
        specsText: product.specs.join(" / ")
      },
      inCart: cartItem ? cartItem.quantity : 0
    });
  },

  minus() {
    this.setData({ quantity: Math.max(1, this.data.quantity - 1) });
  },

  add() {
    this.setData({ quantity: this.data.quantity + 1 });
  },

  addCart() {
    if (this.data.product.status !== "available") {
      wx.showToast({ title: "当前不可购买", icon: "none" });
      return;
    }
    store.addToCart(this.data.product.id, this.data.quantity);
    wx.showToast({ title: "已加入购物车", icon: "success" });
    setTimeout(() => wx.navigateBack(), 450);
  }
});
