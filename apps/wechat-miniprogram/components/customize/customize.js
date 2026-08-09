/* 口味偏好 + 备注 选择弹层 (菜品级, 复用于菜单/详情/确认订单) */
const data = require('../../utils/data.js');

Component({
  properties: {
    item: { type: Object, value: null },          // 菜品
    visible: { type: Boolean, value: false },
    init: { type: Object, value: null },           // { qty, flavors, note }
    confirmLabel: { type: String, value: '加入购物车' },
  },
  data: {
    chips: [],            // [{ name, on }]
    flavors: [],
    note: '',
    qty: 1,
    total: 0,
    safeBottom: 0,
  },
  observers: {
    'visible, init, item': function (visible, init, item) {
      if (!visible) return;
      const flavors = (init && init.flavors) ? init.flavors.slice() : [];
      const note = (init && init.note) || '';
      const qty = (init && init.qty) || 1;
      this.setData({
        flavors, note, qty,
        total: qty * ((item && item.price) || 0),
        chips: this._chips(flavors),
        safeBottom: getApp().globalData.safeBottom,
      });
    },
  },
  methods: {
    _chips(flavors) { return data.FLAVORS.map(f => ({ name: f, on: flavors.indexOf(f) > -1 })); },
    toggleFlavor(e) {
      const f = e.currentTarget.dataset.f;
      const flavors = this.data.flavors.slice();
      const i = flavors.indexOf(f);
      if (i > -1) flavors.splice(i, 1); else flavors.push(f);
      this.setData({ flavors, chips: this._chips(flavors) });
    },
    onNote(e) { this.setData({ note: e.detail.value }); },
    sub() { const qty = Math.max(1, this.data.qty - 1); this.setData({ qty, total: qty * this.data.item.price }); },
    add() { const qty = this.data.qty + 1; this.setData({ qty, total: qty * this.data.item.price }); },
    close() { this.triggerEvent('close'); },
    confirm() {
      this.triggerEvent('confirm', { qty: this.data.qty, flavors: this.data.flavors, note: this.data.note });
    },
    noop() {},
  },
});
