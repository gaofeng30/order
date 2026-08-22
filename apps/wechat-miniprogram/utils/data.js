/* 绥安食品 — 菜品 / 订单 / 分类 / 预约 数据 (真实物料 p001–p007) */

/* 单取餐点（§3.1「一期为单门店单取餐点」；§13.3 已由客户确认）。
   不拆名称与地址：客户只给了一个字符串，拆成两个字段会让不同写入路径
   各挑一个，装入语义不同的值。 */
const PICKUP_POINT = '党政办公中心后院老食堂北门';

const STORE = {
  name: '绥安食品',
  addr: '党政办公中心后院老食堂',
  cutoff: '今日 16:30 截单',
  pickup: '预计 17:10 可取',
  status: '营业中',
  // 门店公告，商户在 PC 营业设置里维护（§5.1、§6.9）
  notice: '今日卤味新鲜出锅，欢迎到店自提～',
};

// 占位兜底渐变色 (按品类, 真实图缺失时使用)
const HUES = {
  今日套餐: ['#3f6e8c', '#264a63'], 热销菜品: ['#b07a3c', '#7d5325'],
  轻食低脂: ['#5b8f3f', '#3d6b2f'], 汤饮甜品: ['#4f8aa8', '#356b86'],
};

const CATS = ['今日套餐', '热销菜品', '轻食低脂', '汤饮甜品'];

const MENU = [
  { id: 'p001', name: '商务双拼饭', cat: '今日套餐', price: 32, img: '/assets/dishes/p001.jpg',
    desc: '黑椒牛肉配低温鸡胸，适合午间快速取餐。双主菜、时蔬、米饭和例汤组合，兼顾饱腹与清爽口感。',
    meal: 'all', status: 'on',
    specs: [['规格', '标准份 / 少饭 / 加饭'], ['含汤', '例汤一份'], ['建议', '现取现食 · 当日食用']] },
  { id: 'p002', name: '江南三鲜套餐', cat: '今日套餐', price: 28, img: '/assets/dishes/p002.jpg',
    desc: '虾仁、菌菇与时蔬组合，口味清淡。适合会议日常餐，搭配清炒时蔬和紫菜蛋花汤。',
    meal: 'lunch', status: 'on',
    specs: [['规格', '标准份 / 少盐'], ['含汤', '紫菜蛋花汤'], ['建议', '口味清淡 · 老少咸宜']] },
  { id: 'p003', name: '招牌红烧牛腩', cat: '热销菜品', price: 36, img: '/assets/dishes/p003.jpg',
    desc: '慢炖牛腩，适合搭配米饭。红烧汁浓郁但不过甜，当前午市已售罄。',
    meal: 'all', status: 'on',
    specs: [['规格', '标准份'], ['建议', '加热后食用']] },
  { id: 'p004', name: '蒜香鸡腿排', cat: '热销菜品', price: 26, img: '/assets/dishes/p004.jpg',
    desc: '高频复购单品，窗口取餐快。去骨鸡腿排配蒜香酱汁，适合加班餐。',
    meal: 'all', status: 'on',
    specs: [['规格', '标准份 / 双拼'], ['酱汁', '蒜香酱'], ['建议', '趁热食用']] },
  { id: 'p005', name: '藜麦鸡胸能量碗', cat: '轻食低脂', price: 30, img: '/assets/dishes/p005.jpg',
    desc: '低脂高蛋白，配油醋汁。鸡胸、藜麦、牛油果和季节蔬菜，适合轻食需求。',
    meal: 'lunch', status: 'on',
    specs: [['规格', '标准份 / 酱汁分装'], ['酱汁', '油醋汁'], ['建议', '低脂高蛋白']] },
  { id: 'p006', name: '山药排骨汤', cat: '汤饮甜品', price: 12, img: '/assets/dishes/p006.jpg',
    desc: '温热汤品，午市限量。清炖排骨汤，适合搭配套餐。',
    meal: 'all', status: 'on',
    specs: [['规格', '小份 / 大份'], ['建议', '温热饮用 · 限量供应']] },
  { id: 'p007', name: '鲜橙气泡水', cat: '汤饮甜品', price: 10, img: '/assets/dishes/p007.jpg',
    desc: '冷饮，当前暂未上架。外卖和冷链规则未确认，暂不开放购买。',
    meal: 'dinner', status: 'off',
    specs: [['规格', '标准杯'], ['温度', '冰镇']] },
];

/* 运行时菜品表：商户端可编辑，改动写入 globalData.menu；
   未初始化（如单元测试直接 require）时回落到上面的种子常量。
   页面一律走 menuList()/itemById()，不再直接引用 MENU 常量。 */
function menuList() {
  try {
    const g = getApp().globalData;
    if (g && g.menu && g.menu.length) return g.menu;
  } catch (e) { /* App 尚未实例化 */ }
  return MENU;
}
const itemById = id => menuList().find(m => m.id === id);

/* ---- 当日售罄（§6.5、§15.6.1）----
   售罄不落在 Product 上，而是按取餐日期的独立记录，唯一键为
   (productId, serviceDate)。与后端 product_sold_out_dates 同形。

   只存记录的「有无」，不存布尔：次日清零时，昨天的 false 与今天的
   「还没标过」会成为两种形态表示同一件事，判断就得处理两遍。
   存在性只有一种形态，且天然满足「未来的日期默认可售」。

   serviceDate 是取餐日期而非操作日期 —— §6.5：商户在营业日 D 标记售罄，
   只屏蔽 D 当天的下单，不影响 D+1 的预约。 */
const PRODUCT_SOLD_OUT_DATES = [
  { productId: 'p003', serviceDate: '2026-08-21' },   // 今日售罄
  { productId: 'p006', serviceDate: '2026-08-20' },   // 昨日售罄，今日已自然清零
];

function soldOutList() {
  try {
    const g = getApp().globalData;
    if (g && g.soldOut) return g.soldOut;
  } catch (e) { /* App 尚未实例化 */ }
  return PRODUCT_SOLD_OUT_DATES;
}

// 该商品在该取餐日期是否售罄
function isSoldOut(productId, serviceDate) {
  return soldOutList().some(r => r.productId === productId && r.serviceDate === serviceDate);
}

// 可售 = 上架 且 该取餐日期无售罄记录。两个维度分别判断，不合成第三个枚举。
function isSellable(product, serviceDate) {
  const m = typeof product === 'string' ? itemById(product) : product;
  return !!m && m.status === 'on' && !isSoldOut(m.id, serviceDate);
}

// 用户端 我的订单
/* items: [id, name, qty, price, discountedPrice, flavors?, note?]
   name 是下单当刻固化的名称快照，不是指向菜品表的外键（§15.6.2）：
   订单是历史记录，商品改名或删除后它必须仍复述下单当时的事实。
   price / discountedPrice 为整数分（§5.6：金额一律以整数分保存与计算）。
   口味与备注绑定在行内；整单级只有 orderNote，没有整单级口味（§15.6.4）。
   一期身份识别链路未接后端，所有人按访客原价，折后价等于原价。 */
const USER_ORDERS = [
  { id: 'o1', no: 'SA2406100126', code: '0126', status: '待取餐',
    pickupDate: '2026-08-21', pickupTime: '17:00', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:42:00', subtotal: 7600, discountRate: 100, discountCut: 0, total: 7600, isStaff: false,
    orderNote: '双拼饭加饭', contact: '林先生', phone: '138****6620',
    items: [['p001', '商务双拼饭', 2, 3200, 3200, ['加饭', '加辣'], '双拼饭加饭'], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'r1', no: 'SA2406100140', code: '0208', status: '已预约',
    pickupDate: '2026-08-21', pickupTime: '18:30', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:30:00', subtotal: 6000, discountRate: 100, discountCut: 0, total: 6000, isStaff: false,
    orderNote: '会议餐，准点取', contact: '林先生', phone: '138****6620',
    items: [['p001', '商务双拼饭', 1, 3200, 3200], ['p002', '江南三鲜套餐', 1, 2800, 2800, ['少盐'], '会议餐，准点取']] },
  { id: 'r2', no: 'SA2406100138', code: '0176', status: '已预约',
    pickupDate: '2026-08-21', pickupTime: '17:06', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:28:00', subtotal: 4200, discountRate: 100, discountCut: 0, total: 4200, isStaff: false,
    orderNote: '', contact: '林先生', phone: '138****6620',
    items: [['p005', '藜麦鸡胸能量碗', 1, 3000, 3000, ['酱汁分装']], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'o3', no: 'SA2406090311', code: '0311', status: '已完成',
    pickupDate: '2026-08-20', pickupTime: '12:30', mealPeriod: 'lunch', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-20 12:08:00', subtotal: 6800, discountRate: 100, discountCut: 0, total: 6800, isStaff: false,
    orderNote: '鸡腿排双拼', contact: '林先生', phone: '138****6620',
    items: [['p004', '蒜香鸡腿排', 1, 2600, 2600, ['双拼']], ['p005', '藜麦鸡胸能量碗', 1, 3000, 3000, ['酱汁分装']], ['p006', '山药排骨汤', 1, 1200, 1200]] },
];

/* 当前营业日。§7.8：取餐号按取餐日期从 0001 累计，跨营业日可能重复，
   因此按号定位必须限定营业日。取值与 PC 后台 data/api.js 的 BUSINESS_DAY 一致；
   后端就位后由服务端下发，替换的是取值来源而非消费方。 */
const BUSINESS_DAY = '2026-08-21';

// 商户端 订单 (六态履约模型: 已预约 → 制作中 → 待取餐 → 已完成；旁路 退款中 → 已退款)
const ADMIN_ORDERS = [
  { id: 'a0', no: 'SA2406100145', code: '0145', status: '已预约',
    pickupDate: '2026-08-21', pickupTime: '18:00', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:55:00', subtotal: 5800, discountRate: 100, discountCut: 0, total: 5800, isStaff: false,
    orderNote: '预约 18:00 取', contact: '孙女士', phone: '150****3322',
    items: [['p002', '江南三鲜套餐', 1, 2800, 2800, '少盐'], ['p005', '藜麦鸡胸能量碗', 1, 3000, 3000]] },
  { id: 'a1', no: 'SA2406100131', code: '0131', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:20', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:51:00', subtotal: 6000, discountRate: 100, discountCut: 0, total: 6000, isStaff: false,
    orderNote: '打包分开装', contact: '陈女士', phone: '159****2031',
    items: [['p001', '商务双拼饭', 1, 3200, 3200, '双拼饭加辣 ×1'], ['p002', '江南三鲜套餐', 1, 2800, 2800]] },
  { id: 'a2', no: 'SA2406100129', code: '0129', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:20', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:49:00', subtotal: 2600, discountRate: 100, discountCut: 0, total: 2600, isStaff: false,
    orderNote: '', contact: '吴先生', phone: '137****7788',
    items: [['p004', '蒜香鸡腿排', 1, 2600, 2600]] },
  { id: 'a3', no: 'SA2406100126', code: '0126', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:10', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:42:00', subtotal: 7600, discountRate: 100, discountCut: 0, total: 7600, isStaff: false,
    orderNote: '双拼饭加饭', contact: '林先生', phone: '138****6620',
    items: [['p001', '商务双拼饭', 2, 3200, 3200, '加饭 · 加辣'], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'a4', no: 'SA2406100120', code: '0120', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:10', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:35:00', subtotal: 6800, discountRate: 100, discountCut: 0, total: 6800, isStaff: false,
    orderNote: '能量碗酱汁分装', contact: '黄小姐', phone: '135****9012',
    items: [['p005', '藜麦鸡胸能量碗', 1, 3000, 3000, '酱汁分装'], ['p004', '蒜香鸡腿排', 1, 2600, 2600], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'a5', no: 'SA2406100118', code: '0118', status: '待取餐',
    pickupDate: '2026-08-21', pickupTime: '17:00', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:30:00', subtotal: 6800, discountRate: 100, discountCut: 0, total: 6800, isStaff: false,
    orderNote: '', contact: '郑先生', phone: '133****4456',
    items: [['p002', '江南三鲜套餐', 2, 2800, 2800, '少盐'], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'a6', no: 'SA2406100112', code: '0112', status: '待取餐',
    pickupDate: '2026-08-21', pickupTime: '17:00', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 16:22:00', subtotal: 3800, discountRate: 100, discountCut: 0, total: 3800, isStaff: false,
    orderNote: '', contact: '王女士', phone: '188****0021',
    items: [['p004', '蒜香鸡腿排', 1, 2600, 2600], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  { id: 'a7', no: 'SA2406100090', code: '0090', status: '已完成',
    pickupDate: '2026-08-21', pickupTime: '13:00', mealPeriod: 'lunch', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 12:35:00', subtotal: 6200, discountRate: 100, discountCut: 0, total: 6200, isStaff: false,
    orderNote: '', contact: '刘先生', phone: '130****5567',
    items: [['p001', '商务双拼饭', 1, 3200, 3200, '加饭'], ['p005', '藜麦鸡胸能量碗', 1, 3000, 3000]] },
  { id: 'a8', no: 'SA2406100071', code: '0071', status: '已退款',
    pickupDate: '2026-08-21', pickupTime: '12:30', mealPeriod: 'lunch', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 11:55:00', subtotal: 2600, discountRate: 100, discountCut: 0, total: 2600, isStaff: false,
    orderNote: '用户取消', contact: '孙女士', phone: '150****3322',
    items: [['p004', '蒜香鸡腿排', 1, 2600, 2600]] },
  /* a9 与 a5 取餐号相同、取餐日期不同 —— 没有这一组，「限定当前营业日」不可证伪。
     它同时是 §6.7 的「未取餐」样本：营业日结束后仍停在 待取餐。 */
  { id: 'a9', no: 'SA2406090118', code: '0118', status: '待取餐',
    pickupDate: '2026-08-20', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-20 16:30:00', subtotal: 3200, discountRate: 100, discountCut: 0, total: 3200, isStaff: false,
    orderNote: '昨日未取', contact: '许先生', phone: '186****7742',
    items: [['p001', '商务双拼饭', 1, 3200, 3200]] },
  // a10 的取餐号只存在于旧营业日：用于验证跨日提示，而不是空列表。
  { id: 'a10', no: 'SA2406090203', code: '0203', status: '已完成',
    pickupDate: '2026-08-20', pickupTime: '12:30', mealPeriod: 'lunch', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-20 12:14:00', subtotal: 4000, discountRate: 100, discountCut: 0, total: 4000, isStaff: false,
    orderNote: '', contact: '钱女士', phone: '159****8830',
    items: [['p002', '江南三鲜套餐', 1, 2800, 2800, '少盐'], ['p006', '山药排骨汤', 1, 1200, 1200]] },
  /* §7.1 允许订单长期停在 退款中（退款结果无法确定时即停在此处）。
     §6.6 的泳道只有五档、不含 退款中，因此该状态只能经「全部」泳道或搜索找到。 */
  { id: 'a11', no: 'SA2406100102', code: '0102', status: '退款中',
    pickupDate: '2026-08-21', pickupTime: '18:00', mealPeriod: 'dinner', pickupPoint: PICKUP_POINT,
    paidAt: '2026-08-21 15:12:00', subtotal: 3000, discountRate: 100, discountCut: 0, total: 3000, isStaff: false,
    orderNote: '商户取消，退款处理中', contact: '曹先生', phone: '187****2214',
    items: [['p005', '藜麦鸡胸能量碗', 1, 3000, 3000]] },
];


// 分类管理
const ADMIN_CATS = [
  { id: 'c1', name: '今日套餐', sort: 1, on: true, count: 2 },
  { id: 'c2', name: '热销菜品', sort: 2, on: true, count: 2 },
  { id: 'c3', name: '轻食低脂', sort: 3, on: true, count: 1 },
  { id: 'c4', name: '汤饮甜品', sort: 4, on: true, count: 2 },
  { id: 'c5', name: '节庆礼盒', sort: 5, on: false, count: 0 },
];

// 口味选项 (菜品级, 可多选)
const FLAVORS = ['少饭', '加饭', '少盐', '加辣', '酱汁分装', '免葱蒜', '打包分装', '多双餐具'];

/* 预约取餐（生效 spec: Every first-phase order uses one discrete pickup time）
   - 仅预约取餐，可预约今天与明天
   - 每个餐段一个固定截单时刻，餐段内全部取餐时间共用，不随取餐时间滚动
   - 取餐时间为餐段范围内的离散时间点，粒度由 PICKUP_STEP_MIN 决定（商户可配）
   - 取餐时间是约定时刻，不是必须到场的窗口：备好后推送提醒，凭提醒随时取
   NOW_MINS 为演示时钟（16:48），真实实现由服务端按门店时区计算。 */
const NOW_MINS = 16 * 60 + 48;

const PICKUP_POINTS = [{ name: PICKUP_POINT }];

const RESERVE_DATES = [{ k: '今天', off: 0 }, { k: '明天', off: 1 }];

// 商户在 PC 后台配置：截单时刻、取餐时间范围、粒度
const PICKUP_STEP_MIN = 30;
const MEAL_PERIODS = [
  { key: 'lunch', name: '午餐', cutoff: '11:30', from: '11:30', to: '13:30' },
  { key: 'dinner', name: '晚餐', cutoff: '17:00', from: '17:00', to: '19:00' },
];

const toMins = t => { const [h, m] = t.split(':').map(Number); return h * 60 + m; };
const toText = m => String(Math.floor(m / 60)).padStart(2, '0') + ':' + String(m % 60).padStart(2, '0');
const periodBy = key => MEAL_PERIODS.find(p => p.key === key);

// 取餐时间点：由范围与粒度推导，不写死
function pickupTimes(periodKey) {
  const p = periodBy(periodKey);
  if (!p) return [];
  const out = [];
  for (let m = toMins(p.from); m <= toMins(p.to); m += PICKUP_STEP_MIN) out.push(toText(m));
  return out;
}

// 该营业日期的该餐段是否已截单。只有「今天」会被当前时刻截掉。
function isPeriodCutOff(off, periodKey) {
  const p = periodBy(periodKey);
  if (!p) return true;
  return off === 0 && NOW_MINS >= toMins(p.cutoff);
}

// 某营业日期是否两个餐段都已截单
function isDateCutOff(off) { return MEAL_PERIODS.every(p => isPeriodCutOff(off, p.key)); }

// 默认选中：当前时刻之后第一个未截单的取餐时间点
function defaultPickup() {
  for (const d of RESERVE_DATES) {
    for (const p of MEAL_PERIODS) {
      if (isPeriodCutOff(d.off, p.key)) continue;
      const times = pickupTimes(p.key);
      const pick = d.off === 0 ? times.find(t => toMins(t) >= NOW_MINS) || times[0] : times[0];
      return { off: d.off, period: p.key, time: pick };
    }
  }
  const last = RESERVE_DATES[RESERVE_DATES.length - 1];
  return { off: last.off, period: MEAL_PERIODS[0].key, time: pickupTimes(MEAL_PERIODS[0].key)[0] };
}

/* ---- 纯字符串日历运算 ----
   刻意不使用 Date：营业日必须是数据里的事实，不能由运行时时钟推导，
   否则同一份种子数据的断言结果会随日期翻转。
   算法为 days-from-civil / civil-from-days，输入输出都是 YYYY-MM-DD。 */
function daysFromCivil(iso) {
  const [y0, m, d] = iso.split('-').map(Number);
  const y = y0 - (m <= 2 ? 1 : 0);
  const era = Math.floor(y / 400);
  const yoe = y - era * 400;
  const doy = Math.floor((153 * (m + (m > 2 ? -3 : 9)) + 2) / 5) + d - 1;
  const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
  return era * 146097 + doe - 719468;
}
function civilFromDays(z0) {
  const z = z0 + 719468;
  const era = Math.floor(z / 146097);
  const doe = z - era * 146097;
  const yoe = Math.floor((doe - Math.floor(doe / 1460) + Math.floor(doe / 36524) - Math.floor(doe / 146096)) / 365);
  const doy = doe - (365 * yoe + Math.floor(yoe / 4) - Math.floor(yoe / 100));
  const mp = Math.floor((5 * doy + 2) / 153);
  const d = doy - Math.floor((153 * mp + 2) / 5) + 1;
  const m = mp + (mp < 10 ? 3 : -9);
  const y = era * 400 + yoe + (m <= 2 ? 1 : 0);
  return `${y}-${String(m).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
}
// 相对当前营业日的天数
const dayOffsetOf = iso => daysFromCivil(iso) - daysFromCivil(BUSINESS_DAY);

const dateLabel = off => (RESERVE_DATES.find(d => d.off === off) || RESERVE_DATES[0]).k;

/* 取餐选择 {off,time} → 绝对营业日期。off 是相对当前营业日的天数。 */
function pickupDateOf(pk) {
  return civilFromDays(daysFromCivil(BUSINESS_DAY) + (pk && pk.off ? pk.off : 0));
}

/* 下单时刻。演示时钟固定为 NOW_MINS，真实实现由服务端按门店时区给出。 */
function nowStamp() {
  const hh = String(Math.floor(NOW_MINS / 60)).padStart(2, '0');
  const mm = String(NOW_MINS % 60).padStart(2, '0');
  return `${BUSINESS_DAY} ${hh}:${mm}:00`;
}
const pickupLabel = pk => `${dateLabel(pk.off)} ${pk.time}`;

// 距取餐分钟数
function slotMins(off, t) { return off * 1440 + toMins(t) - NOW_MINS; }

// 取餐前可取消的最短分钟数
const CANCEL_LIMIT_MIN = 30;

/* 距取餐的剩余分钟数，从订单的取餐日期与时刻现算（§15.6.2 删除了 minsToPickup）。
   剩余时间是随时间变化的量，落在记录上就会陈旧：时钟一旦真实流动，
   取消窗口会按下单时刻而不是当前时刻判定，放行本该拒绝的取消。 */
function minsToPickup(o) {
  if (!o || !o.pickupDate || !o.pickupTime) return NaN;
  return dayOffsetOf(o.pickupDate) * 1440 + toMins(o.pickupTime) - NOW_MINS;
}

// 可自助取消：`已预约` 且距取餐大于 30 分钟（生效 spec §7.6）。
// 订单类型字段已随即时单删除，不再参与判定。
function canCancelReserve(o) {
  return !!(o && o.status === '已预约' && minsToPickup(o) > CANCEL_LIMIT_MIN);
}

/* 取餐文案，同样现算（§15.6.2 删除了 pickupLabel 字段）。 */
function orderPickupLabel(o) {
  if (!o || !o.pickupDate || !o.pickupTime) return '';
  const off = dayOffsetOf(o.pickupDate);
  const day = off === 0 ? '今天' : off === 1 ? '明天' : off === -1 ? '昨天' : o.pickupDate;
  return day + ' ' + o.pickupTime;
}

// 当前登录用户（微信授权手机号；真实实现由服务端用 code 换取，前端拿不到明文）
const ME = { phone: '13800006620', nick: '林先生', avatarChar: '林' };

module.exports = {
  STORE, HUES, CATS, MENU, menuList, itemById, USER_ORDERS, ADMIN_ORDERS, ADMIN_CATS, FLAVORS,
  PRODUCT_SOLD_OUT_DATES, soldOutList, isSoldOut, isSellable,
  BUSINESS_DAY,
  NOW_MINS, PICKUP_POINT, PICKUP_POINTS, RESERVE_DATES, MEAL_PERIODS, PICKUP_STEP_MIN,
  pickupTimes, isPeriodCutOff, isDateCutOff, defaultPickup, dateLabel, pickupLabel,
  CANCEL_LIMIT_MIN, slotMins, canCancelReserve, minsToPickup, orderPickupLabel,
  pickupDateOf, nowStamp,
  ME,
};
