> 2026-10-10 当前执行约束：用户已明确停止全部测试与回归并删除测试用例；旧测试及 Gate 记录仅作为历史证据，未运行不能记为通过。本轮直接完成 UAT SQL 安装、服务构建部署及小程序预览交付；不等待旧测试门禁。实际数据库已安装 113 个函数并登记 38 条平台执行记录，既有业务数据与账本保持不变。

# 固定 UAT 运行组合

Status: APPROVED_REQUIREMENTS / IMPLEMENTING。gate_type=W3；ui_level_target=UI0；ui_level_actual=NOT_RUN。

## 入口与配置

main.run 创建 logger 后、config.Load/database.Open 前，仅 ORDER_ENV=uat 分派 runUAT(logger)。其他环境执行原 native 主体。新 LoadUATRuntime 返回 UATRuntimeConfig：HTTP Config（Database 零值）、EnvID、SharedPG Credentials、COSMaterial、两项具名 COS 源凭据。

固定核对现有 UAT env/AppID/region/bucket/服务 UID。缺失、非法或错误身份拒绝启动；显式 native DB、管理员 API Key 或 generic 腾讯凭据拒绝。原 config.Load/LoadWithSecretSource 保持合同。AppSecret 用原字段规则；原支付、核销和订阅加载器在启动 HTTP/worker 前完整成功。PG anon audience/project 和普通登录身份校验仍由原 client 执行，无重复 JWT parser 或管理 API。

## 接线

一个普通 cloudbasepg.Client 接所有已验证 SharedPG 构造。身份、商户身份、目录、价格、报价、付款、订单查询、履约、退款、订阅、营业状态、售罄、对账、后台命令和审计完整接入原 router。私有图片使用受限 STS、原 COSAdapter 与 SharedPGVersionedService。订阅直接组合原 WeChatProvider、Shared 身份 resolver 和原模板 resolver；退款用同一个 Shared subscription。履约在既有 RPC 内原子 enqueue，不追加通知。

付款 Config 显式固定 `Environment=uat`。退款及账单各使用批准的五字段 Config：Environment、AppID、MerchantID、PaymentNotifyURL、RefundNotifyURL；前四项取同一 payment runtime，退款回调由原 productionRefundNotifyURL 生成。native main 同样用原环境与同源材料适配批准构造。Finance `61df2807938839b27b85248a553ba1f904239c4e` 完整86路径与 Storage `2811cabfb21005330c38abb205ce7f4810974f8b` 独占4文档现已精确 git blob 采用；逐项见外部 exact-dependency-adoption.json。最终采用后的组合 ordinary23项通过，配置/provider构造源未变；Finance唯一 native账单保存增量由其候选验证覆盖。最终发布仍等待依赖精确独验及本组合生命周期。

后台命令沿原 CommandAdapter 委托 Shared payment/refund，原 reader 投影。保留原鉴权分组、Idempotency-Key、DTO、错误码、微信验签与支付金额规则。没有 fake provider、local 文件路由、native DB 或缺失端口后的半启动。

## 生命周期

复用五个原 worker 接口与循环、四个 1 秒 ticker 和每小时上海昨日对账；无立即对账。复用 signal.NotifyContext/runWithWorkers 的 HTTP drain、取消和有界等待。ready 以 2 秒上下文调用当前 sharedpg.Check，四个绑定值为普通服务 UID、同一付款 Config 的 Environment/PaymentNotifyURL、原派生 RefundNotifyURL；不增加 ENV 兜底。live 保持进程语义。原完整注册表随依赖提供，不在 main 写死 RPC 数量。

测试通过本地 TLS 路由使用原 HTTP/provider/Shared 服务，不发送外部请求。生成夹具固定来自实际隔离 PG 普通身份的113实体/109有效执行权；成功 ready、定义/ACL/身份漂移安全失败及2秒 HTTP超时均已 ordinary 验证。漂移夹具使用 UseNumber 保留序列 int64，只修改预定字段。race 首次与无并行构建的最小复测均在成功 ready 路径超过2秒，未通过；不得放宽生产预算或用此前未覆盖完整定义的 race 代替。阶段实测RPC前ordinary 97.55ms、race 2262.47ms；真实Check成功/三漂移的独立10秒有界测试context及原登录/worker取消race已通过，不能改写完整HTTP race结果。四个 due RPC 的原限额与五循环取消已通过；小时对账未提前执行，真实 due 推进、lease 交接、CloudRun SIGTERM 仍需实际运行回执。

## 镜像与发布

专用 UAT Dockerfile 的最终 stage 提供冻结 11 项公开订阅完整元组。14 项运行身份与秘密固定由服务版本注入，root 本轮实际 EnvParams 4603B≤5120，25项完整原provider/crypto本地构造通过；真实 API 接受与容器 25 项核对待执行。构建上下文、镜像、layer、history、日志及命令不包含秘密。构建 SHA/image digest/配置版本/数据库能力清单绑定同一发布。

CloudRun 发布唯一采用精确候选编译→外部final-stage包→现有环境内授权的 `manageCloudRun targetPath` 代码包通道。完整原Dockerfile保留编译权威，候选以同一Go1.26.5、CGO0/linux/amd64、trimpath及原ldflags编译。外部包Dockerfile逐字保留完整final stage，仅将从build stage复制产物改为复制同包中已绑定SHA的二进制；runtime digest、11ENV、65532、8080与入口不变。候选/编译器/产物/包摘要绑定实际BuildId/RunId和image digest。手动TCR仓库发现已遭CAM拒绝，该实现依赖不构成UAT平台不可用结论；不变更身份/权限、采购或增加CODING/TCR凭据。代码包实际装配/部署尚未执行，须在最终精确候选与实际普通113/109 ready后取得真实结果读回。

实际部署沿已批准 Min1/Max1/0.25 CPU/0.5 GiB起点测量，不自动采购、开启 PAYG 或改轮询周期。平台 HTTPS 入口不依赖 ICP 的验收；小程序传输按官方能力与真实核心 UI3 闭合，IDE 转换受平台阻断时不将其视为唯一接入前提。环境关联、真实设备、财务资金范围及必要运维回执分别完成。

UAT 发布前闭合最终普通 RPC/schema/ACL/身份/账本检查，不先部署未 ready 的业务版本等待迁移；ready 不控制 worker 启动。95 场景/25 页面及不影响本次部署的全量验收保留为 `DEFERRED_PRE_PROD`。条件未变的 exact 证据直接复用，变化按影响面和必要相邻验证，不把延期或未运行记为通过。

## 恢复与失效

配置错误在监听前失败。候选变更需重新自测与 clean detached exact 验证；依赖、配置、镜像或 schema 变化使相应验证失效。容器回滚只用与当前 schema 兼容且已经验证的 digest；不逆转已提交财务数据。发布前一致性备份、隔离恢复和实际旧新 worker 租约交接必须闭合。
