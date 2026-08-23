# Red / Green / Refactor evidence

**Base:** `5e937f3599a16f4813d6021f4cd2dd637c3156a2`

以下结果均已在 v13 exact base 从头重跑；旧 base 证据仅保留在历史文件。

## Red

```yaml
test: TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically
command: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -run '^TestNormalizeCanonicalizesSuccessfulCallbackAndQueryIdentically$' -count=1
result: FAIL
first_decisive_error: no non-test Go files in services/api/internal/paymentobservation
isolation: fresh detached base with only normalize_test.go copied; production implementation absent
```

## Green

逐个 tracer bullet 先暴露并修复：callback/query canonical equality、mismatch 最小耐久结果、
malformed expectation、typed provider errors、支持状态映射、关键事实 collision、mismatch precedence、
重复/并发稳定与 interface 数据最小化。

最终 focused：

```yaml
command: GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/internal/paymentobservation -count=1
result: PASS
```

## Refactor

公共 interface 不变；将公共类型与实现分离为 `types.go` / `normalize.go`，简化 precedence test
fixture。随后重跑同一 suite：

```yaml
focused: PASS
race: PASS
mutation: PAYMENT_OBSERVATION_MUTATION_GATE=PASS killed=7
```

Mutation 分别破坏 canonical domain、引入本地时间、交换 precedence、把 mismatch 伪装为 accepted、
让 rejected 保留 provider 事实、放过 malformed expectation、移除 transaction ID collision 输入；
7 个均被公共 interface 测试杀死。
