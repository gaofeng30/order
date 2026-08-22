const { nav } = require('../../utils/util.js');

/* 身份选择页。**不是入口页** —— §4.4：未绑定商户手机号的用户直接进用户端
   首页，只有已绑定的才落到这里。绑定判定由服务端在启动时静默 wx.login 后
   给出（§16.5 待补齐），在此之前入口一律走「未绑定」分支。

   本页不做任何身份索取：§14 要求启动不弹手机号授权，商户的首次绑定入口在
   个人中心的「商户登录」（§4.4），不在这里。 */
Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: require('../../utils/data.js').STORE,
  },
  back() { nav.back(); },
  go(e) { nav.go(e.currentTarget.dataset.to); },
});
