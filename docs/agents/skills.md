# Matt Skills

- 仓库 Skill 位于 `.agents/skills/<skill-name>/SKILL.md`。
- Codex 显式调用使用 `$skill-name`，例如 `$wait-what`。`/wait what` 会被当成 slash command，不会调用 Skill。
- Skill 刚安装或更新后，从仓库根目录新建 Codex 任务；如仍未出现，重启 Codex 后再试。
- 常用入口：`$ask-matt`、`$grill-with-docs`、`$to-spec`、`$to-tickets`、`$tdd`、`$implement`、`$code-review`、`$wait-what`。
- 需要初始化本地 tracker 或 domain 约定时，运行 `$setup-matt-pocock-skills`。

Skills 来自 `mattpocock/skills` 固定提交 `5b15a47f2d7150f545fbcacbfe381787fc0230dc`，许可见根目录 `THIRD_PARTY_NOTICES.md`。
