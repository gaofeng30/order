#!/usr/bin/env bash
set -euo pipefail

evidence_gate_repo=$(git rev-parse --show-toplevel)
evidence_gate_checker="${evidence_gate_repo}/.scratch/implement-quote-pricing-core/verify-evidence.sh"
evidence_gate_source="${evidence_gate_repo}/.scratch/implement-quote-pricing-core/tasks.md"
evidence_gate_tmp=$(mktemp -d "${TMPDIR:-/tmp}/quote-evidence-gate.XXXXXX")

cleanup_evidence_gate() {
  case "${evidence_gate_tmp}" in
    "${TMPDIR:-/tmp}"/quote-evidence-gate.*)
      rm -rf -- "${evidence_gate_tmp}"
      ;;
    *)
      printf 'refusing unsafe evidence Gate cleanup: %s\n' "${evidence_gate_tmp}" >&2
      exit 99
      ;;
  esac
}
trap cleanup_evidence_gate EXIT

expect_evidence_rejection() {
  evidence_gate_name=$1
  evidence_gate_mutant=$2

  set +e
  evidence_gate_output=$(bash "${evidence_gate_checker}" "${evidence_gate_mutant}" 2>&1)
  evidence_gate_status=$?
  set -e

  if [[ ${evidence_gate_status} -eq 0 ]]; then
    printf '%s\n' "--- FAIL: ${evidence_gate_name}" >&2
    printf 'evidence mutant survived with exit 0\n' >&2
    return 1
  fi
  printf 'EVIDENCE_NEGATIVE=%s rejected exit=%s\n' \
    "${evidence_gate_name}" "${evidence_gate_status}"
}

bash "${evidence_gate_checker}" "${evidence_gate_source}" >/dev/null

evidence_gate_missing_field="${evidence_gate_tmp}/missing-top-level-field.md"
awk '
  /^artifact_or_environment:/ {
    seen++
    if (seen == 1) {
      next
    }
    if (seen == 2) {
      print
      print
      next
    }
  }
  { print }
  END { if (seen < 2) exit 2 }
' "${evidence_gate_source}" >"${evidence_gate_missing_field}"

evidence_gate_missing_phase="${evidence_gate_tmp}/missing-phase.md"
awk '
  /^phase: writer$/ {
    seen++
    if (seen == 1) {
      next
    }
    if (seen == 2) {
      print
      print
      next
    }
  }
  { print }
  END { if (seen < 2) exit 2 }
' "${evidence_gate_source}" >"${evidence_gate_missing_phase}"

evidence_gate_missing_asset="${evidence_gate_tmp}/missing-external-asset-owner.md"
awk '
  !removed && /^  owner:/ {
    removed = 1
    next
  }
  { print }
  END { if (!removed) exit 2 }
' "${evidence_gate_source}" >"${evidence_gate_missing_asset}"

evidence_gate_missing_invalidation="${evidence_gate_tmp}/missing-old-receipt-invalidation.md"
awk '
  !removed && /^- All writer review and detached receipts bound to the invalidated SHA are void; it must not be integrated\./ {
    removed = 1
    next
  }
  { print }
  END { if (!removed) exit 2 }
' "${evidence_gate_source}" >"${evidence_gate_missing_invalidation}"

evidence_gate_missing_replacement_invalidation="${evidence_gate_tmp}/missing-replacement-receipt-invalidation.md"
awk '
  !removed && /^- Its Spec 0-finding receipt and all writer\/review receipts are void; it must not be integrated\./ {
    removed = 1
    next
  }
  { print }
  END { if (!removed) exit 2 }
' "${evidence_gate_source}" >"${evidence_gate_missing_replacement_invalidation}"

expect_evidence_rejection EvidenceMissingTopLevelField "${evidence_gate_missing_field}"
expect_evidence_rejection EvidenceMissingPhase "${evidence_gate_missing_phase}"
expect_evidence_rejection EvidenceMissingExternalAssetOwner "${evidence_gate_missing_asset}"
expect_evidence_rejection EvidenceMissingOldReceiptInvalidation "${evidence_gate_missing_invalidation}"
expect_evidence_rejection EvidenceMissingReplacementReceiptInvalidation "${evidence_gate_missing_replacement_invalidation}"

printf 'EVIDENCE_FAILURE_SHIELD=PASS negatives=5\n'
