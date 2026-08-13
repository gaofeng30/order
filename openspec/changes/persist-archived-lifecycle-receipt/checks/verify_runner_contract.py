#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import sys


def fail(message: str) -> None:
    raise SystemExit(message)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".")
    args = parser.parse_args()
    repo = Path(args.repo).resolve()
    skill = (repo / ".agents/skills/order-run-loop/SKILL.md").read_text(encoding="utf-8")
    reference = (
        repo / ".agents/skills/order-run-loop/references/self-evolution.md"
    ).read_text(encoding="utf-8")

    skill_tokens = [
        "lifecycle-receipt.md",
        "tools/lifecycle-receipts/verify_receipt.py",
        "REQUIRED_DERIVED",
        "receipt-head",
        "repository-controlled exact-SHA profile registry",
        "Do not let a business module define or edit its own judge",
        "binding-head PASS",
    ]
    reference_tokens = [
        "lifecycle receipt",
        "FAILED",
        "superseded_by",
        "never write",
        "UNTRUSTED_FOR_MECHANICAL_PASS",
        "EXPECTED_MECHANICAL_FAIL",
        "mechanically_reproducible=true",
        "cannot edit its judge",
        "candidate, binding-head and receipt-head independent verifiers",
    ]
    for token in skill_tokens:
        if token not in skill:
            fail(f"runner receipt contract missing: {token}")
    for token in reference_tokens:
        if token not in reference:
            fail(f"runner recovery contract missing: {token}")

    legacy_tokens = [
        "$order-plan-change",
        "$order-implement-tdd",
        "$order-verify-change",
        "$order-integrate-change",
        "| queue → `DRAFT` |",
        "| `DRAFT` → `APPROVED` |",
        "| `APPROVED` → `IMPLEMENTING` |",
        "| `IMPLEMENTING` → `CANDIDATE` |",
        "| `CANDIDATE` → `INDEPENDENT_VERIFIED` |",
        "| `INDEPENDENT_VERIFIED` → `INTEGRATED` |",
        "| `INTEGRATED` → `ARCHIVED` |",
        "at most two active change lane slots",
        "third consecutive occurrence",
        "BLOCKED_EXTERNAL",
        "Product decisions and scope | 25",
        "Quality Gates and independent verification | 10",
        "Score >= 85 AND OPEN(P0/P1) = 0",
    ]
    for token in legacy_tokens:
        if token not in skill:
            fail(f"legacy invariant missing: {token}")
    forbidden = [
        "receipt_head_verification=PASS",
        "receipt-head PASS back",
        "candidate_verification=PASS",
        "verifier_provenance=PASS",
    ]
    for token in forbidden:
        if token in skill or token in reference:
            fail(f"derived receipt result must not be persisted: {token}")
    print("runner receipt contract PASS")


if __name__ == "__main__":
    try:
        main()
    except (OSError, UnicodeError) as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
