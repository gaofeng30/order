Page({
  data: {
    products: [],
    categories: []
  },

  onShow() {
    const app = getApp();
    this.setData({
      products: app.globalData.products,
      categories: app.globalData.categories
    });
  },

  editProduct(event) {
    wx.navigateTo({ url: `/pages/admin-product-edit/admin-product-edit?id=${event.currentTarget.dataset.id}` });
  },

  toggleStatus(event) {
    const id = event.currentTarget.dataset.id;
    const product = getApp().globalData.products.find((item) => item.id === id);
    if (product) {
      product.status = product.status === "available" ? "offline" : "available";
      wx.showToast({ title: product.status === "available" ? "已上架" : "已下架", icon: "none" });
      this.onShow();
    }
  },

  goCategories() {
    wx.navigateTo({ url: "/pages/admin-categories/admin-categories" });
  }
});
