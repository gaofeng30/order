## Why

服务端匿名目录已经集成并归档，但小程序首页、菜单、详情和结算商品展示仍从进程内 `MENU/globalData.menu` mock 读取；合法空目录会回退 mock，未知详情会回退首商品，网络失败也没有可恢复状态。现在只把这条用户端目录消费链接到已冻结的服务端契约，并用本地可复现的小程序页面模拟达到 UI1。

## What Changes

- 新增只拼接 app 级 `apiBaseUrl` 的 catalog HTTP wrapper 与独立用户端 catalog store，调用且只调用匿名 `GET /api/v1/catalog` 与 `GET /api/v1/catalog/products/:id`。
- 首页按服务端目录稳定顺序展示前四个商品，菜单按服务端分类/商品顺序展示，详情按路由中的十进制字符串 ID 单独请求；三页显式实现 `loading/empty/error/ready` 与错误重试，404 不回退首商品，任一失败不回退 mock。
- 购物车在加购时复制本次服务端目录响应中的商品快照，菜单购物车与确认页只读该快照；ID 始终为 string，价格事实始终保留非负整数 `price_cents`。展示与既有 confirm promo/pay handler 所需的 `price_text`/兼容元值只能临时由 cents 确定性派生，不写回 snapshot/store，也不成为业务事实。
- 从本轮用户目录 UI 移除 catalog 契约不存在的库存、售罄、可售、员工价、销量、标签、图片和过敏原读取；不把“目录可见”表述为“当前可购买”。
- 在 `apps/wechat-miniprogram/` 内新增零第三方依赖的局部 `package.json`/lock 与 Node 页面模拟 harness，实际加载 Page 脚本及相关 WXML 状态契约、模拟 `wx.request`，覆盖主场景、关键错误/恢复路径，以及既有 confirm promo/pay 入口不被目录 item shape 破坏的 non-regression。

## Capabilities

### New Capabilities

- `miniprogram-menu-catalog`: 冻结小程序匿名目录 transport、页面状态、稳定渲染、严格 ID/金额、商品快照以及本地 UI1 验收。

### Modified Capabilities

无。`persistent-menu-catalog` 的 provider 路径、JSON、404/503 与 availability 非目标契约保持只读，不由本 change 修改。

## Impact

### 状态、Outcome 与 Acceptance Verdict

- 状态：`DRAFT`。尚未批准、未进入实现、未提交候选，也未 push、创建 PR、部署或写入外部系统。
- 唯一 outcome：匿名小程序用户在首页、菜单、详情和确认页看到的目录商品事实只来自已集成 catalog API；网络/服务错误与 404 有可恢复且不污染 mock 的页面结果，购物车商品快照可稳定带到确认页。
- 单一 acceptance verdict：只有同一 candidate 在 provider 契约回归、全部受影响 consumer、真实 Red→Green→Refactor、本地 Node 小程序 Page/UI 状态模拟 UI1、strict、owned-path、敏感与 exact-SHA 独立验证全部通过后才可接受；页面或快照任一部分不能独立发布或回滚。
- `base_sha=94e04bf26e37e93299c26ef2c9c8aa7552619444`；`candidate_sha=none`。
- `gate_type=W2`：改变公共 API consumer 与用户可见 UI；`ui_level_target=UI1`、`ui_level_actual=NOT_RUN`。
- candidate 目标 `C=9,T=10,V=8,R=9,total=36`，每项不低于 8、硬阻断为零；V=8 只允许表示 exact-SHA verifier 尚待执行，不能冒充独立 PASS。

### 调用方、场景与时机

- 调用方：无需登录或手机号的微信小程序用户端首页、菜单、商品详情，以及本地购物车/确认页商品明细。
- 业务场景：进入或重新显示首页/菜单时读取公开分类和商品；进入详情时按点击得到的 product ID 读取公开详情；加购时复制该次服务端响应商品，进入确认页时展示复制的商品快照。
- 调用时机：只在公开浏览与本地选择阶段。catalog 200 不代表任一日期/餐段可购买；本 change 不创建报价、订单、库存预占或支付。

### Owner、Branch 与 Worktree

- owner：`Miniprogram Catalog Connection Writer`；本 change 从 DRAFT 到 CANDIDATE 只有该 writer。
- branch：`codex/connect-miniprogram-menu-catalog`。
- worktree：`/Users/vivix/.codex/worktrees/order-connect-miniprogram-menu-catalog.Writer`。

### Writer Owned Paths

- `openspec/changes/connect-miniprogram-menu-catalog/**`
- `apps/wechat-miniprogram/app.js`（只新增本地 UI1 的 app 级 `apiBaseUrl`，不改变既有 mock 初始化）
- `apps/wechat-miniprogram/utils/catalogApi.js`
- `apps/wechat-miniprogram/utils/catalogStore.js`
- `apps/wechat-miniprogram/utils/util.js`（只修改购物车商品快照与整数分合计边界）
- `apps/wechat-miniprogram/pages/home/home.js`
- `apps/wechat-miniprogram/pages/home/home.wxml`
- `apps/wechat-miniprogram/pages/home/home.wxss`
- `apps/wechat-miniprogram/pages/menu/menu.js`
- `apps/wechat-miniprogram/pages/menu/menu.wxml`
- `apps/wechat-miniprogram/pages/menu/menu.wxss`
- `apps/wechat-miniprogram/pages/detail/detail.js`
- `apps/wechat-miniprogram/pages/detail/detail.wxml`
- `apps/wechat-miniprogram/pages/detail/detail.wxss`
- `apps/wechat-miniprogram/pages/confirm/confirm.js`
- `apps/wechat-miniprogram/pages/confirm/confirm.wxml`
- `apps/wechat-miniprogram/components/customize/customize.js`（只改为用整数分商品快照计算/格式化本地选择合计）
- `apps/wechat-miniprogram/components/customize/customize.wxml`
- `apps/wechat-miniprogram/package.json`
- `apps/wechat-miniprogram/package-lock.json`
- `apps/wechat-miniprogram/tests/page-harness.js`
- `apps/wechat-miniprogram/tests/catalog-ui1.test.js`

禁止扩大 ownership。尤其不拥有 Go 后端、migration、admin CRUD/页面、`globalData.menu`/`utils/data.js` mock 事实、`utils/api.js`、`utils/promo.js`、result/order/payment/history 页面、库存、订单服务、身份、支付、根 `package`/Node 工具链、`project.config.json`、`app.json`、PRD、canonical/archived specs、skills 或 `AGENTS.md`；这些非 owned 文件必须 byte-unchanged。

### 只读共享契约与依赖

- 依赖 `serve-persistent-menu-catalog` 已在 local main 的 `2209c071a21860231827b2a8c8c81d9b7745e6e1` 归档；该 exact SHA 是当前 base 的 ancestor，canonical `openspec/specs/persistent-menu-catalog/spec.md` 与生产 handler 均存在。
- 只读 provider 契约：`GET /api/v1/catalog`、`GET /api/v1/catalog/products/:id`，成功 DTO 的字符串 ID/整数分、空数组、404 `PRODUCT_NOT_FOUND`、503 `CATALOG_UNAVAILABLE`，以及“目录不承诺 availability”的边界。
- `apps/wechat-miniprogram/utils/data.js`、`utils/api.js`、`utils/promo.js`、商户页面、result/order/payment/history 页面与既有 P0 mock 仍是 byte-unchanged 只读共享面；独立 catalog store 不覆盖 `globalData.menu`，避免把服务端最小 DTO 塞入依赖 stock/status/image 的商户 mock 链。confirm 保留现有 `loadPromo/recalc/openCoupon/pay` handler/入口，只让其从 cart view 临时取得 snapshot-derived name/string ID/兼容元值；不把 promo/pay 结果计作目录或支付正确性。
- 规划时当前机器为 Node `v25.8.1`、npm `11.11.0`；仓库原先没有任何 package、lock、Node 版本约束或页面测试。局部包只使用 Node 内建 `node:test`，不引入第三方 runtime/build dependency，也不改变根工具链。

### API Base 与必要资产

- 仓库没有现有小程序公共 `apiBaseUrl`/origin，也没有 `wx.request` wrapper。唯一方案是在 `app.globalData.apiBaseUrl` 新增 `http://127.0.0.1:8080`，且明确只服务仓库本地 UI1；catalog wrapper 只在其后拼接冻结 path，不做多环境配置、自动探测、缓存兜底或 mock fallback。
- UI1 本地资产由本 change 在仓库内实现：零依赖 Node Page harness、受控 `wx.request` fixture、真实 Page JS/WXML 状态契约与局部 package/lock。它当前尚未运行，不得记录 PASS。
- UI2=`BLOCKED_EXTERNAL`：owner 为开发方与客户小程序管理员；缺少已锁定微信开发者工具/体验版、项目登录权限和真实 HTTPS API 域名；恢复条件是指定工具/版本、具备权限的受控项目与已配置域名可重复运行相同场景。
- UI3=`BLOCKED_EXTERNAL`：owner 为 UAT owner 与客户平台管理员；缺少指定真机、受控账号/目录数据、体验版和可达 HTTPS 域名；恢复条件是这些资产就绪并按同一矩阵记录版本、设备/账号边界与最终页面结果。
- UI2/UI3 不冒充 PASS，也不阻塞用户已冻结的本地 UI1 Goal。

### 最小成功标准

- wrapper 只发两种匿名 GET，base/path 精确；响应只保留冻结目录字段，ID 不经过 Number，`price_cents` 是非负安全整数分。
- 首页、菜单、详情分别取得 `loading/empty/error/ready`，错误页可重试；首请求失败后重试成功，空目录，详情 404/503 与 ready 都由真实页面 lifecycle + `wx.request` fixture 证明，任一失败不出现 mock 商品。
- 首页按 list response 中 category/product 的既有稳定顺序 flatten 后取前四个；菜单保留 category 顺序及启用空分类；详情 404 始终不设置商品、不回退第一件。
- cart 以 string ID 保存服务端商品快照、数量和本地偏好；目录刷新/失败后确认页仍展示该快照，且金额事实从 `price_cents` 整数运算得到。cart view 可为 byte-unchanged promo/pay 临时派生价格文本/兼容元值，但不得写回 snapshot。该快照不锁价/库存，不替代未来服务端 quote/order validation。
- confirm UI1 必须证明现有 `loadPromo/recalc/openCoupon/pay` 仍可调用：受控 all-scope mock promo 可渲染且不因新 item shape 抛错；pay 仍产生既有 P0 mock order/导航，confirm 商品名称、string ID 与兼容价格来自 snapshot-derived view，catalog 与 mock-menu request/read 均为 0。该检查只证明目录接入未破坏入口，不证明 category/item coupon 适用性、订单或支付正确。
- 大于 JS 安全整数的合法十进制 string ID fixture 从 list/detail、store、route 到 cart/confirm 全程逐字节保持；未知/隐藏 ID 不被替换。
- provider focused test、局部 UI1、JS/JSON static、OpenSpec strict、diff/owned/sensitive/forbidden-field checks 与 exact-SHA verifier 全部 PASS；未删/放宽失败测试，未用 mock 替换被测页面/production catalog client，只有 `wx.request` transport 是可控 UI1 fixture。

### Integration / Archive Boundary

- writer 只在批准后实现并形成一个本地中文 CANDIDATE commit；不 push、不创建 PR、不 deploy。
- verifier 只对完整 candidate SHA 在另一 clean detached worktree 从头执行声明 Gate，任何 code/artifact/task/base/SHA 变化使旧验证失效。
- 获得单独集成授权且 exact-SHA independent PASS 后，只允许把 candidate 纯 fast-forward 到未推进的 local main；若 main 推进或无法 FF，返回原 writer 更新并重验，不 merge/rebase 后沿用旧 attestation。
- main 集成验证通过后才由 integration handler 更新 canonical `openspec/specs/miniprogram-menu-catalog/spec.md` 并归档 change；archive 也只在 local main 完成，不 push/PR/deploy。

### Non-Goals

- 不改 Go catalog provider、schema、migration、错误契约或公开路由。
- 不做 admin 分类/商品 CRUD、seed、图片/COS、标签、销量或商户 mock 数据迁移。
- 不做库存、售罄、availability/orderable、日期、餐段、时段、营业状态、员工价、身份、登录、手机号或 RBAC。
- 不做服务端购物车、报价/算价、订单创建/快照、支付、退款或库存预占；不得删除、禁用或扩展现有 confirm promo/pay handler/展示，现有 P0 假订单/假支付只做 byte-unchanged 依赖上的 non-regression，不属于本 change 的业务验收结果。
- 不做缓存、离线目录、mock fallback、重试策略抽象、通用 HTTP SDK、多环境配置、npm 第三方测试框架、根 Node 工具链或相邻重构。
- 不把 UI1 结果升级为微信开发者工具、体验版、真机、合法域名、真实平台或生产 PASS。
