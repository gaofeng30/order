## Why

仓库已有 OpenSpec 七态、change 级四阶段 skill 和 `order-run-loop` 跨 change 控制面，但尚未统一规定不同风险 change 最低需要什么 Red/Green/Refactor、运行态证据、评分和硬阻断。若每个 writer 临时决定门禁，容易把静态检查当成业务验收、把外部资产缺失写成 PASS，或为探索/DRAFT 滥开 verifier。

## What Changes

- 建立一份 change-local 质量门禁协议：每个 change 按最高风险归类为 W0 结构、W1 内部逻辑、W2 公共契约/UI 或 W3 数据/资金/并发，并执行该类最低 Red/Green/Refactor 和 writer Gate。
- 建立 UI0 静态、UI1 浏览器/模拟器、UI2 微信开发者工具/体验版、UI3 真机/真实平台的运行证据等级；没有实际运行的级别只能记录为未验证或 `BLOCKED_EXTERNAL`。
- 固化 C/T/V/R 四维评分、硬阻断、敏感信息红线、exact-SHA verifier 触发/失效、失败回流和集成条件；评分不得覆盖任何硬阻断。
- 新增 `docs/quality/change-quality-gates.md` 作为详细协议，四个既有 change stage skill 仅增加指向该协议的最小分类与检查入口；不复制 `order-run-loop` 的跨 change 调度规则。
- 不安装或配置 Playwright、微信工具、数据库、CI、监控，不实现业务测试或运行外部 Gate。

## Capabilities

### New Capabilities

- `change-quality-gates`: 定义单个 OpenSpec change 如何按最高风险选择最低质量证据、形成 candidate、触发 exact-SHA 验证并满足集成 Gate。

### Modified Capabilities

无。

## Impact

- 状态：`DRAFT`；本轮只完成规划 artifact 和 strict validation，不执行任何 tasks。
- 本 change 的 `gate_type`：W0；`ui_level`：UI0。理由是 apply 只新增质量协议文档并修改四个 stage skill 的文字入口，不改变产品运行行为、公共契约、持久化数据、资金或并发结果；若 apply 发现更高风险面，必须先更新并重新批准本 OpenSpec。
- owner：`codex/enforce-change-quality-gates` 当前独立 worktree 的 Loop Writer；同一时刻其他 worker 不得写入 owned paths。
- owned paths：
  - `openspec/changes/enforce-change-quality-gates/**`
  - `docs/quality/change-quality-gates.md`（apply 前不得创建）
  - `.agents/skills/order-plan-change/**`
  - `.agents/skills/order-implement-tdd/**`
  - `.agents/skills/order-verify-change/**`
  - `.agents/skills/order-integrate-change/**`
- 只读共享契约：根 `AGENTS.md`、`.agents/skills/order-run-loop/**` 和 `openspec/specs/loop-engineering-control-plane/spec.md`；本 change 不修改它们，也不复制其 lane、调度、session、checkpoint 或主动回传规则。
- 依赖：`loop-engineering-control-plane` 已归档并集成到本地 `main@69cc9b6437dc3181681603d1bb060c07acba97f1`；同时沿用已集成的 `change-governance` 七态与 exact-SHA 规则。本 change 不依赖正在其他 worktree 推进的产品基线，也不得回退其改动。
- 非目标：修改 `.agents/skills/order-run-loop/**`、根 `AGENTS.md`、业务代码、产品/架构文档；安装工具；创建 CI/监控；连接数据库、微信或支付；推送、部署或写入外部系统。
- 最小成功标准：DRAFT 四件 artifact 完整且 strict PASS；apply 后决策表覆盖 W0-W3 × UI0-UI3，每类命令/证据模板、C/T/V/R rubric、硬阻断、敏感信息和未验证边界可机器检查；四个 stage skill 只做最小引用/检查；diff 只包含 owned paths。

### Acceptance Commands

```bash
openspec validate enforce-change-quality-gates --strict
```

```bash
test -f docs/quality/change-quality-gates.md
for token in W0 W1 W2 W3 UI0 UI1 UI2 UI3 BLOCKED_EXTERNAL C/T/V/R; do
  rg -q "$token" docs/quality/change-quality-gates.md
done
for skill in order-plan-change order-implement-tdd order-verify-change order-integrate-change; do
  rg -q 'docs/quality/change-quality-gates.md' ".agents/skills/$skill/SKILL.md"
done
```

```bash
git diff --check 69cc9b6437dc3181681603d1bb060c07acba97f1...HEAD
git diff --name-only 69cc9b6437dc3181681603d1bb060c07acba97f1...HEAD | awk '
  !/^openspec\/changes\/enforce-change-quality-gates\// &&
  !/^docs\/quality\/change-quality-gates\.md$/ &&
  !/^\.agents\/skills\/order-(plan-change|implement-tdd|verify-change|integrate-change)\// { print; bad=1 }
  END { exit bad }
'
```

DRAFT 阶段只运行 strict 与仅规划 owned-path 检查；上述文档和 skill 检查必须等 apply 完成后再执行，不得在本轮写成已 PASS。
