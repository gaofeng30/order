# Spec: pickup options read contract

## HTTP interface

匿名只读：

```http
GET /api/v1/menu/pickup-options
```

请求无 body，本 change 不定义或读取任何业务 query 参数。成功设置 `Cache-Control: no-store`，并精确返回：

```json
{
  "timezone": "Asia/Shanghai",
  "dates": [
    {
      "date": "YYYY-MM-DD",
      "orderable": true,
      "meals": [
        {
          "code": "lunch",
          "cutoff_at": "RFC3339 with +08:00",
          "orderable": false,
          "pickup_times": ["HH:MM"]
        }
      ]
    }
  ]
}
```

失败统一为 HTTP 503：

```json
{"error":{"code":"MENU_UNAVAILABLE","message":"menu temporarily unavailable"}}
```

不得泄漏 SQL、DSN、内部错误或部分 dates/meals。既有 `/api/v1/menu` 的 DTO、错误、匿名性、query 选择语义不变。

## 数据与投影语义

- 每请求只调用一次 `Reader.MealPeriods` 读取完整 `meal_periods` 配置，只调用一次注入 clock。
- 唯一字段事实源：`code`、`cutoff_time`、`pickup_start_time`、`pickup_end_time`、`interval_minutes`。不调用 `Reader.List`，不读其他状态。
- 响应时区固定 `Asia/Shanghai`，dates 精确为 clock 对应上海本地“今天、明天”，按日期升序。
- 每个 date 完整返回一行 lunch 与一行 dinner；按 pickup start 稳定排序。已截餐段及其所有时间点仍完整返回。
- pickup times 是闭区间 `[start,end]`，从 start 每次增加 interval；格式 `HH:MM`，末端必须恰好包含。
- `cutoff_at` 是响应 date 与该餐段 cutoff 的上海时区 RFC3339 值，必须为 `+08:00`。
- `meal.orderable = now.Before(cutoff_at)`；精确到 cutoff 即 `false`，cutoff 后也为 `false`，明日依据明日 cutoff 独立计算。
- `date.orderable` 是当天任一 meal.orderable 的 OR。
- `orderable` 只表达餐段截单事实，与现有 `/api/v1/menu` 的 `meal.orderable` 同义，不是 selection、商品餐段、售罄、checkout 或支付许可。

## 完整配置与 fail-closed

配置必须精确一行 lunch + 一行 dinner。以下任一情况返回统一 503，且不得返回部分成功：

- Reader 的 query/scan/iteration/close error；
- 行数不是 2、重复或未知 code；
- 存储时间不是 exact `HH:MM:SS`、不是 ASCII 数字、越界或秒不为 0；
- interval 不在 `1..1440`；
- cutoff 晚于 pickup start；start 晚于 end；range 不能被 interval 整除；
- 两个闭区间餐段相交，包括只共享一个端点。

跨截单竞态不在本 read contract 内做锁定或缓存；未来客户端选定时刻后重新请求现有 `/menu`，checkout 再做最终校验。

## 非目标与 evidence binding

不读取或返回 storefront business status、特殊日期、商品/售罄、身份、报价、支付、订单；不新增 repo/port/migration/缓存/配置 fallback。

- `base_sha`: `f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`
- `lifecycle`: `PRE_CANDIDATE_WIP`
- `ui_level_target`: `UI1`
- `ui_level_actual`: `NOT_RUN`（current）
- `ui1_historical_receipt`: `PO-08-ui1`
- `ui1_receipt_status`: `INVALIDATED_NOT_CURRENT` after replay/Gate/receipt/spec/tasks changes; the exact historical observation was Chrome for Testing `151.0.7922.34`, `TOTAL 3 SUCCESS`, but it is not current evidence.
- `candidate_status`: `NOT_CREATED`
- `post_freeze_writer_gate`: `NOT_RUN_AFTER_FINAL_FREEZE`
- `post_freeze_order`: first Writer Gate exit 0 -> record exact receipt and check PO-08 -> stage final governance tree -> rerun staged Standards/Spec pre-review to zero finding -> rerun Writer Gate on that identical final staged tree -> keep final terminal receipt external to the frozen tree -> only then commit candidate.
- 只有提交完整实现、spec/tasks 后才记录 `candidate_sha`；任何实现、spec、tasks、Gate 或 SHA 变化都使 review/verifier receipt 失效。
