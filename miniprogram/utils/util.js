/* 绥安食品 — 通用工具: 路由 / 状态语义 / 购物车 / 订单推进 */
const data = require('./data.js');

// ---- 状态 → 胶囊语义 (颜色统一来源) ----
const STATUS_MAP = {
  待取餐: 'info', 待制作: 'info', 制作中: 'info', 进行中: 'info', 配送中: 'info', 已预约: 'info',
  已完成: 'ok', 成功: 'ok', 已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
  待支付: 'warn', 待取超时: 'warn', 库存告急: 'warn',
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
  /* Tab 切换：栈感知路由，避免 reLaunch 整栈销毁重建导致的切换卡顿。
     - 目标页已在栈中 → navigateBack 弹回（最快，且顺带清掉上层流程页）
     - 不在栈中 → redirectTo 只替换栈顶（栈深不增长，不会溢出）
     - 带参数或栈过深（≥8，接近微信 10 层上限）→ 退回 reLaunch 兜底 */
  tabTo(id, params) {
    const url = buildUrl(id, params);
    const pages = getCurrentPages();
    if ((params && Object.keys(params).length) || pages.length >= 8) {
      return wx.reLaunch({ url });
    }
    const route = `pages/${id}/${id}`;
    const idx = pages.findIndex(p => p.route === route);
    if (idx === pages.length - 1) return;  // 已在目标页
    if (idx > -1) return wx.navigateBack({ delta: pages.length - 1 - idx, fail: () => wx.reLaunch({ url }) });
    return wx.redirectTo({ url, fail: () => wx.reLaunch({ url }) });
  },
  reset: () => wx.reLaunch({ url: '/pages/launch/launch' }),             // 回启动选择
  toBrand: () => wx.reLaunch({ url: '/pages/brand/brand' }),            // 回品牌选择
  back: () => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/home/home' }) }),
};

// ---- 下单模式 (尽快 now / 预约 reserve, 跨页共享) ----
const orderMode = {
  get: () => getApp().globalData.orderMode || 'now',
  set: (m) => { getApp().globalData.orderMode = m; },
};

// ---- 购物车 (操作 globalData.cart) ----
// cart: { [id]: { qty, flavors:[], note:'' } }  口味/备注绑定到菜品
function cartRaw() { return getApp().globalData.cart; }
const cart = {
  get: () => cartRaw(),
  qty(id) { const e = cartRaw()[id]; return e ? e.qty : 0; },
  entry(id) { return cartRaw()[id] || null; },
  add(id) {
    const c = cartRaw();
    const e = c[id] || { qty: 0, flavors: [], note: '' };
    c[id] = Object.assign({}, e, { qty: e.qty + 1 });
  },
  sub(id) {
    const c = cartRaw();
    const e = c[id];
    if (!e) return;
    if (e.qty <= 1) delete c[id];
    else c[id] = Object.assign({}, e, { qty: e.qty - 1 });
  },
  // 加购并写入口味/备注 (来自口味弹层)
  setPrefs(id, prefs) {
    const c = cartRaw();
    const e = c[id] || { qty: 1, flavors: [], note: '' };
    c[id] = Object.assign({}, e, prefs);
  },
  remove(id) { delete cartRaw()[id]; },
  clear() { getApp().globalData.cart = {}; },
  count() { return Object.values(cartRaw()).reduce((a, e) => a + e.qty, 0); },
  total() {
    const c = cartRaw();
    return Object.keys(c).reduce((a, id) => a + (data.itemById(id).price * c[id].qty), 0);
  },
  list() {
    const c = cartRaw();
    return Object.keys(c).map(id => ({
      item: data.itemById(id), q: c[id].qty, flavors: c[id].flavors || [], note: c[id].note || '',
    }));
  },
};

// ---- 订单状态机 (商户端履约推进, 单向) ----
// 待制作 ──备好──▶ 待取餐 ──核销──▶ 已完成
const NEXT = { 待制作: '待取餐', 待取餐: '已完成' };
const ACT = { 待制作: '备好', 待取餐: '核销', 已完成: '查看', 已取消: '查看' };

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
  STATUS_MAP, statusTone, nav, buildUrl, orderMode, cart, NEXT, ACT, itemsSummary, advanceOrder, advanceMeta,
};
