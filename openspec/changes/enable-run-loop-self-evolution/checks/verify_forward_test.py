#!/usr/bin/env python3
"""Validate structured output from the independent fresh-session forward-test."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path


class ForwardTestError(RuntimeError):
    pass


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ForwardTestError(message)


def validate(result: dict, candidate_sha: str) -> None:
    require(re.fullmatch(r"[0-9a-f]{40}", candidate_sha) is not None, "candidate SHA is not full lowercase hex")
    require(result.get("candidate_sha") == candidate_sha, "result candidate_sha does not match exact input SHA")
    require(result.get("repository_clean") is True, "repository_clean must be true")
    require(result.get("current_module_rule_changed") is False, "current_module_rule_changed must be false")

    expected_files = [
        "AGENTS.md",
        ".agents/skills/order-run-loop/SKILL.md",
        ".agents/skills/order-run-loop/references/self-evolution.md",
        "openspec/changes/enable-run-loop-self-evolution/forward-test.md",
    ]
    require(result.get("read_files") == expected_files, "minimal read_files boundary drifted")

    base = result.get("runner_base")
    require(isinstance(base, dict), "runner_base must be an object")
    require(base.get("repo_sha") == candidate_sha, "runner_base.repo_sha must equal candidate SHA")
    require(re.fullmatch(r"[0-9a-f]{40}", str(base.get("skill_blob", ""))) is not None, "skill_blob is invalid")
    require(re.fullmatch(r"[0-9a-f]{64}", str(base.get("skill_sha256", ""))) is not None, "skill_sha256 is invalid")
    require(base.get("runner_version") in {"explicit", "unversioned"}, "runner_version marker is invalid")

    observations = result.get("observations")
    require(isinstance(observations, list), "observations must be an array")
    classes = [item.get("class") for item in observations if isinstance(item, dict)]
    require(classes == ["candidate", "environment", "checker", "external"], "observation classes/order drifted")
    require(all(item.get("current_module_applied") is False for item in observations), "an observation changed the current module")
    checker = observations[2]
    require(checker.get("derived_candidate_id") not in {None, ""}, "checker observation lacks a derived candidate link")

    require(result.get("incomplete_screen") == "stay_observation", "incomplete DRAFT screen was not rejected")
    require(
        result.get("complete_screen") == "draft_admissible_not_promoted",
        "complete DRAFT screen was confused with promotion",
    )
    require(result.get("verified_outside_main") == "inactive", "verified-only rule became active")
    require(
        result.get("integrated_local_main") == "activate_next_module_only",
        "local-main/next-module activation drifted",
    )

    expected_routes = {
        "DRAFT": "$order-plan-change",
        "APPROVED": "$order-implement-tdd",
        "CANDIDATE": "$order-verify-change",
        "INDEPENDENT_VERIFIED": "$order-integrate-change",
    }
    require(result.get("routes") == expected_routes, "legacy stage routes drifted")
    require(result.get("hard_gates_weakened") is False, "a hard Gate was weakened")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--candidate-sha", required=True)
    parser.add_argument("result", type=Path)
    args = parser.parse_args()
    try:
        result = json.loads(args.result.read_text(encoding="utf-8"))
        require(isinstance(result, dict), "result must be a JSON object")
        validate(result, args.candidate_sha)
    except Exception as exc:
        print(f"forward_test_validation=FAIL: {exc}", file=sys.stderr)
        return 1
    print("forward_test_validation=PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
