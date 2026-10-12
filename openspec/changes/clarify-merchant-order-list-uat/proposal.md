# 商户订单列表阅读状态
用户2026-10-12授权三端UIUX实现与UAT验证，当前直接AGENTS要求TDD/独验，优先于旧流水停测覆盖，仅本新流水适用。单一能力：商户准确识别列表加载/失败/空结果与订单取餐时间，不改变履约操作。
- base_sha: 01312131fb16c7121fc72b2d02532af4bd4ab65d
- owner: 本会话；branch: codex/order-merchant-reading-uiux；独立worktree: /Users/vivix/.codex/worktrees/order-order-reading-uiux/order
- gate_type: W2；ui_level_target: UI2；ui_level_actual: UI2
- owned paths: apps/wechat-miniprogram/pages/admin-orders/admin-orders.{js,wxml}；tools/merchant-reading-uiux/**；本change目录。
- 只读契约: utils/merchantStore.js、identityStore.js、storefrontStore.js；现有DTO只提供取餐日期/时间/金额/状态，不提供件数/联系人/菜品摘要。
- 依赖: UAT executor已明确以上路径无冲突，其继续独占IDE/浏览器/部署直到真实退款演练完成；以精确候选交付其统一UI2/发布。
- 范围: 列表加载/失败重试/搜索无匹配/泳道无单分别显示；移除空件数/联系人/摘要，显示实际pickupDate/pickupTime；原“备好”按钮仍导航详情，文案改“查看并备好”；请求generation保护新查询，卸载后不写页面。
- 非目标: 不改列表API/分页、商户身份、营业状态写入、登录入口、扫码、详情、订阅、真实资金。无新的详情请求。
- 验收: Node Page seam可控orders promises复现乱序与错误，无上游；UI2实际compile/页面正常列表和navigation由executor在交接窗口执行。隔离UI2 fixture覆盖错误/加载/空结果，不宣称假响应是UAT。
- 状态: DRAFT，编码前原生Codex review；修正阻断后按用户授权实现。部署前绑定完整组合SHA并重验。
