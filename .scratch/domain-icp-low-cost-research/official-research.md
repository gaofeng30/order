# Order 小程序域名与备案最低成本官方研究

核对日期：2026-09-02  
范围：工信部、微信开放文档、腾讯云官方资料。价格均为当前官方公开价；动态计价项以购买控制台最终结算为准。

## 结论：推荐的单一路径

1. 在**客户企业主体实名的腾讯云主账号**注册一个简短 `.cn` 域名，域名持有人填写营业执照上的企业全称。不要用开发者个人持有，也不要购买多个域名。
2. 复用本项目正式 API 所需的**一台中国大陆 CVM**作为备案云资源，不再购买“备案专用服务器”。购买时选包年包月至少 3 个月、实例自带中国大陆公网 IP、带宽大于 0；不要使用按量、竞价、试用或 EIP 作为备案资源。
3. 域名实名认证完成并等待 3 个自然日后，先通过腾讯云提交网站/域名 ICP 备案；PC Admin 是网页服务，因此按真实业务提交一个网站服务，备案二级主域名即可，随后 `order.<主域名>`、`api.<主域名>` 等子域名可使用。
4. 腾讯云域名备案完成后，再在微信公众平台提交**微信小程序备案**。不要同时在两个渠道跑进行中的备案订单；腾讯云官方说明，其他服务商正在办理的备案（含小程序/快应用）可能导致“网站主办者冲突”。
5. ICP 通过后再把域名解析到中国大陆 CVM；使用 DNSPod 免费解析。给实际供小程序请求的一个精确主机名申请腾讯云免费单域名 SSL 证书，并在微信后台“开发 - 开发设置 - 服务器域名”配置 `https://...` 的 request 合法域名。
6. 网站/PC Admin 开通后 30 日内提交公安联网备案。若最终真的只有 API、没有任何面向用户的网页，腾讯云官方说可暂不办理；Order 有 PC Admin 网页，因此推荐直接办理，不把例外作为常态方案。

该路径只买**一个域名 + 原本就需要的一个 CVM**；ICP备案、DNS 和首期 HTTPS 不增加费用。

## 官方硬规则

### 域名和腾讯云备案资源

- 腾讯云普通 ICP 备案服务本身不收费；成本来自备案所需云资源。[腾讯云：备案产品定价](https://cloud.tencent.com/document/product/243/18793)
- 可用于腾讯云备案的 CVM 必须位于中国大陆、包年包月购买累计不少于 3 个月、备案期间剩余有效期不少于 1 个月、绑定实例公网 IP 且带宽不为 0；EIP、竞价实例不满足该路径。[腾讯云：备案云资源](https://cloud.tencent.com/document/product/243/18908)
- 单位备案时，域名所有者通常需要与备案主体企业或法定代表人一致；域名实名完成满 3 个自然日（非腾讯云注册为 3 个工作日）后才能提交。[腾讯云：备案限制说明](https://cloud.tencent.com/document/product/243/18911)
- 网站备案只备案二级主域名，完成后其三级、四级子域名可以使用。[腾讯云：准备 ICP 备案域名](https://cloud.tencent.com/document/product/243/18905)
- 未完成备案前，域名不得解析到腾讯云中国大陆服务器对外访问；否则可能被拦截或影响审核。[腾讯云：备案场景](https://cloud.tencent.com/document/product/243/18910)

### 微信小程序

- 微信小程序只能与后台配置过的通讯域名通信；`wx.request` 只支持 HTTPS，不能使用 IP 或 localhost，域名必须完成 ICP 备案。[微信开放文档：网络](https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html)
- HTTPS 证书必须有效、受系统信任、域名匹配、证书链完整，TLS 至少支持 1.2；iOS 不接受自签名证书。[微信开放文档：网络](https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html)
- 小程序备案包含信息填写、微信平台初审、工信部短信核验、管局审核；微信公开说明平台初审通常 1–2 个工作日，管局 1–20 个工作日。管局通过后才能进入版本发布步骤；发布后备案号由微信自动展示。[微信开放文档：小程序备案操作指引](https://developers.weixin.qq.com/miniprogram/product/record/record_guidelines.html)
- 工信部规定，包含小程序等分发形态在内，在境内提供互联网信息服务需履行备案，材料齐全准确时省管局应在 20 个工作日内办理。[工信部：开展移动互联网应用程序备案工作的通知](https://www.miit.gov.cn/jgsj/xgj/wjfb/art/2023/art_dd783a581c9644a4aee10afa582811db.html)

### 公安联网备案

- 网站/APP 在工信部备案通过并正式开通后，应在 30 个自然日内提交公安联网备案。[腾讯云：公安联网备案常见问题](https://cloud.tencent.com/document/product/243/19616)
- 腾讯云明确列出两种可暂不办理的情形：网站未实际上线、无法公网访问；或网站只提供 API、没有面向用户的网页。[腾讯云：新增/接入服务](https://cloud.tencent.com/document/product/243/97670)
- 本项目会提供 PC Admin 网页，因此推荐按“需办理”执行；这属于基于项目形态的判断，不是官方对 Order 的个案裁定。

## 当前官方可见价格表

| 项目 | 当前官方可核实价格 | 本项目处理 | 价格边界/官方依据 |
|---|---:|---|---|
| `.cn` 域名注册 | 33 元/首年 | 购买 1 个 | 腾讯云标准价；特殊/白金域名、活动价除外，以结算页为准。[域名价格](https://buy.cloud.tencent.com/domain/price) |
| `.cn` 域名续费 | 38 元/年 | 次年开始 | 开启自动续费可享指定后缀续费 9.5 折，但即时扣费后不可退款。[域名价格](https://buy.cloud.tencent.com/domain/price)、[自动续费说明](https://cloud.tencent.com/document/product/242/10525) |
| DNSPod 解析 | 0 元 | 免费版 | 免费版无 SLA；MVP 只需 A/CNAME 记录。[版本对比](https://cloud.tencent.com/document/product/302/106264) |
| 普通 ICP 备案 | 0 元 | 必做 | 腾讯云明确“不收取费用”，但必须有合规备案资源。[备案产品定价](https://cloud.tencent.com/document/product/243/18793) |
| 备案用 CVM | **复用业务 CVM，备案增量 0 元** | 必做、不得另买备案机 | 实际总价随地域、机型、磁盘和带宽变化，必须在 [CVM 价格计算器](https://buy.cloud.tencent.com/pricing/cvm/calculator) 对最终选型询价。首次使用预付费 CVM 备案可按规则获赠备案耗时、最多 45 天；轻量服务器不参加。[备多久送多久](https://cloud.tencent.com/document/product/243/18912) |
| 仅作为备案资源的最低公开替代 | Lighthouse 2核2G/40GB/2Mbps 为 35 元/月，满足最低 3 个月即 105 元 | **不推荐另购** | Lighthouse 官方基础套餐价且也是合规备案资源；但项目本来需要业务 CVM，另买会重复支出。[轻量应用服务器价格](https://cloud.tencent.com/document/product/1207/73452)、[备案云资源](https://cloud.tencent.com/document/product/243/18908) |
| HTTPS 证书 | 0 元/张，90 天有效 | UAT/MVP 用 1 张精确单域名证书 | 免费证书不支持通配符/多域名，需到期前重新申请和部署；微信只要求有效且受信任，并未要求付费证书。[免费 SSL 概述](https://cloud.tencent.com/document/product/400/89868)、[免费证书续期](https://cloud.tencent.com/document/product/400/61353) |
| 公安联网备案 | 官方流程未列收费项；固定服务价格 N/A | 网站开通后 30 日内办 | 行政备案，不购买第三方代办；以公安平台实际页面为准。[公安联网备案流程](https://cloud.tencent.com/document/api/243/19142) |
| 微信小程序备案 | 微信官方指引未列收费项；固定服务价格 N/A | 必做 | 直接在微信公众平台办理，不购买第三方代办；以后台实际提交页为准。[微信小程序备案指引](https://developers.weixin.qq.com/miniprogram/product/record/record_guidelines.html) |
| 微信认证 | 与域名/ICP无直接关系；官方公开页未核到一个可适用于当前主体的固定现价 | 按小程序后台当前订单核价 | 主体类型和后台资格会影响实际路径；不得引用非官方历史“300元”等数字代替当前订单价。 |
| 微信支付商户申请 | 与域名/ICP无直接关系；固定申请价格 N/A | 后续真实支付联调时申请 | 支付费率以商户平台签约产品和合同显示为准，不把费率当成域名成本。 |
| COS 私有桶 | 新企业用户标准存储 1TB、6 个月免费；请求和流量不在免费范围 | 与 CVM 同地域、按量 | 广州示例标准存储 0.118 元/GB/月、外网下行 0.5 元/GB，实际按地域和计费项结算。[COS 免费额度](https://cloud.tencent.com/document/product/436/6240)、[COS 成本优化](https://cloud.tencent.com/document/product/436/50201) |
| TencentDB MySQL 8.0 | 官方公开最低“25 元/月起”，最终价非固定 | 购买页对实际地域/规格询价 | 具体价格由架构、规格、存储、地域、计费模式组成；官方要求登录后获取准确价格。最低宣传价不等于本项目订单价。[MySQL 产品优势](https://cloud.tencent.com/document/product/236/5148)、[计费概述](https://cloud.tencent.com/document/product/236/18335) |

### 可落地的最低增量预算

- 已经要购买业务 CVM 的前提下：首年确定的域名增量为 **33 元**；DNS、ICP备案、首张 90 天 SSL 证书为 0 元。
- CVM、CDB、COS 请求/流量属于运行环境成本，不是“域名备案费”，必须在选定地域和规格后从控制台形成一次实际订单报价。
- 若尚未确定业务 CVM，不能把 Lighthouse 的 105 元替代价与 CVM 同时购买；两者只选一个备案资源。当前 Order 代码和部署目标已按 CVM 设计，推荐保留 CVM。

## 备案期间可以继续做什么

### 可以继续

- 本地单元测试、接口集成测试、数据库迁移测试、微信开发者工具编译与自动化。
- 微信开发者工具可临时开启“不校验请求域名、TLS 版本及 HTTPS 证书”；手机开启调试模式时也可跳过域名校验。[微信开放文档：网络](https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html)
- 使用本机 API、测试夹具和本地数据库完成开发回归。

### 不能视为线上提测通过

- 跳过域名校验只适用于开发者工具或手机调试模式，不能证明体验版、审核版、正式版的真实 HTTPS 链路可用。
- ICP 未通过前，不应把未备案域名解析到中国大陆 CVM 开站；腾讯云可能拦截，且存在影响备案审核的风险。
- 没有微信小程序备案号时不能把“代码已上传”表述为“具备发布条件”。

## 时间预期

官方上限不能压缩成“明天一定完成”：

- 域名实名认证同步：腾讯云注册域名满 3 个自然日后才可提交备案。
- 腾讯云初审通常约 1–2 个工作日；省管局审核不超过 20 个工作日，实际按省份和材料为准。[腾讯云快速备案](https://cloud.tencent.com/document/product/243/39038)
- 微信小程序备案：平台初审通常 1–2 个工作日，管局 1–20 个工作日。[微信小程序备案指引](https://developers.weixin.qq.com/miniprogram/product/record/record_guidelines.html)

因此开发与本地回归可持续进行，但真实大陆域名、关闭调试模式的体验版和最终发布，必须等待外部备案状态完成。

## 事实与推断边界

- 上述域名、服务器资格、ICP、微信通讯域名、HTTPS、小程序备案和公安备案时限均为官方明文规则。
- “选择 `.cn`”“复用业务 CVM”“先腾讯云网站备案、后微信小程序备案”“PC Admin 导致推荐办理公安备案”是为 Order 当前 MVP 做的工程与流程裁决；不是法律机关对个案出具的意见。
- 微信认证、微信支付商户费率以及 CDB 最终订单价没有统一适用于当前主体的公开固定值，必须读取客户实际后台/购买控制台，本文未猜价。
