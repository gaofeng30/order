const { nav } = require('../../utils/util.js');

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    opts: [
      { key: 'food', name: '绥安食品', icon: 'bowl', desc: '到店点单 · 预约取餐', ready: true },
      { key: 'wash', name: '绥安洗衣', icon: 'washer', desc: '即将上线', ready: false },
      { key: 'car', name: '绥安洗车', icon: 'car', desc: '即将上线', ready: false },
    ],
  },
  tap(e) {
    const { name, ready } = e.currentTarget.dataset;
    if (ready) {
      nav.go('launch');
    } else {
      this.selectComponent('#toast').show(name + ' 暂未上线，敬请期待', { icon: 'clock' });
    }
  },
});
