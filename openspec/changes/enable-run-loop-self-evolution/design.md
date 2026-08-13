## Context

当前 `order-run-loop` 是 121 行薄控制面，稳定路由四个 change stage skills，并维护 lane、证据 Gate、失败熔断和 readiness 评分。它没有正式的模块版本冻结与经验晋升机制；同时 canonical `loop-engineering-control-plane` 仍规定 package 不含 references，必须通过本 delta 修改后才能合法承载详细协议。

本 change 从本地 clean `main@2209c071a21860231827b2a8c8c81d9b7745e6e1` 建立。模块基线已核验为 Skill git blob `d529461de5af1bf7cc65562e59ec3c84f0750963`、SHA256 `558b549a4410d72d4c22acad621ffae96af3aeccd26adc186ede76601097aa59`，legacy front matter 没有 version。规划和运行 checkpoint 放在 change 自有路径，canonical spec 只保存稳定规则。

## Goals / Non-Goals

**Goals:**

- 固定每个模块使用的 runner base，允许中断后仅从仓库、OpenSpec tasks 与精确 SHA 恢复。
- 在模块内只记录、分类 observation，以全部晋升 Gate 和独立 forward-test 控制规则变化。
- 新规则只对集成后开始的下一模块生效，不允许同模块热切换。
- 用可执行回归覆盖七项跨平台 checker 教训，同时保留现有 route、Gate 和 metadata。
- 以一个短 `SKILL.md` 加一个一层 reference 表达协议，不复制 stage skill。

**Non-Goals:**

- 不自动修改文件、自动提交或自动集成；“自进化”仍由独立 change、writer、verifier 与 integrator 执行。
- 不修改根 `AGENTS.md`、四个 stage skills、Go/小程序/业务代码、外部系统或当前模块运行规则。
- 不建立通用插件、数据库、后台服务、脚本运行时或重复 Skill。
- 不把环境、外部阻塞、主观总结或单次偶发现象直接晋升为规则。

## Decisions

### 1. 以不可变 runner identity 固定模块版本

模块开始时记录四元组：`repo_sha`、`git blob(Skill)`、`sha256(Skill)`、`frontmatter version or explicit unversioned marker`。本 change 不为了“看起来有版本”新增 version 字段；git blob 与内容哈希已经给出不可歧义版本，version marker 只表达 metadata 事实。

checkpoint 同时保存 lane/state、candidate/integrated/archive SHA、错误指纹次数、依赖、blocker、C/T/V/R 和 observation ledger。恢复时现场重算 identity 并与冻结值比较；不一致只形成 observation/blocker，不能改写当前 base。

不选择只记录 branch 或 `main`，因为它们是 moving ref；不选择依赖聊天 thread，因为聊天不是 repository evidence。

### 2. observation 保持 immutable，candidate 单独派生

四类含义固定：

| class | meaning | draft admission behavior |
| --- | --- | --- |
| `candidate` | 一个明确、可测试的 runner 规则提案 | 唯一可在模块边界接受 dedicated `DRAFT` 前置筛选的记录；进入 DRAFT 不等于晋升 |
| `environment` | 本机、工具版本、临时服务或网络状态 | 只记录环境与恢复条件；若形成通用规则则新建 candidate 引用它 |
| `checker` | 验收/checker 的正确性、可移植性或假绿问题 | 保留原始缺陷；新建 candidate 引用复现证据 |
| `external` | 真实资料、资质、密钥、账号、设备、不可逆授权 | 记录 owner/missing/recovery，不能直接晋升 |

原记录不改分类，避免通过重命名把环境噪声伪装成规则。本轮七项 checker 契约是用户已冻结的 change 输入，不是由当前模块自动晋升；实现后的新观察仍遵守上述流程。

### 3. DRAFT 前置筛选、晋升 Gate 与生效点严格分离

模块边界的 dedicated DRAFT admission 只做可实现性前置筛选：`reproducible evidence AND (generalizable rationale OR safety-critical rationale) AND documented non-weakening intent AND executable regression plan AND executable minimal-context fresh-session forward-test plan`。满足后只允许排队或创建一个遵守七态的 runner change `DRAFT`，不构成批准、实现、candidate、验证或规则晋升；缺项则继续留在 observation ledger。

实际 promotion Gate 固定为：`revalidated reproducible AND (generalizable OR safety-critical) AND non-weakening PASS AND implemented regression PASS AND clean-detached exact-SHA independent minimal-context fresh-session forward-test PASS AND full independent verification PASS`。任何缺项、FAIL 或 `BLOCKED_EXTERNAL` 都不得晋升；C/T/V/R 只总结质量，不能覆盖 Gate。writer 的 exact candidate 是该 Gate 的被测对象，不是晋升结论。

晋升后仍保持 inactive；只有 dedicated change 纯 fast-forward 集成本地 main 后，后续模块捕获的新 identity 才采用。已经开始的模块保持旧 base。本 Goal 明确只在 self-evolution 集成后让菜单目录模块采用新版本，菜单模块 retrospective 只可按 DRAFT 前置筛选排入下一 Goal，不在本 Goal 再改 Skill。

不选择在模块结束前直接 patch Skill，因为这会让同一次验收同时改变判卷规则；不选择 feature flag/双轨，因为会产生两个活跃规则源。

### 4. 生产规则只用一个一层 reference，验收工具留在 change

`SKILL.md` 只新增短入口：何时捕获 base、何时读 `references/self-evolution.md`、当前模块 observation-only、下一模块生效。详细分类、ledger 字段、晋升 Gate、checker 约束和 retrospective 模板放在唯一 `references/self-evolution.md`。`agents/openai.yaml` 保持当前字符串和 `$order-run-loop` default prompt，除非 metadata checker 发现真实不一致。

可执行验收脚本与 fixtures 放在 `openspec/changes/enable-run-loop-self-evolution/checks/`，不进入 Skill package；计划文件包括：

- `verify_contract.py`: 校验 package 文件集合/深度、frontmatter 与 `agents/openai.yaml` 一致、reference 链接、四 handler 路由、七态、lane/retry/external/scoring/hard-Gate 不漂移，并拒绝复制四个 stage skill 完整标题。
- `checker_contract.py`: 只用 Python 标准库实现 fixture 需要的安全解析、文字匹配、bounded poll、临时路径验证和 archive 行终止处理，不调用 shell 解释 fixture 数据。
- `run_checker_regressions.py`: 对七项正反例执行回归；任何 contract violation 非零退出，禁止 `|| true`。
- `forward-test.md` 与 `verify_forward_test.py`: 固定最小上下文 prompt、结构化输出 schema 和 PASS/FAIL validator。

这些脚本是本 change 的验收 surface，不成为 runner 的运行依赖。archive 后它们随 change artifact 保留为可复现证据。

### 5. fresh-session forward-test 属于 exact-SHA verifier Gate

writer 先完成本地 contract/checker 自测、strict、quick_validate、owned-path audit 和候选 commit；writer 的 V 最高为 8。独立 verifier 在 candidate 的 clean detached worktree 启动 fresh session，仅让它读取：

1. root `AGENTS.md`；
2. `.agents/skills/order-run-loop/SKILL.md`；
3. `.agents/skills/order-run-loop/references/self-evolution.md`；
4. change 内 `forward-test.md` 的最小场景。

场景要求结构化输出：冻结 base、对 `candidate`/`checker`/`environment`/`external` 正确分类、同模块拒绝改 Skill、缺前置材料的 candidate 留在 observation、前置筛选合格者只标为 dedicated-DRAFT-admissible（明确不是晋升）、已集成本地 main 的规则仅对 next module 生效，以及 `DRAFT`/`APPROVED`/`CANDIDATE`/`INDEPENDENT_VERIFIED` 的既有 handler 映射。verifier 使用安全临时目录承接结果并运行 validator；不修改 repo。fresh session 或 exact-SHA 任一失败即 verifier FAIL。

不把 writer 自述当独立 forward-test；也不在 candidate 前把 forward-test attestation 写回 tasks，因为任何 metadata commit 都会改变 SHA 并使证据失效。独立结果由 verifier 的外部 attestation 回传，candidate 文件保持不动。

### 6. 七项 checker 约束用正反 fixture 固定

- 零匹配：允许零时返回 `0`；仅 no-match 不失败，工具/解析错误仍失败。
- Markdown：状态字段按 fenced-code-aware 的精确 key 行解析；缺失/重复失败，不用松散 substring。
- 反引号：fixture 数据只经 argv/stdin；禁止 `shell=True`、反引号或 `$()` transport。
- `awk` 大小写：contract 明示字段是否 case-sensitive；不区分时显式 normalize/ignore-case。
- 健康恢复：attempt 与 deadline 都有上限；注入 probe/clock 以无真实 sleep 测 ready 与 never-ready。
- 临时目录：标准库创建窄目录，realpath 后验证 parent/type/link/count；清理失败停止，不扩大目标。
- archive 尾随换行：只移除一个 LF 或 CRLF record terminator；保留其他字符并做 exact compare。

### 7. Writer、verifier、integration 证据分离

本 change 为 `W1/UI0`。评分定义和证据边界固定如下；每分必须绑定实际 artifact/命令，任何未达到的分数不得预填 PASS：

| dimension | definition | candidate target and evidence |
| --- | --- | --- |
| C | 契约正确性 | `9`：delta spec、metadata、legacy route 和 non-weakening contract 全部一致 |
| T | 测试证据 | `10`：真实 Red/Green/Refactor、七项正反例和关键状态组合均可复现 |
| V | 验证独立性 | `8`：精确 candidate 与 clean detached 验证包齐全，独立结果仍待 verifier；writer 不得声称更高 |
| R | 可恢复性 | `9`：冻结身份、拒绝晋升、bounded recovery、next-module 队列和 rollback 可执行 |

writer 目标总分 36；所有维度必须至少 8、总分至少 36 且硬阻断为 0。independent PASS 后 V 至少 9，但任何分数都不能替代 mandatory Gate。

candidate 的 owned-path audit 绝对只接受：

- `openspec/changes/enable-run-loop-self-evolution/**`
- `.agents/skills/order-run-loop/**`

exact-SHA verifier 只读。集成只允许 main 是 candidate ancestor 时执行本地 pure fast-forward；main 推进需回原 writer 更新、新 SHA 和全量重验。集成后 `openspec archive` 产生的 canonical delta 合并与 archive move 是 `$order-integrate-change` 的确定性 lifecycle output，不属于 writer candidate diff；integrator 必须单独确认只出现预期 `openspec/specs/loop-engineering-control-plane/spec.md` 与日期化 archive 路径，不得借 archive 扩大实现范围。不 push、PR、deploy 或外部写。

## Risks / Trade-offs

- [Markdown 协议可能比程序化状态弱] → 固定字段、唯一 checkpoint、contract parser 与 strict/forward-test 共同验收；不引入额外数据库。
- [Fresh agent 输出有非确定性] → 最小上下文、结构化 schema、明确正反场景和机器 validator；失败即不晋升。
- [Reference 增加上下文跳转] → `SKILL.md` 只在模块开始/retrospective 需要时指向唯一文件，保持一层且不复制 stage skill。
- [Archive 必然写 canonical/archive 路径] → writer candidate 仍严格限定 owned paths；只有授权 integrator 在集成后运行确定性 archive 并单独审计生成路径。
- [Main 前进使 pure FF 失败] → 不 merge 绕过；返回 writer 基于新 main 形成新 candidate 并重跑 exact-SHA 验证。
- [回滚后已有模块使用新版] → 活跃模块 base 仍不可变；回滚只影响回滚后开始的模块，避免模块内热切换。

## Migration Plan

1. 保持 `DRAFT`：完成 proposal/design/delta spec/tasks、checkpoint 与 strict；主 Agent明确批准前不修改 Skill。
2. `APPROVED` 后由同一 writer 读取 apply context，先建立能证明缺失行为的 contract/checker Red，再最小修改 `SKILL.md`、新增唯一 reference 与 change-local checks。
3. 运行 Green/Refactor、strict、quick_validate、metadata/route/non-weakening/checker regressions、owned-path audit；评分达标且硬阻断 0 后只提交 owned paths，形成 exact candidate。
4. 独立 verifier 在新 clean detached worktree 重跑全部 Gate，并执行最小上下文 fresh-session forward-test；任何失败返回原 writer，新 SHA 从头验证。
5. 授权 integrator确认 main 未漂移后用 pure fast-forward 本地集成，验证 integrated main；再运行 OpenSpec archive、审计唯一 canonical/archive 产物并提交归档结果。
6. 下一模块开始时捕获新的 runner identity；回滚则撤销 Skill/reference 变更及 canonical delta，且只对回滚后开始的模块生效。

## Open Questions

无。模块边界、四类 observation、晋升公式、生效点、七项 checker、fresh-session 范围、owned paths 和外部权限均已由 Goal 冻结。
