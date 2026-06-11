# 绥安食品 · 微信点餐小程序（前端）

按 `绥安食品-前端开发PRD.md` 与设计稿原型 1:1 还原的微信小程序前端，含**用户端**与**商户端**双端，启动页自选身份进入。

## 运行方式

1. 用**微信开发者工具**导入本仓库的 `wechat-miniprogram/` 目录（`project.config.json` 已将 `miniprogramRoot` 指向 `miniprogram/`）。
2. AppID 选择「测试号 / 不使用 AppID」即可预览（配置中为 `touristappid`）。
3. 基础库建议 `3.x`（已用到 canvas 2d、`getWindowInfo` 等接口，并有低版本回退）。

## 多机型自适应（核心）

- **全量 `rpx` 布局**：设计基准 375pt，所有尺寸按 `1px → 2rpx` 换算，`750rpx = 屏宽`，在任意手机宽度下等比缩放。
- **动态状态栏 / 胶囊适配**：`app.js` 启动时读取 `statusBarHeight` 与右上胶囊 `getMenuButtonBoundingClientRect()`，算出标题栏高度与右侧操作避让宽度，注入 `navbar` 组件与各页 `navBehavior`。沉浸式深色头按真机状态栏高度留白。
- **底部安全区**：`safeArea` 计算 `safeBottom`，TabBar、底部操作栏、弹层均叠加 `env(safe-area-inset-bottom)` / 实测高度，适配全面屏 Home 条。
- 全部页面 `navigationStyle: custom`，自绘导航，避免不同机型原生导航高度差异。

## 目录结构

```
miniprogram/
├─ app.js / app.json / app.wxss      全局：适配信息采集、跨页状态(globalData)、设计 tokens
├─ utils/
│  ├─ data.js        菜品/订单/分类 mock 数据（对应 PRD 第 4 节）
│  ├─ util.js        路由(go/replace/tabTo/reset/back)、状态语义色、购物车、订单状态机
│  ├─ icons.js       线性图标路径集
│  └─ navBehavior.js 页面适配尺寸注入
├─ components/        navbar / tabbar / pill / money / stepper / imageph / toast / qrcode / icon
└─ pages/            18 屏（用户端 9 + 商户端 9）
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

跨页共享状态集中在 `app.globalData`：购物车 `cart`、用户订单 `orders`、商户订单 `aOrders`、营业状态 `store`、上下架 `products`。订单状态机（接单→备好→核销）与「撤销」Toast 见 `utils/util.js` 的 `advanceOrder`。

> 数据与交互均为前端 mock；支付、登录、真实二维码、打印机等以后端契约对接（详见 PRD 第 9 节「待业务确认 / 二期」）。
