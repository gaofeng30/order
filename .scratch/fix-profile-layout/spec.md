# fix-profile-layout

关闭 `docs/quality/order-production-delivery-handoff.md` 的 **P0-6：修复已复现的个人中心布局异常**。该项是交付矩阵里唯一标注「未完成」的条目，且已进入 NO-GO 条件。

## 问题

真机尺寸下 `pages/profile/profile` 两处布局崩坏，根因彼此独立。

### 根因 1：附加手机号表单被挤压

`profile.wxml` 的附加手机号 `.prow` 是单个 flex 行，横向排了 4 个子元素：`.prow-main` + `input.extra-phone` + `input.extra-name` + `view.extra-save`。

`.extra-phone` / `.extra-name` / `.extra-save` 三个类在全仓库 `.wxss` 中**没有任何定义**，两个 `input` 因此使用微信原生默认宽度且几乎不收缩。`.prow-main { flex:1; min-width:0 }` 成为行内唯一可收缩元素，承担全部挤压，被压至接近零宽，导致标题「附加手机号」逐字换行。

可用宽度 614rpx（750 − `.pad` 32×2 − `.group` 36×2），再减 3 个 `gap:24rpx`，四个元素实际分配 542rpx。

### 根因 2：原生 button 承担布局

三处交互行把 `.prow` 直接扣在 `<button>` 上（绑定主手机号、商户登录、联系客服）。`app.wxss` 对 `button` 仅有 `box-sizing: border-box`，微信原生的 `margin:auto`、左右 14px padding、`#f8f8f8` 底色与 `button::after` 圆角伪边框全部仍在生效，导致行宽不占满卡片且带非预期灰底与伪边框。

## 目标行为

### 附加手机号

- 默认**收起**，收起态只显示「附加手机号」与展开箭头，**不显示**脱敏号、姓名或命中状态
- 点击行头展开，展开后才显示已保存结果、状态文案与输入表单
- 展开态：结果行独占一行；手机号与姓名并排（宽度比 5:3）；保存独立全宽行
- 手机号与姓名均非空时保存可点
- 保存成功后自动收起
- 姓名从服务端 `extra_phone.name` 预填；手机号服务端只返回脱敏值，仅作 placeholder，不预填

### 交互行

- 三处行的布局完全由 `view` 控制，`<button>` 绝对定位、透明、退出布局流
- 三处必须保留 `<button>` 元素：两处 `open-type="getPhoneNumber"`、一处 `open-type="contact"`，换成 `view` 会丢失能力
- 行宽占满卡片可用宽度，无灰底、无伪边框

## 第二批：首页品牌与图标补齐

用户看过真机效果后追加三项。因「个人中心图标」要再动 `profile.wxml`，与第一批 owned paths 重叠、按 AGENTS.md 不能并行，故并入本 change 重出候选 SHA。

### 首页删「服务功能」标题

`.body { margin-top: -88rpx }` 把首个元素推进深蓝 hero，而那个位置一直是 `.sec-h` 标题——深色字压深色底，真机上近乎隐形。

删除标题；`.home-hero` 底部 padding `128rpx → 48rpx`，`.body` margin-top `-88rpx → 24rpx`。深蓝区收到搜索框下方，白卡片紧随其后不重叠，hero 的 52rpx 圆角完整可见。首屏纵向缩短约 184rpx。

### 首页展示门店徽标

`/assets/emblem.png` 原本只出现在启动页。远端改版后未绑定用户冷启动直接进首页、跳过启动页，徽标因此完全没有曝光位。

在 hero 内把门店名包进品牌锁定行：徽标 92rpx 白底圆角容器 + 72rpx `aspectFit` 图，与 50rpx 门店名同行。「你好，欢迎光临」保持在最上方不动。

类名用 `hero-*` 而非复用 `launch.wxss` 的 `.brand-*`：wxss 是页面作用域，同名类在两页各自定义会被误读成共享规则。

### 个人中心补齐行图标

卡片内五行全部带 `.prow-ico`，与既有的 `receipt` / `headset` 同风格。

| 行 | 图标 | 颜色 |
|---|---|---|
| 我的订单 | `receipt` | `#467a32` |
| 绑定主手机号 | `phone` | `#467a32` |
| 附加手机号 | `user` | `#467a32` |
| 商户登录 | `store` | `#2a5fa6` |
| 联系客服 | `headset` | `#2a5fa6` |

绑定主手机号未被点名但一并补上——不补则未绑定主手机号的用户会看到一行缺口。附加手机号用 `user` 而非 `phone`：它本质是「手机号 + 姓名」双要素身份匹配，与上一行不撞义。图标全部取自 `utils/icons.js` 已有的 50 个定义，不新增资源。

## 边界

- **不改后端**。附加手机号维持单条：PRD §4.1 明确为「第 2 个手机号」，`000018_extend_miniprogram_users.sql` 以四列加 `chk_miniprogram_users_extra_group` 约束固化，`CONTEXT.md` 领域词汇为单数。不新增加号、不做多行。
- **不改 `app.wxss`**。button 复位以 `button.prow` 局部选择器完成。`confirm.wxml:78` 是无 class 的裸 `<button>`，完全依赖原生外观，全局复位会使其退化为裸文字。
- **不改 `confirm.wxml`**。该页 `:79` 有同构的无样式裸「保存」`<view>`，属同类问题但不在本 change owned paths。
- **不改 `order-production-delivery-handoff.md`**。该文件由他人维护中，P0-6 勾选在集成后回填。
- 不改 `onGetPhoneNumber` / `onMerchantPhone` / `saveExtraPhone` 的任何请求分支与文案。
- 不改 `.prow-ico` / `.prow-label` / `.prow-sub` 等共用规则。

## 验收

| P0-6 条目 | 本 change 对应 |
|---|---|
| 1. 附加手机号表单独立全宽布局，五个元素不相互挤压 | 折叠面板 + 结果行独占 + 5:3 并排 + 全宽保存 |
| 2. 输入控件有最小宽度、可点高度、焦点态；小屏不逐字换行 | `.extra-in` 固定 80rpx 高、`min-width:0`、`flex` 比例分配 |
| 3. button 行占满卡片宽度，清除原生 margin/padding/伪边框影响 | 点击层方案：按钮退出布局流 |
| 4. 小屏与标准屏分别验证 | 开发者工具 iPhone SE + iPhone 14 |
| 5. 能断言 bounding box、行宽、无逐字换行的 rendered 回归 | UI2 几何断言 + receipt |

## 会使旧验证失效的变更面

- 附加手机号行的 DOM 层级与 `wx:if` 条件
- `.prow` 是否仍为 flex 容器
- 三处 button 与其行容器的包含关系
- `profile.js` 的 `extraOpen` 状态机
