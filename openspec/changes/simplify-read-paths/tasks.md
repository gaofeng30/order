# Tasks

- [x] 1 Red：支付恢复、菜单请求放大、图片读取、计数、统计/导出复现。
  - 决定性失败与基线消融见 evidence.md；修复后审查发现撤权缺陷也先复现再修改。
- [x] 2 Green：逐切片实现并重跑。
  - JS请求预算/支付恢复；Go真实适配器/HTTP；PG16日期、售罄、分类、权限、图片pin和报表一致性通过。
- [x] 3 Refactor：仅必要的批量投影与CSV投影提取，重跑主路径和相邻回归。
  - 无业务缓存、新基础设施或通用框架；上传、引用、价格、鉴权仍验证。共享无状态投影避免两份CSV实现漂移。
- [x] 4 Writer：编译、静态、隔离数据库与HTTP性能验证、独立代码审查。
  - Standards/Spec各1项同一撤权问题已修复；最终run-local.sh全部通过（JS、Go race/vet、隔离PG、OpenSpec、manifest和diff）。UI3仍BLOCKED_EXTERNAL。
- [x] 5 Candidate：仅本任务本地候选提交；完整SHA由提交回执记录（避免将自引用SHA写回候选）；UI3外部缺口明确保留。
- [ ] 6 Verifier：独立detached exact SHA源码验证；缺UI3 receipt不进入INDEPENDENT_VERIFIED，不集成/发布。

- [x] UAT readiness 回归：完整安装后执行实际 Check，修复注册表顺序、运行授权函数数量与 import_result UTC 配置；Red/Green 均有可复现测试。
