const data = require('../../utils/data.js');
const api = require('../../utils/api.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    cats: ['全部'].concat(data.CATS),
    cat: '全部',
    list: [],
  },
  onShow() { this.build(); },
  build() {
    const list = data.menuList().filter(m => this.data.cat === '全部' || m.cat === this.data.cat).map(m => {
      const s = m.status;
      const low = m.stock <= 8;
      return {
        id: m.id, name: m.name, price: m.price, stock: m.stock, sold: m.sold, cat: m.cat, img: m.img,
        imgCount: (m.imgs || []).length,
        s, low,
        pillLabel: s === 'on' ? '可购' : (s === 'soldout' ? '售罄' : '已下架'),
        pillTone: s === 'on' ? 'ok' : 'mute',
        soldoutLabel: s === 'soldout' ? '恢复售卖' : '标记售罄',
        on: s !== 'off',
      };
    });
    this.setData({ list });
  },
  switchCat(e) { this.setData({ cat: e.currentTarget.dataset.c }, () => this.build()); },
  toggleSoldout(e) {
    const id = e.currentTarget.dataset.id;
    const cur = data.itemById(id);
    const nx = cur.status === 'soldout' ? 'on' : 'soldout';
    api.setProductStatus(id, nx).then(() => {
      this.build();
      this.selectComponent('#toast').show(nx === 'soldout' ? '已置售罄' : '已恢复售卖', { icon: 'tag' });
    });
  },
  toggleShelf(e) {
    const id = e.currentTarget.dataset.id;
    const cur = data.itemById(id);
    const nx = cur.status === 'off' ? 'on' : 'off';
    api.setProductStatus(id, nx).then(() => {
      this.build();
      this.selectComponent('#toast').show(nx === 'on' ? '已上架' : '已下架', { icon: nx === 'on' ? 'check' : 'box' });
    });
  },
  edit(e) { nav.go('admin-product-edit', { id: e.currentTarget.dataset.id }); },
  newProduct() { nav.go('admin-product-edit'); },
});
