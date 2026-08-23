#!/usr/bin/env bash
set -euo pipefail

evidence_repo=$(git rev-parse --show-toplevel)
evidence_tasks="${evidence_repo}/.scratch/implement-quote-pricing-core/tasks.md"
evidence_expected_ids=(
  QP-01 QP-02 QP-03 QP-04 QP-05 QP-06 QP-07
  QP-08 QP-09 QP-10 QP-11 QP-12 QP-13
)

if [[ $(grep -c '^- \[x\]' "${evidence_tasks}") -ne ${#evidence_expected_ids[@]} ]]; then
  printf 'completed task count mismatch\n' >&2
  exit 84
fi
if [[ $(grep -c '^- \[ \]' "${evidence_tasks}") -ne 3 ]]; then
  printf 'pending post-commit task count mismatch\n' >&2
  exit 85
fi
for evidence_task_id in "${evidence_expected_ids[@]}"; do
  if ! grep -Fqx "task_id: ${evidence_task_id}" "${evidence_tasks}"; then
    printf 'missing unified evidence for task_id=%s\n' "${evidence_task_id}" >&2
    exit 86
  fi
done

evidence_record_count=$(grep -c '^evidence_id:' "${evidence_tasks}")
for evidence_field in \
  task_id evidence_id evidence_origin change gate_type ui_level_target \
  ui_level_actual base_sha candidate_sha phase command_or_action exit_result \
  sanitized_summary artifact_or_environment unverified_boundary external_asset; do
  evidence_field_count=$(grep -c "^${evidence_field}:" "${evidence_tasks}")
  if [[ ${evidence_field_count} -ne ${evidence_record_count} ]]; then
    printf 'field count mismatch field=%s count=%s records=%s\n' \
      "${evidence_field}" "${evidence_field_count}" "${evidence_record_count}" >&2
    exit 87
  fi
done

if ! grep '^phase:' "${evidence_tasks}" | \
  awk '$2 !~ /^(red|green|refactor|writer|verifier|integration)$/ {bad=1} END {exit bad}'; then
  printf 'invalid phase enum\n' >&2
  exit 88
fi
if ! grep -Fqx 'phase: green' "${evidence_tasks}"; then
  printf 'green phase missing\n' >&2
  exit 89
fi
evidence_required_phase_pairs=(
  QP-03:green
  QP-05:red QP-05:green
  QP-06:red QP-06:green
  QP-07:red QP-07:green
  QP-08:red QP-08:green
  QP-09:refactor QP-10:refactor QP-11:writer
  QP-12:green QP-13:red QP-13:green
)
for evidence_pair in "${evidence_required_phase_pairs[@]}"; do
  evidence_pair_task=${evidence_pair%%:*}
  evidence_pair_phase=${evidence_pair##*:}
  if ! awk -v wanted_task="${evidence_pair_task}" -v wanted_phase="${evidence_pair_phase}" '
    $1 == "task_id:" { current_task = $2 }
    $1 == "phase:" && current_task == wanted_task && $2 == wanted_phase { found = 1 }
    END { exit !found }
  ' "${evidence_tasks}"; then
    printf 'missing task phase task_id=%s phase=%s\n' \
      "${evidence_pair_task}" "${evidence_pair_phase}" >&2
    exit 92
  fi
done
if ! grep '^exit_result:' "${evidence_tasks}" | \
  awk '$2 !~ /^(PASS|FAIL|BLOCKED_EXTERNAL|exit-[0-9]+|[0-9]+)$/ {bad=1} END {exit bad}'; then
  printf 'invalid exit_result enum\n' >&2
  exit 90
fi
if ! grep -Fq '7a5412546e9d1c59e1213ea668245e60db52e63e' "${evidence_tasks}" || \
  ! grep -Fq 'INVALIDATED_BY_INDEPENDENT_STANDARDS_AND_SPEC_REVIEW' "${evidence_tasks}"; then
  printf 'old candidate invalidation missing\n' >&2
  exit 91
fi

printf 'EVIDENCE_STRUCTURE=PASS completed=%s records=%s pending=3 green=present old_candidate=invalidated\n' \
  "${#evidence_expected_ids[@]}" "${evidence_record_count}"
