const data = require('../../utils/data.js');
const { nav, cart } = require('../../utils/util.js');
const catalogStore = require('../../utils/catalogStore.js');

// 可选时段: 今天需在当前时间 +40 分钟之后
function slotsFor(off) { return data.RESERVE_SLOTS.filter(t => (off > 0 ? true : data.slotMins(0, t) >= 40)); }

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: data.STORE,
    cancelMin: data.CANCEL_LIMIT_MIN,
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
    const slots = slotsFor(data.RESERVE_DATES[0].off);
    this.setData({ slots, slot: slots[0] });
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
    const d = this.data.dates[this.data.dateIdx];
    this.setData({ payLabel: `预约 ${d.k} ${this.data.slot} 取`, payBtn: '提交预约' });
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
    const d = this.data.dates[this.data.dateIdx];
    const minsToPickup = data.slotMins(d.off, this.data.slot);
    // 取餐号为 4 位数字，按取餐日期累计；即时单已删除，不再有前缀
    const code = String(130 + Math.floor(Math.random() * 60)).padStart(4, '0');
    const order = {
      id: 'o' + Date.now(),
      no: 'SA24061001' + (40 + Math.floor(Math.random() * 50)),
      code,
      // 支付成功即建单。距取餐不足 30 分钟时直接进 制作中（生效 spec §7.4）
      status: minsToPickup <= data.CANCEL_LIMIT_MIN ? '制作中' : '已预约',
      time: '06-10 16:48',
      total: this.data.payable_text,
      subtotal: this.data.subtotal_text,
      pickupPoint: data.STORE.pickupWindow,
      contact: this.data.form.contact || '林先生',
      phone: this.data.form.phone || '138****6620',
      note: '',
      flavors: [],
      items: list.map(({ item, q, flavors, note }) => [item.id, q, item.price, flavors, note]),
    };
    order.pickupLabel = `${d.k} ${this.data.slot}`;
    order.minsToPickup = minsToPickup;
    g.orders = [order, ...g.orders];
    g.lastOrder = order;
    cart.clear();
    nav.replace('result');
  },
});
