const api = require('../../utils/apiClient.js');
const phoneStore = require('../../utils/phoneStore.js');
const transaction = require('../../utils/transactionStore.js');
const catalogStore = require('../../utils/catalogStore.js');
const { nav, cart, pickup } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    pickup: null, items: [], count: 0, form: { contact: '', orderNote: '', extraPhone: '' },
    phoneState: 'loading', maskedPhone: '', paymentState: 'idle', quote: null,
    payLabel: '服务端应付金额', payBtn: '确认支付', czVisible: false, czItem: null, czInit: null, flavors: [],
    subtotal_cents: 0, subtotal_text: '0.00', payable_cents: 0, payable_text: '0.00',
  },
  onLoad() { this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() }); this.syncPickup(); this.refreshItems(); },
  async onShow() { this.syncPickup(); return this.refreshPhone(); },
  syncPickup() {
    const selected = pickup.get();
    this.setData({ pickup: selected ? Object.assign({}, selected, { label: pickup.label() }) : null });
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
      return true;
    } catch (error) {
      this.setData({ phoneState: 'unavailable', maskedPhone: '' });
      return false;
    }
  },
  async saveExtraPhone() {
    const phone = this.data.form.extraPhone.trim();
    const name = this.data.form.contact.trim();
    if (!phone || !name) return false;
    try {
      const result = await phoneStore.setExtra(phone, name, api.newIdempotencyKey('extra-phone'));
      this.setData({ maskedPhone: result.extraPhone.masked_phone });
      return true;
    } catch (error) { return false; }
  },
  editPickup() { nav.tabTo('menu'); },
  refreshItems() {
    const items = cart.list();
    const subtotal = cart.totalCents();
    this.setData({
      items, count: items.reduce((sum, item) => sum + item.q, 0),
      subtotal_cents: subtotal, subtotal_text: catalogStore.formatCents(subtotal),
      payable_cents: subtotal, payable_text: catalogStore.formatCents(subtotal),
    });
  },
  onInput(e) { this.setData({ [`form.${e.currentTarget.dataset.k}`]: e.detail.value }); },
  editItem(e) {
    const entry = cart.entry(e.currentTarget.dataset.id);
    if (!entry) return;
    this.setData({ czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note } });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) { cart.setPrefs(this.data.czItem, e.detail); this.setData({ czVisible: false }); this.refreshItems(); },
  quoteRequest() {
    const selected = pickup.get();
    return {
      contact_name: this.data.form.contact.trim(),
      pickup_date: selected.date,
      pickup_time: selected.time,
      order_note: this.data.form.orderNote.trim(),
      items: cart.list().map(line => ({
        product_id: line.item.id, quantity: line.q, flavors: line.flavors.slice(), note: line.note,
      })),
    };
  },
  async confirmExisting() {
    try {
      const confirmed = await transaction.confirm(this._prepayment.id, this._confirmKey);
      if (confirmed.state !== 'ORDER_CREATED') {
		this._confirmKey = api.newIdempotencyKey('confirm');
        this.setData({ paymentState: 'pending', payBtn: '重新确认支付结果' });
        return false;
      }
      cart.clear();
      this._prepayment = null;
      this.setData({ paymentState: 'created' });
      nav.replace('result', { id: confirmed.orderID });
      return true;
    } catch (error) {
      this.setData({ paymentState: 'error', payBtn: '重试' });
      return false;
    }
  },
  async pay() {
    if (this._paying) return this._paying;
    const action = this.payOnce();
    this._paying = action;
    return action.finally(() => { if (this._paying === action) this._paying = null; });
  },
  async payOnce() {
    if (!cart.count() || !pickup.get()) return false;
    if (this._prepayment) return this.confirmExisting();
    if (this.data.phoneState !== 'bound') {
      this.selectComponent('#toast').show('请先绑定手机号', { icon: 'warn' });
      return false;
    }
    if (!this.data.form.contact.trim()) {
      this.selectComponent('#toast').show('请填写取餐人姓名', { icon: 'warn' });
      return false;
    }
    this.setData({ paymentState: 'quoting', payBtn: '正在核价…' });
    try {
      const quote = await transaction.createQuote(this.quoteRequest(), api.newIdempotencyKey('quote'));
      this.setData({
        quote, payable_cents: quote.payable_cents,
        payable_text: catalogStore.formatCents(quote.payable_cents), paymentState: 'preparing', payBtn: '正在创建支付…',
      });
      this._prepayment = await transaction.createPrepayment(quote.id, api.newIdempotencyKey('prepay'));
      this._confirmKey = api.newIdempotencyKey('confirm');
      this.setData({ paymentState: 'paying', payBtn: '等待微信支付…' });
      await transaction.requestPayment(this._prepayment.wx_request_payment);
      this.setData({ paymentState: 'confirming', payBtn: '正在确认支付结果…' });
      return this.confirmExisting();
    } catch (error) {
      this._prepayment = null;
      this.setData({ paymentState: 'error', payBtn: '重试', quote: null,
        payable_cents: this.data.subtotal_cents, payable_text: this.data.subtotal_text });
      return false;
    }
  },
});
