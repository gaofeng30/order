# 共享 PG 完整履约与订单读取

Status: APPROVED / IMPLEMENTING。用户已授权整体 CloudBase UAT 改造；本 change 交付现有履约能力的完整下一切片，不启用部分后端。内部写RPC与只读身份seam已由需求和writer审定，未实施项不记完成。

## Why

首切片已实现普通服务 JWT 下的 REDEEM_ORDER，但 MARK_READY、扫码/当日取餐码及真实订单读模型仍依赖原生连接。要让现有履约 HTTP 响应经过真实读模型完成，必须一起迁移剩余三命令、五个读取端口和 MARK_READY 事务内订阅授权消费/通知意图。

## What Changes

- 扩展现有 fulfillment.Application，保留四命令各自定位、状态、权限与重放差异；复用首切片身份、传输、受控安装及 readiness。
- 新共享 PG orderquery.Application 实现全部五个方法，保留原过滤/分页/投影/错误及商户当前身份依赖；Go 保持解密与公共 DTO。
- MARK_READY 一次 RPC 原子提交订单、ACCEPTED consent 消费、不可变 READY outbox 和 audit receipt；不迁移 RecordConsent/RunDue，不向微信发送消息。
- 扩展固定 RPC allowlist、精确安装/ACL/RLS/readiness；原44迁移和账本不改，不引入任意 SQL/事务模拟/管理员回退。

## Ownership and Gate

- gate_type: W3；ui_level_target: UI0；ui_level_actual: NOT_RUN。无客户端文件改动，不宣称 Mini UI3。
- base_sha: 775e6746ded9555f9dd243682db011a869b72fbd；candidate_sha: NOT_CREATED。
- branch: codex/order-uat-shared-pg-fulfillment；worktree: /Users/vivix/.codex/worktrees/order-uat-shared-pg-fulfillment/order。
- requirement owner /root/astra_cloudbase_ablation 仅写本 change 四工件。
- root owns orderquery 共享适配/对应测试、固定 RPC registry/安装/readiness/client allowlist 和受控管理迁移。root亦独占merchantidentity/repository.go的必要只读投影提取、新增sharedpg_read.go/test及sharedpg/identity_read.sql；不改身份写流程。
- fulfillment writer owns root 分配的 fulfillment SharedPG Go/SQL 及对应测试；履约SQL独占services/api/sharedpg/fulfillment.sql，与root的registry/identity/query SQL分离。一文件仅一个 writer，不能改首切片候选 worktree 或其规格。
- dependencies: adopt-cloudbase-shared-pg-runtime exact base；exact 775e6746ded9555f9dd243682db011a869b72fbd 的独验已完成，尚未集成 main；不得据此绕过本 change 的候选、独验和 Integration Gate。
- non-goals: 登录/手机绑定/商户管理写端、subscription consent HTTP/发送worker、支付退款/其他workers、存储迁移、全runtime wiring、UI改造、采购、Prod、实际资金、push/PR/merge/archive。

## Minimum Success

原生与共享实现经相同 Application 和现有 handler 的真实 v44 差分通过，读取使用真实共享读模型；真实 UAT 普通 JWT 完成 MARK_READY→用户读取解密token→三种核销→真实商户响应、并发/响应丢失/授权和事务故障矩阵，所有意图与回执副作用精确一次且清理复读。安装/readiness 检出新增定义与有效权限漂移。Writer Gate 后候选 exact SHA，另 clean detached 独验；不代表全量 UAT。

## External Assets

继续使用 root 控制的 Order UAT 普通服务 UID 和受控管理安装通道，秘密不进源码/日志。合成 consent/outbox 只证明持久化，不证明微信发送或到达。真实 AppSecret/商户资料/template/ownerphone、Mini UI3、完整私有存储最小权限与恢复、95Case/25页面仍后续；无 broad APIKey 的完整 runtime 硬门保持。


真实 UAT 已发现原 v44 已知业务关系的 CloudBase 默认宽授权；本切片受控权限安装须按权威25业务表加账本及16 sequence清单撤销 PUBLIC/anon/authenticated 现有权限，再精确授予当前10表/2sequence。该修正属于已批准最小权限的必要配置，不改变原44/schema/管理owner；未知对象不自动撤权。安装前回执已记录60条额外有效role-relation授权、10条显式列ACL和0额外可执行SECDEF；已由受控版本20261008222000纠正，并以普通JWT readiness及实际catalog复核通过；这是独立平台证据，不继承本地无默认授权环境的结论。


本切片真实 UAT 安装及限定业务矩阵、权限负例、合成资产清理已通过，证据为platform-receipt.json（SHA256 df73a04cb28cfe1bc5368699d2cf07dcabbe3fa1b5c425621a5dcbeba2d11057）。Go handler使用合成业务transport auth，数据库连接使用真实普通服务JWT；不代表真实Order/WeChat登录或CloudRun上线。最终Writer Gate已完成；候选及独验尚未完成，完整UAT目标不变。候选exact SHA提交后由外部receipt/harness记录，避免同一commit自引用。
