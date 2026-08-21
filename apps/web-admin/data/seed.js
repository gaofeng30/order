/* 绥安食品 — 菜品 / 订单 / 分类 / 预约 数据 (真实物料 p001–p007)
   ------------------------------------------------------------
   本文件由 apps/wechat-miniprogram/utils/data.js 拷贝而来，仅改动两处：
   1. 结尾 module.exports → window.Seed（浏览器无模块系统）
   2. menuList() 的数据源由 getApp().globalData 改为 window.__store
   种子内容与小程序端保持一致，改业务规则时两处需同步。
   ------------------------------------------------------------ */

const STORE = {
  name: '绥安食品',
  branch: '县前直营店',
  addr: '绥芬河市青云镇通商路',
  cutoff: '今日 16:30 截单',
  pickup: '预计 17:10 可取',
  pickupWindow: '县前大厦 1F · 2 号取餐窗口',
  status: '营业中',
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
    meal: 'all', status: 'soldout',
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

/* 运行时菜品表：商户端可编辑，改动写入 window.__store.menu；
   未初始化时回落到上面的种子常量。页面一律走 menuList()/itemById()。 */
function menuList() {
  const s = window.__store;
  if (s && s.menu && s.menu.length) return s.menu;
  return MENU;
}
const itemById = id => menuList().find(m => m.id === id);

// 商户端 订单 (六态履约模型: 已预约 → 制作中 → 待取餐 → 已完成；旁路 退款中 → 已退款)
/* 订单（PRD §15.6.2）。金额一律为整数分：财务与对账页要按分核对微信账单，
   任何浮点或元为单位的中间态都会在求和时产生一分钱的差。
   items 行为 [菜品 id, 数量, 原价单价(分), 折后单价(分), 口味?, 备注?]，
   口味与备注绑定在行内，整单级只有 orderNote。
   员工折扣按 §6.6 逐商品先舍入到分再乘数量，因此 discountCut 恒等于逐行差之和。 */
const ADMIN_ORDERS = [
  { id: 'a0', no: 'SA2406100145', code: '0145', status: '已预约',
    pickupDate: '2026-08-21', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:55:12', txnId: '4200002318202608210001',
    subtotal: 5800, discountRate: 100, discountCut: 0, total: 5800, isStaff: false,
    contact: '孙女士', phone: '150****3322', orderNote: '预约 18:00 取',
    items: [['p002', 1, 2800, 2800, '少盐', ''], ['p005', 1, 3000, 3000, '', '']] },

  { id: 'a1', no: 'SA2406100131', code: '0131', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:51:40', txnId: '4200002318202608210002',
    subtotal: 6000, discountRate: 85, discountCut: 900, total: 5100, isStaff: true,
    contact: '陈女士', phone: '159****2031', orderNote: '打包分开装',
    items: [['p001', 1, 3200, 2720, '加辣', ''], ['p002', 1, 2800, 2380, '', '']] },

  { id: 'a2', no: 'SA2406100129', code: '0129', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:49:03', txnId: '4200002318202608210003',
    subtotal: 2600, discountRate: 100, discountCut: 0, total: 2600, isStaff: false,
    contact: '吴先生', phone: '137****7788', orderNote: '',
    items: [['p004', 1, 2600, 2600, '', '']] },

  { id: 'a3', no: 'SA2406100126', code: '0126', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '18:00', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:42:18', txnId: '4200002318202608210004',
    subtotal: 7600, discountRate: 100, discountCut: 0, total: 7600, isStaff: false,
    contact: '林先生', phone: '138****6620', orderNote: '双拼饭加饭',
    items: [['p001', 2, 3200, 3200, '加饭 · 加辣', ''], ['p006', 1, 1200, 1200, '', '']] },

  { id: 'a4', no: 'SA2406100120', code: '0120', status: '制作中',
    pickupDate: '2026-08-21', pickupTime: '18:00', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:35:55', txnId: '4200002318202608210005',
    subtotal: 6800, discountRate: 85, discountCut: 1020, total: 5780, isStaff: true,
    contact: '黄小姐', phone: '135****9012', orderNote: '',
    items: [['p005', 1, 3000, 2550, '酱汁分装', ''], ['p004', 1, 2600, 2210, '', ''], ['p006', 1, 1200, 1020, '', '']] },

  { id: 'a5', no: 'SA2406100118', code: '0118', status: '待取餐',
    pickupDate: '2026-08-21', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:30:07', txnId: '4200002318202608210006',
    subtotal: 6800, discountRate: 100, discountCut: 0, total: 6800, isStaff: false,
    contact: '郑先生', phone: '133****4456', orderNote: '',
    items: [['p002', 2, 2800, 2800, '少盐', ''], ['p006', 1, 1200, 1200, '', '']] },

  { id: 'a6', no: 'SA2406100112', code: '0112', status: '待取餐',
    pickupDate: '2026-08-21', pickupTime: '17:30', mealPeriod: 'dinner', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 16:22:31', txnId: '4200002318202608210007',
    subtotal: 3800, discountRate: 100, discountCut: 0, total: 3800, isStaff: false,
    contact: '王女士', phone: '188****0021', orderNote: '',
    items: [['p004', 1, 2600, 2600, '', ''], ['p006', 1, 1200, 1200, '', '']] },

  { id: 'a7', no: 'SA2406100090', code: '0090', status: '已完成',
    pickupDate: '2026-08-21', pickupTime: '12:00', mealPeriod: 'lunch', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 11:40:22', txnId: '4200002318202608210008',
    subtotal: 6200, discountRate: 100, discountCut: 0, total: 6200, isStaff: false,
    contact: '刘先生', phone: '130****5567', orderNote: '',
    items: [['p001', 1, 3200, 3200, '加饭', ''], ['p005', 1, 3000, 3000, '', '']] },

  /* 退款中：已受理未到账，财务页净额里必须已扣除，但退款状态仍需人工盯 */
  { id: 'a8', no: 'SA2406100085', code: '0085', status: '退款中',
    pickupDate: '2026-08-21', pickupTime: '12:00', mealPeriod: 'lunch', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 11:12:09', txnId: '4200002318202608210009',
    subtotal: 3600, discountRate: 100, discountCut: 0, total: 3600, isStaff: false,
    contact: '赵先生', phone: '186****7710', orderNote: '',
    items: [['p003', 1, 3600, 3600, '', '']],
    refund: { no: '50000123452026082100001', amount: 3600, status: '退款中',
              operator: '高特', at: '2026-08-21 11:58:40', reason: '菜品临时售罄，商户取消' } },

  /* 部分退款：退款金额小于订单实付，用于验证净额不是按订单数而是按金额算 */
  { id: 'a9', no: 'SA2406100071', code: '0071', status: '已退款',
    pickupDate: '2026-08-21', pickupTime: '12:00', mealPeriod: 'lunch', pickupPoint: '县前直营店',
    paidAt: '2026-08-21 10:55:14', txnId: '4200002318202608210010',
    subtotal: 3800, discountRate: 100, discountCut: 0, total: 3800, isStaff: false,
    contact: '孙女士', phone: '150****3322', orderNote: '',
    items: [['p004', 1, 2600, 2600, '', ''], ['p006', 1, 1200, 1200, '', '']],
    refund: { no: '50000123452026082100002', amount: 1200, status: '已退款',
              operator: '周敏', at: '2026-08-21 11:20:02', reason: '汤洒了，退单品' } },

  /* 昨天的单，用于验证财务页按营业日期筛选 */
  { id: 'a10', no: 'SA2406090210', code: '0210', status: '已完成',
    pickupDate: '2026-08-20', pickupTime: '12:30', mealPeriod: 'lunch', pickupPoint: '县前直营店',
    paidAt: '2026-08-20 11:48:37', txnId: '4200002318202608200011',
    subtotal: 6600, discountRate: 85, discountCut: 990, total: 5610, isStaff: true,
    contact: '周工', phone: '139****1188', orderNote: '',
    items: [['p003', 1, 3600, 3060, '', ''], ['p005', 1, 3000, 2550, '', '']] },
];

// 单品销量排行
const RANK = [
  { id: 'p003', sold: 320 }, { id: 'p001', sold: 286 }, { id: 'p004', sold: 254 }, { id: 'p002', sold: 198 }, { id: 'p005', sold: 142 },
];

// 分类管理
const ADMIN_CATS = [
  { id: 'c1', name: '今日套餐', sort: 1, on: true, count: 2 },
  { id: 'c2', name: '热销菜品', sort: 2, on: true, count: 2 },
  { id: 'c3', name: '轻食低脂', sort: 3, on: true, count: 1 },
  { id: 'c4', name: '汤饮甜品', sort: 4, on: true, count: 2 },
  { id: 'c5', name: '节庆礼盒', sort: 5, on: false, count: 0 },
];

// 取餐点（同 apps/wechat-miniprogram/utils/data.js 的 PICKUP_POINTS）
const PICKUP_POINTS = [
  { id: 'pp1', name: '县前直营店', addr: '绥芬河市青云镇通商路', tag: '直营', hours: '09:00–19:00' },
  { id: 'pp2', name: '绥芬河北站取餐点', addr: '绥芬河市站前广场东侧', tag: '取餐点', hours: '07:00–20:00' },
  { id: 'pp3', name: '青云镇综合市场点', addr: '绥芬河市青云镇市场街', tag: '合作点', hours: '08:00–18:30' },
];

/* 营业设置。每个餐段一个固定截单时刻，餐段内全部取餐时间共用；
   取餐时间由 from/to 与 pickupStepMin 推导为离散时间点（生效 spec §5.5、§6.9）。 */
const SETTINGS = {
  discountRate: 85,      // 员工实付百分比，整数 1-100；100 表示无折扣（PRD §6.4）
  pickupStepMin: 30,
  mealPeriods: [
    { key: 'lunch', name: '午餐', cutoff: '11:30', from: '11:30', to: '13:30' },
    { key: 'dinner', name: '晚餐', cutoff: '17:00', from: '17:00', to: '19:00' },
  ],
  pickupPoint: '县前直营店',
  notice: '今日卤味新鲜出锅，欢迎到店自提～',
};

// 开屏装饰图层（对应小程序 utils/layer.js DEFAULTS）
const LAYER_DEFAULTS = { v: 1, enabled: false, src: '', cx: 0.5, cy: 0.38, size: 0.35, ar: 1 };

// 当前登录用户（微信授权手机号；真实实现由服务端用 code 换取，前端拿不到明文）
const ME = { phone: '13800006620', nick: '林先生', avatarChar: '林' };

// 演示时钟：与小程序 NOW_MINS 同源，用于券有效期判定
const TODAY = '2026-08-07';

// 店长（PC 端顶栏账号展示；小程序端写在 admin-profile.wxml）
const MANAGER = { name: '高特', role: '店长' };

/* 员工折扣白名单（PRD §6.4）
   只有两个可填字段：手机号（唯一识别键）与姓名（附加手机号双要素的第二要素）。
   enabled 由页面开关切换，joinAt 自动，bound / spend / orders 只读由系统统计。 */
const STAFF_WHITELIST = [
  { id: 's1', phone: '13800006620', name: '林建国', enabled: true,  joinAt: '2026-03-12', bound: true,  spend: 1286, orders: 42 },
  { id: 's2', phone: '13500009012', name: '黄映雪', enabled: true,  joinAt: '2026-01-08', bound: true,  spend: 3140, orders: 96 },
  { id: 's3', phone: '15900002031', name: '陈少芬', enabled: true,  joinAt: '2026-04-02', bound: true,  spend: 468,  orders: 19 },
  { id: 's4', phone: '13700007788', name: '吴国强', enabled: true,  joinAt: '2026-04-02', bound: false, spend: 0,    orders: 0 },
  { id: 's5', phone: '13300004456', name: '郑文彬', enabled: true,  joinAt: '2026-02-20', bound: true,  spend: 902,  orders: 31 },
  { id: 's6', phone: '15000003322', name: '孙丽萍', enabled: false, joinAt: '2026-01-22', bound: true,  spend: 214,  orders: 9 },
];

/* 商户账号名单（PRD §4.4）—— 决定谁能进商户端与 PC 后台。
   与员工折扣白名单是两份互不影响的名单：这份管「能不能登录」，那份管「打不打折」。
   role: 'owner' 主账号（全部权限，可登录 PC）| 'staff' 子账号（仅小程序四屏）
   boundOpenId 由商户首次在小程序「商户登录」时绑定，PC 侧只读。 */
const MERCHANT_ACCOUNTS = [
  { id: 'ma1', phone: '13612340001', name: '高特',   role: 'owner', enabled: true,  boundOpenId: 'o_demo_owner_1' },
  { id: 'ma2', phone: '13612340002', name: '周敏',   role: 'owner', enabled: true,  boundOpenId: '' },
  { id: 'ma3', phone: '13612340003', name: '后厨老陈', role: 'staff', enabled: true,  boundOpenId: 'o_demo_staff_3' },
  { id: 'ma4', phone: '13612340004', name: '窗口小李', role: 'staff', enabled: true,  boundOpenId: '' },
  { id: 'ma5', phone: '13612340005', name: '临时工小王', role: 'staff', enabled: false, boundOpenId: '' },
];

window.Seed = {
  STORE, HUES, CATS, MENU, menuList, itemById, ADMIN_ORDERS, RANK, ADMIN_CATS, PICKUP_POINTS,
  SETTINGS, LAYER_DEFAULTS, STAFF_WHITELIST, MERCHANT_ACCOUNTS, ME, TODAY, MANAGER,
};
