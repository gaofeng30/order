const data = require('../../utils/data.js');
const { advanceOrder, advanceMeta } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { o: null, store: data.STORE, rows: [], meta: {}, flavorShow: false },
  onLoad(opts) {
    this._id = opts.id;
    this.build();
  },
  onShow() { this.build(); },
  build() {
    const orders = getApp().globalData.aOrders;
    const o = orders.find(x => x.id === this._id) || getApp().globalData._aSel || orders[0];
    const rows = o.items.map(([id, q, p]) => {
      const m = data.itemById(id);
      return { name: m.name, q, p, sub: p * q };
    });
    this.setData({ o, rows, meta: advanceMeta(o.status), flavorShow: o.flavor && o.flavor !== '—' });
  },
  advance() {
    advanceOrder(this.data.o.id, this.selectComponent('#toast'), () => this.build());
  },
  print() { this.selectComponent('#toast').show('小票已发送至打印机', { icon: 'printer' }); },
  call() { this.selectComponent('#toast').show('正在拨打 ' + this.data.o.phone, { icon: 'phone' }); },
});
