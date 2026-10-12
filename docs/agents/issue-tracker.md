# 本地工件路由

本仓目前使用现有本地文档、`openspec/` 与 `.scratch/` 工件；本文不配置或假定 GitHub、GitLab 等远端 issue tracker，也不授权外部发布。

- 已进入门禁的 change：读取 `openspec/changes/<change>/proposal.md`、`design.md`、`tasks.md` 与 `specs/`；稳定规格在 `openspec/specs/`，历史 change 在 `openspec/changes/archive/`。
- Matt Skills 的讨论草稿和拆解：可在 `.scratch/<feature>/` 保存 spec 和 `issues/<NN>-<slug>.md`。这些是本地草稿，进入生命周期前按 [流程接入](skills.md#当前流程与工件接入) 映射到既有 change 工件；不得作为门禁通过、客户确认或已完成的证明。
- 技能要求 fetch ticket/spec 时，优先读取用户提供的本地路径或对应 change 工件，再按实际 Git 引用追溯来源。未配置远端映射时不得猜测 issue 编号或仓库；只有确需远端 tracker 才运行 `$setup-matt-pocock-skills` 并取得相应授权。
- 技能要求 publish 时，默认仅生成上述本地草稿；发布到外部系统、修改远端 issue 状态或发送消息须另有明确授权。不得用新草稿覆盖已确认的公共契约。

实际验证和状态按 [质量门禁](../quality/change-quality-gates.md) 与根 [AGENTS.md](../../AGENTS.md) 执行；缺少旧入口或 runner 时记录具体阻断，不宣称迁移完成或独立验收通过。
