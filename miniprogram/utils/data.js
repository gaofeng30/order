/* 绥安食品 — 菜品 / 订单 / 分类 数据 (真实物料 p001–p007) */

const STORE = {
  name: '绥安食品',
  branch: '县前直营店',
  addr: '漳浦县绥安镇府前路 18 号',
  cutoff: '今日 16:30 截单',
  pickup: '预计 17:10 可取',
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

const itemById = id => MENU.find(m => m.id === id);

const USER_ORDERS = [
  { id: 'o1', no: 'SA2406100126', code: 'A126', status: '待取餐', time: '06-10 16:42', total: 70,
    contact: '林先生', phone: '138****6620', note: '双拼饭加饭', flavors: ['加饭', '加辣'],
    items: [['p001', 2, 32], ['p006', 1, 12]] },
  { id: 'o2', no: 'SA2406100098', code: 'A098', status: '待支付', time: '06-10 16:20', total: 54,
    contact: '林先生', phone: '138****6620', note: '', flavors: [],
    items: [['p002', 1, 28], ['p005', 1, 30]] },
  { id: 'o3', no: 'SA2406090311', code: 'A311', status: '已完成', time: '06-09 12:08', total: 62,
    contact: '林先生', phone: '138****6620', note: '鸡腿排双拼', flavors: ['双拼', '酱汁分装'],
    items: [['p004', 1, 26], ['p005', 1, 30], ['p006', 1, 12]] },
  { id: 'o4', no: 'SA2406080077', code: 'A077', status: '已取消', time: '06-08 18:30', total: 58,
    contact: '林先生', phone: '138****6620', note: '', flavors: [],
    items: [['p001', 1, 32], ['p004', 1, 26]] },
];

const ADMIN_ORDERS = [
  { id: 'a1', no: 'SA2406100131', code: 'A131', status: '待接单', time: '16:51', mins: 1, total: 60, count: 2,
    contact: '陈女士', phone: '159****2031', flavor: '双拼饭加辣 ×1', note: '打包分开装',
    items: [['p001', 1, 32], ['p002', 1, 28]] },
  { id: 'a2', no: 'SA2406100129', code: 'A129', status: '待接单', time: '16:49', mins: 3, total: 26, count: 1,
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
  { id: 'a8', no: 'SA2406100071', code: 'A071', status: '已取消', time: '14:55', mins: 0, total: 26, count: 1,
    contact: '孙女士', phone: '150****3322', flavor: '—', note: '用户取消',
    items: [['p004', 1, 26]] },
];

const RANK = [
  { id: 'p003', sold: 320 }, { id: 'p001', sold: 286 }, { id: 'p004', sold: 254 }, { id: 'p002', sold: 198 }, { id: 'p005', sold: 142 },
];

const ADMIN_CATS = [
  { id: 'c1', name: '今日套餐', sort: 1, on: true, count: 2 },
  { id: 'c2', name: '热销菜品', sort: 2, on: true, count: 2 },
  { id: 'c3', name: '轻食低脂', sort: 3, on: true, count: 1 },
  { id: 'c4', name: '汤饮甜品', sort: 4, on: true, count: 2 },
  { id: 'c5', name: '节庆礼盒', sort: 5, on: false, count: 0 },
];

const FLAVORS = ['少饭', '加饭', '少盐', '加辣', '酱汁分装', '免葱蒜', '打包分装', '多双餐具'];

module.exports = {
  STORE, HUES, CATS, MENU, itemById, USER_ORDERS, ADMIN_ORDERS, RANK, ADMIN_CATS, FLAVORS,
};
