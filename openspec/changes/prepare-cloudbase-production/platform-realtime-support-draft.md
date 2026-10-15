# 未发送的诊断记录：Prod Realtime握手503

用户已要求优先自查调用方式；本草稿暂停提交。尚未确定平台根因，最新HTTP RPC成功与Realtime最小调用对照见tasks.md。

状态：待用户授权提交；未发送。不申请付费支持、套餐升级或新增资源。

目标环境：order-prod-d0g4xnw49c5ada5ce，上海，现有标准版。
数据库：postgres-gy041jr4，共享PostgreSQL17.11，控制台显示运行中。

问题：使用官方@cloudbase/js-sdk@3.10.2及ws，普通内部身份登录成功。app.realtime()订阅postgres_changes在连接阶段CHANNEL_ERROR，底层Unexpected server response: 503，尚未SUBSCRIBED。按SDK源码beforeTransportConnect相同setAuth()获取握手参数后，直接读取握手响应仍503，RequestId：0a96a62c-c9be-47cc-ac0d-3ec690693d61。日期2026-10-15；具体时刻以平台请求ID记录为准。

对照：管理SQL可用；普通登录返回正确环境和authenticated角色。目标库起初没有publication及cloudbase_realtime_admin角色。曾创建独立测试表、UID限制SELECT RLS与仅含测试表的publication；握手仍503，connector未创建。无业务数据与支付操作。测试表/publication已清理并确认均0；清理读回RequestId f1087e9a-480a-4dc6-a13f-d96472226050。未修改复制槽或创建系统connector角色。

请确认该Prod环境Realtime服务是否已部署/可用，503的服务端原因，以及现有套餐内官方启用或修复路径。若需另付费，请先只提供明确报价与替代方案，不进行付费变更。不要将管理SQL正常等同于Realtime可用。

无附件凭据，不发送密码、API key、access token、个人或经营数据。
