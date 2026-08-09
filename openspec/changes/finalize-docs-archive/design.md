## Context

历史资料已由仓库维护者移动到三个用途明确的目录：正式合同、旧版商品与展示资料、微信小程序开发和运维指南。合同草稿另行进入 `docs/archive/contracts/`。当前问题是旧导航仍指向原目录，而且大量二进制文件的移动需要可重复验证，不能仅凭 Git 的重命名展示判断是否完整。

## Goals / Non-Goals

**Goals:**

- 固化已经确认的文档分类和唯一删除项。
- 让仓库级文档入口只指向现存路径，并明确区分当前文档与历史资料。
- 用 Git blob hash 的多重集比对证明移动前后文件内容及数量一致。

**Non-Goals:**

- 不修改合同、指南、图片、压缩包、脚本或 manifest 的内部内容。
- 不处理大文件外部存储、Git LFS 或历史压缩。
- 不调整小程序、Web 管理端和后端目标目录。

## Decisions

### D1. 保留已确认的分类目录

正式合同保留在 `docs/合同相关/`，合同草稿进入 `docs/archive/contracts/`，旧版商品与展示资料保留在 `docs/商品列表和展示（旧版已归档）/`，运维指南保留在 `docs/微信小程序开发和运维指南/`。本 change 不再进行二次命名或层级重构，避免扩大范围。

### D2. 通过 blob hash 多重集验证移动完整性

将 `main` 中六个旧资料目录下的全部文件 blob hash，与四个新目录下的全部文件 blob hash 排序比较；旧集合仅排除已确认删除的 `docs/design/posters/Untitled`。选择内容 hash 而不是依赖 `git diff` 的 rename detection，因为后者受相似度阈值影响，不能作为完整性证明。

### D3. 只修正文档入口，不改写归档内容

根 `README.md` 和 `docs/README.md` 属于当前导航入口，必须更新。归档中的 README、manifest 和脚本即使包含历史绝对路径，也保持原样，以保证归档内容可追溯且 hash 可验证。

### D4. 唯一删除项按精确路径限定

只删除空文件 `docs/design/posters/Untitled`。名称中包含 `Untitled` 的其他历史材料属于移动集合，不因名称相似而删除。

## Risks / Trade-offs

- [Git 可能把移动显示为删除加新增] → 以 blob hash、文件数量和精确路径验收，不依赖界面展示。
- [中文目录对命令行引用要求更高] → 所有自动检查使用 NUL 分隔或精确引用路径。
- [归档内容内仍有历史绝对路径] → 明确将归档内容视为只读历史证据，只保证当前入口无旧路径。

## Migration Plan

1. 建立 OpenSpec 契约和验收任务。
2. 修正当前 README 导航。
3. 验证移动文件 hash 多重集、指定删除项和所有当前导航链接。
4. 提交候选版本，并在独立 worktree 对精确提交 SHA 复验。

回滚时整体撤销本 change 的提交即可恢复原目录布局和导航。

## Open Questions

无。
