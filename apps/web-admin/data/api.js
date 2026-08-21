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

  const MEALS = ['all', 'lunch', 'dinner'];
  const MEAL_LABEL = { all: '全天', lunch: '午餐', dinner: '晚餐' };

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
  // body: { name, price, cat, meal, desc, imgs }
  //   meal: 'all' 全天 | 'lunch' 午餐 | 'dinner' 晚餐（必填，PRD §6.3）
  function saveProduct(p) {
    const s = g();
    const name = (p.name || '').trim();
    if (!name) return fail('菜品名称必填');
    const price = Number(p.price);
    if (!(price > 0)) return fail('价格需大于 0');
    if (!p.cat) return fail('请选择分类');
    if (!MEALS.includes(p.meal)) return fail('请选择餐段可售');
    const imgs = (p.imgs || []).slice(0, 3);

    const patch = { name, price, cat: p.cat, meal: p.meal, desc: p.desc || '', imgs, img: imgs[0] || '' };
    if (p.id) {
      const i = s.menu.findIndex(x => x.id === p.id);
      if (i < 0) return fail('菜品不存在');
      s.menu[i] = Object.assign({}, s.menu[i], patch);
      return ok(clone(s.menu[i]));
    }
    const created = Object.assign({
      id: uid('p'), status: 'on', specs: [],
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

  /* 六态状态机（生效 spec: Orders use one six-state production state machine）
     已预约 ──取餐前 30 分钟，服务端定时推进──▶ 制作中 ──备好──▶ 待取餐 ──核销──▶ 已完成
     NEXT 只含商户可执行的转换；`已预约 → 制作中` 由服务端定时任务驱动。
     生产禁止撤销或回退已完成的转换。 */
  const NEXT = { 制作中: '待取餐', 待取餐: '已完成' };
  const ACT = { 已预约: '待开做', 制作中: '备好', 待取餐: '核销', 已完成: '查看', 退款中: '查看', 已退款: '查看' };
  const LANES = ['已预约', '制作中', '待取餐', '已完成', '已退款', '全部'];

  const STATUS_MAP = {
    已预约: 'info', 制作中: 'info', 待取餐: 'info',
    已完成: 'ok', 成功: 'ok', 已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
    退款中: 'warn',
    已退款: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute',
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
    const act = ACT[o.status];
    o.status = nx;
    return ok({ next: nx, act, code: o.code });
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


  /* ---------------- 员工折扣白名单 / 全局折扣率（PRD §6.4） ---------------- */

  const PHONE_RE = /^1[3-9]\d{9}$/;
  const normPhone = v => String(v == null ? '' : v).replace(/[\s-]/g, '');

  // GET /admin/staff-whitelist → { list: StaffWhitelist[] }
  function listStaff(kw) {
    const q = (kw || '').trim();
    const all = clone(g().staff);
    if (!q) return ok(all);
    return ok(all.filter(r => r.phone.includes(q) || r.name.includes(q)));
  }

  // POST /admin/staff-whitelist       新增（无 id）
  // PUT  /admin/staff-whitelist/:id   修改
  // body: { phone, name } —— 只有这两个可填字段
  function saveStaff(r) {
    const s = g();
    const phone = normPhone(r.phone);
    if (!phone) return fail('手机号必填');
    if (!PHONE_RE.test(phone)) return fail('手机号格式不正确');
    const name = String(r.name == null ? '' : r.name).trim();
    if (!name) return fail('姓名必填');
    const dup = s.staff.find(x => x.phone === phone && x.id !== r.id);
    if (dup) return fail(`手机号 ${phone} 已在名单中（${dup.name}）`);

    if (r.id) {
      const i = s.staff.findIndex(x => x.id === r.id);
      if (i < 0) return fail('记录不存在');
      // 只覆盖两个可填字段，系统字段原样保留
      s.staff[i] = Object.assign({}, s.staff[i], { phone, name });
      return ok(clone(s.staff[i]));
    }
    const created = {
      id: uid('s'), phone, name,
      enabled: true, joinAt: window.Seed.TODAY, bound: false, spend: 0, orders: 0,
    };
    s.staff.push(created);
    return ok(clone(created));
  }

  // PUT /admin/staff-whitelist/:id/enabled  body: { enabled }
  // 停用保留记录但暂停折扣，用于离职人员；系统字段不受影响
  function setStaffEnabled(id, enabled) {
    const r = g().staff.find(x => x.id === id);
    if (!r) return fail('记录不存在');
    r.enabled = !!enabled;
    return ok(clone(r));
  }

  // DELETE /admin/staff-whitelist/:id
  function deleteStaff(id) {
    const s = g();
    const i = s.staff.findIndex(x => x.id === id);
    if (i < 0) return fail('记录不存在');
    s.staff.splice(i, 1);
    return ok({});
  }

  // GET /admin/discount-rate → number（员工实付百分比，整数 1-100）
  function getDiscountRate() { return ok(g().settings.discountRate); }

  // PUT /admin/discount-rate  body: { rate }
  // 只影响新报价，不回算历史订单（PRD §6.4、§9.1）
  function saveDiscountRate(rate) {
    const n = Number(rate);
    if (!Number.isInteger(n)) return fail('折扣率必须是整数百分比');
    if (n < 1 || n > 100) return fail('折扣率需在 1 到 100 之间');
    g().settings.discountRate = n;
    return ok(n);
  }


  /* ---------------- 批量导入（PRD §6.13） ----------------
     三步：预览 → 确认 → 提交。预览不写任何数据，只返回计数、异常行与一次性令牌；
     提交按令牌生效且幂等，重复提交只返回首次结果。
     解析由 window.Xlsx 承担（P0 原型例外，见 data/xlsx.js 文件头）。 */

  const MAX_IMPORT_ROWS = 500;
  const _imports = {};                       // token -> { kind, plan, done, result }
  const MEAL_BY_LABEL = { 全天: 'all', 午餐: 'lunch', 晚餐: 'dinner' };

  const cell = (row, i) => String(row && row[i] != null ? row[i] : '').trim();

  // 按表头名匹配列，不依赖列顺序；返回 { index, ignored }
  function mapHeader(header, known) {
    const index = {};
    const ignored = [];
    header.forEach((raw, i) => {
      const name = String(raw == null ? '' : raw).trim();
      if (!name) return;
      if (known.includes(name)) index[name] = i;
      else ignored.push(name);
    });
    return { index, ignored };
  }

  async function readSheet(file, required, known) {
    const rows = await window.Xlsx.readRows(file);
    if (!rows.length) throw new Error('文件为空');
    const { index, ignored } = mapHeader(rows[0], known);
    const missing = required.filter(c => index[c] === undefined);
    if (missing.length) throw new Error(`表头缺少必填列：${missing.join('、')}`);
    const body = rows.slice(1).filter(r => r.some(v => String(v == null ? '' : v).trim() !== ''));
    if (body.length > MAX_IMPORT_ROWS) throw new Error(`单次最多导入 ${MAX_IMPORT_ROWS} 行，请分批导入`);
    return { index, ignored, body };
  }

  function stashPreview(kind, plan, extra) {
    const token = uid('imp');
    _imports[token] = { kind, plan, done: false, result: null };
    return Object.assign({
      token,
      added: plan.add.length,
      updated: plan.update.length,
      errors: plan.errors,
    }, extra);
  }

  function takeCommit(kind, token) {
    const job = _imports[token];
    if (!job || job.kind !== kind) return fail('预览已失效，请重新选择文件');
    if (job.done) return ok(Object.assign({}, job.result, { duplicate: true }));
    return job;
  }

  /* ---- 员工白名单：按手机号覆盖更新，保留状态与统计字段 ---- */
  const STAFF_COLS = ['姓名', '手机号'];

  function previewStaffImport(file) {
    return readSheet(file, STAFF_COLS, STAFF_COLS).then(({ index, ignored, body }) => {
      const s = g();
      const plan = { add: [], update: [], errors: [] };
      const seen = {};
      body.forEach((row, i) => {
        const line = i + 2;                                  // 1-based，含表头
        const name = cell(row, index['姓名']);
        const phone = normPhone(cell(row, index['手机号']));
        if (!phone) return plan.errors.push({ row: line, reason: '手机号必填' });
        if (!PHONE_RE.test(phone)) return plan.errors.push({ row: line, reason: `手机号「${phone}」格式不正确` });
        if (!name) return plan.errors.push({ row: line, reason: '姓名必填' });
        if (seen[phone]) return plan.errors.push({ row: line, reason: `手机号 ${phone} 在本文件中重复（第 ${seen[phone]} 行已出现）` });
        seen[phone] = line;
        const hit = s.staff.find(x => x.phone === phone);
        (hit ? plan.update : plan.add).push({ row: line, phone, name, id: hit ? hit.id : '' });
      });
      return stashPreview('staff', plan, { ignoredColumns: ignored });
    });
  }

  function commitStaffImport(token) {
    const job = takeCommit('staff', token);
    if (job.then || job.catch) return job;                   // fail() 或幂等命中
    const s = g();
    job.plan.add.forEach(r => s.staff.push({
      id: uid('s'), phone: r.phone, name: r.name,
      enabled: true, joinAt: window.Seed.TODAY, bound: false, spend: 0, orders: 0,
    }));
    // 覆盖只动手机号与姓名：状态、加入时间、绑定关系与统计一律保留
    job.plan.update.forEach(r => {
      const i = s.staff.findIndex(x => x.id === r.id);
      if (i >= 0) s.staff[i] = Object.assign({}, s.staff[i], { phone: r.phone, name: r.name });
    });
    job.result = { added: job.plan.add.length, updated: job.plan.update.length };
    job.done = true;
    return ok(Object.assign({}, job.result, { duplicate: false }));
  }

  /* ---- 菜品：只新增不更新；分类不存在则自动新建 ---- */
  const PRODUCT_REQUIRED = ['菜品名称', '售价', '分类', '餐段可售'];
  const PRODUCT_COLS = PRODUCT_REQUIRED.concat(['描述']);

  function previewProductImport(file) {
    return readSheet(file, PRODUCT_REQUIRED, PRODUCT_COLS).then(({ index, ignored, body }) => {
      const s = g();
      const plan = { add: [], update: [], errors: [] };
      const seenName = {};
      const newCats = [];
      body.forEach((row, i) => {
        const line = i + 2;
        const name = cell(row, index['菜品名称']);
        const priceRaw = cell(row, index['售价']);
        const cat = cell(row, index['分类']);
        const mealLabel = cell(row, index['餐段可售']);
        const desc = index['描述'] === undefined ? '' : cell(row, index['描述']);
        if (!name) return plan.errors.push({ row: line, reason: '菜品名称必填' });
        const price = Number(priceRaw);
        if (!priceRaw || !(price > 0)) return plan.errors.push({ row: line, reason: `售价「${priceRaw}」不是大于 0 的数值` });
        if (!cat) return plan.errors.push({ row: line, reason: '分类必填' });
        const meal = MEAL_BY_LABEL[mealLabel];
        if (!meal) return plan.errors.push({ row: line, reason: `餐段可售「${mealLabel}」不是全天 / 午餐 / 晚餐之一` });
        if (s.menu.some(m => m.name === name)) {
          return plan.errors.push({ row: line, reason: `菜品「${name}」已存在，导入只新增不覆盖；改价请用菜品管理的批量调价` });
        }
        if (seenName[name]) return plan.errors.push({ row: line, reason: `菜品「${name}」在本文件中重复（第 ${seenName[name]} 行已出现）` });
        seenName[name] = line;
        if (!s.cats.some(c => c.name === cat) && newCats.indexOf(cat) < 0) newCats.push(cat);
        plan.add.push({ row: line, name, price, cat, meal, desc });
      });
      return stashPreview('product', plan, { newCategories: newCats });
    });
  }

  function commitProductImport(token) {
    const job = takeCommit('product', token);
    if (job.then || job.catch) return job;
    const s = g();
    job.plan.add.forEach(r => {
      if (!s.cats.some(c => c.name === r.cat)) {
        s.cats.push({ id: uid('c'), name: r.cat, sort: s.cats.length + 1, on: true });
      }
      // 图片不进模板：导入的商品先无图上架，商户随后在菜品管理中逐个补图
      s.menu.push({
        id: uid('p'), name: r.name, price: r.price, cat: r.cat, meal: r.meal,
        desc: r.desc, img: '', imgs: [], status: 'on', specs: [],
      });
    });
    job.result = { added: job.plan.add.length, updated: 0, categoriesCreated: job.plan.add.length ? undefined : 0 };
    job.done = true;
    return ok(Object.assign({}, job.result, { duplicate: false }));
  }

  /* ---------------- 营业设置 / 开屏图层 ---------------- */

  // GET /admin/settings → { status, pickupStepMin, mealPeriods[], pickupPoint, notice }
  function getSettings() {
    return ok(Object.assign({ status: g().store.status }, clone(g().settings)));
  }

  // PUT /admin/settings
  function saveSettings(s) {
    const st = g();
    if (!s.notice && s.notice !== '') return fail('公告不能为空对象');
    const step = Number(s.pickupStepMin);
    if (!(step > 0)) return fail('取餐时间粒度需大于 0');
    const periods = s.mealPeriods || [];
    if (!periods.length) return fail('至少需要一个餐段');
    for (const p of periods) {
      if (!p.cutoff || !p.from || !p.to) return fail(`${p.name || p.key} 的截单与取餐时间必填`);
      if (p.from > p.to) return fail(`${p.name || p.key} 的取餐结束时间不能早于开始时间`);
    }
    st.store.status = s.status;
    st.settings = Object.assign({}, st.settings, {
      pickupStepMin: step,
      mealPeriods: periods.map(p => Object.assign({}, p)),
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
    advanceOrder, advanceMeta, statusTone, NEXT, ACT, LANES,
    getSettings, saveSettings, setStoreStatus,
    getLayer, saveLayer, clearLayer,
    imgUrl, MEALS, MEAL_LABEL,
    listStaff, saveStaff, setStaffEnabled, deleteStaff, getDiscountRate, saveDiscountRate,
    previewProductImport, commitProductImport, previewStaffImport, commitStaffImport, MAX_IMPORT_ROWS,
  };
})();
