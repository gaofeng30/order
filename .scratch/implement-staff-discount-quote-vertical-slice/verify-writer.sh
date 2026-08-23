#!/usr/bin/env bash
set -euo pipefail

writer_repo=$(git rev-parse --show-toplevel)
writer_base=1657aa9451f612e4605fabd084ccab07542ac81a
writer_scratch=.scratch/implement-staff-discount-quote-vertical-slice
cd "${writer_repo}"

writer_changed_paths() {
  {
    git diff --name-only "${writer_base}"...HEAD
    git diff --cached --name-only
    git diff --name-only
    git ls-files --others --exclude-standard
  } | LC_ALL=C sort -u
}

verify_writer_source_state() {
  git cat-file -e "${writer_base}^{commit}"
  test "$(git rev-parse HEAD)" = "${writer_base}"
  if ! git diff --quiet; then
    printf 'writer source has unstaged changes\n' >&2
    exit 91
  fi
  if [[ -n $(git ls-files --others --exclude-standard) ]]; then
    printf 'writer source has untracked files\n' >&2
    exit 92
  fi
  if git diff --cached --quiet; then
    printf 'writer index has no candidate delta\n' >&2
    exit 93
  fi
  while IFS= read -r writer_path; do
    [[ -z "${writer_path}" ]] && continue
    case "${writer_path}" in
      "${writer_scratch}"/*|services/api/internal/quote/*|services/api/migrations/000014_create_staff_whitelist.sql|services/api/migrations/000015_create_discount_settings.sql|services/api/migrations/000016_create_quotes.sql|services/api/migrations/000017_create_quote_items.sql|services/api/migrations/embed_test.go|services/api/internal/catalog/migrations_test.go) ;;
      *)
        printf 'out-of-scope path: %s\n' "${writer_path}" >&2
        exit 94
        ;;
    esac
  done < <(writer_changed_paths)
  git diff --cached --check
}

verify_writer_dependencies() {
  test -f services/api/internal/storestatus/core.go
  writer_store_mutation_files=$(rg -l -i \
    '(UPDATE[[:space:]]+storefront_settings|INSERT[[:space:]]+INTO[[:space:]]+storefront_settings|DELETE[[:space:]]+FROM[[:space:]]+storefront_settings|REPLACE[[:space:]]+INTO[[:space:]]+storefront_settings)' \
    services/api --glob '*.go' --glob '!**/*_test.go' || true)
  if [[ "${writer_store_mutation_files}" != services/api/internal/storestatus/core.go ]]; then
    printf 'integrated store-status write ownership is not exact\n' >&2
    exit 95
  fi
  if ! rg -F 'store_name,store_address,pickup_point,business_status,flavor_options_json,record_version' services/api/internal/quote/provider.go >/dev/null; then
		printf 'quote does not retain its complete locked storefront read\n' >&2
    exit 96
  fi
}

verify_writer_hygiene() {
  test -z "$(gofmt -l services/api/internal/quote services/api/migrations/embed_test.go services/api/internal/catalog/migrations_test.go)"
  bash -n "${writer_scratch}"/*.sh
  if rg -i \
    '(time_expire|prepayment|pickup_number)' \
    services/api/migrations/000014_create_staff_whitelist.sql \
    services/api/migrations/000015_create_discount_settings.sql \
    services/api/migrations/000016_create_quotes.sql \
    services/api/migrations/000017_create_quote_items.sql >/dev/null; then
    printf 'out-of-scope TTL, prepay, or pickup-number concept found in production delta\n' >&2
    exit 97
  fi
  if rg -F 'idx_staff_whitelist_enabled_id' services/api/migrations/000014_create_staff_whitelist.sql >/dev/null; then
    printf 'unbudgeted staff whitelist enabled/id index found\n' >&2
    exit 97
  fi
  for writer_required_contract in \
    'contact_name_snapshot VARCHAR(64) NOT NULL' \
    'contact_phone_snapshot VARBINARY(16) NOT NULL' \
    'OCTET_LENGTH(contact_name_snapshot) <= 64' \
    'discount_rate_percent >= 1 AND discount_rate_percent <= 100' \
    'payable_cents > 0'; do
    if ! rg -F "${writer_required_contract}" services/api/migrations/000016_create_quotes.sql >/dev/null; then
      printf 'required quote migration contract missing: %s\n' "${writer_required_contract}" >&2
      exit 97
    fi
  done
  for writer_required_item_contract in \
    'image_object_key_snapshot VARBINARY(1024) NULL' \
    'OCTET_LENGTH(image_object_key_snapshot) BETWEEN 1 AND 1024'; do
    if ! rg -F "${writer_required_item_contract}" services/api/migrations/000017_create_quote_items.sql >/dev/null; then
      printf 'required quote-item migration contract missing: %s\n' "${writer_required_item_contract}" >&2
      exit 97
    fi
  done
  if ! rg -F 'ExpiresAt             time.Time' services/api/internal/quote/types.go >/dev/null ||
    ! rg -F 'if !observedAt.Before(snapshot.ExpiresAt)' services/api/internal/quote/prepay.go >/dev/null ||
    ! rg -F 'ErrSnapshotInvalid' services/api/internal/quote/errors.go >/dev/null; then
    printf 'required public deadline or snapshot-error contract missing\n' >&2
    exit 97
  fi
  for writer_current_fact_contract in \
    'primary_phone,primary_phone_bound_at,extra_phone,extra_name,extra_name_key,extra_phone_set_at,record_version' \
    'FROM service_dates' \
    'flavor_options_json,record_version' \
    'Extra: user.Extra' \
    'json:"meal_period"'; do
    if ! rg -F "${writer_current_fact_contract}" services/api/internal/quote >/dev/null; then
      printf 'required frozen current-fact/DTO contract missing: %s\n' "${writer_current_fact_contract}" >&2
      exit 97
    fi
  done
  if rg -n '(type AdminProvider|type AdminTarget|type OwnerAuthorizer|func NewAdminProvider|SaveDiscountRate|SaveStaffEntry|ReceiptActionDiscountSave|ReceiptActionStaffEntrySave)' \
    services/api/internal/quote --glob '!**/*_test.go' >/dev/null; then
    printf 'Quote package contains StaffDiscount-owned mutator vocabulary\n' >&2
    exit 97
  fi
  if rg -i \
    '(INSERT[[:space:]]+INTO[[:space:]]+(prepayments|orders)|wechatpay|requestpayment|createprepay)' \
    services/api/internal/quote --glob '!**/*_test.go' >/dev/null; then
    printf 'out-of-scope prepayment, order, or payment-provider write found in quote production delta\n' >&2
    exit 97
  fi
  if rg -i \
    '(-----BEGIN[[:space:]][A-Z ]+PRIVATE KEY|Authorization:[[:space:]]*Bearer[[:space:]]+[A-Za-z0-9]|Cookie:[[:space:]]*[^<]|api[_-]?key[[:space:]]*[:=][[:space:]]*[A-Za-z0-9])' \
    "${writer_scratch}" services/api/internal/quote \
    services/api/migrations/000014_create_staff_whitelist.sql \
    services/api/migrations/000015_create_discount_settings.sql \
    services/api/migrations/000016_create_quotes.sql \
    services/api/migrations/000017_create_quote_items.sql \
    --glob '!**/*_test.go' --glob '!verify-writer.sh' >/dev/null; then
    printf 'sensitive literal scan failed\n' >&2
    exit 98
  fi
}

run_without_mysql() {
  env \
    -u ORDER_TEST_MYSQL_HOST \
    -u ORDER_TEST_MYSQL_PORT \
    -u ORDER_TEST_MYSQL_USER \
    -u ORDER_TEST_MYSQL_PASSWORD \
    -u ORDER_TEST_MYSQL_TLS_MODE \
    -u ORDER_TEST_MYSQL_INSTANCE \
    -u ORDER_TEST_MYSQL_ISOLATED \
    GOWORK=off \
    GOCACHE=/private/tmp/order-tx02-final-gocache \
    GOPROXY=off \
    GOTOOLCHAIN=go1.26.5 \
    "$@"
}

verify_writer_source_state
verify_writer_dependencies
verify_writer_hygiene
writer_tree_before=$(git write-tree)
writer_status_before=$(git status --porcelain)

run_without_mysql go test ./services/api/internal/quote -count=1
run_without_mysql go test -race ./services/api/internal/quote -count=20 -timeout=15m
bash "${writer_scratch}/verify-mutation-gate.sh"
bash "${writer_scratch}/verify-mysql.sh"
run_without_mysql go test ./services/api/... -count=1 -timeout=15m
run_without_mysql go test -race ./services/api/... -count=1 -timeout=20m
run_without_mysql go vet ./services/api/...
bash "${writer_scratch}/verify-build.sh"
run_without_mysql bash services/api/scripts/smoke.sh

verify_writer_source_state
verify_writer_dependencies
verify_writer_hygiene
test "$(git write-tree)" = "${writer_tree_before}"
test "$(git status --porcelain)" = "${writer_status_before}"
if [[ -n $(docker ps --filter 'name=order-quote-w3-' --format '{{.Names}}') ]]; then
  printf 'temporary quote MySQL container remains\n' >&2
  exit 99
fi
for writer_temp_pattern in \
  'order-quote-w3-password.*' \
  'order-quote-mutations.*' \
  'order-quote-build.*'; do
  if find /private/tmp -maxdepth 1 -name "${writer_temp_pattern}" -print -quit | grep -q .; then
    printf 'temporary quote artifact remains: %s\n' "${writer_temp_pattern}" >&2
    exit 100
  fi
done

printf 'WRITER_GATE=PASS base_sha=%s head_sha=%s tree=%s\n' \
  "${writer_base}" "$(git rev-parse HEAD)" "${writer_tree_before}"
