# 生产架构问题与消融记录（2026-10-15）

状态：局部消融完成，整改未实施，完整对抗性架构审查未完成，整体 NO-GO。
分析基线及已部署 API SHA：`e1f2ea1fe3335047a2bd3e0cc6766a63e10d61b0`。
唯一生产执行入口：[生产 Runbook](../../../deploy/runbooks/order-production-release.md)。

## 用户指出的问题

1. 短期数据库凭证又用于数据库→SCF补偿唤醒，因而额外引入空闲续期依赖。官方支持刷新 token，不等于推荐这一组合。
2. API 常驻的依据来自认证和对账后台任务，尚无实际冷启动与成本测量证明必须常驻。即使常驻也会崩溃或重启。
3. 对账失败日期在内存 map 中，重启可能丢失旧账单重试；进程停机跨日也不自动补全遗漏日期。
4. 正常 PENDING/READY、等待租约及真实失败进入同一补偿机制，未满足仅对真实异常登记的收敛目标。
5. 失败等待30分钟期间每分钟检查一次，开销有精简空间；直接删调度会同时破坏可靠恢复。
6. 无原始 PRD 依据的图片历史版本和误删恢复已从生产方案移除；其余复杂度也需以业务契约和实测必要性为依据。
7. API部署、PC文件上传、小程序上传、正式发布、代码提交及业务验收必须分别记账；当前尚未形成完整上线验收闭环。

## 消融结果及证据边界

| 项目 | 基线 | 直接删除后的结果 | 证据层级 |
| --- | --- | --- | --- |
| 主动认证续期 | 空闲时可取得新凭证 | 普通请求仍可按需重登；空闲时登记凭证过期，唤醒不能自行续期 | 真实Go Client及实际SQL，本地PG16，可控时钟；认证provider与腾讯扩展为替身 |
| API常驻 | 定时处理前一天账单 | 定时前退出无处理；48小时后启动仅处理当时昨天，旧日期未补回 | 实际billing调度代码、模拟进程取消与时钟；未实际修改CloudRun最小实例数 |
| 每分钟cron | 30次等待检查，第30次新增一次发送 | 首次正常投递仍成功；失败30分钟后、重启后和租约到期均不再恢复 | 生产SQL、本地真实PG16与重启；cron/SCF为记录替身，期限受控推进，未真实等待30分钟 |

空库与正常成功清理后，补偿实验60个tick均无新增检查；不能据此推断有业务时无消耗。认证与API常驻实验以Go `-race`通过表示观察结果成立，不代表可安全删除后上线。没有测量真实云端token过期、冷启动延迟或节约资源点。

本机实验工件（不随业务代码上传，不含生产凭证）：
- `/tmp/order-auth-ablation-20261015/diagnostics/auth-ablation/REPORT.md` 及 `combined-results.log`
- `/tmp/order-api-idle-ablation-20261015/report.md`；命令：`go test -race -v ./cmd/order-api -run 'TestIdleAblationRestartSkipsOlderBill|TestBillingDailyAndRetrySchedule' -count=1`
- `/tmp/order-wake-ablation-20261015/report.md`、`experiment.py`、`experiment-results.json`；命令：`python3 /tmp/order-wake-ablation-20261015/experiment.py`

这些路径仅用于本机追溯；GitHub中本文件保存结论及限制，不声称已发布完整可复现平台测试套件。

## 官方依据与整改边界

- [CloudRun运行模式](https://docs.cloudbase.net/run/deploy/configuring/autoscaling/about-instance-autoscaling)：允许缩到0，来请求时冷启动；实例销毁会清除内存。
- [服务开发说明](https://cloud.tencent.com/document/product/1243/53551)：要求无状态并建议避免请求外后台活动；固定实例可以维持后台任务，但不是业务正确性的充分保证。
- [认证刷新接口](https://docs.cloudbase.net/http-api/auth/auth-grant-token)：提供自动获取和刷新能力，不证明本项目跨数据库/函数组合为官方最佳实践。

以上在2026-10-15核对。后续唯一整改目标：保留事务内持久业务意图、幂等与租约约束，解除进程内定时器及主动续期依赖，减少等待期无效检查。具体平台触发路径尚未验证，不将一次性到期触发当成已有可用能力，不擅自改为高权限API Key，不增加收费资源。

## 本轮规整清单

- [x] 保留已上传小程序的正式域名及模板配置，预览/体验/正式阶段均使用Prod；UAT分支保持独立。
- [x] 更新API003、COS小样本、初始化数据恢复及OWNER授权的脱敏回执。
- [x] 修正Runbook中“仅异常登记已实现”和“必须常驻”的过度结论。
- [x] 记录消融事实、官方依据及证据限制。
- [ ] 修复上述架构缺口并取得平台恢复证据；本次未改业务实现。
- [ ] 精确候选独立验证、真机、PC业务、资金/通知及小程序正式发布；本次不宣称通过。

提交范围：3个小程序配置/解析文件、1个既有端点测试、Runbook、部署回执、小程序门禁记录及本记录，共8个文件。无原始日志、用户资料、token、签名URL、数据库备份或临时实验脚本。
