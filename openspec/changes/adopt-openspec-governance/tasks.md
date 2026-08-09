## 1. Specification and Red Check

- [x] 1.1 Define fine-grained change boundaries, parallel ownership, TDD evidence, exact-SHA verification, and integration requirements.
  - proposal、design 和 `change-governance` capability spec 已完成。
- [x] 1.2 Run the governance baseline check and record the expected failures.
  - 基线缺少 `AGENTS.md`、OpenSpec config 和四个项目 skills，且 `CLAUDE.md` 与待采用规则重复。

## 2. Shared Governance Entry

- [x] 2.1 Add authoritative root `AGENTS.md` rules for OpenSpec, worktrees, TDD, verification, and integration.
  - 根规则明确了单能力 change、局部 Gate、单 writer、owned paths、状态流和精确 SHA 失效条件。
- [x] 2.2 Replace duplicated `CLAUDE.md` content with a thin compatibility pointer.
  - `CLAUDE.md` 只引导读取 `AGENTS.md`、`openspec/` 和 `.agents/skills/`，不再复制规则。
- [x] 2.3 Replace duplicated Cursor rules and conflicting legacy workflow skills with thin pointers to the shared governance sources.
  - Cursor always-on rule 与四个同名 skills 均为薄入口；旧 planning/TDD 名称保留兼容入口，但不再创建第二套流程。

## 3. OpenSpec Tooling

- [x] 3.1 Initialize OpenSpec for Codex and Cursor without changing product code.
  - OpenSpec 1.3.1 生成 Codex/Cursor 各四个官方入口，并以 `openspec/config.yaml` 固化项目上下文和 artifact 规则。
- [x] 3.2 Confirm generated tool instructions point to the same repository OpenSpec artifacts.
  - 两端四组 OpenSpec skills 内容一致，命令均通过仓库 `openspec/changes/` 和 CLI 读取同一 artifacts。

## 4. Project Skills

- [x] 4.1 Create and validate `order-plan-change`.
- [x] 4.2 Create and validate `order-implement-tdd`.
- [x] 4.3 Create and validate `order-verify-change`.
- [x] 4.4 Create and validate `order-integrate-change`.
  - 四个 skill 的 frontmatter、命名、描述、行数和 UI metadata 全部通过等价结构校验；官方 `quick_validate.py` 已尝试，但本机 Python 缺少其 `PyYAML` 运行依赖。

## 5. Candidate Verification

- [x] 5.1 Run strict OpenSpec validation, skill validation, link checks, and diff checks.
  - OpenSpec strict、项目 context/rules 注入、两端官方生成入口一致性、shared skills、Cursor wrappers、占位符与 diff 检查全部通过。
- [x] 5.2 Verify the candidate commit in a separate clean worktree at its exact SHA.
  - 首个候选 `b1294c9` 独立验证为 FAIL：change 自身缺少显式 owner、owned paths、依赖处置和验收命令；其余检查通过。
  - 修正候选 `d8bc3d6` 在独立 detached worktree 验证为 PASS：owner、owned paths、依赖和验收命令完整，OpenSpec strict、Skill/metadata、Cursor wrapper、owned-path、diff 与 clean-worktree 检查全部通过。
