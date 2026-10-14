# 简化读取链路与支付恢复

## Why
用户已授权按诊断最小完整修复加载慢和同类读取问题，接受上传验证/固定版本签名、图片失败不阻断菜单的方向。

## What Changes
- 修复已有预支付恢复被当前售罄状态阻断。
- 轻量可售餐段查询，当前菜单只读一次。
- 图片批量固定版本签名，浏览不重下原图；上传与业务引用确认保留校验。
- PC服务端状态计数、统计候选范围过滤、收款导出最小投影。

## Impact
base_sha: `0aba16b086bd3d2770e7dce41ccb86311e697587`; owner: Codex; branch: `codex/order-read-performance`; gate_type: W3; ui_level_target: UI3; ui_level_actual: UI0。
owned paths: apps/wechat-miniprogram、apps/web-admin/data/api.js、services/api/internal/{menu,catalog,objectstore,adminreport}、services/api/sharedpg、services/api/internal/cloudbasepg/client.go及SharedPG登记/发布SQL、当前change及性能验证工件。共享文件单writer。
依赖：部署SharedPG函数、API与前端必须同一发布窗口。无新资源、缓存平台或公开SLA承诺。
本次授权为实现及本地验证；不含push/PR/部署/生产写入。真机及真实支付UI3未运行不得记通过。
