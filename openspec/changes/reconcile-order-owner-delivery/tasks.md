# Tasks

- [x] 1. 读取最新交接与实际版本，确认writer ownership；bd29721干净，执行者释放3docs，云/IDE/PC仍归执行者。
- [x] 2. Red：2026-10-12运行仓库外check_docs.py，exit1，两个新入口missing。
- [x] 3. Green：写入验收页和生产Runbook；同check_docs.py exit0，必需内容与本地链接PASS。
- [x] 4. Refactor：无需额外结构重构；同内容/链接检查与diff检查重跑PASS，不扩运行范围。
- [x] 5. Writer：openspec validate reconcile-order-owner-delivery --strict、owned-path/diff、原生Codex对抗审查；修复范围内问题。
- [x] 6. Candidate准备：owned-path和干净index检查完成；后续中文本地提交的完整SHA与完成状态以仓库外candidate-receipt.json登记，避免自引用SHA。
- [ ] 7. Independent：另一个clean detached worktree对exact候选只读重跑W0门并核实证据边界；提交后结果写仓库外independent-receipt.json，不改已验证候选。
- [ ] 8. Integration：无main/push/PR授权，本轮NOT_RUN，未archive。

命令入口：python3 /Users/vivix/.codex/visualizations/2026/10/12/order-owner-acceptance/check_docs.py <worktree>。外部receipt根同目录；副作用为本地文档和隔离临时资产，Prod仅只读元数据。新候选使旧独验失效。

Writer C/T/V/R=10/9/8/9，总36：契约/版本/外部停点具备交叉证据，W0内容链接/严格spec/diff通过，独验待候选，文档回退不影响运行；实际独验结果另记。Go测试、race、build、UI gate为N/A：本change只改文档。完整UAT/Prod外部缺口不是本W0的虚假PASS。

原生审查首轮两项：P1补写最新人类授权原话及AGENTS后续指示例外，不改变跨任务规则；P2将架构spec同步明确为生产适配先决条件，本W0只记录后续方向，不覆盖现行spec。最终原生复审无待修复项，日志native-review-r2.log；内容与源码/引用收据一致。
