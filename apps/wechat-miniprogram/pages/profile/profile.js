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
    phone: '', phoneBinding: false, extraPhone: '', merchantBound: false, merchantLoginState: 'idle',
    extraForm: { phone: '' }, extraSaving: false,
    defaultName: '', nameInput: '', nameSaving: false, nameSavable: false,
    // 附加手机号默认收起
    extraOpen: false, extraSavable: false,
    servicePhone: SERVICE_PHONE_PLACEHOLDER,
  },
  onLoad() { this.loadServicePhone(); },
  reloadIdentity() { return this.onShow(); },
  async onShow() {
    this.setData({ identityState: 'loading', pend: 0 });
    const identityTask = identityStore.load().then(identity => {
      const name = textOf(identity.default_contact_name);
      const extra = phoneStore.displayPhone(identity.extra_phone.phone);
      this.setData({
        identityState: 'ready', phone: phoneStore.displayPhone(identity.primary_phone.phone),
        extraPhone: extra, 'extraForm.phone': extra, defaultName: name, nameInput: name, nameSavable: false,
        merchantBound: identity.merchant.bound,
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
    const name = this.data.defaultName;
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
      this.setData({ phone: status.phone, phoneBinding: false });
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
  onNameInput(e) {
    const nameInput = e.detail.value;
    const name = nameInput.trim();
    this.setData({ nameInput, nameSavable: !!name && name !== this.data.defaultName && this.nameBytes(name) <= 64 });
  },
  nameBytes(name) {
    return Array.from(name).reduce((sum, char) => {
      const n = char.codePointAt(0); return sum + (n < 0x80 ? 1 : n < 0x800 ? 2 : n < 0x10000 ? 3 : 4);
    }, 0);
  },
  async saveName() {
    if (this.data.nameSaving || this.data.identityState !== 'ready') return false;
    const name = this.data.nameInput.trim();
    if (!this.data.nameSavable) {
      if (!name || this.nameBytes(name) > 64) this.selectComponent('#toast').show(!name ? '请填写默认取餐姓名' : '姓名过长，请缩短后保存', { icon: 'warn' });
      return false;
    }
    this.setData({ nameSaving: true });
    try {
      const result = await api.post('/api/v1/me/default-contact-name', { name }, api.newIdempotencyKey('default-name'));
      if (!result || typeof result.default_contact_name !== 'string' || !result.default_contact_name) throw new Error('name unavailable');
      this.setData({ defaultName: result.default_contact_name, nameInput: result.default_contact_name, nameSaving: false, nameSavable: false });
      this.refreshName();
      this.selectComponent('#toast').show('默认取餐姓名已保存');
      return true;
    } catch (error) {
      this.setData({ nameSaving: false });
      this.selectComponent('#toast').show('保存失败，请重试', { icon: 'warn' });
      return false;
    }
  },
  toggleExtra() { this.setData({ extraOpen: !this.data.extraOpen }); },
  // 附加手机号独立保存；员工匹配使用已保存的默认取餐姓名。
  refreshExtraSavable() {
    const { phone } = this.data.extraForm;
    const savable = EXTRA_PHONE.test(phone.trim()) && phone.trim() !== this.data.extraPhone;
    if (savable !== this.data.extraSavable) this.setData({ extraSavable: savable });
  },
  onExtraInput(e) {
    this.setData({ [`extraForm.${e.currentTarget.dataset.k}`]: e.detail.value });
    this.refreshExtraSavable();
  },
  async saveExtraPhone() {
    if (this.data.extraSaving || this.data.identityState !== 'ready') return false;
    if (!this.data.phone) {
      this.selectComponent('#toast').show('请先绑定主手机号', { icon: 'warn' });
      return false;
    }
    if (!this.data.extraSavable) {
      const { phone } = this.data.extraForm;
      const reason = !EXTRA_PHONE.test(phone.trim()) ? '请输入 11 位手机号' : '';
      if (reason) this.selectComponent('#toast').show(reason, { icon: 'warn' });
      return false;
    }
    this.setData({ extraSaving: true });
    try {
      const result = await phoneStore.setExtra(this.data.extraForm.phone.trim(), api.newIdempotencyKey('extra-phone'));
      const extraPhone = phoneStore.displayPhone(result.extraPhone.phone);
      this.setData({ extraPhone, 'extraForm.phone': extraPhone, extraSaving: false, extraOpen: false });
      this.refreshExtraSavable();
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
