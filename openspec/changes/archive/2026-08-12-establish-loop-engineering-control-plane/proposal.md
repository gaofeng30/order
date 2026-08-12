## Why

仓库已有按单个 change 执行规划、TDD、独立验证和集成的四个职责 skill，但缺少一个面向长任务的薄控制面来选择下一阻断项、限制并行 lane、约束 session 复用并依据证据判断停止。MVP 准备工作进入多 change 阶段后，若继续由主会话临时编排，容易产生 session 泛滥、过早验证、重复失败和用评分掩盖硬阻断。

## What Changes

- 新增 `order-run-loop` 薄编排 skill，只负责控制面决策，并按 change 状态路由到现有 `order-plan-change`、`order-implement-tdd`、`order-verify-change` 和 `order-integrate-change`；不复制四个 skill 的执行规则。
- 固化主 Goal、最多两条 change lane、单 writer、session/worktree 复用、精确 SHA verifier、失败熔断、外部阻塞、主动回传和等待策略。
- 固化与仓库七态一致的进入/退出证据，以及 100 分开发准备度 rubric 和不可被评分覆盖的硬停止条件。
- 使用 `skill-creator` 的 `init_skill.py` 初始化 skill 与 `agents/openai.yaml`，并使用 `quick_validate.py` 验证；`default_prompt` 显式调用 `$order-run-loop`。
- 不修改根治理、现有四个 order skills、业务代码、产品文档或任何外部系统。

## Capabilities

### New Capabilities

- `loop-engineering-control-plane`: 定义 order 主 Goal 如何以有界 session、证据门禁和固定评分循环调度独立 OpenSpec changes。

### Modified Capabilities

无。

## Impact

- 状态：`CANDIDATE`。
  - `APPROVED` 证据（2026-08-12）：主 Agent 基于薄编排边界、主+2 lane、冲突串行、exact-SHA verifier、三次失败升级、固定 rubric 与硬 Gate 明确批准；授权来源为主 Goal `019ff620-bb1e-7702-b305-b1dd7c6651ca`。
  - `IMPLEMENTING` 证据（2026-08-12）：依赖、branch/worktree、owned paths 与 strict Gate 通过后，结构性 Red `test -f .agents/skills/order-run-loop/SKILL.md && test -f .agents/skills/order-run-loop/agents/openai.yaml` 返回 exit 1。
  - `CANDIDATE` 证据（2026-08-12）：writer tasks 1.1-4.3、quick validation、strict validation、metadata/contract checks 与 owned-path audit 全部 PASS；完整候选 SHA 和 clean status 由产生本记录的本地 commit 及主动回传给出，独立验证尚未开始。
- owner：`codex/establish-loop-engineering-control-plane` 当前 worktree 的唯一 writer；其他 worker 不得写入 owned paths。
- owned paths：
  - `openspec/changes/establish-loop-engineering-control-plane/**`
  - `.agents/skills/order-run-loop/**`
- 只读共享契约：根 `AGENTS.md`，以及 `.agents/skills/order-plan-change/**`、`order-implement-tdd/**`、`order-verify-change/**`、`order-integrate-change/**`；本 change 不修改这些文件。
- 依赖：已集成的 `adopt-openspec-governance`；基线为 `c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af`。本 change 不依赖任何业务 change，也不阻塞 owned paths 无冲突的本地 change。
- 非目标：实现业务能力、修改产品范围、创建主 Goal 或其他 session、创建 verifier、推送、部署、安装工具、写入外部系统。
- 最小成功标准：skill 结构、metadata、薄引用、状态机、评分表、session/升级/停止规则和 owned-path 验收全部通过，形成仅含 owned paths 的本地 `CANDIDATE` commit；独立验证、集成和归档由后续角色执行。

### Acceptance Commands

```bash
openspec validate establish-loop-engineering-control-plane --strict
```

```bash
python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop
```

```bash
for name in order-plan-change order-implement-tdd order-verify-change order-integrate-change; do
  rg -q "\\$$name" .agents/skills/order-run-loop/SKILL.md
done
```

```bash
git diff --check c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af...HEAD
git diff --name-only c47135b660a9ca3f9f9ee6ded6b09fbf0ee6f1af...HEAD | awk '
  !/^openspec\/changes\/establish-loop-engineering-control-plane\// &&
  !/^\.agents\/skills\/order-run-loop\// { print; bad=1 }
  END { exit bad }
'
```
