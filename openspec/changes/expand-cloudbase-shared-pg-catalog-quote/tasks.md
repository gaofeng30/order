# Tasks

Status: APPROVED_QUOTE_SQL_ATOMIC / IMPLEMENTING；gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN；base_sha=22bc009a3a6b278d08d09b8d50d64c31cf5c332f；candidate_sha=NOT_CREATED。

## 1. 需求与接口
- [x] 1.1 亲读native端口图与实施brief，限定完整本轮业务及付款caller-Tx未实现边界。
- [x] 1.2 冻结17非Quote typed named ABI/ownership，允许独立纵向实现；Quote最终两RPC裁定见1.3。
- [x] 1.3 root真实无deadline活性/取消/赢家优先原型已取证；需求owner否决乐观刷新，批准Quote锁内SQL派生与两RPC最终ABI。
- [x] 1.4 四工件strict与writer17非Quote逐参数核对；Quote两RPC由common依新裁定登记，证据更新不假称测试通过。

## 2. 非Quote纵向TDD
- [x] 2.1 Catalog/Menu/Storefront读与真实Confirmed依赖：native行为Red→完整facts+原投影Green，HTTP匿名/坏token/URL规则及快照barrier。
- [x] 2.2 Catalog十kind命令：ignored/late bytes、确认/receipt、target/重排/售罄原规则、原子故障及并发/unknownack。
- [x] 2.3 Storefront Configure/Launch：全部原输入/NULL组/NaNInf晚消费、重复receipt回滚、原DB日期与独立审计clock。
- [x] 2.4 DisplayPricing单snapshot、primary/extra算法、nil与half-up完整差分。
- [x] 2.5 MerchantSoldOut：当前授权/历史receipt、今日明日/locator竞态、noop receipt、原retry与unknownack。
- [x] 2.6 真实第三身份适配+原HTTP完整端口和权限撤销回归；必要Refactor后重跑。

## 3. Quote（最终锁内SQL方案；已有本地Green，完整门禁待收束）
- [x] 3.1 Unicode/金融/canonical SQL primitive真实native差分，Create/Read真实Red→锁内完整派生最小实现；Go独立校验与owner隐藏，GET不加receipt要求。
- [x] 3.2 NULL receipt材料的key/replay/错误优先，源锁内计算进展、capturedAt/金额/hash/金融范围、故障全聚合回滚。
- [x] 3.3 16同异key、receipt唯一性、unknowncommit及实际活性/取消门；证明未改业务资源边界。

## 4. 统一与真实UAT
- [x] 4.1 root统一固定registry/精确body/ACL/RLS/readiness、普通与anon/otherUID/额外列对象负例；不以临时独立安装作终验。
- [x] 4.2 普通JWT完整本轮业务HTTP/并发/故障/未知提交矩阵，外部provider证据分层。
- [x] 4.3 合成数据/临时权限/故障/身份清理并最终readiness/44ledger复读。

## 5. 生命周期
- [x] 5.1 必需ordinary/race/相邻/静态/owned/secret/strict Writer Gate；未运行/skip/基线失败单列。
- [ ] 5.2 两轴precommit/C-T-V-R、候选与外部精确SHA绑定包；整个切片不能凭17非Quote部分通过形成完成声明。
- [ ] 5.3 clean detached候选独验及已声明父依赖核验。
- [ ] 5.4 依赖main集成且授权满足后才集成；不自动push/PR/archive。

## 最终实现与平台证据（候选前）

需求方亲读正式 receipts 与 JSONL：非Quote六包 ordinary/race各172 PASS，common及前序八包各920 PASS，Quote完整包各204 PASS，合计每模式1296 test PASS、0 FAIL/SKIP（含子测试，不把package终态再计入）。非Quote使用原v44+统一49 InstallSQL/普通身份Check，非临时自定义授权fixture。510项Go/SQL冻结清单逐hash复核无漂移；最终Root Writer Gate总包已亲核，5.1闭合；5.2仅提交前条件完成，候选外部SHA绑定未完成。

- `shared-pg-catalog-quote-writer/final-writer-receipt.json`：22 owned源稳定，ordinary SHA `8baef3242025013e8303542b52324c4377a557a08ed01f249f34a97714d0e295`，race `7402032c1c7b906b0570dbfb38b12e7fc1dad515716a226fd3b2fd775e2c3d7f`。诊断性FK/NUL fixture与中断cleanup尝试单列，不冒业务Red或完整PASS。
- `shared-pg-catalog-quote-common/final-common-writer-receipt.json`：ordinary `94c79e07404beca88c536e4710670733cff6b8a108d095d8bdb567221237026e`，race `f6368895d9ada0b81db0864870209880aaa9bb13a75ad7c6493970a5e837cb42`；49RPC/20RLS/8seq，首两Redeem body不变。
- `shared-pg-catalog-quote-root/quote-full-ordinary.jsonl` SHA `a1a5711f0278f3662b28383974adda61cf3530b299779f2715b88905bf0825d6`，race SHA `f4f51787abf48a6927f91000960eec4b1548c39f1e84732009e08bdcf7f41ebc`。含真实身份/HTTP、strict receipt、金额/uint64/负epoch、16并发、丢提交响应、原子故障。旧Unicode attempt FAIL等过程记录保留；以最终整包计数。
- Unicode完整PG17.11回执 `unicode-cloud-conformance-receipt.json` SHA `1206714692a1d1f57ccd4881ecc310189f1a16df30d72f2fabd282d5ae86f801`：24组1,310,430cases×5结果count/hash全匹配、null0、临时对象0；24附件实际bytes/hash已核。仅原语层，不单独证明业务。

实际部署采用精确差量：完整SQL authority SHA `7fb4827dc2920bcea30fe32399d42d2c819fbe9b866029b81ba5e94ae3dd7c69`，delta SHA `4ac6010b8b42f2d779b630e9b6e81fbd3bafd7c9488f13923011ff7e7dac8817`。仅省29条与第三既有定义完全相同函数；19历史Query不改，19新函数+readiness及权限语句保留。原full payload Push失败/仅pending preview冲突均保留，不用repair/执行DDL绕过。`delta-upgrade-rollback-tests.jsonl` 2 PASS/0FAIL/SKIP，SHA `3b238900fcd3d991a992236907375a6d5b6545265a09d2b36605f5fc3a494ea1`；正式task `task-ae6113f2` Succeed，普通身份Check true。

`http-business-evidence.json` SHA `e38ff43c2a99be1dc069cb2b156520db2848ba52455efc2e683630d7dd2294ed`：总96标签141响应；6个harness输入/预期诊断有correction字段单列，修正后90标签135响应全部匹配预期。真实CloudBase普通JWT、实际Go业务及loopback原HTTP、opaque hash session/PC审批；WeChat/provider/URL为synthetic，不是CloudRun或真实微信UI。全部19新RPC调用；Quote16同key=1create15replay、16异key=16create；五写族lostack重放及五CHECK故障前后持久聚合相同、撤fault后同key恢复。当前merchant/PC撤权、GET不可变/其他owner隐藏、maxuint64及GET不要求receipt/Create严格要求receipt均实际验证。

anon19与otherUID19全部42501；otherUID20张RLS关系仅空行。额外列/未审关系grant让readiness=false，revoke后恢复true。临时UID删除Success1/Failed0且Total0。final-business-counts实际25业务关系除原meal2行外全0、ledger44；final-cleanup-and-readiness记录原meal逐字不变、fault/Unicode临时对象0、49实际函数逐字SHA匹配、ordinaryCheck true。平台source对应冻结510项，无drift。

本轮Spec precommit review 0 must-fix，报告 external `shared-pg-catalog-quote-root/final-precommit-spec-review.md`。5.1已完成；5.2候选外部SHA绑定、clean detached独验仍各自待完成；第四没有集成main/push、没有完整CloudRun、真实支付/微信/COS、95/25/UI3通过结论。第五brief仅规划，不计当前实现。


## Writer freeze

正式 `final-root-writer-receipt.json` SHA `0619ee7dfff2f11972c0314861de1f1f1722faa865061de4198190c554dc4bfc` 已亲核：六组日志实际bytes/hash匹配，15个包每模式均终态PASS；`platform-receipt.json` SHA `39401a3e70d282de99115fb66e38d9642c37b4988ddb60f91fbbf9e7c5ef34cc` 的92个附件逐项hash/bytes核对。实际92个私密值在129文件扫描0匹配（值未输出）；全services vet、allcmd Linux amd64 CGO0 build、smoke session67922、25 owned Go格式、diff检查均成功。最终strict在本次四文档收束后重跑通过。

5.2的双轴precommit零must-fix及C/T/V/R提交前评估条件已满足；整项继续未勾，需root创建候选并外部绑定精确SHA/验证包。文档header的NOT_CREATED为本提交前快照，候选与独验状态随后仅由外部lifecycle/candidate/independent receipt记录，禁止为写入自身SHA再改同一候选。四工件到此冻结；变更需writer新SHA并重新验证。5.3/5.4仍未完成，非完整UAT。
