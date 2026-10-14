# 任务
- [x] 复现：三图预览 201、提交 200、结果 503；数据库批次 93 COMMITTED。
- [x] Red：go test ./internal/cloudbasepg -run TestImportResultRPCReachesDatabase -count=1；calls=0，shared PG unavailable。
- [x] Green：补齐单项 RPC；go test ./internal/cloudbasepg ./internal/importbatch ./sharedpg 通过。
- [x] Refactor：无需重构，保留既有授权和数据库契约。
- [x] Writer：openspec validate --strict、git diff --check、go test ./services/api/... 全部通过；C/T/V/R=9/9/9/9（UI 回归作为部署后证据继续记录）。
- [x] 精确 SHA 独立 worktree 验证，详见 verification.md。
- [x] UAT 部署与后台验收矩阵完成。原批次页面状态已离开，使用同条件新三图批次 95 验证结果读取；原批次 93 入库由只读 SQL 确认，不伪记原批次 HTTP 重读通过。

- [x] 表单错误提示 Red：Node 3 例均失败；Green：已知错误码中文映射，3 例全部通过。
- [x] API 候选 2df1b3f2 在 /tmp/order-admin-import-verify 干净 detached worktree 独立重跑真实客户端测试与 sharedpg 测试，通过后构建并部署 order-api-007。

- [x] 导入后图片编辑 Red：真实弹窗 undefined、图片为零；HTTP adapter 测试复现 undefined。Green：按既有 {product} 契约解包，4 项 Node 回归通过。
