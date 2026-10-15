# 固定图片版本

## 单一路线

公开 key 仍为 `images/<sha256(decimalActor+':'+rawIdempotency)>/<contentSHA>.png|jpg`。确认先按原前置规则验证，再查当前 OWNER scope 回执；未确认时一次 GET 取得同一响应的字节和原生版本，按原规格验证后，crypto/rand32字节生成 `images/published/<64lowerhex>/<contentSHA>.png|jpg`，只做一次 PUT。客户端从不取得该物理 key 的 PUT 链接。

确认事务锁当前 OWNER，再锁 target hash；唯一命令回执及全局一致 pin 选出单 winner，不 update 既有回执。返回公开 key 保持等于请求。重放从固定发布版本读回并验证；源暂存对象被覆盖或删除不影响结果。缺 pin、历史无版本回执、冲突 pin、损坏响应均安全失败，不回退 latest 或自动猜版本。16并发最多产生16发布对象，只有一个引用；未引用对象保留受控管理清单，runtime 不自动删除。

`NewSharedPGVersionedService(*COSAdapter,sharedPGExecutor)` 实现现有 Application、PublicURL、Confirmed。native Service、Adapter、ConfirmationStore 的公开接口不增加方法。原 422/413/409/503 及校验优先保持，尤其同 key 大小/MIME不符按固定图片校验失败，不能提前改为幂等冲突。

## 内部 RPC 与证据

replay(actor,keyHash) 返回 ABSENT 或原 digest/response；commit(actor,publicKey,keyHash,requestHash,digest,publishedKey,version,at) 原子追加并返回 winner；versions(keys1..3) 返回请求范围内唯一一致合法 pin。三个固定 INVOKER RPC 均校验既有普通服务 UID/role/UTC/searchpath。原 action、target、scope、operation、request digest 与 requestID hash不变。原请求 struct Go json.Marshal 的字段顺序和原始 opaque key 字节参与 hash；不把 raw idempotency/requestID 放入 PG text。

内部响应仅保留 object_key 并增加 published_key、version_id。准确发布 GET 与原公开 key 中的 SHA/MIME 绑定验证内容，原 digest 核对完整请求；不增加 RPC 参数或公开 DTO。receipt 时间只捕获一次 UTC，遵循原 timestamptz 微秒存储；重放不刷新。签 URL 采用另一个相同 signing snapshot，600秒、硬上限900秒，凭据须覆盖到期。

## 身份

显式 UAT 固定 bucket/region 与专用 CAM 源 key 换 STS GetFederationToken1800秒，只允许该桶 images/ 的 GetObject/PutObject；源身份和 session 都要实际负面权限验证。原子三元组、singleflight 刷新提前至少900秒，不足期/异常/撤销失败不签 URL。源 key 仅版本环境注入，不进仓库、镜像、日志、命令行、客户端；Prod 默认 CVM provider 保持原决策。

## 安装通道

用户已授权在同一UAT库的事务内经官方 `ExecutePGSql` 执行冻结delta。实际读回精确定义、owner、ACL、数据和序列，保持旧36项平台历史与44项业务账本，再以普通身份完整readiness核验108。只有实际安装完成后，才经 `RepairPGUserMigrationHistory` 登记 Version `20261009193330` 与该版本精确 Query；读回37项平台历史、确认旧36项不变，并核对同字节完整37项 `Preview`。Repair只登记已完成事实，不执行SQL，不以自写管理表替代。安装和登记不是跨接口原子操作；未知结果先核实，已装后不重执行105基线guard。旧Push故障属于历史通道证据，不再作为此路线的阻断。实际SQL安装、读回、已执行Query登记、37项Preview及原普通身份readiness108均已通过，回执位于 `direct-pg-install-closure/closure-receipt.json`；真实4.2也已闭环，见 external `actual-image-core/closure-receipt.json`；候选独验仍待完成。实际SQL登记与普通登录认证是分别验证的事实，不把独立认证失败当作真实安装记录的阻断。

## 发布和恢复门

聚合4文件与第七/第八串行；最终新基线必须重新候选与独验。版本 Enabled、非当前30天 lifecycle、确切版本恢复、DB备份与全部引用图片一致性、环境参数整体5KiB及 CloudRun 无流量 worker 存活均需实际证明。配置30天不能说已观察30天。未配置的真实资源或未跑页面不得计 PASS。

实际恢复证据：原版本A经覆盖B和删除标记后仍可按原pin读取，删除精确新增marker/B后原版本A恢复为latest，原应用pin及字节未变；实际快照在断网副本恢复，数据/序列/ACL及全部原图片引用一致。逐版本私有字节备份后，本次测试数据与21精确对象版本已清理，最终数据库完整快照回到安装后基线、images版本清单为空。云端worker、环境配置整体大小及手机UI3由后续运行部署change验收，不作为本次已完成存储核心的未实现依赖。
