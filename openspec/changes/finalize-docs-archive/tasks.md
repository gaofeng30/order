## 1. Specification and Red Check

- [x] 1.1 Define the archive scope, integrity contract, precise deletion, and navigation acceptance scenarios.
  - OpenSpec 将移动内容保真、唯一删除项和当前入口可达性定义为独立验收条件。
- [x] 1.2 Run the pre-change navigation check and confirm it fails on the migrated paths.
  - 实施前在根 `README.md` 和 `docs/README.md` 检出 5 处指向旧目录的当前导航引用。

## 2. Archive Implementation

- [x] 2.1 Preserve the confirmed document moves and remove only `docs/design/posters/Untitled`.
  - 正式合同、合同草稿、旧版商品与展示资料、开发运维指南均按确认目录收口。
- [x] 2.2 Update repository and documentation indexes to resolve to the new locations.
  - 当前资料与历史归档在入口中分区展示，不修改归档文件内部内容。

## 3. Candidate Verification

- [x] 3.1 Compare old and new file counts and Git blob hash multisets, excluding only the confirmed deletion.
  - 旧集合排除指定空文件后为 183 个，新集合为 183 个；排序后的 blob hash 多重集完全一致。
- [x] 3.2 Validate current README links and run strict OpenSpec validation.
  - 两个当前 README 的本地链接全部存在，旧目录引用为零，`openspec validate --strict` 通过。

## 4. Independent Verification

- [x] 4.1 Verify the candidate commit in a separate clean worktree at its exact SHA.
  - 候选提交 `c54cb7f` 在 detached worktree 中通过文件完整性、唯一删除项、README 链接、OpenSpec strict 和工作区洁净检查。
