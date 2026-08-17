## Context

当前仓库已经有成熟的 OpenSpec、七态生命周期、stage skills、exact-SHA verifier、worktree 隔离、checkpoint 与受控自进化协议。缺口不是规则数量，而是这些事实没有一个低成本入口：OpenSpec CLI 只能给出 artifact/task 完成度，历史 checkpoint 不代表当前状态，main control session 的 lane ledger 也没有统一可执行格式。

该状态包含 thread/worktree/blocker 等本机调度信息，不适合写进业务 candidate 或要求每个 writer 共同编辑一个 tracked 文件；那会污染 diff、制造多 writer 冲突，并产生“状态 commit 改变 candidate SHA”的自引用问题。与此同时，产品与交付结论仍必须由 tracked OpenSpec、Git SHA、verifier、integration 与 receipt 证明，便捷账本不能升级为证明源。

## Goals / Non-Goals

**Goals:**

- 一个命令在新 session 中显示当前 checkout、active changes、每个 task 状态、blocker 与下一步。
- 同一仓库所有 worktree 共享本地控制状态，更新原子、失败关闭、容易恢复。
- 通过 observation 形成“发现摩擦 → 留证 → 模块边界筛选 → 独立 change 晋升 → 下一模块生效”的自进化闭环。
- 保持现有 OpenSpec、stage handlers、独立验证、授权和 archive receipt 为权威证明。

**Non-Goals:**

- 不做 daemon、Web UI、数据库、网络服务、CI、自动调度或 graph engine。
- 不自动批准、实现、验证、集成、archive、推送或部署。
- 不尝试从 task checkbox 猜测 lifecycle，也不把 ledger/observation 当成 PASS。
- 不清理 legacy change、不修复 README 或任何业务能力。

## Decisions

### D1. 一个 Python 标准库可执行文件

选择仓库根 `./tools/harness`，只含 `status`、`check`、`checkpoint`、`observe`。Python 标准库已经被 lifecycle receipt 工具使用，无需增加依赖、构建步骤或常驻进程。所有命令从 `git rev-parse --show-toplevel` 定位仓库，不能依赖调用目录。

未选择 Makefile/Taskfile，因为仓库当前没有统一 task runner；未选择服务或插件，因为本地文件与 Git 已覆盖目标。

### D2. 操作状态放在 Git common dir，事实证据留在 tracked repository

工具只在 `git rev-parse --git-common-dir` 返回目录下创建 `codex-harness/state.json` 和 `codex-harness/observations.jsonl`。因此同一 repository 的独立 worktree/session 共享状态，不产生业务 diff、merge 冲突或 candidate SHA 自引用。

该目录只是本地、可重建的控制索引。每个状态仍引用 tracked change、task 和 Git SHA；`status` 明示 evidence，`check` 现场验证可机械验证部分。clone 到新机器后状态缺失时显示 `UNKNOWN/NO-GO`，通过 checkpoint 从现有证据恢复，不静默猜测。

### D3. Task 状态派生，lifecycle 显式记录

工具只解析 `tasks.md` 中 `- [ ] <id>` / `- [x] <id>`，不复制 task 内容到 ledger。`DONE` 来自 checkbox，`DOING` 来自唯一 `current_task`，`BLOCKED` 来自唯一 blocked task，其余为 `TODO`。重复或失效 ID 直接失败。

Lifecycle 无法由 task 完成度可靠推断，因此 checkpoint 显式记录既有七态、evidence 和 next。工具只允许同态、相邻前进与 candidate/verifier 失败回流；`BLOCKED_EXTERNAL` 保持独立 blocker。Git SHA、ancestry、archive path 等可机械判断的条件由工具验证，人类批准和 actor independence 仍由现有 Gate 判断。

### D4. 原子写与修复导向错误

state 使用同目录临时文件、flush/fsync 后 `os.replace`；observation 在持锁后验证全文件、追加一行、flush/fsync。工具使用专属 lock 文件避免同仓库并发写覆盖。所有错误统一输出 `WHAT / WHY / FIX`，返回非零且不改原文件。

### D5. 自进化只记录，不在模块内应用

`observe` 固定四类 observation，写入唯一 ID、UTC 时间、summary、evidence、next。它没有 edit/promote/apply 子命令。模块边界由主 Goal 根据现有 self-evolution protocol 决定是否建立新的 OpenSpec DRAFT；规则只有独立验证并集成本地 main 后，才对下一模块生效。

## Risks / Trade-offs

- [本地状态不会随 clone 自动复制] → 这是有意边界；状态是调度索引，不是证明。新 clone 必须从 tracked OpenSpec/Git evidence 显式 bootstrap，不能继承旧机器的 session 假设。
- [人工 evidence 可能不真实] → 工具只保存引用并验证 Git/path/task 等可机械部分；独立验证、批准和外部事实仍由既有 Gate 决定，输出不把 ledger 称为 proof。
- [多个进程同时 checkpoint] → 使用 Git common dir 下的专属排他 lock 与原子替换；锁存在时失败，不等待或覆盖。
- [Markdown task 格式变化] → 只接受现有稳定 checkbox + task ID 语法；无法解析即 fail closed，并通过独立 OpenSpec change 演进 parser。
- [工具膨胀成新控制面] → 固定四个子命令，禁止实现调度、网络或 stage handler；根规则和 Skill 只保留最小入口。

## Migration Plan

1. 先在测试 fixture 中证明当前没有入口、缺状态、非法跃迁和损坏 observation 会失败。
2. 实现工具与单元/CLI 测试；在 writer worktree 初始化当前四个 active change 的本地状态，其中三个 legacy change 只允许通过可验证 Git ancestry 显式 bootstrap。
3. 最小更新 root governance、`order-run-loop` 和 canonical spec，使后续主 control session 开工先 `status`、转移后 `checkpoint`、收尾 `check`、经验用 `observe`。
4. candidate 在 clean detached exact-SHA worktree 从空本地状态开始，先证明 `UNKNOWN/NO-GO`，再 bootstrap fixture 并完成 fresh-session PASS。
5. 回滚只需撤销 tracked change；本地 `codex-harness/` 可保留为无调用方数据，工具不存在时不会影响代码、OpenSpec 或运行产品。

## Open Questions

无。用户已明确要求最小、提速、标准化、自进化并闭环；本设计不改变产品、公共契约、数据或授权边界。
