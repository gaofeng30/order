/* 口味偏好 + 备注 选择弹层 (菜品级, 复用于菜单/详情/确认订单) */
const catalogStore = require('../../utils/catalogStore.js');

Component({
  properties: {
    item: { type: Object, value: null },          // 菜品
    visible: { type: Boolean, value: false },
    init: { type: Object, value: null },           // { qty, flavors, note }
    confirmLabel: { type: String, value: '加入购物车' },
    flavorOptions: { type: Array, value: [] },
  },
  data: {
    chips: [],            // [{ name, on }]
    flavors: [],
    note: '',
    qty: 1,
    totalCents: 0,
    totalText: '0.00',
    safeBottom: 0,
  },
  observers: {
    'visible, init, item': function (visible, init, item) {
      if (!visible) return;
      const flavors = (init && init.flavors) ? init.flavors.slice() : [];
      const note = (init && init.note) || '';
      const qty = (init && init.qty) || 1;
      const totalCents = qty * ((item && item.price_cents) || 0);
      this.setData({
        flavors, note, qty,
        totalCents,
        totalText: catalogStore.formatCents(totalCents),
        chips: this._chips(flavors),
        safeBottom: getApp().globalData.safeBottom,
      });
    },
  },
  methods: {
    _chips(selected) { return this.data.flavorOptions.map(name => ({ name, on: selected.indexOf(name) > -1 })); },
    toggleFlavor(e) {
      const f = e.currentTarget.dataset.f;
      const flavors = this.data.flavors.slice();
      const i = flavors.indexOf(f);
      if (i > -1) flavors.splice(i, 1); else flavors.push(f);
      this.setData({ flavors, chips: this._chips(flavors) });
    },
    onNote(e) { this.setData({ note: e.detail.value }); },
    sub() {
      const qty = Math.max(1, this.data.qty - 1);
      const totalCents = qty * this.data.item.price_cents;
      this.setData({ qty, totalCents, totalText: catalogStore.formatCents(totalCents) });
    },
    add() {
      const qty = this.data.qty + 1;
      const totalCents = qty * this.data.item.price_cents;
      this.setData({ qty, totalCents, totalText: catalogStore.formatCents(totalCents) });
    },
    close() { this.triggerEvent('close'); },
    confirm() {
      this.triggerEvent('confirm', { qty: this.data.qty, flavors: this.data.flavors, note: this.data.note });
    },
    noop() {},
  },
});
