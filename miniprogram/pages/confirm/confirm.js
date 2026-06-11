const data = require('../../utils/data.js');
const { nav, cart } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    flavorsAll: data.FLAVORS,
    items: [],
    total: 0,
    count: 0,
    form: { contact: '', phone: '', note: '' },
    flavorMap: {},
  },
  onLoad() {
    const items = cart.list();
    this.setData({
      items,
      total: cart.total(),
      count: items.reduce((a, b) => a + b.q, 0),
    });
  },
  onInput(e) {
    const k = e.currentTarget.dataset.k;
    this.setData({ ['form.' + k]: e.detail.value });
  },
  toggleFlavor(e) {
    const f = e.currentTarget.dataset.f;
    const map = Object.assign({}, this.data.flavorMap);
    if (map[f]) delete map[f]; else map[f] = true;
    this.setData({ flavorMap: map });
  },
  pay() {
    const g = getApp().globalData;
    const list = cart.list();
    if (!list.length) return;
    const code = 'A' + (130 + Math.floor(Math.random() * 60));
    const order = {
      id: 'o' + Date.now(),
      no: 'SA24061001' + (40 + Math.floor(Math.random() * 50)),
      code,
      status: '待取餐',
      time: '06-10 16:48',
      total: cart.total(),
      contact: this.data.form.contact || '林先生',
      phone: this.data.form.phone || '138****6620',
      note: this.data.form.note || '',
      flavors: Object.keys(this.data.flavorMap),
      items: list.map(({ item, q }) => [item.id, q, item.price]),
    };
    g.orders = [order, ...g.orders];
    g.lastOrder = order;
    cart.clear();
    nav.replace('result');
  },
});
