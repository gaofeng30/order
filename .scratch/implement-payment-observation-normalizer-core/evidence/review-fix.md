# 双轴 review findings 修复证据

## 校验优先级

首次 candidate `25af3402818da1c6ef6b42b3178331f2dad65415`：Standards `PASS / 0`；Spec
`FAIL / 1 P2`。该 SHA 与两份 review 已失效，不作为最终验收。

Finding：Spec 固定 Transaction 结构校验先于 source/state 组合校验，但
`CALLBACK + NOTPAY + Amount{Total:1, Currency:""}` 实际先返回
`UNSUPPORTED_SOURCE_STATE`，遮蔽 `MALFORMED_INPUT`。

RED：

```text
TestNormalizeReturnsStableTypedErrorsForUnusableProviderInput/malformed_transaction_precedes_callback_state
actual error: paymentobservation: UNSUPPORTED_SOURCE_STATE
expected kind: MALFORMED_INPUT
exit: 1
```

Green：增加该公共 seam 组合测试，仅把 `malformedStateFacts` 移到 source/state 判断前；随后该测试、
focused、focused race、7/7 mutation、fresh MySQL 8.0.46 完整邻接矩阵、全 API test/race、vet、build、
smoke、format、PII 与 owned/protected 检查全部 PASS。新 exact candidate 形成后须重新运行两轴 review。

## 全 typed 字符串 NUL 校验

第二个 candidate `30537be025e6d17b19240b1f4061ad633ae685dc`：Standards `PASS / 0`；
Spec `FAIL / 1 P2`。该 SHA 与两份 review 已失效，不作为最终验收。

Finding：Spec 要求所有输入字符串含 NUL 时拒绝，但 Transaction 结构校验遗漏 `TradeType`、
`TradeStateDescription`、`BankType`、`Attach`、payer identifier 和 `PayerCurrency`。

RED：12 个 typed Transaction 字符串逐项注入 NUL；首个决定性失败为 `trade_type` 返回
`ACCEPTED`，同次还确认其余 5 个遗漏，exit 1。

Green：只扩充结构 NUL 校验，不把被排除字段写入 Observation/canonical。另加公共 seam 测试证明
正常的 trade metadata、payer identifier 与 payer amount/currency 不改变 Observation 或 dedupe；随后
focused/race、既定 7/7 mutation、fresh MySQL 8.0.46 完整邻接矩阵、全 API test/race、vet、build、
smoke、format、PII 与 owned/protected 检查全部 PASS。新 exact candidate 形成后须再次运行两轴 review。
