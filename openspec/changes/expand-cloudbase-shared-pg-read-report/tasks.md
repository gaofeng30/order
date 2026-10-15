# 第八切片任务

Status: APPROVED_REQUIREMENTS / LOCAL_FULL24_PASS / ACTUAL_UAT_DATABASE_GATE_PASS / CANDIDATE_NOT_CREATED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## 1 基线与结构
- [x] 1.1 核对批准15 named ABI、17公开方法、25唯一owned路径和并行依赖；当前精确父第七ea76e348已独验并采用。结构Red为change不存在时strict exit1；四文档写入后的strict Green只证明文档结构。
- [x] 1.2 root采用已独验第七ea76e348精确SHA；18未提交路径hash保持，adoption receipt SHA151b3266ca0dcdcd954801ef9b46d3575b840521fef8ce07a0e289bf67bb0f92。共同文件root独占；Astrareadiness owned25勘误见proposal。105内容/传输Red日志common15-content-red.log，安装器缺15入口及Go名单拒绝均为实际失败；同命令Green日志common15-content-green-first.log exit0。不是公共业务Red替身或完整Gate。

## 2 原生差分 RGR
- [x] 2.1 billing五方法：原授权/范围/分页/CSV、固定+08账单、CompareBill、provider事务外、intrinsic重放、晚clock/投影/证据同事务、Mismatch提交后返回。
- [x] 2.2 adminreport十方法：原多statement可见性与Scan错误顺序、日期IANA/Local、nullable/slices/金额、CSV每页授权；两写方法原委托。
- [x] 2.3 audit.Search：原Forbidden映射、脱敏、uint16 Limit、排除最后微秒、原scanner范围。
- [x] 2.4 storestatus.Apply：输入/clock/currentauth/singleton/receipt顺序、三状态与UNCHANGED、两次封闭retry、并发、晚故障和丢ack。
- [x] 2.5 原handler/publiccaller与真实持久facts；16并发、跨支付退款投影、取消、原多statement barrier及9个growing/6个bounded成功响应分类；必要Refactor和同场景复跑。

## 3 统一能力
- [x] 3.1 adopted父+15=105固定ABI/body/Invoker/UID/RLS/精确列及sequence权限；原44ledger/checksum保持，未知grant/SECDEF failclosed。
- [x] 3.2 原fulluint64/金额wrap/nullable/Time integerµs/PG范围/原Local/SQLSTATE优先及封闭wire；无新行数/姓名/账单上限。
- [x] 3.3 删除开发临时追加fixture，统一installer+Check下完整影响面ordinary和serialized race、vet/Linux build/smoke/fmt/diff/strict及source无漂移。

## 4 普通UAT
- [x] 4.1 root受控版本化安装、执行/登记实际history核验、普通身份105Check和源码绑定。
- [x] 4.2 全15入口/原Go服务与handler、关键并发/晚故障/未知ack、真实增长响应与CSV；provider合成与真实服务边界分列。
- [x] 4.3 anon/otherUID拒绝/RLS零行/DDL和ledger拒绝/权限漂移拒绝恢复。
- [x] 4.4 所有合成业务/身份/probe/fault精确清理，count/hash回baseline、原ledger/body/readiness/source闭合。

## 5 生命周期
- [ ] 5.1 完整Writer Gate、实际秘密扫描、两轴precommit及owned-only staged检查。
- [ ] 5.2 本地候选完整SHA和外部exact包；提交内不自引用SHA。
- [ ] 5.3 clean detached exactSHA独验；失败由writer修复并新SHA重验。
- [ ] 5.4 依赖满足且独验通过后另行main集成；无push/merge/archive/上线完成声明。

证据必须记录命令/exit/source/log SHA和层级；缺函数Red不替代公共业务Red。临时70+9安装器仅开发证据。计划在services/api按实际影响面执行go test -p 1及-race；最终还须覆盖父切片必要相邻回归，不用leaf Green替代105完整验收。本地完整最终统一源已冻结并完成 writer 回执：full24 ordinary/race 各2786 test PASS、24 package PASS、0 FAIL/SKIP；vet、Linux all-cmd build、smoke、两项fmt、diff、strict七项exit0。决定性 runner 为external shared-pg-read-report-root/full24-writer-runtime/e8-writer-full24-runner-v1.py，SHA f4d1d368e90aee057356c3689f27359ea2392d333357e4364de05cb5ac7b983d；最终 runtime receipt SHA1cdbd84ee087bddca3e8eefa3c2da0df0d45baf2d772f9a886eb2be5cb1b0a6d，source559/owned25/index零漂移。完整本地结果覆盖早期临时fixture；真实云105及负权限已闭合，候选及独验仍待完成。

真实UAT决定性闭合：external shared-pg-read-report-root/actual-platform-final-closeout-v3.json；受控357789B增量Execute→105 body/Invoker/普通Check→官方history登记20261009152250（stored仅去空行，36含原35）。v1原Client全15ABI/17public、33业务断言和原Admin Handler 11010048B字段/11010725B JSON通过；两个admin写端口仅原委托spy，provider合成。v2两写各16路竞争，持久projection各一次；namespace限定最后audit CHECK故障使两族Unavailable而25表全hash相等；精确故障hash移除/0读回。真实提交后local wrapper dropACK各一次，原公开方法Unavailable，重试25表/16seq完全不变。30个有效签名权限负例、meal2对otherUID0、ledger INSERT和实际角色DDL拒绝、额外授权Check失败并恢复通过。全部业务25表/16seq/native44/history36/body105最终保持，临时身份/约束/进程已清理。CloudRun/真实provider/95cases仍NOT_RUN。

真实大账单请求容量v3：external actual-large-billing-request-closeout-v3.json；原Client普通UID发送794871B/4096与12714231B/65536，实际HTTP200、RECONCILED、原CompareBill全量结果与DB SYSTEM_EVIDENCE数量一致，prepay版本1→2→3；相同4096大请求在namespace限定最后audit fault下UNAVAILABLE且全25表hash不变。只action_audits/prepayments变化，精确故障移除、25表/16序列/native44/history36名称版本/105body/普通Check和进程闭合。12.7MB实际接受，不声称发生网关超限拒绝；50MB未运行，large system-only未云验，不增业务限制。
