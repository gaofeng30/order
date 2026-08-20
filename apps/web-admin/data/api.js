/* ============================================================
   接口契约层 —— 菜品 / 订单
   与 apps/wechat-miniprogram/utils/api.js 同方法名、同入参、同返回形状。

   约定：
   1. 页面只调用本文件的方法，一律返回 Promise，不直接读写数据源。
   2. 每个方法上方注释即后端接口契约（方法 + 路径 + 出入参）。
      一期后端就位后，只替换本文件内部实现，页面代码不动。
   3. 当前实现为进程内内存态（window.__store），刷新回到种子数据。
      不使用 localStorage —— 商户端与用户端是不同设备，
      本机存储在架构上不成立，必须由服务端承载。
   4. 本文件是 PC 端与小程序端的公共资产：后端就位时两端各自把
      内部实现替换为 fetch，方法签名不变，页面代码不动。
   ============================================================ */
(function () {
  const Seed = window.Seed;

  function g() { return window.__store; }

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
    return ok({});
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
  // 当前返回浏览器 blob: 地址，仅在本次运行期内有效（对应小程序端的临时路径）。
  function uploadImage(file) {
    return ok(URL.createObjectURL(file));
  }

  /* ---------------- 分类 ---------------- */

  // GET /admin/categories → Category[]
  function listCategories() {
    return ok(clone(g().cats.slice().sort((a, b) => a.sort - b.sort)));
  }

  // POST /admin/categories  body: { name }
  function addCategory(name) {
    const s = g();
    const nm = (name || '').trim();
    if (!nm) return fail('请输入分类名称');
    if (s.cats.some(c => c.name === nm)) return fail('该分类已存在');
    const sort = s.cats.reduce((a, c) => Math.max(a, c.sort), 0) + 1;
    const created = { id: uid('c'), name: nm, sort, on: true, count: 0 };
    s.cats.push(created);
    return ok(clone(created));
  }

  // PUT /admin/categories/:id/enabled  body: { on }
  function setCategoryEnabled(id, on) {
    const c = g().cats.find(x => x.id === id);
    if (!c) return fail('分类不存在');
    c.on = !!on;
    return ok(clone(c));
  }

  // DELETE /admin/categories/:id  分类下仍有菜品时不允许删除
  function deleteCategory(id) {
    const s = g();
    const i = s.cats.findIndex(x => x.id === id);
    if (i < 0) return fail('分类不存在');
    const used = s.menu.filter(m => m.cat === s.cats[i].name).length;
    if (used) return fail(`该分类下还有 ${used} 个菜品，请先移出后再删除`);
    s.cats.splice(i, 1);
    return ok(true);
  }

  // PUT /admin/categories/order  body: { ids: string[] }
  function reorderCategories(ids) {
    const s = g();
    ids.forEach((id, idx) => {
      const c = s.cats.find(x => x.id === id);
      if (c) c.sort = idx + 1;
    });
    return listCategories();
  }

  /* ---------------- 订单（对应小程序 utils/util.js 的订单状态机） ---------------- */

  // 待制作 ──备好──▶ 待取餐 ──核销──▶ 已完成
  const NEXT = { 待制作: '待取餐', 待取餐: '已完成' };
  const ACT = { 待制作: '备好', 待取餐: '核销', 已完成: '查看', 已取消: '查看' };
  const LANES = ['待制作', '待取餐', '已完成', '全部'];

  const STATUS_MAP = {
    待取餐: 'info', 待制作: 'info', 制作中: 'info', 进行中: 'info', 配送中: 'info', 已预约: 'info',
    已完成: 'ok', 成功: 'ok', 已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
    待支付: 'warn', 待取超时: 'warn', 库存告急: 'warn',
    已取消: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute', 未开放: 'mute',
  };
  const statusTone = s => STATUS_MAP[s] || 'mute';

  // GET /admin/orders?lane= → Order[]
  function listOrders(lane) {
    const all = g().aOrders;
    const list = (!lane || lane === '全部') ? all : all.filter(o => o.status === lane);
    return ok(clone(list));
  }

  // 各泳道计数（前端聚合，后端可合并进 listOrders 的响应头）
  function laneCounts() {
    const all = g().aOrders;
    const c = {};
    LANES.forEach(l => { c[l] = l === '全部' ? all.length : all.filter(o => o.status === l).length; });
    return c;
  }

  function findOrder(id) { return g().aOrders.find(o => o.id === id); }
  function findOrderByCode(code) {
    return g().aOrders.find(o => o.code.toUpperCase() === String(code).toUpperCase());
  }

  // 菜品摘要串
  function itemsSummary(items) {
    return items.map(([id, q]) => (Seed.itemById(id) || { name: '已删除菜品' }).name + '×' + q).join('，');
  }

  // PUT /admin/orders/:id/advance  单向推进一步，返回 { prev, next, act }
  // 与小程序端签名差异：小程序传 (id, toastComp, refresh)，PC 端 Toast 是全局单例，
  // 因此收敛为 (id) 并由调用方处理 Toast 与刷新。
  function advanceOrder(id) {
    const o = findOrder(id);
    if (!o) return fail('订单不存在');
    const nx = NEXT[o.status];
    if (!nx) return fail('该订单已是终态');
    const prev = o.status;
    o.status = nx;
    return ok({ prev, next: nx, act: ACT[prev], code: o.code });
  }

  // 回退一步（供 Toast 撤销用，对应小程序 onUndo 直接改 o.status）
  function revertOrder(id, prev) {
    const o = findOrder(id);
    if (o) o.status = prev;
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

  /* ---------------- 营业设置 / 开屏图层 ---------------- */

  // GET /admin/settings → { status, openTime, closeTime, cutoff, pickupPoint, notice }
  function getSettings() {
    return ok(Object.assign({ status: g().store.status }, clone(g().settings)));
  }

  // PUT /admin/settings
  function saveSettings(s) {
    const st = g();
    if (!s.notice && s.notice !== '') return fail('公告不能为空对象');
    st.store.status = s.status;
    st.settings = Object.assign({}, st.settings, {
      openTime: s.openTime, closeTime: s.closeTime, cutoff: s.cutoff,
      pickupPoint: s.pickupPoint, notice: s.notice,
    });
    return ok(clone(st.settings));
  }

  // PUT /admin/store/status  body: { status }  顶栏与工作台的快捷切换
  function setStoreStatus(status) {
    g().store.status = status;
    return ok(status);
  }

  // GET /admin/launch-layer → LayerConfig
  function getLayer() { return ok(clone(g().layer)); }

  // PUT /admin/launch-layer  body: LayerConfig
  function saveLayer(cfg) {
    g().layer = Object.assign({}, Seed.LAYER_DEFAULTS, cfg, { v: 1 });
    return ok(clone(g().layer));
  }

  // DELETE /admin/launch-layer
  function clearLayer() {
    g().layer = Object.assign({}, Seed.LAYER_DEFAULTS);
    return ok(true);
  }

  /* ---------------- 资源路径 ---------------- */
  // 种子里的图片是小程序绝对路径（/assets/...），PC 端映射到同一批真实物料，
  // 不复制文件；blob:/data:/http: 开头的（新上传）原样返回。
  function imgUrl(src) {
    if (!src) return '';
    if (src.indexOf('/assets/') === 0) return '../wechat-miniprogram' + src;
    return src;
  }

  window.Api = {
    listProducts, getProduct, saveProduct, deleteProduct, setProductStatus, uploadImage,
    listCategories, addCategory, setCategoryEnabled, deleteCategory, reorderCategories,
    listOrders, laneCounts, findOrder, findOrderByCode, itemsSummary,
    advanceOrder, revertOrder, advanceMeta, statusTone, NEXT, ACT, LANES,
    getSettings, saveSettings, setStoreStatus,
    getLayer, saveLayer, clearLayer,
    imgUrl,
  };
})();
