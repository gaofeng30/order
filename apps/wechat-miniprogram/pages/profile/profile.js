const identityStore = require('../../utils/identityStore.js');
const phoneStore = require('../../utils/phoneStore.js');
const merchantLoginStore = require('../../utils/merchantLoginStore.js');
const api = require('../../utils/apiClient.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    identityState: 'loading', pend: 0, nick: '微信用户', avatarText: '客', avatarUrl: '',
    phoneMask: '', extraPhoneMask: '', pricingKind: 'VISITOR', merchantBound: false, merchantLoginState: 'idle',
    extraForm: { phone: '', name: '' }, extraState: 'idle',
  },
  async onShow() {
    this.setData({ identityState: 'loading', pend: 0 });
    const identityTask = identityStore.load().then(identity => {
      this.setData({
        identityState: 'ready', phoneMask: identity.primary_phone.masked_phone || '',
        extraPhoneMask: identity.extra_phone.masked_phone || '', pricingKind: identity.pricing_identity.kind,
        merchantBound: identity.merchant.bound,
      });
    }).catch(() => this.setData({ identityState: 'error' }));
    const ordersTask = api.get('/api/v1/orders?active=true', true).then(body => {
      if (!body || !Array.isArray(body.orders)) throw new Error('orders');
      this.setData({ pend: body.orders.length });
    }).catch(() => this.setData({ pend: 0 }));
    await Promise.all([identityTask, ordersTask]);
  },
  chooseProfile() {
    if (!wx.getUserProfile) return Promise.resolve(false);
    return new Promise(resolve => wx.getUserProfile({ desc: '仅用于本次显示头像昵称',
      success: result => {
        const info = result && result.userInfo || {};
        const nick = typeof info.nickName === 'string' && info.nickName.trim() ? info.nickName.trim() : '微信用户';
        this.setData({ nick, avatarText: nick.slice(0, 1), avatarUrl: typeof info.avatarUrl === 'string' ? info.avatarUrl : '' });
        resolve(true);
      }, fail: () => resolve(false) }));
  },
  async onGetPhoneNumber(e) {
    const code = e && e.detail && e.detail.code;
    if (!code) return false;
    try { const status = await phoneStore.bind(code); this.setData({ phoneMask: status.maskedPhone }); return true; }
    catch (error) { return false; }
  },
  async onMerchantPhone(e) {
    const code = e && e.detail && e.detail.code;
    if (!code) return false;
    this.setData({ merchantLoginState: 'loading' });
    try {
      const merchant = await merchantLoginStore.login(code);
      getApp().globalData.entryRouting = { state: 'merchant', role: merchant.role };
      this.setData({ merchantBound: true, merchantLoginState: 'ready' });
      nav.reset();
      return true;
    } catch (error) {
      this.setData({ merchantLoginState: 'error' });
      return false;
    }
  },
  onExtraInput(e) { this.setData({ [`extraForm.${e.currentTarget.dataset.k}`]: e.detail.value }); },
  async saveExtraPhone() {
    this.setData({ extraState: 'saving' });
    try {
      const result = await phoneStore.setExtra(this.data.extraForm.phone.trim(), this.data.extraForm.name.trim(), api.newIdempotencyKey('extra-phone'));
      const kind = result.pricingIdentity && result.pricingIdentity.kind;
      if (kind !== 'STAFF' && kind !== 'VISITOR') {
        this.setData({ extraState: 'error' });
        return false;
      }
      this.setData({
        extraPhoneMask: result.extraPhone.masked_phone,
        pricingKind: kind,
        extraState: kind === 'STAFF' ? 'matched' : 'unmatched',
      });
      return true;
    } catch (error) { this.setData({ extraState: 'error' }); return false; }
  },
  toOrders() { nav.tabTo('orders'); },
  reset() { nav.reset(); },
});
