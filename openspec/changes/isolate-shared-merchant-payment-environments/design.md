# 财务隔离实现

gate_type: W3；ui_level_target: UI0；ui_level_actual: UI0。此 change 无 UI diff；实际资金/手机 UI3 和发布验收由 root 单独记录，后端本地证据不替代它们。

遵循冻结 requirements、technical-amendment-v1、implementation-contract-v1 和执行 v4。native development/test 保留既有编号与 HTTP/fake；cloud 使用固定新号与环境。支付 provider 前保留原请求+claim，最终 receipt 不前移；退款同事务 receipt 保持。五 RPC ABI 不变。S5 registry 本地接线完成；运行时调用点、真实 UAT 容量及最终依赖门禁由 root/R9 在源精确绑定后完成。

## S5 完整安装器与普通身份

`InstallSQL` 和 `Check` 显式绑定 UID、environment、paymentNotifyURL、refundNotifyURL 四值。environment 仅 uat/production；两回调采用相同 HTTPS origin，固定 `/api/v1/payments/wechat/notify` 与 `/api/v1/refunds/wechat/notify`，拒绝 query、fragment、userinfo 和非原始固定路径。安装顺序在 access.sql 后追加 payment_environment.sql，保留原四个 v1 函数实体及原 body，并撤销 PUBLIC/anon/authenticated 的有效执行权限。readiness 检查实际 113 个实体的完整定义、ABI 和有效权限；普通 client 仅允许 109 个已审核入口，旧四个入口在发出 HTTP 前拒绝。

Storage business bytes 暂从冻结源采用，已逐项对应本地候选 `2811cabfb21005330c38abb205ce7f4810974f8b`；其精确 SHA 独验尚需单独收口，本段本地实现与局部 PG PASS 不代替依赖、Finance 候选或发布门禁。普通 response 上限 8MiB、HTTP 15s、readiness 2s 未放宽；31,438,849 B 是完整请求 JSON 的既有本地实测，request 没有该 response cap，最终真实普通 UAT 输入容量仍未验证。

## S3 固定 wire（首次测试落盘前冻结）

三个位置参数 ABI 不变。`p_entries` 每项严格八字段：`ordinal`（无前导零 uint64 十进制 string）、`kind`（PAYMENT/REFUND）、`appid_hex`、`mchid_hex`、`out_trade_no_hex`、`out_refund_no_hex`、`provider_id_hex`、`payment_provider_id_hex`。字节字段为小写 UTF8 hex，无空字符；每批最多 100，顺序与 ordinal 精确对应。

响应严格 `{outcome:"FOUND",evidence_cutoff:string,entries:[{entry:<精确输入echo>,payment_proof:{state,digest_hex,environment},refund_proof:{state,digest_hex,environment},confirmations:[{audit_id,environment,source_ref_hex,evidence_sha256,supersedes}]}]}`。proof.state 为 ABSENT/VALID/CORRUPT/CONFLICT；VALID 但未证明精确当前路由时 environment 空，不能推为 Production。固定 cutoff 来自全部 `SYSTEM_EVIDENCE/SYSTEM/billing.ownership_confirmed` 最大 audit ID，而非本页命中。首批 0 取当前值；非零后批只读 <= 原值；首批冻结 0 后出现非零即拒绝整次，不落 receipt。

确认 `after_state_json` 严格 `{rule_version:"1",entry:<上述精确金融标识去掉ordinal>,environment,source_ref_hex,evidence_sha256,supersedes}`。`source_ref_hex` 非空、证据摘要 64 小写 hex、supersedes 为无前导零 uint64 string（0 表示无）；audit_id 从行 ID 取得。只读既有受审阅确认，不提供写入口。按 audit ID 排序，supersedes 必须指向同一精确身份的已存在较旧确认；未解决冲突或分叉保持 UNKNOWN。

原 request 必须按原 typed Go JSON 字段顺序、omitempty 与转义规则重算摘要，再验证持久身份/父关系。不得 hash JSONB::text，不向 host facts/receipt 输出 raw request 或 openid。证明摘要只含实际 proof 和被用确认链，不含运行时钟、随机数或对账自身 receipt。

### Root 已批准 worker locator 补充

仅已有 QUERY claim 的 REFUND worker 可令 parent/provider/payment-provider 三字段空，仍使用原 3 参数/8 字段。facts DB 只对当前未过期 QUERY lease 返回 `resolved_parent_hex`（原 r→p 父号）与 `query_claim:{version,owner_hex}`，其余 proof/digest/env 来自原持久请求。host 比对既有 claim version 和本次 owner、原父号和 proof；后续 observe/finish 仍原 fencing。普通账单/callback 四字段 fact 回包与全标识约束不变，不改 echo 或挪用 ordinal。来源：root 明确批准及 v4 S3补充。

### Root 已批准的完整留证与 PAYMENT locator 补充

完整对账 receipt 增加 `system_facts`，内部顶层为14字段。`evidence_digest` 绑定 `{bill_facts,system_facts}`；bill 按 ordinal，system 按 kind/number 排序，保存实际参与判断和投影的最小原请求 proof 与确认链，并在保存 receipt 的同一事务复核。真正消费的 proof 更正产生新 identity，旧 receipt 保留；strict `system_only` BillEntry 结构保持原样。系统退款的父付款号及 transaction ID 仅从现有 immutable refund proof 定位，随后使用完整身份执行原 proof/确认规则。

仅 PAYMENT 内部 locator 可在两个 provider 输入均空时，从正确 UID/环境/商户/AppID/对象的已有完整 MATCH 观察解析唯一 transaction ID，回包增加 `resolved_provider_id_hex`。同 ID 重复可定位，多不同 ID 返回 CONFLICT，无完整观察返回 UNAVAILABLE；调用方随后以完整身份执行原 proof/受审阅确认规则。原五 RPC、facts 三参数/八字段、echo、旧观察及物化 RPC 保持原契约。来源：root 已批准的 v4 2026-10-10 补充。
