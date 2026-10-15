# Writer 验证记录

基线 01312131fb16c7121fc72b2d02532af4bd4ab65d，独占已声明路径。只运行 loopback 合成 fixture，不持有或更改真实 UAT 会话。

## Red → Green → Refactor
- 原设置表单无 dirty/saving/unknown 恢复反馈；原生方案 review 定位 PUT 后 GET 失败不能当作未保存，双入口必须互斥。
- 实现 review 首轮复现顶栏在途仍可编辑随后丢草稿、退出登录旧页面仍发 GET；修正后实际浏览器分别观察等待态和登录页，fixture unauthorizedReads=0。
- 二轮原生 review 临时隔离复现：保存状态后离页，serverStatus=closed、重入表单显示休息中，但 chromeRefreshCalls=0。修复后慢写期间跳工作台，结束后顶栏与账户菜单均为休息中；页面无旧成功弹窗。
- 实际浏览器：普通公告编辑→有未保存修改→保存成功；故障模式 PUT成功/GET503→保存结果未确认→只读核对→已保存（writes保持2）；PUT400→只读核对差异→输入保留→重新载入恢复；保存时顶栏被拒绝；切页重入等待原写入后读取新值；退出后仍为登录页。
- fixtures 初版仅午餐不符合真实保存契约，已改午餐+晚餐并拒绝缺餐段请求；使用修正 fixture 重做上述浏览器主路径/失败态，未将旧夹具结果单独作为通过证据。
- Refactor：保持当前页面实现，无必要新增抽象；node --check apps/web-admin/{app.js,pages/settings.js} 和 git diff --check 通过。

## 命令与边界
`node tools/uiux-clarity/server.cjs` 启动随机本地端口；真实 Admin HTML/app/settings/api 的浏览器交互，无上游连接。`openspec validate simplify-admin-uiux-uat --strict` 通过。
原生 review 日志在 /tmp/order-admin-settings-code-review{,2,3}.log；第三轮结论为 No concrete introduced blockers found；修复版本无具体阻断。
未完成：候选 SHA、干净 detached 独验、UAT组合验证/部署/实际公告恢复读回。不能以本地 fixture 代替 UAT。

## Writer Gate
C=10：保存与恢复契约、共用写锁和退出行为齐备；87dcc0b后续独立逻辑及真实UAT公告保存/离页读回/恢复契约证据完整；T=9：隔离真实Api浏览器主路径及故障/竞态；V=8：Writer仅形成候选，独验结果另记收据；R=9：只有owned路径，无后端/公共契约/真实状态写入，总分36/40。无本地Writer硬阻断。提交时仅加入本change路径，提交后exact SHA和clean状态记录仓库外。

评分元数据更正：原Writer误将V记为9，违反上限8。此文档修正以已完成的独立契约和真实UAT证据支持C10，T9/V8/R9，总36。业务文件与已发布87dcc0b逐字节相同；新文档SHA仍须复验，不能把分值视作运行通过证据。
