> 已停用：按用户本轮停测与删除要求，关联的测试、合成数据及 runner 已从本分支删除。以下仅保留历史记录，不是当前执行步骤或本轮通过证据。

# Admin settings UI verification

`node tools/uiux-clarity/server.cjs` serves the actual Admin sources from this checkout on a random **127.0.0.1-only** port. No requests are forwarded, no credentials are needed, and all settings/accounts are synthetic. Restarting resets the fixture. Open the printed settings URL and `/fixture` in two fresh browser tabs; leave unrelated UAT tabs untouched.

1. Edit the notice. Expect persistent `有未保存修改` and enabled Save. Save in normal mode. Expect `设置已保存`, no dirty flag, Save disabled.
2. Choose `readback-failure`, edit notice AND business status, save. PUT succeeds, following GET fails. Expect `保存结果未确认`, disabled edits/Save, visible `核对已保存设置`. Note write count. Click it: only GET increases, normalized status matches, success appears. No extra PUT.
3. Choose `reject`, edit notice, save, then check. Expect preserved input and mismatch text; Save and reload become available. Reload restores saved synthetic notice. Never claim a failed network request proves no write occurred.
4. Choose `slow`, edit notice, save; during its 2.5-second delay open account popover and click another business status. Expect a wait hint, no second PUT. Navigate dashboard then settings before completion: no editable stale form; after settlement it reads the new notice.
5. From dashboard in `slow`, initiate toolbar status write then immediately navigate settings. Expect loading until write settles, then current canonical status. Existing toolbar writes remain whole-settings API operations.
6. With a dirty settings form, click toolbar status: expect instruction to handle unsaved form and unchanged write count. No automatic write.

Run these on base for Red, current writer for Green, and a clean exact-SHA detached checkout for independent verification. Browser evidence must distinguish local fixture UI1 from real UAT. Stop only the fixture server this task started; close only its own tabs.

Current change has no Go/API/database implementation modifications. Static checks: `node --check apps/web-admin/app.js`, `node --check apps/web-admin/pages/settings.js`, `node --check tools/uiux-clarity/server.cjs`, `openspec validate simplify-admin-uiux-uat --strict`, `git diff --check`.
