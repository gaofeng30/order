# 在线点单系统文档

本仓库用于整理“政府/企业餐饮在线点单系统”的需求、技术方案和客户沟通材料。

系统目标是为餐饮企业承包政府/企业单位食堂场景提供微信小程序点单能力：用户在线浏览菜品、加入购物车、微信支付、线下自提；商户通过后台管理商品、订单、核销和经营数据。

## 当前优先级

当前 P0 不是直接开发完整生产系统，而是先做一套可以给客户预览的整体页面交互：

- 用户端：微信小程序风格页面，覆盖首页、点单、购物车、确认订单、订单详情、取餐码。
- 商户端：Admin 后台页面，覆盖商品管理、订单管理、扫码/手动核销、数据看板。
- 数据：可以先使用 mock 数据。
- 目标：先让客户确认页面效果、业务流程和功能范围，再进入正式开发。

## 文档结构

| 文档 | 用途 |
| --- | --- |
| [`online-ordering-system-requirements.md`](./online-ordering-system-requirements.md) | 纯需求文档，适合客户、产品、项目负责人阅读 |
| [`online-ordering-system-technical.md`](./online-ordering-system-technical.md) | 技术文档，适合开发、架构和实施人员阅读 |
| [`online-ordering-system-customer-discussion.md`](./online-ordering-system-customer-discussion.md) | 客户沟通与待讨论事项，适合开会确认业务规则 |
| [`online-ordering-system-prd.md`](./online-ordering-system-prd.md) | 总 PRD 草稿，保留完整上下文 |

## 已确认方向

- 首期入口：微信小程序。
- 用户范围：单位员工 + 外部访客。
- 履约方式：线下自提。
- 取餐凭证：订单号 + 订单二维码。
- 支付方式：微信支付，资金进入餐饮企业微信支付商户号。
- 退款方式：已支付订单由商户后台审核退款。
- 后台系统：商户 Admin 后台。
- 外卖配送：首期只做 UI 预留，标记为“待开放”。
- 视觉方向：商务蓝 + 绿色，强调稳定、清晰、可信。

## 核心功能范围

用户端：

- 首页
- 菜品分类和列表
- 菜品详情
- 购物车
- 确认订单
- 微信支付
- 订单列表
- 订单详情
- 订单号和二维码取餐

商户后台：

- 商品分类管理
- 商品管理
- 图片、价格、库存、上下架、售罄管理
- 订单管理
- 扫码核销和手动核销
- 数据看板
- 营业时间、截单时间、取餐地点和公告配置

## 版本规划

### P0 可预览原型

用于给客户看整体效果，重点验证页面、流程和业务范围。可以使用 mock 数据，不要求真实支付、真实数据库和正式上线。

### MVP 正式开发

实现微信小程序点单、微信支付、订单、取餐码、后台商品管理、订单管理、核销和基础数据看板。

### V1 增强

补充员工身份绑定、访客标识、餐段预订、截单时间、每日库存、退款审核、订单导出、微信订阅消息、AI 菜品图片优化等能力。

### V2 待开发

外卖配送、多门店/多食堂、多档口、优惠券、会员、评价、排队叫号屏、发票、企业月结和第三方系统对接。

## 待客户确认事项

详细内容见 [`online-ordering-system-customer-discussion.md`](./online-ordering-system-customer-discussion.md)，主要包括：

- 用户身份怎么识别。
- 是否需要早餐、午餐、晚餐等餐段。
- 菜品库存和售罄规则。
- 退款审核规则。
- 是否只有一个门店/食堂，还是有多个窗口或档口。
- 后台账号和角色权限。
- 是否需要对接财务、收银或食堂现有系统。

## 建议阅读顺序

1. 先看 [`online-ordering-system-requirements.md`](./online-ordering-system-requirements.md)，了解产品需求。
2. 再看 [`online-ordering-system-customer-discussion.md`](./online-ordering-system-customer-discussion.md)，准备客户会议。
3. 开发前看 [`online-ordering-system-technical.md`](./online-ordering-system-technical.md)，确认技术方案和数据结构。
