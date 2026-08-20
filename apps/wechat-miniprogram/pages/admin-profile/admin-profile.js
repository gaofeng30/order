const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { storeStatus: '营业中' },
  onShow() { this.setData({ storeStatus: getApp().globalData.store.status }); },
  toSettings() { nav.go('admin-settings'); },
  toCategories() { nav.go('admin-categories'); },
  toLayer() { nav.go('admin-layer'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  shift() { this.toast('交班对账 · 建设中', 'refresh'); },
  members() { this.toast('成员管理 · 建设中', 'user'); },
  service() { this.toast('正在拨打 400-021-8866', 'phone'); },
  reset() { nav.reset(); },
});
