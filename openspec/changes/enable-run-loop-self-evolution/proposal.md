## Why

`order-run-loop` 目前能有界调度 change，但没有把执行经验冻结、分类、筛选、前测并延迟到下一模块生效的受控晋升协议。下一阶段要先补齐这一控制面，避免 runner 在模块执行中自改规则、降低 Gate，或把偶发环境问题固化为长期流程。

## What Changes

- 为每个模块固定 runner base（仓库 SHA、Skill git blob、内容 SHA256 和显式版本状态），模块内只追加 observation，不改当前模块规则。
- 把 observation 固定分类为 `candidate`、`environment`、`checker`、`external`。模块边界只有已记录可复现、可泛化或安全关键、不降低 Gate 的意图，以及可执行 regression/forward-test 计划的 candidate，才能排入或创建 dedicated OpenSpec `DRAFT`；进入 `DRAFT` 不等于晋升。
- dedicated change 实现后，只有重新确认上述事实、regression PASS、clean detached exact-SHA 独立最小上下文 fresh-session forward-test PASS 且完整 independent verification PASS，规则才算晋升；缺一项均不得晋升。
- 晋升规则只有集成本地 main 后才从下一模块生效；当前模块继续使用固定 runner base，所有状态可从仓库 checkpoint、OpenSpec tasks 和精确 SHA 恢复。
- 将 checker 的跨平台约束固化为可回归检查的七项契约：零匹配计数、Markdown 字段解析、工具层反引号、`awk` 大小写、健康恢复有界轮询、安全临时目录、archive 尾随换行。
- 保持 `SKILL.md` 为薄控制面；自进化详细协议只放在一层 `references/` 中，不复制 `$order-plan-change`、`$order-implement-tdd`、`$order-verify-change`、`$order-integrate-change`。
- writer 通过 strict、Skill quick validation、metadata 一致性、旧路由不漂移和 regression checker 后形成 exact candidate；随后由 clean detached exact-SHA verifier 独立执行 minimal-context fresh-session forward-test 与完整验证，最后才允许纯 fast-forward 本地集成和归档。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `loop-engineering-control-plane`: 增加模块级 runner base、observation 分类与晋升 Gate、下一模块生效、可恢复 checkpoint、跨平台 checker 契约及独立 forward-test 要求；并将原“Skill 最终不得包含 references”调整为“`SKILL.md` 精简、详细自进化规则只放一层 references”。

## Impact

- 状态：`CANDIDATE`。2026-08-13 主 Agent 基于 4/4 artifacts 一致、strict PASS、base/owner/branch/worktree/owned paths/dependencies 匹配、架构循环已消除、Open Questions 为 0、Skill 尚未修改且硬阻断为 0 明确批准；批准后 writer 完成真实 Red/Green/Refactor、全部 writer Gate 和本地 candidate commit。完整 candidate SHA 与 clean status 由 commit 后的 `git rev-parse HEAD`/`git status --porcelain` 提供且不再写回 candidate artifacts；尚未独立验证、集成或归档。
- 唯一 outcome：`order-run-loop` 能在不改变当前模块规则、不降低现有 Gate 的前提下，把合格 observation 受控晋升为下一模块 runner 规则。
- acceptance verdict：四件规划 artifact 一致且 strict PASS 后仅可进入待批准 `DRAFT`；实现阶段只有全部本地 Gate PASS、`C/T/V/R >= 36` 且每项 `>= 8`、硬阻断为 0，才可形成 `CANDIDATE`。candidate 尚未晋升；晋升仍要求其 exact-SHA 独立 forward-test 与完整 verification PASS。
- `gate_type`: `W1`（改变内部控制面行为，不改变业务公共 API、持久化数据、资金、权限或并发结果）。
- `ui_level_target`: `UI0`；无用户界面，实际等级由实现证据记录，UI1/UI2/UI3 不适用且不得写 PASS。
- owner：branch `codex/enable-run-loop-self-evolution`、worktree `/Users/vivix/.codex/worktrees/order-run-loop-self-evolution.Writer` 的唯一 writer。
- owned paths 仅限：
  - `openspec/changes/enable-run-loop-self-evolution/**`
  - `.agents/skills/order-run-loop/**`
- 只读共享契约：根 `AGENTS.md`、`docs/quality/change-quality-gates.md`、`openspec/specs/loop-engineering-control-plane/spec.md`、`openspec/specs/change-quality-gates/spec.md` 和四个现有 `$order-*` stage skills。
- dependencies：当前本地 `main@2209c071a21860231827b2a8c8c81d9b7745e6e1` 已集成并归档 `loop-engineering-control-plane` 与 `change-quality-gates`；无未满足 change 依赖。后续 `connect-miniprogram-menu-catalog` 必须等待本 change 实际 `INTEGRATED` 到 main。
- required external assets：无；forward-test 使用本地隔离临时目录与独立 fresh session，不需要账号、密钥、网络、真实平台或外部写入。
- non-goals：修改根治理或四个 stage skills；修改 Go/小程序/业务代码；在本 change 接入菜单目录；模块执行中自改 runner；新增重复 Skill；降低或绕过既有 Gate；push、PR、deploy、购云或外部系统写入。
- base：repo SHA `2209c071a21860231827b2a8c8c81d9b7745e6e1`；当前 Skill git blob `d529461de5af1bf7cc65562e59ec3c84f0750963`；Skill SHA256 `558b549a4410d72d4c22acad621ffae96af3aeccd26adc186ede76601097aa59`；legacy front matter 无 `version` 字段。
- 计划验收命令：
  - `openspec validate enable-run-loop-self-evolution --strict`
  - `/usr/bin/python3 /Users/vivix/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/order-run-loop`
  - `/usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/verify_contract.py --repo .`
  - `/usr/bin/python3 openspec/changes/enable-run-loop-self-evolution/checks/run_checker_regressions.py`
  - 独立 fresh session 在 clean detached candidate worktree 按 `forward-test.md` 执行最小上下文场景，再用 `verify_forward_test.py` 校验脱敏结果；writer 自测不得替代该独立证据。
  - `git diff --check 2209c071a21860231827b2a8c8c81d9b7745e6e1...HEAD` 与固定 owned-path audit。
  - exact-SHA verifier 在另一 clean detached worktree 重跑全部声明 Gate；集成人仅在授权后验证 `git merge-base --is-ancestor main <candidate-sha>` 且以纯 fast-forward 集成，归档后 strict PASS。
