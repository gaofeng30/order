const identityStore = require('../../utils/identityStore.js');
const { nav } = require('../../utils/util.js');

const ROLE_LABELS = { OWNER: '主账号', SUBACCOUNT: '子账号' };

// 商户端「我的」：身份卡、登录电脑后台（仅主账号）、切换身份（A2）
Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { loadState: 'loading', roleLabel: '', isOwner: false, phoneMask: '' },
  onLoad() { this.load(); },
  onUnload() { this._generation = (this._generation || 0) + 1; },
  async load() {
    const generation = (this._generation || 0) + 1;
    this._generation = generation;
    this.setData({ loadState: 'loading' });
    try {
      const identity = await identityStore.load();
      if (generation !== this._generation) return false;
      const role = identity.merchant.bound ? identity.merchant.role : '';
      this.setData({
        loadState: 'ready',
        roleLabel: ROLE_LABELS[role] || '未绑定商户账号',
        isOwner: role === 'OWNER',
        phoneMask: identity.primary_phone.masked_phone || '',
      });
      return true;
    } catch (error) {
      if (generation === this._generation) this.setData({ loadState: 'error' });
      return false;
    }
  },
  toPCLogin() {
    if (!this.data.isOwner) return;
    nav.go('admin-login');
  },
  switchIdentity() { nav.reset(); },
});
