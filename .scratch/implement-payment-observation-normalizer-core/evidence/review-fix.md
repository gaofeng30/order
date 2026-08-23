# 首次双轴 review finding 修复证据

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
