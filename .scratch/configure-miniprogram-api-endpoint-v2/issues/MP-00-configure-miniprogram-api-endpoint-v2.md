# MP-00 v2：配置小程序运行时 API endpoint

- 状态：`DRAFT / LOCAL_WIP / UI1_PASS / UI2_BLOCKED_EXTERNAL`
- authoritative base：`01cc4896c3c6c8531b1a2f08778deeccab3dd43b`
- branch：`codex/configure-miniprogram-api-endpoint-v2`
- worktree：`/Users/vivix/.codex/worktrees/569e/order`
- owner：当前 runtime writer；`app.js` 在本 change 内由本 owner 独占
- Gate：`gate_type=W2`，`ui_level_target=UI2`，`ui_level_actual=UI1`
- Candidate：`not-yet-created`；本任务只冻结 local-ready WIP，由主控机械 stage/commit，禁止标记 Candidate 或 Independent Verified
- 禁止：push、merge、deploy、preview、upload、微信后台写入、外部资料申请、Docker

## 新基线与旧证据边界

- authoritative base 的 `app.js` 无条件使用 `http://127.0.0.1:8080`；运行时 endpoint provider/consumer seam 不存在。
- 主控只把旧 `7839cc9e9b824b417c704f0381ab1231f0e2dfc9` 的小程序源码 patch 以未提交形式应用到本 branch；未迁移旧 `.scratch`。
- 旧 SHA、Red/Green/Refactor、Gate、Review、计数和 verdict 全部失效。旧实现只作为实现参考，本文件只记录 01cc replacement tree 的新证据。

## 已确认 public seams

1. `resolveRuntimeEndpoint(wxApi, deploymentConfig)`：读取 `wx.getAccountInfoSync().miniProgram.envVersion`，公开 ready/error 状态。
2. `isRuntimeOrigin(envVersion, origin)`：provider 与全部 consumer 共用的环境绑定 origin 判定。
3. `App.onLaunch` / `globalData.runtimeEndpoint`：冷启动公开 endpoint 状态；非 ready 时 `apiBaseUrl=''`。
4. `catalogApi`：只有 `state=ready`、`runtimeEndpoint.origin===apiBaseUrl` 且 `isRuntimeOrigin` 为真时才允许 `wx.request`。
5. 副作用 seam：endpoint 非 ready、unknown 或非法时，冷启动与 catalog 初次/重试的 `wx.login` 和 `wx.request` 调用数都必须为零。

## 业务契约

- 测试环境缺少 `wx.getAccountInfoSync` 时只按明确 develop seam 处理；API 存在但返回 unknown/空值或抛错时 fail-closed，不回退 develop。
- develop 配置只允许规范化的本机 HTTP loopback `127.0.0.1` origin，默认 `http://127.0.0.1:8080`；不得把远端 HTTP/HTTPS origin 当成本地 develop。
- trial/release 配置当前为空；只接受合法 HTTPS domain origin，无 path/query/fragment/userinfo，规范化不保留单一尾斜杠；拒绝 IP、全部编码 loopback、精确 `localhost` 与 `.localhost` 后缀。
- endpoint 未配置、环境 unknown 或配置非法时，公开稳定 `runtimeEndpoint.errorCode`，清空 `apiBaseUrl`，不调用 login/request。
- catalog 只复用统一 runtime seam；错误时保持页面现有可重试 error、空 groups、无 mock fallback、无外发。
- endpoint 不是 secret。真实 trial/release origin 仅在域名、HTTPS 与微信 request 合法域名齐备后通过单独配置提交填写，不发明 placeholder。

## Owned paths

- `.scratch/configure-miniprogram-api-endpoint-v2/**`
- `apps/wechat-miniprogram/app.js`
- `apps/wechat-miniprogram/utils/runtimeEndpoint.js`
- `apps/wechat-miniprogram/utils/runtimeEndpointConfig.js`
- `apps/wechat-miniprogram/utils/catalogApi.js`
- `apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js`
- `apps/wechat-miniprogram/README.md`（仅 endpoint 小节）

## Protected / 非目标

- 不修改 session、phone、reservation menu、全部 pages、router/main/migrations、package files、page-harness、`catalog-ui1.test.js`、backend/deploy/product docs、历史 `openspec/**`。
- 不实现身份、手机号、菜单业务、订单、支付、商户能力、云部署、域名购买/备案或微信后台配置。

## Tracer bullets 与 Gate

- [x] R1/G1：develop 只允许本地 loopback；远端 HTTP/HTTPS 配置必须 `RUNTIME_ENDPOINT_INVALID`，consumer 不外发。
- [x] R2/G2：trial/release、unknown/非法环境与所有非法 origin fail-closed。
- [x] R3/G3：endpoint 非 ready 时 App/catalog 初次与重试均 `wx.login=0`、`wx.request=0`，页面保持可重试 error 且不回退 mock。
- [x] Refactor/UI0：focused、catalog consumer、默认/全量 Node、JS/JSON/WXML static 全部重跑。
- [x] UI1：主控在允许 loopback 的同一冻结 runtime tree 上以锁定 Chromium + simulator 真实运行，3/3 场景 PASS，`UI1_RESULT {status:PASS,scenarios:3}`、exit 0。
- [x] UI2 权限预检：CLI 已安装但 login inactive，记录 `BLOCKED_EXTERNAL`；未进入 AppID developer probe，不运行 preview/upload/deploy，不冒充 UI2 PASS。
- [x] Local freeze：8 个 status path 仅属于 owned paths；protected、whitespace、敏感模式与 manifest 检查 PASS；tree-content hash/status 在最终 handoff 返回给主控机械 stage/commit。
- [ ] Candidate/Verifier/Integration：本任务不启动；stage/commit 与 external attestation 由主控后续执行。

## 外部恢复

- UI2 owner：小程序 AppID 管理员与当前 Developer Tools 操作人。
- 若当前账号不是该 AppID developer：管理员授予该登录账号 developer 权限，操作人重新确认 `cli islogin`，在主控提交后的同一 exact SHA 上重跑 UI2；不得改用 preview/upload 代替。
- 真实 trial/release owner：客户小程序管理员、开发方与 UAT owner。域名、DNS/备案、有效 HTTPS、request 合法域名和测试入口齐备后，单独配置真实 origin，对新 exact SHA 重跑 UI0/UI1/UI2，再按独立流程做外部 attestation。

## Evidence

本节仅记录本 replacement tree 当前实际命令。未提交 tree 不能形成 Candidate 或 independent receipt。

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI0
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: red
command_or_action: node --test apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js
exit_result: FAIL
sanitized_summary: 35 pass / 5 fail；develop 远端 HTTP/HTTPS 被错误标为 ready，伪造 develop remote endpoint 实际产生 1 次 wx.request
artifact_or_environment: apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js / Node harness
unverified_boundary: 零 login 断言在同一 Red 运行中已通过，但不证明 UI1/UI2 或真实微信能力
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI0
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: green
command_or_action: node --test apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js
exit_result: PASS
sanitized_summary: 45/45；develop 本地 loopback、trial/release HTTPS、unknown/非法 fail-closed、App/catalog 零 login/零 request、无 mock fallback 全部通过
artifact_or_environment: apps/wechat-miniprogram/tests/runtime-endpoint-ui0.test.js / Node harness
unverified_boundary: Node harness 仅计 UI0；不证明 Chromium UI1、Developer Tools UI2、真机或真实域名
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI0
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: refactor
command_or_action: focused 45/45; catalog 14/14; npm default 65/65; all Node 162/162; apps JS check; JSON parse 28; lint_wx.py
exit_result: PASS
sanitized_summary: provider、catalog consumer、零 login/request 失败路径与全部适用 UI0/static 回归通过
artifact_or_environment: local Node/static harness on uncommitted replacement tree
unverified_boundary: 所有 Node/harness 只计 UI0，不证明 UI1/UI2/真机或 committed exact SHA
external_asset:
  owner: N/A
  missing: N/A
  recovery: N/A
```

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI1
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: writer
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: PASS
sanitized_summary: Chrome for Testing 151.0.7922.34；TOTAL 3 SUCCESS；UI1_RESULT status=PASS scenarios=3；exit 0
artifact_or_environment: 主控允许 loopback 的执行环境 / 同一冻结 runtime tree / order-miniprogram-ui-gates@1.0.0 / locked Chromium
unverified_boundary: UI1 不证明 Developer Tools UI2、体验版、真机、真实 trial/release 域名或 committed exact SHA external attestation
external_asset:
  owner: N/A
  missing: N/A
  recovery: 主控机械提交后对 exact SHA 重跑 post-commit external attestation
```

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI1
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: writer
command_or_action: WeChat Developer Tools CLI existence and islogin preflight
exit_result: BLOCKED_EXTERNAL
sanitized_summary: Developer Tools CLI installed；cli islogin 未处于 active login，未继续 AppID developer permission probe
artifact_or_environment: local WeChat Developer Tools CLI permission preflight only
unverified_boundary: 不证明 AppID developer permission 或 UI2；未执行 preview/upload/deploy
external_asset:
  owner: Developer Tools 操作人；若登录后非 developer，则为小程序 AppID 管理员
  missing: active CLI login；其后仍需确认当前账号 AppID developer 权限
  recovery: 操作人登录 Developer Tools 并确认 cli islogin=true；若权限 probe 显示非 developer，由管理员授权后在同一 committed exact SHA 重跑 UI2
```

```yaml
change: configure-miniprogram-api-endpoint-v2
gate_type: W2
ui_level_target: UI2
ui_level_actual: UI1
base_sha: 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
candidate_sha: not-yet-created
phase: writer
command_or_action: git diff --check; status owned allowlist; protected-path diff; trailing-whitespace and sensitive-pattern scans; manifest JSON parse; final local Gate rerun
exit_result: BLOCKED_EXTERNAL
sanitized_summary: local hygiene PASS、8 个 tree path 全部 owned 且 UI1 3/3 PASS；但 UI2 CLI login inactive，Writer Gate 仍不得 PASS
artifact_or_environment: uncommitted replacement tree based on 01cc4896c3c6c8531b1a2f08778deeccab3dd43b
unverified_boundary: 未 stage/commit、不是 Candidate、不是 Independent Verified；UI2 未通过，UI1 尚未绑定 committed exact SHA external attestation
external_asset:
  owner: Developer Tools 操作人及 AppID 管理员
  missing: active CLI login 与后续 AppID developer permission
  recovery: 主控机械提交同一 tree content 后，在 exact SHA 上做 post-commit UI1 attestation，并在登录/授权后重跑 UI2 与全部 Writer Gate
```
