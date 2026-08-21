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
    // 名称取订单自身的快照，不回查菜品表（§15.6.2）
    const rows = o.items.map(([, name, q, p, dp, flavor, note]) => (
      { name, q, p: dp, sub: dp * q, band: [flavor, note].filter(Boolean).join(' · ') }
    ));
    // 整单级口味已删除：聚合行内口味展示（§15.6.4）
    const flavorText = [...new Set(rows.map(r => r.band).filter(Boolean))].join('；');
    this.setData({ o, rows, meta: advanceMeta(o.status), flavorText, flavorShow: !!flavorText,
                   paidTime: String(o.paidAt).slice(11, 16) });
  },
  advance() {
    advanceOrder(this.data.o.id, this.selectComponent('#toast'), () => this.build());
  },
  print() { this.selectComponent('#toast').show('小票已发送至打印机', { icon: 'printer' }); },
  call() { this.selectComponent('#toast').show('正在拨打 ' + this.data.o.phone, { icon: 'phone' }); },
});
