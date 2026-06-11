/* 绥安食品 — 通用工具: 路由 / 状态语义 / 购物车 / 订单推进 */
const data = require('./data.js');

// ---- 状态 → 胶囊语义 (颜色统一来源) ----
const STATUS_MAP = {
  待取餐: 'info', 制作中: 'info', 进行中: 'info', 配送中: 'info',
  已完成: 'ok', 成功: 'ok', 已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
  待支付: 'warn', 待接单: 'warn', 待取超时: 'warn', 库存告急: 'warn',
  已取消: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute', 未开放: 'mute',
};
const statusTone = s => STATUS_MAP[s] || 'mute';

// ---- 路由 (映射到原生导航) ----
function buildUrl(id, params) {
  let url = `/pages/${id}/${id}`;
  if (params && Object.keys(params).length) {
    const qs = Object.keys(params)
      .filter(k => params[k] !== undefined && params[k] !== null)
      .map(k => `${k}=${encodeURIComponent(params[k])}`)
      .join('&');
    if (qs) url += `?${qs}`;
  }
  return url;
}
const nav = {
  go: (id, params) => wx.navigateTo({ url: buildUrl(id, params) }),       // 入栈
  replace: (id, params) => wx.redirectTo({ url: buildUrl(id, params) }),  // 替换栈顶
  tabTo: (id, params) => wx.reLaunch({ url: buildUrl(id, params) }),      // 重置为单页 (Tab 切换)
  reset: () => wx.reLaunch({ url: '/pages/launch/launch' }),             // 回启动选择
  back: () => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/home/home' }) }),
};

// ---- 购物车 (操作 globalData.cart) ----
function getApp_() { return getApp(); }
const cart = {
  get: () => getApp_().globalData.cart,
  add(id) { const c = getApp_().globalData.cart; c[id] = (c[id] || 0) + 1; },
  sub(id) { const c = getApp_().globalData.cart; if ((c[id] || 0) <= 1) delete c[id]; else c[id]--; },
  clear() { getApp_().globalData.cart = {}; },
  count() { return Object.values(getApp_().globalData.cart).reduce((a, b) => a + b, 0); },
  total() {
    const c = getApp_().globalData.cart;
    return Object.keys(c).reduce((a, id) => a + (data.itemById(id).price * c[id]), 0);
  },
  list() {
    const c = getApp_().globalData.cart;
    return Object.keys(c).map(id => ({ item: data.itemById(id), q: c[id] }));
  },
};

// ---- 订单状态机 (商户端推进) ----
const NEXT = { 待接单: '制作中', 制作中: '待取餐', 待取餐: '已完成' };
const ACT = { 待接单: '接单', 制作中: '备好', 待取餐: '核销', 已完成: '查看', 已取消: '查看' };

// 菜品摘要串
function itemsSummary(items) {
  return items.map(([id, q]) => data.itemById(id).name + '×' + q).join('，');
}

// 商户端单向推进订单状态 + Toast 撤销
function advanceOrder(id, toastComp, refresh) {
  const g = getApp().globalData;
  const o = g.aOrders.find(x => x.id === id);
  if (!o) return;
  const nx = NEXT[o.status];
  if (!nx) return;
  const prev = o.status;
  const act = ACT[prev];
  o.status = nx;
  if (refresh) refresh();
  if (toastComp) {
    toastComp.show(`已${act}「${o.code}」`, {
      icon: 'check',
      onUndo: () => { o.status = prev; if (refresh) refresh(); },
    });
  }
}

// 推进按钮的展示元信息
function advanceMeta(status) {
  const isView = status === '已完成' || status === '已取消';
  return {
    label: ACT[status],
    isView,
    cls: isView ? 'btn--ghost-blue' : (status === '待取餐' ? 'btn--blue' : 'btn--primary'),
    scan: status === '待取餐',
  };
}

module.exports = {
  STATUS_MAP, statusTone, nav, buildUrl, cart, NEXT, ACT, itemsSummary, advanceOrder, advanceMeta,
};
