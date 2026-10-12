const api = require('../../utils/apiClient.js');
const phoneStore = require('../../utils/phoneStore.js');
const transaction = require('../../utils/transactionStore.js');
const catalogStore = require('../../utils/catalogStore.js');
const { nav, cart, pickup } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    pickup: null, items: [], count: 0, form: { contact: '', orderNote: '', extraPhone: '' },
    phoneState: 'loading', maskedPhone: '', paymentState: 'idle', quote: null, quoteError: '', selectionUnavailable: false, editingLocked: false, extraPhoneSaving: false, cartEmpty: true,
    payLabel: '预估合计', payBtn: '确认支付', czVisible: false, czItem: null, czInit: null, flavors: [],
    subtotal_cents: 0, subtotal_text: '0.00', discount_text: '0.00', payable_cents: 0, payable_text: '0.00',
  },
  onLoad() { this.setData({ flavors: (getApp().globalData.storefrontFlavors || []).slice() }); this.syncPickup(); this.refreshItems(); },
  async onShow() {
    if (this._paying || this._prepayment) return;
    this.syncPickup(); this.refreshItems();
    if (this._editingPickup) {
      this._editingPickup = false;
      this.setData({ quoteError: '', selectionUnavailable: false, paymentState: 'idle', payBtn: '确认支付' });
    }
    return this.refreshPhone();
  },
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
    if (this.data.editingLocked || this.data.extraPhoneSaving) return false;
    const phone = this.data.form.extraPhone.trim();
    const name = this.data.form.contact.trim();
    if (!phone || !name) return false;
    this.setData({ extraPhoneSaving: true });
    try {
      const result = await phoneStore.setExtra(phone, name, api.newIdempotencyKey('extra-phone'));
      this.setData({ maskedPhone: result.extraPhone.masked_phone });
      return true;
    } catch (error) { return false; }
    finally { this.setData({ extraPhoneSaving: false }); }
  },
  editPickup() {
    if (this.data.editingLocked) return;
    this._editingPickup = true;
    wx.navigateTo({ url: '/pages/menu/menu?returnToConfirm=1', fail: () => {
      this._editingPickup = false;
      this.selectComponent('#toast').show('暂时无法打开菜单，请稍后重试', { icon: 'warn' });
    } });
  },
  refreshItems() {
    const items = cart.list();
    const subtotal = cart.totalCents();
    this.setData({
      quote: null, discount_text: '0.00', payLabel: '预估合计', cartEmpty: items.length === 0,
      items, count: items.reduce((sum, item) => sum + item.q, 0),
      subtotal_cents: subtotal, subtotal_text: catalogStore.formatCents(subtotal),
      payable_cents: subtotal, payable_text: catalogStore.formatCents(subtotal),
    });
  },
  showQuote(quote) {
    const items = quote.items.map(line => ({
      item: { id: line.product_id, name: line.name,
        price_text: catalogStore.formatCents(line.discounted_unit_price_cents),
        original_price_text: catalogStore.formatCents(line.original_unit_price_cents),
        isStaffPrice: line.discounted_unit_price_cents < line.original_unit_price_cents },
      q: line.quantity, flavors: line.flavors, note: line.note,
      line_total_text: catalogStore.formatCents(line.payable_subtotal_cents),
    }));
    this.setData({ quote, items, count: items.reduce((sum, line) => sum + line.q, 0),
      subtotal_cents: quote.original_subtotal_cents,
      subtotal_text: catalogStore.formatCents(quote.original_subtotal_cents),
      discount_text: catalogStore.formatCents(quote.discount_cents),
      payable_cents: quote.payable_cents, payable_text: catalogStore.formatCents(quote.payable_cents),
      payLabel: '应付合计',
    });
  },
  onInput(e) { if (this.data.editingLocked) return; this.setData({ [`form.${e.currentTarget.dataset.k}`]: e.detail.value }); },
  editItem(e) {
    if (this.data.editingLocked) return;
    const entry = cart.entry(e.currentTarget.dataset.id);
    if (!entry) return;
    this.setData({ czVisible: true, czItem: entry.product,
      czInit: { qty: entry.qty, flavors: entry.flavors, note: entry.note } });
  },
  onCzClose() { this.setData({ czVisible: false }); },
  onCzConfirm(e) { if (this.data.editingLocked) return; cart.setPrefs(this.data.czItem, e.detail); this.setData({ czVisible: false, quoteError: '', selectionUnavailable: false, paymentState: 'idle', payBtn: '确认支付' }); this.refreshItems(); },
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
        this.setData({ paymentState: 'pending', payBtn: this._needsPayment ? '继续支付' : '重新确认支付结果' });
        return false;
      }
      cart.clear();
      this._prepayment = null;
      this._needsPayment = false;
      this.setData({ paymentState: 'created' });
      nav.replace('result', { id: confirmed.orderID });
      return true;
    } catch (error) {
      this._confirmKey = api.newIdempotencyKey('confirm');
      this.setData({ paymentState: 'error', payBtn: '重新确认支付结果' });
      return false;
    }
  },
  async requestExistingPayment() {
    this.setData({ paymentState: 'paying', payBtn: '等待微信支付…' });
    try {
      const paid = await transaction.requestPayment(this._prepayment.wx_request_payment);
      this._needsPayment = !paid;
    } catch (error) {
      this._needsPayment = true;
    }
    // 微信客户端返回不能判定到账，始终查询同一笔预支付的服务端结果。
    this.setData({ paymentState: 'confirming', payBtn: '正在确认支付结果…' });
    return this.confirmExisting();
  },
  async pay() {
    if (this._paying) return this._paying;
    const action = this.payOnce();
    this._paying = action;
    return action.finally(() => {
      if (this._paying === action) {
        this._paying = null;
        this.setData({ editingLocked: !!this._prepayment });
      }
    });
  },
  async payOnce() {
    if (!cart.count() || !pickup.get()) return false;
    if (this.data.extraPhoneSaving) {
      this.selectComponent('#toast').show('正在保存手机号，请稍候', { icon: 'warn' });
      return false;
    }
    if (this._prepayment) {
      const continuePayment = this._needsPayment && this.data.paymentState === 'pending';
      const created = await this.confirmExisting();
      if (created || this.data.paymentState !== 'pending' || !continuePayment) return created;
      return this.requestExistingPayment();
    }
    if (this.data.phoneState !== 'bound') {
      this.selectComponent('#toast').show('请先绑定手机号', { icon: 'warn' });
      return false;
    }
    if (!this.data.form.contact.trim()) {
      this.selectComponent('#toast').show('请填写取餐人姓名', { icon: 'warn' });
      return false;
    }
    this.setData({ paymentState: 'quoting', payBtn: '正在核价…', editingLocked: true, quoteError: '', selectionUnavailable: false });
    try {
      const quote = await transaction.createQuote(this.quoteRequest(), api.newIdempotencyKey('quote'));
      this.showQuote(quote);
      this.setData({ paymentState: 'preparing', payBtn: '正在创建支付…' });
      this._prepayment = await transaction.createPrepayment(quote.id, api.newIdempotencyKey('prepay'));
      this._confirmKey = api.newIdempotencyKey('confirm');
      this._needsPayment = true;
    } catch (error) {
      this._prepayment = null;
      this._needsPayment = false;
      this.refreshItems();
      this.setData({ paymentState: 'error', payBtn: '重试',
        selectionUnavailable: error && error.code === 'QUOTE_SELECTION_UNAVAILABLE',
        quoteError: error && error.code === 'QUOTE_SELECTION_UNAVAILABLE'
          ? '商品或取餐时间已不可用，请返回菜单重新选择' : '暂时无法完成本次操作，请稍后重试',
      });
      return false;
    }
    return this.requestExistingPayment();
  },
});
