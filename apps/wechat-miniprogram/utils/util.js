/* 绥安食品 — 通用工具: 路由 / 状态语义 / 购物车 / 订单推进 */
const data = require('./data.js');
const catalogStore = require('./catalogStore.js');

// ---- 状态 → 胶囊语义 (颜色统一来源) ----
const STATUS_MAP = {
  已预约: 'info', 制作中: 'info', 待取餐: 'info',
  已完成: 'ok', 成功: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
  退款中: 'warn',
  已退款: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute',
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
  back: () => wx.navigateBack({ fail: () => wx.reLaunch({ url: '/pages/home/home' }) }),
};

// ---- 取餐时间 (跨页共享；菜单顶部条选择，结算页只读) ----
const pickup = {
  get() {
    const g = getApp().globalData;
    if (!g.pickup) g.pickup = data.defaultPickup();
    return g.pickup;
  },
  set(pk) { getApp().globalData.pickup = pk; },
  label() { return data.pickupLabel(this.get()); },
};

// ---- 购物车 (操作 globalData.cart) ----
// cart: { [id]: { product, qty, flavors:[], note:'' } }，product 是首次选择时的目录快照
function cartRaw() { return getApp().globalData.cart; }
function invalidCart() {
  const error = new Error('cart unavailable');
  error.code = 'CART_INVALID';
  return error;
}
function validQty(qty) {
  if (!Number.isSafeInteger(qty) || qty <= 0) throw invalidCart();
  return qty;
}
function validLineCents(product, qty) {
  const line = product.price_cents * validQty(qty);
  if (!Number.isSafeInteger(line)) throw invalidCart();
  return line;
}
function prefsOf(prefs, fallbackQty) {
  const qty = prefs && Object.hasOwn(prefs, 'qty') ? validQty(prefs.qty) : validQty(fallbackQty);
  return {
    qty,
    flavors: Array.isArray(prefs && prefs.flavors) ? prefs.flavors.slice() : [],
    note: typeof (prefs && prefs.note) === 'string' ? prefs.note : '',
  };
}
const cart = {
  get: () => cartRaw(),
  qty(id) { const e = cartRaw()[id]; return e ? validQty(e.qty) : 0; },
  entry(id) { const e = cartRaw()[id]; if (e) validQty(e.qty); return e || null; },
  add(product) {
    const snapshot = catalogStore.snapshotProduct(product);
    const c = cartRaw();
    const e = c[snapshot.id];
    if (e) {
      const nextQty = validQty(e.qty) + 1;
      validLineCents(e.product, nextQty);
      c[snapshot.id] = Object.assign({}, e, { qty: nextQty });
      return;
    }
    validLineCents(snapshot, 1);
    c[snapshot.id] = { product: snapshot, qty: 1, flavors: [], note: '' };
  },
  sub(id) {
    const c = cartRaw();
    const e = c[id];
    if (!e) return;
    const qty = validQty(e.qty);
    if (qty === 1) delete c[id];
    else c[id] = Object.assign({}, e, { qty: qty - 1 });
  },
  // 加购并写入口味/备注 (来自口味弹层)
  setPrefs(product, prefs) {
    const snapshot = catalogStore.snapshotProduct(product);
    const c = cartRaw();
    const existing = c[snapshot.id];
    const next = Object.assign(
      { product: existing ? existing.product : snapshot },
      prefsOf(prefs, existing ? existing.qty : 1),
    );
    validLineCents(next.product, next.qty);
    c[snapshot.id] = next;
  },
  remove(id) { delete cartRaw()[id]; },
  clear() { getApp().globalData.cart = {}; },
  count() {
    return Object.values(cartRaw()).reduce((sum, entry) => {
      const next = sum + validQty(entry.qty);
      if (!Number.isSafeInteger(next)) throw invalidCart();
      return next;
    }, 0);
  },
  totalCents() {
    return Object.values(cartRaw()).reduce((sum, entry) => {
      const line = validLineCents(entry.product, entry.qty);
      if (!Number.isSafeInteger(sum + line)) throw invalidCart();
      return sum + line;
    }, 0);
  },
  total() { return Number(catalogStore.formatCents(this.totalCents())); },
  list() {
    return Object.values(cartRaw()).map(entry => {
      const item = catalogStore.withPrice(entry.product);
      item.price = Number(item.price_text);
      const qty = validQty(entry.qty);
      const lineTotalCents = validLineCents(item, qty);
      return {
        item,
        q: qty,
        flavors: entry.flavors.slice(),
        note: entry.note,
        line_total_cents: lineTotalCents,
        line_total_text: catalogStore.formatCents(lineTotalCents),
      };
    });
  },
};

/* ---- 六态订单状态机 (生效 spec: Orders use one six-state production state machine) ----
   已预约 ──取餐前 30 分钟，服务端定时推进──▶ 制作中 ──备好──▶ 待取餐 ──核销──▶ 已完成
   旁路：  任一已支付态 ──退款──▶ 退款中 ──微信确认──▶ 已退款

   NEXT 只包含商户可执行的转换。`已预约 → 制作中` 由服务端定时任务驱动，
   前端不得提供该转换；退款由 PC 后台发起，不在商户端推进链路上。
   生产禁止撤销或回退已完成的转换，因此不提供任何回退入口。 */
const NEXT = { 制作中: '待取餐', 待取餐: '已完成' };
const ACT = { 已预约: '待开做', 制作中: '备好', 待取餐: '核销', 已完成: '查看', 退款中: '查看', 已退款: '查看' };

/* ---- 取餐号解析与订单检索（PRD §7.8、§6.6）----
   §7.8：取餐号按取餐日期从 0001 累计，跨营业日可能重复，因此按号定位
   必须限定当前营业日。这条规则只实现一次 —— 搜索与手工核销共用 findByCode，
   分开实现两处必然漂移。 */
const CODE_RE = /^\d{4}$/;

// 按取餐号定位当前营业日的订单；不在本营业日的号一律不解析。
function findByCode(code) {
  const key = String(code == null ? '' : code).trim();
  if (!CODE_RE.test(key)) return null;
  const list = getApp().globalData.aOrders;
  return list.find(o => o.code === key && o.pickupDate === data.BUSINESS_DAY) || null;
}

/* 该号在当前营业日无果、却存在于别的营业日时，报出这个事实和替代定位方式。
   空列表是最差的反馈：它和「这个号不存在」不可区分，而两者处置完全不同。 */
function codeHint(code) {
  const key = String(code == null ? '' : code).trim();
  if (!CODE_RE.test(key) || findByCode(key)) return '';
  const others = getApp().globalData.aOrders.filter(o => o.code === key);
  if (!others.length) return '';
  const days = [...new Set(others.map(o => o.pickupDate))].sort().join('、');
  return `取餐号「${key}」属于 ${days} 的订单，不在当前营业日 ${data.BUSINESS_DAY}。请改用订单号或手机号定位该单。`;
}

/* 商户端订单检索：取餐号 / 订单号 / 手机号 / 联系人，跨全部状态泳道。
   4 位纯数字同时按手机号片段匹配 —— 跨营业日歧义只是取餐号的属性，
   手机尾号没有这个问题；只按取餐号解释会让输入手机尾号的商户得到空列表。 */
function searchOrders(keyword) {
  const key = String(keyword == null ? '' : keyword).trim();
  if (!key) return [];
  const list = getApp().globalData.aOrders;
  if (CODE_RE.test(key)) {
    const byPhone = list.filter(o => String(o.phone).includes(key));
    const hit = findByCode(key);
    if (hit && !byPhone.some(o => o.id === hit.id)) return [hit].concat(byPhone);
    return byPhone;
  }
  const up = key.toUpperCase();
  return list.filter(o =>
    String(o.no).toUpperCase().includes(up) ||
    String(o.phone).includes(key) ||
    String(o.contact).includes(key) ||
    String(o.code).includes(key));
}

// 菜品摘要串
function itemsSummary(items) {
  return items.map(([id, q]) => data.itemById(id).name + '×' + q).join('，');
}

// 商户端单向推进订单状态。不可回退：生效 spec 禁止生产撤销。
function advanceOrder(id, toastComp, refresh) {
  const g = getApp().globalData;
  const o = g.aOrders.find(x => x.id === id);
  if (!o) return;
  const nx = NEXT[o.status];
  if (!nx) return;
  const act = ACT[o.status];
  o.status = nx;
  if (refresh) refresh();
  if (toastComp) toastComp.show(`已${act}「${o.code}」`, { icon: 'check' });
}

// 推进按钮的展示元信息
function advanceMeta(status) {
  const isView = !NEXT[status];
  return {
    label: ACT[status],
    isView,
    cls: isView ? 'btn--ghost-blue' : (status === '待取餐' ? 'btn--blue' : 'btn--primary'),
    scan: status === '待取餐',
  };
}

module.exports = {
  STATUS_MAP, statusTone, nav, buildUrl, pickup, cart, NEXT, ACT, itemsSummary, advanceOrder, advanceMeta,
  findByCode, codeHint, searchOrders,
};
