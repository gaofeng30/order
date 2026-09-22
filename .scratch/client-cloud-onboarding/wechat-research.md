# 微信侧客户配置清单研究

日期：2026-09-22。代码基线 `/Users/vivix/order` HEAD=`dc63312de16ad4b795bd68ef664c835aabc880a7`。仅官方公开资料和本地代码只读核查，未登录客户后台、未查询真实AppID/商户状态、未操作任何账号。本文件是研究依据，不把待确认业务判断直接变成客户已确认结论。

用户已报告小程序注册、支付安排完成：以下均按“核验当前状态，缺项补齐”，不要求重新注册、重复进件、重置密钥或切换全商户验签方案。主体类型与名称以当前营业执照和后台核准页为准，不沿用旧资料中的矛盾项，不收集证件号码到聊天。

## 小程序：责任、入口、完成凭证

| 项目 | 必要性 / 操作人 | 后台入口与完成凭证 |
| --- | --- | --- |
| 当前主体、认证、AppID | 必核；甲方确认主体，开发方可核验 | 公众平台账号基本信息、微信认证、开发设置；记录AppID和认证有效状态，截图遮挡个人证件。支付和手机号能力需要认证，注册成功不等于认证通过。[1][2] |
| 小程序备案 | 上线前必核；负责人本人核身/短信验证，开发方可协助资料 | 首页“去备案”或设置中的备案入口；凭证为审核通过状态/备案号。域名ICP备案与小程序备案是两个事项，不互相替代。[3] |
| 开发方项目成员 | 开发必需；甲方管理员发起授权 | 成员管理→项目成员；开发者、后台登录及开发管理等权限按实际工作勾选；凭证为开发方自己微信登录后能识别正确AppID并完成所需操作。体验成员只是测试资格，不等于上传/后台配置授权。[4] |
| 甲方验收成员 | 体验验收时必需；管理员或有权限者添加 | 成员管理→体验成员；凭证为指定甲方微信可打开正式配置的体验版。不要把甲方管理员转给开发方。 |
| AppSecret | 当前Go登录/服务端API需用；甲方管理员配合必要验证，授权技术人员配置 | 公众平台→管理→开发管理→AppID/AppSecret（下级菜单以实际页面为准）。复用有效现存值并注入密钥管理；完成只记录“已安全配置/接口验证通过”，不发聊天、不提交仓库，不为交接强制重置。入口及重置影响见[19]；服务端保管要求见[5]。 |
| request / uploadFile / downloadFile合法域名 | 按实际网络调用必需；开发方整理精确清单，有权限者录入 | 开发管理→开发设置→服务器域名；HTTPS及备案须有效。凭证为域名列表+关闭调试/不校验域名后真机验证。不要把`api.weixin.qq.com`填入小程序合法域名。[5] |
| 服务端IP白名单 | 条件项；依当前微信API/安全设置启用状态核验 | 核验开发设置相应IP白名单；开发方提供实际固定出口，不让甲方猜IP。与下列“代码上传IP白名单”、合法域名是三件事；本轮未读到证明所有登录API一律必须配置的官方条款，不写为全局必填。 |
| 手机号能力与额度 | 本项目使用`getPhoneNumber`，必核；甲方确认支付/额度安排，开发方验证接口 | 非个人且已认证；查看手机号能力与付费管理中可用次数/有效期。官方有1000次共用体验额度和特定免费主体，不能强制购买指定套餐；普通成功调用标准价0.03元，按实际剩余额度及业务量判断补充。凭证为能力可用+剩余额度截图+真实授权成功。[2] |
| 隐私保护指引 | 本项目处理手机号必需；甲方确认收集目的、保存期限、联系渠道，开发方填报和接入 | 设置/账号设置→服务内容声明→用户隐私保护指引；声明手机号及实际其他个人信息用途。`getPhoneNumber`需隐私声明，但**不在**`requiredPrivateInfos`字段支持范围；该字段是指定位置/地址类接口，不能把手机号塞进去。[6][7] |
| 服务类目与资质 | 必核；甲方决定真实经营模式并提供有效资质，开发方协助上传 | 设置→服务类目及对应申请页。自营餐饮候选“餐饮服务场所/餐饮服务管理企业”；官方列食品经营许可证等不同资质路径。不能把自营店强行选为“点餐平台”；后者面向商家入驻平台且要求增值电信业务经营许可证。最终按真实模式/后台要求确认，不能只见“点餐”二字就下结论。[8] |
| 订阅消息模板 | 已确认功能需要时必需；开发方配置，甲方确认业务文案 | 功能→订阅消息；模板ID、关键词ID及字段结构需匹配代码，不是仅给模板标题；用户仍需自主授权。凭证为模板配置与实机订阅/发送验证。[9] |
| 上传、提审、发布 | 上线必需；开发方上传/准备审核，甲方或获授权人员执行对应操作，管理员完成页面要求的安全验证 | 开发工具上传→公众平台版本管理→体验→提交审核→通过后发布。不能宣称每次上传都必须甲方扫码，也不能承诺所有敏感操作授权后完全无需甲方。CI上传密钥及其IP白名单仅采用CI上传时需要，普通开发工具上传不应额外强索取。[4][10] |

## 支付：当前Go实现所需材料

普通商户直连模式需要现存商户状态正常、已开通JSAPI/小程序支付权限、目标AppID已成功关联。**异主体不等于禁止绑定**：官方有补充AppID主体信息与确认流程；客户核对收款主体、经营责任及当前关联结果即可，不强迫换主体。[11][12]

| 材料/配置 | 谁操作、入口 | 完成凭证与用途 |
| --- | --- | --- |
| 商户号、JSAPI支付权限、AppID关联 | 超级管理员确认；开发方技术操作员可获查询权限；账户中心→商户信息 / 产品中心→JSAPI支付、AppID账号管理 | 商户号、权限已开通、正确AppID关联成功；不以“支付已安排”替代结果截图。[11][12] |
| 开发方技术操作员 | 超级管理员在账户中心→员工账号管理创建并授予所需权限；API安全工作可按需将技术人员设为安全联系人 | 开发方自有账号可登录并查看所需页面；超级管理员保留甲方。安全联系人可管密钥证书，属敏感授权；不是所有日常技术操作必须超级管理员本人做。[13] |
| APIv3密钥 | 超级管理员/授权技术负责人；账户中心→API安全 | 用于通知解密，32字节；配置后后台无法再查看。已有安全副本优先复用，只有遗失/泄露等才另行安排更新，不要求把值发群。[14] |
| 商户API私钥+匹配的商户API证书序列号 | 超级管理员/授权技术负责人按API安全申请证书流程办理；开发方可协助本地证书工具 | 私钥用于请求签名，序列号标识对应商户证书；通过安全渠道写入密钥管理。不是HTTPS证书，也不是微信支付平台证书。[14] |
| 微信侧验签材料 | 先核验商户当前模式；API安全→微信支付公钥 / 平台证书 | 公钥模式：公钥ID+公钥PEM；平台证书模式：有效平台证书及序列号，由开发方提取公钥。不要要求甲方同时重新申请两套，不能用商户证书替代微信侧验签密钥。[14][15] |
| 支付/退款回调 | 开发方部署真实公网HTTPS端点并配置下单/退款参数，甲方不需猜路径 | 真实支付、退款通知验签解密及订单入账证明；回调不能要求用户登录。现有代码支持两条通知配置。[16] |
| JSAPI支付授权目录 | 当前纯原生小程序支付**不需要** | 2026-09-18小程序支付官方文档明确小程序不校验、无需配置。不要把公众号网页JSAPI/H5要求混入清单。[11] |

代码依据：`services/api/internal/config/wechatpay.go:50`起读取`merchant_id`、`merchant_certificate_serial`、`merchant_private_key_pem`、`wechatpay_public_keys[{serial,public_key_pem}]`、`api_v3_key`、`notify_url`。退款通知地址由`cmd/order-api/refund_notify_url.go`从支付回调同源派生为`/api/v1/refunds/wechat/notify`，没有单独`refund_notify_url`密钥字段。公钥解析仅接受RSA2048 PKIX `PUBLIC KEY` PEM，不接受直接塞入`CERTIFICATE`；`client.go`按响应/通知`Wechatpay-Serial`查公钥表。配置测试含`PUB_KEY_ID_001`，所以公钥ID作为映射键受支持。当前请求没有主动添加`Wechatpay-Serial`选择新公钥的头；现有平台证书商户进入公钥灰度时需两边材料及全商户项目兼容，**不能仅凭“支持公钥字段”让客户立即切换全局模式**。这属于开发方适配验证任务。[15]

## 发货与交易结算：必须核验的条件项

开发方用目标AppID查询`is_trade_managed`以及`is_trade_management_confirmation_completed`，或在后台核对对应交易管理状态并保留结果；甲方/商户管理员按后台要求确认商户订单管理授权。接口文档明确存在“用户自提”`logistics_type=4`，因此“自提没有快递单号”不等于不需要上报；如被纳入管理，应按实际自提履约提交发货信息，不能伪造快递物流。[17][18]

基线仓库搜索未找到`is_trade_managed`、`upload_shipping`、`logistics_type`、`shipping_list`接入代码。这只说明搜索未找到实现，不能代表账户豁免；如目标AppID需管理，开发方必须补齐并验收，不把平台接口实现转嫁给甲方。

## 官方来源与读取状态

以下均已读取相关正文。微信developers站点通过web工具多次返回Internal Error，但同一官方URL经curl取得静态正文并提取了对应章节；不是使用第三方转载代替。微信支付文档通过web官方正文读取。后台真实权限、审核结果、额度、商户验签模式、交易管理状态均未现场验证。

1. [小程序支付接入准备](https://pay.wechatpay.cn/doc/v3/merchant/4015459512)
2. [手机号快速验证组件](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/getPhoneNumber.html)
3. [小程序备案操作指引](https://developers.weixin.qq.com/miniprogram/product/record/record_guidelines.html)
4. [小程序接入指南](https://developers.weixin.qq.com/miniprogram/introduction/)（当前页面部分叙述较旧；细粒度角色名称以后台核验，不据旧页面强说每次管理员扫码）
5. [网络与合法域名](https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html)
6. [隐私保护指引内容](https://developers.weixin.qq.com/miniprogram/dev/framework/user-privacy/miniprogram-intro.html)
7. [app.json requiredPrivateInfos](https://developers.weixin.qq.com/miniprogram/dev/reference/configuration/app.html#requiredPrivateInfos)
8. [非个人主体服务类目](https://developers.weixin.qq.com/miniprogram/product/material/)
9. [订阅消息](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/subscribe-message.html)
10. [miniprogram-ci代码上传密钥/IP配置](https://developers.weixin.qq.com/miniprogram/dev/devtools/ci.html)
11. [小程序支付接入准备，2026-09-18](https://pay.wechatpay.cn/doc/v3/merchant/4015459512)
12. [商户号关联AppID](https://pay.wechatpay.cn/doc/v3/merchant/4013289251)
13. [技术负责人和安全联系人授权](https://pay.wechatpay.cn/doc/v3/merchant/4015423618)
14. [普通商户开发参数](https://pay.wechatpay.cn/doc/v3/merchant/4013070756)
15. [微信支付公钥和切换FAQ](https://pay.wechatpay.cn/doc/v3/merchant/4013038816)
16. [回调通知注意事项](https://pay.wechatpay.cn/doc/v3/merchant/4012075420)（本任务上轮已读正文）
17. [小程序发货信息管理](https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/business-capabilities/order-shipping/order-shipping.html)
18. [发货信息录入与用户自提枚举](https://developers.weixin.qq.com/miniprogram/dev/server/API/order_shipping/api_uploadshippinginfo.html)
19. [AppID和AppSecret获取指引](https://developers.weixin.qq.com/doc/oplatform/developers/dev/appid)（独立复查后补充：后台研究者通过curl读取官方正文，“小程序和小游戏”章节明确公众平台→管理→开发管理；同时说明重置会使旧AppSecret失效。用于纠正原[5]不能证明获取入口的来源粒度问题。）

密钥交付原则：甲方控制账号和主体，开发方通过已授权账号及安全密钥存储完成配置。收集完成状态、脱敏截图、ID、密钥存储引用和验证结果；不收账号密码、验证码、私钥/APIv3/AppSecret原文到聊天或普通文档。
