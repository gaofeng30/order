#!/usr/bin/env bash
set -euo pipefail

evidence_repo=$(git rev-parse --show-toplevel)
evidence_tasks=${1:-"${evidence_repo}/.scratch/implement-quote-pricing-core/tasks.md"}
evidence_expected_ids=(
  QP-01 QP-02 QP-03 QP-04 QP-05 QP-06 QP-07
  QP-08 QP-09 QP-10 QP-11 QP-12 QP-13
)

if [[ ! -f ${evidence_tasks} ]]; then
  printf 'evidence ledger missing: %s\n' "${evidence_tasks}" >&2
  exit 83
fi

if [[ $(grep -c '^- \[x\]' "${evidence_tasks}") -ne ${#evidence_expected_ids[@]} ]]; then
  printf 'completed task count mismatch\n' >&2
  exit 84
fi
if [[ $(grep -c '^- \[ \]' "${evidence_tasks}") -ne 3 ]]; then
  printf 'pending post-commit task count mismatch\n' >&2
  exit 85
fi
for evidence_task_id in "${evidence_expected_ids[@]}"; do
  evidence_checkbox='- [x] `'"${evidence_task_id}"'`'
  if ! grep -Fq -- "${evidence_checkbox}" "${evidence_tasks}"; then
    printf 'missing completed checkbox for task_id=%s\n' "${evidence_task_id}" >&2
    exit 86
  fi
done

evidence_record_count=$(grep -c '^evidence_id:' "${evidence_tasks}")
evidence_expected_csv=$(IFS=,; printf '%s' "${evidence_expected_ids[*]}")
if ! awk -v expected_csv="${evidence_expected_csv}" '
  function fail(message) {
    printf "invalid evidence record: %s\n", message > "/dev/stderr"
    invalid = 1
  }

  function reset_record(    key) {
    for (key in top_count) delete top_count[key]
    for (key in asset_count) delete asset_count[key]
    task = ""
    evidence_id = ""
    phase = ""
    exit_result = ""
  }

  function field_value(line,    value) {
    value = line
    sub(/^[^:]+:[[:space:]]*/, "", value)
    return value
  }

  function validate_record(    field_index_local, field, pair) {
    record_count++
    for (field_index_local = 1; field_index_local <= top_field_count; field_index_local++) {
      field = top_fields[field_index_local]
      if (top_count[field] != 1) {
        fail("record=" record_count " field=" field " count=" (top_count[field] + 0))
      }
    }
    for (field_index_local = 1; field_index_local <= asset_field_count; field_index_local++) {
      field = asset_fields[field_index_local]
      if (asset_count[field] != 1) {
        fail("record=" record_count " external_asset." field " count=" (asset_count[field] + 0))
      }
    }
    if (!(task in expected_task)) {
      fail("record=" record_count " unexpected task_id=" task)
    } else {
      task_record_count[task]++
    }
    if (evidence_id == "") {
      fail("record=" record_count " empty evidence_id")
    } else if (seen_evidence_id[evidence_id]++) {
      fail("record=" record_count " duplicate evidence_id=" evidence_id)
    }
    if (phase !~ /^(red|green|refactor|writer|verifier|integration)$/) {
      fail("record=" record_count " invalid phase=" phase)
    }
    if (exit_result !~ /^(PASS|FAIL|BLOCKED_EXTERNAL|exit-[0-9]+|[0-9]+)$/) {
      fail("record=" record_count " invalid exit_result=" exit_result)
    }
    pair = task ":" phase
    observed_pair[pair] = 1
  }

  BEGIN {
    top_field_count = split("task_id evidence_id evidence_origin change gate_type ui_level_target ui_level_actual base_sha candidate_sha phase command_or_action exit_result sanitized_summary artifact_or_environment unverified_boundary external_asset", top_fields, " ")
    asset_field_count = split("owner missing recovery", asset_fields, " ")
    expected_count = split(expected_csv, expected_ids, ",")
    for (expected_index = 1; expected_index <= expected_count; expected_index++) {
      expected_task[expected_ids[expected_index]] = 1
    }
    required_pair["QP-03:green"] = 1
    required_pair["QP-05:red"] = 1
    required_pair["QP-05:green"] = 1
    required_pair["QP-06:red"] = 1
    required_pair["QP-06:green"] = 1
    required_pair["QP-07:red"] = 1
    required_pair["QP-07:green"] = 1
    required_pair["QP-08:red"] = 1
    required_pair["QP-08:green"] = 1
    required_pair["QP-09:refactor"] = 1
    required_pair["QP-10:refactor"] = 1
    required_pair["QP-11:writer"] = 1
    required_pair["QP-12:red"] = 1
    required_pair["QP-12:green"] = 1
    required_pair["QP-13:red"] = 1
    required_pair["QP-13:green"] = 1
  }

  /^```yaml$/ {
    if (in_record) fail("nested yaml fence")
    in_record = 1
    reset_record()
    next
  }

  /^```$/ && in_record {
    validate_record()
    in_record = 0
    next
  }

  in_record {
    for (field_index = 1; field_index <= top_field_count; field_index++) {
      current_field = top_fields[field_index]
      if (index($0, current_field ":") == 1) top_count[current_field]++
    }
    for (asset_index = 1; asset_index <= asset_field_count; asset_index++) {
      current_asset_field = asset_fields[asset_index]
      if (index($0, "  " current_asset_field ":") == 1) {
        asset_count[current_asset_field]++
        if (top_count["external_asset"] != 1) {
          fail("external_asset." current_asset_field " is not below a single external_asset field")
        }
      }
    }
    if ($0 ~ /^task_id:[[:space:]]*/) task = field_value($0)
    if ($0 ~ /^evidence_id:[[:space:]]*/) evidence_id = field_value($0)
    if ($0 ~ /^phase:[[:space:]]*/) phase = field_value($0)
    if ($0 ~ /^exit_result:[[:space:]]*/) exit_result = field_value($0)
  }

  END {
    if (in_record) fail("unterminated yaml fence")
    if (record_count == 0) fail("no fenced records")
    for (expected_index = 1; expected_index <= expected_count; expected_index++) {
      expected_id = expected_ids[expected_index]
      if (task_record_count[expected_id] == 0) {
        fail("missing record for task_id=" expected_id)
      }
    }
    for (pair in required_pair) {
      if (!observed_pair[pair]) fail("missing required task phase=" pair)
    }
    if (invalid) exit 1
  }
' "${evidence_tasks}"; then
  exit 87
fi

if ! grep -Fq '7a5412546e9d1c59e1213ea668245e60db52e63e' "${evidence_tasks}" || \
  ! grep -Fq 'INVALIDATED_BY_INDEPENDENT_STANDARDS_AND_SPEC_REVIEW' "${evidence_tasks}" || \
  ! grep -Fqx -- '- All writer review and detached receipts bound to the invalidated SHA are void; it must not be integrated. Replacement remains `candidate_sha: external-post-commit` until immutable handoff.' "${evidence_tasks}"; then
  printf 'old candidate invalidation missing\n' >&2
  exit 91
fi
if ! grep -Fq '8650359395bd0b5117217dee967ec6b09d831a0b' "${evidence_tasks}" || \
  ! grep -Fq 'INVALIDATED_BY_INDEPENDENT_STANDARDS_REVIEW' "${evidence_tasks}" || \
  ! grep -Fqx -- '- Its Spec 0-finding receipt and all writer/review receipts are void; it must not be integrated. The next replacement also remains `candidate_sha: external-post-commit` until immutable handoff.' "${evidence_tasks}"; then
  printf 'replacement candidate invalidation missing\n' >&2
  exit 93
fi

printf 'EVIDENCE_STRUCTURE=PASS completed=%s records=%s pending=3 per_record=validated invalidated_candidates=2\n' \
  "${#evidence_expected_ids[@]}" "${evidence_record_count}"
