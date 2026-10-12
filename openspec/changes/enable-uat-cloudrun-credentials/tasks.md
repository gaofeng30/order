> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# Tasks

Status: UAT_SCOPE_CONFIRMED / IMPLEMENTING; local candidate NOT_CREATED. Full cloud UAT NOT_VERIFIED. Base/ownership/dependencies are fixed in proposal.md.

- [x] 1.1 Preserve other worktrees/index; create clean isolated implementation worktree from exact 53ad8bc655dce474da2f54280865a6e7dfa6cc40 and branch codex/order-uat-cloudrun-credentials.
- [x] 1.2 Read TDD, quality gates, native credential/COS interfaces and actual UAT probe evidence; fix the smallest scope and SDK test seam.
- [x] 1.3 Strictly validate this change before implementation.
- [x] 2.1 Exchange protocol Red → Green through native provider interface.
- [x] 2.2 Refresh/concurrency and unavailable-response Red → Green; bounded wait, recovery and endpoint/redirect safety.
- [x] 2.3 Route UAT config/SSM/payment/redemption/COS to CloudBase and preserve production CVM behavior; Red → Green.
- [x] 2.4 COS atomic tuple and expired/unavailable signing Red → Green.
- [x] 2.5 Refactor only necessary slice details and rerun affected regression; record reasons if no additional refactor needed.
- [x] 3.1 Ordinary/race, vet/build, isolated smoke, strict validation, formatting, owned diff and secret review; distinguish failed/not-run/skip.
- [x] 3.2 Two-axis review and scoped TTL repair; local Writer readiness score C9/T9/V8/R10 =36.
- [ ] 3.3 Owned-only local candidate commit and full SHA recorded in external lifecycle receipt.
- [ ] 3.4 Clean detached exact-SHA independent verification and separate receipt.
- [ ] 4.1 Native PG/TLS and SSM activation/secrets available, actual new CloudRun identity/provider deployment, signed COS roundtrip and real platform acceptance. NOT_VERIFIED; dependent full UAT gate.

## Evidence

Task artifact directory: `/Users/vivix/.codex/visualizations/2026/10/06/01a11042-3601-7850-a38f-974660ec7462`. `uat-api-key-sdk-probe-timed-receipt.json` proves exchange and COS HeadBucket200 in a local SDK process against real UAT, SSM ResourceUnavailable.NotPurchased and diagnostic-key deletion. It is not container business deployment or SSM permission proof. Both failed and successful probe observations remain preserved.

## Red / Green and scope decisions

- Initial 01 compile diagnostic is retained and is not claimed as runtime Red. The same public UAT Load/payment/redemption seam was executed with one synthetic HTTP test overlay on a git archive of exact base 53ad8bc655dce474da2f54280865a6e7dfa6cc40: native UAT Load failed with production_secret_source_unavailable. Business source was untouched, no cloud endpoint was contacted, temporary archive cleaned. Receipt: uat-credential-baseline-public-seam-red.json.
- Protocol/cache Green: uat-credential-rgr-01-green.log. Behavior Red 02: 32 SDK calls retained the old tuple; Green refresh/cache shared complete tuples.
- Red 03: unsafe env/key/region, redirect, oversized/trailing JSON and unsafe token accepted; Green bounds endpoint/body and collapses invalid responses.
- Red 04: 12 failed concurrent exchanges queued for 618ms at a 50ms request budget; Green shares in-flight exchange and bounded waits, fails closed and recovers after a later success.
- Red 05: actual public UAT Load rejected old CVM identity; Green native Go SSM signatures and all four purpose names were observed through synthetic HTTP responses. Production official CVM selection and absent-key UAT failure regression passed.
- Red 06: native COS request signed separate components; Green atomic tuple signs request/GET/PUT URLs, and unavailable tuple prevents network and URL issuance.
- Independent writer reviews identified short-lived credential P2. Red 07 first proves cached 15-minute tuple reuse, then separately proves a fresh 10-minute tuple accepted. Green requires a complete tuple with more than the maximum15-minute COS URL window, for both cache and new exchange. uat-credential-rgr-07-green-race.log PASS. Earlier full ordinary1458PASS/4helperSKIP and race were before this repair and remain historical evidence. Final gates must bind the repaired hashes.
- Necessary refactor: common environment selection joins existing SSM/material/COS compositions, and COS snapshots adapt the actual native SDK signing seam. No additional refactor, dependencies, business/SQL/UI changes or new fallback needed.
- Local tests use public SDK credential interfaces, native SSM/COS signing and isolated synthetic HTTP/clock inputs. They do not prove CloudRun deployment, SSM activation/permissions, real COS object roundtrip, UI3, real financial flows or full95 acceptance.

## Writer checkpoint before candidate commit

Final repaired source: full ordinary and race each1459PASS/4explicit opt-in UI helperSKIP/0FAIL, native isolated PG16.13; all clusters stopped, disposable database/password removed, source hashes unchanged. Final vet/build and isolated HTTP smoke PASS; strict/format/diff/owned source and credential-value review PASS. Both independent pre-commit review axes report zero remaining findings on repaired12Go+4artifacts; no actionable baseline smell. Earlier full1458 runs bind older code and are retained separately. No additional adjacent changes or broad retesting is required after these final gates without a new change/failure. UI static is N/A for this owned Go-only delta.

This tracked checkpoint deliberately precedes the candidate SHA. Candidate creation and detached verification update the external uat-credential-candidate.json/independent receipts without editing the frozen candidate. A local candidate or verifier result does not complete cloud UAT, integrate dependencies/main, or authorize remote publication/financial transactions.
