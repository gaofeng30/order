const categories = [
  { id: "set", name: "今日套餐", enabled: true, sort: 1 },
  { id: "hot", name: "热销菜品", enabled: true, sort: 2 },
  { id: "light", name: "轻食低脂", enabled: true, sort: 3 },
  { id: "drink", name: "汤饮甜品", enabled: true, sort: 4 }
];

const products = [
  {
    id: "p001",
    categoryId: "set",
    name: "XX商务双拼饭",
    price: 32,
    image: "/assets/dishes/p001.svg",
    imageTone: "blue",
    intro: "黑椒牛肉配低温鸡胸，适合午间快速取餐。",
    desc: "双主菜、时蔬、米饭和例汤组合，兼顾饱腹与清爽口感。",
    sales: 286,
    stock: 42,
    tags: ["今日推荐", "热销"],
    status: "available",
    specs: ["标准份", "少饭", "加饭"],
    allergens: "含牛肉、乳制品调味"
  },
  {
    id: "p002",
    categoryId: "set",
    name: "江南三鲜套餐",
    price: 28,
    image: "/assets/dishes/p002.svg",
    imageTone: "green",
    intro: "虾仁、菌菇与时蔬组合，口味清淡。",
    desc: "适合会议日常餐，搭配清炒时蔬和紫菜蛋花汤。",
    sales: 198,
    stock: 18,
    tags: ["清淡"],
    status: "available",
    specs: ["标准份", "少盐"],
    allergens: "含虾仁、鸡蛋"
  },
  {
    id: "p003",
    categoryId: "hot",
    name: "招牌红烧牛腩",
    price: 36,
    image: "/assets/dishes/p003.svg",
    imageTone: "amber",
    intro: "慢炖牛腩，适合搭配米饭。",
    desc: "红烧汁浓郁但不过甜，当前午市已售罄。",
    sales: 320,
    stock: 0,
    tags: ["售罄"],
    status: "soldout",
    specs: ["标准份"],
    allergens: "含牛肉"
  },
  {
    id: "p004",
    categoryId: "hot",
    name: "蒜香鸡腿排",
    price: 26,
    image: "/assets/dishes/p004.svg",
    imageTone: "dark",
    intro: "高频复购单品，窗口取餐快。",
    desc: "去骨鸡腿排配蒜香酱汁，适合加班餐。",
    sales: 254,
    stock: 35,
    tags: ["热销"],
    status: "available",
    specs: ["标准份", "双拼"],
    allergens: "含大豆、小麦"
  },
  {
    id: "p005",
    categoryId: "light",
    name: "藜麦鸡胸能量碗",
    price: 30,
    image: "/assets/dishes/p005.svg",
    imageTone: "mint",
    intro: "低脂高蛋白，配油醋汁。",
    desc: "鸡胸、藜麦、牛油果和季节蔬菜，适合轻食需求。",
    sales: 142,
    stock: 22,
    tags: ["低脂"],
    status: "available",
    specs: ["标准份", "酱汁分装"],
    allergens: "含坚果"
  },
  {
    id: "p006",
    categoryId: "drink",
    name: "山药排骨汤",
    price: 12,
    image: "/assets/dishes/p006.svg",
    imageTone: "cream",
    intro: "温热汤品，午市限量。",
    desc: "清炖排骨汤，适合搭配套餐。",
    sales: 96,
    stock: 24,
    tags: ["限量"],
    status: "available",
    specs: ["小份", "大份"],
    allergens: "无常见过敏原"
  },
  {
    id: "p007",
    categoryId: "drink",
    name: "鲜橙气泡水",
    price: 10,
    image: "/assets/dishes/p007.svg",
    imageTone: "orange",
    intro: "冷饮，当前暂未上架。",
    desc: "外卖和冷链规则未确认，暂不开放购买。",
    sales: 63,
    stock: 0,
    tags: ["已下架"],
    status: "offline",
    specs: ["标准杯"],
    allergens: "无常见过敏原"
  }
];

const orders = [
  {
    id: "OD20260503001",
    pickupNo: "A126",
    status: "ready",
    statusText: "待取餐",
    amount: 58,
    createdAt: "今日 11:22",
    contact: "陈先生",
    phone: "138 0000 2468",
    remark: "少辣，12:10 后取",
    tastes: ["免葱", "少辣"],
    tasteText: "免葱、少辣",
    pickupPlace: "A 座 2 楼 XX食堂自提窗口",
    pickupTime: "预计 12:10-12:25",
    items: [
      { productId: "p001", name: "XX商务双拼饭", price: 32, quantity: 1 },
      { productId: "p004", name: "蒜香鸡腿排", price: 26, quantity: 1 }
    ]
  },
  {
    id: "OD20260502009",
    pickupNo: "B038",
    status: "completed",
    statusText: "已完成",
    amount: 40,
    createdAt: "昨日 12:04",
    contact: "陈先生",
    phone: "138 0000 2468",
    remark: "不要香菜",
    tastes: ["不要香菜"],
    tasteText: "不要香菜",
    pickupPlace: "A 座 2 楼 XX食堂自提窗口",
    pickupTime: "已于昨日 12:24 核销",
    items: [
      { productId: "p002", name: "江南三鲜套餐", price: 28, quantity: 1 },
      { productId: "p006", name: "山药排骨汤", price: 12, quantity: 1 }
    ]
  }
];

const dashboard = {
  revenue: 12680,
  orders: 386,
  visits: 1824,
  clicks: 932,
  rank: [
    { name: "XX商务双拼饭", count: 86 },
    { name: "蒜香鸡腿排", count: 72 },
    { name: "江南三鲜套餐", count: 61 }
  ]
};

const settings = {
  brand: "XX企业食堂",
  status: "营业中",
  pickupPlace: "A 座 2 楼 XX食堂自提窗口",
  businessTime: "10:30-13:30 / 17:00-19:30",
  cutoffTime: "午餐 13:00 截单",
  notice: "今日午餐高峰预计 12:00-12:30，请按取餐号有序取餐。"
};

const user = {
  name: "陈先生",
  role: "单位员工",
  phone: "138 0000 2468",
  company: "政企服务中心"
};

module.exports = {
  categories,
  products,
  orders,
  dashboard,
  settings,
  user
};
