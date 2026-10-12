# Matt Skills

- 仓库 Skill 位于 `.agents/skills/<skill-name>/SKILL.md`。
- Codex 显式调用使用 `$skill-name`，例如 `$wait-what`。`/wait what` 会被当成 slash command，不会调用 Skill。
- Skill 刚安装或更新后，从仓库根目录新建 Codex 任务；如仍未出现，重启 Codex 后再试。
- 常用入口：`$ask-matt`、`$grill-with-docs`、`$to-spec`、`$to-tickets`、`$tdd`、`$implement`、`$code-review`、`$wait-what`。
- 需要初始化本地 tracker 或 domain 约定时，运行 `$setup-matt-pocock-skills`。

## 当前流程与工件接入

- 根 [AGENTS.md](../../AGENTS.md) 定义硬规则，[质量门禁](../quality/change-quality-gates.md) 定义适用验证与生命周期要求。旧 OpenSpec slash command / Skill 入口已退出；`openspec/` 的规格、change 和 `tools/harness` 依赖仍在使用，未完成工具或门禁迁移。
- `$ask-matt`、`$to-spec`、`$to-tickets` 等负责讨论、草稿和拆解；按 [本地工件路由](issue-tracker.md) 查找或保存来源。进入 gated change 前，把已授权范围及验收映射到现有 `openspec/changes/<change>/{proposal,design,tasks}.md` 与 `specs/`；草稿与 tickets 不单独批准 change，不替代这些工件。
- `$implement` 与 `$tdd` 执行已授权范围内的实现、自检和修复，遵循完整 Red → Green → Refactor。提交前 Writer Gate 通过后才形成本地候选 SHA；`$code-review` 是代码审查，不替代另一个干净 detached worktree 的 exact-SHA 独立验收。
- 独立验收失败返回 writer 修复并生成新 SHA。集成、归档与外部操作继续遵循现有门禁及用户授权；Skill 切换不产生新的发布授权，也不要求重复确认已有决策。

Skills 基于 `mattpocock/skills` 固定提交 `5b15a47f2d7150f545fbcacbfe381787fc0230dc`，包含 Order 的仓库适配；许可见根目录 `THIRD_PARTY_NOTICES.md`。
