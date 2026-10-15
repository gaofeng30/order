# Writer 验证记录
基线01312131fb16c7121fc72b2d02532af4bd4ab65d。方案及实现原生Codex review无阻断。
Red：反序完成旧/新泳道查询，期望订单2却得到订单1。添加generation守卫后同一断言通过。测试导航对象跨VM深比较最初失败属于测试夹具问题，改为实际route/id原始值比较，不作为产品Red。
Green/Refactor：node --test tools/merchant-reading-uiux/page.test.cjs 三项通过，覆盖旧成功/旧失败/卸载后响应、当前失败重试到空结果、动作仍进入相同详情。保持最小页面实现，无额外抽象必要。
openspec validate clarify-merchant-order-list-uat --strict与git diff --check通过。删除未提供的联系人/件数/summary，展示现有pickup字段；无营业状态写入/扫码/后端改动。
Node逻辑证据不替代实际渲染。Writer UI2已通过；候选SHA、exact独验和UAT仍待仓库外收据；隔离夹具由客户确认页change的build-ui2-fixture.cjs构建，须显式传本商户source root并记录页面hash。


2026-10-12 Writer UI2：独立生成项目 /tmp/order-uiux-ui2-tqSVvy，基础库3.6.6，商户四个页面文件hash与本writer一致。实际点击打开列表→取餐2026-10-12 11:30/09:00下单/查看并备好→同id1详情；无空件数联系人。合成读取失败实际重试到empty；两次泳道查询反序释放仍显示新id2；搜索999显示没有匹配的订单。/tmp/order-uiux-writer-merchant 保存实际CLI响应与截图。无真实营业状态、扫码或登录写入。
C=10/T=9/V=8/R=9，总36，无Writer硬阻断。原生实现review无发现，所有必需逻辑/模板编译与实际UI2完成。隔离验证工具属于customer change，仅为测试依赖；exact独验需记录其固定SHA和本商户业务hash，不能把customer候选当作本商户候选。

评分元数据更正：Writer V上限8；C10来自cf28aa1原生独立审查的15项实际DTO解析、字段、动作导航和异步契约检查，未发现未解释差异。T9/V8/R9，总36。业务页面无变化，新文档SHA需重新验证；评分不替代UI2和运行收据。
