const data = require('../../utils/data.js');
const { nav, cart, orderMode } = require('../../utils/util.js');
const catalogStore = require('../../utils/catalogStore.js');

// 可选时段: 今天需在当前时间 +40 分钟之后
function slotsFor(off) { return data.RESERVE_SLOTS.filter(t => (off > 0 ? true : data.slotMins(0, t) >= 40)); }

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    cancelMin: data.CANCEL_LIMIT_MIN,
    mode: 'now',                 // now | reserve
    dates: data.RESERVE_DATES,
    dateIdx: 0,
    slots: [],
    slot: '',
    items: [],
    count: 0,
    form: { contact: '', phone: '' },
    payLabel: '应付金额',
    payBtn: '确认支付',
    // 编辑口味/备注
    czVisible: false,
    czItem: null,
    czInit: null,
    // 一期没有任何优惠机制，应付金额等于商品小计（生效 spec：员工折扣由全局折扣率承担，尚未实现）
    subtotal_cents: 0,
    subtotal_text: '0.00',
    payable_cents: 0,
    payable_text: '0.00',
  },
  onLoad() {
    const mode = orderMode.get();
    const slots = slotsFor(data.RESERVE_DATES[0].off);
    this.setData({ mode, slots, slot: slots[0] });
    this.refreshItems();
    this.syncPayLabel();
  },
  refreshItems() {
    const items = cart.list();
    const subtotal = cart.totalCents();
    this.setData({
      items,
      count: items.reduce((a, b) => a + b.q, 0),
      subtotal_cents: subtotal,
      subtotal_text: catalogStore.formatCents(subtotal),
      payable_cents: subtotal,
      payable_text: catalogStore.formatCents(subtotal),
    });
  },

  setMode(e) { this.setData({ mode: e.currentTarget.dataset.m }, () => this.syncPayLabel()); },
  pickDate(e) {
    const idx = +e.currentTarget.dataset.idx;
    const slots = slotsFor(data.RESERVE_DATES[idx].off);
    let slot = this.data.slot;
    if (slots.indexOf(slot) === -1) slot = slots[0];
    this.setData({ dateIdx: idx, slots, slot }, () => this.syncPayLabel());
  },
  pickSlot(e) { this.setData({ slot: e.currentTarget.dataset.t }, () => this.syncPayLabel()); },
  syncPayLabel() {
    if (this.data.mode === 'reserve') {
      const d = this.data.dates[this.data.dateIdx];
      this.setData({ payLabel: `预约 ${d.k} ${this.data.slot} 取`, payBtn: '提交预约' });
    } else {
      this.setData({ payLabel: '应付金额', payBtn: '确认支付' });
    }
  },
  onInput(e) { this.setData({ ['form.' + e.currentTarget.dataset.k]: e.detail.value }); },
  editItem(e) {
    const id = e.currentTarget.dataset.id;
    const entry = cart.entry(id);
    this.setData({
      czVisible: true,
      czItem: entry.product,
      czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null,
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.czItem, e.detail);
    this.setData({ czVisible: false });
    this.refreshItems();
  },
  pay() {
    const g = getApp().globalData;
    const list = cart.list();
    if (!list.length) return;
    const reserve = this.data.mode === 'reserve';
    const code = (reserve ? 'B' : 'A') + (130 + Math.floor(Math.random() * 60));
    const order = {
      id: 'o' + Date.now(),
      no: 'SA24061001' + (40 + Math.floor(Math.random() * 50)),
      code,
      status: reserve ? '已预约' : '待取餐',
      time: '06-10 16:48',
      total: this.data.payable_text,
      subtotal: this.data.subtotal_text,
      type: reserve ? 'reserve' : 'now',
      pickupPoint: data.STORE.pickupWindow,
      contact: this.data.form.contact || '林先生',
      phone: this.data.form.phone || '138****6620',
      note: '',
      flavors: [],
      items: list.map(({ item, q, flavors, note }) => [item.id, q, item.price, flavors, note]),
    };
    if (reserve) {
      const d = this.data.dates[this.data.dateIdx];
      order.pickupLabel = `${d.k} ${this.data.slot}`;
      order.minsToPickup = data.slotMins(d.off, this.data.slot);
    } else {
      order.pickupLabel = '尽快 · 约 17:10';
    }
    g.orders = [order, ...g.orders];
    g.lastOrder = order;
    cart.clear();
    nav.replace('result');
  },
});
