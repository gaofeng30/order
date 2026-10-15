# 商户核销入口直接打开相机

本切片复用现有 UAT writer worktree 和分支，基线为 `7e7f2f9a75e87a4f311e0ff7e2496022d8569f82`。仅拥有 `pages/admin-verify/admin-verify.js`、同页 WXML 和本目录工件。二维码生成器由另一 writer 独占，后端、鉴权和核销幂等契约不变。

用户已明确要求不运行或恢复测试套件。本切片仅做必要语法、差异和工件内容检查，真实手机相机验收单独记录；本地检查不等于 UI3 Green。

- [x] 1.1 Red：用户在真实手机报告从商户核销入口进入后还需再次点「打开相机扫码」，且第二次相机未识别订单 132 的真实二维码。e793 与当前基线源码均无页面生命周期调用 scan，仅 WXML 按钮触发；两步入口失败已成立。二维码识别故障属于并行 writer，未归因到本切片。
- [x] 1.2 Green 实现与静态检查：页面 onReady 直接调用一次 wx.scanCode，保持 onlyFromCamera=true/qrCode；scanning/loading guard 防止双调用，取消后恢复重新扫码按钮；manual 并发保护，原 lookup/核销结果和权限仍由既有服务端契约决定。`node --check apps/wechat-miniprogram/pages/admin-verify/admin-verify.js` 和 `git diff --check` 通过。只登记语法与结构检查，不登记真相机 PASS。
- [x] 1.3 Refactor：无需重构；不改 tabbar/nav/merchantStore/后端或二维码生成器，不在 onShow 自动触发相机。未运行测试套件或生命周期 Gate。
- [ ] 2.1 UI3 主路径：真实手机从商户订单页点核销，直接出现一次相机；扫码可识别的有效取餐二维码后，服务端核销成功且页面显示已核销。待 root 发布和手机读回；当前 NOT_RUN。
- [ ] 2.2 UI3 相邻路径：取消相机后页面不自动重开；点重新扫码可再次打开一次相机，扫描同一已完成订单仍显示已核销；手工核销在扫码/请求期间不并发执行。当前 NOT_RUN。

当前状态：IMPLEMENTED / STATIC_CHECKED。root 统一形成提交与 UAT 发布；未声明 CANDIDATE、INDEPENDENT_VERIFIED 或完整验收通过。
