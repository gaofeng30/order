# Design

一份中文工程操作指南按发布顺序组织。先验私网/TLS、配置容量及发布版本，再迁移、首次初始化、API/readiness、静态/HTTP网关、体验版和验收放量；回退遵守 schema 兼容、forward-only 和外部资金事实。

以代码事实列完整变量与材料格式，不依赖简略 env 示例。API 自带 worker，服务常驻且数据库连接预算包含发布并存。PC 固定同源 `/api/v1`，采用 HTTP 网关根静态与 `/api` CBR 路径透传。当前生产批准架构为 CVM/MySQL/SSM，明确其与 UAT 的差异，不把 CloudBase 生产资源存在当作生产迁移已授权或通过。

文档只写正确步骤和可验收条件；诊断尝试、时间线、测试探针和实时未完成状态保存在仓库外事实回执。5 KB 是已观察的目标配置接口限制，不虚构套餐升配可解除或所有生产已实测。

W0 验证：缺文件形成真实 Red；Green/Refactor 用同一文件检查，并检查完整代码变量、材料结构、相对链接、Bash/Dockerfile入口、部署/生产/回退边界和 owned diff；OpenSpec strict。Go/云写入/UI/真实部署检查 N/A（没有修改对应实现）。提交本地候选后，不同 verifier 在 clean detached 精确 SHA 只读重跑文档检查。候选与后续关闭状态留仓库外 receipt，避免自引用 SHA 修改候选。
