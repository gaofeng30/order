# OpenSpec change 质量门禁协议

本文是单个 OpenSpec change 的质量分类、证据和 verdict 事实源。根 `AGENTS.md` 继续定义仓库硬规则，`openspec/specs/loop-engineering-control-plane/spec.md` 与 `order-run-loop` 继续定义跨 change 的 lane、调度、session、checkpoint、错误指纹计数和升级；本文不复制这些控制面规则。

## 1. Change 必填声明

每个 change 在 DRAFT 中必须声明：

- 唯一 `gate_type`：W0、W1、W2 或 W3，取全部影响面的最高风险，不取平均；
- 目标 `ui_level`：UI0、UI1、UI2 或 UI3，以及实际已达到的等级；
- owner、branch/worktree、owned paths、只读共享契约、依赖、非目标；
- Red、Green、Refactor、writer、verifier 和 integration 的验收命令或实际操作；
- 必要外部资产、asset owner、当前是否具备及恢复条件；
- `base_sha`；形成候选后再记录完整 `candidate_sha`。

实现发现更高风险面时，先上调 `gate_type`、同步 proposal/design/spec/tasks 并重新批准；不得用文件数量、实现难度、平均风险、测试数量或覆盖率降级。

## 2. W0-W3 最高风险分类

| 类型 | 任一命中即进入的最高风险 | 最低业务证据 |
| --- | --- | --- |
| W0 结构 | 只改文档、链接、文件结构、非运行配置或内容完整性；不改变运行行为、公共契约或数据结果 | 结构、链接、schema、内容完整性和 owned-path |
| W1 内部逻辑 | 改变模块内部行为或边界；不改变公共契约、持久化结果、资金、权限或并发不变量 | 单元、边界、错误路径、受影响回归；共享状态运行 race |
| W2 公共契约/UI | 改变公共 API/schema、任一调用方契约或用户可见 UI 行为 | provider、全部受影响 consumer、兼容/错误态和至少 UI1 |
| W3 数据/资金/并发 | 改变持久化数据、migration、权限、订单、支付、退款、库存、核销、幂等、事务、恢复或并发结果 | 并发、幂等、事务、恢复、非法状态、故障证据和业务不变量 |

一个 change 命中多行时只取编号最高者。可以独立验收或回滚的低风险结果应拆 change，不得留在高风险 change 内再降低门禁。

## 3. UI0-UI3 实际运行等级

| 等级 | PASS 必须来自 | 不能证明 |
| --- | --- | --- |
| UI0 静态 | JS/JSON/模板/资源/页面结构检查 | 页面可交互、跨端状态或真实平台行为 |
| UI1 浏览器/模拟器 | 浏览器或非真实平台模拟器实际运行主场景与错误态 | 微信体验版、真机原生能力、真实支付结果 |
| UI2 微信开发者工具/体验版 | 微信开发者工具实际编译/运行或指定体验版验收 | 未实际覆盖的真机差异、账号、支付或回调结果 |
| UI3 真机/真实平台 | 指定真机、真实账号或真实平台的受控验收，含版本、环境与最终业务结果 | 未覆盖机型、账号、环境或全量生产正确性 |

未实际运行的等级不得写 PASS。所选等级依赖的工具、权限、账号、版本或设备缺失时，记录 `BLOCKED_EXTERNAL`，不得用低一级结果冒充。

### W0-W3 × UI0-UI3 决策表

每格先满足行内 W Gate，再追加列内 UI 证据：

| `gate_type` \ `ui_level` | UI0 | UI1 | UI2 | UI3 |
| --- | --- | --- | --- | --- |
| W0 | 结构/链接/完整性 | W0 + 浏览器/模拟器启动证据 | W0 + 微信工具/体验版证据 | W0 + 真机/真实平台证据 |
| W1 | 单元/边界/错误态 | W1 + 浏览器/模拟器场景 | W1 + 微信工具/体验版场景 | W1 + 真机/真实平台场景 |
| W2 | **硬阻断：W2 最低 UI1** | 契约/consumer/兼容与错误态 + UI1 | W2 + UI2 | W2 + UI3 |
| W3 | 并发/幂等/事务/恢复；无 UI 可 UI0 | W3 + UI1 | W3 + UI2 | W3 + UI3 |

## 4. 各类最低 Red → Green → Refactor

Red 必须是目标行为缺失造成的可观察失败；Green 和 Refactor 必须重跑同一检查。只改变断言、同时改坏 provider/consumer 或展示覆盖率数字，不构成 Green。

| 类型 | Red | Green | Refactor |
| --- | --- | --- | --- |
| W0 | 目标结构、链接、schema、内容或白名单检查真实失败 | 最小内容/结构修改使同一检查通过 | 重跑同一检查、链接/内容完整性、owned-path 与 diff |
| W1 | 最小单元、边界或错误路径因目标行为缺失失败 | focused test 与目标错误路径通过 | 重跑 focused test 和受影响回归；共享状态或并发代码加 race |
| W2 | provider/consumer 契约或 UI 主场景/错误态失败 | provider、全部 consumer、兼容/错误态和至少 UI1 通过 | 重跑同一契约、consumer 与 UI 回归 |
| W3 | 重复、并发、事务中断、恢复或非法状态暴露不变量失败 | 幂等、原子性、并发、恢复和失败语义通过 | 重跑同一故障/并发场景、race、真实存储或等价可执行证据，确认无重复副作用 |

### 分类命令或操作模板

尖括号参数必须替换为当前 change 的真实路径、包或测试名。对象或 runner 不存在时不得执行或记为 PASS。

| 类型 | 当前可执行模板 | 外部边界 |
| --- | --- | --- |
| W0 | `test -f <path>`；`rg -q '<required-token>' <path>`；`git diff --check <base-sha>...HEAD` | 需登录或外网才能检查的链接未实际访问时记未验证或 `BLOCKED_EXTERNAL` |
| W1 | `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/<package> -run '<TestName>' -count=1`；共享状态同路径加 `-race` | `<package>` 与 `<TestName>` 必须解析为仓库真实对象 |
| W2 | provider 运行真实 Go focused test；consumer 运行受影响测试和前端 static；另外记录实际 UI1 主场景与错误态 | 当前没有锁定的浏览器/微信 runner；缺少 UI1 资产即 `BLOCKED_EXTERNAL`，没有当前 PASS 命令 |
| W3 | `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/<package> -run '<IdempotencyOrConcurrencyTest>' -count=1`，再运行受影响包与 API smoke | 缺少隔离真实数据库、支付、退款或回调资产时 `BLOCKED_EXTERNAL`；纯 mock 不能冒充真实集成 PASS |

## 5. 统一证据模板

每个 task 完成后在 `tasks.md` 条目下记录以下字段；普通证据只保留首个决定性且已脱敏的错误或 PASS 摘要：

```yaml
change: <change-name>
gate_type: <W0|W1|W2|W3>
ui_level_target: <UI0|UI1|UI2|UI3>
ui_level_actual: <UI0|UI1|UI2|UI3|NOT_RUN>
base_sha: <full-sha>
candidate_sha: <full-sha-or-not-yet-created>
phase: <red|green|refactor|writer|verifier|integration>
command_or_action: <actual-command-or-runtime-steps>
exit_result: <exit-code|PASS|FAIL|BLOCKED_EXTERNAL>
sanitized_summary: <first-decisive-result>
artifact_or_environment: <artifact-path-or-controlled-environment>
unverified_boundary: <what-this-evidence-does-not-prove>
external_asset:
  owner: <responsible-role-or-N/A>
  missing: <missing-asset-or-N/A>
  recovery: <condition-to-rerun-or-N/A>
```

历史命令、计划命令、推断和另一 SHA 的结果不得填作当前 PASS。UI 或外部操作没有仓库命令时，记录实际步骤、版本和受控环境，不虚构 shell 命令。

## 6. Writer 永久 Gate

形成 CANDIDATE 前必须全部满足：

1. proposal/design/spec/tasks 完整一致，状态已经批准；`openspec validate <change-name> --strict` PASS。
2. 声明唯一最高 `gate_type`、目标/实际 `ui_level`、依赖、owned paths、外部资产与验收命令。
3. tasks 记录真实 Red、Green、Refactor；必要命令未运行即硬阻断。
4. diff 只含 owned paths；没有越权写入、推送、部署或外部系统变更。
5. 当前适用的 Go/static checks 与 change 声明的全部 Gate PASS。
6. 普通日志、diff 和证据满足敏感信息红线。
7. C/T/V/R 总分不低于 36、每项不低于 8，且硬阻断为零。
8. 只提交本 change；提交后记录完整 SHA，worktree 与 index clean。

### 当前仓库可直接运行

只按实际影响面选择；workflow/质量基建 change 为防治理回归应全部运行：

```bash
openspec validate <change-name> --strict
git diff --check <base-sha>...HEAD
git diff --name-only <base-sha>...HEAD
test -z "$(gofmt -l services/api)"
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...
GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
find apps -type f -name '*.js' -print0 | xargs -0 -n 1 node --check
node -e 'const fs=require("fs"),path=require("path");const walk=d=>fs.readdirSync(d,{withFileTypes:true}).flatMap(e=>{const p=path.join(d,e.name);return e.isDirectory()?walk(p):[p]});const files=[...walk("apps").filter(f=>f.endsWith(".json")),"project.config.json"].filter(fs.existsSync);for(const f of files)JSON.parse(fs.readFileSync(f,"utf8"));console.log(`JSON OK ${files.length}`)'
```

命令必须在当前 change 的证据中有真实退出结果。某命令因为当前 change 不涉及对应影响面而不适用时，记录 `N/A` 与设计理由；不得写 PASS。

## 7. 外部 Gate 与 `BLOCKED_EXTERNAL`

以下资产当前未建立；本文只登记启用条件，不提供假命令：

| Gate | 当前状态 | asset owner | 恢复并启用条件 |
| --- | --- | --- | --- |
| Web UI1 | 未建立 | 质量基建 change owner | 仓库引入并锁定浏览器 E2E runner、浏览器版本与可复现入口 |
| 小程序 UI1/UI2 | 未建立 | 开发方与客户小程序管理员 | runner/微信开发者工具、项目权限、账号和可复现入口具备 |
| UI3 | 未建立 | UAT owner 与客户平台管理员 | 指定真机、真实账号/平台、版本、受控验收权限和最终状态检查具备 |
| 真实数据库 W3 | 未建立 | 后端/平台 owner | 数据库引擎、migration runner、隔离测试库和清理策略由独立 change 确认 |
| 微信支付/退款 | 未建立 | 客户商户管理员与开发方 | 商户权限、密钥/证书、公网回调、受控资金范围、清理和对账方案获授权 |
| CI/监控 | 未建立 | 平台/运维 owner | CI 或可观测性后端、权限、配置和通知责任由独立 change 建立 |

缺失外部资产只在当前 change 必须使用它时成为 `BLOCKED_EXTERNAL`。记录 `owner`、`missing`、`recovery` 和已经完成的低层证据；不得把低层证据升级成 PASS，也不得让无依赖 change 被全局阻塞。

## 8. Independent verifier 与失效

仓库生命周期 verifier 只接收已提交的 CANDIDATE 完整 SHA，在另一 clean detached worktree 重跑全部声明 Gate。探索、DRAFT、APPROVED、未提交 diff、branch 名或 moving ref 不启动 verifier。

待发布公共契约或支付、真实数据、真实 UAT 等高风险结论还必须有第二份独立契约/运行证据；没有 candidate SHA 时，这类观察不能获得仓库 `INDEPENDENT_VERIFIED` 状态。

以下任一变化使旧验证失效：

- 实现、proposal、design、spec、tasks、验收命令或 candidate SHA；
- `base_sha`、依赖、rebase、merge 或当前 main 推进后需要更新 candidate；
- 与证据相关的数据库 schema、运行配置、微信 AppID/基础库/体验版、支付证书/回调、测试账号或真机条件。

verifier FAIL 必须返回原 writer；writer 修复并产生新 SHA 后，复用 verifier session 但为新 SHA 重建 clean detached worktree，从头重跑。session 复用、同一错误指纹第三次升级与停止条件只遵循 `loop-engineering-control-plane` 主 spec 和 `order-run-loop`，不得在 stage skill 中另建算法。

## 9. Integration Gate

integration 只接受：

- 所有声明依赖已经在当前 main 进入 `INTEGRATED`；依赖分支、candidate 或 main 外 PASS 不算满足；
- 当前 candidate exact SHA 获得未失效的 independent PASS；
- required review、writer Gate、C/T/V/R 和所有仓库检查满足，硬阻断为零；
- 集成动作已单独授权。

若 main 自验证后推进，原 writer 必须在原 change worktree 更新到最新 main、形成新 SHA，重跑 writer Gate 与独立验证。集成人不得在集成阶段修复后沿用旧 attestation。只有集成 main 后才能 archive。

## 10. C/T/V/R 评分与一票否决

评分只总结证据完整度，不替代任何必跑 Gate、业务不变量或运行验收。

| 维度 | 8 分最低合格 | 9 分 | 10 分 |
| --- | --- | --- | --- |
| C 契约正确性 | 风险/UI 分类、受影响契约/调用方、不变量和未验证边界完整 | 全部兼容/错误态和依赖边界可追溯 | 独立契约证据也完整，且无未解释差异 |
| T 测试证据 | 类别最低真实 RGR、失败路径和适用回归可复现 | 额外覆盖关键组合或故障路径 | 声明范围内全部关键组合、恢复/故障与运行证据闭环 |
| V 验证独立性 | 已提交 exact SHA、clean detached 验证包完整但结果待 verifier；只能形成 CANDIDATE | exact SHA independent PASS，且无额外外部独立证据要求 | independent PASS 与所有必要高风险外部独立证据完整 |
| R 可恢复性 | 回退/失败回流、验证失效和恢复条件明确可执行 | 已验证适用的回退或恢复检查 | 故障恢复、重复副作用与最终状态均有可复现证据 |

公式：`C + T + V + R >= 36` 且每项 `>= 8`。writer 阶段的 V 最高为 8，评分通过只表示可形成 CANDIDATE；integration 仍必须要求 exact-SHA independent PASS。

以下任一项一票否决，分数、覆盖率或其他 PASS 不得抵消：

1. 敏感信息泄漏；
2. 必要 Gate 未运行或把 `BLOCKED_EXTERNAL`/未验证写成 PASS；
3. candidate/verified SHA 不符或 worktree 不 clean；
4. P0 业务不变量失败；
5. diff 越过 owned paths；
6. 未经授权写入外部系统、推送或部署。

## 11. 敏感信息红线

普通日志、tasks、命令输出、trace artifact 和主动回传禁止记录：

- Authorization、Cookie、session/login code、私钥、证书、APIv3 key；
- 手机号、姓名、openid、工号、用户备注；
- 原始 query/body、支付或退款回调原文；
- 完整核销 token/二维码、完整外部交易号或其他可重放凭据。

允许保留服务端 request/trace/event ID、内部订单 ID、模板化 path、status、duration 和枚举化错误原因，但关联 ID 不得作为高基数指标 label。证据仅保留脱敏后的首个决定性错误、退出结果和受控环境标识。

发现泄漏立即 FAIL：先停止传播、清理证据面、由原 writer 修复并产生新 SHA，再从头验证；访问权限、事后承诺和评分均不能豁免。
