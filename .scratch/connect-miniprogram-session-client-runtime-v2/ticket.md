# connect-miniprogram-session-client-runtime-v2

- Base: `2e78d1afca0a334739fb377f1f5772edb461f4fd`
- Dependency: MP-00 runtime endpoint WIP at the same SHA.
- Public seam: cold-start `wx.login` -> `POST /api/v1/auth/miniprogram/session` -> runtime-memory Bearer session.
- Owned paths: this ticket/gate, `apps/wechat-miniprogram/app.js`, `utils/sessionApi.js`, `tests/session-ui0.test.js`, and the minimal login seam in `tests/page-harness.js`.
- Read-only: `runtimeEndpoint*.js`, package/runner, pages, backend, confirm/result/order paths.
- Fail closed: non-ready endpoint means zero `wx.login`, zero request, and no credential; rejected/malformed responses never retry automatically.
- UI target: local UI1; UI2 remains `BLOCKED_EXTERNAL` while Developer Tools login/permission is unavailable.
- Candidate rule: stacked local WIP only while MP-00/UI2 is blocked.

## Red -> Green -> Refactor

- [x] Red: exact runtime base produced 12/12 failures; first failure was missing session state/startup exchange.
- [x] Green: focused session suite is 12/12; runtime endpoint drives exactly one exchange and strict in-memory state.
- [x] Refactor: runtime+session 57/57, all Node 174/174, JS syntax PASS, locked Chromium 151 UI1 3/3.

## Boundary

- Runtime endpoint WIP remains the stack base and UI2 is externally blocked, so this change is local-ready WIP, not Candidate or independently verified.
- Trial/release remain intentionally unconfigured; their cold launch performs zero login and zero request.
- No package, UI runner, page, backend, payment, order, push, deploy, preview, upload, or external system was changed.
