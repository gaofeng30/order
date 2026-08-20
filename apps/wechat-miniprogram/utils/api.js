/* ============================================================
   接口契约层 —— 菜品

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
  s.menu.splice(i, 1);
  return ok({ });
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
  listProducts, getProduct, saveProduct, deleteProduct, setProductStatus, uploadImage,
};
