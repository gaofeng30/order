#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
EXPECTED = {
    "enable-run-loop-self-evolution": (
        "VERIFIED",
        "7a5e8bb261b994d68ce9af5eada347df6700c490",
        "94e04bf26e37e93299c26ef2c9c8aa7552619444",
    ),
    "connect-miniprogram-menu-catalog": (
        "FAILED",
        "6d77bdd6319722b7c71b4726c6159955da9a84b6",
        "7d01fe22ded67aeded78cb7d03de87aa12416ada",
    ),
    "supersede-miniprogram-catalog-evidence": (
        "VERIFIED_SUPERSESSION",
        "109c8e828f6f5a10adff33ccdb73d4fd784b2f3d",
        "510a84baddfb5b61200c159fd2041b7c512a92db",
    ),
}
EXPECTED_ROUTES = {
    "DRAFT": "$order-plan-change",
    "APPROVED": "$order-implement-tdd",
    "IMPLEMENTING": "$order-implement-tdd",
    "CANDIDATE": "$order-verify-change",
    "INDEPENDENT_VERIFIED": "$order-integrate-change",
    "INTEGRATED": "$order-integrate-change",
}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def run(command: list[str], label: str) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        command,
        check=False,
        capture_output=True,
        text=True,
    )
    require(
        result.returncode == 0,
        f"{label} failed: {(result.stderr or result.stdout).strip()}",
    )
    return result


def resolve_repo(path: str) -> Path:
    requested = Path(path).resolve()
    require(requested.is_dir(), "repo path is not a directory")
    result = run(
        ["git", "-C", str(requested), "rev-parse", "--show-toplevel"],
        "resolve git repository",
    )
    repo = Path(result.stdout.strip()).resolve()
    require(repo.is_dir(), "resolved git repository is not a directory")
    return repo


def require_repo_state(repo: Path, candidate_sha: str, phase: str) -> None:
    commit = subprocess.run(
        ["git", "-C", str(repo), "cat-file", "-e", f"{candidate_sha}^{{commit}}"],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    require(commit.returncode == 0, "candidate SHA does not resolve to a commit")
    head = run(["git", "-C", str(repo), "rev-parse", "HEAD"], f"read {phase} HEAD")
    require(head.stdout.strip() == candidate_sha, f"{phase} HEAD is not the exact candidate")
    symbolic = subprocess.run(
        ["git", "-C", str(repo), "symbolic-ref", "-q", "HEAD"],
        check=False,
        capture_output=True,
        text=True,
    )
    require(symbolic.returncode == 1, f"{phase} HEAD must be detached")
    status = run(
        ["git", "-C", str(repo), "status", "--porcelain=v1", "--untracked-files=all"],
        f"read {phase} repository status",
    )
    require(not status.stdout, f"{phase} repository must be clean")


def parse_object(raw: str, label: str) -> dict[str, object]:
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"{label} is not valid JSON: {exc.msg}") from exc
    require(isinstance(payload, dict), f"{label} must be a JSON object")
    return payload


def run_receipt_checker(repo: Path, *selection: str) -> dict[str, object]:
    checker = repo / "tools/lifecycle-receipts/verify_receipt.py"
    require(checker.is_file(), "repo-local receipt checker is missing")
    for relative in (
        "tools/lifecycle-receipts/receipt-schema-v1.json",
        "tools/lifecycle-receipts/profile_runner.py",
        "tools/lifecycle-receipts/mechanical-profiles-v1.json",
        "tools/lifecycle-receipts/mechanical-bindings-v1.json",
    ):
        path = repo / relative
        require(path.is_file() and not path.is_symlink(), f"controlled checker input missing: {relative}")
    result = run(
        [sys.executable, str(checker), "--repo", str(repo), *selection, "--json"],
        f"receipt checker {' '.join(selection)}",
    )
    return parse_object(result.stdout, f"receipt checker {' '.join(selection)} output")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--candidate-sha", required=True)
    parser.add_argument("result")
    args = parser.parse_args()
    require(SHA_RE.fullmatch(args.candidate_sha) is not None, "candidate SHA is not exact")
    repo = resolve_repo(args.repo)
    require_repo_state(repo, args.candidate_sha, "before")
    result = parse_object(Path(args.result).read_text(encoding="utf-8"), "forward result")
    require(result.get("candidate_sha") == args.candidate_sha, "result candidate_sha mismatch")
    require(
        result.get("verdict") == "MECHANICAL_EVIDENCE_ONLY",
        "fresh-session result must not claim lifecycle or independent PASS",
    )
    require(result.get("repository_clean_before") is True, "repository not clean before")
    require(result.get("repository_clean_after") is True, "repository not clean after")
    require(result.get("invalid_fixture_verdict") == "NO-GO", "invalid fixture must be NO-GO")
    require(result.get("receipt_head_result_persisted") is False, "receipt-head result was persisted")
    require(result.get("routes") == EXPECTED_ROUTES, "stage routes drifted")
    receipts = result.get("receipts")
    require(isinstance(receipts, list) and len(receipts) == 3, "three receipts required")
    by_name = {item.get("change_name"): item for item in receipts if isinstance(item, dict)}
    require(len(by_name) == len(receipts) and set(by_name) == set(EXPECTED), "receipt enumeration mismatch")
    checked_self = run_receipt_checker(repo, "--change", "enable-run-loop-self-evolution")
    checked_delivery = run_receipt_checker(repo, "--chain")
    checked_chain_receipts = checked_delivery.get("receipts")
    require(
        isinstance(checked_chain_receipts, list) and len(checked_chain_receipts) == 2,
        "controlled chain must return two receipt histories",
    )
    checked_by_name = {checked_self["change_name"]: checked_self}
    for item in checked_chain_receipts:
        require(isinstance(item, dict) and isinstance(item.get("change_name"), str), "chain receipt invalid")
        checked_by_name[item["change_name"]] = item
    for name, (verdict, candidate, archive) in EXPECTED.items():
        receipt = checked_by_name[name]
        require(by_name[name] == receipt, f"{name} session result does not match repo checker")
        require(receipt.get("expected_historical_verdict") == verdict, f"{name} verdict mismatch")
        require(receipt.get("candidate_sha") == candidate, f"{name} candidate mismatch")
        require(receipt.get("integrated_sha") == candidate, f"{name} integrated mismatch")
        require(receipt.get("archive_sha") == archive, f"{name} archive mismatch")
        require(SHA_RE.fullmatch(str(receipt.get("receipt_head"))) is not None, f"{name} receipt-head invalid")
        require(receipt.get("receipt_head_verification") == "PASS_DERIVED", f"{name} derived Gate missing")
        required_mechanical = "EXPECTED_MECHANICAL_FAIL" if verdict == "FAILED" else "MECHANICAL_PASS"
        require(
            receipt.get("mechanical_verification") == required_mechanical,
            f"{name} mechanical result mismatch",
        )
        require(
            receipt.get("actor_independence") == "NOT_PROVEN_BY_MECHANICAL_REPLAY",
            f"{name} replay falsely claims actor independence",
        )
        attestation = receipt.get("recorded_attestation")
        require(
            isinstance(attestation, dict)
            and attestation.get("trust") == "UNTRUSTED_FOR_MECHANICAL_PASS",
            f"{name} recorded attestation trust mismatch",
        )
    delivery = result.get("delivery")
    require(delivery == checked_delivery, "session delivery result does not match repo checker")
    require(checked_delivery.get("mechanically_reproducible") is True, "delivery chain is not reproducible")
    require(checked_delivery.get("historical_change_verdict") == "FAILED", "historical verdict changed")
    require(
        checked_delivery.get("actor_independence") == "NOT_PROVEN_BY_MECHANICAL_REPLAY",
        "delivery replay falsely claims actor independence",
    )
    require_repo_state(repo, args.candidate_sha, "after")
    print("fresh-session forward result PASS")


if __name__ == "__main__":
    main()
