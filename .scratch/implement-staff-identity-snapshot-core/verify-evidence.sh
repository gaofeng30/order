#!/usr/bin/env bash
set -euo pipefail

evidence_file=${1:-.scratch/implement-staff-identity-snapshot-core/tasks.md}

awk '
  BEGIN {
    development_base = "f3c4efa4cd665652d93d5da76f92d18c4bdc59ac"
    candidate_base = "19ca1e46e106f293070f0cdf820951e31107cba6"
  }
  function fail(message) {
    if (!failed) print "evidence error: " message > "/dev/stderr"
    failed = 1
    exit 65
  }
  function reset_record() {
    task_id = change = gate = ui_target = ui_actual = base_sha = phase = command = exit_result = expected = observed = summary = exact_sha = exact_type = candidate_sha = artifact = boundary = asset = owner = missing = recovery = ""
  }
  function finish_record() {
    if (!in_record) return
    records++
    if (task_id == "" || change == "" || gate == "" || ui_target == "" || ui_actual == "" || base_sha == "" || phase == "" || command == "" || exit_result == "" || expected == "" || observed == "" || summary == "" || exact_sha == "" || exact_type == "" || candidate_sha == "" || artifact == "" || boundary == "" || asset == "" || owner == "" || missing == "" || recovery == "") {
      fail("record " records " is missing a required field")
    }
    if (change != "implement-staff-identity-snapshot-core" || gate != "W3" || ui_target != "UI0" || ui_actual != "UI0") fail("record " records " has mismatched change/gate/UI")
    if (phase !~ /^(red|green|refactor|writer|verifier|integration)$/) fail("record " records " has invalid phase")
    if (phase ~ /^(red|green)$/) {
      if (base_sha != development_base) fail("record " records " historical Red/Green base_sha mismatch")
    } else {
      if (base_sha != candidate_base) fail("record " records " candidate-lifecycle base_sha mismatch")
    }
    if (exit_result !~ /^(exit-[0-9]+|PASS|FAIL|BLOCKED_EXTERNAL|N\/A)$/) fail("record " records " has non-scalar exit_result")
    if (exact_sha !~ /^[0-9a-f]+$/ || length(exact_sha) != 40 || exact_sha ~ /^0+$/) fail("record " records " has invalid exact_sha")
    if (exact_type != "tree") fail("record " records " exact_object_type must be tree")
    object_command = "git cat-file -t " exact_sha " 2>/dev/null"
    object_type = ""
    if ((object_command | getline object_type) != 1) {
      close(object_command)
      fail("record " records " exact_sha does not exist")
    }
    object_status = close(object_command)
    if (object_status != 0) fail("record " records " exact_sha lookup failed")
    if (object_type != exact_type) fail("record " records " object type is " object_type ", want " exact_type)
    if (candidate_sha != "NOT_CREATED" && (candidate_sha !~ /^[0-9a-f]+$/ || length(candidate_sha) != 40)) fail("record " records " has invalid candidate_sha")
    if (candidate_sha == "NOT_CREATED" && (phase == "verifier" || phase == "integration")) fail("record " records " claims verifier/integration before candidate creation")
    if (command ~ /<[^>]+>|TODO|TBD|PLACEHOLDER/) fail("record " records " has placeholder command/action")
    evidence_for[task_id]++
    in_record = 0
    reset_record()
  }
  /^- \[x\] `SI-[0-9][0-9]`/ {
    match($0, /`SI-[0-9][0-9]`/)
    completed = substr($0, RSTART + 1, 5)
    completed_task[completed] = 1
  }
  /^```yaml$/ {
    if (in_record) fail("nested yaml record")
    in_record = 1
    reset_record()
    next
  }
  /^```$/ && in_record {
    finish_record()
    next
  }
  in_record {
    if ($0 ~ /^task_id: /) { task_id = substr($0, 10); next }
    if ($0 ~ /^change: /) { change = substr($0, 9); next }
    if ($0 ~ /^gate_type: /) { gate = substr($0, 12); next }
    if ($0 ~ /^ui_level_target: /) { ui_target = substr($0, 18); next }
    if ($0 ~ /^ui_level_actual: /) { ui_actual = substr($0, 18); next }
    if ($0 ~ /^base_sha: /) { base_sha = substr($0, 11); next }
    if ($0 ~ /^phase: /) { phase = substr($0, 8); next }
    if ($0 ~ /^command_or_action: /) { command = substr($0, 20); next }
    if ($0 ~ /^exit_result: /) { exit_result = substr($0, 14); next }
    if ($0 ~ /^expected: /) { expected = substr($0, 11); next }
    if ($0 ~ /^observed: /) { observed = substr($0, 11); next }
    if ($0 ~ /^sanitized_summary: /) { summary = substr($0, 20); next }
    if ($0 ~ /^exact_sha: /) { exact_sha = substr($0, 12); next }
    if ($0 ~ /^exact_object_type: /) { exact_type = substr($0, 20); next }
    if ($0 ~ /^candidate_sha: /) { candidate_sha = substr($0, 16); next }
    if ($0 ~ /^artifact_or_environment: /) { artifact = substr($0, 26); next }
    if ($0 ~ /^unverified_boundary: /) { boundary = substr($0, 22); next }
    if ($0 ~ /^external_asset:$/) { asset = "present"; next }
    if ($0 ~ /^  owner: /) { owner = substr($0, 10); next }
    if ($0 ~ /^  missing: /) { missing = substr($0, 12); next }
    if ($0 ~ /^  recovery: /) { recovery = substr($0, 13); next }
  }
  END {
    if (failed) exit 65
    if (in_record) fail("unterminated yaml record")
    if (records == 0) fail("no evidence records")
    for (task in completed_task) {
      if (!(task in evidence_for)) fail("completed task " task " has no evidence")
    }
    print "EVIDENCE_LEDGER=PASS records=" records " completed_tasks=" length(completed_task)
  }
' "${evidence_file}"
