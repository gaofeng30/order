const identityStore = require('../../utils/identityStore.js');
const phoneStore = require('../../utils/phoneStore.js');
const merchantLoginStore = require('../../utils/merchantLoginStore.js');
const storefrontStore = require('../../utils/storefrontStore.js');
const api = require('../../utils/apiClient.js');
const { nav } = require('../../utils/util.js');

// 营业设置未配置客服电话（B2）时显示的占位号码
const SERVICE_PHONE_PLACEHOLDER = '13945392488';
const EXTRA_PHONE = /^1\d{10}$/;

function textOf(value) { return typeof value === 'string' ? value.trim() : ''; }

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    identityState: 'loading', pend: 0, nick: '微信用户', avatarText: '',
    phoneMask: '', phoneBinding: false, extraPhoneMask: '', merchantBound: false, merchantLoginState: 'idle',
    extraForm: { phone: '', name: '' }, extraSaving: false,
    // 附加手机号默认收起
    extraOpen: false, extraName: '', extraSavable: false,
    servicePhone: SERVICE_PHONE_PLACEHOLDER,
  },
  onLoad() { this.loadServicePhone(); },
  reloadIdentity() { return this.onShow(); },
  async onShow() {
    this.setData({ identityState: 'loading', pend: 0 });
    const identityTask = identityStore.load().then(identity => {
      // 姓名取值顺序：员工白名单姓名（B3）→ 自己填写的姓名 → 无（U45）
      this._staffName = textOf(identity.pricing_identity.staff_name);
      // 接口只回脱敏手机号，无法预填；姓名是明文，可以。
      const extraName = textOf(identity.extra_phone.name);
      this.setData({
        identityState: 'ready', phoneMask: identity.primary_phone.masked_phone || '',
        extraPhoneMask: identity.extra_phone.masked_phone || '',
        merchantBound: identity.merchant.bound,
        extraName, 'extraForm.name': extraName,
      });
      this.refreshName();
      this.refreshExtraSavable();
    }).catch(() => this.setData({ identityState: 'error' }));
    const ordersTask = api.get('/api/v1/orders?active=true', true).then(body => {
      if (!body || !Array.isArray(body.orders)) throw new Error('orders');
      this.setData({ pend: body.orders.length });
    }).catch(() => this.setData({ pend: 0 }));
    await Promise.all([identityTask, ordersTask]);
  },
  async loadServicePhone() {
    try {
      const settings = await storefrontStore.load();
      if (settings.servicePhone) this.setData({ servicePhone: settings.servicePhone });
    } catch (error) { /* 读不到营业设置时保留占位号码 */ }
  },
  refreshName() {
    const name = this._staffName || this.data.extraName;
    this.setData({ nick: name || '微信用户', avatarText: name ? Array.from(name)[0] : '' });
  },
  callService() {
    const phone = this.data.servicePhone;
    wx.showModal({
      title: '客服电话', content: phone, confirmText: '复制号码', cancelText: '取消',
      success: res => { if (res.confirm) wx.setClipboardData({ data: phone }); },
    });
  },
  async onGetPhoneNumber(e) {
    const code = e && e.detail && e.detail.code;
    if (!code || this.data.phoneBinding) return false;
    this.setData({ phoneBinding: true });
    try {
      const status = await phoneStore.bind(code);
      this.setData({ phoneMask: status.maskedPhone, phoneBinding: false });
      this.selectComponent('#toast').show('手机号已绑定');
      return true;
    } catch (error) {
      this.setData({ phoneBinding: false });
      this.selectComponent('#toast').show('绑定失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  async onMerchantPhone(e) {
    const code = e && e.detail && e.detail.code;
    if (!code || this.data.merchantLoginState === 'loading') return false;
    this.setData({ merchantLoginState: 'loading' });
    try {
      const merchant = await merchantLoginStore.login(code);
      getApp().globalData.entryRouting = { state: 'merchant', role: merchant.role };
      this.setData({ merchantBound: true, merchantLoginState: 'ready' });
      nav.reset();
      return true;
    } catch (error) {
      this.setData({ merchantLoginState: 'error' });
      this.selectComponent('#toast').show('商户登录失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  toggleExtra() { this.setData({ extraOpen: !this.data.extraOpen }); },
  // 手机号 + 姓名双要素（PRD §4.1），缺一不发请求（D8）
  refreshExtraSavable() {
    const { phone, name } = this.data.extraForm;
    const savable = EXTRA_PHONE.test(phone.trim()) && !!name.trim();
    if (savable !== this.data.extraSavable) this.setData({ extraSavable: savable });
  },
  onExtraInput(e) {
    this.setData({ [`extraForm.${e.currentTarget.dataset.k}`]: e.detail.value });
    this.refreshExtraSavable();
  },
  async saveExtraPhone() {
    if (this.data.extraSaving) return false;
    if (!this.data.extraSavable) {
      const { phone, name } = this.data.extraForm;
      const reason = !EXTRA_PHONE.test(phone.trim()) ? '请输入 11 位手机号' : !name.trim() ? '请填写姓名' : '';
      if (reason) this.selectComponent('#toast').show(reason, { icon: 'warn' });
      return false;
    }
    this.setData({ extraSaving: true });
    try {
      const result = await phoneStore.setExtra(this.data.extraForm.phone.trim(), this.data.extraForm.name.trim(), api.newIdempotencyKey('extra-phone'));
      const pricing = result.pricingIdentity || {};
      if (pricing.kind !== 'STAFF' && pricing.kind !== 'VISITOR') throw new Error('pricing identity');
      this._staffName = textOf(pricing.staff_name);
      this.setData({
        extraPhoneMask: result.extraPhone.masked_phone,
        extraName: textOf(result.extraPhone.name) || this.data.extraForm.name.trim(),
        extraSaving: false,
        // 保存成功即收起；不提示是否命中员工名单（U46）
        extraOpen: false,
      });
      this.refreshName();
      this.selectComponent('#toast').show('已保存');
      return true;
    } catch (error) {
      this.setData({ extraSaving: false });
      this.selectComponent('#toast').show('保存失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  toOrders() { nav.tabTo('orders'); },
  reset() { nav.reset(); },
});
