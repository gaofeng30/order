## Why

仓库即将由两名开发者和多个 Agent 并行推进，但当前只有业务协作原则，没有统一的细粒度规格、TDD、worktree 验证和集成闭环。若先直接拆代码，规格边界、所有权和验收证据会迅速混乱。

## What Changes

- 将根 `AGENTS.md` 设为全工具共享的强制协作规则，`CLAUDE.md` 与 Cursor rules 只保留兼容入口，避免规则复制漂移。
- 引入 OpenSpec spec-driven 工作流，并为 Codex、Cursor 生成官方工具指令。
- 明确一个 OpenSpec change 只承载一个可独立验收的能力，不允许用大 change 汇总多个需求。
- 允许无依赖、无路径冲突的 changes 在不同 worktree 并行，不设置全局串行 Gate。
- 固化每个 change 的 TDD、候选 SHA、独立 worktree 验证、SHA 失效和集成状态流。
- 新增四个仓库级 skills，分别负责规划、TDD 实现、独立验证和集成。

## Capabilities

### New Capabilities

- `change-governance`: 定义团队如何拆分、实现、验证和集成一个独立 OpenSpec change。

### Modified Capabilities

无。

## Impact

- owner：`Governance Writer`，即 `codex/adopt-openspec-governance` branch/worktree 的唯一 writer；验证者只读，不修改候选文件。
- owned paths：
  - `AGENTS.md`、`CLAUDE.md`
  - `openspec/config.yaml`、`openspec/changes/adopt-openspec-governance/**`
  - `.agents/skills/order-*/**`
  - `.codex/skills/openspec-*/**`
  - `.cursor/commands/opsx-*.md`
  - `.cursor/rules/first-principles-cursor.mdc`、`.cursor/rules/karpathy-guidelines.mdc`
  - `.cursor/skills/openspec-*/**`、`.cursor/skills/order-*/**`
  - `.cursor/skills/planning-with-files/SKILL.md`、`.cursor/skills/test-driven-development/SKILL.md`
- 依赖：无；基线为 `main@6c8848305ca152d62a544c112808a05b2e1797f6`，不依赖文档归档或目录重构 changes。
- 不影响：产品需求、客户端行为、服务端接口、业务数据和部署环境。
- 所有后续研发 change 都必须遵循本规范；本 change 之前的原型代码不追溯补规格。

### Acceptance Commands

```bash
openspec validate adopt-openspec-governance --strict
```

```bash
for name in apply-change archive-change explore propose; do
  diff -q ".codex/skills/openspec-$name/SKILL.md" ".cursor/skills/openspec-$name/SKILL.md"
done
```

```bash
ruby -e 'require "yaml"; Dir[".agents/skills/order-*/SKILL.md"].each { |f| data = YAML.safe_load(File.read(f).split("---", 3)[1]); abort(f) unless data["name"] && data["description"] }'
git diff --check main...HEAD
```

独立验证还必须在 detached worktree 确认 `git rev-parse HEAD` 等于候选 SHA、`git status --porcelain` 为空，并按本 change tasks 执行完整的 skill metadata、Cursor wrapper 和 owned-path 检查。
