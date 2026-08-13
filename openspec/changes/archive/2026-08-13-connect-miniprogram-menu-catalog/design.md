## Context

local main 与本 writer 的冻结 base 都是 `94e04bf26e37e93299c26ef2c9c8aa7552619444`；`serve-persistent-menu-catalog` 已在其祖先提交中集成并归档，匿名列表与详情的 JSON、错误和非 availability 边界由 canonical `persistent-menu-catalog` spec 冻结。当前小程序用户端却仍通过 `globalData.menu`、`data.menuList()` 和 `data.itemById()` 读取 P0 mock：首页按 tags 筛选，菜单按本地分类重组，详情未知 ID 回退第一件商品，购物车在结算时再次按 ID 回查 mock。

本 change 只替换首页、菜单、详情和确认页的商品目录事实来源，并建立仓库内可复现的局部 Node Page 模拟器达到 UI1。商户页、历史 mock、P0 假订单/假支付、真实 availability/quote/order、Go provider 与根工具链都不在本 change 内。UI2 微信开发者工具与 UI3 真机所需域名、权限、版本和账号当前不可用，分别保持 `BLOCKED_EXTERNAL`。

## Goals / Non-Goals

**Goals:**

- 用户端首页、菜单和详情只通过已集成的匿名 catalog API 取得商品目录事实，失败与 404 不回退 mock。
- list 页面具备 `loading/empty/error/ready`，错误态有显式 retry；只有 `categories:[]` 是 empty，启用空分类仍是 ready group。
- detail 页面具备 `loading/not_found/error/ready`；404 不设置商品，未知 ID 不回退 `p001` 或首商品。
- 首页按服务端分类与商品稳定顺序 flatten 后取前四件；菜单原样保留分类、商品和启用空分类顺序。
- 本地购物车保存 canonical server product snapshot、数量与用户口味/备注；ID 全程为 string，金额事实全程以整数 `price_cents` 运算，确认页商品事实不回查 mock-menu 或 catalog，并通过临时 snapshot-derived view 保持既有 promo/pay 入口可运行。
- 用零第三方依赖的局部 Node harness 实际加载 Page JS、behavior 与 lifecycle，并结合 WXML 状态契约完成 UI1 主场景和错误/恢复矩阵。

**Non-Goals:**

- 不修改 Go catalog provider、migration、admin CRUD、商户页面或 `globalData.menu`/`utils/data.js` 的 legacy mock。
- 不新增库存、售罄、availability/orderable、日期/餐段、员工价、标签、销量、图片或过敏原契约。
- 不实现服务端购物车、quote、订单、身份、支付、退款、库存预占或真实下单；不删除、禁用或扩展现有 confirm promo/pay handler/入口，`utils/api.js`、`utils/promo.js` 与 result/order/payment/history 页面保持 byte-unchanged。
- 不新增缓存、离线目录、自动重试、mock fallback、通用 HTTP SDK、多环境配置、第三方 npm 依赖或根 Node 工具链。
- 不把 UI1 记为 UI2/体验版或 UI3/真机 PASS。

## Decisions

### D1. 用单向的 `catalogApi → catalogStore → Page` 路径替换 public mock 读取

`app.globalData.apiBaseUrl` 只新增 UI1 本地默认值 `http://127.0.0.1:8080`。`utils/catalogApi.js` 只暴露 list/detail 两个 Promise 函数，并只发送匿名 GET：

- `${apiBaseUrl}/api/v1/catalog`
- `${apiBaseUrl}/api/v1/catalog/products/${encodeURIComponent(idString)}`

请求不带身份、日期、餐段、价格、库存或自定义降级参数。HTTP 200 交给 store 校验；detail HTTP 404 映射为可识别的 `PRODUCT_NOT_FOUND`，网络失败、503、其他非 200 或畸形 200 统一映射为非敏感的 catalog unavailable 错误。API 层不重试、不缓存、不读取 `utils/data.js`。

`utils/catalogStore.js` 只复制 provider 冻结字段并构造新的普通对象：category 为 `id/name/products`，product 为 `id/category_id/name/description/specification/price_cents`。id/category_id 必须是规范正十进制 string，`price_cents` 必须是非负 safe integer，文本字段必须是 string；额外字段不进入 public view/store。store 不排序、不补字段、不回退 mock，因而服务端顺序原样成为渲染顺序。选择该窄层而不是改造二期 `utils/api.js`，避免把真实匿名目录混入仍依赖 `globalData.menu` 的 admin/P0 mock 契约。

### D2. list 状态由响应的 `categories` 唯一决定

首页和菜单只在实际 `onShow` lifecycle 调用 list；retry action 再调用同一加载函数。每次请求前先设置 `listState=loading` 并清除旧错误；成功后：

- `categories.length === 0` → `empty`；
- `categories.length > 0` → `ready`，即使所有分类的 `products` 都是 `[]`；
- transport/HTTP/schema 失败 → `error`，保留显式 retry action，且不展示任何 mock 商品。

菜单将 server category 直接映射为 group，使用 category string id 做 key/active anchor，空 products 仍渲染分类与“暂无目录商品”。首页按 category 顺序、再按各自 products 顺序 flatten，取常量前四件作为招牌；categories 非空但无商品时仍是 ready，只显示“暂无招牌商品”。不再按 tag/sold/status 筛选，也不再出现 `p001`/`p005` 等假 product ID。静态营销内容不得携带 catalog 商品 ID、名称或价格；商品卡只展示 provider 字段和由 cents 生成的价格文本。

### D3. detail 只按原始 string ID 请求并显式区分 404

detail 在实际 `onLoad(opts)` 保存 `String(opts.id)`，立即进入 `detailState=loading` 并调用详情 API；retry 使用同一未数值化 ID。全链路禁止 `Number(id)`、一元 `+` 或 parseInt，因此大于 `Number.MAX_SAFE_INTEGER` 的合法 uint64 十进制 ID 不丢精度。

- 200 且 DTO 合法 → `ready` 与 canonical product；
- HTTP 404 → `not_found`，product 保持 `null`；
- 网络失败、503、其他非 200 或畸形 200 → `error` 与 retry；
- 任一失败都不读取列表、不回退第一件、不构造假商品。

详情只显示 name、description、specification 与整数分价格；无图片、tags、sales、sold/status、allergens 或可购胶囊。底部动作只表述“加入本地选择”，不把目录可见性声明为可购买。

### D4. cart entry 固化首次选择的 canonical product snapshot

现有 `utils/util.js` 的 public `cart` 仍使用 `globalData.cart`，但 entry 收敛为：

```text
{
  product: { id, category_id, name, description, specification, price_cents },
  qty,
  flavors,
  note
}
```

首次加入某 ID 时深复制当次 list/detail response 的 canonical product；同一 entry 后续只改变 qty/flavors/note，直到移除或 clear，不用后来的目录刷新隐式改写已选快照。`cart.list()` 返回临时 view：包含快照副本、由整数分生成的 `price_text/line_total_text`，以及只供现有 promo/pay 调用签名使用的兼容 `price` 元值；这些派生字段不写回 entry/store，不是价格事实。兼容元值先由精确 cents 文本确定性生成，既有 promo 的 round-to-cent 只作为 non-regression，`cart.totalCents()` 与 confirm 商品合计仍只做 safe integer 加法/乘法。

确认页 `onLoad` 继续执行现有 `loadPromo`，`recalc/openCoupon/pay` handler 与 WXML 入口也保留；只把 `cart.list()` 的 snapshot-derived view 作为 items。confirm 商品名、string ID、单价、数量、小计和合计都来自快照/cents；编辑口味/备注从 entry.product 取得商品，不调用 `data.itemById`，页面不发 catalog request。受控 all-scope promo 必须能渲染且不因新 item shape 抛错；pay 必须继续产生既有 P0 mock order tuple 与导航，其中 ID/兼容价格来自 snapshot-derived view。即使原 response 被修改、后续 catalog 失败或 `globalData.menu` 读取被设置为抛错，以上 non-regression 仍成立。

`utils/api.js`、`utils/promo.js` 与 result/order/payment/history 页面全部 byte-unchanged。catalog DTO 没有 category name，且本 change 不扩充 snapshot，因此 category/item-scoped mock coupon 的适用语义不在本 change 验证；现有 mock order/pay 也只证明入口未崩溃，不构成真实 promo、订单或支付 PASS。

### D5. 页面只呈现目录事实与可恢复边界

home/menu/detail/confirm 的商品区域移除 `imageph`、mock image、tags、sales、stock、sold/status 与 availability 文案。无图片时使用不承载业务事实的文字占位。菜单和详情的动作是本地选择；确认页显示“目录快照不锁价/库存，真实结算需后续服务端校验”的边界，不把当前 mock submit/pay 计入 acceptance。

现有 `globalData.menu` 初始化、`utils/data.js`、`utils/api.js`、`utils/promo.js`、admin/result/order/payment/history 页面与其他 P0 mock 保持原样；本 change 只保证四个 public 页面执行目录/商品/确认路径时不调用 `menuList/itemById` 获取商品事实。confirm 保留的 promo/pay 调用只消费 snapshot-derived item view 与现有非商品 mock 状态。

### D6. UI1 harness 加载真实页面，而不是复制页面控制器

`apps/wechat-miniprogram/tests/page-harness.js` 提供局部模拟：注册 `App/Page/Behavior`，合并 behavior data/lifecycle，实现必要的 `setData`、selector query、component stub 与 navigation stub，并用可控队列实现异步 `wx.request`。测试从仓库路径 `require` 实际 app/page/store/api/cart 文件，调用真实 `onLoad/onShow/retry/add/edit/confirm` handlers；不在测试中复制页面状态机。

`catalog-ui1.test.js` 同时读取实际 WXML，断言 loading/empty/error/not_found/ready 区块与 retry binding 存在。fixture 覆盖：首请求失败后 retry 成功、空目录、启用空分类、404、503/网络失败、大于 JS 安全整数的 string ID、整数 cents、稳定 server order、无 mock fallback、cart snapshot 与 confirm 零请求/零 mock-menu read。另以受控 all-scope coupon 运行真实 confirm `onLoad/loadPromo/recalc/openCoupon`，并运行真实 `pay` handler，断言现有 mock order tuple/导航继续工作且 item 来自 snapshot-derived view。只有 `wx.request` transport、微信宿主 API 和既有 mock 状态是 fixture；production catalog client/store/page/cart 与 confirm handlers 均真实执行。

局部 `package.json` 只声明 `node --test tests/catalog-ui1.test.js`，lockfile 不含第三方 package。该 harness 是本仓库冻结的非真实平台模拟器，因此最多证明 UI1，不能证明微信开发者工具、体验版、合法域名或真机行为。

### D7. Red 必须命中 legacy 的两个真实行为

测试先提供一个不依赖新 `catalogApi/catalogStore` 顶层 import 的 focused legacy boundary：

1. 实际运行 legacy home/menu list lifecycle，期望发起 catalog request；当前 decisive result 必须是 request count `0`。
2. 实际运行 legacy detail unknown ID，期望不设置商品；当前 decisive result 必须是 fallback product `p001`。

writer 在业务实现前运行该 focused pattern 并保留两项失败。Green/Refactor 重跑同一 pattern，再运行完整 UI1。verifier 计划在系统临时目录从 base `94e04bf…` 解包小程序，只覆盖 candidate 的 `tests/page-harness.js` 与 `tests/catalog-ui1.test.js`，重放同一 focused pattern；这样 legacy Red 由旧页面行为触发，不会因新模块缺失而提前终止。随后 verifier 只在 clean detached exact candidate worktree 运行完整 Green Gate。

### D8. 依赖、ownership、回退和失效条件保持单一

server catalog 已在本 base 的 local main 集成，apply 无未满足代码依赖。本 writer 只拥有 proposal 列出的 change、public page、catalog API/store、cart、customize 与局部 npm/test paths；不拥有任何 Go、admin、canonical spec、root tooling 或外部系统。实现与其他无重叠 change 可并行，但这些 owned paths 同时只允许本 writer。

回退为对本 change 单一 candidate commit 的整体回退：public 页面回到 P0 mock，不迁移或清理数据；`globalData.menu` 与 admin mock 因未改而不需恢复。任一实现、proposal/design/spec/tasks、验收命令、base、依赖、candidate SHA、Node/微信 runner 条件或 main 推进都会使旧 writer/verifier 证据失效；返回本 worktree 同步后从 strict、RGR 和完整 Gate 重跑。

## Risks / Trade-offs

- [API 可达但 DTO 漂移] → store 按冻结类型校验并进入可重试 error，不猜测字段或回退 mock。
- [首页 categories 非空但无商品看起来像空] → 保持规范上的 ready，并在招牌区域显示非错误的“暂无招牌商品”。
- [本地快照价格过时] → 页面明确不锁价/库存；本 change 不把本地合计当作 server quote 或真实订单价格。
- [现有 promo 接受元价格且 category/item coupon 依赖 legacy 商品语义] → cart 只临时派生兼容元值并保持 snapshot/cents 为唯一商品金额事实；UI1 只覆盖 all-scope non-regression，category/item coupon 适用性与真实支付明确未验证，且不修改 `utils/promo.js`/`utils/api.js`。
- [Node harness 与微信宿主存在差异] → 只声明 UI1；UI2/UI3 分别保持 `BLOCKED_EXTERNAL` 并记录恢复资产。
- [旧 admin/mock 仍能改 `globalData.menu`] → public catalog store 与 cart snapshot 不读取该对象；legacy 路径继续保留给未迁移页面。

## Migration Plan

1. 在批准后的 writer worktree 先新增 harness/tests，运行 focused legacy boundary，确认 Red 为 `0 request` 与 `p001 fallback`。
2. 最小实现 API/store/list/detail/cart snapshot/confirm cents 与 snapshot-derived compatibility view，使 focused、完整 UI1 及现有 confirm promo/pay non-regression Green。
3. 整理页面结构与样式但不改变 spec，重跑同一 focused、完整 UI1、provider regression、JS/JSON static、strict 与 owned/sensitive Gate。
4. 只提交 owned paths，产生一个本地中文 CANDIDATE commit；exact SHA 通过主动消息外部记录，writer worktree/index clean。
5. 独立 verifier 先隔离重现 base Red，再在另一 clean detached exact-SHA worktree 从头运行全部 Green Gate。任何差异返回原 writer 并产生新 SHA。
6. 本 change 不 push/PR/deploy；只有另行授权且 independent PASS 后才可进入 integration。UI2/UI3 仍不随本地 candidate 解除。

回退只需整体回退该单一 candidate；没有 migration、持久化写入或外部清理。

## Open Questions

无。影响业务、公共契约、数据结果、授权和验收的决策已经由冻结输入、canonical provider spec 与本设计收敛；UI2/UI3 缺失资产是已登记外部阻塞，不是实现未决项。
