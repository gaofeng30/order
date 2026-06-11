const data = require('../../utils/data.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    cats: data.CATS,
    isEdit: false,
    m: null,
    f: { name: '', price: '', stock: '', cat: data.CATS[0], desc: '' },
  },
  onLoad(opts) {
    if (opts.id) {
      const m = data.itemById(opts.id);
      if (m) {
        this.setData({
          isEdit: true,
          m,
          f: { name: m.name, price: String(m.price), stock: String(m.stock), cat: m.cat, desc: m.desc },
        });
      }
    }
  },
  onName(e) { this.setData({ 'f.name': e.detail.value }); },
  onPrice(e) { this.setData({ 'f.price': (e.detail.value || '').replace(/[^0-9.]/g, '') }); },
  onStock(e) { this.setData({ 'f.stock': (e.detail.value || '').replace(/[^0-9]/g, '') }); },
  onDesc(e) { this.setData({ 'f.desc': e.detail.value }); },
  pickCat(e) { this.setData({ 'f.cat': e.currentTarget.dataset.c }); },
  cancel() { wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-products/admin-products' }) }); },
  save() {
    this.selectComponent('#toast').show('已保存', { icon: 'check' });
    setTimeout(() => this.cancel(), 600);
  },
});
