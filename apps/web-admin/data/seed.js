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
const ADMIN_ORDERS = [
  { id: 'a0', no: 'SA2406100145', code: '0145', status: '已预约', time: '16:55', mins: 0, total: 58, count: 2,
    contact: '孙女士', phone: '150****3322', flavor: '少盐', note: '预约 18:00 取',
    items: [['p002', 1, 28], ['p005', 1, 30]] },
  { id: 'a1', no: 'SA2406100131', code: 'A131', status: '制作中', time: '16:51', mins: 1, total: 60, count: 2,
    contact: '陈女士', phone: '159****2031', flavor: '双拼饭加辣 ×1', note: '打包分开装',
    items: [['p001', 1, 32], ['p002', 1, 28]] },
  { id: 'a2', no: 'SA2406100129', code: 'A129', status: '制作中', time: '16:49', mins: 3, total: 26, count: 1,
    contact: '吴先生', phone: '137****7788', flavor: '—', note: '',
    items: [['p004', 1, 26]] },
  { id: 'a3', no: 'SA2406100126', code: 'A126', status: '制作中', time: '16:42', mins: 10, total: 76, count: 3,
    contact: '林先生', phone: '138****6620', flavor: '加饭 · 加辣', note: '双拼饭加饭',
    items: [['p001', 2, 32], ['p006', 1, 12]] },
  { id: 'a4', no: 'SA2406100120', code: 'A120', status: '制作中', time: '16:35', mins: 17, total: 58, count: 2,
    contact: '黄小姐', phone: '135****9012', flavor: '酱汁分装', note: '能量碗酱汁分装',
    items: [['p005', 1, 30], ['p004', 1, 26], ['p006', 1, 12]] },
  { id: 'a5', no: 'SA2406100118', code: 'A118', status: '待取餐', time: '16:30', mins: 22, total: 68, count: 3,
    contact: '郑先生', phone: '133****4456', flavor: '少盐', note: '',
    items: [['p002', 2, 28], ['p006', 1, 12]] },
  { id: 'a6', no: 'SA2406100112', code: 'A112', status: '待取餐', time: '16:22', mins: 30, total: 38, count: 2,
    contact: '王女士', phone: '188****0021', flavor: '—', note: '',
    items: [['p004', 1, 26], ['p006', 1, 12]] },
  { id: 'a7', no: 'SA2406100090', code: 'A090', status: '已完成', time: '15:40', mins: 0, total: 62, count: 2,
    contact: '刘先生', phone: '130****5567', flavor: '加饭', note: '',
    items: [['p001', 1, 32], ['p005', 1, 30]] },
  { id: 'a8', no: 'SA2406100071', code: 'A071', status: '已退款', time: '14:55', mins: 0, total: 26, count: 1,
    contact: '孙女士', phone: '150****3322', flavor: '—', note: '用户取消',
    items: [['p004', 1, 26]] },
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

window.Seed = {
  STORE, HUES, CATS, MENU, menuList, itemById, ADMIN_ORDERS, RANK, ADMIN_CATS, PICKUP_POINTS,
  SETTINGS, LAYER_DEFAULTS, STAFF_WHITELIST, ME, TODAY, MANAGER,
};
