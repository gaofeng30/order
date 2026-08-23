/* PC Admin HTTP Adapter. 业务事实只来自服务端；网络或契约异常一律失败关闭。 */
(function () {
  'use strict';

  const API = '/api/v1';
  const MEALS = ['all', 'lunch', 'dinner'];
  const MEAL_LABEL = { all: '全天', lunch: '午餐', dinner: '晚餐' };
  const ROLES = ['owner', 'staff'];
  const ROLE_LABEL = { owner: '主账号', staff: '子账号' };
  const LANES = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款', '全部'];
  const MAX_IMPORT_ROWS = 500;
  const MAX_STAFF_IMPORT_ROWS = 5000;
  const STATUS_MAP = {
    已预约: 'info', 制作中: 'info', 待取餐: 'info', 已完成: 'ok', 成功: 'ok',
    已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
    退款中: 'warn', 已退款: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute',
  };
  const ORDER_STATE_LABEL = { RESERVED: '已预约', PREPARING: '制作中', READY_FOR_PICKUP: '待取餐', COMPLETED: '已完成', REFUNDING: '退款中', REFUNDED: '已退款' };
  const state = {
    products: [], categories: [], orders: [], pending: [], soldOut: new Set(),
    settings: null, account: null, imageURLs: new Map(),
  };

  class ApiError extends Error {
    constructor(message, status, code) {
      super(message || '服务暂时不可用，请稍后重试');
      this.name = 'ApiError';
      this.status = status || 0;
      this.code = code || 'UNAVAILABLE';
    }
  }

  function token() { return window.sessionStorage && window.sessionStorage.getItem('pc_session_token'); }
  function idempotencyKey() {
    return window.crypto && window.crypto.randomUUID ? window.crypto.randomUUID() : `pc-${Date.now()}-${Math.random()}`;
  }
  function qs(values) {
    const pairs = [];
    Object.keys(values || {}).forEach(k => {
      const v = values[k];
      if (v !== undefined && v !== null && v !== '') pairs.push(encodeURIComponent(k) + '=' + encodeURIComponent(v));
    });
    return pairs.length ? '?' + pairs.join('&') : '';
  }
  function unwrap(body) { return body && Object.prototype.hasOwnProperty.call(body, 'data') ? body.data : body; }
  async function request(path, options) {
    const opts = Object.assign({ method: 'GET' }, options || {});
    const headers = Object.assign({ Accept: 'application/json' }, opts.headers || {});
    const bearer = token();
    if (bearer) headers.Authorization = 'Bearer ' + bearer;
    if (opts.body !== undefined && !(opts.body instanceof FormData)) {
      headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(opts.body);
    }
    if (opts.method !== 'GET' && opts.method !== 'HEAD' && !opts.intrinsic) headers['Idempotency-Key'] = opts.idempotencyKey || idempotencyKey();
    delete opts.idempotencyKey;
    delete opts.intrinsic;
    opts.headers = headers;
    let response;
    try { response = await window.fetch(API + path, opts); }
    catch (_) { throw new ApiError('无法连接服务端，请检查网络后重试', 0, 'NETWORK_UNAVAILABLE'); }
    let body = null;
    const kind = response.headers && response.headers.get ? response.headers.get('content-type') || '' : '';
    try { body = kind.indexOf('application/json') >= 0 ? await response.json() : await response.text(); }
    catch (_) { throw new ApiError('服务端响应无法解析，请稍后重试', response.status, 'INVALID_RESPONSE'); }
    if (!response.ok) {
      const err = body && (body.error || body);
      throw new ApiError((err && (err.message || err.detail)) || `请求失败（${response.status}）`, response.status, err && err.code);
    }
    return unwrap(body);
  }
  function listOf(body, key) { return Array.isArray(body) ? body : ((body && (body[key] || body.items || body.list)) || []); }
  function centsToYuan(v) { return Math.round(Number(v || 0)) / 100; }
  function productOf(p) {
    const images = p.images || p.imgs || [];
    images.forEach(image => {
      if (image && typeof image === 'object' && image.object_key && image.url) state.imageURLs.set(image.object_key, image.url);
    });
    const imageKeys = images.map(x => typeof x === 'string' ? x : (x.object_key || x.key || '')).filter(Boolean);
    return {
      id: String(p.id), name: p.name, price: p.price !== undefined ? Number(p.price) : centsToYuan(p.price_cents),
      cat: p.category_name || p.cat || '', categoryId: String(p.category_id || p.categoryId || ''),
      meal: p.meal || p.meal_period || 'all', desc: p.description || p.desc || '',
      imgs: imageKeys, img: imageKeys[0] || p.image_url || p.img || '',
      status: p.listed !== undefined ? (p.listed ? 'on' : 'off') : (p.status === 'OFF' || p.enabled === false ? 'off' : (p.status === 'ON' ? 'on' : (p.status || 'off'))),
      soldOut: !!(p.sold_out || p.soldOut), specs: [],
    };
  }
  function categoryOf(c) {
    return { id: String(c.id), name: c.name, sort: Number(c.sort_order || c.sort || 0), on: c.enabled !== undefined ? !!c.enabled : !!c.on, count: Number(c.product_count || c.count || 0) };
  }
  function orderOf(o) {
    return Object.assign({}, o, {
      id: String(o.id), no: o.order_no || o.no, code: o.pickup_number || o.code,
      status: o.status_label || ORDER_STATE_LABEL[o.state] || o.status, pickupDate: o.pickup_date || o.service_date || o.pickupDate,
      pickupTime: o.pickup_time || o.pickupTime, mealPeriod: o.meal_period || o.mealPeriod, paidAt: o.paid_at || o.paidAt,
      txnId: o.transaction_id || o.provider_transaction_id || o.txnId, contact: o.contact_name || o.contact,
      phone: o.phone_masked || o.phone, orderNote: o.note || o.orderNote,
      subtotal: o.subtotal_cents !== undefined ? o.subtotal_cents : o.subtotal,
      discountRate: o.discount_rate_percent !== undefined ? o.discount_rate_percent : o.discountRate,
      discountCut: o.discount_cents !== undefined ? o.discount_cents : o.discountCut,
      total: o.payable_cents !== undefined ? o.payable_cents : o.total,
      items: (o.items || []).map(i => Array.isArray(i) ? i : [String(i.product_id || ''), i.product_name || i.name, i.quantity, i.line_total_cents]),
    });
  }
  function accountOf(a) {
    const rawRole = String(a.role || '').toUpperCase();
    return Object.assign({}, a, { id: String(a.id), phone: a.phone_masked || a.phone, role: rawRole === 'OWNER' ? 'owner' : 'staff', boundOpenId: a.bound ? 'bound' : (a.boundOpenId || '') });
  }

  async function bootstrap() {
    const results = await Promise.all([request('/admin/settings'), request('/admin/me')]);
    state.settings = results[0];
    state.account = accountOf(results[1].account || results[1]);
    return { settings: state.settings, account: state.account };
  }
  function logout() { if (window.sessionStorage) window.sessionStorage.removeItem('pc_session_token'); state.account = null; }
  function beginPCLogin() { return request('/admin/auth/qrcode', { method: 'POST', body: {}, intrinsic: true }); }
  async function pollPCLogin(loginID, pollSecret) {
    const result = await request('/admin/auth/poll', { method: 'POST', body: { login_id: loginID, poll_secret: pollSecret }, intrinsic: true });
    if (result.state === 'APPROVED' && result.session && result.session.token && window.sessionStorage) {
      window.sessionStorage.setItem('pc_session_token', result.session.token);
    }
    return result;
  }

  async function listProducts() {
    const body = await request('/admin/products' + qs({ service_date: today() }));
    state.products = listOf(body, 'products').map(productOf);
    state.soldOut = new Set(state.products.filter(p => p.soldOut).map(p => p.id + '|' + today()));
    return state.products.slice();
  }
  async function getProduct(id) { return productOf(await request('/admin/products/' + encodeURIComponent(id) + qs({ service_date: today() }))); }
  function productBody(p) {
    const category = state.categories.find(c => c.name === p.cat);
    return { name: String(p.name || '').trim(), price_cents: Math.round(Number(p.price) * 100), category_id: p.categoryId || (category && category.id) || '', meal_period: p.meal, description: p.desc || '', images: (p.imgs || []).slice(0, 3).map((key, sort) => ({ object_key: key, sort_order: sort })) };
  }
  async function saveProduct(p) {
    const saved = await request('/admin/products' + (p.id ? '/' + encodeURIComponent(p.id) : ''), { method: p.id ? 'PUT' : 'POST', body: productBody(p) });
    return productOf(saved.product || saved);
  }
  function deleteProduct(id) { return request('/admin/products/' + encodeURIComponent(id), { method: 'DELETE' }); }
  async function setShelf(id, status) { await request('/admin/products/' + encodeURIComponent(id) + '/status', { method: 'PUT', body: { status: status === 'on' ? 'ON' : 'OFF' } }); const product = state.products.find(p => p.id === String(id)); if (product) product.status = status; return product || { id: String(id), status }; }
  const setProductStatus = setShelf;
  async function setSoldOut(id, soldOut, serviceDate) {
    const day = serviceDate || today();
    const saved = await request('/admin/products/' + encodeURIComponent(id) + '/soldout', { method: 'PUT', body: { service_date: day, sold_out: !!soldOut } });
    const key = String(id) + '|' + day;
    if (soldOut) state.soldOut.add(key); else state.soldOut.delete(key);
    return saved;
  }
  function reorderProducts(categoryID, ids) { return request('/admin/products/order', { method: 'PUT', body: { category_id: String(categoryID), ids } }); }
  function isSoldOut(id, serviceDate) { return state.soldOut.has(String(id) + '|' + (serviceDate || today())); }
  function isSellable(product, serviceDate) { const p = typeof product === 'string' ? state.products.find(x => x.id === product) : product; return !!p && p.status === 'on' && !isSoldOut(p.id, serviceDate); }

  async function uploadImage(file) {
    const form = new FormData(); form.append('file', file);
    const body = await request('/upload', { method: 'POST', body: form });
    const image = body.image || body;
    if (image.object_key && image.url) state.imageURLs.set(image.object_key, image.url);
    return image.object_key || image.key;
  }
  async function listCategories() { const body = await request('/admin/categories'); state.categories = listOf(body, 'categories').map(categoryOf); return state.categories.slice(); }
  async function addCategory(name) { const b = await request('/admin/categories', { method: 'POST', body: { name: String(name || '').trim() } }); return categoryOf(b.category || b); }
  async function renameCategory(id, name) { const b = await request('/admin/categories/' + encodeURIComponent(id), { method: 'PUT', body: { name: String(name || '').trim() } }); return categoryOf(b.category || b); }
  async function setCategoryEnabled(id, on) { const b = await request('/admin/categories/' + encodeURIComponent(id), { method: 'PUT', body: { enabled: !!on } }); return categoryOf(b.category || b); }
  function deleteCategory(id) { return request('/admin/categories/' + encodeURIComponent(id), { method: 'DELETE' }); }
  async function reorderCategories(ids) { await request('/admin/categories/order', { method: 'PUT', body: { ids } }); return listCategories(); }

  async function listOrders(lane, opts) {
    const body = await request('/admin/orders' + qs({ state: lane && lane !== '全部' ? lane : '', unclaimed: opts && opts.uncollected ? 'true' : '' }));
    state.orders = listOf(body, 'orders').map(orderOf); return state.orders.slice();
  }
  async function searchOrders(q) { const body = await request('/admin/orders' + qs({ q })); state.orders = listOf(body, 'orders').map(orderOf); return state.orders.slice(); }
  function laneCounts() { const c = {}; LANES.forEach(l => { c[l] = l === '全部' ? state.orders.length : state.orders.filter(o => o.status === l).length; }); return c; }
  function uncollectedCount() { return state.orders.filter(o => o.unclaimed).length; }
  function findOrder(id) { return state.orders.find(o => o.id === String(id)); }
  function findOrderByCode(code) { return state.orders.find(o => o.code === String(code)); }
  function codeHint() { return ''; }
  function nestedObject(body, key) {
    const value = body && body[key];
    if (!value || typeof value !== 'object' || Array.isArray(value) || value.id === undefined || value.id === null || String(value.id) === '') throw new ApiError('服务端响应无法解析，请稍后重试', 200, 'INVALID_RESPONSE');
    return value;
  }
  async function refundOrder(id, reason) {
    const body = await request('/admin/orders/' + encodeURIComponent(id) + '/refund', { method: 'POST', body: { reason: String(reason || '').trim() } });
    return orderOf(nestedObject(body, 'order'));
  }
  const canRefund = status => ['已预约', '制作中', '待取餐', '已完成'].includes(status);

  const PENDING_REASON_LABEL = { PRODUCT_UNAVAILABLE: '商品不可售', PICKUP_EXPIRED: '取餐时间已过', SNAPSHOT_INVALID: '数据校验不通过' };
  async function listPendingPayments() {
    const b = await request('/admin/pending-payments');
    state.pending = listOf(b, 'prepayments').map(p => {
      const reason = p.blocking_reason || '';
      return Object.assign({}, p, {
        id: String(p.id), amount: p.amount_cents, outTradeNo: p.out_trade_no,
        txnId: p.transaction_id, contact: p.contact_name, phone: p.phone_masked,
        pickupDate: p.pickup_date, pickupTime: p.pickup_time, mealPeriod: p.meal_period,
        paidAt: p.paid_at, blockingReason: reason, cause: PENDING_REASON_LABEL[reason] || '需人工处理', causeDetail: reason,
        items: (p.items || []).map(i => [String(i.product_id || ''), i.name || i.product_name, i.quantity, i.line_total_cents]),
      });
    });
    return state.pending.slice();
  }
  function pendingPaymentCount() { return state.pending.length; }
  async function rebuildOrder(id) {
    const body = await request('/admin/pending-payments/' + encodeURIComponent(id), { method: 'POST', body: { action: 'MATERIALIZE', reason: '' } });
    return orderOf(nestedObject(body, 'order'));
  }
  async function refundPendingPayment(id, reason) {
    const current = state.pending.find(p => p.id === String(id)) || {};
    const body = await request('/admin/pending-payments/' + encodeURIComponent(id), { method: 'POST', body: { action: 'REFUND', reason: String(reason || '').trim() } });
    const refunded = nestedObject(body, 'refund');
    return Object.assign({}, current, { refund: refunded, refundId: String(refunded.id), refundState: refunded.state });
  }
  function blockingReason(p) { return p.blocking_reason || p.blockingReason || ''; }

  function rangePath(path, range) { return path + qs({ from: range && range.from, to: range && range.to }); }
  async function listPayments(range) { return listOf(await request(rangePath('/admin/finance/payments', range)), 'payments').map(orderOf); }
  async function listRefunds(range) { return listOf(await request(rangePath('/admin/finance/refunds', range)), 'refunds').map(r => Object.assign({}, r, { no: r.provider_refund_id || r.id, amount: r.amount_cents, status: r.state, at: r.requested_at, orderNo: r.order_no, paidAt: r.paid_at || '', txnId: r.transaction_id || '' })); }
  function financeSummary(range) { return request(rangePath('/admin/finance/summary', range)); }
  function buildPaymentExport(range) { return request(rangePath('/admin/finance/export', range)); }
  function dashboardStats() { return request('/admin/stats'); }

  function normalizePerson(r) {
    return Object.assign({}, r, { id: String(r.id), phone: r.phone_masked || r.phone, joinAt: r.created_at || r.joinAt, bound: !!(r.bound || r.bound_user_id), spend: r.spend_cents !== undefined ? centsToYuan(r.spend_cents) : Number(r.spend || 0), orders: Number(r.order_count || r.orders || 0) });
  }
  async function listStaff(kw) { return listOf(await request('/admin/staff-whitelist' + qs({ q: kw })), 'staff').map(normalizePerson); }
  async function saveStaff(r) { return normalizePerson(await request('/admin/staff-whitelist' + (r.id ? '/' + encodeURIComponent(r.id) : ''), { method: r.id ? 'PUT' : 'POST', body: { name: r.name, phone: r.phone } })); }
  async function setStaffEnabled(id, enabled) { return normalizePerson(await request('/admin/staff-whitelist/' + encodeURIComponent(id), { method: 'PUT', body: { enabled: !!enabled } })); }
  function deleteStaff(id) { return request('/admin/staff-whitelist/' + encodeURIComponent(id), { method: 'DELETE' }); }
  async function getDiscountRate() { const b = await request('/admin/discount-rate'); return Number(b.rate_percent); }
  async function saveDiscountRate(rate) { const b = await request('/admin/discount-rate', { method: 'PUT', body: { rate_percent: Number(rate) } }); return Number(b.rate_percent); }

  async function listMerchantAccounts(kw) { const b = await request('/admin/merchant-accounts' + qs({ q: kw })); return listOf(b, 'accounts').map(accountOf); }
  async function saveMerchantAccount(a) { return accountOf(await request('/admin/merchant-accounts' + (a.id ? '/' + encodeURIComponent(a.id) : ''), { method: a.id ? 'PUT' : 'POST', body: { name: a.name, phone: a.phone, role: a.role === 'owner' ? 'OWNER' : 'SUBACCOUNT' } })); }
  async function setMerchantAccountEnabled(id, enabled) { return accountOf(await request('/admin/merchant-accounts/' + encodeURIComponent(id), { method: 'PUT', body: { enabled: !!enabled } })); }
  function deleteMerchantAccount(id) { return request('/admin/merchant-accounts/' + encodeURIComponent(id), { method: 'DELETE' }); }

  async function previewImport(kind, file) {
    if (!file || !/\.xlsx$/i.test(file.name || '')) throw new ApiError('只接受 .xlsx 文件', 400, 'INVALID_FILE');
    if (file.size > 10 * 1024 * 1024) throw new ApiError('文件不能超过 10 MiB', 413, 'FILE_TOO_LARGE');
    const form = new FormData(); form.append('file', file);
    const path = kind === 'product' ? '/admin/products/import/preview' : '/admin/staff-whitelist/import/preview';
    const p = await request(path, { method: 'POST', body: form });
    return { token: p.preview_token, added: Number(p.new_count || 0), updated: Number(p.update_count || 0), errors: (p.rows || []).filter(r => r.outcome === 'ERROR').map(r => ({ row: r.row, reason: r.reason })), ignoredColumns: p.ignored_columns || [], newCategories: p.new_categories || [] };
  }
  async function commitImport(tokenValue) { const r = await request('/import/commit', { method: 'POST', body: { preview_token: tokenValue, skip_invalid: true } }); return { added: Number(r.new_count || 0), updated: Number(r.update_count || 0), skipped: Number(r.skipped_count || 0), duplicate: !!r.duplicate }; }
  const previewProductImport = file => previewImport('product', file);
  const previewStaffImport = file => previewImport('staff', file);
  const commitProductImport = commitImport;
  const commitStaffImport = commitImport;

  async function getSettings() {
    const s = await request('/admin/settings');
    const meals = s.meal_periods || s.mealPeriods || [];
    const receivedDates = s.service_dates || s.serviceDates || [];
    const byDate = new Map(receivedDates.map(d => [d.date, d.status]));
    const baseDate = s.service_date || s.serviceDate;
    const dates = baseDate ? [0, 1].map(offset => { const date = addISODate(baseDate, offset); return { date, status: byDate.get(date) || 'closed' }; }) : [];
    state.settings = Object.assign({}, s, { status: s.store_status || s.status, pickupStepMin: s.pickup_step_min || s.pickupStepMin, pickupPoint: s.pickup_point || s.pickupPoint, mealPeriods: meals.map(m => ({ key: (m.key || m.code || '').toLowerCase(), name: m.name, cutoff: m.cutoff_time || m.cutoff, from: m.pickup_from || m.from, to: m.pickup_to || m.to })), serviceDates: dates.map(d => ({ date: d.date, status: d.status })) });
    return state.settings;
  }
  async function saveSettings(s) { await request('/admin/settings', { method: 'PUT', body: { store_status: s.status, pickup_step_min: Number(s.pickupStepMin), pickup_point: s.pickupPoint, notice: s.notice, meal_periods: s.mealPeriods.map(p => ({ code: p.key, name: p.name, cutoff_time: p.cutoff, pickup_from: p.from, pickup_to: p.to })), service_dates: s.serviceDates || [] } }); await getSettings(); return state.settings; }
  async function setStoreStatus(status) { const current = await getSettings(); await saveSettings(Object.assign({}, current, { status })); return status; }
  async function getLayer() { const l = await request('/admin/launch-layer'); return { src: l.image_object_key || l.src || '', enabled: !!l.enabled, size: Number(l.size_ratio || l.size || 0.3), cx: Number(l.center_x || l.cx || 0.5), cy: Number(l.center_y || l.cy || 0.35), ar: Number(l.aspect_ratio || l.ar || 1), v: l.version || l.v || 1 }; }
  function saveLayer(c) { return request('/admin/launch-layer', { method: 'PUT', body: { image_object_key: c.src, enabled: !!c.enabled, size_ratio: Number(c.size), center_x: Number(c.cx), center_y: Number(c.cy), aspect_ratio: Number(c.ar) } }); }
  function clearLayer() { return request('/admin/launch-layer', { method: 'DELETE' }); }

  function statusTone(v) { return STATUS_MAP[v] || 'mute'; }
  function yuan(cents) { const n = Number(cents); if (!Number.isFinite(n)) return '—'; const v = Math.abs(Math.round(n)); return (n < 0 ? '-' : '') + Math.floor(v / 100) + '.' + String(v % 100).padStart(2, '0'); }
  function itemsSummary(items) { return (items || []).map(i => Array.isArray(i) ? i[1] + '×' + i[2] : (i.name || i.product_name) + '×' + i.quantity).join('，'); }
  function addISODate(value, days) { const date = new Date(value + 'T00:00:00Z'); date.setUTCDate(date.getUTCDate() + days); return date.toISOString().slice(0, 10); }
  function today() { return state.settings && (state.settings.service_date || state.settings.serviceDate) || ''; }
  function currentAccount() { return state.account; }
  function storeView() { return { status: state.settings && (state.settings.store_status || state.settings.status) || '未知', name: state.settings && (state.settings.store_name || state.settings.storeName) || '门店', account: state.account }; }
  function imgUrl(src) { if (!src) return ''; if (/^(?:https?:|blob:|data:)/.test(src)) return src; return state.imageURLs.get(src) || ''; }

  window.Api = {
    ApiError, bootstrap, logout, beginPCLogin, pollPCLogin, storeView, dashboardStats,
    listProducts, getProduct, saveProduct, deleteProduct, setProductStatus, uploadImage, setShelf, setSoldOut, reorderProducts, isSoldOut, isSellable,
    listCategories, addCategory, renameCategory, setCategoryEnabled, deleteCategory, reorderCategories,
    listOrders, laneCounts, findOrder, findOrderByCode, itemsSummary, yuan, today, currentAccount, refundOrder, canRefund,
    uncollectedCount, searchOrders, codeHint, listPendingPayments, pendingPaymentCount, rebuildOrder, refundPendingPayment, blockingReason,
    listPayments, listRefunds, financeSummary, buildPaymentExport, statusTone, LANES,
    getSettings, saveSettings, setStoreStatus, getLayer, saveLayer, clearLayer, imgUrl, MEALS, MEAL_LABEL,
    listStaff, saveStaff, setStaffEnabled, deleteStaff, getDiscountRate, saveDiscountRate,
    previewProductImport, commitProductImport, previewStaffImport, commitStaffImport, MAX_IMPORT_ROWS, MAX_STAFF_IMPORT_ROWS,
    listMerchantAccounts, saveMerchantAccount, setMerchantAccountEnabled, deleteMerchantAccount, ROLES, ROLE_LABEL,
  };
})();
