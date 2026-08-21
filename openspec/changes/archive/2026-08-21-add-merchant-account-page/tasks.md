## 1. 规格与文档

- [x] 1.1 PRD §4.4 补账号字段、两种角色的可用范围、最后一个主账号不可失效、停用与删除的区别
- [x] 1.2 PRD §15.5.3 商户账号名单行补姓名字段与最后主账号保护
- [x] 1.3 PRD §16.5 该行由「PC 侧未建」转为已建，改记待接后端的两处
- [x] 1.4 `web-admin-scope-conformance` 三条 ADDED requirement

## 2. 门禁（Red）

- [x] 2.1 写 `checks/check_merchant_accounts.js` 九项
- [x] 2.2 在 base_sha `adfb005` 树上确认九项全红、`exit=1`

## 3. 实现（Green）

- [x] 3.1 `data/seed.js` 加 `MERCHANT_ACCOUNTS`，覆盖五种形态
- [x] 3.2 `data/api.js` 加 `ROLES` / `ROLE_LABEL` 与四个契约方法
- [x] 3.3 `guardLastOwner` 守住删除、停用、降级三条路径
- [x] 3.4 `pages/accounts.js` 七列表格、搜索、行内启停、编辑抽屉、删除二次确认、单主账号提示条
- [x] 3.5 `app.js` 「名单」组加入口；`index.html` 挂载
- [x] 3.6 门禁在候选树 `MERCHANT_ACCOUNTS_GATE=PASS`

## 4. UI1 证据

- [x] 4.1 `python3 -m http.server 8091` 起站，页面渲染五条种子账号、七列齐全
- [x] 4.2 契约层：压到只剩一个启用主账号后，删 / 停 / 降级三次调用全被拒，账号 `role`/`enabled` 不变
- [x] 4.3 契约层：启用第二个主账号后，停用第一个成功
- [x] 4.4 契约层：编辑姓名后 `boundOpenId` 保持
- [x] 4.5 契约层：重复手机号被拒并指出占用者
- [x] 4.6 UI 层：点行内「停用」唯一主账号 → toast 拒绝，行状态不变
- [x] 4.7 UI 层：点「删除」并确认唯一主账号 → toast 拒绝，记录仍在
- [x] 4.8 UI 层：编辑抽屉把唯一主账号改为子账号并保存 → toast 拒绝，角色仍为 owner
- [x] 4.9 只剩一个启用主账号时提示条出现且不重复；恢复第二个主账号后消失
- [x] 4.10 页面重载后控制台无 error / warn

## 5. 回归

- [x] 5.1 小程序 `npm test` 65/65
- [x] 5.2 历史归档门禁全跑；两处 FAIL（`check_residue.py`、`check_bulk_import_prd.py`）在 base_sha 树上表现完全相同，确认为预存在

## 6. 交付

- [x] 6.1 提交
- [x] 6.2 clean worktree 独立验证
- [x] 6.3 合入 main、应用 delta、归档、推送
