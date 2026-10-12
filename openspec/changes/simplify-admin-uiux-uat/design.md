# 保存反馈单一路线
保留 vanilla JS 与 Api.saveSettings 接口，不修改共享 api.js。App 保存两个进程内标志 settingsWriting 与 settingsDirty；settings 页和顶栏写入均同步检查/设写锁，并在 finally释放。切路由只清dirty，不提前释放在途写锁。
设置 form 本页维护 dirty/saving/uncertain 状态。保存期间禁用input/textarea/button并使span点击无效。读取成功后的规范字段投影含status、pickupStepMin、pickupPoint、notice、mealPeriods与serviceDates；比较忽略服务端审计/版本元数据。异常后禁再次保存，先GET核对；匹配本次提交则已保存，不匹配则保留输入提示可重新保存/重新载入。不会根据网络失败推断未写入。
回调捕获当前render generation并检查el.isConnected，旧回调仅释放锁，不修改页面和dirty状态。部署按静态页面覆盖且保留最新api.js；回退仅恢复本次三个文件。

## 原生 Review 第二轮闭环
1. 状态规范化：比较与渲染双侧先把营业中/open、休息中/closed、已截单/cutoff统一到open/closed/cutoff。fixture的读取只返回服务端canonical枚举，不用中文伪造返回。
2. App持有当前settings写入Promise，入口同步登记，完成后释放；设置页render若有Promise先等待settle，再GET初始化。重新进入和顶栏先写入再进入均必须覆盖，等待失败也先GET不能直接开放旧表单。dirty只归属当前form generation。
3. 部署组合固定完整集成SHA；依赖/接口/实现变更后产生新候选并重验。绝不临时拼入未验证最新api.js。
4. gate_type上调W3，对前端同进程并发控制追加反复点击、跨入口竞态、离页/重入、PUT成功GET失败与读回恢复证据。后端事务和金额规则未变，Go/race与数据库migration本change不适用；UAT真实设置写入和恢复读回仍必需。

## 实现审查后的竞态修正
顶栏先开始写入时，未编辑的设置页立即改为等待再读，不能继续编辑后被刷新丢弃。旧页面在等待写入结束后、发起GET之前检查有效性，退出登录不会触发旧延迟读取。成功写入先用当前Api缓存刷新共享顶栏，再判断旧表单是否有效；登录页屏蔽共享刷新。页面局部提示仍不能污染新页面。
