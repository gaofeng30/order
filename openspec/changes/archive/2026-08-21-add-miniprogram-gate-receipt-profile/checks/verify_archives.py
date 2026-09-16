#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import re
import subprocess
from typing import Optional


FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
GOVERNANCE_CHANGE = "enforce-miniprogram-user-regression-gate"
GOVERNANCE_ACTIVE = f"openspec/changes/{GOVERNANCE_CHANGE}"
GOVERNANCE_ARCHIVE = (
    "openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate"
)
PROFILE_CHANGE = "add-miniprogram-gate-receipt-profile"
PROFILE_ACTIVE = f"openspec/changes/{PROFILE_CHANGE}"
PROFILE_ARCHIVE_RE = re.compile(
    rf"^openspec/changes/archive/[0-9]{{4}}-[0-9]{{2}}-[0-9]{{2}}-{PROFILE_CHANGE}$"
)
PROFILE_CANONICAL = "openspec/specs/miniprogram-gate-lifecycle-profile/spec.md"
PROFILE_EXPECTED = (
    f"{PROFILE_ACTIVE}/checks/expected-canonical-miniprogram-gate-lifecycle-profile-spec.md"
)
GOVERNANCE_CANONICAL = {
    "openspec/specs/change-quality-gates/spec.md": (
        "M",
        f"{PROFILE_ACTIVE}/checks/expected-governance-change-quality-gates-spec.md",
    ),
    "openspec/specs/loop-engineering-control-plane/spec.md": (
        "M",
        f"{PROFILE_ACTIVE}/checks/expected-governance-loop-engineering-control-plane-spec.md",
    ),
    "openspec/specs/miniprogram-user-regression-gate/spec.md": (
        "A",
        f"{PROFILE_ACTIVE}/checks/expected-governance-miniprogram-user-regression-gate-spec.md",
    ),
}
GOVERNANCE_PROTECTED = (
    ".agents/skills/order-run-loop/SKILL.md",
    ".agents/skills/order-run-loop/references/self-evolution.md",
    "tools/lifecycle-receipts",
    "apps",
    "services",
    "docs/product/online-ordering-system-prd-0818.md",
)
PROFILE_PROTECTED = (
    ".agents/skills/order-run-loop/SKILL.md",
    ".agents/skills/order-run-loop/references/self-evolution.md",
    "tools/lifecycle-receipts/mechanical-profiles-v1.json",
    "tools/lifecycle-receipts/mechanical-bindings-v1.json",
    "tools/lifecycle-receipts/profile_runner.py",
    "tools/lifecycle-receipts/profiles/miniprogram_user_regression_gate.py",
    "tools/lifecycle-receipts/tests/test_miniprogram_user_regression_profile.py",
    "tools/lifecycle-receipts/verify_receipt.py",
    "tools/miniprogram_gate.py",
    "tools/harness",
    "apps",
    "services",
    "docs/product/online-ordering-system-prd-0818.md",
)


class ArchiveViolation(ValueError):
    pass


def fail(message: str) -> None:
    raise ArchiveViolation(message)


def git(repo: Path, *args: str, binary: bool = False) -> bytes | str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=not binary,
        timeout=30,
    )
    if result.returncode != 0:
        stderr = result.stderr if not binary else result.stderr.decode("utf-8", "replace")
        stdout = result.stdout if not binary else result.stdout.decode("utf-8", "replace")
        fail(f"git {' '.join(args)} failed: {(stderr or stdout).strip()}")
    return result.stdout


def require_commit(repo: Path, sha: str, label: str) -> None:
    if FULL_SHA.fullmatch(sha) is None:
        fail(f"{label} must be a full SHA")
    resolved = str(git(repo, "rev-parse", "--verify", f"{sha}^{{commit}}")).strip()
    if resolved != sha:
        fail(f"{label} is not an exact commit")


def blob_bytes(repo: Path, revision: str, path: str) -> bytes:
    value = git(repo, "cat-file", "blob", f"{revision}:{path}", binary=True)
    assert isinstance(value, bytes)
    return value


def object_id(repo: Path, revision: str, path: str) -> str:
    value = str(git(repo, "rev-parse", f"{revision}:{path}")).strip()
    if FULL_SHA.fullmatch(value) is None:
        fail(f"invalid object at {revision}:{path}")
    return value


def authority_bytes(repo: Path, authority_candidate: Optional[str], path: str) -> bytes:
    if authority_candidate is not None:
        require_commit(repo, authority_candidate, "authority candidate")
        return blob_bytes(repo, authority_candidate, path)
    source = repo / path
    if source.is_symlink() or not source.is_file():
        fail(f"working authority fixture is missing or unsafe: {path}")
    return source.read_bytes()


def active_paths(repo: Path, revision: str, root: str) -> list[str]:
    output = str(git(repo, "ls-tree", "-r", "--name-only", revision, "--", root))
    paths = [line for line in output.splitlines() if line]
    if not paths:
        fail(f"active change is missing at {revision}:{root}")
    return paths


def diff_rows(repo: Path, before: str, after: str) -> list[str]:
    output = str(
        git(repo, "diff", "--find-renames=100%", "--name-status", before, after)
    )
    return [line for line in output.splitlines() if line]


def require_single_parent(repo: Path, before: str, after: str) -> None:
    parents = str(git(repo, "rev-list", "--parents", "-n", "1", after)).strip()
    if parents != f"{after} {before}":
        fail("archive parent is not exact candidate")


def require_expected_rows(actual: list[str], expected: set[str], label: str) -> None:
    if len(actual) != len(expected) or set(actual) != expected:
        fail(f"{label} diff is not the exact archive move and canonical update")


def require_protected_equal(
    repo: Path, before: str, after: str, paths: tuple[str, ...], label: str
) -> None:
    for path in paths:
        if object_id(repo, before, path) != object_id(repo, after, path):
            fail(f"{label} protected object changed: {path}")


def validate_governance_archive(
    repo: Path,
    *,
    authority_candidate: Optional[str],
    target: str,
    archive: str,
) -> str:
    repo = repo.resolve()
    require_commit(repo, target, "governance target")
    require_commit(repo, archive, "governance archive")
    require_single_parent(repo, target, archive)
    sources = active_paths(repo, target, GOVERNANCE_ACTIVE)
    expected = {
        f"R100\t{source}\t{GOVERNANCE_ARCHIVE}/{source.removeprefix(GOVERNANCE_ACTIVE + '/')}"
        for source in sources
    }
    for canonical, (status, _) in GOVERNANCE_CANONICAL.items():
        expected.add(f"{status}\t{canonical}")
    require_expected_rows(diff_rows(repo, target, archive), expected, "governance archive")
    for canonical, (_, fixture) in GOVERNANCE_CANONICAL.items():
        if blob_bytes(repo, archive, canonical) != authority_bytes(
            repo, authority_candidate, fixture
        ):
            fail(f"governance canonical bytes mismatch: {canonical}")
    require_protected_equal(
        repo, target, archive, GOVERNANCE_PROTECTED, "governance archive"
    )
    return GOVERNANCE_ARCHIVE.rsplit("/", 1)[1]


def validate_profile_archive(
    repo: Path,
    *,
    authority_candidate: str,
    candidate: str,
    archive: str,
) -> str:
    repo = repo.resolve()
    require_commit(repo, authority_candidate, "authority candidate")
    require_commit(repo, candidate, "profile candidate")
    require_commit(repo, archive, "profile archive")
    if authority_candidate != candidate:
        fail("profile archive authority must be exact candidate")
    require_single_parent(repo, candidate, archive)
    sources = active_paths(repo, candidate, PROFILE_ACTIVE)
    rows = diff_rows(repo, candidate, archive)
    archive_roots: set[str] = set()
    for row in rows:
        fields = row.split("\t")
        if len(fields) != 3 or fields[0] != "R100" or fields[1] not in sources:
            continue
        relative = fields[1].removeprefix(PROFILE_ACTIVE + "/")
        suffix = "/" + relative
        if fields[2].endswith(suffix):
            root = fields[2][: -len(suffix)]
            if PROFILE_ARCHIVE_RE.fullmatch(root):
                archive_roots.add(root)
    if len(archive_roots) != 1:
        fail("profile archive move does not resolve one dated directory")
    archive_root = next(iter(archive_roots))
    expected = {
        f"R100\t{source}\t{archive_root}/{source.removeprefix(PROFILE_ACTIVE + '/')}"
        for source in sources
    }
    expected.add(f"A\t{PROFILE_CANONICAL}")
    require_expected_rows(rows, expected, "profile archive")
    if blob_bytes(repo, archive, PROFILE_CANONICAL) != authority_bytes(
        repo, authority_candidate, PROFILE_EXPECTED
    ):
        fail("profile canonical bytes mismatch")
    require_protected_equal(repo, candidate, archive, PROFILE_PROTECTED, "profile archive")
    return archive_root.rsplit("/", 1)[1]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=("governance", "profile"), required=True)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--authority-candidate", required=True)
    parser.add_argument("--target", required=True)
    parser.add_argument("--archive", required=True)
    args = parser.parse_args()
    try:
        if args.mode == "governance":
            validate_governance_archive(
                Path(args.repo),
                authority_candidate=args.authority_candidate,
                target=args.target,
                archive=args.archive,
            )
            print("governance-archive=PASS")
        else:
            validate_profile_archive(
                Path(args.repo),
                authority_candidate=args.authority_candidate,
                candidate=args.target,
                archive=args.archive,
            )
            print("profile-archive=PASS")
        return 0
    except ArchiveViolation as exc:
        print(str(exc), file=__import__("sys").stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
