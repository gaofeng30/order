> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Design

UAT has one credential path: the environment API Key exchanges temporary Tencent credentials. Production retains its CVM instance-role path. Do not fall back from UAT exchange to metadata, permanent Tencent credentials or raw business secrets.

The endpoint is derived only from validated `CLOUDBASE_ENV_ID` and `ORDER_TENCENT_REGION`: `https://<env>.<region>.tcb-api.tencentcloudapi.com/capi/credential`. POST JSON contains only the environment ID; the API Key is a Bearer header. Redirects are rejected. Bound request duration and response size; validate HTTP200, numeric code0, all three credential components and expiry covering the maximum fifteen-minute COS signed URL lifetime. Collapse failures to stable errors without logging provider bodies, headers, keys or signed URLs.

Expose the existing native `common.Provider`/`common.CredentialIface` boundary from config. Refresh at least fifteen minutes before expiry, matching the existing COS adapter maximum signed URL lifetime. Reject even fresh tuples whose remaining lifetime cannot cover that window. A refresh is shared by concurrent callers, with bounded waits and no expired/stale-on-error fallback. Time and HTTP transport are controlled at the isolated constructor seam in tests; behavior assertions use the public SDK interfaces and HTTP protocol, not private cache fields.

SSM uses the selected credential through the current four-purpose secret boundary. The native COS SDK currently reads ID/key/token separately. Adapt its transport's existing credential getter and request-signing boundary to take one atomic tuple and reject an incomplete tuple before generating URLs or sending requests. Use the SDK's official AuthorizationTransport with a per-request snapshot.

No business source, SQL, PC/mini source, dependency version or production identity is changed. Current UAT environment is order-uat-d1gtzpzj50615c300 in ap-shanghai. Native PG/TLS, SSM activation and authentic business assets remain cloud deployment prerequisites, not implied by credential exchange or local tests.

Primary protocol evidence: official `@cloudbase/cloudbase-mcp` v2.34.8 `exchangeApiKeyForCredential`, [CloudBase connection modes](https://docs.cloudbase.net/ai/cloudbase-ai-toolkit/connection-modes), and the real UAT SDK probe receipts in the task artifact directory. Diagnostic keys have been revoked. No credential values are part of evidence or source.
