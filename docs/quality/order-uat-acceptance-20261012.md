# Order UAT 全局验收

2026-10-12，北京时间。结论：**完整 UAT 未通过，生产未发布，2000–3000 人使用规模尚未验收。** 核心交易、三端页面与若干异常边界已有真实证据，下方按来源和范围分别记录。Owner 主清单为[当前验收清单](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/current-acceptance.md)。

## 1. 版本与已完成结果

| 层次 | 当前版本 / 实证 | 边界 |
|---|---|---|
| 小程序 | b3f1f6da24935a9a46e4615b87374a56027b70af；136 源文件匹配；开发版 uat-20261012-b3f1f6d 上传接受 | 正式体验版选择 NOT_RUN；不是正式发布 |
| API | a833507a809decd718729501d2bab96cef2d0f35，CloudRun 004，normal、100% 流量；msgid 兼容已部署 | 新通知 SENT、设备到达、点击仍分项待验 |
| PC | 初始包370991、finance7fe、api700、Admin87对应切片已实际发布 | 不虚构统一 PC 发布 SHA |
| 三端UI | b3确认金额、修改菜单返回后保留输入、商户列表/退款详情实际通过；Admin公告保存/离页读回/恢复通过 | 无制作中真实订单，“查看并备好”仅有隔离 exact UI2 |
| 交易与履约 | 131/133顾客退款、129 OWNER退款；132价格/两级备注冻结、自然worker、备好、真实相机核销；员工80%报价及恢复；原生支付取消主路径 | 有效旧证据按源码和条件不变复用，不为定位通知重复支付 |
| 财务 | N3前四笔CSV收43分、PC退12分/净31分；133之后当天收1分/退1分/净0；账单967系统核平已读回 | 四行CSV是历史快照；不宣称退款CSV或单次微信账单下载已取得 |
| 配置恢复 | H14关闭、A恢复10分、F当日可售、员工空名单100%；PC退出/保护页仅登录 | 本轮没有新资金动作或修改生产 |

专项发布、真实资金及UI收据见[原交付汇总](order-production-delivery-handoff.md)；交接截止文档提交 bd29721a2640ea9b56d853304651f768c713a691。该提交不是统一运行版本。

## 2. 完整验收差距

| 项目 / 状态 | 下一动作与负责人 | 最小依赖 |
|---|---|---|
| 体验版 / BLOCKED_EXTERNAL | 本人在版本管理选择并读回 b3 开发版为体验版，核对实际体验成员 | 微信后台曾明确拒绝自动访问，不换浏览器或API绕过 |
| READY深链 / RED | 本人点击原132通知，执行者区分微信版本/页面不存在与应用订单404；先核对trial版本、成员、endpoint及历史发送参数 | 后端trial、源码page/id正确仍不足以定根因；禁止放宽订单鉴权 |
| 退款通知 / PARTIAL | 新自然业务通知分别核对provider接受、本地SENT、设备到达、正确点击 | 旧13/14/15终态保留，不重放；不为诊断创建新资金交易 |
| 独立角色及手机号 / BLOCKED_EXTERNAL | 真实STAFF只能履约、不能进入OWNER配置/财务；未绑手机号参与者正常授权并核对隐私流程 | 真实第二参与者及合法授权；不能把同一OWNER当独立STAFF |
| 重复扫码UI / NOT_RUN | 复用132已完成凭证，真机再次扫码应明确已核销且不改业务结果 | 本人相机操作；本地并发SQL通过不能替代页面 |
| 最后OWNER重登 / BLOCKED_EXTERNAL | 用户准备好后由现有执行者生成短时PC码并正常原生批准 | 沿用已发出的准备问题，不重复催问、不伪造会话 |
| 宽覆盖异常边界 / PARTIAL_LOCAL_PASS | 29个优先边界已通过本地应用/真实SQL，继续按产品映射保留未覆盖片段 | 18现场场景不等于95条契约；不机械重跑全仓已删测试 |

产品经理已保留[95条契约映射](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/product/acceptance-mapping.md)及全部唯一ID。重点补证方向包括回调丢失/乱序、已付无单恢复、worker中断、退款未知、跨日同号、手机号姓名错配/越权/会话期限、报价到预支付漂移/截单/恰30分钟、图片/开屏、导入失败幂等和财务来源。此列表表示需要证据，未认定实现有缺陷；无数量库存仍是既有契约，不扩验旧技术文档里的库存预占。

2026-10-12用户明确要求完整测试，本轮按影响面执行必要局部验证；原话、会话来源及继承AGENTS允许后续指示的条款见[本change授权范围](../../openspec/changes/reconcile-order-owner-delivery/proposal.md#本轮授权与历史停测的适用范围)。此前停止套件与删除485文件是历史操作记录，不再用来拒绝当前必要验收，也不恢复无关旧套件。

## 3. 容量与可靠性实测

既有业务文档记录高峰5–10分钟100–300单、日500–1000单；技术门槛200并发、300单/5分钟、1000日订单/300商品、读p95≤500ms、写p95≤800ms。它们是目标，不是达成值。2000–3000位使用者的新峰值尚待用户回答；先测现有门槛不等于默认3000并发。

本轮已执行的隔离层：

- 本地 PostgreSQL 16、实际44份关系迁移、300商品/1000合成订单；200并发菜单SQL在20秒完成57025事务、0 SQL失败，p95 339.105ms。
- 订单推进10000次竞争后，1000订单各推进一次、1000审计，无重复目标；注入审计错误后整笔事务回滚。
- 同key1000次并发核销只产生1回执、订单版本只增加1；换key再次核销返回P1004，没有重复改写。
- 1000重复支付观察只产生1观察；该切片未覆盖支付物化，不当完整支付结论。1000重复退款成功观察只形成1观察/1审计，退款金额与订单结果一致。
- 本地pg_dump恢复至独立新库后数量/金额一致；只证明本地PG16路径，不能证明线上Shared PG可恢复或达到RPO/RTO。

准确命令、输入、复跑及本地集群清理见[容量收据](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/capacity/receipt.json)。合成数据、无真实provider；本机SQL绕过CloudRun/HTTPS网关，不推算线上吞吐。

真实UAT只执行两个匿名接口的小样本：20次串行GET、总速率≤1次/秒，全200；菜单10次p95 981.37ms，门店设置10次p95 444.69ms。另一个初始门店请求1594.01ms。菜单样本高于读目标，不能认定性能通过；该样本也不能定位SQL、网络或网关根因。见[原始采样](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/uat-anonymous-read-sample.json)及[初始请求](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/uat-anonymous-read-smoke.json)。

产品边界另完成29个末级场景（35测试节点），无失败/跳过：报价至预支付变价/停售/停分类/休息/身份漂移、精确截单、距取餐30分钟边界、手机号姓名错配、PC角色/会话过期、导入失败与重放。使用bd29721精确归档、原Go应用与生成SQL、隔离PG16；342个原API文件逐字保持，无云/微信外呼。临时夹具错误经修正后同源通过，不登记为业务缺陷。见[产品边界报告](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/product/focused/report.md)。这些不代替真实HTTP/UI/原生授权或95条全PASS。

通知解析另已补本地真实provider边界矩阵：a833候选31/31通过，父提交21通过/10失败，差异均为msgid处理；没有联网发送，不能替代SENT/设备结果。见[通知局部验证](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/notification/report.md)。

后续需同形态隔离HTTP/RPC负载、worker重启/发布并存、资源及锁等待指标、超时和幂等核对。禁止对真实UAT订单制造退款、通知或扣款副作用来做压测。

## 4. 生产前结果与责任

只读确认独立Prod标准版与Shared PG已存在；没有生产业务服务、域名和备份恢复完成证明，不因材料缺失宣称资源不存在。Prod现存自动续费/超额使用为true，UAT为false，本次均未修改。见[资源读回](/Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/platform-live-readback.json)。

依据晚于旧CVM/MySQL规范的人类托管要求，唯一后续生产方向采用现有CloudBase（生产适配change仍需先同步现行架构spec，本W0不覆盖规范）：CloudRun＋Shared PG＋私有COS＋同源网关/静态托管。当前源码SharedPG入口仍限UAT；生产适配、恢复、正式域名、release配置及平台资格未验收。完整步骤及停止条件见[生产发布Runbook](../../deploy/runbooks/order-production-release.md)。

客户保留主体、管理员、经营及收款责任；开发方完成技术实现和运维支持。人工动作聚焦体验版/资格后台读回、原生授权、真实独立参与者、现有通知点击和重复扫码；不索取密钥或重复已有门店资料。
