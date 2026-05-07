Page({
  data: {
    categories: []
  },

  onShow() {
    this.setData({ categories: getApp().globalData.categories });
  },

  toggle(event) {
    const id = event.currentTarget.dataset.id;
    const item = getApp().globalData.categories.find((category) => category.id === id);
    item.enabled = !item.enabled;
    this.onShow();
  }
});
