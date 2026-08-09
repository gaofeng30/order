/* 页面通用行为：注入屏幕适配尺寸 (statusBarH / navH / navTotal / safeBottom)
   每个页面 behaviors:[require('../../utils/navBehavior.js')] 即可使用，
   WXML 中用 {{navTotal}}px / {{safeBottom}}px 做顶部、底部安全区适配。

   说明：behavior 应用于 Page 时，其页面生命周期函数 (onLoad) 会与页面自身的
   同名函数一并被调用（behavior 先执行），因此可在此统一注入适配尺寸。 */
function applyNav(ctx) {
  const g = getApp().globalData;
  ctx.setData({
    statusBarH: g.statusBarHeight,
    navH: g.navBarHeight,
    navTotal: g.navTotalHeight,
    safeBottom: g.safeBottom,
  });
}

module.exports = Behavior({
  data: {
    statusBarH: 20,
    navH: 44,
    navTotal: 64,
    safeBottom: 0,
  },
  methods: {
    _initNav() { applyNav(this); },
  },
  // 组件场景
  lifetimes: { attached() { applyNav(this); } },
  // 页面场景
  onLoad() { applyNav(this); },
});
