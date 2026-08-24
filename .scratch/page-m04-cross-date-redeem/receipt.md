# PAGE-M04 跨日逐单核销变更回执

- Base: `7a3fd996a4702c5514dc69aeff6c49f7145bb7ae`
- Branch: `codex/page-m04-cross-date-redeem-l3`
- Scope: 仅扩展已集成商户页面闭环 runner/spec/gate；不改 production、backend、schema、manifest、用户小程序或 PC。

## TDD

- Red: `node --test apps/wechat-miniprogram/tests/merchant-pages-closure-m04-cross-date-contract.test.js` 为 0/1，首错 `missing PAGE-M04 cross-date runner`。
- Green(static): contract 1/1、runner/spec syntax、gate syntax/JSON、WX lint 与 owned-path audit 全部 PASS。
- Green(W3): Chrome 151 rendered 5/5 PASS；106 次 root HTTP；今日/明日同码订单均为 COMPLETED；`fulfillment.redeem_order` 审计恰好 2 条；API 停止、临时 schema 与 tmp 残留均为 0。提交后 clean exact-SHA receipt 输出到 `/private/tmp`，不写回仓库以免自失效。

首轮 W3 的真实错误为明日 fixture 的 `preparing_at=UTC_TIMESTAMP(6)` 导致 `mark ready` 返回 `503/FULFILLMENT_UNAVAILABLE`。最小 harness 修复为使用该 HTTP 订单自身的 `materialized_at`，不绕过生产 fail-closed 校验；同窗第二轮 Green。

## 验收语义

- 今日与明日订单均经 private root HTTP 创建，两个日期各自分配首个四位码 `0001`。
- Chrome 151 真实 rendered `admin-order-detail` 按订单 ID 加载并逐单核销，响应中的订单 ID、服务日期、取餐码必须保持一致。
- 同码手工核销只命中当前服务日期的今日订单；新幂等键重复请求返回首次 durable COMPLETED 结果，不得串到明日订单。
- MySQL 断言两单分别保留各自服务日期、均为 COMPLETED，并存在恰好两条 `fulfillment.redeem_order` 审计。

## Fixture 边界

真实时钟下，明日 HTTP 订单按契约先处于 RESERVED。Gate 仅用一条带订单 ID、服务日期、原状态和空生命周期字段守卫的 SQL，把该明日 fixture 推到 PREPARING；这不作为生产定时推进证据。其后 READY、页面读取、逐单核销、同码回放与最终审计全部走 root HTTP/MySQL。
