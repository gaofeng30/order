# Writer 验证记录
基线 01312131fb16c7121fc72b2d02532af4bd4ab65d；合成Node Page seam不等于UI1/UI2。

Red：原商品小计0.20与Quote0.42不一致；选择失败字段undefined；执行中联系人被修改；返回菜单使用tabTo销毁表单；附加手机号未完成时quoteCalls=1。分别增加断言观察失败后最小实现。
实现review又复现商品编辑后 selectionUnavailable 仍true；新增第6项先失败后修复。第二轮发现修改菜单无可见返回；第7项先失败undefined != true，再添加仅returnToConfirm栈开启的navbar back。
Green/Refactor：`node --test tools/mini-uiux-clarity/page.test.cjs` 九项通过，含相同prepayment ID三次确认、清报价后恢复估价、表单保留。无额外抽象/重构必要。
`openspec validate simplify-mini-uiux-uat --strict` 与 `git diff --check` 通过；最后实现review4无引入的正确性缺陷；八项隔离回归通过。
外部生成器 build-ui2-fixture.cjs 复制业务页字节并记录hash；测试依赖无上游、无真实原生支付。详见tools README。Writer实际UI2已执行，candidate/exact独验/UAT尚待独立收据。

## 实际 UI2 与 Writer Gate
2026-10-12 微信开发者工具独立 /tmp/order-uiux-ui2-tqSVvy、基础库3.6.6、iPhone12/13(Pro)模拟器。12个业务页面文件hash与writer源一致；测试依赖单独hash。实际tap/input读取UI与Page结果，不改业务Page数据。外部可复用执行脚本 /tmp/order-uiux-ui2-run.py；原始响应与截图 /tmp/order-uiux-writer-customer 和 /tmp/order-uiux-writer-customer-tail。
报价0.42/折扣0.08/应付0.34同一快照；预支付失败恢复0.20。不可售不增prepay，返回修改到12:00并保留联系人与备注。清车后顶栏返回保留表单且不发Quote。菜单失败可返回；请求未完成时返回、迟到12:00响应不改原11:30。核价及已有prepayment锁定输入，真实页面onHide/onShow保留事务，重试始终id88且quote/prepay均1次；confirm失败仍锁定。附加手机号在途时quote0；普通菜单入口无return标识。
第一次尾部脚本使用bindtap属性选择器超时，是测试定位错误；已查询实际.field > view并重跑尾部两项通过。未将该错误当业务失败或掩盖跳过。测试依赖不执行真实微信手机号授权/微信支付、系统级后台恢复或云请求；页面onShow恢复已实际覆盖。
C=9/T=9/V=8/R=10，总分36：范围和公共契约保持，Red/Green/Refactor与实际UI2主路径/失败态齐备，fixture和真实支付边界明确。无Writer硬阻断；只提交owned paths。UAT/API/原生UI3和exact候选独验不在此Writer通过声明内。

## 独立验证返回的工具修正
fccef07业务、节点测试和实际UI2通过，但独立review因runner证据处理整体FAIL，不能集成。writer新增4项隔离Python回归：旧PASS目录复用、非JSON、超时和非零退出，先得到2失败+2错误；修复后4/4通过。runner现在拒绝已有证据目录，先保存原始stdout/stderr/退出状态再解析。业务页面未变化；新SHA必须重新跑独验/UI2，不沿用旧receipt。

## 返回故障与评分修正
74c8e61独验仍FAIL：原navbar在navigateBack失败时reLaunch丢表单、Writer V超上限、非UTF8字节未保留。新增真实Component失败回归先观察页面栈0而非2，改为菜单专用返回确认，失败只toast；正常/错误/迟到返回继续验证。runner新增真实子进程非法UTF8输出先失败，改为捕获原始bytes并同时保存base64与可读解码，5项回归通过。Writer评分按上限V8；R10基于本范围可复现的恢复闭环：核价/预支付失败恢复估价、不可售改选保留表单、在途手机保存阻止Quote、同笔prepayment重试不新建、菜单导航失败保持栈、迟到响应不改共享选择、验证重跑隔离与原始失败证据保存。C9/T9/V8/R10=36，不以分数替代任何Gate；真实原生支付/云端资金边界未变化。

当前writer增量UI2：/tmp/order-uiux-ui2-PsWhPQ 实际专用返回按钮、navigateBack故障注入后停留、恢复后保留联系人、菜单失败返回、迟到选项不覆写均通过；/tmp/order-uiux-return-writer保存原始调用。此前74的全量UI2仅用于未变化部分Writer证据，新candidate必须重跑完整UI2。
