# 共享 PG 目录、店铺配置与报价闭合

Status: APPROVED_QUOTE_SQL_ATOMIC / IMPLEMENTING。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。本地必需ordinary/race各1296 PASS，第四普通身份业务矩阵、权限与清理已完成；最终Root Writer Gate已完成；候选与独验尚未完成。平台为真实UAT PG，Go原HTTP在loopback；不是CloudRun全部署。

## Why

前三切片已提供共享PG核销、查询与身份端口，本切片迁移浏览商品、菜单、店铺配置、售罄、展示价格和公开报价所需原生能力。保留原业务与公共接口，为后续完整runtime提供真实端口，不启用部分后台。

## What Changes

- Catalog Reader两方法及Admin三读/完整十kind命令；Menu两Reader；Storefront公共/管理读与两配置命令。
- Merchant SoldOut、展示PricingResolver只读快照，以及读取真实image-confirm receipt的Confirmed依赖；不实现上传或确认Record。
- Quote公开Create/Read完整不可变聚合。报价以两个固定RPC完成Create/Read；实际原型否决跨REST完整proof刷新，改为key与源锁内完整SQL派生、quote/items/receipt同提交。原型证明旧方案活性差异；新方案已完成本地完整差分和本切片平台矩阵，仍待候选独验。
- 普通服务JWT、固定SECURITY INVOKER/RLS/最小列ACL与readiness延续前三切片；原44迁移/ledger/public DTO不变。

## Ownership and Dependencies

base_sha: 22bc009a3a6b278d08d09b8d50d64c31cf5c332f；branch: codex/order-uat-shared-pg-catalog-quote；worktree: /Users/vivix/.codex/worktrees/order-uat-shared-pg-catalog-quote/order。第三精确候选22bc009a3a6b278d08d09b8d50d64c31cf5c332f独验已PASS（Spec/Standards均0 findings、必需八包ordinary/race各897）；前三切片均未main集成。开发可并行，候选/集成依赖分别核验。candidate_sha由提交后外部receipt记录，不自引用。

Astra仅owned本change四工件。业务writer owned以下16个Go/测试文件及6个SQL：

- internal/catalog/{sharedpg.go,sharedpg_test.go,sharedpg_admin.go,sharedpg_admin_test.go}
- internal/menu/{sharedpg.go,sharedpg_test.go}
- internal/storefront/{sharedpg.go,sharedpg_test.go,sharedpg_admin.go,sharedpg_admin_test.go}
- internal/objectstore/{sharedpg_confirmation.go,sharedpg_confirmation_test.go}
- internal/staffdiscount/{sharedpg_pricing.go,sharedpg_pricing_test.go}
- internal/merchantsoldout/{sharedpg.go,sharedpg_test.go}
- sharedpg/{catalog.sql,menu.sql,storefront.sql,image_confirmation.sql,display_pricing.sql,merchant_soldout.sql}

上述路径均在services/api下。root owned internal/quote/{sharedpg.go,sharedpg_source.go,sharedpg_test.go,sharedpg_codec_test.go}的Go/codec/tests；quote_sql_writer仅owned sharedpg/quote.sql。common writer sharedpg_standards_review独占internal/cloudbasepg/{client.go,client_test.go}、sharedpg/{schema.go,schema_test.go,access.sql,redeem_order.sql}和internal/fulfillment/sharedpg_test.go七路径，负责固定registry/readiness事实投影及必要权限fixture（首两核销业务body不改），不担任本切片独立验证者。该writer另独占internal/quote/sharedpg_unicode_generate.go（buildignore生成器），仅生成固定Unicode元数据/SQL局部片段到外部，quote_sql_writer审定并整合输出到quote.sql，root负责Go接口与验收，任何人不同时写该SQL。额外原native helper提取必须先具体划分ownership，不自动允许修改main、payment或import。

## Minimum Success and Boundaries

17非Quote RPC可立即逐纵切真实native Red→最小Green；Quote原子SQL实现与这些端口按ownership并行。整个change候选要求全部19固定业务入口含Quote完整差分与活性门通过，不能用17已通冒充完成。Writer Gate、真实普通JWT业务/权限/故障/清理、候选精确SHA、clean detached独验均各自完成才可进入相应状态。

不迁移import、staffdiscount管理写、独立storestatus、图片上传/Record、付款退款及worker。Quote付款caller-Tx接口明确未实现，不能以独立finalize RPC或sql.Tx shim冒充付款原子性。真实微信/COS版本恢复/5持续worker/CloudRun与PC部署/95Case25页面/UI3仍后续，ICP除外；不采购、不改Prod、不新增公共业务限制、timeout或fallback，不授权push/PR/merge。
