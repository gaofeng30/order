## 1. Approval and W0 Red

- [ ] 1.1 获得主 Agent 对 DRAFT 的明确批准后，确认 branch、`base_sha=69cc9b6437dc3181681603d1bb060c07acba97f1`、已集成依赖、W0/UI0 分类与固定 owned paths 未变化；若发现更高风险面，先更新并重新批准 OpenSpec。完成后在本条下记录决定性命令与结果。
- [ ] 1.2 运行 `test -f docs/quality/change-quality-gates.md`，记录协议文件尚不存在的真实 W0 Red；不得提前创建该文件。完成后在本条下记录退出结果与脱敏摘要。
- [ ] 1.3 对 `order-plan-change`、`order-implement-tdd`、`order-verify-change`、`order-integrate-change` 逐一运行 `rg -q 'docs/quality/change-quality-gates.md' ".agents/skills/<skill>/SKILL.md"`，记录四个最小引用尚不存在的真实 W0 Red。完成后在本条下记录每项退出结果。

## 2. Green: One Quality Protocol

- [ ] 2.1 创建 `docs/quality/change-quality-gates.md`，写入最高风险 W0-W3 分类、UI0-UI3 证据定义和完整 16 格决策表，并明确 W2/UI0 硬阻断与 `BLOCKED_EXTERNAL` 恢复字段。完成后在本条下记录内容检查命令与结果。
- [ ] 2.2 在同一协议中写入 W0-W3 最低 Red/Green/Refactor、每类可复制命令/操作模板、统一证据字段、当前可执行 Gate 与外部资产满足后才启用的 Gate；不得用覆盖率或计划命令替代业务验收。完成后在本条下记录决定性检查结果。
- [ ] 2.3 在同一协议中写入 writer 永久 Gate、exact-SHA verifier 触发与失效条件、FAIL 回流、集成条件、C/T/V/R 公式、硬阻断、敏感信息红线和未验证边界；第三次相同错误指纹与 session 复用只引用 `loop-engineering-control-plane`。完成后在本条下记录决定性检查结果。
- [ ] 2.4 最小修改 `.agents/skills/order-plan-change/SKILL.md`：读取统一协议，并要求 DRAFT 声明最高 `gate_type`、目标 `ui_level`、外部资产与验收命令；不复制完整协议或跨 change 调度。完成后在本条下记录检查命令与结果。
- [ ] 2.5 最小修改 `.agents/skills/order-implement-tdd/SKILL.md`：按声明类型执行真实 Red/Green/Refactor、writer 永久 Gate 与脱敏证据模板；外部资产缺失返回 `BLOCKED_EXTERNAL`。完成后在本条下记录检查命令与结果。
- [ ] 2.6 最小修改 `.agents/skills/order-verify-change/SKILL.md`：只接收 CANDIDATE exact SHA，在 clean detached worktree 重跑声明 Gate，并拒绝把未运行或外部阻塞记为 PASS。完成后在本条下记录检查命令与结果。
- [ ] 2.7 最小修改 `.agents/skills/order-integrate-change/SKILL.md`：只接收当前 main 中依赖已 `INTEGRATED`、exact-SHA PASS 且未失效的 candidate；main 推进则返回原 writer 更新和重验。完成后在本条下记录检查命令与结果。

## 3. Refactor: Remove Drift and Scope Creep

- [ ] 3.1 复核详细规则只存在于 `docs/quality/change-quality-gates.md`，四个 stage skill 仅保留各阶段最小引用/检查，没有复制 lane、scheduler、ledger、checkpoint、session、错误指纹计数或主动回传算法。完成后在本条下记录搜索命令与设计理由。
- [ ] 3.2 重跑与 1.2、1.3 相同的结构检查，并检查 W0-W3 × UI0-UI3、分类命令/证据模板、C/T/V/R、硬阻断、敏感信息和 `BLOCKED_EXTERNAL` 关键内容完整。完成后在本条下记录同一命令的 Green/Refactor 结果。
- [ ] 3.3 检查协议未把 Playwright、微信工具、真实数据库、支付、真机、CI 或监控写成当前可执行命令，且 `.agents/skills/order-run-loop/**`、根 `AGENTS.md`、业务代码和产品/架构文档保持不变。完成后在本条下记录 changed-path 与内容检查结果。

## 4. Writer Local Gate and Candidate

- [ ] 4.1 运行 `openspec validate enforce-change-quality-gates --strict`、`git diff --check 69cc9b6437dc3181681603d1bb060c07acba97f1...HEAD` 和固定 owned-path audit，确认 spec/tasks/base/依赖/验收一致。完成后在本条下记录命令与结果。
- [ ] 4.2 运行当前 Go/API Gate：gofmt、`go test`、`go test -race`、`go vet`、`go build` 和 `services/api/scripts/smoke.sh`；所有 Go 命令使用 `GOPROXY=off GOTOOLCHAIN=go1.26.5`。完成后在本条下记录每个退出结果，不得把未运行项写 PASS。
- [ ] 4.3 运行当前前端 static Gate：全部 `apps/**/*.js` 的 `node --check`，以及使用 Node `JSON.parse` 检查 `apps/**/*.json` 与 `project.config.json`。完成后在本条下记录命令与结果。
- [ ] 4.4 对 diff 和证据执行敏感信息红线检查，按 C/T/V/R 各 10 分给出可追溯评分；只有总分不低于 36、每项不低于 8 且硬阻断为零才继续。完成后在本条下记录评分依据和 verdict。
- [ ] 4.5 仅提交固定 owned paths，形成新的完整 candidate SHA；记录 `git status --short --branch`、`git diff --exit-code` 与 `git diff --cached --exit-code` 为 clean，并主动回传主会话。完成后在本条下记录 SHA 与结果。

## 5. Independent Verification and Integration Handoff

- [ ] 5.1 仅在 4.5 已产生 CANDIDATE 完整 SHA 后，交给独立 verifier 在另一 clean detached worktree 对该 SHA 从头重跑所有声明 Gate；探索、DRAFT、未提交 diff 不创建 verifier。完成后记录 attestation 或 FAIL 摘要。
- [ ] 5.2 verifier FAIL 时交回原 writer；修复产生新 SHA 后复用 verifier session，但必须为新 SHA 重建 clean detached worktree 并完整重验。同一错误指纹第三次升级遵循 `loop-engineering-control-plane`，完成后记录交接结果。
- [ ] 5.3 仅在依赖已进入当前 main 的 `INTEGRATED`、candidate exact-SHA PASS 未失效且获得集成授权后交给 integration；若 main 推进，返回原 writer 更新、形成新 SHA并重验。完成后记录最终 Gate 结果；集成前不得 archive。
