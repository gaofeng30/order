const data = require('../../utils/data.js');

const BIZ = ['营业中', '休息中', '已截单'];

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    biz: BIZ,
    storeStatus: '营业中',
    store: data.STORE,
    notice: '今日卤味新鲜出锅，欢迎到店自提～',
  },
  onShow() { this.setData({ storeStatus: getApp().globalData.store.status }); },
  setBiz(e) {
    const b = e.currentTarget.dataset.b;
    getApp().globalData.store.status = b;
    this.setData({ storeStatus: b });
    this.selectComponent('#toast').show('已切换为「' + b + '」', { icon: 'power' });
  },
  onNotice(e) { this.setData({ notice: e.detail.value }); },
  pick(e) {
    const map = { time: '时间选择器 · 建设中', cutoff: '截单时间 · 建设中', pickup: '取餐点设置 · 建设中' };
    const icon = { time: 'clock', cutoff: 'clock', pickup: 'pin' };
    const k = e.currentTarget.dataset.k;
    this.selectComponent('#toast').show(map[k], { icon: icon[k] });
  },
  save() {
    this.selectComponent('#toast').show('设置已保存', { icon: 'check' });
    setTimeout(() => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/admin-profile/admin-profile' }) }), 600);
  },
});
