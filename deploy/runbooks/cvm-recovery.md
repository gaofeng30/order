# Order production CVM rebuild and startup runbook

## Scope and evidence boundary

This runbook rebuilds the stateless production CVM runtime from a reviewed release artifact. It does not create cloud resources, restore or modify TencentDB data, create SSM secrets, bind a CAM Role, configure Nginx/TLS, or prove the documented RPO/RTO targets. Those remain separately authorized external gates.

The production runtime is systemd/CVM. Docker is N/A. The only runtime SecretNames are `order-production-db-password` and `order-production-wechat-miniprogram-app-secret`; both are read at startup as `SSM_Current` through the CVM CAM Role. No secret value belongs in a command, file, unit, log, or release artifact.

## External prerequisites

Before changing the CVM, confirm by non-sensitive resource reference that:

1. the target is the production CVM/VPC and the TencentDB endpoint is private;
2. the CVM has the production CAM Role, restricted to `ssm:GetSecretValue` for the two exact production SecretNames;
3. both SSM secrets exist and are enabled in the configured Tencent region;
4. the reviewed release contains Linux executables `order-api` and `order-migrate` from one exact commit;
5. Nginx, TLS, CLS/LogListener, CAT and Cloud Monitor are handled by their approved external runbooks;
6. a manual database backup is confirmed only when the release contains a destructive migration.

Stop on the first failed prerequisite. Do not substitute local files, long-lived SecretId/SecretKey values, command-line secrets, or a development configuration.

## Rebuild runtime files

Run these steps only inside a separately authorized deployment change:

1. Create the fixed non-login `order` user and the root-owned directories `/opt/order/releases/<exact-sha>` and `/etc/order`.
2. Install `order-api`, `order-migrate`, `order-preflight`, and `order-healthcheck` into the release directory. Set owner `root:root`, executable mode `0755`, then atomically point `/opt/order/current` at that release.
3. Copy `order-production.env.example` to `/etc/order/order-production.env`, replace every `TODO_`/`REPLACE_` value with the approved non-sensitive production value, then set owner `root:order` and mode `0640`. Never add a database password, Mini Program AppSecret, Tencent Cloud key/token, or DSN.
4. Install `order-api.service` and `order-migrate.service` in `/etc/systemd/system/`, set owner `root:root` and mode `0644`, then run `systemctl daemon-reload`.
5. Run `/opt/order/current/order-preflight`. A failure blocks migration and startup without printing configuration values.

## Migrate, start and verify

Execute in this order and stop on the first non-zero result:

1. `systemctl start order-migrate.service`
2. `journalctl -u order-migrate.service -n 50 --no-pager` and confirm the terminal event is `migration_complete` without credentials or raw provider responses.
3. `systemctl restart order-api.service`
4. `/opt/order/current/order-healthcheck live`
5. `/opt/order/current/order-healthcheck ready`
6. Verify Nginx HTTPS routing, CLS ingestion, CAT, Cloud Monitor and alert delivery through their external gates.

`live` proves only that the process is serving. Traffic and release success require `ready`, which verifies the real database connection and exact embedded migration history. The API never runs migration or down migration automatically.

## Failure and rollback

- Configuration, CAM metadata or SSM failure: keep the service failed, inspect only the stable journal reason, correct the external role/secret/configuration, and restart. Never print or copy the secret value for diagnosis.
- Migration failure: do not start the new API. Preserve the first stable migration reason and use a forward fix or the approved database recovery path; never run automatic down.
- API/readiness failure after a successful migration: point `/opt/order/current` to the previously reviewed binary only if it is compatible with the current schema, restart, and rerun both health checks.
- CVM loss: rebuild this stateless runtime and reattach the production CAM Role. Business facts remain in TencentDB; do not restore them from the CVM filesystem.
- Logical database damage: stop this runbook and use the separately authorized TencentDB backup/binlog point-in-time recovery procedure in an isolated instance before any production cutover.

Record the exact release SHA, migration terminal result, live/ready results and external gate references. Do not record secrets, cloud credentials, account identifiers, personal data or raw provider responses.
