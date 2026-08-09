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
  { id: 'p001', name: '商务双拼饭', cat: '今日套餐', price: 32, sold: 286, stock: 42, img: '/assets/dishes/p001.jpg',
    desc: '黑椒牛肉配低温鸡胸，适合午间快速取餐。双主菜、时蔬、米饭和例汤组合，兼顾饱腹与清爽口感。',
    tags: ['今日推荐', '热销'], status: 'on', allergens: ['牛肉', '乳制品'],
    specs: [['规格', '标准份 / 少饭 / 加饭'], ['含汤', '例汤一份'], ['建议', '现取现食 · 当日食用']] },
  { id: 'p002', name: '江南三鲜套餐', cat: '今日套餐', price: 28, sold: 198, stock: 18, img: '/assets/dishes/p002.jpg',
    desc: '虾仁、菌菇与时蔬组合，口味清淡。适合会议日常餐，搭配清炒时蔬和紫菜蛋花汤。',
    tags: ['清淡'], status: 'on', allergens: ['虾仁', '鸡蛋'],
    specs: [['规格', '标准份 / 少盐'], ['含汤', '紫菜蛋花汤'], ['建议', '口味清淡 · 老少咸宜']] },
  { id: 'p003', name: '招牌红烧牛腩', cat: '热销菜品', price: 36, sold: 320, stock: 0, img: '/assets/dishes/p003.jpg',
    desc: '慢炖牛腩，适合搭配米饭。红烧汁浓郁但不过甜，当前午市已售罄。',
    tags: ['售罄'], status: 'soldout', allergens: ['牛肉'],
    specs: [['规格', '标准份'], ['建议', '加热后食用']] },
  { id: 'p004', name: '蒜香鸡腿排', cat: '热销菜品', price: 26, sold: 254, stock: 35, img: '/assets/dishes/p004.jpg',
    desc: '高频复购单品，窗口取餐快。去骨鸡腿排配蒜香酱汁，适合加班餐。',
    tags: ['热销'], status: 'on', allergens: ['大豆', '小麦'],
    specs: [['规格', '标准份 / 双拼'], ['酱汁', '蒜香酱'], ['建议', '趁热食用']] },
  { id: 'p005', name: '藜麦鸡胸能量碗', cat: '轻食低脂', price: 30, sold: 142, stock: 22, img: '/assets/dishes/p005.jpg',
    desc: '低脂高蛋白，配油醋汁。鸡胸、藜麦、牛油果和季节蔬菜，适合轻食需求。',
    tags: ['低脂'], status: 'on', allergens: ['坚果'],
    specs: [['规格', '标准份 / 酱汁分装'], ['酱汁', '油醋汁'], ['建议', '低脂高蛋白']] },
  { id: 'p006', name: '山药排骨汤', cat: '汤饮甜品', price: 12, sold: 96, stock: 24, img: '/assets/dishes/p006.jpg',
    desc: '温热汤品，午市限量。清炖排骨汤，适合搭配套餐。',
    tags: ['限量'], status: 'on', allergens: ['无'],
    specs: [['规格', '小份 / 大份'], ['建议', '温热饮用 · 限量供应']] },
  { id: 'p007', name: '鲜橙气泡水', cat: '汤饮甜品', price: 10, sold: 63, stock: 0, img: '/assets/dishes/p007.jpg',
    desc: '冷饮，当前暂未上架。外卖和冷链规则未确认，暂不开放购买。',
    tags: ['已下架'], status: 'off', allergens: ['无'],
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

// 商户端 订单 (履约模型: 待制作 → 待取餐 → 已完成 / 已取消)
const ADMIN_ORDERS = [
  { id: 'a1', no: 'SA2406100131', code: 'A131', status: '待制作', time: '16:51', mins: 1, total: 60, count: 2,
    contact: '陈女士', phone: '159****2031', flavor: '双拼饭加辣 ×1', note: '打包分开装',
    items: [['p001', 1, 32], ['p002', 1, 28]] },
  { id: 'a2', no: 'SA2406100129', code: 'A129', status: '待制作', time: '16:49', mins: 3, total: 26, count: 1,
    contact: '吴先生', phone: '137****7788', flavor: '—', note: '',
    items: [['p004', 1, 26]] },
  { id: 'a3', no: 'SA2406100126', code: 'A126', status: '待制作', time: '16:42', mins: 10, total: 76, count: 3,
    contact: '林先生', phone: '138****6620', flavor: '加饭 · 加辣', note: '双拼饭加饭',
    items: [['p001', 2, 32], ['p006', 1, 12]] },
  { id: 'a4', no: 'SA2406100120', code: 'A120', status: '待制作', time: '16:35', mins: 17, total: 58, count: 2,
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
  { id: 'a8', no: 'SA2406100071', code: 'A071', status: '已取消', time: '14:55', mins: 0, total: 26, count: 1,
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

// 营业设置（小程序端 admin-settings 里为写死展示值，PC 端收敛为可编辑状态）
const SETTINGS = {
  openTime: '09:00',
  closeTime: '17:00',
  cutoff: '16:30',
  pickupPoint: '县前直营店',
  notice: '今日卤味新鲜出锅，欢迎到店自提～',
};

// 开屏装饰图层（对应小程序 utils/layer.js DEFAULTS）
const LAYER_DEFAULTS = { v: 1, enabled: false, src: '', cx: 0.5, cy: 0.38, size: 0.35, ar: 1 };

/* ============================================================
   以下为「会员等级 / 会员名单 / 优惠券」种子数据
   —— 二期能力，不在一期合同范围。数据经 data/api.js 读写，
      页面不得直接引用这三个常量。
   ============================================================ */

// 会员等级档位（有序，一人一档；商户可增删改名、调折扣率）
// discount: 折扣百分比，100 = 无折扣，85 = 打 8.5 折
const LEVELS = [
  { id: 'lv1', name: '初级会员', sort: 1, discount: 95, desc: '入册员工与常客' },
  { id: 'lv2', name: '中级会员', sort: 2, discount: 90, desc: '在职满一年员工' },
  { id: 'lv3', name: '高级会员', sort: 3, discount: 85, desc: '管理层与长期合作单位' },
];

// 会员名单（手机号为唯一识别键，与微信授权手机号比对）
// bound: 该手机号是否已有用户在小程序完成授权；spend/orders 为累计消费与单量，只读
const MEMBERS = [
  { id: 'm1', phone: '13800006620', name: '林建国', levelId: 'lv2', org: '县前管理处', dept: '综合科', jobNo: 'XQ0107', remark: '', enabled: true, joinAt: '2026-03-12', bound: true, spend: 1286, orders: 42 },
  { id: 'm2', phone: '13500009012', name: '黄映雪', levelId: 'lv3', org: '县前管理处', dept: '办公室', jobNo: 'XQ0031', remark: '会议餐常客', enabled: true, joinAt: '2026-01-08', bound: true, spend: 3140, orders: 96 },
  { id: 'm3', phone: '15900002031', name: '陈少芬', levelId: 'lv1', org: '青云镇站所', dept: '窗口', jobNo: 'QY0212', remark: '', enabled: true, joinAt: '2026-04-02', bound: true, spend: 468, orders: 19 },
  { id: 'm4', phone: '13700007788', name: '吴国强', levelId: 'lv1', org: '青云镇站所', dept: '后勤', jobNo: 'QY0330', remark: '', enabled: true, joinAt: '2026-04-02', bound: false, spend: 0, orders: 0 },
  { id: 'm5', phone: '13300004456', name: '郑文彬', levelId: 'lv2', org: '县前管理处', dept: '财务科', jobNo: 'XQ0088', remark: '', enabled: true, joinAt: '2026-02-20', bound: true, spend: 902, orders: 31 },
  { id: 'm6', phone: '18800000021', name: '王秀莲', levelId: 'lv3', org: '合作单位', dept: '绥芬河北站', jobNo: '', remark: '长期团餐对接人', enabled: true, joinAt: '2025-11-15', bound: true, spend: 5620, orders: 138 },
  { id: 'm7', phone: '13000005567', name: '刘志海', levelId: 'lv1', org: '县前管理处', dept: '综合科', jobNo: 'XQ0142', remark: '', enabled: true, joinAt: '2026-05-06', bound: false, spend: 0, orders: 0 },
  { id: 'm8', phone: '15000003322', name: '孙丽萍', levelId: 'lv1', org: '青云镇站所', dept: '窗口', jobNo: 'QY0401', remark: '已调离，暂停权益', enabled: false, joinAt: '2026-01-22', bound: true, spend: 214, orders: 9 },
];

// 优惠券（按等级自动生效，无领取动作；一单只用一张，与等级折扣叠加）
// type: 'cut' 满减(amount) | 'discount' 折扣(rate + cap 封顶必填)
// scope: 'all' 全场 | 'cat' 指定分类(catNames) | 'item' 指定菜品(itemIds)
const COUPONS = [
  { id: 'cp1', name: '入册好礼', type: 'cut', amount: 5, rate: 0, cap: 0, threshold: 30,
    levelIds: ['lv1', 'lv2', 'lv3'], scope: 'all', catNames: [], itemIds: [],
    start: '2026-08-01', end: '2026-08-31', perLimit: 3, enabled: true },
  { id: 'cp2', name: '中高级专享满减', type: 'cut', amount: 10, rate: 0, cap: 0, threshold: 50,
    levelIds: ['lv2', 'lv3'], scope: 'all', catNames: [], itemIds: [],
    start: '2026-08-01', end: '2026-08-31', perLimit: 2, enabled: true },
  { id: 'cp3', name: '高级会员八折券', type: 'discount', amount: 0, rate: 80, cap: 15, threshold: 0,
    levelIds: ['lv3'], scope: 'all', catNames: [], itemIds: [],
    start: '2026-08-01', end: '2026-08-15', perLimit: 1, enabled: true },
  { id: 'cp4', name: '今日套餐立减', type: 'cut', amount: 8, rate: 0, cap: 0, threshold: 55,
    levelIds: ['lv2', 'lv3'], scope: 'cat', catNames: ['今日套餐'], itemIds: [],
    start: '2026-08-01', end: '2026-08-31', perLimit: 5, enabled: true },
  { id: 'cp5', name: '开业尝鲜券', type: 'cut', amount: 6, rate: 0, cap: 0, threshold: 20,
    levelIds: ['lv1', 'lv2', 'lv3'], scope: 'all', catNames: [], itemIds: [],
    start: '2026-06-01', end: '2026-06-30', perLimit: 1, enabled: true },
];

// 当前登录用户（微信授权手机号；真实实现由服务端用 code 换取，前端拿不到明文）
const ME = { phone: '13800006620', nick: '林先生', avatarChar: '林' };

// 当前用户各券的已用次数（真实实现由服务端按 openid + couponId 统计）
const MY_COUPON_USED = { cp1: 1, cp2: 0, cp3: 0, cp4: 0, cp5: 1 };

// 演示时钟：与小程序 NOW_MINS 同源，用于券有效期判定
const TODAY = '2026-08-07';

// 店长（PC 端顶栏账号展示；小程序端写在 admin-profile.wxml）
const MANAGER = { name: '高特', role: '店长' };

window.Seed = {
  STORE, HUES, CATS, MENU, menuList, itemById, ADMIN_ORDERS, RANK, ADMIN_CATS, PICKUP_POINTS,
  SETTINGS, LAYER_DEFAULTS, LEVELS, MEMBERS, COUPONS, ME, MY_COUPON_USED, TODAY, MANAGER,
};
