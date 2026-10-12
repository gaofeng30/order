# 隔离页面验证

`node --test tools/mini-uiux-clarity/page.test.cjs` 运行真实 Page 定义和合成服务 seam；它是逻辑证据，不能替代 UI1/UI2。

`node tools/mini-uiux-clarity/build-ui2-fixture.cjs <merchant-source-root>` 在 `/tmp/order-uiux-ui2-*` 生成独立微信开发项目。confirm/menu 和指定 merchant source 的 admin-orders 页面四种文件逐字节复制，manifest 保存 SHA256；组件、样式、util 保持真实代码。测试 bootstrap 与服务响应独立列出 hash。项目 AppID 保留已确认的公开标识，基础库与源项目一致。它没有真实会话、真实 API 或原生支付实现，禁止上传。

在微信工具打开返回目录，初始控制页可打开确认、普通菜单、商户列表。自动化仅用真实元素 tap/input 验证页面；允许通过 `automation_evaluate` 控制 `getApp().fixture` 的合成服务模式、挂起响应和计数，不能通过 setData 伪造业务页面结果。`mode` 支持 prepay-fail（在报价后停住，release('prepay') 后失败）、selection-error、quote-hold（release('quote') 后进入合成 pending）、confirm-error、extra-hold、orders-error、orders-empty、orders-hold、menu-error、menu-hold（release("menu-options") 完成迟到选项）。默认其余模式返回合成正常数据；reset() 重置合成购物车与计数，不代表真实业务恢复。订单挂起键为 orders-1、orders-2，按反序 release 验证旧响应不覆盖新结果。

每个 candidate 必须从干净 detached worktree 重新生成并验证其业务文件 hash，再执行 manifest 全场景。只把真实执行结果写入仓库外 exact SHA receipt；此夹具不能证明微信原生支付/手机号授权、云端订单或真实到账。

执行 `python3 tools/mini-uiux-clarity/ui2.py <generated-project> customer <external-evidence-dir>`；merchant组使用同一脚本但读取已显式复制的商户页面。每条CLI调用固定project；拒绝复用已存在的evidence目录；失败即停止，先保存命令、stdout/stderr、退出码或超时，再解析响应，不写PASS。customer_tail仅用于测试定位错误后的尾部重跑，不替代exact候选全量customer执行。
