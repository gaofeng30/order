const { nav } = require('../../utils/util.js');

/* 身份选择页 —— 入口页（项目方 2026-08-22 决策，已回写 §4.4）。

   用户端一侧零索取：§14 要求用户能免手机号浏览、启动时不弹手机号授权。
   商户端一侧触发微信真实的手机号授权面板（§4.4 的绑定链路）。

   前端拿到的只是加密数据，明文手机号与商户账号名单的比对必须由服务端完成，
   所以这里不做也不声称做任何校验。§4.4 末条：客户端隐藏入口不能代替鉴权，
   商户端四屏的访问控制在服务端，本页不给它任何前端替身。 */
Page({
  behaviors: [require('../../utils/navBehavior.js')],
  data: {
    store: require('../../utils/data.js').STORE,
    hint: '',
  },
  back() { nav.back(); },
  go(e) { nav.go(e.currentTarget.dataset.to); },

  /* 微信手机号授权回调。允许时 detail 带 code / encryptedData，拒绝时只有 errMsg。 */
  onMerchantPhone(e) {
    const d = (e && e.detail) || {};
    if (!d.code && !d.encryptedData) {
      // 拒绝是合法选择，不渲染成失败，也不拦路
      this.setData({ hint: '商户端需要验证手机号身份。未授权时仍可从上方进入用户端浏览。' });
      return;
    }
    this.setData({ hint: '' });
    /* 拿到的是加密数据，比对商户账号名单需服务端换取明文（§4.4）。
       此处如实告知校验尚未发生 —— 省掉这句，演示现场看到的就是一个
       「验证通过」的假象，而实际上什么都没验。 */
    wx.showToast({ title: '已授权 · 身份校验待服务端接入', icon: 'none', duration: 2600 });
    nav.go('admin-orders');
  },
});
