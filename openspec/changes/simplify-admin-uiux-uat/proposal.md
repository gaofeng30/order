# Admin 与小程序交互减负：分流水实施

## 授权与目标
2026-10-12 用户授权评估 Admin、用户端、商户端交互，在独立流水实现、部署验证，验证通过合入 UAT；编码前用 Codex review 对抗审查，不与现有 UAT 执行者冲突。此授权不扩展至 Prod、真实资金确认、权限或业务规则调整。

## 基线与边界
- base_sha: 01312131fb16c7121fc72b2d02532af4bd4ab65d
- owner: 本会话；branch: codex/order-uiux-clarity
- worktree: /Users/vivix/.codex/worktrees/order-uiux-clarity/order
- gate_type: W3；ui_level_target: UI1；ui_level_actual: UI1
- 状态: APPROVED_IMPLEMENTATION：用户已授权此范围；原生review两轮13项已逐项落实，见design补充。W3为前端并发互斥，未扩展业务写入/后端范围。
- 只读共享契约: services/api/**、openspec/specs/**、docs/product/**。
- owned paths: apps/web-admin/pages/settings.js；apps/web-admin/app.js、app.css；tools/uiux-clarity/**；本 change 目录。
- 现有 UAT executor 独占现有 IDE、浏览器会话、云部署及 UAT 验收账本，未交接窗口前不触碰。合入 UAT 指协调后的 UAT 集成分支，不擅自将尚未完成生命周期的父分支合并 main。

## 单一主能力：营业设置的保存与恢复反馈
1. 设置修改明确未保存，点击保存后显示保存中并禁止本页继续编辑和重复提交。
2. 与顶栏即时营业切换共享同一页面进程的写入互斥。设置有未保存内容或结果未确认时，顶栏切换提示先处理当前设置；任一入口在写入时拒绝另一入口。本地互斥不宣称替代多窗口/多账号服务端一致性。
3. saveSettings 的 PUT 后 GET 可能失败，因此 reject 一律保守显示“保存结果未确认”，保留表单并提供“核对已保存设置”。该动作只 GET：与提交的规范字段一致则确认已保存；不一致则提示保留本页输入、可重新保存或重新载入。读回失败继续未确认，不能自动重发 PUT。
4. 本页离开后旧回调不改新页面或弹成功提示，但释放全局写入互斥；未保存编辑状态在离开本页时清理。不会自动保存或提示已成功。
5. 将“切换后同步”改为“修改后点击保存才生效”；顶栏即时入口标明立即生效。其他工作台、分页、导入、账号/员工文案独立排队，不混入本候选。

## 非目标
无金额、权限、时区、导入、分页、原生能力或交易流程改动；无 API 行为、后端、数据模型或保存次数扩展。不修改现有 executor 的 api.js（含已发布700031c售罄修复）、IDE、云和验收账本。

## 当前规则适用说明
本会话用户于2026-10-12直接提供的新 AGENTS 指令明确要求 Red/Green/Refactor 与 exact SHA 独立验证，优先于基线分支2026-10-10的历史停测段落。本流水仅新增其自身最小验证，既有 UAT executor 的停测策略和已删除文件保持不动。用户最新授权实现、部署验证和合入 UAT，不扩大到 Prod、资金或 main。

## 验收与部署
本地隔离HTTP fixture运行真实 Admin HTML、app、settings、api；仅监听127.0.0.1，不连UAT。实际浏览器先观察Red，再Green：未保存→保存、连续点击仅一次写入、保存期间顶栏切换被阻止、PUT成功GET失败→结果未确认→GET核对恢复、PUT拒绝→读回不同→保留输入可重试、离页后无旧回调污染。fixture日志只记录计数/状态，不存表单或凭据。JS检查、strict与diff检查明确PASS。
Writer Gate 后候选提交；干净detached worktree重复本地场景并记录exact SHA。UAT部署窗口与executor协调，先固定executor的完整UAT集成SHA，将本change合成新候选，在干净detached worktree重验app/settings/api整个组合；仅从该已验证组合构建；部署包SHA与API版本记录。现场通过正常OWNER页面读取基线，编辑可逆公告后保存→离页再回读→恢复原公告并再读；验证保存提示、顶栏互斥。UAT不注入网络故障或动营业状态打断客户，故障语义以隔离实测为独立证据。不能取得身份/部署窗口则停止部署，保留候选和实际限制。
