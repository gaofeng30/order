# 执行任务

Status: APPROVED_IMPLEMENTATION / IN_PROGRESS / CANDIDATE_NOT_FORMED。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

- [x] 1.1 固定 Astra 需求、UAT 授权、独立 base 和22 owned paths；采用已独验 E8，资源现状见 proposal 和外部 receipt。
- [x] 1.2 原实现同名覆盖 Red，以及新服务/回执/STS必要公共链路 Red。外部 `service-business-red.log`、`sts-business-red.log`、`pinned-provider-oversize-red.log` 退出1；client/installer Red 各退出1。
- [x] 2.1 COS 同 GET 版本、一次发布、准确签名及有界 provider 响应 Green，原临时 tuple 相邻回归。外部 pinned-provider-and-service-green ordinary/race 日志及 root focused 回归退出0。
- [x] 2.2 原公开接口的新 UAT 具体服务，16并发/replay/错误优先/当前权限/损坏 pin Green。同上本地合成回归；实际云应用链路另列4.2。
- [x] 2.3 3 RPC 原子回执、全局 pin、µs/uint64/hash/最小权限及 rollback；隔离 PG 普通/race。`common-storage-runtime-v1/common-storage-runtime-handoff-v1.json` SHA `30356621c9f6e97a0de5690bb0592d45ab5a12b2fba9530334de7241215767ab`：普通/race各348 PASS、60顶层、零FAIL/SKIP，registry108/表26/序列16、SECURITY DEFINER0、旧leaf不变、临时PG和端口清理完成；仅6文件切片，非完整writer。
- [x] 2.4 固定 STS1800/900提前刷新、32并发/异常/撤销及合成响应 Green；仅必要 Refactor 后重跑同验证。focused ordinary/race退出0；无额外抽象的Refactor理由见外部回归记录。
- [x] 2.5 Standards STD01 的原公开确认大小边界差分 Red→Green：实际 GET 为5MiB+1时仍返回原 ErrInvalidImage（422），5MiB+2仍为 ErrImageTooLarge（413），失败零发布/回执/pin；仅readUpload读取门槛沿原MaxImageBytes+1，readPin仍MaxImageBytes。外部 `std01-rgr-receipt-v1.json`：原边界Red退出1，边界与相邻Green退出0、2 PASS；versioned全部相邻race退出0、10 PASS、零FAIL/SKIP。无必要Refactor。旧full25受控中断并清理，不作完整Writer通过。
- [x] 3.1 等待依赖并串行聚合，完整 registry/readiness/历史44、匿名/其他 UID/漂移负例。完整25包 ordinary/race各3056 PASS（913顶层、2143子场景），零FAIL/SKIP；`full25-writer-runtime-v2/storage-full25-writer-v2-runtime-receipt.json` SHA `5925015dd848ba0ad5a3f5ce5819ed0c4df2e4c15bdf7536b9958279db5f3b19`。隔离PG已清理，56591/56592已释放；此为Writer证据，非候选独验或云端108安装。
- [x] 3.2 必需影响面普通/race、vet/build/fmt/diff/strict；源码及秘密扫描。上述完整回归及七项静态检查通过，source575/owned22/HEAD/index验证期间零漂移；fresh audit SHA `281773adfe4ad8c1db802ce16fb08a90fa5003e86a90fc7b2906eebe44cf8fd2`。原80私有模式扫描为零，临时管理凭据另做owned22扫描；后续本任务文档状态更新单列，不将旧源hash当作新候选。
- [x] 4.1 真实 UAT 专用源身份及 STS正向/负权限、私有桶版本/lifecycle读回；原25业务表仅meal2、无旧图片回执/引用。`actual-source-probe/actual-source-native-configuration-closeout-v2.json` SHA `8727665393b3c3d63f06babfcf04fb1678e2cd7f34122b981ea79f5e68697960`；不声称物理30天过期已观察。
- [x] 4.2 真实 UAT 原应用链路及恢复、清理闭环。
  官方直接 SQL 安装/精确权限与旧104读回、普通身份108 readiness、实际 Query 登记、37项 Preview 无 pending/conflicts已闭环，见 external `direct-pg-install-closure/closure-receipt.json`。历史 Push InternalError 保留为原提交接口故障，不再阻断。
  external `actual-image-core/closure-receipt.json`：实际矩阵32场景全通过，含原上传/确认/旧签名PUT重放、16 GET barrier及16并发单 winner、原引用服务、丢ACK回执恢复；新进程重放、当前OWNER撤销拒绝及恢复、COS原版本覆盖/删除标记/精确删除后恢复原pin与字节均通过。真实108快照在network-none副本恢复，全表数据/16序列/权限一致，全部2原pin与2引用保留、原版本字节读取一致。辅助清理脚本解析Red后只补CASE括号，两个错key负例无变更、正确清理通过；云端只执行一次受保护清理，完整逻辑快照与安装后基线精确相等。全部21观察版本逐一备份/核SHA后按精确key+version删除，完整读回0版本；临时恢复容器已清理，finance副本未动。未引用候选归属采用实际时间/内容/数量/单writer联合证据，不伪造PUT ledger。该结果不替代候选独验或CloudRun/手机UI3，不声称30天物理过期、PITR或清桶灾难恢复。
- [ ] 5.1 Writer Gate、两轴审查、owned-only候选及外部 SHA 证据。
- [ ] 5.2 clean detached exact SHA 独验。修改形成新候选重验。
- [ ] 5.3 有对应授权且依赖独验满足后才集成/发布；本地切片不称完整 UAT。

每项完成记录外部 exact source/命令/退出码/receipt SHA。真实资源/PC/Mini/95场景25页/UI3未验证单列，不用 synthetic provider 结果替代。
