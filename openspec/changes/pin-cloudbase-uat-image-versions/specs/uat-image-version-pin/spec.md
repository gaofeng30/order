## ADDED Requirements

### Requirement: 确认后的图片引用绑定不可被客户端覆盖的原生版本

系统 SHALL 保留现有公开 object_key、HTTP 201/no-store、校验及错误合同，在 UAT shared-PG 组合中校验上传字节后发布到仅服务端可写的随机物理 key，并以原子回执绑定其准确 COS versionId。未提交回执的图片不能成为业务引用。

#### Scenario: 旧上传 URL 被重放
- **WHEN** 已确认 A 后客户端向原上传 key 再 PUT B
- **THEN** 已返回 URL 和新签 URL 仍指向发布 A 的固定版本，发布 A 仍为其物理 key 当前版本，原非当前版本30天清理合同不改变

#### Scenario: 并发或提交返回丢失
- **WHEN** 16个同请求确认竞争，或提交后响应丢失/进程重启
- **THEN** 同 scope 只存在一个命令回执和 winner，重放当前 OWNER 授权并从 winner 恢复，不重写发布物理 key

#### Scenario: 缺失或损坏版本证据
- **WHEN** 回执无 pin、同公开 key 存在冲突 pin、版本不存在或响应包含非请求 key
- **THEN** 系统安全失败，不使用 latest，不自动补写历史事实

### Requirement: UAT COS 运行身份只访问既有桶必要图片数据

系统 SHALL 使用显式专用 UAT 源身份换1800秒临时 STS，只授既有 UAT 桶 images/ 的 GetObject/PutObject，签名600秒且最多900秒，整组三元组及签名/expiry时钟一致。源身份和 session 权限均以实际正向及负面测试验证。

#### Scenario: 并发刷新或异常凭据
- **WHEN** 32个并发签名需要刷新，或凭据不足期/空值/超时/撤销
- **THEN** 有效刷新只执行一次 exchange；异常不返回签名 URL，秘密不出现在客户端/仓库/镜像/日志

### Requirement: 存储切片完成与完整 UAT 分别验证

系统 SHALL 以精确候选 SHA、干净独立 worktree 的完整必要回归证明切片，并实际读回版本/lifecycle、原生恢复、DB+引用图片一致性。完整运行还须 CloudRun worker、PC/Mini 及既有95场景25页/UI3的真实回归。

#### Scenario: 仅本地或合成验证通过
- **WHEN** 本地或合成 COS/STS/PG 测试通过而真实环境验证未完成
- **THEN** 状态仅记录对应层，不宣称部署或完整 UAT 已完成
