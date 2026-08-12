## 1. Approval and W0 Red

- [x] 1.1 获得主 Agent 对 DRAFT 的明确批准后，确认 branch、当前 `base_sha`、已集成依赖、W0/UI0 分类与固定 owned paths 未变化；若发现更高风险面，先更新并重新批准 OpenSpec。完成后在本条下记录决定性命令与结果。
  - Evidence (2026-08-12): 主 Agent 明确批准；`git branch --show-current`=`codex/enforce-change-quality-gates`，`git rev-parse main`=`69cc9b6437dc3181681603d1bb060c07acba97f1`，基线是 HEAD 祖先；loop 主 spec 与归档 change 可读，`openspec validate enforce-change-quality-gates --strict` PASS，worktree clean。
- [x] 1.2 运行 `test -f docs/quality/change-quality-gates.md`，记录协议文件尚不存在的真实 W0 Red；不得提前创建该文件。完成后在本条下记录退出结果与脱敏摘要。
  - Evidence (Red, 2026-08-12): `test -f docs/quality/change-quality-gates.md` exit 1；决定性失败为目标协议文件尚不存在，无敏感输出。
- [x] 1.3 对 `order-plan-change`、`order-implement-tdd`、`order-verify-change`、`order-integrate-change` 逐一运行 `rg -q 'docs/quality/change-quality-gates.md' ".agents/skills/<skill>/SKILL.md"`，记录四个最小引用尚不存在的真实 W0 Red。完成后在本条下记录每项退出结果。
  - Evidence (Red, 2026-08-12): 四个 `rg -q` 依次 exit 1；决定性失败为四个 stage skill 均未引用统一质量协议，无敏感输出。

## 2. Green: One Quality Protocol

- [x] 2.1 创建 `docs/quality/change-quality-gates.md`，写入最高风险 W0-W3 分类、UI0-UI3 证据定义和完整 16 格决策表，并明确 W2/UI0 硬阻断与 `BLOCKED_EXTERNAL` 恢复字段。完成后在本条下记录内容检查命令与结果。
  - Evidence (Green, 2026-08-12): `test -f docs/quality/change-quality-gates.md` exit 0；W0-W3 四行、UI0-UI3 四列、决策表标题、W2/UI0 硬阻断和 `BLOCKED_EXTERNAL` token 检查全部 PASS。
- [x] 2.2 在同一协议中写入 W0-W3 最低 Red/Green/Refactor、每类可复制命令/操作模板、统一证据字段、当前可执行 Gate 与外部资产满足后才启用的 Gate；不得用覆盖率或计划命令替代业务验收。完成后在本条下记录决定性检查结果。
  - Evidence (Green, 2026-08-12): `rg -q` 检查“各类最低 Red → Green → Refactor”“分类命令或操作模板”“统一证据模板”“当前仓库可直接运行”“外部 Gate”均 PASS；不可用 runner 只登记 owner/恢复条件。
- [x] 2.3 在同一协议中写入 writer 永久 Gate、exact-SHA verifier 触发与失效条件、FAIL 回流、集成条件、C/T/V/R 公式、硬阻断、敏感信息红线和未验证边界；第三次相同错误指纹与 session 复用只引用 `loop-engineering-control-plane`。完成后在本条下记录决定性检查结果。
  - Evidence (Green, 2026-08-12): `rg -q` 检查 Writer Gate、verifier/失效、Integration、C/T/V/R、一票否决、敏感信息章节及第三次升级主 spec 引用全部 PASS；`openspec validate ... --strict` PASS。
- [x] 2.4 最小修改 `.agents/skills/order-plan-change/SKILL.md`：读取统一协议，并要求 DRAFT 声明最高 `gate_type`、目标 `ui_level`、外部资产与验收命令；不复制完整协议或跨 change 调度。完成后在本条下记录检查命令与结果。
  - Evidence (Green, 2026-08-12): `rg -q 'docs/quality/change-quality-gates.md' .agents/skills/order-plan-change/SKILL.md` exit 0；新增内容仅为 planning 分类、外部资产和验收入口。
- [x] 2.5 最小修改 `.agents/skills/order-implement-tdd/SKILL.md`：按声明类型执行真实 Red/Green/Refactor、writer 永久 Gate 与脱敏证据模板；外部资产缺失返回 `BLOCKED_EXTERNAL`。完成后在本条下记录检查命令与结果。
  - Evidence (Green, 2026-08-12): implement skill 协议引用检查 exit 0；新增内容仅为 implement 上下文、RGR/脱敏证据、`BLOCKED_EXTERNAL` 和 writer C/T/V/R 硬门。
- [x] 2.6 最小修改 `.agents/skills/order-verify-change/SKILL.md`：只接收 CANDIDATE exact SHA，在 clean detached worktree 重跑声明 Gate，并拒绝把未运行或外部阻塞记为 PASS。完成后在本条下记录检查命令与结果。
  - Evidence (Green, 2026-08-12): verify skill 协议引用检查 exit 0；新增内容仅为 exact-SHA 分类/运行证据检查、失效与 FAIL 回 writer 边界。
- [x] 2.7 最小修改 `.agents/skills/order-integrate-change/SKILL.md`：只接收当前 main 中依赖已 `INTEGRATED`、exact-SHA PASS 且未失效的 candidate；main 推进则返回原 writer 更新和重验。完成后在本条下记录检查命令与结果。
  - Evidence (Green, 2026-08-12): integrate skill 协议引用检查 exit 0；新增内容仅为 current-main 依赖、exact-SHA、评分/硬阻断入口；moving-main 原 writer 重验规则保持原位。

## 3. Refactor: Remove Drift and Scope Creep

- [x] 3.1 复核详细规则只存在于 `docs/quality/change-quality-gates.md`，四个 stage skill 仅保留各阶段最小引用/检查，没有复制 lane、scheduler、ledger、checkpoint、session、错误指纹计数或主动回传算法。完成后在本条下记录搜索命令与设计理由。
  - Evidence (Refactor, 2026-08-12): `git diff --numstat` 显示四个 skill 净新增分别为 plan 2、implement 4、verify 4、integrate 2 行（其余为原句替换）；逐段 diff 只含阶段入口，错误指纹计数与控制面算法仍由 `order-run-loop` 持有。
- [x] 3.2 重跑与 1.2、1.3 相同的结构检查，并检查 W0-W3 × UI0-UI3、分类命令/证据模板、C/T/V/R、硬阻断、敏感信息和 `BLOCKED_EXTERNAL` 关键内容完整。完成后在本条下记录同一命令的 Green/Refactor 结果。
  - Evidence (Refactor, 2026-08-12): 原 Red 的 `test -f` 与四个 `rg -q` 均 exit 0；W/UI、证据模板、评分、硬阻断、敏感信息与外部边界 token 检查全部 PASS。
- [x] 3.3 检查协议未把 Playwright、微信工具、真实数据库、支付、真机、CI 或监控写成当前可执行命令，且 `.agents/skills/order-run-loop/**`、根 `AGENTS.md`、业务代码和产品/架构文档保持不变。完成后在本条下记录 changed-path 与内容检查结果。
  - Evidence (Refactor, 2026-08-12): 外部 Gate 表只登记当前“未建立”、owner 与恢复条件；`git diff --name-only -- AGENTS.md .agents/skills/order-run-loop services apps docs/product docs/requirement-list` 为空，使用 `git status --porcelain --untracked-files=all` 的 owned-path audit PASS。

## 4. Writer Local Gate and Candidate

- [ ] 4.1 运行 `openspec validate enforce-change-quality-gates --strict`、`git diff --check d68886931dcfb01d50d65cc0bd8c4cc7cea54a4e...HEAD` 和固定 owned-path audit，确认 spec/tasks/base/依赖/验收一致。完成后在本条下记录命令与结果。
  - Evidence (Writer, 2026-08-12): 最终 Gate 前 `git rev-parse main` 仍为 `69cc9b6437dc3181681603d1bb060c07acba97f1` 且是 HEAD 祖先；strict、`git diff --check`、结构/链接/内容检查与 `git status --porcelain --untracked-files=all` owned-path audit PASS。最终 commit 后另用 `base...HEAD` 复核。
  - Invalidation (2026-08-12): 本地 main 随后推进到 `d68886931dcfb01d50d65cc0bd8c4cc7cea54a4e`，`git merge-base --is-ancestor main HEAD` exit 1；按协议撤销旧 writer verdict，待 rebase 后重跑。
  - Revalidation (2026-08-12): `git rebase main` 无冲突；新 main `d68886931dcfb01d50d65cc0bd8c4cc7cea54a4e` 是 HEAD 祖先。strict、`base...HEAD`/working `git diff --check`、结构/链接/内容、敏感模式、forbidden path 与 owned-path audit 全部 PASS。
  - Invalidation (2026-08-12): 本地 main 再次推进到 `b6e24f97bb20f37543e10a1dc354cf75f07d47a6`，祖先检查 exit 1；上一轮 writer verdict 作废。
- [ ] 4.2 运行当前 Go/API Gate：gofmt、`go test`、`go test -race`、`go vet`、`go build` 和 `services/api/scripts/smoke.sh`；所有 Go 命令使用 `GOPROXY=off GOTOOLCHAIN=go1.26.5`。完成后在本条下记录每个退出结果，不得把未运行项写 PASS。
  - Evidence (Writer, 2026-08-12): `test -z "$(gofmt -l services/api)"`、`GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...`、同路径 `go test -race`、`go vet`、`go build` 与 `bash services/api/scripts/smoke.sh` 全部 exit 0；smoke 输出 `smoke: PASS`。
  - Invalidation (2026-08-12): base 变化使旧 writer 验证失效，待最新 main 成为祖先后重跑。
  - Revalidation (2026-08-12): rebase 后重新运行 gofmt、`go test`、`go test -race`、`go vet`、`go build`、API smoke 全部 exit 0，smoke 输出 `smoke: PASS`。
  - Invalidation (2026-08-12): main 再次变化，上一轮 Go/static writer 验证失效，待 rebase 后重跑。
- [ ] 4.3 运行当前前端 static Gate：全部 `apps/**/*.js` 的 `node --check`，以及使用 Node `JSON.parse` 检查 `apps/**/*.json` 与 `project.config.json`。完成后在本条下记录命令与结果。
  - Evidence (Writer, 2026-08-12): `find apps -type f -name '*.js' -print0 | xargs -0 -n 1 node --check` exit 0；协议内 Node `JSON.parse` 命令 exit 0，输出 `JSON static PASS files=42`。仅证明 UI0 static，未声称 UI1/UI2/UI3。
  - Invalidation (2026-08-12): base 变化使旧 writer 验证失效，待最新 main 成为祖先后重跑。
  - Revalidation (2026-08-12): rebase 后重新运行全部 JS `node --check` 与 JSON `JSON.parse` 均 exit 0，JSON 文件数 42；仅证明 UI0 static，未声称 UI1/UI2/UI3。
  - Invalidation (2026-08-12): main 再次变化，上一轮前端 static writer 验证失效，待 rebase 后重跑。
- [ ] 4.4 对 diff 和证据执行敏感信息红线检查，按 C/T/V/R 各 10 分给出可追溯评分；只有总分不低于 36、每项不低于 8 且硬阻断为零才继续。完成后在本条下记录评分依据和 verdict。
  - Evidence (Writer, 2026-08-12): 敏感值模式与禁止路径检查 PASS；C=10（协议/调用边界完整）、T=10（真实 W0 RGR 和全量 current gates）、V=8（exact-SHA 验证包完整，独立结果待 verifier）、R=9（失败回流/失效/恢复条件完整），总分 37，六项硬阻断均为零，verdict=`CANDIDATE_READY`。
  - Invalidation (2026-08-12): moving-main 硬门触发，旧 `CANDIDATE_READY` 未形成 candidate，待 rebase 后重新评分。
  - Revalidation (2026-08-12): 新 base 上敏感模式和禁止路径检查 PASS；C=10、T=10、V=8、R=9，总分 37，六项硬阻断为零，verdict=`CANDIDATE_READY`。
  - Invalidation (2026-08-12): moving-main 硬门再次触发，上一轮 `CANDIDATE_READY` 未形成最终 candidate。
- [ ] 4.5 仅提交固定 owned paths，形成新的完整 candidate SHA；记录 `git status --short --branch`、`git diff --exit-code` 与 `git diff --cached --exit-code` 为 clean，并主动回传主会话。完成后在本条下记录 SHA 与结果。

## 5. Independent Verification and Integration Handoff

- [ ] 5.1 仅在 4.5 已产生 CANDIDATE 完整 SHA 后，交给独立 verifier 在另一 clean detached worktree 对该 SHA 从头重跑所有声明 Gate；探索、DRAFT、未提交 diff 不创建 verifier。完成后记录 attestation 或 FAIL 摘要。
- [ ] 5.2 verifier FAIL 时交回原 writer；修复产生新 SHA 后复用 verifier session，但必须为新 SHA 重建 clean detached worktree 并完整重验。同一错误指纹第三次升级遵循 `loop-engineering-control-plane`，完成后记录交接结果。
- [ ] 5.3 仅在依赖已进入当前 main 的 `INTEGRATED`、candidate exact-SHA PASS 未失效且获得集成授权后交给 integration；若 main 推进，返回原 writer 更新、形成新 SHA并重验。完成后记录最终 Gate 结果；集成前不得 archive。
