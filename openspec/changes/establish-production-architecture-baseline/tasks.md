> 状态：`DRAFT`。本轮只规划，不执行以下任务；所有 checkbox 保持未勾选。用户批准 apply 后，每完成一项必须按 `docs/quality/change-quality-gates.md` 的统一证据模板记录 `change/gate_type/ui_level/base_sha/candidate_sha/phase/command/exit_result/sanitized_summary/artifact/unverified_boundary/external_asset`。

## 1. Approval, Ownership and Red

- [ ] 1.1 取得用户对本 change 的明确批准，将状态从 `DRAFT` 更新为 `APPROVED`；确认唯一 writer 仍位于 branch `codex/establish-production-architecture-baseline` 的既有 worktree，基线为 `cb2605f477e58ac5471a0c535b85256c6be80a00`，目标文档不存在并行 writer 或未吸收变化。
- [ ] 1.2 完整重读 proposal、spec、design、tasks、根 `AGENTS.md` 与 `docs/quality/change-quality-gates.md`，运行 `openspec validate establish-production-architecture-baseline --strict`；确认 `gate_type=W0`、`ui_level_target=UI0`、外部链接 owner/恢复条件和 `Open Questions=无` 后才能进入 `IMPLEMENTING`。
- [ ] 1.3 在修改两个实际文档前运行以下内容检查并记录 Red；失败必须来自当前文档缺少唯一拓扑、事务/worker、迁移、隔离、恢复或容量规则，以及仍含二选一、公有读、长期 key 等旧口径，不得来自脚本或环境错误：

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

- [ ] 1.4 逐项对照归档 `mvp-product-baseline`、`bootstrap-api-service` 与当前两个文档，记录现有抽象部署形态、推荐状态/角色、数据库二选一、公有读 COS、长期访问 key、CDN 和未量化恢复/容量口径的位置；库存三维键与 15 分钟软预占只作为已确认依赖记录，不新增或改变其业务规则。

## 2. Green: Single Architecture Documentation

- [ ] 2.1 只修改 `docs/product/online-ordering-system-technical.md`，按 design D1–D9 写入组件图、同步/异步数据流、MySQL 事实源、inbox/outbox/worker、迁移、环境隔离、配置/密钥、私有 COS、恢复目标、观测阈值、容量门和升级顺序；明确 RPO/RTO 与容量未实测，不把目标写成已达成。
- [ ] 2.2 只修改 `docs/微信小程序开发和运维指南/腾讯云操作指南.md`，把客户采购与责任口径固定为 CVM、TencentDB MySQL 8.0 双节点多可用区、私有 COS、SSM/CAM、CLS/云监控/CAT；删除数据库二选一、公有读 COS、长期 SecretId/SecretKey 交付和默认 CDN，只保留 spec 白名单外部占位符及官方直链。
- [ ] 2.3 重跑 1.3 的同一内容检查并记录 Green；必须输出 `architecture baseline content PASS`。逐行确认两个实际文档与 spec/design 使用同一组件、时限、保留期、阈值、非目标和升级条件。

## 3. Refactor: Consistency, Links and Safety

- [ ] 3.1 运行以下歧义与外部占位符检查；两个实际文档合计必须只含完整白名单，不得残留普通占位、行为未决或旧选型口径：

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

- [ ] 3.2 检查 Markdown 本地链接存在、所有架构外链为腾讯云或微信支付官方 HTTPS 直链，并实际访问每个去重 URL；public internet 是必要外部资产，owner=`Production Architecture Writer`。任一页面不可访问时记录 `BLOCKED_EXTERNAL` 与恢复条件，不得记 PASS：

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

  python3 - <<'PY' >/tmp/order-architecture-official-urls.txt
  import re
  from pathlib import Path
  paths = [
      Path("openspec/changes/establish-production-architecture-baseline/design.md"),
      Path("docs/product/online-ordering-system-technical.md"),
      Path("docs/微信小程序开发和运维指南/腾讯云操作指南.md"),
  ]
  urls = set()
  for path in paths:
      urls.update(re.findall(r"https://[^\s，、）)>]+", path.read_text(encoding="utf-8")))
  print("\n".join(sorted(urls)))
  PY
  while IFS= read -r url; do
    curl --location --fail --silent --show-error --max-time 20 --output /dev/null "$url"
  done </tmp/order-architecture-official-urls.txt
  ```

- [ ] 3.3 运行敏感信息静态检查；允许回环地址与命名占位符，禁止真实凭据、个人数据、生产资源标识或可重放正文：

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

- [ ] 3.4 对照 `mvp-product-baseline` 确认技术文档只引用而不改变`营业日期 × 餐段 × 商品`、15 分钟软预占、九态和四角色；确认 RPO/RTO、容量、云 SKU 和成本均明确区分目标、未实测与外部值。

## 4. Local Writer Gate

- [ ] 4.1 运行 `openspec validate establish-production-architecture-baseline --strict` 与 `openspec status --change establish-production-architecture-baseline --json`，确认四类 artifacts 完整、tasks 可 apply，且状态仍按实际阶段记录。
- [ ] 4.2 检查 owned Markdown 文件标题、围栏、表格和相对链接结构完整；执行 `git diff --check cb2605f477e58ac5471a0c535b85256c6be80a00`。
- [ ] 4.3 运行 owned-path Gate；只允许本 change 目录和两个获批实际文档发生变化，任何 PRD、客户清单、合同、质量/loop skill 或业务代码 diff 都立即 FAIL：

  ```bash
  unexpected_paths="$({
    git diff --name-only cb2605f477e58ac5471a0c535b85256c6be80a00
    git diff --cached --name-only
  } | sort -u | grep -Ev '^(openspec/changes/establish-production-architecture-baseline/|docs/product/online-ordering-system-technical\.md$|docs/微信小程序开发和运维指南/腾讯云操作指南\.md$)' || true)"
  test -z "$unexpected_paths"
  ```

- [ ] 4.4 运行当前仓库 Gate；本 change 虽为 W0，也必须证明规划的文档修改没有伴随代码、前端或基础进程回归：

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

- [ ] 4.5 汇总真实 Red、Green、Refactor、链接、敏感、owned-path、strict 和仓库 Gate 证据，按 C/T/V/R 评分；形成 candidate 前必须硬阻断为零、总分 `≥36`、每项 `≥8`，且 writer 阶段 V 最高为 8。未实测 RPO/RTO、容量和真实云配置继续明确标注，不作为本 W0 change 的 PASS 证据。

## 5. Candidate and Independent Verification

- [ ] 5.1 只暂存 owned paths，提交一个完整 `CANDIDATE`，记录完整 candidate SHA，并确认 `git status --short --branch` 仅显示 clean 分支、`git diff --exit-code` 与 `git diff --cached --exit-code` 均为 0；不得推送、创建 PR、部署、购云或修改外部系统。
- [ ] 5.2 verifier 在另一个 clean detached worktree 检出 5.1 的 exact SHA，只读重跑 1.3、3.1–3.4、4.1–4.4 和完整 diff 审查；验证前后 `git rev-parse HEAD` 必须等于 candidate SHA，worktree 必须 clean。
- [ ] 5.3 任一 proposal、design、spec、tasks、实际文档、验收命令、base、依赖、rebase、merge 或 candidate SHA 变化都使旧验证失效；原 writer 修复并产生新 SHA，verifier 复用 session 但从新 clean detached worktree 重跑全部 Gate。

## 6. Integration and Rollback

- [ ] 6.1 仅在 exact-SHA independent PASS、全部依赖已集成、C/T/V/R 达标、硬阻断为零且取得单独集成授权后，按 `$order-integrate-change` 集成；main 推进后视为新候选并重验，集成前不得 archive。
- [ ] 6.2 回滚时整体撤销两个实际文档和本 change 的实现状态记录；确认 Go/JS 代码、数据库、云资源和外部系统无需回滚。只有集成 main 后才能 archive 本 change。
