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
- `lifecycle`: `REPLACEMENT_CANDIDATE_READY_FOR_EXACT_REVIEW`
- `ui_level_target`: `UI1`
- `ui_level_actual`: `UI1`；最终 Writer Gate 的锁定 Chromium runner 已实跑 `3/3 PASS`。UI2/UI3、生产与真实菜单 UAT 未运行。
- `business_candidate_sha`: `b6154f3c17f709223f35dbc8b0b49db7a5a2c9e0`
- `business_candidate_status`: `INVALIDATED_BY_STANDARDS_GOVERNANCE_AUDIT`；正式 Spec 轴 0 finding，业务/API 实现无 finding；唯一 Standards P1 是治理状态滞后。
- `final_staged_pre_review`: Git tree `818b9a591707669f1ddcbb7e995727b70d5e1751`, Standards 0 finding, Spec 0 finding.
- `final_writer_terminal`: PASS，`base_sha=head_sha=f3c4efa4cd665652d93d5da76f92d18c4bdc59ac`，`source_tree_sha256=8ba00c120c6bd97eda4990fa3c68f92bf3ca6454f600dbcc4b09083c57d05ece`。
- `old_candidate_detached_verifier`: exact `b6154f3c17f709223f35dbc8b0b49db7a5a2c9e0` 曾 PASS，同一 `source_tree_sha256`；因本治理修复统一标记 `INVALIDATED_BY_GOVERNANCE_REPLACEMENT`，不得声明 verified。
- `replacement_final_sha`: 由包含本治理修复的 commit 形成，并在 immutable external handoff 中绑定完整 SHA；禁止未来 SHA 自写造成无限 amend。
- replacement exact SHA 的正式 Standards/Spec review 与 clean detached verifier 均 `PENDING`；任何 replacement 实现、spec、tasks、Gate 或 SHA 变化都会使对应 receipt 失效。
