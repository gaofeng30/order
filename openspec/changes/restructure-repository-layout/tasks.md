## 1. Specification and Red Check

- [x] 1.1 Define target application paths, dependency, owned paths, non-goals and migration acceptance scenarios.
  - change 只负责两个现有前端的目录边界与路径适配，不包含框架升级或后端初始化。
- [x] 1.2 Run the pre-migration checks and record the expected failures.
  - `apps/wechat-miniprogram/` 与 `apps/web-admin/` 不存在，`miniprogramRoot` 仍为旧值，Web 仍有 4 处运行时 `../miniprogram/` 引用。

## 2. Directory Migration

- [x] 2.1 Move the complete native mini program and Web Admin trees under `apps/` without dropping files.
  - 小程序 174 个文件、Web 21 个文件，迁移前后相对路径集合完全一致。
- [x] 2.2 Update WeChat configuration and Web runtime resource paths for the new sibling directories.
  - `miniprogramRoot` 指向 `apps/wechat-miniprogram/`，Web 四处运行时共享图片路径改为兄弟目录。
- [x] 2.3 Update active repository docs, source comments and image-generation tooling to the new application paths.
  - 根 README、小程序 README、当前 PRD、Web 源码注释和图片生成工具已统一使用新路径；历史归档未改写。

## 3. Candidate Verification

- [x] 3.1 Verify old/new file path sets, modified-file whitelist, content hashes and absence of stale active paths.
  - 18 个路径适配文件与白名单一致，其余 177 个应用文件 blob 完全不变，活跃旧路径为零。
- [x] 3.2 Run JS syntax, JSON parsing, mini-program page completeness and Web local-resource checks.
  - 66 个 JS、41 个 JSON、27 个小程序页面四件套和 22 个 Web 入口本地资源全部通过。
- [x] 3.3 Run strict OpenSpec validation and Git diff checks.
  - `openspec validate --strict`、README 本地链接、Python 语法和 `git diff --check` 通过。

## 4. Independent Verification

- [x] 4.1 Verify the candidate commit in a separate clean worktree at its exact SHA.
  - 候选 `d89006d` 在独立 detached worktree 验证为 PASS：174 个小程序文件与 21 个 Web 文件集合一致，18 个路径适配文件符合白名单，其余 177 个应用文件 blob 不变；JS、JSON、Python、小程序页面四件套、Web 资源、README 链接、OpenSpec strict、owned-path、diff 与 clean-worktree 检查全部通过。
