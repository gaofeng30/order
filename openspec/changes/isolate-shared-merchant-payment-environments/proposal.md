# 共用商户支付环境隔离

用户已确认继续共用商户，隔离支付、退款及对账环境，未知归属保留异常；S1–S5 本地实现和本地候选已获明确授权。

- gate_type: W3（持久化、支付/退款、事务、幂等、权限与恢复）；ui_level_target: UI0；ui_level_actual: UI0。本 change 不修改小程序或 UI，真实资金和手机 UI3 属于 root 的 UAT 发布验收，未由本地后端证据替代。
- base_sha: b991bd06e661f0d0d88f4ad1527ad11965f1ceca；candidate_sha: 本地提交后由外部 exact-SHA receipt 记录。
- owner: Finance writer；worktree: /Users/vivix/.codex/worktrees/order-uat-payment-env-isolation/order；分支使用 codex/ 前缀。root/R9 管理运行时调用点、组合源与实际 UAT，Finance 不改其 worktree。
- 执行权威：architecture-control/finance-execution-order-amendment-v4.md SHA256 712c168975cc985e9cc998f252e6963951e35c78d80016710f7cca41c918a16c，含已批准的 system_facts/PAYMENT locator 补充；S5 四值安装器、完整 registry 与普通身份收口按同一已确认契约执行。
- 完整 owned paths 与 SHA256：外部 finance-progress/finance-candidate-owned-manifest.json，包含原 Finance49、最终 Finance 增量、S5 必要四值消费者 fixture 与已授权 Storage18 business adoption。除此不写其他路径，不回退并行任务改动。
- 依赖：Storage exact candidate 2811cabfb21005330c38abb205ce7f4810974f8b 的冻结业务源；其独验结果单独绑定，不将临时采用或另一 SHA 的 PASS 当独验。聚合路径的 Finance S5 差异逐项记录；集成前仍须依赖集成及 Finance exact SHA 独验。
- Red/Green/Refactor 和已完成 Writer 结果见 tasks.md 及外部 s5-local-writer-receipt.json。独立验证入口为 finance-progress/finance-exact-sha-gate.py，针对 clean detached exact SHA 运行原最终受影响 selector 与 S5 必要 Gate，不重复 Storage25 或扩大整个仓库测试。
- 外部资产：已具备隔离 PostgreSQL 与 synthetic-only 本地 provider；root 是实际云数据库、公开回调、真实支付及真机资产 owner。实际 UAT 完整容量、原子保存/清理与发布验收仍单独待证实。
- 非目标：推送、main 集成、部署、实际云/商户/支付写入、扩大商户配置和业务规则；本地候选不代表独验、真实 UAT 或发布。

本轮外部证据目录：/Users/vivix/.codex/visualizations/2026/10/10/01a12421-c305-7371-b534-45fdffd1b88a/finance-progress/。
