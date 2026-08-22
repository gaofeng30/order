# establish-miniprogram-ui-gates evidence

## Checklist

- [x] Confirm exact base SHA and read applicable rules, quality contract, PRD sections 10/14, and required Skills.
- [x] Declare scope, W/UI Gate, external assets, non-goals, acceptance commands, and public TDD seams.
- [x] Red: cold-start anonymous browse fails because the UI1 runner does not exist.
- [x] Green: cold-start anonymous browse runs in the real browser/simulator.
- [x] Red/Green: network error is visible and user retry recovers to rendered catalog.
- [x] Red/Green: category and product-selection interactions run from rendered UI.
- [x] Refactor: rerun the same UI1 scenarios and affected regression/static checks.
- [x] UI2: emit a sanitized `BLOCKED_EXTERNAL` receipt after the developer-tools permission probe failed before page compilation.
- [ ] Two-axis Standards/Spec review.
- [ ] Commit candidate and rerun declared gates in a clean detached worktree at the exact SHA.

## Evidence

Evidence entries are appended only after the command/action actually runs.

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: locked Chromium ran cold anonymous browse, recoverable network failure, category switch, and rendered product-selection sheet; 3/3 passed
artifact_or_environment: order-miniprogram-ui-gates@1.0.0, miniprogram-simulate@1.6.2, Chromium 151.0.7922.34
unverified_boundary: simulator evidence does not prove WeChat Developer Tools, experience build, device, real account, payment, or production behavior
external_asset:
  owner: N/A
  missing: N/A
  recovery: npm ci then npm run browser:install and npm run ui1 from tools/miniprogram-ui
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: refactor
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: warning-free runner rerun passed the identical 3/3 browser/simulator scenarios
artifact_or_environment: order-miniprogram-ui-gates@1.0.0, miniprogram-simulate@1.6.2, Chromium 151.0.7922.34
unverified_boundary: WeChat Developer Tools and higher platform behavior remain outside UI1
external_asset:
  owner: N/A
  missing: N/A
  recovery: npm ci, install the locked browser, and rerun ui1
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: writer
command_or_action: npm --prefix tools/miniprogram-ui run ui2
exit_result: BLOCKED_EXTERNAL
sanitized_summary: developer-tools permission probe confirmed the logged-in account is not a developer for the configured project; no UI2 page compiled or ran
artifact_or_environment: sanitized receipt at tools/miniprogram-ui/receipts/ui2-latest.json, excluded from Git
unverified_boundary: all UI2 scenario behavior
external_asset:
  owner: mini-program AppID administrator
  missing: developer permission for the currently logged-in WeChat Developer Tools account
  recovery: grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui2
exit_result: exit-1
sanitized_summary: the UI2 command was absent, so no WeChat Developer Tools run or receipt occurred
artifact_or_environment: local developer tools 36.6.0 with CLI login already confirmed
unverified_boundary: all UI2 behavior
external_asset:
  owner: quality-infrastructure writer
  missing: UI2 automator and receipt runner
  recovery: add the isolated UI2 runner and rerun the identical command without preview or upload
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: NOT_RUN
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-254
sanitized_summary: package.json was absent, so the requested UI1 runner could not start
artifact_or_environment: local writer worktree at exact base
unverified_boundary: no browser, simulator, page, or scenario ran
external_asset:
  owner: quality-infrastructure writer
  missing: repository UI1 runner
  recovery: add the locked isolated runner and rerun the identical command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: real loopback 503 rendered the error; user retry crossed the same transport and a 200 response rendered the recovered catalog, 2/2 scenarios passed
artifact_or_environment: locked Chromium 151.0.7922.34, actual menu WXML/page definition, loopback fixture
unverified_boundary: menu selection interaction and WeChat runtime are not yet covered
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the identical command after a clean install
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: external-post-commit
phase: writer
command_or_action: UI1 3/3; existing mini-program 65/65; OpenSpec all strict 20/20; JS and JSON static; Go format/test/race/vet/build/smoke; owned-path and sensitive-value audits
exit_result: PASS
sanitized_summary: all applicable writer gates passed; change-specific OpenSpec validation is N/A because the user-owned path contract forbids openspec edits and explicitly requires a scratch ticket; C9/T10/V8/R9=36 with hard blockers 0
artifact_or_environment: final owned candidate tree before immutable commit
unverified_boundary: V8 requires clean detached exact-SHA verification; UI2 is BLOCKED_EXTERNAL on missing project developer permission; no preview, upload, integration, push, deploy, real order/payment, production, or UI3 proof
external_asset:
  owner: mini-program AppID administrator for optional UI2
  missing: developer permission for the currently logged-in WeChat Developer Tools account
  recovery: grant that account developer access, confirm cli islogin, then rerun npm --prefix tools/miniprogram-ui run ui2
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: red
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-1
sanitized_summary: Chrome rendered the catalog failure and accepted the retry touch, but recovery stayed in error because no loopback transport existed
artifact_or_environment: locked Chromium 151.0.7922.34 and actual menu WXML/page definition
unverified_boundary: no successful catalog response or ready-state menu rendered
external_asset:
  owner: quality-infrastructure writer
  missing: loopback catalog fixture and browser wx.request adapter
  recovery: add the isolated loopback transport and rerun the identical command
```

```yaml
change: establish-miniprogram-ui-gates
gate_type: W1
ui_level_target: UI1
ui_level_actual: UI1-partial
base_sha: 2c549f2c413a06ec160b4e886fb16cd62e18176f
candidate_sha: not-yet-created
phase: green
command_or_action: npm --prefix tools/miniprogram-ui run ui1
exit_result: exit-0
sanitized_summary: locked Chromium 151.0.7922.34 rendered the actual home page; anonymous cold start and touch navigation passed 1/1
artifact_or_environment: order-miniprogram-ui-gates@1.0.0 with miniprogram-simulate@1.6.2
unverified_boundary: network recovery and menu interaction are not implemented yet; no WeChat runtime ran
external_asset:
  owner: N/A
  missing: N/A
  recovery: rerun the identical command after a clean install
```
