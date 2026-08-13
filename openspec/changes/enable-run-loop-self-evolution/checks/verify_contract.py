#!/usr/bin/env python3
"""Verify the thin order-run-loop package and its stable control contracts."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


class ContractError(RuntimeError):
    """Raised for the first decisive contract mismatch."""


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ContractError(message)


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as exc:
        raise ContractError(f"cannot read {path}: {exc}") from exc


def parse_frontmatter(skill_text: str) -> dict[str, str]:
    match = re.match(r"\A---\n(.*?)\n---\n", skill_text, re.DOTALL)
    require(match is not None, "SKILL.md front matter is missing or malformed")
    fields: dict[str, str] = {}
    for line in match.group(1).splitlines():
        key, separator, value = line.partition(":")
        require(bool(separator) and bool(key.strip()), f"invalid front matter line: {line!r}")
        normalized = key.strip()
        require(normalized not in fields, f"duplicate front matter field: {normalized}")
        fields[normalized] = value.strip()
    return fields


def parse_interface(yaml_text: str) -> dict[str, str]:
    fields: dict[str, str] = {}
    for line in yaml_text.splitlines():
        match = re.fullmatch(r'  ([a-z_]+): "([^"]*)"', line)
        if match:
            fields[match.group(1)] = match.group(2)
    return fields


def verify(repo: Path) -> None:
    repo = repo.resolve()
    skill_dir = repo / ".agents/skills/order-run-loop"
    expected_files = [
        "SKILL.md",
        "agents/openai.yaml",
        "references/self-evolution.md",
    ]
    actual_files = sorted(
        path.relative_to(skill_dir).as_posix()
        for path in skill_dir.rglob("*")
        if path.is_file()
    )
    require(
        actual_files == expected_files,
        f"skill package files mismatch: expected {expected_files}, got {actual_files}",
    )

    skill_text = read_text(skill_dir / "SKILL.md")
    reference_text = read_text(skill_dir / "references/self-evolution.md")
    interface_text = read_text(skill_dir / "agents/openai.yaml")

    frontmatter = parse_frontmatter(skill_text)
    require(set(frontmatter) == {"name", "description"}, "front matter fields drifted")
    require(frontmatter["name"] == skill_dir.name, "Skill name and directory disagree")
    require("Use when" in frontmatter["description"], "description is not trigger-oriented")

    interface = parse_interface(interface_text)
    expected_interface = {
        "display_name": "Order Run Loop",
        "short_description": "有界调度 OpenSpec change 并以证据闭环长任务",
        "default_prompt": "Use $order-run-loop to coordinate this goal through bounded OpenSpec change lanes.",
    }
    require(interface == expected_interface, f"agents/openai.yaml metadata drifted: {interface}")
    require("$order-run-loop" in interface["default_prompt"], "default prompt misses $order-run-loop")

    expected_routes = [
        "Route planning and OpenSpec refinement to `$order-plan-change`.",
        "Route approved implementation through candidate creation to `$order-implement-tdd`.",
        "Route read-only verification of an exact candidate SHA to `$order-verify-change`.",
        "Route authorized integration and archive handling to `$order-integrate-change`.",
    ]
    for route in expected_routes:
        require(skill_text.count(route) == 1, f"legacy route drifted: {route}")

    expected_transitions = [
        "queue → `DRAFT`",
        "`DRAFT` → `APPROVED`",
        "`APPROVED` → `IMPLEMENTING`",
        "`IMPLEMENTING` → `CANDIDATE`",
        "`CANDIDATE` → `INDEPENDENT_VERIFIED`",
        "`INDEPENDENT_VERIFIED` → `INTEGRATED`",
        "`INTEGRATED` → `ARCHIVED`",
    ]
    for transition in expected_transitions:
        require(skill_text.count(transition) == 1, f"legacy lifecycle transition drifted: {transition}")

    legacy_tokens = [
        "Allow at most two active change lane slots",
        "third consecutive occurrence of the same fingerprint",
        "do not make a fourth blind attempt",
        "`BLOCKED_EXTERNAL`",
        "Score >= 85 AND OPEN(P0/P1) = 0 AND first_blocking_openspec_strict = PASS",
        "Return `NO-GO` when any hard Gate fails",
        "Invalidate independent verification after any code, spec, task, rebase, merge, or SHA change.",
    ]
    for token in legacy_tokens:
        require(token in skill_text, f"legacy control invariant drifted: {token}")

    expected_weights = {
        "Product decisions and scope": 25,
        "Cross-client truth source and state": 15,
        "Money, inventory, and authorization": 15,
        "API and executable acceptance": 15,
        "Architecture, data, and recovery": 15,
        "Quality Gates and independent verification": 10,
        "External dependency governance": 5,
    }
    observed_weights: dict[str, int] = {}
    for label, value in re.findall(r"^\| ([^|]+?) \| (\d+) \|$", skill_text, re.MULTILINE):
        if label in expected_weights:
            observed_weights[label] = int(value)
    require(observed_weights == expected_weights, f"readiness weights drifted: {observed_weights}")
    require(sum(observed_weights.values()) == 100, "readiness weights no longer total 100")

    require(
        "[self-evolution protocol](references/self-evolution.md)" in skill_text,
        "SKILL.md does not link the one-level self-evolution reference",
    )
    skill_evolution_tokens = [
        "Freeze the runner base before each module",
        "Keep runner evolution observation-only during the module",
        "Activate an integrated rule only for the next module",
    ]
    for token in skill_evolution_tokens:
        require(token in skill_text, f"thin Skill entry is missing: {token}")

    reference_tokens = [
        "repo_sha",
        "skill_blob",
        "skill_sha256",
        "explicit unversioned marker",
        "`candidate`",
        "`environment`",
        "`checker`",
        "`external`",
        "DRAFT admission is not promotion",
        "non-weakening intent",
        "regression plan",
        "forward-test plan",
        "implemented regression PASS",
        "clean detached exact-SHA",
        "independent verification PASS",
        "local main",
        "next module",
        "zero-match",
        "Markdown field",
        "literal backticks",
        "explicit case semantics",
        "bounded health polling",
        "safe temporary directories",
        "archive trailing newline",
    ]
    for token in reference_tokens:
        require(token in reference_text, f"self-evolution reference is missing: {token}")

    forbidden_headings = [
        "# Plan an Order Change",
        "# Implement an Order Change with TDD",
        "# Verify an Order Change",
        "# Integrate an Order Change",
    ]
    combined = f"{skill_text}\n{reference_text}"
    for heading in forbidden_headings:
        require(heading not in combined, f"stage procedure heading was copied: {heading}")

    require(len(skill_text.splitlines()) <= 160, "SKILL.md is no longer thin (over 160 lines)")
    require(len(reference_text.splitlines()) <= 200, "self-evolution reference exceeds 200 lines")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".", type=Path)
    args = parser.parse_args()
    try:
        verify(args.repo)
    except ContractError as exc:
        print(f"contract_check=FAIL: {exc}", file=sys.stderr)
        return 1
    print("contract_check=PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
