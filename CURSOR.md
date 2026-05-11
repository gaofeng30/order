# 本仓库的 Cursor 配置说明

开发者克隆本仓库后，以下配置随仓库生效，无需每人单独「安装」到用户目录。

## 位置

| 路径 | 作用 |
|------|------|
| `.cursor/rules/*.mdc` | 项目规则（持续约束与第一性原理） |
| `.cursor/skills/<name>/SKILL.md` | 项目级 Agent Skills（可复用工作流） |

个人全局技能仍可使用 `~/.cursor/skills/`，与本仓库技能并存；同名时以 Cursor 加载顺序为准，建议在对话中显式 `@技能目录名` 选用本仓库版本。

## 使用方式

1. **规则**：由 Cursor 按 `alwaysApply` 与 `globs` 自动附加，一般不必复制进聊天。
2. **技能**：在聊天中 `@技能名`（目录名）或选用「技能」相关命令；复杂任务声明将使用哪个 Skill，减少跑偏。
3. **与 [andrej-karpathy-skills](https://github.com/forrestchang/andrej-karpathy-skills) 思想一致**：短规则管边界，长技能管流程；成功标准要可验证。

新增或修改 Skill 时请遵守 [Cursor Skills 约定](https://cursor.com/docs)：根文件 `SKILL.md`、YAML frontmatter 含 `name` 与 `description`（第三人称、含触发场景）。
