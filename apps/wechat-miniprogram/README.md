# 绥安食品 · 微信点餐小程序（前端）

按 `绥安食品-前端开发PRD.md` 与设计稿原型 1:1 还原的微信小程序前端，含**用户端**与**商户端**双端，启动页自选身份进入。

## 运行方式

1. 用**微信开发者工具**导入仓库根目录（`project.config.json` 已将 `miniprogramRoot` 指向 `apps/wechat-miniprogram/`）。
2. AppID 选择「测试号 / 不使用 AppID」即可预览（配置中为 `touristappid`）。
3. 基础库建议 `3.x`（已用到 canvas 2d、`getWindowInfo` 等接口，并有低版本回退）。

首页推荐、菜单和商品详情需要先按仓库根 [README](../../README.md#api-服务基线) 启动 MySQL、执行 migration 并运行 Go API。`develop` 默认使用 `utils/runtimeEndpointConfig.js` 中的 `http://127.0.0.1:8080`；API endpoint 不可用时页面会展示可重试错误，不会回退到 mock 目录。

## API endpoint 配置与恢复

- App 冷启动通过 `wx.getAccountInfoSync().miniProgram.envVersion` 选择 `develop`、`trial` 或 `release` 配置；测试环境缺少该 API 时仅按 `develop` 处理。
- `develop` 只允许本机 HTTP `127.0.0.1` origin；默认值用于开发者工具连接本地 API，不接受远端 HTTP/HTTPS 地址。
- `trial` / `release` 当前保持空值。真实域名未知时不要填写 placeholder；两者只接受无 path、query、fragment、userinfo 的 HTTPS origin，并拒绝 IP、`localhost` 与 `.localhost` 回环命名空间。
- 外部域名、DNS/备案、有效 HTTPS 和微信后台 request 合法域名齐备后，以单独配置提交替换 `utils/runtimeEndpointConfig.js` 的对应空值；随后在同一 exact SHA 上重跑 Node/static Gate，并用指定体验版和真机分别冷启动 trial/release，核对 `envVersion`、脱敏请求 host 和目录最终状态均无 loopback。

## 多机型自适应（核心）

- **全量 `rpx` 布局**：设计基准 375pt，所有尺寸按 `1px → 2rpx` 换算，`750rpx = 屏宽`，在任意手机宽度下等比缩放。
- **动态状态栏 / 胶囊适配**：`app.js` 启动时读取 `statusBarHeight` 与右上胶囊 `getMenuButtonBoundingClientRect()`，算出标题栏高度与右侧操作避让宽度，注入 `navbar` 组件与各页 `navBehavior`。沉浸式深色头按真机状态栏高度留白。
- **底部安全区**：`safeArea` 计算 `safeBottom`，TabBar、底部操作栏、弹层均叠加 `env(safe-area-inset-bottom)` / 实测高度，适配全面屏 Home 条。
- 全部页面 `navigationStyle: custom`，自绘导航，避免不同机型原生导航高度差异。

## 目录结构

```
apps/wechat-miniprogram/
├─ app.js / app.json / app.wxss      全局：适配信息采集、跨页状态(globalData)、设计 tokens
├─ utils/
│  ├─ data.js        菜品/订单/分类 mock 数据（对应 PRD 第 4 节）
│  ├─ catalogApi.js  匿名目录 HTTP 客户端
│  ├─ catalogStore.js 目录响应校验、金额格式化与购物车快照
│  ├─ util.js        路由(go/replace/tabTo/reset/back)、状态语义色、购物车、订单状态机
│  ├─ icons.js       线性图标路径集
│  └─ navBehavior.js 页面适配尺寸注入
├─ components/        navbar / tabbar / pill / money / stepper / imageph / toast / qrcode / icon
└─ pages/            27 个已注册页面
```

## 双端导航

原生导航 API 直接映射 PRD 的页面栈语义：

| PRD 操作 | 实现 |
| --- | --- |
| `go(id,params)` 入栈 | `wx.navigateTo` |
| `replace` 替换栈顶 | `wx.redirectTo` |
| `tabTo` 重置单页（Tab 切换） | `wx.reLaunch` |
| `reset` 回启动选择 | `wx.reLaunch(launch)` |
| `back` 出栈 | `wx.navigateBack` |

两端 TabBar 共 9 个 Tab 页（超过原生 TabBar 5 个上限），故采用自定义 `tabbar` 组件 + `reLaunch` 切换，用户端绿色高亮、商户端蓝色高亮，角标分别为待支付数 / 待接单数。

## 业务状态

跨页共享状态集中在 `app.globalData`：购物车 `cart`、用户订单 `orders`、商户订单 `aOrders`、营业状态 `store` 和商户菜单 `menu`。订单状态机（接单→备好→核销）与「撤销」Toast 见 `utils/util.js` 的 `advanceOrder`。

## 当前数据边界

- 首页推荐、菜单和商品详情通过 `utils/catalogApi.js` 读取 Go API 的分类、名称、描述、规格和分价；加入购物车时保存服务端目录快照。
- 目录接口不提供图片、库存、销量、售罄、员工价或可购买状态，页面不应把这些本地演示字段当作服务端事实。
- 购物车、下单、订单状态、商户管理、营业设置、会员和优惠券仍保存在 `app.globalData` 或 mock 中，不写入数据库；商户端改动不会更新服务端目录。
- 支付、登录、真实二维码、打印机等尚未接入（详见 PRD 第 9 节「待业务确认 / 二期」）。
