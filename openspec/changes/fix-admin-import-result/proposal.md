# 修复后台导入结果读取并补充验收

## Why
UAT 批量导入提交成功后，结果读取返回 503；真实客户端漏登记 order_import_result_v1，使已入库数据被展示为导入服务不可用。此前单图上传验收未覆盖导入链路。

## What Changes
- 将已部署、已授权的导入结果 RPC 接入现有客户端。
- 修正本轮实测发现的表单/导入英文错误提示，保持错误码和业务行为。
- 通过真实 HTTP 客户端回归测试锁定漏登记错误。
- 补跑 PC 后台主流程及异常场景，逐项记录通过、失败和未运行边界。

## Impact
- W1 / UI2；不改变数据库权限、数据模型、导入业务或生产环境。
- Owned paths: services/api/internal/cloudbasepg/{client.go,import_result_test.go}、apps/web-admin/data/api.js、tools/tests/pc-admin-errors.test.cjs、本 change 文档。
- 外部资产：既有 UAT order-api 服务与测试数据；发布沿用本会话授权。
- 用户本轮已明确要求复现、修复和后台 E2E。
