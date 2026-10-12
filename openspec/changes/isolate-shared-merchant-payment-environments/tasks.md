# 实施与验证

gate_type: W3；ui_level_target: UI0；ui_level_actual: UI0。此 change 无 UI diff；实际资金/手机 UI3 和发布验收由 root 单独记录，后端本地证据不替代它们。

- [x] S0 核对 E8、空 index、六准备源及 v4（s0-before.json）。
- [x] S1 首个公开 Prepare cloud 新号及原请求/receipt 时序 Red→Green：s1-payment-red.log 双环境旧号失败，s1-payment-green.log 同 selector 通过。最初 JSONB 文本误断言 TEST_FIXED，不记业务 Red；无必要 Refactor。完整 S1 后续矩阵见下列已完成条目。
- [x] S1 native 两种退款公开入口、原请求及同事务 receipt Red→Green：s1-refund-{red,green}.log；时间时区指针误断言 TEST_FIXED，未变生产时义。
- [x] S1 native URL 和精确编号碰撞 Red→Green；随机失败/16 并发/重启/未知 commit ACK/退款回滚：s1-{payment,refund}-boundary-green.log exit 0。E8 同条件 16 并发复现 11 临时幂等冲突，现源最终单 intent/provider/receipt 收敛，不扩大为全调用必须成功。
- [x] S1 Shared 三入口新号及首次事务/回放 Red→Green：s1-{payment,refund}-shared-{red,green}.log；未知 ACK/late fault 必要相邻 s1-*-adjacent*.log。其后新增 S3 guards 的同源重验仍按下列 S3 跟踪。
- [x] S2 adapter 新增 Details 保留全行、Entries 仍原成功投影：s2-details-{red,green}.log；早期 Entries4 错期待单独 TEST_CONTRACT_SHAPE_FIXED，不作正式 Red。
- [x] S2 native/Shared 全 28,395 行消费者及完整 receipt、THIS_ENV SUCCESS 5,680，原 SQL 5s 未放宽：s2-native-full-consumer.log、s4-inconsistent-evidence-green.log。原严格差异/上海边界/读报表必要回归 s2-s4-billing-affected-regression.log + 单失败 fixture 修复 s2-s4-billing-boundary-fixture-fixed.log。实际云容量 UNVERIFIED。
- [x] S3 worker locator 原 3 参数/8 输入及 normal/version/owner/lease/proof/env：s3-worker-locator-{red,green}.log。
- [x] S3 全局 cutoff 下按精确身份筛确认、旧记录纠正保留三份历史 receipt：s3-confirmation-behavior-red.log → s3-confirmation-green.log。初次夹具漏必需 target hash 为 TEST_FIXED。
- [x] S3 付款 callback 旧号精确确认/错 provider/腐坏请求：s3-legacy-payment-callback-{red,green}.log；Shared 三个公开物化入口环境拒绝 s3-shared-materialize-{red,green}.log。
- [x] S3 退款旧确认/错 provider/损坏 request；父款未知/腐坏证明首次零 intent/receipt；精确审核父款 native/Shared 两入口恢复：s3-legacy-refund-callback-{red,green}.log、s3-refund-parent-clock-fixed-red.log → s3-refund-parent-green.log、s3-reviewed-parent-{red,green}.log。初次 request-order clock 夹具漏绑定为 TEST_FIXED。
- [x] S3 native 已审核旧 callback 续物化：s3-native-legacy-materialize-{red,green}.log；101 行跨页 cutoff=0 漂移零 receipt/非零固定继续：s2-full-consumer-batched-linear.log。
- [x] S3 Shared 已观察旧号续物化：v4 已批准唯一 PAYMENT locator；原公开行为 Red 为 s3-shared-legacy-materialize-behavior-red.log（native 已物化，Shared Materialized=0/Pending=1），同 selector b-legacy-green.log 通过。新增完整观察/无观察/重复同 ID/冲突 ID/不完整观察行为 Red→Green 为 b-locator-final-{red,green}.log；观察、物化、receipt、恢复及并发相邻 ab-payment-affected-ordinary.log exit 0。初轮重复安装 auth schema 是 TEST_FIXED。
- [x] S3 malformed facts / 相关坏确认 / SQL 非 UTF8 source 明确拒绝：s3-facts-wire-negative.log、s3-facts-malformed-confirmation-{red,green}.log；付款退款 S1/S3 组合回归 s1-s3-{payment,refund}-combined-environment.log。
- [x] S4 native/Shared 16 并发单份完整 receipt（s3-confirmation-green.log）；late audit fault 零部分投影/receipt、提交丢 ACK/重启单次投影（s4-atomic-fault-replay.log）。
- [x] S4 同事务原请求摘要复核、native 单连接不死锁、Shared 损坏 request 不 pending：s4-system-proof-{red,green}.log。
- [x] S4 分类/summary/strict 投影内部一致性真实 Red→Green，合法完整大账单仍通过：s4-inconsistent-evidence-{red,green}.log。空差异 receipt 明确 []，公开 CompareBill 形状不变。
- [x] S4 native 跨环境新号不得 pending；Shared 缺/null 金融身份不得保存 receipt；wrong-kind 退款 UNKNOWN 分类与 native 一致：s4-null-identity-other-system-red.log、s4-null-identity-valid-proof-red.log → s4-null-identity-other-system-green.log。初版缺字段仅 ABSENT proof 未暴露缺口，正式 Red 使用 VALID proof；未将通过的首版冒作 Red。
- [x] S4 system-only 实际 proof 的完整留证/身份绑定及更正回放：v4 已批准 `system_facts` 与 `{bill_facts,system_facts}` 摘要；原行为 Red 为 s4-system-proof-identity-red.log，同 selector a-system-proof-green.log 通过。新增系统退款完整付款/退款 proof、更正后新 receipt、回放单 receipt 的行为 Red→Green 为 a-system-proof-final-{red,green}.log；最终源码及必要事务/并发相邻 ab-billing-sourcebound-ordinary.log exit 0。原 request/raw openid 未进入 receipt。
- [x] 当前受影响三包 host 单测/race：affected-host-unit-race-v2.log PASS；未配置 PG 的 integration 自动跳过，PG 行为证据见各隔离日志，不冒 PG race 通过。
- [x] S5 本地完整安装器/普通 client registry 接线：安装器缺五 RPC 的实际 Red 为 s5-installer-red.log；当前 s5-installer-and-binding-green.log、s5-registry-core-pg-green-v2.log 通过 113实体/109可执行、旧4有效撤权、四值/实际UID/旧grant漂移拒绝及完整 ABI/body/ACL。首轮 SQL list 括号错误是 IMPLEMENTATION_FIXED，保留原失败日志，不作业务 Red。
- [x] S5 普通 client 新5/旧4运输边界 Red→Green：s5-client-registration-red.log 采用冻结旧client overlay，s5-client-affected-race.log exit0；新 facts/reconcile 不加 growing-response 特例，原 response cap/HTTP timeout 保持。
- [x] S5 完整安装器消费者最小回归：s5-billing-complete-installer-consumer-green.log、s5-payment-complete-installer-consumer-green.log、s5-refund-complete-installer-consumer-green-v3.log exit0。移除三个默认 Shared 夹具的重复 leaf 安装，退款 synthetic payment seed 与正式 callback path 校准，原生产行为和断言保持；旧短 URL 失败为聚合后夹具修复，保留 v1/v2 log。未重复 S1–S4 全49场景。
- [x] S5 同类 callback fixture 一次盘点和必要相邻修复：s5-all-callback-fixture-inventory.log；默认退款 seed 正式 URL 校准后，最终 s5-refund-final-minimum-consumer.log 的两公开入口/父证明、时义、通知回滚和未知 ACK 共7个 top tests PASS。schema/refund host race、全部 internal/sharedpg 编译、gofmt、diffcheck及 OpenSpec strict PASS（后者将原缺delta结构校准，语义不变）。
- [ ] 旧 Shared refund 三项相邻 oracle 尚未通过，不能将整个相邻聚合 Gate 记 PASS：ObservationAndRecovery/query 的过期虚拟 lease、ClaimJSONUsesNativeUnmarshalOrder 的 native/test 与 cloud proof 校验差异、OriginalByteStringsAndJSONDigest 的故意非法 callback/被改 immutable 父 proof，均由改 S5 前 sourcebound 二进制同条件复现（s5-adjacent-pre-s5-same-condition-baseline.log），记录既有失败，不扩大生产修复或修改断言掩盖。
- [x] S5 源绑定的最小 108→113 SQL 差异准备：s5-root-cloud-preparation-receipt.json；隔离 PG 先安装原108，再在单事务执行新增5、readiness更新1、旧4撤权一次，完整113定义/ABI/有效ACL与四值 Check PASS。其余107个旧函数定义 hash 与当前云108完整快照逐项相同；此项仅准备与隔离验证，实际云执行、读回和 managed history 待 root 完成。
- [x] Review 必要 native 空账单 cutoff=0 保存前漂移修复：真实隔离 PG cutoff-empty-native-red.log 复现公开 RunReconcile 返回成功且落1receipt；仅在空 Details 的同事务保存路径复核 cutoff，cutoff-empty-native-green.log 同 selector 验证拒绝且零receipt与未漂移合法空账单1receipt；不改变 Shared 生产路径。首轮 Shared barrier fixture 失败单独记录，不当业务 Red。必要相邻与 race 结果由最终 receipt 绑定。
- [ ] S5 root/R9 运行时调用点和部署源组合精确接线（PENDING_S5_CALLSITE_ADOPTION），Storage exactSHA 独验收口及 Finance 最终候选/独验仍待满足。

测试重跑原则：只因本片新增行为/源变化重验相关模块及必要相邻；helper 原 61/125/88 和完整 fixture 96 证据条件未变，直接继承。非部署必需全量 DEFERRED_PRE_PROD。

本轮证据目录：`/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/finance-progress/`。`final-s1-s4-local-receipt.json` 绑定恢复 HEAD `058c6a0eed570a60fda10cb645a05d963fe89251`、最终源码/测试/文档及日志 SHA256；该恢复快照不是候选。最终三个受影响包 host race 为 ab-host-race.log exit 0，未连接 PG 的 integration 跳过；真实隔离 PG ordinary 单独验证，不冒充 PG race。Refactor 仅限 native proof reader 同事务复用和 SQL intrinsic lock 在复核前取得；同一受影响验证已重跑。

- [x] 最终 14 字段结构的本地完整 28,395 行容量与原子 receipt：ab-billing-sourcebound-ordinary.log，完整 RPC 参数 JSON 31,438,849 B，receipt JSONB text 32,583,022 B，summary/明细各 28,395，THIS_ENV SUCCESS 5,680；native/Shared 均一份完整 receipt，原每 RPC SQL timeout 5s 未放宽。上述 fixture 没有 system-only 行，`system_facts` 为 `[]`；系统付款/退款的非空留证由新增相邻场景验证，不将本 fixture 当任意系统行规模的上界。
- [x] 同 facts RPC 的退款 worker 错环境/claim version/owner/lease/proof fencing、旧回调及精确确认父款相邻：ab-refund-facts-adjacent-ordinary.log exit 0，使用最终源码重编译与同一最终 leaf。
- [ ] 真实普通 UAT 最终完整输入容量/原子保存/清理、S5运行时 adopted source 组合、Storage依赖和Finance exact candidate SHA 独验及真实资金/UI3仍分别待 root 执行；S1–S4和S5 registry本地实现完成，未形成 Finance CANDIDATE。当前 S5 真 PG ordinary 与 client race 单独留证；未将未连接PG的 host race 当真实PG race。
