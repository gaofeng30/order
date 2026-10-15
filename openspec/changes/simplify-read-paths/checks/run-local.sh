#!/bin/sh
set -eu
cd "$(dirname "$0")/../../../.."
node openspec/changes/simplify-read-paths/checks/menu-read-budget.cjs
node openspec/changes/simplify-read-paths/checks/payment-recovery.cjs
node openspec/changes/simplify-read-paths/checks/order-counts.cjs
ORDER_TEST_PG_BIN=/opt/homebrew/opt/postgresql@16/bin go test -race ./services/api/...
go vet ./services/api/...
python3 openspec/changes/simplify-read-paths/checks/postgres.py
python3 tools/miniprogram_gate.py plan --change simplify-read-paths
openspec validate simplify-read-paths --strict
git diff --check
