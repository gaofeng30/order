# Order UAT 实际云端复核

当前官方 MCP 账号授权 READY，固定环境 order-uat-d1gtzpzj50615c300；已只读查到环境 NORMAL、上海地域、个人版，到期2026-10-31，PAYG OFF、自动续费 OFF。

实际 PG17.11 数据库 public 业务表0，CloudRun unopened。MCP SQL API 可读不等于 Go 原生连接可用；共享 PG 的后台 max_connections2048 不代表此租户连接预算。原生地址、TLS、运行角色与连接预算仍待实测。

控制台点击独享 PG 升级实际提示须先升标准版，待确认报价80.24元仅覆盖本周期标准版补差；后续独享 PG 规格和费用仍须核实。尚未购买或升级。

部署适配还需单独的 UAT scope：当前 Go 使用 CVM role/SSM，与已确认 CloudBase/排除 SSM 方案不符；官方开发文档证明运行时可能提供 HTTP header/环境变量凭证，但没有证明当前 Go provider 可用或独立 worker持续获CPU。不能猜测角色、复制永久密钥或降低 TLS。

微信开发者工具登录与正式 AppID wx15e136b256d4dc5a 可用；体验版编译/上传可由 agent执行。公众号/小程序管理后台 mp.weixin.qq.com 的浏览器工具实际拒绝仍是当前限制，须人操作正式域名/模板/体验成员等后台设置。扫码、手机号、支付、核销与订阅通知仍需真实账号/真机最终结果。

客户需完成的责任项包括：云费用额度/条款、PG密码设置或重置、人身扫码/验证码、客户管理员及商户资料、受控真实支付额度与真机验收。技术配置、迁移、部署、代码修复、日志及数据核对可由 agent在已授权范围内完成。自定义域名 ICP 为用户明确排除项。

官方参考：[服务开发说明](https://docs.cloudbase.net/run/develop/developing-guide)、[原生PG连接](https://docs.cloudbase.net/database/postgresql/connecting-to-postgresql)。文档说明与当前环境实测分开，原始脱敏只读结果摘要见 cloud-runtime-check.json。
