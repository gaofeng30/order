# 页面改造约定（基础层已完成，页面只按此使用）

基础层已经改好的文件，页面改造时**不要修改**：`app.wxss`、`app.json`、`components/*`、`utils/util.js`、`utils/dateLabel.js`、`utils/navBehavior.js`、`utils/reservationMenuStore.js`、`utils/storefrontStore.js`、`utils/money.js`、`utils/catalogStore.js`。确实需要改它们时，停下来在报告里说明原因。

所有页面 json 已加 `"disableScroll": true`（D18），不要删。

## 1. 字号（G2）——用变量，不写裸数字

| 角色 | 变量 | 值 |
|---|---|---|
| 辅助：标签、角标、提示、tabbar | `var(--fs-xs)` | 22rpx（下限） |
| 次要：描述、时间、地址、说明 | `var(--fs-sm)` | 25rpx |
| 正文：菜名、表单、信息行、按钮 | `var(--fs-md)` | 28rpx |
| 标题：导航、区块、弹层、卡片标题 | `var(--fs-lg)` | 32rpx |
| 大标题：门店名、详情菜名、「支付成功」、头像字 | `var(--fs-xl)` | 40rpx |

- 按**角色**归档，不按原数值就近：菜名原来 30 → 正文 28；卡片区块标题原来 30 → 32。
- 取餐号：取餐码页、支付结果页统一 `136rpx`；商户订单详情、核销成功弹层 `96rpx`；订单列表卡片上的取餐号 `var(--fs-xl)`。
- 金额 `<money size>` 只用 28 / 36 / 44 三档，商品详情大价格保留 60：行内单价、小计 28；列表价格 36；合计、应付 44。
- 启动页主视觉大标题（原 84rpx）保留，作为品牌展示例外。
- 删除所有 `<text class="t" style="font-size:30rpx">` 这类行内覆盖，`.sec-h .t` 已是 32rpx。

## 2. 颜色（G3）

- 绿 = 所有可点击操作与选中；深蓝 = 品牌底色（`.hero-dark`）；金 = 点缀；红 `var(--red)` / `var(--red-bg)` = 警示（售罄、错误、角标）与员工优惠价；灰阶 = 文字层级、不可用。
- `btn--ghost-blue` 已删除 → 改 `btn--ghost`。`<money blue>`、`<stepper blue>` 属性已删除 → 去掉。
- 未定义变量替换：`--ink-1`→`--ink`，`--green-1`→`--green-600`，`--green-2`→`--green`，`--blue-2`/`--blue-3`（选中态）→ 用绿色选中样式，`--brand`→`--green`。
- 写死颜色替换：`#2a5fa6`（链接/图标蓝）→ `#3d6b2f`；`#9fd3ff`（深底上的浅蓝图标）→ `#b79a55`；`#d2483a` `#c0392b` `#9a493e` `#a15245` → `var(--red)`（icon 的 color 属性写 `#c4432f`）。
- icon 的 `color` 属性只能写十六进制：绿 `#3d6b2f`/`#467a32`，灰 `#8f9384`，红 `#c4432f`，金 `#b79a55`，白 `#ffffff`，深 `#22271d`。

## 3. 细线与描边（D33、D34）

- 所有 `1rpx` 边框改 `2rpx`；`3rpx` 描边改 `4rpx`。列表行之间用 `var(--line-2)`，区块分隔（栏上边框、弹层头）用 `var(--line)`；不写 `#ececdf` 这类写死色。
- 列表最后一项去掉下边框（`last` 类或 `wx:if` 判断 index），避免与下方边框/卡片底叠成双线（D32）。
- 输入框统一加 `input-box` 类（`box-shadow: inset 0 0 0 2rpx var(--line)` + 浅底），不用 `border`。

## 4. 共用类（app.wxss）

- `.btn .btn--primary / .btn--ghost / .btn--mute / .btn--sm / .btn--block / .btn.is-disabled`；原生 `<button class="btn">` 已修好宽度（G9）。
- `.link`（绿字链接）、`.capsule`（绿色描边小胶囊，点击区已扩到 88rpx）。
- `.staff-price-label`（红底白字）、`.original-price`（灰色划线）、`.price-staff`。
- `.search-box / .search-in / .search-ph / .search-clear`。
- `.segs / .seg / .seg.on`（选中为绿，两端已留白；外层容器**不要再加左右 padding**，D36）。
- `.sheet-mask / .sheet / .sheet-close`（关闭按钮 88rpx 点击区）。
- `.add-btn`（圆形「+」，见下）、`.tail-space`（有 tabbar 页面滚动区尾部留白 32rpx，D30）、`.is-locked`（锁定输入灰字）。
- 按压反馈（D16）：`.btn` 用 `hover-class="b-active"`；卡片、列表行、宫格、`.prow` 等可点击 view 用 `hover-class="tap-dim"`；都加 `hover-stay-time="80"`。原生 `<button>` 写 `hover-class="b-active"`。

## 5. 组件

- `<state kind="loading|error|empty" text="" action="" icon="" bind:action="...">`：统一的加载中 / 失败 + 重试 / 空 + 引导（G4）。error 不传 action 时按钮文案默认「重试」；empty 不传 action 则无按钮。失败文案写用户能懂的话，例如「网络不太好，订单没加载出来」。
- `<money v|cents size staff color cur>`：无值时整体不渲染（D4）；`staff` 为员工价红色。不要再传 `blue`。
- 员工价标准写法（E3、D5，全站统一，不写「原价」二字）：
  ```
  <view wx:if="{{x.isStaffPrice}}" class="staff-price-label">员工价</view>
  <money v="{{x.price_text}}" size="36" staff="{{x.isStaffPrice}}" />
  <text wx:if="{{x.isStaffPrice}}" class="original-price">¥{{x.original_price_text}}</text>
  ```
  非员工只显示 `<money>`（深色），不显示划线（PRD §5.6）。
- 圆形「+」（U10、U20）：
  ```
  <view class="add-btn {{orderable ? '' : 'is-disabled'}}" catchtap="..." hover-class="tap-dim" hover-stay-time="80" aria-role="button" aria-label="加入购物车">
    <view class="add-dot"><icon name="plus" size="14" color="{{orderable ? '#ffffff' : '#9a9e8c'}}" stroke="2.4" /></view>
  </view>
  ```
  加购后原位换成 `<stepper>`。
- `<stepper value bind:sub bind:add>`：已有 88rpx 点击区。
- `<navbar ... overlay solid="{{navSolid}}">` + 页面 scroll-view `bindscroll="onNavScroll"`（navBehavior 已提供 `onNavScroll` 与 `navSolid`），透明导航页面滚动后变深蓝底（D17）。
- `<imageph item="{{product}}" h w radius tag lazy>`：会自动读 `item.cover.url` 显示真实封面，加载失败退回首字占位（D12、D15、D21）。菜单等长列表传 `lazy="{{true}}"`。页面里需要直接用 `<image>` 显示菜品图时，也必须 `binderror` 退回 `imageph`。
- `<qrcode>` 已有同尺寸占位（D13）。
- `<toast id="toast">`：`this.selectComponent('#toast').show('文案')` 成功；`show('文案', { icon: 'warn' })` 警告/失败（红色图标）。已居中显示（E9）。
- `<customize>`：抬头价格、口味标签、备注计数已在组件内处理。

## 6. 工具

- `require('../../utils/dateLabel.js')`：`dayLabel('2026-10-12')` → 今天/明天/昨天/10月8日/2025年12月30日；`instantLabel(rfc3339)` → 「昨天 21:05」；`shanghaiToday()`。所有日期显示都走它（G8、D44），不要 `slice(5)`，不要 `new Date('YYYY-MM-DD')`。
- `util.pickup.label()` → 「今天 12:00」；`util.pickup.periodLabel()` → 「今天 午餐」。
- `util.cart`：
  - `list(availability)` 每项有顶层 `id`（用于 `wx:key="id"`，D21）和 `unavailableLabel`（''、'该时段不供应'、'已售罄'）。
  - `count(availability)`、`totalCents(availability)`、`total(availability)`（字符串，两位小数，D1）只算可售项；`unavailableCount(availability)`。
  - `reprice(productsById)`：用最新目录价刷新购物车（E3）。
  - 不传 availability 时视为全部可售。
- `reservationMenuStore.unavailableReason(storeStatus, mealPeriod)` → 「午餐已截单，请改选时间」等；可下单时为空串。
- **菜单状态共享约定**（菜单页写，详情页、确认页读）：菜单每次 `loadMenu` 成功后写
  ```
  getApp().globalData.menuState = {
    date, time, mealPeriod,          // 与 util.pickup.get() 对应
    orderable,                       // 当前所选时段整体能否下单
    reason,                          // unavailableReason(...)，可下单时 ''
    availability,                    // { [productId]: { orderable, soldOut } }
  }
  ```
  读取方先核对 `date/time` 与当前 `pickup.get()` 一致再使用；不一致或不存在时，详情页自行调用 `menuStore.loadMenu(selection)` 计算。

## 7. 交互规则

- 请求中按钮置灰并显示进行中文案、防重复点击；结束后 toast 告知结果，失败保留输入（G5）。不可点的按钮置灰，点击时 toast 说明原因。
- 二次确认一律用 `wx.showModal`（官方组件，不自绘）。复制用 `wx.setClipboardData`。
- 点击区域不小于 88rpx（G6）：小图标按钮用外层 88rpx 容器或 `::before` 扩展。
- 图标尺寸（D23）：关闭 20、编辑 14、右箭头 16。
- 纯图标可点击元素加 `aria-role="button" aria-label="…"`（D24）。
- 用词（G7）：取餐号 / 取餐二维码 / 订单；不出现「服务端」「快照」「核销二维码」「校验」等开发用语。
- `wx:key` 用顶层唯一字段（`id`、`code`），对象数组不用 `*this`（D21）。
- 有 tabbar 的页面：滚动区尾部用 `<view class="tail-space"></view>`，不要再加 `safeBottom`（D30）；无 tabbar 的底栏仍按 `safeBottom` 避让。
- 不使用已废弃接口（如 `wx.getUserProfile`）。setData 只传变化的字段。
