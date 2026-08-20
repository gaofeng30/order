# 本 change 未实施即归档：被 realign-mvp-product-baseline 取代

## 发生了什么

两条并行工作线各自把生效 spec `mvp-product-baseline` 对齐到 0818 PRD：

- 本 change `adopt-0818-prd-baseline`（远端线，`f716348`）—— 产物完整但**未归档**，delta 从未应用到生效 spec。
- `realign-mvp-product-baseline` + `complete-baseline-traceability`（本地线，`e451f52` 归档）—— 已应用到生效 spec，且后续四个 change（会员券删除、入口屏删除、目录字段删除、六态状态机、取餐时间点）的 delta 全部基于其 requirement 标题。

两份产物**零文件冲突**但**语义重叠**：requirement 集合高度一致，措辞与标题不同。

## 为什么保留本地线

本 change 的 `MODIFIED` / `REMOVED` 按 requirement 标题匹配，而那些标题在生效 spec 中已被本地线 REMOVED。若在此时归档本 change，delta 无法匹配，会产生失败或重复 requirement。

反向回滚本地线则需重写后续五个已集成 change 的全部 delta。

经用户于 2026-08-21 裁决，保留本地线。

## 本 change 的成果去了哪里

以下两条 requirement 为本 change 独有，已由 `reconcile-baseline-alignment` **逐字吸收**进生效 spec：

- `Pickup identifiers and notifications follow the frozen contract`
- `Production facts and statistics come from server-confirmed data`

其余 requirement 的语义均已由本地线覆盖。

`checks/verify_baseline.py` 在归档前对当时的树为 `BASELINE_CHECK=PASS`；归档后该脚本按自身路径深度推导仓库根（`parents[4]`），目录多一层后解析失效，报 `missing required file`。归档产物不再修改。

其「旧 PRD 必须是薄指针且指向 0818」的约束已被 `reconcile-baseline-alignment` 的门禁 `check_baseline_single_source.py` 继承，并扩展为：生效 spec 覆盖两条吸收进来的 requirement、不得存在第二份未归档的 baseline delta、0818 PRD 不得再自称修订提案、生效 spec 中不得有已废止概念的肯定性表述。

## 未被取代的部分

`docs/product/online-ordering-system-prd.md` 改为薄指针这一动作由本 change 完成并已生效，不在取代范围内。0818 PRD §16.4 C2「与现行基线维护方对齐」因此判定为已完成。
