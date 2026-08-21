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
  /* 六态各一条泳道（PRD §7.1、§15.5.3）。退款中必须自成一格：
     它是唯一需要人工盯着直到到账的状态，混在「全部」里等于没有。 */
  const LANES = ['已预约', '制作中', '待取餐', '已完成', '退款中', '已退款', '全部'];

  const STATUS_MAP = {
    已预约: 'info', 制作中: 'info', 待取餐: 'info',
    已完成: 'ok', 成功: 'ok', 已接单: 'ok', 已核销: 'ok', 营业中: 'ok', 可购: 'ok', 已授权: 'ok',
    退款中: 'warn',
    已退款: 'mute', 售罄: 'mute', 已下架: 'mute', 休息中: 'mute', 已截单: 'mute',
  };
  const statusTone = s => STATUS_MAP[s] || 'mute';

  // GET /admin/orders?lane= → Order[]
  /* opts.uncollected：§6.7 的「未取餐」口径 —— 营业日已结束仍处于 待取餐 的订单。
     它是一个筛选条件而不是第七个状态，所以不进 LANES 也不进 ACT，只在这里做交集。 */
  function listOrders(lane, opts) {
    const all = g().aOrders;
    let list = (!lane || lane === '全部') ? all : all.filter(o => o.status === lane);
    if (opts && opts.uncollected) {
      list = list.filter(o => o.status === '待取餐' && o.pickupDate < BUSINESS_DAY);
    }
    return ok(clone(list));
  }

  /* 订单搜索（§6.6 末条、§6.7）。PC 扫码核销页删除后，手工核销的定位由这里承担。

     一条规则决定了它的形状：**4 位取餐号只匹配当前营业日**。
     §7.8 明写「跨营业日的取餐号可能重复」，§6.6 因此规定手工输入只匹配当前营业日期 ——
     不限定的话，输入 0150 会同时命中今天和前天的两张单，核销就核错了人。

     订单号与手机号没有这个歧义（订单号全局唯一），所以它们跨全部营业日匹配。

     搜索跨泳道：要核销的时候并不知道那张单现在是什么状态。 */
  const CODE_RE = /^\d{4}$/;

  function _search(q) {
    const key = String(q == null ? '' : q).trim();
    if (!key) return [];
    const all = g().aOrders;
    const up = key.toUpperCase();
    /* 4 位纯数字既可能是取餐号，也可能是手机尾号。两者都匹配，但**只有取餐号那一半
       限定当前营业日** —— 跨日歧义是取餐号独有的（§7.8），手机号没有这个问题。 */
    if (CODE_RE.test(key)) {
      return all.filter(o =>
        (o.code === key && o.pickupDate === BUSINESS_DAY) ||
        String(o.phone).includes(key));
    }
    return all.filter(o =>
      o.no.toUpperCase().includes(up) ||
      String(o.phone).includes(key) ||
      String(o.contact).includes(key) ||
      o.code.includes(key));
  }

  // GET /admin/orders?q= → Order[]
  function searchOrders(q) { return ok(clone(_search(q))); }

  /* 「搜不到就以为没有」是上面那条规则最容易造成的误判：单子明明在，只是在别的营业日。
     所以当一个 4 位取餐号在当日无果、却存在于其他营业日时，把这个事实报出来并给出定位办法。 */
  function codeHint(q) {
    const key = String(q == null ? '' : q).trim();
    if (!CODE_RE.test(key)) return '';
    if (g().aOrders.some(o => o.code === key && o.pickupDate === BUSINESS_DAY)) return '';
    const other = g().aOrders.filter(o => o.code === key);
    if (!other.length) return '';
    const days = [...new Set(other.map(o => o.pickupDate))].sort().join('、');
    return `取餐号 ${key} 在当前营业日没有订单，但 ${days} 有同号订单。取餐号按营业日重复使用，`
         + `跨营业日的单不能凭取餐号核销 —— 请改用订单号 / 手机号搜索，或在「待取餐」泳道用「未取餐」筛选定位。`;
  }

  // 「未取餐」的条数，用于筛选按钮上的角标
  function uncollectedCount() {
    return g().aOrders.filter(o => o.status === '待取餐' && o.pickupDate < BUSINESS_DAY).length;
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

  /* 分 → 元的展示串。订单金额一律以整数分存储（PRD §15.6.2），
     页面 MUST NOT 自己做 /100 —— 一处手写就是一处四舍五入口径分叉。 */
  function yuan(cents) {
    const n = Number(cents);
    if (!Number.isFinite(n)) return '—';
    const neg = n < 0, v = Math.abs(Math.round(n));
    return (neg ? '-' : '') + Math.floor(v / 100) + '.' + String(v % 100).padStart(2, '0');
  }

  // 菜品摘要串
  /* 名称取自订单自身的快照（§15.6.2）。不按 id 回查商品表：
     订单是历史记录，商品改名或删除后它必须仍复述下单当时的事实。 */
  function itemsSummary(items) {
    return items.map(([, name, q]) => name + '×' + q).join('，');
  }

  /* 当前营业日。P0 用演示数据的“今天”，接后端后由服务端下发 ——
     页面 MUST NOT 各自硬编日期，否则订单页和财务页会在跨零点时各说各话。 */
  const BUSINESS_DAY = '2026-08-21';
  function today() { return BUSINESS_DAY; }

  /* 当前登录的商户账号。PC 后台仅主账号可登录（§4.4），退款的操作人取自这里，
     不是 Seed.MANAGER 那个装饰用的常量 —— 财务页要按操作人追责。 */
  function currentAccount() {
    const list = g().accounts || [];
    return clone(list.find(a => a.role === 'owner' && a.enabled) || list[0] || null);
  }

  /* POST /admin/orders/:id/refund  主账号发起退款（§6.7、§7.1 旁路、§7.7）

     三条硬规则：
     1. **只有全额。** §7.7 一期只支持原路全额退款，部分退款必须拒绝。这里的做法是
        接口根本不收金额入参 —— 表达不出来的请求不需要校验，也不会被下一个人"顺手加上"。
     2. **只到退款中。** 只有微信确认退款成功才是 已退款（§7.7）。PC 端把订单推到
        退款中就到头了，剩下的由支付回调驱动。前端自行置终态等于伪造到账。
     3. **幂等。** §7.1 要求相同幂等键重复请求返回第一次结果，不重复产生退款副作用。
        订单已在 退款中 / 已退款 时直接拒绝，既有退款记录一个字都不动。 */
  const REFUNDABLE = ['已预约', '制作中', '待取餐', '已完成'];
  const canRefund = st => REFUNDABLE.includes(st);

  function refundOrder(id, reason) {
    const o = findOrder(id);
    if (!o) return fail('订单不存在');
    if (o.refund || !REFUNDABLE.includes(o.status)) {
      return fail(`「${o.code}」当前为 ${o.status}，不能再次发起退款。退款结果以微信为准。`);
    }
    const why = String(reason == null ? '' : reason).trim();
    if (!why) return fail('请填写退款原因。退款不可撤销，原因会记入财务对账。');
    const me = currentAccount();
    o.refund = {
      no: refundNo(),
      amount: o.total,                 // 全额，且不接受调用方指定
      status: '退款中',
      operator: me ? me.name : '未知操作人',
      at: stamp(),
      reason: why,
    };
    o.status = '退款中';
    return ok(clone(o));
  }

  // 退款单号：微信退款单以 5000 开头，与交易号的 42000 区分
  let refundSeq = 100;
  function refundNo() {
    refundSeq += 1;
    return '5000' + BUSINESS_DAY.replace(/-/g, '') + String(refundSeq).padStart(9, '0');
  }
  function stamp() {
    const d = new Date();
    const p = n => String(n).padStart(2, '0');
    return `${BUSINESS_DAY} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
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


  /* ---------------- 财务与对账（PRD §6.11） ----------------
     归集口径是本节唯一要紧的事，两条都容易写反：

     1. 收款按 **支付日期**（paidAt）归集，不是营业日期。微信商户平台的交易账单
        以交易时间为准；预约单可以今天付明天取，按营业日期归集必然对不上。
     2. 退款按 **退款日期**（refund.at）归集，不是原订单的支付日期。跨日退款在
        微信账单里出现在到账那天，跟着原单走就会两天都错。

     净额一律按金额相减，不按订单数 —— 部分退款是合法情形。 */

  const dayOf = ts => String(ts || '').slice(0, 10);
  const inRange = (day, r) => (!r || !r.from || day >= r.from) && (!r || !r.to || day <= r.to);

  // 内部同步取数；对外的契约方法在下面统一包成异步，与其余接口一致
  function _payments(range) {
    return g().aOrders
      .filter(o => inRange(dayOf(o.paidAt), range))
      .slice()
      .sort((a, b) => (a.paidAt < b.paidAt ? 1 : a.paidAt > b.paidAt ? -1 : 0))
      .map(o => clone(o));
  }

  /* GET /admin/finance/refunds?from=&to= → Refund[]（按退款时间倒序）
     每条退款都带回订单号与微信交易号：对账时拿到一条退款要能立刻定位原单。 */
  function _refunds(range) {
    /* 被退款作废的待处理条目也要进台账：顾客确实付过钱，这笔退款同样出现在微信账单上。
       它们没有订单号，用预支付单号顶上，并标出来源 —— 对账时要能看出它不是普通退单。 */
    const voided = (g().voidedPending || [])
      .filter(v => inRange(dayOf(v.refund.at), range))
      .map(v => ({
        no: v.refund.no, amount: v.refund.amount, status: v.refund.status,
        operator: v.refund.operator, at: v.refund.at, reason: v.refund.reason,
        orderId: '', orderNo: v.outTradeNo, orderCode: '—', orderTotal: v.amount,
        txnId: v.txnId, paidAt: v.paidAt, contact: v.contact,
        fromPending: true,
      }));
    return voided.concat(g().aOrders
      .filter(o => o.refund && inRange(dayOf(o.refund.at), range))
      .map(o => ({
        no: o.refund.no, amount: o.refund.amount, status: o.refund.status,
        operator: o.refund.operator, at: o.refund.at, reason: o.refund.reason,
        orderId: o.id, orderNo: o.no, orderCode: o.code, orderTotal: o.total,
        txnId: o.txnId, paidAt: o.paidAt, contact: o.contact,
        fromPending: false,
      })))
      .sort((a, b) => (a.at < b.at ? 1 : a.at > b.at ? -1 : 0));
  }

  function _summary(range) {
    const pays = _payments(range), refunds = _refunds(range);
    const gross = pays.reduce((s, o) => s + o.total, 0);
    const refundAmount = refunds.reduce((s, r) => s + r.amount, 0);
    return {
      count: pays.length,
      gross,
      refundCount: refunds.length,
      refundAmount,
      net: gross - refundAmount,
      pendingCount: refunds.filter(r => r.status === '退款中').length,   // 未到账的退款笔数
      /* 已收款但没能建成订单的条目（§7.3）。钱已经在微信账户里，却不在任何订单上 ——
         不计入实收合计，但必须报出来，否则本页会比微信账单少这么多而没人知道为什么。 */
      unbuiltCount: _unbuilt(range).length,
      unbuiltAmount: _unbuilt(range).reduce((s, p) => s + p.amount, 0),
      staffCount: pays.filter(o => o.isStaff).length,
      discountCut: pays.reduce((s, o) => s + o.discountCut, 0),
    };
  }

  function _unbuilt(range) {
    return (g().pending || []).filter(p => inRange(dayOf(p.paidAt), range));
  }

  /* ---------------- 支付待处理（PRD §7.3） ----------------

     这些条目是「用户已付款、系统无订单」的兜底。它们**不是订单**：没有六态状态，
     也没有取餐号 —— 取餐号在订单生成时才分配（§7.8），而它们正是没能生成订单的那批。

     主账号只有两个出口，对应 §7.3 的「发起退款或人工建单」。 */

  function listPendingPayments() { return ok(clone(g().pending || [])); }
  function pendingPaymentCount() { return (g().pending || []).length; }

  function findPending(id) { return (g().pending || []).find(p => p.id === id); }
  function dropPending(id) {
    const list = g().pending || [];
    const i = list.findIndex(p => p.id === id);
    if (i >= 0) list.splice(i, 1);
  }

  /* 补建时必须重新校验阻塞原因。原因没解除就建单，等于给一道做不出来的菜发取餐号，
     顾客照样白跑一趟，而且这次是我们主动造成的。 */
  function blockingReason(p) {
    for (const [pid] of p.items) {
      const m = g().menu.find(x => x.id === pid);
      if (!m) return `「${pid}」已被删除，无法补建。请退款作废。`;
      if (m.status === 'off') return `「${m.name}」仍处于下架状态。请先在菜品管理重新上架，再补建订单。`;
      if (m.status === 'soldout') return `「${m.name}」当前售罄。请先恢复可售，再补建订单。`;
    }
    if (`${p.pickupDate} ${p.pickupTime}` < `${BUSINESS_DAY} ${nowHm()}`) {
      return `取餐时间 ${p.pickupDate} ${p.pickupTime} 已过，无法排产。请退款作废。`;
    }
    if (p.cause === '数据校验不通过') {
      return `该笔的${p.causeDetail || '数据校验未通过'}，需人工核对后处理。本期只提供退款作废。`;
    }
    return null;
  }

  /* POST /admin/pending-payments/:id/rebuild  人工建单 */
  function rebuildOrder(id) {
    const p = findPending(id);
    if (!p) return fail('该条目已被处理');
    const why = blockingReason(p);
    if (why) return fail(why);
    const order = {
      id: uid('o'),
      no: 'SA' + p.outTradeNo.replace(/\D/g, ''),
      code: nextPickupCode(p.pickupDate),      // §7.8 按取餐日期累计，当日唯一
      status: scheduleState(p),                // §7.4 不足 30 分钟直接进制作中
      pickupDate: p.pickupDate, pickupTime: p.pickupTime,
      mealPeriod: p.mealPeriod, pickupPoint: p.pickupPoint,
      paidAt: p.paidAt, txnId: p.txnId,
      subtotal: p.subtotal, discountRate: p.discountRate, discountCut: p.discountCut,
      total: p.amount, isStaff: p.isStaff,
      contact: p.contact, phone: p.phone, orderNote: p.orderNote || '',
      items: clone(p.items),
      rebuiltBy: (currentAccount() || {}).name || '',
      rebuiltAt: stamp(),
    };
    g().aOrders.unshift(order);
    dropPending(id);                            // 条目已了结，不能再被处理第二次
    return ok(clone(order));
  }

  // §7.8：4 位数字，按取餐日期从 0001 累计，当日唯一
  function nextPickupCode(day) {
    const used = g().aOrders.filter(o => o.pickupDate === day).map(o => Number(o.code));
    const next = (used.length ? Math.max.apply(null, used) : 0) + 1;
    return String(next).padStart(4, '0');
  }

  // §7.4：取餐前 30 分钟自动开做；补建时若已不足 30 分钟，直接进制作中
  function scheduleState(p) {
    const mins = t => Number(t.slice(0, 2)) * 60 + Number(t.slice(3, 5));
    if (p.pickupDate > BUSINESS_DAY) return '已预约';
    return mins(p.pickupTime) - mins(nowHm()) <= 30 ? '制作中' : '已预约';
  }
  function nowHm() {
    const d = new Date(), pad = n => String(n).padStart(2, '0');
    return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  /* POST /admin/pending-payments/:id/refund  退款作废
     同 §7.7：只有全额，没有金额入参；只到退款中，等微信确认。 */
  function refundPendingPayment(id, reason) {
    const p = findPending(id);
    if (!p) return fail('该条目已被处理');
    const why = String(reason == null ? '' : reason).trim();
    if (!why) return fail('请填写作废原因。该笔已收款，退款不可撤销，原因会记入财务对账。');
    const me = currentAccount();
    const voided = {
      ...clone(p),
      voided: true,
      refund: {
        no: refundNo(),
        amount: p.amount,
        status: '退款中',
        operator: me ? me.name : '未知操作人',
        at: stamp(),
        reason: why,
      },
    };
    g().voidedPending = g().voidedPending || [];
    g().voidedPending.push(voided);
    dropPending(id);
    return ok(clone(voided));
  }

  /* 明细导出。CSV 而非 .xlsx：导出是给人拿去和微信账单比对的，纯文本谁都能开，
     而我们没有生成 .xlsx 的能力（xlsx.js 只读不写）。
     两处 Excel 的坑必须防：
     - 无 BOM 的 UTF-8 中文在 Excel 里是乱码；
     - 微信交易号是 20 位数字，Excel 会转成科学计数法。用制表符前缀钉成文本。 */
  function csvCell(v) {
    const s = String(v == null ? '' : v);
    return /[",\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
  }
  const asText = v => '="' + String(v) + '"';

  function _export(range) {
    const head = ['订单号', '取餐号', '支付时间', '微信交易号', '原价小计', '折扣率', '折扣减免',
                  '实付金额', '员工价', '营业日期', '取餐时间', '餐段', '联系人', '手机号',
                  '退款单号', '退款金额', '退款状态', '操作人', '退款时间'];
    const rows = _payments(range).map(o => [
      o.no, o.code, o.paidAt, asText(o.txnId), yuan(o.subtotal), o.discountRate + '%', yuan(o.discountCut),
      yuan(o.total), o.isStaff ? '是' : '否', o.pickupDate, o.pickupTime,
      o.mealPeriod === 'lunch' ? '午餐' : '晚餐', o.contact, o.phone,
      o.refund ? asText(o.refund.no) : '', o.refund ? yuan(o.refund.amount) : '',
      o.refund ? o.refund.status : '', o.refund ? o.refund.operator : '', o.refund ? o.refund.at : '',
    ]);
    const body = [head, ...rows].map(r => r.map(csvCell).join(',')).join('\r\n');
    return '﻿' + body + '\r\n';
  }

  // GET /admin/finance/payments?from=&to= → Payment[]（按支付时间倒序）
  function listPayments(range) { return ok(_payments(range)); }
  // GET /admin/finance/refunds?from=&to= → Refund[]（按退款时间倒序）
  function listRefunds(range) { return ok(_refunds(range)); }
  // GET /admin/finance/summary?from=&to= → 对账汇总。金额一律整数分。
  function financeSummary(range) { return ok(_summary(range)); }
  // GET /admin/finance/export?from=&to= → CSV 文本
  function buildPaymentExport(range) { return ok(_export(range)); }

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



  /* ---------------- 商户账号名单（PRD §4.4） ----------------
     与员工折扣白名单分开管理。核心不变量：名单中至少保留一个启用的主账号——
     否则没有人能登录 PC 后台，且无法从界面恢复。删除、停用、降级三条路径都要挡。 */

  const ROLES = ['owner', 'staff'];
  const ROLE_LABEL = { owner: '主账号', staff: '子账号' };

  function enabledOwners(exceptId) {
    return g().accounts.filter(a => a.role === 'owner' && a.enabled && a.id !== exceptId);
  }
  function guardLastOwner(id, action) {
    const target = g().accounts.find(a => a.id === id);
    if (!target || target.role !== 'owner' || !target.enabled) return null;
    if (enabledOwners(id).length > 0) return null;
    return fail(`「${target.name}」是唯一启用的主账号，不能${action}。请先添加并启用另一个主账号。`);
  }

  // GET /admin/merchant-accounts → { list: MerchantAccount[] }
  function listMerchantAccounts(kw) {
    const q = (kw || '').trim();
    const all = clone(g().accounts);
    if (!q) return ok(all);
    return ok(all.filter(a => a.phone.includes(q) || a.name.includes(q)));
  }

  // POST /admin/merchant-accounts       新增（无 id）
  // PUT  /admin/merchant-accounts/:id   修改
  // body: { phone, name, role }
  function saveMerchantAccount(a) {
    const s = g();
    const phone = normPhone(a.phone);
    if (!phone) return fail('手机号必填');
    if (!PHONE_RE.test(phone)) return fail('手机号格式不正确');
    const name = String(a.name == null ? '' : a.name).trim();
    if (!name) return fail('姓名必填');
    if (!ROLES.includes(a.role)) return fail('请选择角色');
    const dup = s.accounts.find(x => x.phone === phone && x.id !== a.id);
    if (dup) return fail(`手机号 ${phone} 已在商户名单中（${dup.name} · ${ROLE_LABEL[dup.role]}）`);

    if (a.id) {
      const i = s.accounts.findIndex(x => x.id === a.id);
      if (i < 0) return fail('账号不存在');
      if (a.role !== 'owner') {
        const blocked = guardLastOwner(a.id, '降级为子账号');
        if (blocked) return blocked;
      }
      // 微信绑定关系由小程序侧建立，PC 编辑不得覆盖
      s.accounts[i] = Object.assign({}, s.accounts[i], { phone, name, role: a.role });
      return ok(clone(s.accounts[i]));
    }
    const created = { id: uid('ma'), phone, name, role: a.role, enabled: true, boundOpenId: '' };
    s.accounts.push(created);
    return ok(clone(created));
  }

  // PUT /admin/merchant-accounts/:id/enabled  body: { enabled }
  function setMerchantAccountEnabled(id, enabled) {
    const a = g().accounts.find(x => x.id === id);
    if (!a) return fail('账号不存在');
    if (!enabled) {
      const blocked = guardLastOwner(id, '停用');
      if (blocked) return blocked;
    }
    a.enabled = !!enabled;
    return ok(clone(a));
  }

  // DELETE /admin/merchant-accounts/:id
  function deleteMerchantAccount(id) {
    const s = g();
    const blocked = guardLastOwner(id, '删除');
    if (blocked) return blocked;
    const i = s.accounts.findIndex(x => x.id === id);
    if (i < 0) return fail('账号不存在');
    s.accounts.splice(i, 1);
    return ok({});
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
    listOrders, laneCounts, findOrder, findOrderByCode, itemsSummary, yuan,
    today, currentAccount, refundOrder, canRefund, uncollectedCount,
    searchOrders, codeHint,
    listPendingPayments, pendingPaymentCount, rebuildOrder, refundPendingPayment, blockingReason,
    listPayments, listRefunds, financeSummary, buildPaymentExport,
    advanceOrder, advanceMeta, statusTone, NEXT, ACT, LANES,
    getSettings, saveSettings, setStoreStatus,
    getLayer, saveLayer, clearLayer,
    imgUrl, MEALS, MEAL_LABEL,
    listStaff, saveStaff, setStaffEnabled, deleteStaff, getDiscountRate, saveDiscountRate,
    previewProductImport, commitProductImport, previewStaffImport, commitStaffImport, MAX_IMPORT_ROWS,
    listMerchantAccounts, saveMerchantAccount, setMerchantAccountEnabled, deleteMerchantAccount, ROLES, ROLE_LABEL,
  };
})();
