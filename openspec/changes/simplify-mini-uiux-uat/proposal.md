# 顾客确认订单交互减负
## 授权与范围
用户 2026-10-12 授权三端 UI/UX 独立实现与部署验收、通过后合入 UAT，编码前原生 Codex review。此流水仅改显示、状态提示与导航，不改变支付/退款/核销/权限/定价规则。
- base_sha: 01312131fb16c7121fc72b2d02532af4bd4ab65d
- gate_type: W3
- ui_level_target: UI2；ui_level_actual: UI2
- owner: 本会话；branch: codex/order-mini-uiux-clarity
- worktree: /Users/vivix/.codex/worktrees/order-mini-uiux-clarity/order
- owned paths: apps/wechat-miniprogram/pages/confirm/confirm.{js,wxml,wxss}；pages/menu/menu.{js,wxml}（均在 apps/wechat-miniprogram 下）；tools/mini-uiux-clarity/**；本 change目录。
- 只读契约: utils/transactionStore.js、merchantStore.js、orderStore.js，services/api/**，稳定spec/PRD。
- 外部依赖: 现有 UAT executor 独占当前 IDE、浏览器、部署；先核对 owned paths，未交接前不操作现有会话。无共享路径变更则本地先行，集成必须重新核对最新基线。
- 状态: APPROVED_IMPLEMENTATION：用户范围授权与两轮原生review findings已落实design，进入逐切片RGR。

## 最小完整改动
1. 确认页报价前显示“预估合计”；报价后展示服务端 Quote 原价小计、折扣（分转元）和应付，商品行使用同一个 Quote 的 name/quantity/prices/notes。保持原报价→预支付→支付→服务端确认顺序与单笔支付重试语义。预支付失败清除 Quote 后恢复当前购物车估价，不能混用不同快照。不新增变价确认流程。
2. 对 QUOTE_SELECTION_UNAVAILABLE 持续显示“商品或取餐时间已不可用，请返回菜单重新选择”，提供返回菜单入口；保留购物车、联系人和备注。暂时网络失败显示可重试，不误判商品失效。按钮沿用当前支付并发保护，不增加新请求、自动重试或新预支付。商品编辑清除旧报价状态。
3. 去除“服务端 Quote”等内部文案；联系信息、手机号绑定和身份匹配保持原契约，不调整原生授权调用。可选附加手机号不在此切片折叠，避免扩大身份写入面。
4. 返回菜单采用保留确认页的明确导航：confirm.editPickup 使用 wx.navigateTo('/pages/menu/menu?returnToConfirm=1') 打开一个菜单页；该菜单的确认入口检查上一页确为confirm，再navigateBack返回原页。confirm.onShow在无预支付时刷新购物车与取餐选择并清旧Quote，表单留在原Page。普通菜单入口仍用原goConfirm。导航失败保留原页并提示，不在存储/全局新增姓名手机号草稿。
5. 核价、创建预支付、微信支付、确认结果及持有已有预支付期间冻结本页商品/联系信息编辑与修改取餐入口；报价/预支付明确失败且无预支付时恢复编辑并恢复购物车估价。已有预支付pending/unknown时仅继续同笔确认，不清报价或更新为新购物车估价；已创建订单才沿原逻辑清车。此变更不处理退出小程序后交易恢复，也不改原生支付调用与服务端状态机。
6. 商户列表和顾客详情作为另一独立“订单阅读状态”流水，不在本change实现。复制按钮及原生剪贴板/订阅入口不改。

## 当前规则适用
本会话2026-10-12直接收到用户提供的新AGENTS，明确要求TDD与exact-SHA独立验证，优先于基线分支2026-10-10历史停测覆盖；仅此独立UIUX流水执行自己的最小测试，不恢复既有executor测试/runner或改变其停测策略。

## 验收与非目标
按每个切片实际 Red/Green/Refactor；Node 公共 Page seam 隔离测试验证权威报价、错误恢复、旧请求和状态映射。UI2 在独立项目/已交接窗口由开发者工具实际编译运行目标页面及相邻回归，版本与 exact SHA 绑定。不连接生产、不假冒用户支付、不合成真实订单/核销 token。真实原生支付/扫描保持既有验收边界。
Writer Gate + openspec strict + mini candidate Gate 后提交候选，另一干净 detached worktree exact SHA 独立检查。缺 UI2 或 exact receipt 保留候选，不宣称独立验收/部署通过。UAT 集成分支需先固定双方 SHA，组合成新候选并重新验证，不推 main 或 Prod。
