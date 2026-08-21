## Why

用户确认员工白名单的导入模板只要姓名与手机号两个字段。原 §6.13.3 的模板有七列（手机号、姓名、单位、部门、工号、备注、状态），其中五列来自 §6.4 继承自已删除会员名单的字段表。

按用户此前定下的「Excel 字段与前端页面一一对应」约束，模板收敛必须连带页面收敛——否则会出现「页面有字段但模板填不了」的不对应状态。

## What Changes

- §6.4：员工折扣白名单的**可填字段收敛为手机号与姓名两项**，删除单位、部门、工号、备注。明确另有三类非填写字段：状态（页面开关切换，新增默认启用）、加入时间（自动）、已绑定与累计消费单量（只读）。
- §6.13.3：模板收敛为**姓名、手机号两列**，并明写不得包含其他任何列；表头出现未知列时忽略并在预览提示，不因此判定整份文件异常。
- §6.13.3 补一条覆盖规则：按手机号覆盖更新时 MUST 保留状态，**导入不得把已停用的记录重新启用**。
- §15.6.5 `StaffWhitelist` 数据模型删除 `org` / `dept` / `jobNo` / `remark`。
- §16.5 的白名单建页要求同步为两个可填字段加状态开关。

## Capabilities

### Modified Capabilities

无。本 change 只修改产品文档。

## Impact

- Owner：branch `worktree-staff-template`，worktree `/Users/marcusz/Projects/Test Tool/order/.claude/worktrees/staff-template`。
- Owned paths：`docs/product/online-ordering-system-prd-0818.md`、`openspec/changes/simplify-staff-import-template/**`。
- Dependency：`specify-bulk-import`，已集成并归档于 `base_sha`。
- Non-goals：不实现任何代码；不修改生效 spec；不改动菜品导入模板。
- Gate：`gate_type=W0`；`ui_level_target=UI0`；`ui_level_actual=UI0`。
- 最小成功标准：§6.4 只声明两个可填字段且不再列出被删的四项；§6.13.3 模板恰为姓名与手机号两列并明写不得有其他列；覆盖更新保留状态；`StaffWhitelist` 模型同步。
- **范围边界**：§6.13 全节仍为开发方提出的范围新增，待客户确认（§16.4 C4）。本 change 只是收窄该节，不改变其待确认状态。
