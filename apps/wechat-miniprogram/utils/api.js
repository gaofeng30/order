/* ============================================================
   接口契约层 —— 会员等级 / 会员名单 / 优惠券 / 菜品
   二期能力，不在一期合同范围。

   约定：
   1. 页面只调用本文件的方法，一律返回 Promise，不直接读写数据源。
   2. 每个方法上方注释即后端接口契约（方法 + 路径 + 出入参）。
      一期后端就位后，只替换本文件内部实现，页面代码不动。
   3. 当前实现为进程内内存态（globalData），冷启动回到种子数据。
      不使用 wx.setStorage / USER_DATA_PATH —— 商户端与用户端是不同设备，
      本机存储在架构上不成立，必须由服务端承载。
   ============================================================ */
const data = require('./data.js');

function g() { return getApp().globalData; }

// 统一模拟网络往返，让页面从一开始就按异步写，避免后端接入时重写
function ok(payload) {
  return new Promise(resolve => setTimeout(() => resolve(payload), 60));
}
function fail(msg) {
  return new Promise((resolve, reject) => setTimeout(() => reject(new Error(msg)), 60));
}
const clone = v => JSON.parse(JSON.stringify(v));
const uid = p => p + Date.now().toString(36) + Math.floor(Math.random() * 1e4).toString(36);

/* ---------------- 会员等级 ---------------- */

// GET /admin/member-levels → { levels: Level[] }  按 sort 升序
function listLevels() {
  return ok(clone(g().levels.slice().sort((a, b) => a.sort - b.sort)));
}

// POST /admin/member-levels        新增（无 id）
// PUT  /admin/member-levels/:id    修改
// body: { name, discount, desc, sort }
function saveLevel(lv) {
  const list = g().levels;
  const name = (lv.name || '').trim();
  if (!name) return fail('等级名称必填');
  if (list.some(x => x.name === name && x.id !== lv.id)) return fail('等级名称已存在');
  const discount = Number(lv.discount);
  if (!(discount > 0 && discount <= 100)) return fail('折扣需在 1–100 之间');
  if (lv.id) {
    const i = list.findIndex(x => x.id === lv.id);
    if (i < 0) return fail('等级不存在');
    list[i] = Object.assign({}, list[i], { name, discount, desc: lv.desc || '' });
    return ok(clone(list[i]));
  }
  const sort = list.reduce((a, x) => Math.max(a, x.sort), 0) + 1;
  const created = { id: uid('lv'), name, sort, discount, desc: lv.desc || '' };
  list.push(created);
  return ok(clone(created));
}

// GET /admin/member-levels/:id/impact → { memberCount, coupons: [{id,name}] }
// 删除前预览影响面，供迁移弹层展示
function levelImpact(id) {
  const memberCount = g().members.filter(m => m.levelId === id).length;
  const coupons = g().coupons.filter(c => c.levelIds.indexOf(id) > -1).map(c => ({ id: c.id, name: c.name }));
  return ok({ memberCount, coupons });
}

// DELETE /admin/member-levels/:id?migrateTo=<levelId|none>
// 该档会员整体迁移至 migrateTo（'none' = 降为非会员，即移出名单）；
// 券的适用等级中摘除该档，摘空则该券自动停用。
function deleteLevel(id, migrateTo) {
  const s = g();
  if (s.levels.length <= 1) return fail('至少保留一个等级');
  const i = s.levels.findIndex(x => x.id === id);
  if (i < 0) return fail('等级不存在');
  if (migrateTo !== 'none' && !s.levels.some(x => x.id === migrateTo)) return fail('迁移目标等级不存在');

  let disabled = 0;
  if (migrateTo === 'none') {
    s.members = s.members.filter(m => m.levelId !== id);
  } else {
    s.members.forEach(m => { if (m.levelId === id) m.levelId = migrateTo; });
  }
  s.coupons.forEach(c => {
    const j = c.levelIds.indexOf(id);
    if (j > -1) {
      c.levelIds.splice(j, 1);
      if (!c.levelIds.length && c.enabled) { c.enabled = false; disabled++; }
    }
  });
  s.levels.splice(i, 1);
  return ok({ disabledCoupons: disabled });
}

// PUT /admin/member-levels/order  body: { ids: string[] }  按数组顺序重排 sort
function reorderLevels(ids) {
  const s = g();
  ids.forEach((id, idx) => {
    const lv = s.levels.find(x => x.id === id);
    if (lv) lv.sort = idx + 1;
  });
  return listLevels();
}

/* ---------------- 会员名单 ---------------- */

// GET /admin/members?kw=&levelId=&page= → { list: Member[], total }
// kw 同时匹配手机号与姓名
function listMembers(opts) {
  const o = opts || {};
  const kw = (o.kw || '').trim();
  const list = g().members.filter(m => {
    if (o.levelId && m.levelId !== o.levelId) return false;
    if (kw && m.phone.indexOf(kw) < 0 && m.name.indexOf(kw) < 0) return false;
    return true;
  });
  return ok({ list: clone(list), total: list.length });
}

// GET /admin/members/:id → Member
function getMember(id) {
  const m = g().members.find(x => x.id === id);
  return m ? ok(clone(m)) : fail('会员不存在');
}

// POST /admin/members       新增（无 id）
// PUT  /admin/members/:id   修改
// body: { phone, name, levelId, org, dept, jobNo, remark, enabled }
function saveMember(mb) {
  const s = g();
  const phone = (mb.phone || '').trim();
  const name = (mb.name || '').trim();
  if (!/^1[3-9]\d{9}$/.test(phone)) return fail('手机号格式不正确');
  if (!name) return fail('姓名必填');
  if (!s.levels.some(l => l.id === mb.levelId)) return fail('请选择会员等级');
  if (s.members.some(x => x.phone === phone && x.id !== mb.id)) return fail('该手机号已在名单中');

  const patch = {
    phone, name, levelId: mb.levelId,
    org: mb.org || '', dept: mb.dept || '', jobNo: mb.jobNo || '',
    remark: mb.remark || '', enabled: mb.enabled !== false,
  };
  if (mb.id) {
    const i = s.members.findIndex(x => x.id === mb.id);
    if (i < 0) return fail('会员不存在');
    s.members[i] = Object.assign({}, s.members[i], patch);
    return ok(clone(s.members[i]));
  }
  const created = Object.assign({ id: uid('m'), joinAt: data.TODAY, bound: false, spend: 0, orders: 0 }, patch);
  s.members.push(created);
  return ok(clone(created));
}

// DELETE /admin/members/:id
function deleteMember(id) {
  const s = g();
  const i = s.members.findIndex(x => x.id === id);
  if (i < 0) return fail('会员不存在');
  s.members.splice(i, 1);
  return ok(true);
}

// POST /admin/members/import/preview  body: { rows: RawRow[] }
// → { adds: Row[], updates: Row[], errors: [{ line, raw, reason }] }
// 手机号为唯一键：名单中已存在 → 更新（覆盖姓名与等级，保留加入时间/消费/绑定关系）
function previewImport(rows) {
  const s = g();
  const adds = [];
  const updates = [];
  const errors = [];
  const seen = {};
  rows.forEach(r => {
    const phone = (r.phone || '').trim();
    const name = (r.name || '').trim();
    const levelName = (r.levelName || '').trim();
    const push = reason => errors.push({ line: r.line, raw: r.raw, reason });
    if (!/^1[3-9]\d{9}$/.test(phone)) return push('手机号格式不正确');
    if (!name) return push('姓名为空');
    const lv = s.levels.find(l => l.name === levelName);
    if (!lv) return push(levelName ? `等级「${levelName}」不存在` : '等级为空');
    if (seen[phone]) return push(`与第 ${seen[phone]} 行手机号重复`);
    seen[phone] = r.line;
    const row = {
      line: r.line, phone, name, levelId: lv.id, levelName: lv.name,
      org: r.org || '', dept: r.dept || '', jobNo: r.jobNo || '',
    };
    if (s.members.some(m => m.phone === phone)) updates.push(row);
    else adds.push(row);
  });
  return ok({ adds, updates, errors });
}

// POST /admin/members/import/commit  body: { rows: Row[] }  → { added, updated }
function commitImport(rows) {
  const s = g();
  let added = 0;
  let updated = 0;
  rows.forEach(r => {
    const i = s.members.findIndex(m => m.phone === r.phone);
    if (i > -1) {
      s.members[i] = Object.assign({}, s.members[i], {
        name: r.name, levelId: r.levelId, org: r.org, dept: r.dept, jobNo: r.jobNo,
      });
      updated++;
    } else {
      s.members.push({
        id: uid('m'), phone: r.phone, name: r.name, levelId: r.levelId,
        org: r.org, dept: r.dept, jobNo: r.jobNo, remark: '', enabled: true,
        joinAt: data.TODAY, bound: false, spend: 0, orders: 0,
      });
      added++;
    }
  });
  return ok({ added, updated });
}

/* ---------------- 优惠券 ---------------- */

// GET /admin/coupons → Coupon[]
function listCoupons() {
  return ok(clone(g().coupons));
}

// GET /admin/coupons/:id → Coupon
function getCoupon(id) {
  const c = g().coupons.find(x => x.id === id);
  return c ? ok(clone(c)) : fail('优惠券不存在');
}

// POST /admin/coupons      新增（无 id）
// PUT  /admin/coupons/:id  修改
function saveCoupon(c) {
  const s = g();
  const name = (c.name || '').trim();
  if (!name) return fail('券名称必填');
  if (!c.levelIds || !c.levelIds.length) return fail('至少选择一个适用等级');
  if (c.type === 'cut') {
    if (!(Number(c.amount) > 0)) return fail('减免金额需大于 0');
  } else {
    const rate = Number(c.rate);
    if (!(rate > 0 && rate < 100)) return fail('折扣需在 1–99 之间');
    if (!(Number(c.cap) > 0)) return fail('折扣券必须填写封顶金额');
  }
  if (c.scope === 'cat' && !(c.catNames || []).length) return fail('请选择适用分类');
  if (c.scope === 'item' && !(c.itemIds || []).length) return fail('请选择适用菜品');
  if (!c.start || !c.end) return fail('请填写有效期');
  if (c.start > c.end) return fail('结束日期不能早于开始日期');
  if (!(Number(c.perLimit) > 0)) return fail('每人可用次数需大于 0');

  const patch = {
    name, type: c.type,
    amount: c.type === 'cut' ? Number(c.amount) : 0,
    rate: c.type === 'discount' ? Number(c.rate) : 0,
    cap: c.type === 'discount' ? Number(c.cap) : 0,
    threshold: Number(c.threshold) || 0,
    levelIds: c.levelIds.slice(),
    scope: c.scope,
    catNames: c.scope === 'cat' ? (c.catNames || []).slice() : [],
    itemIds: c.scope === 'item' ? (c.itemIds || []).slice() : [],
    start: c.start, end: c.end,
    perLimit: Number(c.perLimit),
    enabled: c.enabled !== false,
  };
  if (c.id) {
    const i = s.coupons.findIndex(x => x.id === c.id);
    if (i < 0) return fail('优惠券不存在');
    s.coupons[i] = Object.assign({}, s.coupons[i], patch);
    return ok(clone(s.coupons[i]));
  }
  const created = Object.assign({ id: uid('cp') }, patch);
  s.coupons.push(created);
  return ok(clone(created));
}

// PUT /admin/coupons/:id/enabled  body: { enabled }
function setCouponEnabled(id, enabled) {
  const c = g().coupons.find(x => x.id === id);
  if (!c) return fail('优惠券不存在');
  c.enabled = !!enabled;
  return ok(clone(c));
}

// DELETE /admin/coupons/:id
function deleteCoupon(id) {
  const s = g();
  const i = s.coupons.findIndex(x => x.id === id);
  if (i < 0) return fail('优惠券不存在');
  s.coupons.splice(i, 1);
  return ok(true);
}

/* ---------------- 用户侧 ---------------- */

// GET /me/membership → { isMember, member, level }
// 真实实现：前端 getPhoneNumber 拿到 code → 服务端换取明文手机号 → 命中名单 → 返回等级
function getMyMembership() {
  const s = g();
  const m = s.members.find(x => x.phone === data.ME.phone && x.enabled);
  if (!m) return ok({ isMember: false, member: null, level: null });
  const level = s.levels.find(l => l.id === m.levelId) || null;
  return ok({ isMember: !!level, member: clone(m), level: clone(level) });
}

// GET /me/coupons → Coupon[]  仅返回本人等级命中且已启用的券（含已过期，由前端分态展示）
function listMyCoupons() {
  const s = g();
  const m = s.members.find(x => x.phone === data.ME.phone && x.enabled);
  if (!m) return ok([]);
  return ok(clone(s.coupons.filter(c => c.enabled && c.levelIds.indexOf(m.levelId) > -1)));
}

// GET /me/coupons/used → { [couponId]: usedCount }
function myCouponUsed() {
  return ok(clone(g().couponUsed));
}

/* ---------------- 菜品 ---------------- */

// GET /admin/products → Product[]
function listProducts() {
  return ok(clone(g().menu));
}

// GET /admin/products/:id → Product
function getProduct(id) {
  const m = g().menu.find(x => x.id === id);
  return m ? ok(clone(m)) : fail('菜品不存在');
}

// POST /admin/products      新增（无 id）
// PUT  /admin/products/:id  修改
// body: { name, price, stock, cat, desc, imgs }
function saveProduct(p) {
  const s = g();
  const name = (p.name || '').trim();
  if (!name) return fail('菜品名称必填');
  const price = Number(p.price);
  if (!(price > 0)) return fail('价格需大于 0');
  const stock = Number(p.stock);
  if (!(stock >= 0)) return fail('库存不能为负');
  if (!p.cat) return fail('请选择分类');
  const imgs = (p.imgs || []).slice(0, 3);

  const patch = { name, price, stock, cat: p.cat, desc: p.desc || '', imgs, img: imgs[0] || '' };
  if (p.id) {
    const i = s.menu.findIndex(x => x.id === p.id);
    if (i < 0) return fail('菜品不存在');
    s.menu[i] = Object.assign({}, s.menu[i], patch);
    return ok(clone(s.menu[i]));
  }
  const created = Object.assign({
    id: uid('p'), sold: 0, status: 'on', tags: [], allergens: ['无'], specs: [],
  }, patch);
  s.menu.push(created);
  return ok(clone(created));
}

// DELETE /admin/products/:id
// 菜品删除后，从券的「指定菜品」范围中自动摘除；摘空则该券自动停用。
function deleteProduct(id) {
  const s = g();
  const i = s.menu.findIndex(x => x.id === id);
  if (i < 0) return fail('菜品不存在');
  let disabled = 0;
  s.coupons.forEach(c => {
    if (c.scope !== 'item') return;
    const j = c.itemIds.indexOf(id);
    if (j > -1) {
      c.itemIds.splice(j, 1);
      if (!c.itemIds.length && c.enabled) { c.enabled = false; disabled++; }
    }
  });
  s.menu.splice(i, 1);
  return ok({ disabledCoupons: disabled });
}

// PUT /admin/products/:id/status  body: { status: 'on'|'soldout'|'off' }
function setProductStatus(id, status) {
  const m = g().menu.find(x => x.id === id);
  if (!m) return fail('菜品不存在');
  m.status = status;
  return ok(clone(m));
}

// POST /upload  multipart → { url }
// 真实实现：上传至对象存储后返回可跨设备访问的 URL。
// 当前返回 wx.chooseMedia 的临时路径，仅在本次运行期内有效。
function uploadImage(tempFilePath) {
  return ok(tempFilePath);
}

module.exports = {
  listLevels, saveLevel, levelImpact, deleteLevel, reorderLevels,
  listMembers, getMember, saveMember, deleteMember, previewImport, commitImport,
  listCoupons, getCoupon, saveCoupon, setCouponEnabled, deleteCoupon,
  getMyMembership, listMyCoupons, myCouponUsed,
  listProducts, getProduct, saveProduct, deleteProduct, setProductStatus, uploadImage,
};
