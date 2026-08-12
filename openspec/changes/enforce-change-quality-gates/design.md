## Context

仓库当前由根 `AGENTS.md` 固定七态治理，由 `$order-plan-change`、`$order-implement-tdd`、`$order-verify-change`、`$order-integrate-change` 分别处理单个 change 的四个阶段；已归档并集成的 `loop-engineering-control-plane` 负责跨 change 的 lane、调度、session、失败升级和主动回传。现有流程尚未回答一个 change 改了不同风险面时最低需要哪些测试与运行证据，导致静态检查、writer 自测、UI 模拟、真实平台结果可能被混写成同一个 PASS。

本 change 从本地 `main@69cc9b6437dc3181681603d1bb060c07acba97f1` 建立，只规划一个 change-local 质量协议。它与另一 worktree 的产品基线 writer 无依赖、无 owned-path 重叠；本 lane 不读取或回退对方未集成改动。详细门禁落在 `docs/quality/change-quality-gates.md`，四个 stage skill 只增加各自阶段必须读取和检查该协议的最小入口；`order-run-loop` 与根治理保持只读。

## Goals / Non-Goals

**Goals:**

- 用 W0-W3 的最高风险分类确定单个 change 的最低 Red/Green/Refactor 和 writer Gate。
- 用 UI0-UI3 明确 UI/微信证据实际运行到哪一层，未运行的层级不得写成 PASS。
- 固化 exact-SHA verifier 的触发、失效、失败回流和集成前提，同时保持与 `order-run-loop` 主 spec 一致。
- 用 C/T/V/R 四维评分表达证据完整度，但让硬阻断始终优先于分数。
- 给出当前仓库可直接执行的命令模板，以及依赖数据库、微信、支付、真机等外部资产后才启用的 Gate。
- 让四个 stage skill 继续分别承担规划、实现、验证、集成，不形成第五套 change 内工作流。

**Non-Goals:**

- 不修改 `.agents/skills/order-run-loop/**`、根 `AGENTS.md` 或其跨 change 调度规则。
- 不安装 Playwright、微信开发者工具、数据库、CI、监控或其他依赖。
- 不创建数据库、支付、微信、浏览器或真机测试，不连接外部环境。
- 不修改业务代码、产品/架构文档，不替产品基线 change 决定业务规则。
- 不用覆盖率阈值、测试数量或总分替代业务验收与硬 Gate。

## Decisions

### D1. 详细质量协议与四个 stage skill 各守单一职责

`docs/quality/change-quality-gates.md` 保存可复用的风险表、UI 表、证据模板、评分和敏感信息边界。四个 stage skill 只增加以下阶段检查：

| stage skill | 最小变化 |
| --- | --- |
| `order-plan-change` | 要求 proposal/design/spec/tasks 声明最高 `gate_type`、目标 `ui_level`、必要外部资产和验收命令 |
| `order-implement-tdd` | 按声明类别执行真实 Red/Green/Refactor、writer 永久 Gate，并记录标准证据模板 |
| `order-verify-change` | 对 exact candidate SHA 重跑已声明 Gate，拒绝把未运行或外部阻塞写成 PASS |
| `order-integrate-change` | 只接受依赖已在 main `INTEGRATED`、exact-SHA PASS 且验证未失效的 candidate |

`order-run-loop` 继续唯一负责 lane 数量、跨 change 依赖调度、错误指纹计数、第三次升级、session 复用和主动回传。质量协议只引用这些事实，不复制 ledger、scheduler 或 checkpoint 算法。

不把完整协议复制到四个 skill，因为复制会产生五份漂移文本；也不修改根 `AGENTS.md`，因为用户已固定本 change 的治理边界。

### D2. `gate_type` 只按最高风险向上取值

每个 change 在 DRAFT 声明唯一 `gate_type`：

| 类型 | 最高风险触发条件 | 最低关注点 |
| --- | --- | --- |
| W0 结构 | 只改变文档、链接、文件结构、非运行配置或内容完整性，不改变运行行为、公共契约或数据结果 | 结构、链接、内容完整性、白名单 |
| W1 内部逻辑 | 改变模块内部运行逻辑或边界，但不改变公共契约、持久化结果、资金、权限或并发不变量 | 单元行为、边界、错误路径、受影响回归 |
| W2 公共契约/UI | 改变公共 API/schema、任一调用方契约或用户可见 UI 行为 | provider、全部调用方、兼容/错误态、至少 UI1 |
| W3 数据/资金/并发 | 改变持久化数据、migration、权限、订单、支付、退款、库存、核销、幂等、事务、恢复或并发结果 | 并发、幂等、事务、恢复、故障证据与业务不变量 |

若一个 change 同时命中多个类型，取编号最高者；不按文件数、实现难度、平均风险或覆盖率降级。若实现发现更高风险面，先更新并重新批准 OpenSpec，再继续实现。

### D3. UI 证据独立分级，并与风险类型组成完整决策表

每个 change 同时声明目标 `ui_level`：

| 等级 | 可接受证据 | 明确不能证明 |
| --- | --- | --- |
| UI0 静态 | JS/JSON/模板/资源/页面结构检查 | 页面可交互、跨端状态或真实平台行为 |
| UI1 浏览器/模拟器 | 浏览器或非真实平台模拟器中运行主场景与错误态 | 微信体验版、真机原生能力、真实支付结果 |
| UI2 微信开发者工具/体验版 | 微信开发者工具真实编译/运行或指定体验版验证 | 真机差异、真实账号/支付/回调最终结果，除非实际覆盖 |
| UI3 真机/真实平台 | 指定真机、真实账号或真实平台受控验证，记录版本、环境和最终业务结果 | 未覆盖机型、账号、环境或全量生产正确性 |

W0-W3 × UI0-UI3 的唯一组合规则如下；“+ UIx”表示先满足行内风险 Gate，再追加列内运行证据：

| `gate_type` \ `ui_level` | UI0 | UI1 | UI2 | UI3 |
| --- | --- | --- | --- | --- |
| W0 | 结构/链接/完整性 | W0 + 浏览器/模拟器启动证据 | W0 + 微信工具/体验版证据 | W0 + 真机/真实平台证据 |
| W1 | 单元/边界/错误态 | W1 + 浏览器/模拟器场景 | W1 + 微信工具/体验版场景 | W1 + 真机/真实平台场景 |
| W2 | **硬阻断：W2 最低必须 UI1** | 契约/调用方/兼容与错误态 + UI1 | W2 + UI2 | W2 + UI3 |
| W3 | 并发/幂等/事务/恢复；无 UI 时可 UI0 | W3 + UI1 | W3 + UI2 | W3 + UI3 |

若小程序专属场景或所选 UI 等级依赖当前缺失的工具、体验版权限、真机、账号或平台，结果记为 `BLOCKED_EXTERNAL`，并记录 owner、缺失资产和恢复条件；不得用低一级证据替代或写成 PASS。

### D4. 每类最低 Red/Green/Refactor 只接受可观察证据

| 类型 | Red | Green | Refactor |
| --- | --- | --- | --- |
| W0 | 目标结构、链接、schema、内容完整性或白名单检查在改动前真实失败 | 最小文档/结构修改使同一检查通过 | 重跑同一检查、链接/内容完整性、owned-path 和 diff |
| W1 | 最小单元、边界或错误路径测试因目标行为缺失而失败 | focused test 和目标错误路径通过 | 重跑同一 focused test、受影响包回归；共享状态或并发代码加 race |
| W2 | provider/consumer contract 或 UI 主场景/错误态因契约缺失而失败 | provider、全部受影响调用方、兼容/错误态和至少 UI1 通过 | 重跑同一契约、调用方与 UI 回归；不得通过同时改坏两端断言消除 Red |
| W3 | 重复、并发、事务中断、恢复或非法状态场景暴露目标不变量失败 | 幂等、原子性、并发、恢复和失败语义通过 | 重跑同一故障/并发场景、race、真实存储或等价可执行证据，确认无重复副作用 |

证据模板固定为：`change`、`gate_type`、`ui_level`、`base_sha`、阶段、命令/操作、退出结果、首个决定性且已脱敏的错误或 PASS 摘要、artifact/运行环境、未验证边界。覆盖率只能作为诊断附件，不能作为任一字段的替代品。

详细协议必须至少给出下列分类命令模板。尖括号参数必须先替换为当前 change 的真实路径、包或测试名；命令不存在或所需资产未满足时，不得执行或记为 PASS：

| 类型 | 当前可执行命令模板 | 证据重点 | 外部边界 |
| --- | --- | --- | --- |
| W0 | `test -f <path>`；`rg -q '<required-token>' <path>`；`git diff --check <base-sha>...HEAD` | 同一结构/链接/内容检查的 Red、Green、Refactor 退出码与 changed paths | 链接目标需要登录或外网且未实际访问时记未验证或 `BLOCKED_EXTERNAL` |
| W1 | `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/<package> -run '<TestName>' -count=1`；共享状态再运行同路径 `go test -race` | focused 单元、边界、错误路径的首个失败与修复后同命令 PASS | `<package>`/`<TestName>` 必须解析为仓库真实对象；不存在的测试 runner 不得伪造 |
| W2 | provider 使用真实 Go focused test；Web/小程序调用方至少运行 `find apps -type f -name '*.js' -print0 \| xargs -0 -n 1 node --check`，并附实际 UI1 操作记录 | provider、全部受影响 consumer、兼容/错误态与至少 UI1 | 当前没有锁定的浏览器/微信 runner；专属运行态缺资产即 `BLOCKED_EXTERNAL`，没有当前 PASS 命令 |
| W3 | `GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/<package> -run '<IdempotencyOrConcurrencyTest>' -count=1`，再运行受影响包与 API smoke | 重复请求、并发交错、事务中断、恢复、非法状态、无重复副作用 | 真实数据库、支付、退款或平台回调缺隔离资产时 `BLOCKED_EXTERNAL`；纯 mock 不能冒充真实集成 PASS |

每条实际证据按单行头 `change=<name> gate_type=<Wn> ui_level=<UIn> base_sha=<full-sha> phase=<red|green|refactor|writer|verifier>` 开始，随后记录 `command_or_action`、`exit_result`、`sanitized_summary`、`artifact_or_environment`、`unverified_boundary`。W2/UI 与 W3 外部操作若没有仓库命令，记录实际人工步骤和受控环境，而不是虚构 shell 命令。

### D5. Writer 永久 Gate 由通用检查加变更类型检查组成

所有 writer 在形成 candidate 前必须满足：

1. OpenSpec strict PASS，且 proposal/design/spec/tasks 中声明的类型、UI 等级、依赖、owned paths 和验收一致。
2. tasks 记录真实 Red、Green、Refactor；没有执行的命令不得写 PASS。
3. diff 只含 owned paths；无敏感信息；工作树提交后 clean。
4. 当前仓库的 Go/static checks 按实际影响面执行，至少不低于 change 声明的命令。
5. 外部资产必需但缺失时返回 `BLOCKED_EXTERNAL`，不能产生 CANDIDATE PASS。

当前仓库已经可执行的命令登记为：

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
```

JSON 检查使用 Node 读取 `apps/**/*.json` 与 `project.config.json` 并逐个 `JSON.parse`；详细文档必须保存可直接复制的完整命令。Go 命令用于 Go/API 或 W3 并发影响面；前端 static 用于前端影响面；本 workflow change 的 apply writer 为防止治理回归，全部重跑。不存在的 Playwright、微信、数据库、CI 或监控命令只登记启用条件，不得伪装为当前可运行入口。

### D6. 独立验证只在有可验证对象时启动

生命周期中的 worktree verifier 只接受已提交的 `CANDIDATE` 完整 SHA，并在 clean detached worktree 重跑全部声明 Gate。探索、DRAFT、APPROVED、未提交 diff、branch 名或 moving ref 不启动 verifier。

待发布公共契约或支付、真实数据、真实 UAT 等高风险外部结论必须在 candidate acceptance 中声明第二份独立契约/运行证据；该观察可以由独立 reviewer 或外部验证者完成，但在没有 candidate SHA 时不得冒充仓库 `INDEPENDENT_VERIFIED` 状态，也不得额外复制 `order-run-loop` lane 规则。

verifier FAIL 后回原 writer 修复；修复生成新 SHA，旧 PASS/FAIL 对新候选均不可沿用。verifier session 复用、clean detached worktree 重建和同一错误指纹第三次升级继续以 `loop-engineering-control-plane` 主 spec 为权威，本协议只要求四个 stage skill 明确引用这一边界。

以下任一变化使旧验证立即失效：实现、spec、tasks、base、依赖、验收命令、rebase、merge 或 candidate SHA。运行环境、数据库 schema、微信 AppID/基础库、体验版、支付证书/回调、测试账号或真机条件变化时，对应 UI/外部证据也失效。

### D7. 集成只接受 main 中已满足的依赖和未失效 exact-SHA PASS

`order-integrate-change` 必须确认所有依赖已经在当前 main 进入 `INTEGRATED`，不能用依赖的 candidate、分支或 main 外 independent PASS 替代。若 main 自 candidate 验证后推进，writer 必须在原 change worktree 更新到新 main、形成新 SHA并重跑 writer 与独立验证；集成人不得在集成阶段修复后沿用旧 attestation。

### D8. C/T/V/R 评分表达完整度，硬阻断决定 verdict

| 维度 | 满分 | 10 分证据 |
| --- | ---: | --- |
| C 契约正确性 | 10 | 风险/UI 分类、公共契约、调用方、业务不变量和未验证边界准确完整 |
| T 测试证据 | 10 | 真实 Red/Green/Refactor、失败路径和类别最低命令全部可复现 |
| V 验证独立性 | 10 | writer 与 verifier 责任分离，exact SHA、clean detached 和外部独立证据完整 |
| R 可恢复性 | 10 | 回滚/恢复、事务或故障语义、失败回流和验证失效条件完整可执行 |

通过公式固定为：`C + T + V + R >= 36` 且每项 `>= 8`，同时所有硬阻断为零。以下任一项一票否决：敏感信息泄漏、必要 Gate 未运行、SHA 不符、P0 业务不变量失败、越过 owned paths，或未经授权写入/推送/部署。总分、覆盖率或其他通过项不得抵消一票否决。

### D9. 敏感信息不进入普通日志与验收证据

协议禁止在普通日志、tasks、命令输出、trace artifact 或回传中记录 Authorization、Cookie、session/login code、私钥、证书、APIv3 key、手机号、姓名、openid、工号、用户备注、原始 query/body、支付/退款回调原文、完整核销 token/二维码或完整外部交易号。允许使用服务端 request/trace/event ID、内部订单 ID、模板化 path、status、duration 和枚举化错误原因；这些关联 ID 不得成为高基数指标 label。

验收证据只保留脱敏后的首个决定性错误、命令、退出结果和受控环境标识。发现泄漏时立即 FAIL，先清理证据面并由 writer 产生新 SHA；不能靠日志访问权限或评分豁免。

### D10. 外部 Gate 只有资产满足后才启用

| Gate | 当前状态 | 启用条件 |
| --- | --- | --- |
| OpenSpec、Git、Go test/race/vet/build/smoke、Node JS/JSON static | 可执行 | 使用仓库当前固定命令和工具链 |
| Web UI1 | 未建立 | 仓库正式引入并锁定浏览器 E2E runner 与浏览器资产 |
| 小程序 UI1/UI2 | 未建立 | 微信相关 runner/开发者工具、项目权限和可复现入口具备 |
| UI3 | 未建立 | 指定真机、真实账号/平台、版本和受控验收权限具备 |
| 真实数据库 W3 | 未建立 | 数据库引擎、migration runner、隔离测试库和清理策略已由独立 change 确认 |
| 微信支付/退款 | 未建立 | 商户权限、密钥/证书、公网回调、受控资金范围和清理/对账方案均获授权 |
| CI/监控 | 未建立 | CI 平台或可观测性后端及其权限由独立 change 建立 |

未建立不代表失败；只有当前 change 的验收要求它时才转为 `BLOCKED_EXTERNAL`。协议不得给这些资产编造已可运行命令。

## Risks / Trade-offs

- [最高风险分类可能提高小改动成本] → 同一 change 已承担高风险结果时必须验证完整链路；可独立验收和回滚的低风险结果应拆成另一个 change，而不是降低分类。
- [UI 等级被当作产品质量总分] → UI 等级只声明实际运行边界，必须与 W 类 Gate 组合，不能替代契约、数据或资金验证。
- [详细文档与 stage skill 漂移] → 四个 skill 只引用一个协议路径并各自检查阶段输入/输出；apply 验收机器检查四个引用和关键 token。
- [外部 Gate 长期 BLOCKED] → 明确 owner、缺失资产和恢复条件；保持事实为未验证，但不阻塞无依赖、无路径冲突的其他 change。
- [新评分与主 Goal readiness 评分混淆] → C/T/V/R 只评价单个 change 的质量证据；`order-run-loop` 的 100 分 readiness rubric 和停止条件保持独立且权威。

## Migration Plan

1. 本轮仅创建 DRAFT proposal、design、spec、tasks，strict PASS 后提交规划 commit；所有 tasks 保持未勾选。
2. 主 Agent 明确批准后，原 writer 在同一 branch/worktree 先执行结构性 Red，证明质量文档和四个 skill 引用尚未满足协议。
3. 创建 `docs/quality/change-quality-gates.md`，再对四个 stage skill 做最小引用/检查修改，不触碰 `order-run-loop`、根治理或业务文件。
4. 重跑同一结构检查、协议 token/矩阵/命令模板检查、四 skill 引用、当前 Go/static regression、strict 和 owned-path audit，形成 candidate SHA。
5. 只有 candidate full SHA 产生后，另一 clean detached worktree 才执行 exact-SHA verifier；通过并获集成授权后再按既有 integration 流程推进。

回滚时整体撤销该 change 对质量文档和四个 stage skill 的修改；`order-run-loop`、根治理、业务行为、产品文档和外部环境均无需迁移。

## Open Questions

无。W/UI 分类、评分、硬阻断、外部资产边界和 owned paths 已由主 Goal 固定；实现前仍需主 Agent 对本 DRAFT 明确批准。
