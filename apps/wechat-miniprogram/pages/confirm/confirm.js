const api = require('../../utils/apiClient.js');
const phoneStore = require('../../utils/phoneStore.js');
const identityStore = require('../../utils/identityStore.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const transaction = require('../../utils/transactionStore.js');
const catalogStore = require('../../utils/catalogStore.js');
const menuStore = require('../../utils/reservationMenuStore.js');
const { dayLabel } = require('../../utils/dateLabel.js');
const { nav, cart, pickup } = require('../../utils/util.js');

const BUSY = ['quoting', 'preparing', 'paying', 'confirming'];
const CONFIRMING_NOTE = '正在确认支付结果，请勿重复支付';
// 联系人姓名后端按 UTF-8 字节限长 64（D8）
const CONTACT_MAX_BYTES = 64;

function utf8Bytes(value) {
  return Array.from(value).reduce((sum, char) => {
    const code = char.codePointAt(0);
    return sum + (code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4);
  }, 0);
}

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    pickup: null, items: [], count: 0, unavailableCount: 0, form: { contact: '', orderNote: '' }, noteFocus: false,
    phoneState: 'loading', maskedPhone: '', extraPhoneMasked: '', storePoint: '', storeAddress: '',
    paymentState: 'idle', payNote: '', quote: null, quoteError: '', selectionUnavailable: false, editingLocked: false, cartEmpty: true,
    payLabel: '预估合计', payBtn: '确认支付', payDisabled: true, czVisible: false, czItem: null, czInit: null, flavors: [],
    subtotal_text: '0.00', discount_text: '0.00', payable_text: '0.00',
    pickerVisible: false, pickupTimes: [], pickupState: 'loading',
  },
  onLoad(options) {
    try { this._groupKey = decodeURIComponent(options.group || ''); }
    catch (error) { nav.replace('cart-confirm'); return; }
    const group = cart.groups().find(item => item.key === this._groupKey);
    if (!group) { nav.replace('cart-confirm'); return; }
    this._selection = group.selection;
    this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() });
    this.syncPickup(); this.refreshItems();
    this.loadStorefront();
    this.loadIdentity();
    const saved = cart.payment(this._selection);
    if (saved && saved.busy) { this._foreignBusy = true; nav.replace('cart-confirm'); return; }
    if (saved) {
      this._prepayment = saved.prepayment;
      this._confirmKey = saved.confirmKey;
      this._needsPayment = saved.needsPayment;
      if (saved.quote) this.showQuote(saved.quote);
      this.setPayment({ editingLocked: true, paymentState: 'pending',
        payBtn: this._needsPayment ? '继续支付' : '刷新',
        payNote: this._needsPayment ? '支付未完成，点击继续支付' : CONFIRMING_NOTE });
    }
  },
  async onShow() {
    this._visible = true;
    if (this._paying || this._prepayment) return;
    this.syncPickup(); this.refreshItems();
    return Promise.all([this.refreshPhone(), this.loadPickupTimes()]);
  },
  syncPickup() {
    const selected = this.selection();
    this.setData({ pickup: selected ? Object.assign({}, selected, {
      periodLabel: `${dayLabel(selected.date)} ${selected.mealPeriod === 'lunch' ? '午餐' : '晚餐'}`,
      label: `${dayLabel(selected.date)} ${selected.timeConfirmed ? selected.time : '请选择时间'}`,
    }) : null });
  },
  selection() {
    const group = cart.groups().find(item => item.key === this._groupKey);
    return group ? group.selection : this._selection;
  },
  rememberPayment() {
    if (this._foreignBusy) return;
    if (this._prepayment || this._requestInFlight) cart.savePayment(this._selection, {
      prepayment: this._prepayment, confirmKey: this._confirmKey,
      needsPayment: this._needsPayment, quote: this.data.quote, busy: !!this._requestInFlight,
    });
    else if (this._selection) cart.savePayment(this._selection, null);
  },
  onHide() { this._visible = false; },
  onUnload() {
    this._pickupRequestID = (this._pickupRequestID || 0) + 1;
    this.rememberPayment();
  },
  async loadPickupTimes() {
    const requestID = (this._pickupRequestID || 0) + 1;
    this._pickupRequestID = requestID;
    this.setData({ pickupState: 'loading', pickupTimes: [] });
    this.syncPayDisabled();
    const selected = this.selection();
    try {
      const options = await menuStore.loadPickupOptions();
      if (requestID !== this._pickupRequestID || this.data.editingLocked) return false;
      const day = selected && options.dates.find(item => item.date === selected.date);
      const meal = day && day.mealPeriods.find(item => item.mealPeriod === selected.mealPeriod);
      const times = day && day.available && meal && meal.available ? meal.pickupTimes : [];
      this.setData({ pickupState: times.length ? 'ready' : 'empty', pickupTimes: times });
      if (selected && !times.includes(selected.time)) {
        cart.setSelection(Object.assign({}, selected, { timeConfirmed: false }));
        this.syncPickup();
      }
      this.syncPayDisabled();
      return times.length > 0;
    } catch (error) {
      if (requestID !== this._pickupRequestID) return false;
      this.setData({ pickupState: 'error' });
      this.syncPayDisabled();
      return false;
    }
  },
  async editPickup() {
    if (this.data.editingLocked || this._paying) return;
    this.setData({ pickerVisible: true });
    return this.loadPickupTimes();
  },
  closePicker() { this.setData({ pickerVisible: false }); },
  pickPickupTime(e) {
    if (this.data.editingLocked || this._paying || this.data.pickupState !== 'ready') return;
    const selected = this.selection();
    const time = e.currentTarget.dataset.time;
    if (!selected || !this.data.pickupTimes.includes(time)) return;
    cart.setSelection(Object.assign({}, selected, { time, timeConfirmed: true }));
    this.setData({ pickerVisible: false });
    this.syncPickup();
    this.resetPayment();
    this.refreshItems();
  },
  // 取餐地点在核价前直接显示营业设置的值（U28）
  async loadStorefront() {
    try {
      const settings = await storefrontStore.load();
      this.setData({ storePoint: settings.pickupPoint, storeAddress: settings.address });
    } catch (error) { /* 核价后以订单返回的地点为准 */ }
  },
  // 附加手机号只读：设置过才显示（U24）
  async loadIdentity() {
    try {
      const identity = await identityStore.load();
      const extra = identity.extra_phone;
      this.setData({ extraPhoneMasked: extra.set && typeof extra.masked_phone === 'string' ? extra.masked_phone : '' });
    } catch (error) { /* 附加手机号只是展示项，读不到就不显示 */ }
  },
  async refreshPhone() {
    this.setData({ phoneState: 'loading', maskedPhone: '' });
    try {
      const status = await phoneStore.status();
      this.setData({ phoneState: status.bound ? 'bound' : 'unbound', maskedPhone: status.maskedPhone });
      return status.bound;
    } catch (error) {
      this.setData({ phoneState: error && error.statusCode === 401 ? 'unauthenticated' : 'unavailable' });
      return false;
    }
  },
  async onGetPhoneNumber(e) {
    const detail = (e && e.detail) || {};
    if (typeof detail.code !== 'string' || !detail.code.trim()) return false;
    this.setData({ phoneState: 'binding' });
    try {
      const status = await phoneStore.bind(detail.code);
      this.setData({ phoneState: 'bound', maskedPhone: status.maskedPhone });
      this.selectComponent('#toast').show('手机号已绑定');
      return true;
    } catch (error) {
      this.setData({ phoneState: 'unbound', maskedPhone: '' });
      this.selectComponent('#toast').show('绑定失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  editMenu() {
    if (this.data.editingLocked) return;
    this.setData({ pickerVisible: false });
    pickup.set(this.selection());
    nav.tabTo('menu');
  },
  availability() {
    const state = getApp().globalData.menuState;
    const selected = this.selection();
    if (!state || !selected || state.date !== selected.date || state.mealPeriod !== selected.mealPeriod) return undefined;
    return state.availability;
  },
  refreshItems() {
    const availability = this.availability();
    const items = cart.list(availability, this.selection());
    const subtotal = cart.totalCents(availability, this.selection());
    const count = cart.count(availability, this.selection());
    this.setData({
      quote: null, discount_text: '0.00', payLabel: '预估合计', cartEmpty: items.length === 0,
      items, count, unavailableCount: cart.unavailableCount(availability, this.selection()),
      subtotal_text: catalogStore.formatCents(subtotal), payable_text: catalogStore.formatCents(subtotal),
    });
    this.syncPayDisabled();
  },
  syncPayDisabled() {
    const selected = this.selection();
    const disabled = BUSY.indexOf(this.data.paymentState) > -1 || (!this._prepayment &&
      (this.data.count === 0 || this.data.unavailableCount > 0 || this.data.pickupState !== 'ready' || !selected || !selected.timeConfirmed));
    if (disabled !== this.data.payDisabled) this.setData({ payDisabled: disabled });
  },
  setPayment(patch) {
    this.setData(patch);
    this.rememberPayment();
    this.syncPayDisabled();
  },
  resetPayment() {
    this.setPayment({ quoteError: '', selectionUnavailable: false, paymentState: 'idle', payBtn: '确认支付', payNote: '' });
  },
  showQuote(quote) {
    const items = quote.items.map(line => {
      const entry = cart.entry(line.product_id, this.selection());
      return {
        id: line.product_id,
        item: { id: line.product_id, name: line.name, cover: entry ? entry.product.cover : null,
          price_text: catalogStore.formatCents(line.discounted_unit_price_cents),
          original_price_text: catalogStore.formatCents(line.original_unit_price_cents),
          isStaffPrice: line.discounted_unit_price_cents < line.original_unit_price_cents },
        q: line.quantity, flavors: line.flavors, note: line.note, unavailableLabel: '',
        line_total_text: catalogStore.formatCents(line.payable_subtotal_cents),
      };
    });
    this.setData({ quote, items, count: items.reduce((sum, line) => sum + line.q, 0),
      subtotal_text: catalogStore.formatCents(quote.original_subtotal_cents),
      discount_text: catalogStore.formatCents(quote.discount_cents),
      payable_text: catalogStore.formatCents(quote.payable_cents),
      payLabel: '应付合计',
    });
  },
  onInput(e) { if (this.data.editingLocked) return; this.setData({ [`form.${e.currentTarget.dataset.k}`]: e.detail.value }); },
  focusNote() { if (!this.data.editingLocked) this.setData({ noteFocus: true }); },
  onNoteBlur() { if (this.data.noteFocus) this.setData({ noteFocus: false }); },
  editItem(e) {
    if (this.data.editingLocked) return;
    const entry = cart.entry(e.currentTarget.dataset.id, this.selection());
    if (!entry) return;
    this.setData({ czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note } });
  },
  removeItem(e) {
    if (this.data.editingLocked) return;
    cart.remove(e.currentTarget.dataset.id, this.selection());
    this.resetPayment();
    this.refreshItems();
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) {
    if (this.data.editingLocked) return;
    cart.setPrefs(this.data.czItem, e.detail, this.selection());
    this.setData({ czVisible: false });
    this.resetPayment();
    this.refreshItems();
  },
  quoteRequest() {
    const selected = this.selection();
    return {
      contact_name: this.data.form.contact.trim(),
      pickup_date: selected.date,
      pickup_time: selected.time,
      order_note: this.data.form.orderNote.trim(),
      items: cart.list(this.availability(), this.selection()).map(line => ({
        product_id: line.item.id, quantity: line.q, flavors: line.flavors.slice(), note: line.note,
      })),
    };
  },
  async confirmExisting() {
    try {
      const confirmed = await transaction.confirm(this._prepayment.id, this._confirmKey);
      if (confirmed.state !== 'ORDER_CREATED') {
        this._confirmKey = api.newIdempotencyKey('confirm');
        this.setPayment(this._needsPayment
          ? { paymentState: 'pending', payBtn: '继续支付', payNote: '支付未完成，点击继续支付' }
          : { paymentState: 'pending', payBtn: '刷新', payNote: CONFIRMING_NOTE });
        return false;
      }
      cart.complete(this._selection);
      this._prepayment = null;
      this._needsPayment = false;
      this.setData({ paymentState: 'created' });
      if (this._visible) nav.replace('result', { id: confirmed.orderID });
      return true;
    } catch (error) {
      this._confirmKey = api.newIdempotencyKey('confirm');
      this.setPayment({ paymentState: 'error', payBtn: '刷新', payNote: CONFIRMING_NOTE });
      return false;
    }
  },
  async requestExistingPayment() {
    if (!this._visible) {
      this.setPayment({ paymentState: 'pending', payBtn: '继续支付', payNote: '支付未完成，点击继续支付' });
      return false;
    }
    this.setPayment({ paymentState: 'paying', payBtn: '等待微信支付…', payNote: '' });
    try {
      const paid = await transaction.requestPayment(this._prepayment.wx_request_payment);
      this._needsPayment = !paid;
    } catch (error) {
      this._needsPayment = true;
    }
    // 微信客户端返回不能判定到账，始终查询同一笔预支付的支付结果。
    this.setPayment({ paymentState: 'confirming', payBtn: '正在确认支付结果…' });
    return this.confirmExisting();
  },
  async pay() {
    if (this._paying) return this._paying;
    if (!this._prepayment && cart.payment(this.selection())) {
      nav.replace('confirm', { group: this._groupKey });
      return false;
    }
    if (this.data.payDisabled) {
      if (this.data.unavailableCount > 0) this.selectComponent('#toast').show('请先移除不可售的菜', { icon: 'warn' });
      else if (!this.data.count) this.selectComponent('#toast').show('请先选择菜品', { icon: 'warn' });
      else this.editPickup();
      return false;
    }
    this._requestInFlight = true;
    this.rememberPayment();
    const action = this.payOnce();
    this._paying = action;
    return action.finally(() => {
      if (this._paying === action) {
        this._paying = null;
        this._requestInFlight = false;
        this.rememberPayment();
        this.setData({ editingLocked: !!this._prepayment });
      }
    });
  },
  async payOnce() {
    if (this._prepayment) {
      const continuePayment = this._needsPayment && this.data.paymentState === 'pending';
      const created = await this.confirmExisting();
      if (created || this.data.paymentState !== 'pending' || !continuePayment) return created;
      return this.requestExistingPayment();
    }
    if (!cart.count(this.availability(), this.selection()) || !this.selection()) return false;
    if (this.data.phoneState !== 'bound') {
      const unreadable = this.data.phoneState === 'unavailable' || this.data.phoneState === 'unauthenticated';
      this.selectComponent('#toast').show(unreadable ? '手机号暂时没读到，请点重试' : '请先绑定手机号', { icon: 'warn' });
      return false;
    }
    const contact = this.data.form.contact.trim();
    if (!contact) {
      this.selectComponent('#toast').show('请填写取餐人姓名', { icon: 'warn' });
      return false;
    }
    if (utf8Bytes(contact) > CONTACT_MAX_BYTES) {
      this.selectComponent('#toast').show('联系人姓名过长', { icon: 'warn' });
      return false;
    }
    this.setPayment({ paymentState: 'quoting', payBtn: '正在确认价格…', payNote: '', editingLocked: true, quoteError: '', selectionUnavailable: false });
    try {
      const quote = await transaction.createQuote(this.quoteRequest(), api.newIdempotencyKey('quote'));
      this.showQuote(quote);
      this.setPayment({ paymentState: 'preparing', payBtn: '正在发起支付…' });
      this._prepayment = await transaction.createPrepayment(quote.id, api.newIdempotencyKey('prepay'));
      this._confirmKey = api.newIdempotencyKey('confirm');
      this._needsPayment = true;
    } catch (error) {
      this._prepayment = null;
      this._needsPayment = false;
      const unavailable = !!(error && error.code === 'QUOTE_SELECTION_UNAVAILABLE');
      this.setData({ paymentState: 'error', payBtn: '重试', selectionUnavailable: unavailable,
        quoteError: unavailable ? '商品或取餐时间已不可用，请返回菜单重新选择' : '暂时无法完成本次操作，请稍后重试',
      });
      this.refreshItems();
      return false;
    }
    return this.requestExistingPayment();
  },
});
