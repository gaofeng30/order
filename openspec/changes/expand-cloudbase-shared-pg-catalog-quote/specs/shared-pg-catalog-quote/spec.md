## ADDED Requirements

### Requirement: 保留目录菜单与店铺公开投影
系统 SHALL 经固定普通服务JWT RPC实现既有Catalog/Menu/Storefront读端口，复用原Go selector、身份、URL及DTO规则。

#### Scenario: 分类、可售与对象展示
- **WHEN** 分类无商品、日期缺失/店铺关闭、商品售罄或对象URL失败
- **THEN** Browse保留活跃空分类，Menu过滤空分类；Menu flags与Detail原错误不同；售罄Detail仍可读
- **AND** 商品URL失败整体拒绝，launch URL失败仅省略launch；坏bearer不降匿名

#### Scenario: 一致事实与展示价格
- **WHEN** 并发配置修改且读取用户、折扣与白名单
- **THEN** 展示Pricing以单SQL快照保留原RR，Go原primary/extra匹配及单价half-up不变，nil价格不变零

### Requirement: 目录与店铺管理保持原原子命令
系统 SHALL 实现Catalog完整十kind及Storefront两配置命令，保留当前OWNER、原错误优先级、完整原命令digest与receipt。

#### Scenario: 重放与晚消费材料
- **WHEN** 旧key重放而图片确认或未使用字段包含异常值
- **THEN** Catalog先currentOWNER及旧receipt再确认错误；Storefront保持原先执行再receipt冲突回滚顺序，不统一early replay
- **AND** 原ignored字段仍参与digest，NUL/invalidUTF8/NaNInf在原消费阶段返回原结果或错误，不因REST表达新增提前拒绝；Launch key原BYTEA与receipt JSON替换分别保留，fresh原bytes与replay持久内容不混用

#### Scenario: 配置与失败回滚
- **WHEN** 修改日期、meal、launch NULL组或发生末尾receipt故障
- **THEN** 原业务列和receipt同提交或全回滚；不删除省略日期、不修改原未要求字段，业务DB clock与audit clock保留

### Requirement: 商户售罄与确认读取保留独立合同
系统 SHALL 实现MerchantSoldOut当前授权、历史receipt及原今日明日规则，并仅从真实image-confirm receipt提供Confirmed只读依赖。

#### Scenario: 历史重放与新命令竞态
- **WHEN** 日期已过期、locator变化、noop或提交响应未知
- **THEN** 先当前授权与历史receipt，新命令才clock/locator；锁内重核全部相关事实，noop仍记receipt，遵循原共享retry预算及当前授权恢复

### Requirement: 报价不可变聚合与重放优先
系统 SHALL 保留Quote公开Create/Read的Go canonical算法，单命令原子提交完整quote/items/receipt，不能模拟跨HTTP事务。

#### Scenario: 同key赢家与预读源失效
- **WHEN** source预读结果可能导致业务拒绝且同key赢家尚在提交
- **THEN** create先取得key锁并处理existing/replay，再锁源并在SQL完整派生；不得在锁前返回预读源业务拒绝

#### Scenario: 源变化或receipt故障
- **WHEN** 并发修改源或末尾receipt失败
- **THEN** key及源锁内派生完整报价，源writer等待释放，故障不留部分聚合；capturedAt/key/request保持原单次语义，合法提交的amount/digest/owner经过完整校验
- **AND** GET不新增receipt存在条件，Create replay保留原严格receipt校验

#### Scenario: 无deadline请求的进展
- **WHEN** 正常无deadline请求与持续source写入竞争
- **THEN** 实现须经native实际对照证明可接受进展，不以有限churn通过或caller-context循环宣称等价
- **AND** 不使用SOURCE_CHANGED跨REST重算，不新增重试次数、timeout或资源拒绝；原子SQL方案须实际完整验证后才标Quote功能PASS

### Requirement: 最小权限与切片交付边界
系统 SHALL 使用固定INVOKER RPC、固定服务UID RLS与精确最小列权限，保留44迁移和原有业务接口。

#### Scenario: 验收与后续付款依赖
- **WHEN** 本轮读写适配通过本地及平台验证
- **THEN** 候选、clean detached独验与main依赖分别核验；不声称完整runtime、真实微信/COS或95Case25页面/UI3已完成
- **AND** FinalizeForPrepayInTx及LoadSnapshotInTx保持本轮未实现，后续付款必须与prepay或order聚合同事务，禁止假Tx或独立finalize提交

### Requirement: 报价派生保持原Unicode、金融与字节合同
系统 SHALL 在同一次源锁内按原业务算法计算身份、selector、价格与摘要，并由Go独立复核返回聚合。

#### Scenario: 不同姓名算法与金融极值
- **WHEN** 宽窄字符、组合字符、兼容字符或不同空白参与员工匹配，或金额乘加接近int64边界
- **THEN** width.Fold/NFC/ASCII空格匹配与NFKC/Unicode空白name_key检查分别保持，primary优先；全部中间乘加按原int64边界检查，单价half-up后再乘quantity
- **AND** 摘要为原domain/BE64/UTF8长度前缀/有序字段及Unix微秒字节，不以JSON表示或新增Unicode折叠替换

#### Scenario: receipt时钟只在需要新材料时采样
- **WHEN** create首次receipt时刻为NULL
- **THEN** 锁内先existing及完整源派生验证，合法新请求返回零写NEED_RECEIPT；Go仅此时采独立审计clock，第二调用重新完整派生并原子提交，不依赖前次proof且不循环刷新

#### Scenario: 锁定Go规范化实现的组合边界
- **WHEN** nonstarter流安全边界、Jamo或astral低16位组合碰撞参与姓名处理
- **THEN** 使用锁定Go有效数据及原segment/reorder/compose语义，包括CGJ和uint16组合键，不能以内置PG normalize、升级依赖或拒绝字符消除差异
- **AND** 生成器可重复再生、源码/版本/字段shape漂移必须失败；不引入通用Unicode服务或业务表


## Verification status

本规格对应实现已完成本地完整ordinary/race及第四普通UID平台业务/权限/清理矩阵，具体hash和分层证据见tasks。此状态不改变任何 SHALL：最终Writer Gate总包已完成，精确候选与独立验证仍待完成；真实微信/provider、CloudRun完整runtime及95/25/UI3不在本次PASS范围。
