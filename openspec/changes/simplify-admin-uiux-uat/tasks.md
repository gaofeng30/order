# 任务与证据
- [x] 1.1 方案 native Codex review；解决范围/契约/并发与验收阻断。
- [x] 1.2 核对 UAT executor 当前 owned paths、基线与部署窗口。
- [x] 2.1 每个切片记录目标行为缺失 Red。
- [x] 2.2 最小实现，同一行为检查 Green。
- [x] 2.3 受影响回归 Refactor，记录不需额外重构的理由。
- [x] 3.1 openspec strict、JS syntax、diff/owned paths、UI1 实际主路径/失败态验证。
- [x] 3.2 原生 Codex review 实现，处理范围内问题；Writer Gate 与候选提交。
- [ ] 4.1 exact candidate SHA 干净 detached worktree 独立验证。
- [ ] 4.2 协调 UAT 集成与部署窗口、部署、静态内容读回和实际页面验证。
已执行证据见 evidence.md；未完成项仍 NOT_RUN。任何未满足 gate 不登记 PASS。候选/独立/部署 receipt 在仓库外记录 exact SHA，避免自引用提交。
