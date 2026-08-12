> 状态：`CANDIDATE`。主 Agent 已明确批准；writer 已完成 W0/UI0 Red → Green → Refactor，等待 exact-SHA independent verification。以下已勾选项共享证据上下文：`change=establish-production-architecture-baseline`、`gate_type=W0`、`ui_level_target=UI0`、`ui_level_actual=UI0`、`base_sha=cb2605f477e58ac5471a0c535b85256c6be80a00`、`candidate_sha=SELF（由本地提交生成并在 handoff 绑定精确 SHA）`。

## 1. Approval, Ownership and Red

- [x] 1.1 取得用户对本 change 的明确批准，将状态从 `DRAFT` 更新为 `APPROVED`；确认唯一 writer 仍位于 branch `codex/establish-production-architecture-baseline` 的既有 worktree，基线为 `cb2605f477e58ac5471a0c535b85256c6be80a00`，目标文档不存在并行 writer 或未吸收变化。
  - Evidence: `phase=Approval`; `command=git branch --show-current; git rev-parse HEAD; git merge-base --is-ancestor <base> HEAD; git worktree list --porcelain; git status --short --branch`; `exit_result=0`; `sanitized_summary=主 Agent 明确批准，writer branch 为 codex/establish-production-architecture-baseline，规划 HEAD 8d1a69360b89a7268b9313d7aaa71e530ab7dcc4 包含 base 且起始 worktree clean，其他 worktree 未占用 owned branch`; `artifact=proposal/tasks`; `unverified_boundary=未做独立验证`; `external_asset=无`。
- [x] 1.2 完整重读 proposal、spec、design、tasks、根 `AGENTS.md` 与 `docs/quality/change-quality-gates.md`，运行 `openspec validate establish-production-architecture-baseline --strict`；确认 `gate_type=W0`、`ui_level_target=UI0`、外部链接 owner/恢复条件和 `Open Questions=无` 后才能进入 `IMPLEMENTING`。
  - Evidence: `phase=Approval`; `command=openspec instructions apply --change establish-production-architecture-baseline --json; openspec validate establish-production-architecture-baseline --strict; openspec status --change establish-production-architecture-baseline --json`; `exit_result=0`; `sanitized_summary=apply 返回四类 context 与 21 tasks，所列 context、根治理和质量协议已完整重读，strict valid，四类 artifact done，Open Questions=无`; `artifact=proposal/design/spec/tasks`; `unverified_boundary=writer V 上限 8`; `external_asset=public internet owner Production Architecture Writer，不可达时 BLOCKED_EXTERNAL`。
- [x] 1.3 在修改两个实际文档前运行以下内容检查并记录 Red；失败必须来自当前文档缺少唯一拓扑、事务/worker、迁移、隔离、恢复或容量规则，以及仍含二选一、公有读、长期 key 等旧口径，不得来自脚本或环境错误：
  - Evidence: `phase=Red`; `command=下列冻结内容检查`; `exit_result=1（expected Red）`; `sanitized_summary=指纹 ARCH_BASELINE_RED；缺少 Nginx/systemd、inbox/outbox、migration、恢复与容量术语，同时命中数据库多选、匿名读、长期 key、默认加速和角色未决旧口径；脚本与文件读取正常`; `artifact=两个实际文档的 base 内容`; `unverified_boundary=尚未实现 Green`; `external_asset=无`。

  ```bash
  python3 - <<'PY'
  from pathlib import Path

  paths = [
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  text = "\n".join(path.read_text(encoding="utf-8") for path in paths)
  required = [
      "Nginx", "systemd", "127.0.0.1:8080", "order-api", "同进程 worker",
      "MySQL 8.0", "双节点", "多可用区", "私有 COS", "SSM", "CAM Role",
      "inbox_events", "outbox_events", "SKIP LOCKED", "at-least-once",
      "第 10 次", "DEAD", "5 秒", "order-migrate", "forward-only",
      "schema_migrations", "GET_LOCK", "/health/ready", "不自动 down",
      "预签名 PUT", "预签名 GET", "30 天", "每日全量备份", "binlog",
      "14 天", "销毁", "7 天", "季度", "RPO ≤5 分钟",
      "RTO ≤5 分钟", "RTO ≤60 分钟", "RTO ≤2 小时", "区域级灾难",
      "CLS", "云监控", "CAT", "200 并发", "300 单/5 分钟", "未实测",
      "SQL", "索引", "纵向升级", "不交付长期 SecretId/SecretKey",
  ]
  forbidden = [
      "MySQL / TDSQL-C", "MySQL 还是 TDSQL-C", "公有读私有写",
      "COS 密钥（SecretId / SecretKey）配置", "配 CDN 加速",
      "实际角色需要和客户确认", "二选一", "按需选择", "视情况选择",
  ]
  missing = [value for value in required if value not in text]
  stale = [value for value in forbidden if value in text]
  if missing or stale:
      raise SystemExit(f"ARCH_BASELINE_RED missing={missing}; stale={stale}")
  print("architecture baseline content PASS")
  PY
  ```

- [x] 1.4 逐项对照归档 `mvp-product-baseline`、`bootstrap-api-service` 与当前两个文档，记录现有抽象部署形态、推荐状态/角色、数据库二选一、公有读 COS、长期访问 key、CDN 和未量化恢复/容量口径的位置；库存三维键与 15 分钟软预占只作为已确认依赖记录，不新增或改变其业务规则。
  - Evidence: `phase=Red`; `command=rg -n <产品与旧架构术语> openspec/specs/mvp-product-baseline/spec.md <两个实际文档> services/api`; `exit_result=0`; `sanitized_summary=定位技术文档抽象 API/数据库/定时任务及旧角色，指南定位旧数据库、COS、访问 key、加速和费用口径；归档库存键与 15 分钟软预占只读`; `artifact=mvp spec/bootstrap baseline/两个实际文档`; `unverified_boundary=不实现库存`; `external_asset=无`。

## 2. Green: Single Architecture Documentation

- [x] 2.1 只修改 `docs/product/online-ordering-system-technical.md`，按 design D1–D9 写入组件图、同步/异步数据流、MySQL 事实源、inbox/outbox/worker、迁移、环境隔离、配置/密钥、私有 COS、恢复目标、观测阈值、容量门和升级顺序；明确 RPO/RTO 与容量未实测，不把目标写成已达成。
  - Evidence: `phase=Green`; `command=apply_patch; rg -n <D1-D9 headings/terms> docs/product/online-ordering-system-technical.md`; `exit_result=0`; `sanitized_summary=开发者视图已固定组件、同步/异步事务、migration、隔离、安全、恢复、观测和证据驱动升级，RPO/RTO 与容量均标记未实测`; `artifact=docs/product/online-ordering-system-technical.md`; `unverified_boundary=未部署/未压测/未恢复演练`; `external_asset=真实云资源不需要`。
- [x] 2.2 只修改 `docs/微信小程序开发和运维指南/腾讯云操作指南.md`，把客户采购与责任口径固定为 CVM、TencentDB MySQL 8.0 双节点多可用区、私有 COS、SSM/CAM、CLS/云监控/CAT；删除数据库二选一、公有读 COS、长期 SecretId/SecretKey 交付和默认 CDN，只保留 spec 白名单外部占位符及官方直链。
  - Evidence: `phase=Green`; `command=apply_patch; rg -n <采购/责任 headings/terms> docs/微信小程序开发和运维指南/腾讯云操作指南.md`; `exit_result=0`; `sanitized_summary=客户视图已固定单 CVM、CDB MySQL 8.0 双节点多可用区、私有 COS、SSM/CAM、CLS/云监控/CAT，并声明不交付长期访问凭据与无实时价格承诺`; `artifact=腾讯云操作指南.md`; `unverified_boundary=白名单外部值未提供`; `external_asset=甲方后续提供账号/规格/域名/微信身份/预算`。
- [x] 2.3 重跑 1.3 的同一内容检查并记录 Green；必须输出 `architecture baseline content PASS`。逐行确认两个实际文档与 spec/design 使用同一组件、时限、保留期、阈值、非目标和升级条件。
  - Evidence: `phase=Green`; `command=重跑 1.3 冻结内容检查`; `exit_result=0`; `sanitized_summary=architecture baseline content PASS；必备组件、5 秒、10 次、14/7/30 天、RPO/RTO、告警阈值、容量门、非目标与升级顺序一致`; `artifact=两个实际文档`; `unverified_boundary=文档契约通过，不代表运行能力已实现`; `external_asset=无`。

## 3. Refactor: Consistency, Links and Safety

- [x] 3.1 运行以下歧义与外部占位符检查；两个实际文档合计必须只含完整白名单，不得残留普通占位、行为未决或旧选型口径：
  - Evidence: `phase=Refactor`; `command=下列术语与占位符检查`; `exit_result=0`; `sanitized_summary=architecture terminology and placeholder check PASS；17 个命名外部值完整且无未知值/行为未决/旧选型口径`; `artifact=两个实际文档`; `unverified_boundary=占位符真实值尚未提供`; `external_asset=由甲方与后续平台 owner 补齐`。
  - Verifier Red: `phase=verifier`; `candidate_sha=b084f7b50561f44bb40001f2157c35145de3d598`; `exit_result=FAIL`; `sanitized_summary=ARCH_BEHAVIOR_TODO_OUTSIDE_WHITELIST（第 1 次）：技术文档第 163 行把部门/工号作为客户条件启用字段，且用户表仍含 department/employee_no，超出 17 个命名外部值白名单；旧 SHA 证据失效`; `artifact=docs/product/online-ordering-system-technical.md`; `unverified_boundary=其余 verifier Gate PASS`; `external_asset=无`。
  - Writer Red: `phase=red`; `command=behavior-decision wording check`; `exit_result=1`; `sanitized_summary=同指纹 ARCH_BEHAVIOR_TODO_OUTSIDE_WHITELIST，唯一命中原第 163 行`; `artifact=两个实际文档`; `unverified_boundary=修复前证据`; `external_asset=无`。
  - Writer Green: `phase=green`; `command=behavior-decision wording check; ! rg -n 'department|employee_no|部门|工号' docs/product/online-ordering-system-technical.md`; `exit_result=0`; `sanitized_summary=删除一期正式用户说明与用户表中的部门/工号三行，自然语言行为未决与目标字段均无命中；未扩张白名单或产品行为`; `artifact=docs/product/online-ordering-system-technical.md`; `unverified_boundary=新 candidate 仍待独立验证`; `external_asset=无`。

  ```bash
  python3 - <<'PY'
  import re
  from pathlib import Path

  paths = [
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  text = "\n".join(path.read_text(encoding="utf-8") for path in paths)
  allowed = {
      "TODO_CLOUD_ACCOUNT_ID", "TODO_TENCENT_REGION", "TODO_VPC_ID",
      "TODO_UAT_CVM_INSTANCE_TYPE", "TODO_PROD_CVM_INSTANCE_TYPE",
      "TODO_UAT_CDB_INSTANCE_TYPE", "TODO_PROD_CDB_INSTANCE_TYPE",
      "TODO_UAT_DOMAIN", "TODO_PROD_DOMAIN", "TODO_UAT_COS_BUCKET",
      "TODO_PROD_COS_BUCKET", "TODO_CERTIFICATE_ID",
      "TODO_UAT_WECHAT_APPID", "TODO_PROD_WECHAT_APPID", "TODO_WECHAT_MCH_ID",
      "TODO_ALERT_RECIPIENTS", "TODO_MONTHLY_BUDGET_CNY",
  }
  found = set(re.findall(r"TODO_[A-Z0-9_]+", text))
  if found != allowed:
      raise SystemExit(f"placeholder mismatch missing={sorted(allowed-found)} unknown={sorted(found-allowed)}")
  forbidden = [
      r"\bTODO\b", r"\bTBD\b", r"待定", r"二选一", r"按需选择", r"视情况选择",
      r"待[^。\n]{0,12}确认", r"确认后启用", r"启用后", r"视客户", r"由客户确认", r"客户[^。\n]{0,8}确认后",
      r"MySQL\s*/\s*TDSQL-C", r"MySQL 还是 TDSQL-C", r"公有读私有写",
      r"COS 密钥（SecretId\s*/\s*SecretKey）配置", r"配 CDN 加速",
      r"实际角色需要和客户确认",
  ]
  hit = [pattern for pattern in forbidden if re.search(pattern, text)]
  if hit:
      raise SystemExit(f"ambiguous architecture remains: {hit}")
  print("architecture terminology and placeholder check PASS")
  PY
  ```

- [x] 3.2 检查 Markdown 本地链接存在、所有架构外链为腾讯云或微信支付官方 HTTPS 直链，并实际访问每个去重 URL；public internet 是必要外部资产，owner=`Production Architecture Writer`。访问必须使用单链接 20 秒、全组 90 秒、零重试的有界检查；任一页面不可访问时记录 `BLOCKED_EXTERNAL`、访问日期与恢复条件，不得记链接 PASS，也不得使已独立通过的本地内容 Gate 失效：
  - Evidence: `phase=Refactor`; `command=下列本地链接/官方 host 检查；有界逐 URL curl`; `exit_result=0`; `sanitized_summary=17 个去重官方 HTTPS URL 均一次访问成功，本地相对链接均存在；per_link_timeout=20s、total_timeout=90s、retries=0、access_date=2026-08-13`; `artifact=design 与两个实际文档`; `unverified_boundary=官方页面未来可变，本地内容 Gate 独立`; `external_asset=public internet/Production Architecture Writer/当前可用；不可达时记 URL/日期/恢复条件并在恢复后单次重跑`。

  ```bash
  python3 - <<'PY'
  import re
  from pathlib import Path
  from urllib.parse import urlparse

  paths = [
      Path("openspec/changes/establish-production-architecture-baseline/design.md"),
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  allowed_hosts = {
      "cloud.tencent.com", "cloud.tencent.com.cn", "pay.wechatpay.cn",
      "console.cloud.tencent.com", "buy.cloud.tencent.com",
  }
  urls = set()
  for path in paths:
      text = path.read_text(encoding="utf-8")
      for target in re.findall(r"\[[^]]*\]\(([^)]+)\)", text):
          if target.startswith("https://"):
              urls.add(target)
          elif "://" in target:
              raise SystemExit(f"non-HTTPS link: {path}:{target}")
          else:
              local = (path.parent / target.split("#", 1)[0]).resolve()
              if not local.exists():
                  raise SystemExit(f"missing local link: {path}:{target}")
      urls.update(re.findall(r"https://[^\s，、）)>]+", text))
  for url in sorted(urls):
      if urlparse(url).hostname not in allowed_hosts:
          raise SystemExit(f"non-official architecture URL: {url}")
      print(url)
  PY

  python3 - <<'PY'
  import re
  import subprocess
  import time
  from pathlib import Path
  paths = [
      Path("openspec/changes/establish-production-architecture-baseline/design.md"),
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  urls = set()
  for path in paths:
      urls.update(re.findall(r"https://[^\s，、）)>]+", path.read_text(encoding="utf-8")))
  deadline = time.monotonic() + 90
  for url in sorted(urls):
      remaining = deadline - time.monotonic()
      if remaining <= 0:
          raise SystemExit(
              "BLOCKED_EXTERNAL total_timeout=90s access_date=2026-08-13 "
              "recovery=public internet or official page recovers, then rerun once"
          )
      try:
          result = subprocess.run(
              ["curl", "--location", "--fail", "--silent", "--show-error",
               "--max-time", "20", "--output", "/dev/null", url],
              timeout=min(21, remaining), check=False, capture_output=True, text=True,
          )
      except subprocess.TimeoutExpired:
          raise SystemExit(
              f"BLOCKED_EXTERNAL url={url} per_link_timeout=20s "
              "access_date=2026-08-13 recovery=official page recovers, then rerun once"
          )
      if result.returncode:
          detail = result.stderr.strip().splitlines()[-1] if result.stderr.strip() else f"curl_exit={result.returncode}"
          raise SystemExit(
              f"BLOCKED_EXTERNAL url={url} detail={detail} "
              "access_date=2026-08-13 recovery=official page recovers, then rerun once"
          )
  print(
      f"official link access PASS urls={len(urls)} per_link_timeout=20s "
      "total_timeout=90s retries=0 access_date=2026-08-13"
  )
  PY
  ```

- [x] 3.3 运行敏感信息静态检查；允许回环地址与命名占位符，禁止真实凭据、个人数据、生产资源标识或可重放正文：
  - Evidence: `phase=Refactor`; `command=下列 architecture sensitive-data check`; `exit_result=0`; `sanitized_summary=architecture sensitive-data check PASS；无私钥头、腾讯云 SecretId、手机号、真实 AppID 或非回环 IPv4`; `artifact=全部 change artifacts 与两个实际文档`; `unverified_boundary=静态模式不能替代真实平台权限审计`; `external_asset=无`。

  ```bash
  python3 - <<'PY'
  import re
  from pathlib import Path

  paths = [
      Path("openspec/changes/establish-production-architecture-baseline/proposal.md"),
      Path("openspec/changes/establish-production-architecture-baseline/design.md"),
      Path("openspec/changes/establish-production-architecture-baseline/specs/production-architecture-baseline/spec.md"),
      Path("openspec/changes/establish-production-architecture-baseline/tasks.md"),
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  text = "\n".join(path.read_text(encoding="utf-8") for path in paths).replace("127.0.0.1", "")
  forbidden = {
      "private_key": r"-----BEGIN [A-Z ]*PRIVATE KEY-----",
      "tencent_secret_id": r"AKID[A-Za-z0-9]{13,}",
      "mainland_phone": r"(?<!\d)1[3-9]\d{9}(?!\d)",
      "wechat_appid_value": r"(?<!TODO_)wx[0-9A-Za-z]{16}",
      "ipv4": r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])",
  }
  hit = [name for name, pattern in forbidden.items() if re.search(pattern, text)]
  if hit:
      raise SystemExit(f"sensitive content detected: {hit}")
  print("architecture sensitive-data check PASS")
  PY
  ```

- [x] 3.4 对照 `mvp-product-baseline` 确认技术文档只引用而不改变`营业日期 × 餐段 × 商品`、15 分钟软预占、九态和四角色；确认 RPO/RTO、容量、云 SKU 和成本均明确区分目标、未实测与外部值。
  - Evidence: `phase=Refactor`; `command=git diff --exit-code <base> -- mvp spec PRD; rg -nF <库存键/15 分钟/九态/四角色>; rg -n <未实测/外部值/RPO/RTO/容量/SKU/成本> <两个实际文档>`; `exit_result=0`; `sanitized_summary=PRD 与 mvp spec 无 diff，技术文档精确引用归档库存、九态、四角色；恢复/容量是未实测目标，SKU/成本是外部值`; `artifact=mvp spec/PRD/两个实际文档`; `unverified_boundary=不实现产品行为或云容量`; `external_asset=无`。

## 4. Local Writer Gate

- [x] 4.1 运行 `openspec validate establish-production-architecture-baseline --strict` 与 `openspec status --change establish-production-architecture-baseline --json`，确认四类 artifacts 完整、tasks 可 apply，且状态仍按实际阶段记录。
  - Evidence: `phase=Writer Gate`; `command=openspec validate establish-production-architecture-baseline --strict; openspec status --change establish-production-architecture-baseline --json`; `exit_result=0`; `sanitized_summary=strict valid，proposal/design/specs/tasks 四类 artifact 均 done，状态记录为 CANDIDATE`; `artifact=change directory`; `unverified_boundary=independent verifier 尚未运行`; `external_asset=无`。
- [x] 4.2 检查 owned Markdown 文件标题、围栏、表格和相对链接结构完整；执行 `git diff --check cb2605f477e58ac5471a0c535b85256c6be80a00`。
  - Evidence: `phase=Writer Gate`; `command=owned markdown structure check; git diff --check <base>`; `exit_result=0`; `sanitized_summary=标题/围栏/表格结构完整，相对链接存在，无 whitespace error`; `artifact=全部 owned Markdown`; `unverified_boundary=未做视觉 UI 验收，UI0 不适用`; `external_asset=无`。
- [x] 4.3 运行 owned-path Gate；只允许本 change 目录和两个获批实际文档发生变化，任何 PRD、客户清单、合同、质量/loop skill 或业务代码 diff 都立即 FAIL：
  - Evidence: `phase=Writer Gate`; `command=下列 unexpected_paths Gate`; `exit_result=0`; `sanitized_summary=相对 base 仅 change 目录与两个获批文档有 diff；PRD、客户资料/合同、quality/loop skills 和业务代码均无 diff`; `artifact=git diff name set`; `unverified_boundary=并行 worker 互斥路径不在本 candidate`; `external_asset=无`。

  ```bash
  unexpected_paths="$({
    git diff --name-only cb2605f477e58ac5471a0c535b85256c6be80a00
    git diff --cached --name-only
  } | sort -u | grep -Ev '^(openspec/changes/establish-production-architecture-baseline/|docs/product/online-ordering-system-technical\.md$|docs/微信小程序开发和运维指南/腾讯云操作指南\.md$)' || true)"
  test -z "$unexpected_paths"
  ```

- [x] 4.4 运行当前仓库 Gate；本 change 虽为 W0，也必须证明规划的文档修改没有伴随代码、前端或基础进程回归：
  - Evidence: `phase=Writer Gate`; `command=gofmt check; Go test/race/vet/build/smoke; JS node --check; JSON parse`; `exit_result=0`; `sanitized_summary=Go test 与 race 全部 PASS，vet/build exit 0，smoke PASS，JS 语法 PASS，JSON OK 42`; `artifact=当前仓库基线`; `unverified_boundary=本 change 未修改运行代码，未执行真实云/UAT`; `external_asset=GOTOOLCHAIN go1.26.5 本地可用`。

  ```bash
  test -z "$(gofmt -l services/api)"
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test ./services/api/...
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go test -race ./services/api/...
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go vet ./services/api/...
  GOPROXY=off GOTOOLCHAIN=go1.26.5 go build ./services/api/...
  GOPROXY=off GOTOOLCHAIN=go1.26.5 bash services/api/scripts/smoke.sh
  find apps -type f -name '*.js' -print0 | xargs -0 -n 1 node --check
  node -e 'const fs=require("fs"),path=require("path");const walk=d=>fs.readdirSync(d,{withFileTypes:true}).flatMap(e=>{const p=path.join(d,e.name);return e.isDirectory()?walk(p):[p]});const files=[...walk("apps").filter(f=>f.endsWith(".json")),"project.config.json"].filter(fs.existsSync);for(const f of files)JSON.parse(fs.readFileSync(f,"utf8"));console.log(`JSON OK ${files.length}`)'
  ```

- [x] 4.5 汇总真实 Red、Green、Refactor、链接、敏感、owned-path、strict 和仓库 Gate 证据，按 C/T/V/R 评分；形成 candidate 前必须硬阻断为零、总分 `≥36`、每项 `≥8`，且 writer 阶段 V 最高为 8。未实测 RPO/RTO、容量和真实云配置继续明确标注，不作为本 W0 change 的 PASS 证据。
  - Evidence: `phase=Writer Gate`; `command=汇总 Red/Green/Refactor/官方链接/敏感/owned/strict/repo Gate`; `exit_result=0`; `sanitized_summary=硬阻断 0；C=10、T=9、V=8、R=9，总分 36；恢复边界明确但真实演练、容量和云配置未验证`; `artifact=本 tasks 证据与全部 Gate 输出`; `unverified_boundary=exact-SHA independent verification、云资源、RPO/RTO、容量`; `external_asset=仅官方链接已实时验证`。

## 5. Candidate and Independent Verification

- [x] 5.1 只暂存 owned paths，提交一个完整 `CANDIDATE`，记录完整 candidate SHA，并确认 `git status --short --branch` 仅显示 clean 分支、`git diff --exit-code` 与 `git diff --cached --exit-code` 均为 0；不得推送、创建 PR、部署、购云或修改外部系统。
  - Evidence: `phase=Candidate`; `command=git add <owned paths>; git diff --cached --name-only; git commit; git rev-parse HEAD; git status --short --branch; git diff --exit-code; git diff --cached --exit-code`; `exit_result=提交与 clean 结果由本地 handoff 绑定`; `sanitized_summary=只提交 owned paths；精确 SHA 与 clean 状态在提交后回传，避免把自引用 SHA 写入同一提交导致失效`; `artifact=本地 candidate commit`; `unverified_boundary=5.2/5.3 与 integration 未执行`; `external_asset=未推送/未建 PR/未部署/未购云/未写外部系统`。
- [ ] 5.2 verifier 在另一个 clean detached worktree 检出 5.1 的 exact SHA，只读重跑 1.3、3.1–3.4、4.1–4.4 和完整 diff 审查；验证前后 `git rev-parse HEAD` 必须等于 candidate SHA，worktree 必须 clean。
- [ ] 5.3 任一 proposal、design、spec、tasks、实际文档、验收命令、base、依赖、rebase、merge 或 candidate SHA 变化都使旧验证失效；原 writer 修复并产生新 SHA，verifier 复用 session 但从新 clean detached worktree 重跑全部 Gate。

## 6. Integration and Rollback

- [ ] 6.1 仅在 exact-SHA independent PASS、全部依赖已集成、C/T/V/R 达标、硬阻断为零且取得单独集成授权后，按 `$order-integrate-change` 集成；main 推进后视为新候选并重验，集成前不得 archive。
- [ ] 6.2 回滚时整体撤销两个实际文档和本 change 的实现状态记录；确认 Go/JS 代码、数据库、云资源和外部系统无需回滚。只有集成 main 后才能 archive 本 change。
