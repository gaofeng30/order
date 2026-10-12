# 共享PG支付完整业务端口

Status: APPROVED_ATOMIC / WRITER_VERIFIED / CANDIDATE_NOT_CREATED。change_id: expand-cloudbase-shared-pg-payment。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## Why

共享PG前序切片已覆盖身份、核销、目录报价及员工导入；支付六方法仍依赖原生事务。必须同时闭合预支付、交易观测、查询恢复、订单生成与独立命令回执，才能保留真实支付时序及崩溃恢复，不能只迁移Prepare后启用部分runtime。

## What Changes

- 保留Go Application六方法/原provider/公开DTO和错误，将原事务映射为15固定普通服务UID INVOKER RPC，registry55→70。
- Prepare同事务Finalize源锁/窗口/96bit随机支付号/创建claim；Observe同事务Normalize/归并；Materialize同事务冻结报价/序号/order/items/状态。无跨HTTP Tx、乐观源刷新、通用SQL或管理credential。
- 保留原独立receipt、时钟/输入/权限优先级、最多两次原类别数据库重试；QUOTE_UNAVAILABLE_RETRY仅编码原quote.ErrUnavailable，不扩大错误类别。provider无底层自动重试。
- 15新增入口成功输出均按原有界合同保留8MiB；前55中24项增长响应策略不变，不新增业务容量限制。

## Ownership and Dependencies

WT: /Users/vivix/.codex/worktrees/order-uat-shared-pg-payment/order；branch: codex/order-uat-shared-pg-payment；base_sha: 5f5af0cd05eafbddfb660631adf6e12daf450a87。
父第五exact 5f5af0cd05eafbddfb660631adf6e12daf450a87已INDEPENDENT_VERIFIED（root生命周期回执；ordinary/race各1679、16包及三轴0finding）；第六已按本地两轴/必要门禁完成后执行限定UAT安装与业务验证，集成依赖另核。前序尚未main集成。

源码相对services/api/：root独占internal/paymentorder/{sharedpg.go,sharedpg_test.go}与common六文件 internal/cloudbasepg/{client.go,client_test.go}、sharedpg/{schema.go,schema_test.go,access.sql,redeem_order.sql}（最后仅readiness投影，原Redeem业务body不变）。payment worker独占 internal/paymentorder/sharedpg_{prepare,observation,materialize,receipts}.go及对应_test.go共8、sharedpg/payment_{prepare,observation,materialize,receipts}.sql共4。Astra独占本change proposal/design/tasks/specs/shared-pg-payment/spec四工件。总24路径；不修改原生业务/原44迁移/其他writer路径，新增fixture需逐路径划分。

## Minimum Success and Scope

六Application方法完整native-v44差分及原handler、provider次数/已知提交时序、16并发、未知ack/进程恢复、晚步故障事务事实、源锁与lease时钟、当前权限、70函数完整readiness与普通UAT合成provider矩阵/清理，随后Writer Gate/exact candidate detached独验。registry缺函数Red不能代替业务行为Red。

不改main/全runtime宿主，不迁移refund、订阅发送或五continuousworkers，不改公共支付规则、AppID/微信provider/Order bearer。真实微信商户材料/验签和手机支付、COS不可变字节+原生版本30天恢复、CloudRun/PC同源/监控备份、95cases25pages/UI3仍独立未闭合。无Prod、采购或新增权限授权；本change不声称完整UAT。

## External Assets and Authority

批准authority：architecture-ablation/sixth-payment-port-abi-draft.md SHA a7bfe1f288252a8b2a2639a1450d6d8430236a02d05b9f8bcfec4bdc1aabe6c7；sixth-payment-approved-abi.json SHA 158d531e61bc6ec6d22087ee55966024efe01bf9f75abf79ba5a03eb0e56ed80。正式协议见design；root外部shared-pg-payment-root持harness/管理artifact/脱敏receipt，leaf外部证据单独归档。

已实测native63PASS及15缺RPC+parent共16FAIL仅baseline/registry层。平台UUID只读能力不等于支付功能PASS。candidate SHA提交后绑定外部生命周期receipt，避免同提交自引用；后续独验状态不回写冻结提交。

当前源码为V4：仅4个payment SQL文件删去15个行尾ASCII空格，Go/原生源码及其余527源文件不变；递归PL/pgSQL词法核验确认token/literal/comment一致，函数体字节和readiness hash确实更新。V3完整ordinary/serialized race各17包1944 PASS保留为历史；V4受影响sharedpg/migrate/paymentorder三个包各模式471 PASS、0 FAIL/SKIP，并完成静态检查。V4实际UAT重新安装14个变化body，完整70函数/44ledger/ordinaryCheck通过，旧29history逐字不变、当前history30，25业务表仍原baseline。V3合成provider完整业务矩阵没有重跑或冒称V4执行。候选未创建、exact独验和main未完成；真实微信/CloudRun/95/25/UI3仍未通过。

实际执行的versioned SQL与当前V3的16个定义逐一一致；V3完整安装权威文件为external sixth-install-current-v3-verified.sql，SHA fb66810fe9ec16ace25a2bea0d149e9897a168cb081ce3e560fa7a726bc9a0fd。旧sixth-install-full.sql是从未用于Execute的V2草稿，明确SUPERSEDED，不能用于最终源比较。

V4权威是source-freeze-v4.json及20261009091500受控format迁移，SHA/结果见tasks；V3 bundle仅用于历史来源，不用于当前V4源体比对。最终staged diff曾实际发现15尾空白并阻止候选（此前普通diff漏掉未跟踪SQL）；修正不降低Git gate，候选前仍须重新owned-only暂存并通过检查。
