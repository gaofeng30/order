const identityStore = require('../../utils/identityStore.js');
const pcLoginStore = require('../../utils/pcLoginStore.js');
const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: { ownerState: 'loading', stage: 'idle', busy: false, message: '' },
  async onLoad() {
    this._challenge = null;
    this._busy = false;
    this._disposed = false;
    await this.loadOwner();
  },
  onUnload() {
    this._disposed = true;
    this._challenge = null;
  },
  async loadOwner() {
    if (this._busy) return false;
    this._busy = true;
    this.setData({ ownerState: 'loading', busy: true, message: '' });
    try {
      const identity = await identityStore.load();
      if (this._disposed) return false;
      const owner = identity.merchant.bound && identity.merchant.role === 'OWNER';
      this.setData({ ownerState: owner ? 'ready' : 'forbidden', message: owner ? '' : '仅门店主账号可以登录电脑后台。' });
      return owner;
    } catch (error) {
      if (!this._disposed) this.setData({ ownerState: 'error', message: '身份加载失败，请重试。' });
      return false;
    } finally {
      this._busy = false;
      if (!this._disposed) this.setData({ busy: false });
    }
  },
  scan() {
    if (this._busy || this.data.ownerState !== 'ready') return false;
    this._challenge = null;
    if (typeof wx.scanCode !== 'function') {
      this.setData({ stage: 'idle', message: '请在手机微信中打开小程序扫码。' });
      return false;
    }
    this._busy = true;
    this.setData({ busy: true, stage: 'idle', message: '' });
    wx.scanCode({
      onlyFromCamera: true,
      scanType: ['qrCode'],
      success: result => {
        if (this._disposed) return;
        try {
          // 扫码密钥只留在页面实例中，避免写入页面数据、日志和本地存储。
          this._challenge = pcLoginStore.parseQR(result && result.result);
          this.setData({ stage: 'confirm', message: '已读取电脑登录码，请确认是你正在使用的电脑。' });
        } catch (error) {
          this.setData({ stage: 'idle', message: '这不是有效的电脑后台登录码，请扫描管理台显示的二维码。' });
        }
      },
      fail: result => {
        if (this._disposed) return;
        const cancelled = result && typeof result.errMsg === 'string' && /cancel/.test(result.errMsg);
        this.setData({ stage: 'idle', message: cancelled ? '已取消扫码，可以重新扫描。' : '扫码未完成，请重试。' });
      },
      complete: () => {
        this._busy = false;
        if (!this._disposed) this.setData({ busy: false });
      },
    });
    return true;
  },
  async onGetPhoneNumber(event) {
    if (this._busy || this.data.ownerState !== 'ready' || this.data.stage !== 'confirm' || !this._challenge) return false;
    const code = event && event.detail && event.detail.code;
    if (!code) {
      this.setData({ message: '尚未确认手机号，可以再次点击确认登录。' });
      return false;
    }
    this._busy = true;
    this.setData({ busy: true, message: '正在确认电脑登录…' });
    try {
      await pcLoginStore.approve(this._challenge, code);
      this._challenge = null;
      if (!this._disposed) this.setData({ stage: 'approved', message: '已确认登录，请回到电脑继续操作。' });
      return true;
    } catch (error) {
      if (this._disposed) return false;
      if (error && error.code === 'PC_LOGIN_EXPIRED') {
        this._challenge = null;
        this.setData({ stage: 'idle', message: '登录码已过期或已使用。请在电脑刷新登录码，再重新扫描。' });
      } else if (error && error.code === 'FORBIDDEN') {
        this._challenge = null;
        this.setData({ ownerState: 'forbidden', stage: 'idle', message: '仅门店主账号可以登录电脑后台。' });
      } else {
        this.setData({ message: '登录确认未完成，请检查电脑是否已登录；未登录时可以再次确认或重新扫码。' });
      }
      return false;
    } finally {
      this._busy = false;
      if (!this._disposed) this.setData({ busy: false });
    }
  },
  back() { nav.back(); },
});
