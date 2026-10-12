> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Design

UAT `newCloudSecretSource` 返回无缓存、无可变共享状态的 CloudRun 环境变量 source。source 只允许四个完整的 `order-uat-*` logical name，固定映射到四个平台变量。拒绝 nil/已取消 context、未知/production 用途、缺值/空值；错误只返回稳定非敏感类别。不存在 SSM/CVM/文件 fallback，不从 caller 拼接任意环境变量名。

`config.Load` 仍验证 region、结构化数据库字段、AppID、值长度/控制字符和 cloud TLS required；只在 UAT 去掉数据库密码与 AppSecret 两个业务变量的旧禁令。生产维持原禁止业务 env 条件；两环境继续拒绝 DSN、永久腾讯 ID/key/token/credential file。支付/核销材料保持单份 JSON，由现有 parser 严格验证，再生成原生支付 client 和版本化 AES key。已有 `UsesCloudProviders` 与三个 cmd 调用路径不变。

云身份与业务材料来源分别承担现有职责：UAT COS 仍须显式 `CLOUDBASE_ENV_ID`/`CLOUDBASE_APIKEY` 换取完整短期 tuple。平台业务变量不能替代腾讯身份。生产 source、身份与部署不变；不宣称平台变量具有 SSM 的集中细粒度 secret ACL、访问审计或自动轮换能力。

服务 revision 的平台配置固定当前运行值；应用启动时保存已经解析的配置/材料。轮换必须更新平台变量、产生新 revision/进程，并核实实际读取与旧 revision 流量收敛。旧 revision 可能仍含旧值；不承诺动态刷新或旧值自动撤销。恢复须使用仍有效的同环境配置；已撤销材料不能通过退回旧 revision 恢复。缺值/非法值修复后重启或重新调用公开 loader 可恢复，失败期间无替代来源。

当前 CloudRun v1/v2 canary 仅证明所测小体积合成值注入与更新，不证明完整支付形态平台上限、真实客户材料、PG/TLS 或任何真实资金。完整合成材料云 probe 绑定本切片新 SHA，由 root 在清理后复读确认服务不存在；记录摘要不包含 env 值、PEM、APIv3/key 或可重放 token。

没有业务/数据/事务/并发算法改动，故真实金融/PG W3 结果属于独立 full-UAT 任务。本切片的 W3 配置安全 Gate 使用原生加密消费者、失败关闭/恢复、并发只读、现有 TLS 和 COS 回归，避免将 mock 材料测试当作资金或存储验收。
