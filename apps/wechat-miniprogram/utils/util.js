/* 绥安食品 — 通用工具: 路由 / 状态语义 / 客户端购物车 */
const catalogStore = require('./catalogStore.js');
const dateLabel = require('./dateLabel.js');

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

const pickup = {
  get() { return getApp().globalData.pickup || null; },
  set(pk) { getApp().globalData.pickup = pk; },
  // 取餐时间条：「今天 12:00」（G8、PRD §5.5）
  label() {
    const selected = this.get();
    if (!selected) return '';
    return `${dateLabel.dayLabel(selected.date)} ${selected.time}`;
  },
  // 详情页取餐时段：「今天 午餐」
  periodLabel() {
    const selected = this.get();
    if (!selected) return '';
    return `${dateLabel.dayLabel(selected.date)} ${selected.mealPeriod === 'lunch' ? '午餐' : '晚餐'}`;
  },
};

// ---- 购物车 (操作 globalData.cart) ----
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
  const qty = prefs && Object.prototype.hasOwnProperty.call(prefs, 'qty') ? validQty(prefs.qty) : validQty(fallbackQty);
  return {
    qty,
    flavors: Array.isArray(prefs && prefs.flavors) ? prefs.flavors.slice() : [],
    note: typeof (prefs && prefs.note) === 'string' ? prefs.note : '',
  };
}
function groupKey(selection) {
  return selection ? `${selection.date}:${selection.mealPeriod}` : '';
}
function lineKey(id, selection) {
  return `${groupKey(selection)}:${id}`;
}
function entries(selection) {
  return Object.values(cartRaw()).filter(entry => selection === null || groupKey(entry.selection) === groupKey(selection));
}
function paymentMap() {
  const data = getApp().globalData;
  if (!data.cartPayments) data.cartPayments = {};
  return data.cartPayments;
}
const cart = {
  groupKey,
  get: () => cartRaw(),
  payment(selection) { return paymentMap()[groupKey(selection)] || null; },
  savePayment(selection, value) {
    if (value) paymentMap()[groupKey(selection)] = value;
    else delete paymentMap()[groupKey(selection)];
  },
  isLocked(selection) { return !!this.payment(selection); },
  qty(id, selection = pickup.get()) { const entry = this.entry(id, selection); return entry ? validQty(entry.qty) : 0; },
  entry(id, selection = pickup.get()) {
    const entry = cartRaw()[lineKey(id, selection)];
    if (entry) validQty(entry.qty);
    return entry || null;
  },
  add(product, selection = pickup.get()) {
    if (!selection || this.isLocked(selection)) return false;
    const entry = this.entry(product.id, selection);
    return this.setPrefs(product, entry
      ? { qty: validQty(entry.qty) + 1, flavors: entry.flavors, note: entry.note }
      : { qty: 1 }, selection);
  },
  sub(id, selection = pickup.get()) {
    if (this.isLocked(selection)) return false;
    const entry = this.entry(id, selection);
    if (!entry) return false;
    if (validQty(entry.qty) === 1) this.remove(id, selection);
    else entry.qty -= 1;
    return true;
  },
  setPrefs(product, prefs, selection = pickup.get()) {
    if (!selection || this.isLocked(selection)) return false;
    const snapshot = catalogStore.snapshotProduct(product);
    const existing = this.entry(snapshot.id, selection);
    const group = entries(selection)[0];
    const next = Object.assign({
      product: snapshot,
      selection: Object.assign({}, group ? group.selection : selection),
    }, prefsOf(prefs, existing ? existing.qty : 1));
    validLineCents(next.product, next.qty);
    cartRaw()[lineKey(snapshot.id, selection)] = next;
    return true;
  },
  remove(id, selection = pickup.get()) {
    if (this.isLocked(selection)) return false;
    delete cartRaw()[lineKey(id, selection)];
    return true;
  },
  clear() {
    Object.keys(cartRaw()).forEach(key => {
      if (!this.isLocked(cartRaw()[key].selection)) delete cartRaw()[key];
    });
  },
  complete(selection) {
    Object.keys(cartRaw()).forEach(key => {
      if (groupKey(cartRaw()[key].selection) === groupKey(selection)) delete cartRaw()[key];
    });
    this.savePayment(selection, null);
  },
  setSelection(selection) {
    if (this.isLocked(selection)) return false;
    entries(selection).forEach(entry => { entry.selection = Object.assign({}, selection); });
    return true;
  },
  reprice(productsById, selection = pickup.get()) {
    if (this.isLocked(selection)) return;
    entries(selection).forEach(entry => {
      const latest = productsById && productsById[entry.product.id];
      if (!latest) return;
      const snapshot = catalogStore.snapshotProduct(latest);
      validLineCents(snapshot, entry.qty);
      entry.product = snapshot;
    });
  },
  count(availability, selection = pickup.get()) {
    return this.list(availability, selection).reduce((sum, line) => {
      const next = sum + (line.unavailableLabel ? 0 : line.q);
      if (!Number.isSafeInteger(next)) throw invalidCart();
      return next;
    }, 0);
  },
  totalCents(availability, selection = pickup.get()) {
    return this.list(availability, selection).reduce((sum, line) => {
      const next = sum + (line.unavailableLabel ? 0 : line.line_total_cents);
      if (!Number.isSafeInteger(next)) throw invalidCart();
      return next;
    }, 0);
  },
  total(availability, selection = pickup.get()) { return catalogStore.formatCents(this.totalCents(availability, selection)); },
  unavailableCount(availability, selection = pickup.get()) {
    return this.list(availability, selection).filter(line => line.unavailableLabel).length;
  },
  list(availability, selection = pickup.get()) {
    return entries(selection).map(entry => {
      const item = catalogStore.withPrice(entry.product);
      const qty = validQty(entry.qty);
      const lineTotalCents = validLineCents(item, qty);
      return {
        id: item.id, key: lineKey(item.id, entry.selection),
        selection: Object.assign({}, entry.selection),
        periodLabel: `${dateLabel.dayLabel(entry.selection.date)} ${entry.selection.mealPeriod === 'lunch' ? '午餐' : '晚餐'}`,
        item, q: qty, flavors: entry.flavors.slice(), note: entry.note,
        line_total_cents: lineTotalCents, line_total_text: catalogStore.formatCents(lineTotalCents),
        unavailableLabel: unavailableLabel(entry, availability),
      };
    });
  },
  groups() {
    const groups = {};
    this.list(undefined, null).forEach(line => {
      const key = groupKey(line.selection);
      if (!groups[key]) groups[key] = { key, selection: line.selection, label: line.periodLabel, items: [], count: 0 };
      groups[key].items.push(line);
      groups[key].count += line.q;
    });
    return Object.values(groups).sort((left, right) =>
      left.selection.date.localeCompare(right.selection.date)
      || ['lunch', 'dinner'].indexOf(left.selection.mealPeriod) - ['lunch', 'dinner'].indexOf(right.selection.mealPeriod));
  },
};

function unavailableLabel(entry, availability) {
  if (!availability) return '';
  const state = availability[entry.product.id];
  if (!state) return '该时段不供应';
  if (state.soldOut) return '已售罄';
  if (!state.orderable) return '当前不可下单';
  return '';
}

module.exports = { STATUS_MAP, statusTone, nav, buildUrl, pickup, cart };
