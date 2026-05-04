const store = require("../../utils/store");

Page({
  data: {
    categories: [],
    activeCategory: "",
    products: [],
    cart: {},
    cartOpen: false
  },

  onShow() {
    const app = getApp();
    const activeCategory = this.data.activeCategory || "all";
    this.setData({
      categories: [{ id: "all", name: "全部菜品" }, ...app.globalData.categories],
      activeCategory
    });
    this.refreshProducts();
  },

  refreshProducts() {
    const summary = store.getCartSummary();
    const quantityMap = {};
    summary.items.forEach((item) => {
      quantityMap[item.productId] = item.quantity;
    });
    const source = this.data.activeCategory === "all"
      ? getApp().globalData.products
      : store.getProductsByCategory(this.data.activeCategory);
    const products = source.map((item) => ({
      ...item,
      quantity: quantityMap[item.id] || 0
    }));

    this.setData({ products, cart: summary });
  },

  switchCategory(event) {
    this.setData({ activeCategory: event.currentTarget.dataset.id });
    this.refreshProducts();
  },

  openDetail(event) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${event.detail.id}` });
  },

  addProduct(event) {
    const id = event.detail.id || event.currentTarget.dataset.id;
    store.addToCart(id, 1);
    this.refreshProducts();
  },

  minusProduct(event) {
    const id = event.detail.id || event.currentTarget.dataset.id;
    store.addToCart(id, -1);
    this.refreshProducts();
  },

  toggleCart() {
    if (!this.data.cart.totalQuantity) {
      wx.showToast({ title: "购物车为空", icon: "none" });
      return;
    }
    this.setData({ cartOpen: !this.data.cartOpen });
  },

  clearCart() {
    store.clearCart();
    this.setData({ cartOpen: false });
    this.refreshProducts();
  },

  goConfirm() {
    if (!this.data.cart.totalQuantity) {
      wx.showToast({ title: "请先选择菜品", icon: "none" });
      return;
    }
    wx.navigateTo({ url: "/pages/confirm/confirm" });
  }
});
