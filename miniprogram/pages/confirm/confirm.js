const data = require('../../utils/data.js');
const api = require('../../utils/api.js');
const promo = require('../../utils/promo.js');
const { nav, cart, orderMode } = require('../../utils/util.js');

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
    // 会员与优惠券
    isMember: false,
    level: null,
    couponId: '',       // '' 自动选最优 | 券 id | 'none' 不使用
    calc: { subtotal: '0', levelCut: '0', couponCut: '0', payable: '0', totalCut: '0', totalCutC: 0, couponCutC: 0, hasLevel: false, levelName: '', levelLabel: '', usable: [], unusable: [] },
    cpVisible: false,
    pickId: '',
  },
  onLoad() {
    const mode = orderMode.get();
    const slots = slotsFor(data.RESERVE_DATES[0].off);
    this.setData({ mode, slots, slot: slots[0] });
    this.refreshItems();
    this.syncPayLabel();
    this.loadPromo();
  },
  // 会员身份与本人可见券：真实实现由服务端按微信授权手机号命中名单后下发
  loadPromo() {
    Promise.all([api.getMyMembership(), api.listMyCoupons(), api.myCouponUsed()]).then(([me, cps, used]) => {
      this._coupons = cps;
      this._used = used;
      this.setData({ isMember: me.isMember, level: me.level }, () => this.recalc());
    });
  },
  recalc() {
    const calc = promo.calc({
      items: this.data.items,
      level: this.data.level,
      coupons: this._coupons || [],
      used: this._used || {},
      couponId: this.data.couponId,
    });
    this.setData({ calc });
  },
  refreshItems() {
    const items = cart.list();
    this.setData({ items, count: items.reduce((a, b) => a + b.q, 0) }, () => this.recalc());
  },

  // ---- 选券 ----
  openCoupon() {
    if (!this.data.calc.usable.length && !this.data.calc.unusable.length) return;
    this.setData({ cpVisible: true, pickId: this.data.couponId || this.data.calc.couponId || 'none' });
  },
  closeCoupon() { this.setData({ cpVisible: false }); },
  pickCoupon(e) { this.setData({ pickId: e.currentTarget.dataset.id }); },
  pickNone() { this.setData({ pickId: 'none' }); },
  confirmCoupon() {
    this.setData({ cpVisible: false, couponId: this.data.pickId }, () => this.recalc());
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
      czItem: data.itemById(id),
      czInit: entry ? { qty: entry.qty, flavors: entry.flavors, note: entry.note } : null,
    });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    cart.setPrefs(this.data.czItem.id, e.detail);
    this.setData({ czVisible: false });
    this.refreshItems();
  },
  pay() {
    const g = getApp().globalData;
    const list = cart.list();
    if (!list.length) return;
    const reserve = this.data.mode === 'reserve';
    const code = (reserve ? 'B' : 'A') + (130 + Math.floor(Math.random() * 60));
    const c = this.data.calc;
    const order = {
      id: 'o' + Date.now(),
      no: 'SA24061001' + (40 + Math.floor(Math.random() * 50)),
      code,
      status: reserve ? '已预约' : '待取餐',
      time: '06-10 16:48',
      total: c.payable,
      // 优惠留痕：订单详情与商户端按此展示，不再回算
      subtotal: c.subtotal,
      levelName: c.hasLevel ? c.levelName : '',
      levelLabel: c.hasLevel ? c.levelLabel : '',
      levelCut: c.levelCut,
      couponName: c.couponName,
      couponCut: c.couponCut,
      totalCut: c.totalCut,
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
    // 券按「有效期内每人可用次数」计次，支付成功即计入
    if (c.coupon) g.couponUsed[c.coupon.id] = (g.couponUsed[c.coupon.id] || 0) + 1;
    g.orders = [order, ...g.orders];
    g.lastOrder = order;
    cart.clear();
    nav.replace('result');
  },
});
