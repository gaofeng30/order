const { nav } = require('../../utils/util.js');
const api = require('../../utils/api.js');
const data = require('../../utils/data.js');
const promo = require('../../utils/promo.js');

const maskPhone = p => (p && p.length === 11 ? p.slice(0, 3) + '****' + p.slice(7) : p);

Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    pend: 0,
    nick: data.ME.nick,
    phoneMask: maskPhone(data.ME.phone),
    isMember: false,
    levelName: '',
    levelLabel: '',
    couponCount: 0,
  },
  onShow() {
    this.setData({ pend: getApp().globalData.orders.filter(o => o.status === '待取餐' || o.status === '已预约').length });
    this.loadMember();
  },
  // 会员身份由服务端按微信授权手机号命中名单后下发
  loadMember() {
    Promise.all([api.getMyMembership(), api.listMyCoupons(), api.myCouponUsed()]).then(([me, cps, used]) => {
      const today = data.TODAY;
      const usable = cps.filter(c => today >= c.start && today <= c.end && (used[c.id] || 0) < c.perLimit);
      this.setData({
        isMember: me.isMember,
        levelName: me.level ? me.level.name : '',
        levelLabel: me.level ? promo.discountLabel(me.level.discount) : '',
        couponCount: usable.length,
      });
    });
  },
  toOrders() { nav.tabTo('orders'); },
  toCoupons() { nav.go('my-coupons'); },
  toast(msg, icon) { this.selectComponent('#toast').show(msg, { icon: icon || 'check' }); },
  service() { this.toast('正在拨打 0596-388 1688', 'phone'); },
  settings() { this.toast('设置建设中', 'settings'); },
  reset() { nav.reset(); },
});
