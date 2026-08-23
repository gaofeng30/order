# Order MVP canonical PRD 验收矩阵

唯一产品基线：[online-ordering-system-prd-0818.md](../product/online-ordering-system-prd-0818.md)。冻结设计：[order-mvp-domain-schema-interfaces.md](../architecture/order-mvp-domain-schema-interfaces.md)。本矩阵只定义验收闭环，不把模块测试、mock、设计或 `BLOCKED_EXTERNAL` 冒充完整提测。

证据等级：`L1` 单元/纯规则；`L2` HTTP + fresh MySQL；`L3` UI1 + fake-provider 本地 E2E；`L4` 微信 DevTools 真机/真实支付资产。当前所有行均未绑定最终 Candidate SHA，故状态只能是 `NOT_RUN`；L4 另列 `BLOCKED_EXTERNAL`，不阻塞可替代的 L1–L3 本地证据。

每行固定列：CaseID、角色、UI 操作、HTTP、MySQL 事实、预期、失败保护、证据等级、状态。

## A. 25 个活动页面（U9 / M4 / PC12）

| CaseID | 角色 | UI 操作 | HTTP | MySQL 事实 | 预期 | 失败保护 | 证据等级 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| PAGE-U01 | 用户 | 身份选择；进入员工/访客；已绑定商户可切商户 | GET `/me/identity`; GET `/storefront/settings` | users、staff、merchant_accounts、storefront | 只展示服务端可用身份与开屏层 | 未知身份不猜；不提前请求手机号 | L3; L4 UI3 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-U02 | 用户 | 首页看门店/公告/进行中单；点菜单、订单、取餐码 | GET `/storefront/settings`; GET `/orders?active=true` | storefront、service_dates、orders | 服务端门店事实；进行中条准确 | 读失败不显示可下单/伪订单 | L3 | NOT_RUN |
| PAGE-U03 | 用户 | 选今天/明天与离散时间；搜索；加购 | GET `/menu/pickup-options`; GET `/menu?date=&time=` | service_dates、meal_periods、products、soldout、discount | 餐段过滤；截单折叠；员工价正确 | 缺日期事实/售罄未知即不可加购 | L3 | NOT_RUN |
| PAGE-U04 | 用户 | 看图片、说明、只读规格、口味与价格 | GET `/catalog/products/:id?date=&time=` | products.images_json/specification、soldout、staff | 强制 date/time；0–3 图；spec 只读 | 缺 date/time 400；当前事实不可读 503 | L3 | NOT_RUN |
| PAGE-U05 | 用户 | 编辑联系人/附加号、时间、口味、备注；提交 Quote | POST `/me/bind-phone`; POST `/me/extra-phone`; POST `/quotes` | users、staff、quotes、quote_items | 服务端手机号快照与算价；封面 key 入 digest | 未绑定/截单/漂移不支付且购物车保留 | L2+L3; L4 phone | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-U06 | 用户 | 调起支付；等待确认；请求待取餐订阅 | POST `/orders/prepay`; POST `/orders/confirm`; POST `/orders/:id/subscriptions` | prepayments、observations、orders、consents | 仅服务端确认支付才出现订单 | 客户端 success 不建单；未知仅 loading | L3 fake; L4 real pay | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-U07 | 用户 | 看订单/取餐码；补订阅；规则内取消 | GET `/orders/:id`; POST `/orders/:id/cancel`; POST `/orders/:id/subscriptions` | orders、refunds、consents/outbox | QR 仅待取餐；取消进入退款中 | token 不明文落库；重复/非法转换拒绝 | L3; L4 subscription | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-U08 | 用户 | 按六态筛选订单并翻页 | GET `/orders?state=&after_id=&limit=` | orders | 只显示六态与 owner 数据 | 无待支付/取消/异常伪状态；越权统一 404 | L3 | NOT_RUN |
| PAGE-U09 | 用户 | 看/绑主手机号；设附加号；商户登录；切身份 | GET `/me/identity`; POST `/me/bind-phone`; POST `/me/extra-phone`; POST `/me/merchant-login` | users、staff、merchant_accounts、audit | masked PII；身份实时重算 | 客户端手机号/角色不可信；失败不授权 | L2+L3; L4 phone | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-M02 | 商户 | 看四泳道/搜索；切营业状态；返回用户身份 | GET `/merchant/orders`; PUT `/merchant/store-status` | accounts、orders、storefront、audit | live RBAC；订单与状态同源 | 子账号仅现场动作；写失败不假成功 | L3 | NOT_RUN |
| PAGE-M03 | 商户 | 看订单详情；标记备好 | GET `/merchant/orders/:id`; POST `/merchant/orders/:id/ready` | orders、outbox、audit | 仅 PREPARING→READY；生成加密 token | 非法态 409；通知失败不回滚订单 | L3 | NOT_RUN |
| PAGE-M04 | 商户 | 扫 QR；输当日 4 位码；跨日逐单核销 | POST `/verify/scan`; POST `/verify/code`; POST `/merchant/orders/:id/redeem` | orders token hash/ciphertext、audit | 核销一次；跨日同号不误命中 | 非 READY/退款单拒绝；token 不进日志 | L3; L4 camera | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-M05 | 商户 | 对商品切今日可售/售罄 | PUT `/merchant/products/:id/soldout` | product_sold_out_dates、audit | 只影响指定日期，次日自然恢复 | 读写未知按售罄；不建库存 | L3 | NOT_RUN |
| PAGE-PC01 | 主账号 | 看今日/月指标、待制作、销量 | GET `/admin/stats` | orders、order_items、refunds | 实时派生，无汇总表 | 查询绝不改交易；未取餐不算完成/有效营收 | L3 | NOT_RUN |
| PAGE-PC02 | 主账号 | 搜索/筛选订单；退款；处理未取餐 | GET `/admin/orders`; POST `/admin/orders/:id/refund` | orders、refunds、observations、audit | 六态/日期/号码/手机号筛选；全额退款 | provider 未确认保持 REFUNDING | L3 fake; L4 real refund | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-PC03 | 主账号 | 看支付/退款/汇总；下载 CSV；核对微信账单 | GET `/admin/finance/payments`; `/refunds`; `/summary`; `/export` | orders、prepayments、observations、refunds、audit | 逐笔派生账单与单边账 | 账单不可用不宣称已对账；无财务汇总表 | L2+L3; L4 bill | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-PC04 | 主账号 | 查看支付待处理；人工建单或退款 | GET `/admin/pending-payments`; POST `/admin/pending-payments/:id` | prepayments、payment_observations、orders/refunds、audit | 仅 trusted payment；动作幂等 | snapshot 损坏继续 shield；不造异常订单 | L3 fake | NOT_RUN |
| PAGE-PC05 | 主账号 | 商品增删改/上下架/图片/上移下移/日期售罄 | `/admin/products`; PUT `/admin/products/order`; POST `/upload` | products.images_json、soldout、audit | 最多 3 图；完整分类内顺序；spec 不可改 | FK 阻止悬挂分类；上传失败不写商品 | L3; L4 OOS | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-PC06 | 主账号 | 分类增删改、启停、上移下移 | `/admin/categories`; PUT `/admin/categories/order` | categories、products FK、audit | 顺序落库并联动商品编辑 | 有商品删除 409；同名 key 拒绝 | L3 | NOT_RUN |
| PAGE-PC07 | 主账号 | 配门店、两餐段、今天/明天营业日期 | GET/PUT `/admin/settings`; `/admin/meal-periods` | storefront、meal_periods、service_dates、audit | 单店/单点/午晚餐；缺日期行 closed | 第三餐段/第二取餐点/非法时间拒绝 | L3 | NOT_RUN |
| PAGE-PC08 | 主账号 | 上传/定位/启停开屏 PNG | GET/PUT `/admin/launch-layer`; POST `/upload` | storefront launch object key/geometry、audit | 服务端下发且换设备可见 | 旧 URL 清空；对象不可读不展示 | L3; L4 OOS | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-PC09 | 主账号 | 设全局折扣；员工手工 CRUD/停启/统计 | GET/PUT `/admin/discount-rate`; `/admin/staff-whitelist` | staff、discount_settings、orders derived、audit | phone+name 双要素；1..100 | PII masked；停用/漂移重新校验 | L3 | NOT_RUN |
| PAGE-PC10 | 主账号 | 商户账号增删改/停启/角色；PC 扫码登录 | `/admin/merchant-accounts`; `/admin/auth/qrcode`; `/admin/auth/poll` | accounts、pc_sessions、audit | 2m QR；12h absolute；并发不互踢 | 最后 OWNER 保护；无 refresh/remember | L3; L4 scan | NOT_RUN; L4 BLOCKED_EXTERNAL |
| PAGE-PC11 | 主账号 | 下载模板；上传菜品 xlsx；预览并提交 | POST `/admin/products/import/preview`; POST `/import/commit` | import_batches、categories、products、audit | <=10MiB/500 行；新分类一次创建 | 未确认零写；重复 token 只一次 | L3 | NOT_RUN |
| PAGE-PC12 | 主账号 | 下载模板；上传员工 xlsx；预览并提交 | POST `/admin/staff-whitelist/import/preview`; POST `/import/commit` | import_batches、staff、discount_settings、audit | <=10MiB/5000 行；按手机号覆盖 | 文件内重复 phone 异常；不重启停用项 | L3 | NOT_RUN |

## B. PRD §14 验收标准（19）

| CaseID | 角色 | UI 操作 | HTTP | MySQL 事实 | 预期 | 失败保护 | 证据等级 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| AC-01 | 用户 | 启动浏览，首次提交才授权手机 | GET storefront/menu; POST bind/quotes | users/sessions/quotes | 启动零手机号弹窗；提交前可信绑定 | 拒绝授权不写入/不支付 | L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| AC-02 | 用户 | 填附加手机号+姓名 | POST `/me/extra-phone` | users、staff | 双要素同时命中才员工 | 只中手机号按访客 | L2+L3 | NOT_RUN |
| AC-03 | 用户 | 选择今天/明天时间 | GET `/menu/pickup-options` | dates、meal_periods | 截单整组折叠；今天全截默认明天 | 日期事实缺失则今天/明天不可选 | L3 | NOT_RUN |
| AC-04 | 用户 | 浏览菜单 | GET `/menu?date=&time=` | products、staff、soldout | 餐段过滤；绑定前原价、员工展示双价 | 身份未知不展示员工价 | L3 | NOT_RUN |
| AC-05 | 用户 | 比对菜单/详情/结算单价 | GET menu/detail; POST quotes | products、quote_items | 每项员工价一致且整数分 | 客户端价被忽略 | L2+L3 | NOT_RUN |
| AC-06 | 用户 | 取消/失败/成功支付 | prepay/confirm/notify | prepayments、observations、orders、sequence | 仅 confirmed success 建单/分号 | cancel/fail 不建单不烧号 | L3 fake; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| AC-07 | 系统/主账号 | 丢 callback 后跑 query | worker; pending routes | prepayments、observations、orders | 已付自动补单，失败进 pending | Query/Apply SQL 错保留 durable 事实 | L3 fake | NOT_RUN |
| AC-08 | 用户 | 截单边界提交 | POST quotes/prepay | quote、current store facts | 支付拦截并回时间选择，购物车保留 | 零 provider Create/零清购物车 | L3 | NOT_RUN |
| AC-09 | 系统/用户 | 观察 30m 自动排产/近时支付 | worker; GET order | orders | 准时 PREPARING；近时新单直接 PREPARING | 漏跑可补偿、无手工提前开做 | L2+L3 | NOT_RUN |
| AC-10 | 三端 | 触发合法/非法六态转换 | fulfillment/refund/order routes | orders | 仅六态单向、无撤销 | 非法 409；无待支付/取消/异常态 | L2+L3 | NOT_RUN |
| AC-11 | 商户/用户 | READY 展码并重复核销 | GET order; POST verify | orders token columns、audit | READY 才有 QR；重复不重复统计 | ciphertext 清除、hash 保留防重放 | L3 | NOT_RUN |
| AC-12 | 商户 | 输入跨日同号 | POST `/verify/code` | orders unique(date,number) | 只核销当前营业日期 | 昨日同号不匹配 | L2+L3 | NOT_RUN |
| AC-13 | 商户/用户 | 今日售罄后选明天 | soldout; menu | soldout date PK | 今日受影响、明日可售 | 不产生库存/跨日状态 | L3 | NOT_RUN |
| AC-14 | 用户/主账号 | 自助取消、后台退款、模拟退款未知 | cancel/refund/notify | refunds、refund observations、orders | 规则内全额退款；未知停 REFUNDING | provider accepted 不等于 REFUNDED | L3 fake; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| AC-15 | 用户 | 支付页/取消弹窗订阅；拒绝后补订阅 | POST subscriptions | consents、outbox | 只有 READY/REFUND_RESULT 两类 | 拒绝/发送失败不改订单 | L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| AC-16 | 主/子账号 | 尝试各权限及 PC 扫码 | merchant/admin auth routes | accounts、pc_sessions、audit | server RBAC；PC 仅 OWNER | client guard 不授权；最后 owner 保护 | L2+L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| AC-17 | 主账号/用户 | 改分类/开屏后换设备看 | admin config; public reads | categories/products/storefront | 改动落同一 MySQL 并三端可见 | 不读 window/globalData mock | L3 | NOT_RUN |
| AC-18 | 主账号 | 查未取餐与营收 | GET admin orders/stats | READY orders、refunds | 未取餐可筛且不算完成/有效营收 | 不引入第七状态 | L2+L3 | NOT_RUN |
| AC-19 | 验证者 | 执行各阶段 Gate 并查台账 | n/a | sanitized audit/evidence | Gate 仅对应阶段生效，无敏感数据 | BLOCKED_EXTERNAL 不冒充本地 PASS | L1+L2+L3+L4 | NOT_RUN |

## C. PRD §15.8 边界与异常（35）

| CaseID | 角色 | UI 操作 | HTTP | MySQL 事实 | 预期 | 失败保护 | 证据等级 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BE-01 | 用户 | 休息/全截时浏览并尝试下单 | storefront/menu/quotes | storefront、dates | 仅浏览，入口禁用 | 不建 quote/prepay | L3 | NOT_RUN |
| BE-02 | 用户 | 选已截餐段 | pickup-options | meal_periods | 整组折叠；今天全截默认明天 | 不返回伪可用时间 | L3 | NOT_RUN |
| BE-03 | 用户 | 看售罄/下架与已加购项 | menu/detail/quotes | products、soldout | 售罄可见禁用；下架不见；提交重验 | 不按客户端购物车绕过 | L3 | NOT_RUN |
| BE-04 | 用户 | 选择不匹配餐段商品 | menu/detail/quotes | products.meal | 用户端不展示，商户可见 | Quote 拒绝不匹配项 | L2+L3 | NOT_RUN |
| BE-05 | 用户 | 截单瞬间提交 | quotes/prepay | quote/current meal facts | 返回时间选择、保留购物车 | 零 provider call | L3 | NOT_RUN |
| BE-06 | 用户 | Quote 后变价/售罄 | prepay | quote digest、products/soldout | 要求重新确认/报价 | 未确认不支付；折扣单漂移例外 | L2+L3 | NOT_RUN |
| BE-07 | 用户 | 取消/失败支付后重试 | prepay/confirm | prepayments、orders | 不建单，停确认页可重试 | Create 不重复；歧义仅 Query | L3 fake | NOT_RUN |
| BE-08 | 用户 | 支付结果未确认 | confirm | prepayments、observations | 临时 loading | 不建订单/异常态 | L3 fake | NOT_RUN |
| BE-09 | 系统/主账号 | 有支付无订单 | reconcile/pending routes | observations、prepayments | 自动补建或进 pending | 外部事实不丢、不造异常单 | L3 fake | NOT_RUN |
| BE-10 | 用户 | 支付时不足 30m | confirm/get order | orders timestamps | 新单即 PREPARING，无取消 | InitialState 后 Advance 同 tx | L2+L3 | NOT_RUN |
| BE-11 | 系统 | 跳过一次排产 worker 再补跑 | worker | orders | 补偿到 PREPARING | 幂等、无回退 | L2 | NOT_RUN |
| BE-12 | 用户 | >30m 自助取消 | cancel | order/refund | REFUNDING 并请求订阅 | 全额一次；provider 外调在 tx 外 | L3 fake | NOT_RUN |
| BE-13 | 用户 | <=30m 尝试取消 | cancel | orders | 无入口/提示联系商户 | 服务端仍 409 | L3 | NOT_RUN |
| BE-14 | 主账号/系统 | 退款受理但未知 | refund/query | refunds、observations、orders | 保持 REFUNDING + pending | 不进 REFUNDED/异常态 | L3 fake | NOT_RUN |
| BE-15 | 用户 | RESERVED/PREPARING 看详情 | GET order | orders | 不渲染 QR | token 列全 NULL | L2+L3 | NOT_RUN |
| BE-16 | 商户 | 重复核销完成单 | verify | orders、audit | 提示已核销、不重复计数 | hash 防重放 | L2+L3 | NOT_RUN |
| BE-17 | 商户 | 扫已退款单 | verify | orders | 提示已退款并拒绝 | token ciphertext 已清 | L2+L3 | NOT_RUN |
| BE-18 | 商户 | 输入跨日同号 | verify code | orders unique(date,number) | 仅当前日 READY 命中 | 昨日订单需列表逐单处理 | L2+L3 | NOT_RUN |
| BE-19 | 主账号 | 营业日后筛未取餐并退款/核销 | admin orders/refund/redeem | READY orders | 状态不自动变；可后处理 | 不计完成/有效营收 | L3 | NOT_RUN |
| BE-20 | 商户 | 今日标售罄再查明日 | soldout/menu | soldout date PK | 只今日售罄 | 次日无复制行 | L3 | NOT_RUN |
| BE-21 | 用户 | 拒绝待取餐订阅 | subscriptions/home/order | consents | 首页条与取餐码页补入口 | 拒绝不写 outbox/不改订单 | L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| BE-22 | 用户 | 未绑手机号点结算并返回 | bind-phone/quotes | users | 绑定后回结算继续 | 拒绝/失败保留购物车 | L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| BE-23 | 用户 | 附加 phone 命中但姓名错 | extra-phone/identity | users、staff | 不授员工，提示核对 | byte-exact name_key 双要素 | L2+L3 | NOT_RUN |
| BE-24 | 访客 | 进入确认页 | quotes | quote identity snapshot | 显示访客原价，无折扣行 | 不伪造 100% 员工折扣 | L3 | NOT_RUN |
| BE-25 | 用户 | 空购物车点结算 | none | 无写入 | 按钮禁用 | 零请求/零 quote | L3 | NOT_RUN |
| BE-26 | 用户 | 无 READY 单点取餐码 | GET orders | orders | Toast 无待取餐订单 | 不显示旧 token | L3 | NOT_RUN |
| BE-27 | 主账号 | 上传非 xlsx | import preview | 无 batch | 整体拒绝 | 不猜格式、不上传业务数据 | L2+L3 | NOT_RUN |
| BE-28 | 主账号 | 上传缺必填表头 xlsx | import preview | import batch rejected/无业务写 | 文件级异常，不逐行 | 零 category/product/staff 写入 | L2+L3 | NOT_RUN |
| BE-29 | 主账号 | 超 10MiB/500/5000 行 | import preview | 无或 rejected batch | 中止并提示分批 | DB CHECK + streaming body limit | L2+L3 | NOT_RUN |
| BE-30 | 主账号 | 菜品导入命中同名 | preview/commit | products.name_key、batch | 行异常并跳过，不覆盖 | name_key unique | L2+L3 | NOT_RUN |
| BE-31 | 主账号 | 菜品分类不存在 | preview/commit | categories、products、batch | 分类一次新建、末尾、启用 | 同文件同名只一条 | L2+L3 | NOT_RUN |
| BE-32 | 主账号 | 员工文件内 phone 重复 | import preview | batch | 异常，不取最后一条 | commit 不写该冲突 | L2+L3 | NOT_RUN |
| BE-33 | 主账号 | 重复提交同 preview token | import commit | batch unique idempotency | 只生效一次并重放结果 | 不重复版本/审计 | L2+L3 | NOT_RUN |
| BE-34 | 用户 | 浏览无图商品 | menu/detail | products images_json=[] | 品类渐变+宋体首字占位 | 无 image 表/假 URL | L3 | NOT_RUN |
| BE-35 | 用户 | 浏览单图商品并预览 | detail | images_json length=1 | 无计数/滑动，允许全屏 | 不访问第二图 | L3 | NOT_RUN |

## D. PRD §12 正式业务不变量（16）

| CaseID | 角色 | UI 操作 | HTTP | MySQL 事实 | 预期 | 失败保护 | 证据等级 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| INV-01 | 全部 | 选择履约方式/门店 | storefront/pickup | singleton storefront | 单门店单取餐点，仅预约自提 | 拒绝配送/即时/第二取餐点 | L1+L2+L3 | NOT_RUN |
| INV-02 | 用户 | 选择日期餐段 | pickup-options | dates、meal_periods | 仅今天/明天、午/晚、上海时区 | 其他日期/餐段拒绝 | L1+L2+L3 | NOT_RUN |
| INV-03 | 商户/用户 | 上下架/日期售罄 | catalog/soldout | products、soldout | 无数量库存/预占/扣减 | 不建库存表/返还逻辑 | L1+L2 | NOT_RUN |
| INV-04 | 用户 | 选择取餐点 | pickup-options | meal cutoff/start/end/interval | 每餐段单 cutoff，离散约定时刻 | 时间窗口/越界时刻拒绝 | L1+L2+L3 | NOT_RUN |
| INV-05 | 用户/系统 | 支付各结果 | prepay/confirm/notify | prepayment/observation/order | confirmed success 才建单 | prepayment 不是 order | L2+L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| INV-06 | 系统/主账号 | 处理有支付无订单 | reconcile/pending | prepayment/observation | 补建或人工，不加异常态 | durable-first，SQL 错重试 | L2+L3 | NOT_RUN |
| INV-07 | 三端 | 推进/查看订单状态 | order/fulfillment/refund | orders exact CHECK | 仅六态单向，无撤销 | 非法状态/转换 DB+Module 拒绝 | L1+L2+L3 | NOT_RUN |
| INV-08 | 系统/商户 | 等待 30m / 尝试提前开做 | production worker | orders | 30m 自动制作，无接单/手动提前 | worker 漏跑补偿 | L1+L2 | NOT_RUN |
| INV-09 | 用户 | >/<30m 取消 | cancel | orders/refunds | 仅 RESERVED 且 >30m 自助取消 | 边界等于 30m 拒绝 | L1+L2+L3 | NOT_RUN |
| INV-10 | 用户/主账号 | 退款并观察未知/成功 | refund ingress/query | refunds/observations/orders | 全额；confirmed success 才 REFUNDED | 未知停 REFUNDING；无库存返还 | L1+L2+L3 | NOT_RUN |
| INV-11 | 用户 | 浏览、提交、附加号识别 | session/phone/quotes | users/staff/quote snapshots | 浏览免手机号；提交前可信主号；附加双要素 | 客户端 phone/name 不授信 | L2+L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| INV-12 | 用户 | 比对商品级折扣与总额 | menu/detail/quotes | discount/quote items | 原价→全局员工折扣→应付；逐项 half-up；整数分 | rate 非 1..100、payable<1、溢出拒绝 | L1+L2 | NOT_RUN |
| INV-13 | 商户/主账号 | 尝试越权/PC 登录 | merchant/admin routes | accounts/pc_sessions/audit | OWNER/SUBACCOUNT 两角色；服务端权限 | 子账号 PC 403；客户端角色无效 | L2+L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| INV-14 | 系统/商户 | 并发建单/跨日核销 | confirm/verify | pickup_sequences/orders | 按日期 0001..9999；同单幂等；手码限当日 | 同 tx 不烧号；9999 后 fail closed | L2+L3 | NOT_RUN |
| INV-15 | 用户/系统 | 授权/发送订阅 | subscriptions/outbox worker | consents/outbox | 仅待取餐与退款结果，主动授权 | 拒绝/失败不改订单 | L2+L3; L4 | NOT_RUN; L4 BLOCKED_EXTERNAL |
| INV-16 | 验证者 | 注入 mock/客户端 success/简化状态 | all | authoritative MySQL facts | 仅服务端 MySQL/provider observation 是事实源 | mock/内存/前端结果不得晋升生产事实 | L1+L2+L3 | NOT_RUN |

## E. 计数与状态规则

- 页面：`PAGE-` 必须 25 行，其中 `PAGE-U` 9、`PAGE-M` 4、`PAGE-PC` 12。
- §14：`AC-` 必须 19 行。
- §15.8：`BE-` 必须 35 行。
- §12：`INV-` 必须 16 行。
- 合计必须 95 行；任何合并只允许共享证据，不允许删除 CaseID。
- 只有绑定最终 Candidate exact SHA 的实测证据才能把单行改为 PASS；L4 未完成时保留 `BLOCKED_EXTERNAL`，不得污染已完成的 L1–L3，也不得据此声称“可提审”。

## F. 文档 TDD 记录

- Red（修改前）：结构检查 exit 1，缺少 acceptance matrix，并缺 AEAD ciphertext、Billing reconcile、detail date/time、product reorder、2m/12h、导入硬上限、bounded VARBINARY、三范围索引、Quote image-key digest 等冻结词。
- Green（当前内容结构，不代表设计 Review PASS）：架构 requirements `15/15`；CaseID `95/95` 且唯一；页面 `25=U9+M4+PC12`、§14 `19`、§15.8 `35`、§12 `16`；每个 case 9 列且状态含 `NOT_RUN`；两个相对链接存在；`git diff --check` 无输出；变更路径仅两份 owned 文档。
- 下一 Gate：Product/Spec 与 Standards/MySQL 双轴只读 Review；任一 finding 修改文档后上述结构证据重跑，旧 review receipt 失效。
