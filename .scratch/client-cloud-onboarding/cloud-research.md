# 腾讯云中国站甲方开户与授权清单：官方核验

访问时间：2026-09-22 00:03–00:16 Asia/Shanghai（工具时钟 2026-09-21 16:13 UTC）。仅公开网页读取；未登录、下单、配置、测试账号权限或访问真实资源。使用 research Skill，作为主控的后台研究交付。本文不是已完成回执，也不是权限/付费授权书。

## 已知边界

- 已确认：主账号和云资源归甲方；不引入 SSM。当前没有甲方云账号、实名、余额或资源状态证据。
- 生产技术建议基线：CloudBase Run + 独立腾讯云 MySQL HA + 私有 COS；测试最小独立环境。具体地域、规格、费用、续费、日志保留和权限范围仍须甲方确认或采购前验证，不在本文悄悄批准。
- “一次性交齐”是一次列齐资料和依赖，不是保证一次会话办完，也不是保证以后免扫码/MFA/验证码。
- 人类 CAM 子用户（控制台登录）≠ 云服务角色（平台代调其他云资源）≠ 程序凭据（API Secret/STS/CloudBase API Key）≠ MySQL 数据库用户。[账号概念](https://cloud.tencent.cn/document/product/378/124853)、[CAM 服务角色](https://cloud.tencent.com/document/product/598/56395)

## A. 甲方必须持有/决定和亲自配合的事项

这里“甲方办理”是本项目责任安排；不是声称平台技术上所有动作都只允许 root。CAM 官方允许有相应管理权限的子账号创建/授权用户，但本项目不为方便而把此管理权及主账号凭据交给开发方。[新建子用户](https://cloud.tencent.com/document/product/598/13674)

| 事项与后台位置 | 甲方准备/亲自操作 | 可交开发方代办部分 | 完成证据（不含秘密） |
| --- | --- | --- | --- |
| 中国站账号：腾讯云中国站官网右上角注册/登录 → 账号中心 → 账号信息/安全设置 | 指定甲方长期保管人、甲方可持续控制的登录方式、安全手机和邮箱；注册校验由本人操作，保留主账号密码/MFA | 可屏幕指导，不接收主密码、短信验证码或主账号 API 密钥 | 主 UIN、APPID、账号归属确认；隐去手机号/邮箱局部及个人信息的截图 |
| 企业实名：账号中心 → 实名认证 → 企业认证 | 企业名称、统一社会信用代码、彩色营业执照、行业、通讯地址、企业联系人姓名/身份证号/手机号；所选认证方式需要的法人/对公账户材料由甲方直接提交平台 | 可解释字段、核对企业名称，不代法人扫码/刷脸，不在研究文档收集身份证或银行信息 | 企业认证通过状态、主体名称与拟备案主体一致 |
| 实名方式选择 | 工商企业可选法人微信扫码/人脸；公众号认证及对公认证有各自条件，不假设甲方已有公众号；法人扫码必须本人，银行卡验证由财务处理 | 可预约配合时间；认证等待不属于开发方可强制缩短事项 | 认证状态而非“材料已上传”；法人扫码认证本身不会把法人微信绑定为账号登录方式 |
| CAM：访问管理 → 用户 → 用户列表 → 新建用户 → 自定义创建 → 可访问资源并接收消息 | 创建实名对应开发人员的普通子用户，启用控制台访问；交付主 UIN/子用户名/登录入口及初始密码的安全渠道；强制首次改密是建议待执行 | 开发人员首次登录后自行改密、按提示绑定自己的 MFA；不以 API Key 代替登录账号 | 子用户 UIN、控制台访问开启、首登成功/MFA 开启、精确策略清单 |
| 子用户联系信息：CAM 用户详情 → 用户信息；安全页 → MFA/登录保护/操作保护 | 由甲方管理员填写开发人员自己的安全手机、邮箱；甲方自己的安全联系信息保持不变 | 子用户本人完成其设备校验。普通子用户不能自行改手机/邮箱，后续变更仍可能需要甲方管理员 | 已绑定状态（脱敏）、开发者能收自己的校验/通知；不收 MFA 种子或恢复码 |
| 云开发初次开通：云开发控制台 → 服务条款及服务角色授权 | 甲方确认条款、逐项核对页面显示的服务角色/策略并授权；本项目建议甲方保留 CAM 管理权 | 开发方说明用途；普通子用户若无角色创建授权，回到甲方操作，不索要 AdministratorAccess | 环境可进入；CAM 角色详情显示角色名、服务载体、关联策略；可能有多次授权弹窗 |
| 费用中心 → 账户概览/充值/订单/续费管理 | 确认一次性采购和月度可接受费用、付款人、余额预警接收人、是否自动续费/续订；甲方财务充值付款，网银及微信安全验证本人完成 | 开发方准备规格、费用测算和订单；无付款授权不得代付/开续费 | 订单号/付款状态、脱敏余额或已付凭据、明确续费选择及费用提醒回执 |

依据：[中国站新手入口](https://cloud.tencent.cn/document/product/378/52700)、[中国站注册登录方式](https://cloud.tencent.com.cn/document/product/567/14457)、[法人扫码材料与流程](https://cloud.tencent.com/document/product/378/56765)、[企业实名方式](https://cloud.tencent.com/document/api/378/10496)、[子用户登录](https://cloud.tencent.cn/document/product/598/48026)、[联系方式](https://cloud.tencent.com/document/product/598/39365)、[普通子用户限制](https://cloud.tencent.com/document/product/598/74572)、[MFA 配置](https://cloud.tencent.cn/document/product/598/73859)、[MFA 本人绑定](https://cloud.tencent.cn/document/product/598/134897)、[充值](https://cloud.tencent.cn/document/product/555/7425)。

企业对公认证可能需1–5工作日，新办/变更企业可能需等待工商信息公示；选择实际符合的方式，不要求甲方为了开云账号先新注册公众号。具体时长不是保证。[企业实名方式](https://cloud.tencent.com/document/api/378/10496)

## B. 开发方获得范围授权后可代办：权限与回执

推荐授权形式（待甲方批准）：一个普通人类子用户，关联项目专用用户组/自定义策略；按产品操作与项目资源收敛。建资源阶段部分接口只能 resource:*，仅给必要创建/枚举 action，不能声称所有接口都能精确限定到尚不存在的资源。资源落定后按真实 ID 收敛并复验；不默认授 AdministratorAccess、QcloudCamFullAccess、QCloudResourceFullAccess 或全局财务。

| 工作与入口 | 所需能力边界（不是可直接盲贴的最终 JSON） | 完成证据 |
| --- | --- | --- |
| CloudBase 控制台 → 选择环境/新建环境；环境 → 云托管 → 创建/部署服务/服务设置 | 环境查询、必要创建；指定测试/生产环境服务部署、版本/配置/日志/监控；创建的付费步骤与角色授权单列，开发人员不自授策略 | 测试/生产 EnvId、服务名、地域、套餐与状态；开发者子用户完成可见/可管理验证；两环境非同一数据库 |
| 私有网络控制台 → VPC/子网/安全组；Run 服务详情 → 服务设置 → 网络设置 → 私有网络 | 必要 VPC/子网查询及项目网络创建配置；项目安全组规则；Run 选择对应 VPC/子网。生产 CDB 使用独立 MySQL 控制台，不误点 CloudBase 内置数据库 | VPC/子网/安全组 ID；Run 与 CDB 内网地址实际 TCP/MySQL 连接成功；数据库公网入口状态；公网微信接口出站实测 |
| 云数据库 MySQL 控制台 → 实例 → 网络/安全组、账号管理、备份恢复 | 项目 CDB 实例创建/查看/配置/数据库账号及备份管理；CAM 不等于 SQL 账号。业务 SQL 用户和迁移所需权限分别按应用要求配置；不得为开发便捷把 root 暴露公网 | 实例 ID、MySQL 版本、HA/主备可用区、磁盘/备份及 binlog 保留截图、私网连通及受限数据库用户回执；无明文密码 |
| COS 控制台 → 存储桶列表 → 指定桶 → 权限管理 | 创建桶是 CAM 服务级操作；建成后仅项目桶/必要前缀的对象操作及所需配置。只要访问指定桶，可用访问路径而不授全部桶读写；运行时签名权限另设 | 桶名/地域、私有 ACL + Policy 检查、匿名读取拒绝、合法签名访问成功；禁止为解决403开放公有读 |
| SSL 证书控制台 → 证书申请/管理；Run 域名设置 | 证书查询及本项目申请/部署动作；证书私钥保密；自动 DNS/部署能力若要求服务角色，甲方单独确认该角色，不把角色策略授成人类全权限 | 证书 ID/绑定域名/到期日、HTTPS 成功、续期责任；证书签发依赖域名验证，不保证开户当天完成 |
| 云解析 DNS → 我的解析 → 项目域名 → 记录管理/权限管理 | 限定项目解析域名的记录权限；域名购买/实名/转移、DNS套餐/共享权限不是普通解析记录权限。既有域名不在腾讯云时由其管理员添加指定记录，不要求转移域名 | 域名归属、需添加的 TXT/CNAME 等实际记录与解析验证；不共享注册商主密码 |
| Run 服务详情 → 日志/监控；CLS → 日志主题 | 项目日志主题查询/检索；建立采集、索引、保留、告警另给配置权限。只读策略不能新建主题或配置告警；不默认读取公司全部日志 | 主题 ID、脱敏业务日志查询、确认后的保留期和告警接收测试；测试环境日志不混生产 |

权限证据：[CloudBase 预设策略及风险](https://cloud.tencent.com/document/product/876/47056)、[创建环境](https://docs.cloudbase.net/quick-start/create-env)、[VPC 配置](https://docs.cloudbase.net/run/deploy/networking/vpc)、[CDB 策略粒度](https://cloud.tencent.cn/document/product/236/14468)、[COS 桶策略](https://cloud.tencent.com/document/product/436/18031)、[COS 用户策略](https://cloud.tencent.cn/document/product/436/68280)、[SSL 策略](https://cloud.tencent.com/document/product/400/40435)、[SSL 服务角色](https://cloud.tencent.com/document/product/400/102955)、[DNS CAM](https://cloud.tencent.com/document/product/302/105690)、[DNS 权限限制](https://cloud.tencent.com/document/product/302/105430)、[CLS 子账号授权](https://cloud.tencent.cn/document/product/614/68373)。

### 不可混淆的权限事实

- CloudBase `TCB_QcsRole` 是云服务角色。官方列 `QcloudAccessForTCBRole` 及 `QcloudAccessForTCBRoleInAccessCloudBaseRun`，后者涉及 VPC/CVM；前者官方警告底层多种资源宽读写。开通时应核对实际角色，不照抄文档把宽策略同时挂给开发人员。[TCB 授权](https://cloud.tencent.com/document/product/876/47056)
- `QcloudTCBFullAccess` 出现在 SDK 指南；这不证明单独挂它足够完成 Run 控制台全流程，更不证明它只影响本项目。最终权限 action 清单须在选定版本、环境及实际页面核验，缺权逐项补，而不是预授权整账号。[Manager SDK](https://docs.cloudbase.net/api-reference/manager/node/introduction)
- `QcloudCDBFullAccess` 描述还含安全组、监控、VPC、KMS 等相关操作，非只管一个数据库；`QcloudSSLFullAccess` 是 ssl:* 对所有资源；CLS FullAccess 含删除主题。产品全权也不是最小权限。[CDB](https://cloud.tencent.cn/document/product/236/14468)、[SSL](https://cloud.tencent.com/document/product/400/40435)、[CLS](https://cloud.tencent.cn/document/product/614/68373)
- SSL 自动 DNS 验证可触发 `SSL_QCSLinkedRoleInCertificateDependence`；部署/更新等可触发 `SSL_QCSLinkedRoleInReplaceLoadCertificate`。仅启用实际需要功能时授权，不预开 WAF/监控等所有角色。[SSL角色](https://cloud.tencent.com/document/product/400/102955)
- 应用访问私有 COS 的程序身份与开发者登录身份分离。可行的运行时角色/STS支持及现有代码兼容性需另验；不从 CVM/SCF 文档推断 Run 一定自动注入临时凭据。若现有实现必须固定密钥，仅考虑项目专用、最小权限程序身份并安排轮换，不索要甲方主账号 SecretKey；无 SSM 不等于把密钥写代码/聊天/工件。[角色机制](https://cloud.tencent.com/document/product/598/85616)、[COS身份](https://cloud.tencent.cn/document/product/436/30749)
- 付款可细分：专项计费文档提供账单只读、余额只读及产品支付自定义能力；`QCloudFinanceFullAccess` 含付款/开票，`QCloudFinanceTrade` 是全产品支付而非某一订单。默认甲方支付，若要开发方代付，须另确认产品/权限/预算，不能仅因“甲方承担合理费用”就自动授全财务。[计费CAM](https://cloud.tencent.com/document/product/555/61542)

## C. 不能预先保证一次完成 / 需后续门槛

1. 法人、财务、主账号保管人与开发者各自扫码/MFA/验证码需要本人实时配合；普通子用户安全手机/邮箱后续修改需管理员。开发者有自己的 MFA 不等于可以绕过平台风控。[CAM用户FAQ](https://cloud.tencent.com/document/product/598/74572)
2. 服务角色首次授权可能多次；后续平台角色策略可能升级。此次授权不是承诺以后永远不再需要甲方确认。[创建Run环境](https://cloud.tencent.com/document/product/1243/77190)、[角色变更](https://cloud.tencent.com/document/product/598/56395)
3. 环境、VPC、CDB、COS等实际 ID 出现后才能最终收敛资源策略；尚无云账号证据，因此本文不虚构一份已经验证可用的最小权限 JSON。
4. Run VPC 新文档要求同地域、子网IP充足；旧内网互联文档仍写上海。新网络 FAQ 区分平台公网出站与关闭公网后的 VPC 出站，而 VPC 页面笼统要求 NAT。采购前核验当前账户选区及出网模式，必要官方工单确认；不擅自加购 NAT，也不承诺 VPC 免费/无额外费。[VPC](https://docs.cloudbase.net/run/deploy/networking/vpc)、[旧内网互联](https://docs.cloudbase.net/run/deploy/networking/internal-link)、[网络FAQ](https://docs.cloudbase.net/run/faq/network)
5. 自定义域名绑定依赖备案与域名所有权；SSL 依赖验证与签发；DNS传播、第三方审核不能一次性承诺时长。[Run概念](https://docs.cloudbase.net/run/related)
6. 待甲方确认：实际主体与账号、保管人/开发人员身份、预算和付款/续费责任、最终域名/地域、费用报警接收人、日志保留期与代办权限。生产 HA 建议与测试最小化不代表已同意下单；本轮也不新增平台版、多租户套餐、SSM或自动扩容预算。

## D. 官方来源可读性与冲突登记

以下均于上述访问时段读取。`正文可读`表示工具得到该官方页面正文或官方搜索完整正文摘录，不表示已登录其控制台；本轮没有任何后台实际验证。未显示更新时间记“未显示”，不以抓取时间冒充页面更新日期。

| 官方URL | 页面更新时间/读取状态 | 本次用途或限制 |
| --- | --- | --- |
| https://cloud.tencent.cn/document/product/378/52700 | 2026-01-08；正文可读 | 中国站注册/实名入口，不套国际站绑信用卡流程 |
| https://cloud.tencent.com.cn/document/product/567/14457 | 页面搜索标约2.6年前；正文摘录可读 | 中国站注册方式；精确当日页面选项待实见 |
| https://cloud.tencent.cn/document/product/378/124853 | 2025-11-10；正文可读 | 主/子账号、联系人、安全手机/邮箱区别 |
| https://cloud.tencent.com/document/api/378/10496 | 未显示；正文可读 | 企业实名方式、周期、地域与公示限制 |
| https://cloud.tencent.com/document/product/378/56765 | 未显示；正文可读 | 法人扫码、材料和本人验证，不代表唯一认证方式 |
| https://cloud.tencent.com/document/product/598/13674 | 未显示；正文可读 | 普通子用户选自定义创建；快速创建默认管理员风险 |
| https://cloud.tencent.cn/document/product/598/48026 | 2026-04-17；正文可读 | 子用户名@主账号ID登录 |
| https://cloud.tencent.com/document/product/598/39365 | 2024-10-11；正文可读 | 子用户手机/邮箱用途和配置位置 |
| https://cloud.tencent.com/document/product/598/74572 | 2025-09-24；正文可读 | 普通子用户不能自己改手机号/邮箱；需管理员 |
| https://cloud.tencent.cn/document/product/598/73859 | 2026-07-21；正文可读 | 新/存量子用户 MFA 配置入口不同 |
| https://cloud.tencent.cn/document/product/598/134897 | 2026-07-23；正文可读 | 子用户本人首登/下次登录绑定 MFA |
| https://cloud.tencent.com/document/product/876/47056 | 2024-06-05；正文可读 | 首次开通、TCB服务角色、宽权限警告 |
| https://docs.cloudbase.net/api-reference/manager/node/introduction | 未显示；正文可读 | SDK示例要求预设权限；不是最小化控制台权限证明 |
| https://docs.cloudbase.net/quick-start/create-env | 未显示；正文可读 | 环境创建与初次服务角色；页面还混存旧FAQ，不据此断言微信通道全部限制 |
| https://cloud.tencent.com/document/product/1243/77190 | 2024-11-07；正文可读 | 独立Run控制台旧版、多次角色授权，勿混同现行CloudBase套餐入口 |
| https://docs.cloudbase.net/run/deploy/networking/vpc | 未显示；首开工具Internal Error，重开正文可读 | 同地域、VPC/子网与NAT条件 |
| https://docs.cloudbase.net/run/deploy/networking/internal-link | 未显示；正文可读 | 仍保留上海限制，与新版同地域文案存在差异 |
| https://docs.cloudbase.net/run/faq/network | 未显示；正文可读 | 出向公网开关非公网入站开关；需结合当前模式核验 |
| https://docs.cloudbase.net/run/introduction | 未显示；正文可读 | 服务日志/监控入口 |
| https://docs.cloudbase.net/run/related | 未显示；正文可读 | 自定义域名备案依赖 |
| https://cloud.tencent.cn/document/product/236/14468 | 2025-10-29；正文可读；同路径.com工具Internal Error | CDB预设宽权限、资源级/非资源级限制；未以国际站作中国站主依据 |
| https://cloud.tencent.com/document/product/436/18031 | 2025-08-08；正文可读 | COS桶策略与创建/列表服务级权限 |
| https://cloud.tencent.cn/document/product/436/68280 | 未显示；正文摘录可读 | 按桶资源描述策略 |
| https://cloud.tencent.cn/document/product/436/30749 | 2025-01-17；正文可读 | 私有与身份/策略体系 |
| https://cloud.tencent.com/document/product/400/40435 | 2024-10-17；正文可读 | SSL FullAccess/ReadOnly范围 |
| https://cloud.tencent.com/document/product/400/102955 | 2024-01-17；正文可读 | SSL按功能触发服务角色 |
| https://cloud.tencent.com/document/product/598/85213 | 2026-09-21；正文可读 | SSL角色载体与具体策略二次核对 |
| https://cloud.tencent.com/document/product/302/105690 | 未显示；正文可读 | DNS API与控制台均支持CAM，多数接口资源级 |
| https://cloud.tencent.com/document/product/302/105430 | 2024-11-28；正文可读 | 解析写权限不涵盖套餐/共享/权限管理 |
| https://cloud.tencent.cn/document/product/614/68373 | 2026-07-15；正文可读 | CLS主题级自定义权限；.com同页搜索缓存显示2024-10-18，内容核心一致 |
| https://cloud.tencent.com/document/product/598/56395 | 2024-09-30；正文可读 | 服务角色自动创建及后续策略更新 |
| https://cloud.tencent.com/document/product/598/85616 | 2024-07-02；正文可读 | STS资源角色一般原理，不证明Run代码已适配 |
| https://cloud.tencent.cn/document/product/555/7425 | 未显示；正文可读 | 财务充值可能要求经办人扫码/网银配合 |
| https://cloud.tencent.com/document/product/555/7424 | 2026-03-02；正文可读 | 云费用账户管理权限 |
| https://cloud.tencent.com/document/product/555/61542 | 未显示；正文可读 | 专项计费细粒度策略；余额只读与支付分离 |
| https://cloud.tencent.com/document/product/598/18795 | 2026-06-12；正文可读 | 旧FAQ称无单独账单只读，与专项计费文档冲突；采用专项页并待后台验证，不因此授全财务 |
| https://cloud.tencent.com.cn/document/product/555/12212 | 未显示；正文可读 | 自动续费会扣余额，产品独立规则可能优先，须甲方选择 |

## 本轮交付检查

- [x] 只写本研究文件；未提交，未改其他 writer 文件。
- [x] 将账号事实、官方要求、技术建议及待确认选择分开；每类关键结论有官方URL。
- [x] 已二次检索核对普通子用户安全联系方式、CAM/服务角色、DNS权限和财务文档矛盾。
- [ ] 真正开户/实名/授权/付款/资源配置与验收：均未执行，不可标已完成。
