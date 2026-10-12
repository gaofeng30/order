#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import sys
from typing import Any, Callable


EXACT_CHANGE = "enforce-miniprogram-user-regression-gate"
EXACT_CANDIDATE = "f5719e98690d0b1301ed567c8b616e074846b445"
EXACT_ARCHIVE = "b53cb520c4cac0505803a9d1e6dcc1807ac34540"
EXACT_ARCHIVE_PATH = (
    "openspec/changes/archive/2026-08-21-enforce-miniprogram-user-regression-gate"
)
EXACT_CHECKPOINT_SHA256 = (
    "535a53c39b1177a795422acd6cad85ebf9c60ac5a545ff169306b6c674a564e3"
)
EXACT_TASKS_SHA256 = (
    "da0d82d6a66830b19c6375e4cd014b0a0214d7e10a96079c91818c9cf119349e"
)
LEGACY_RECEIPT_RELATIVE = f"{EXACT_ARCHIVE_PATH}/legacy-lifecycle-receipt.md"
STANDARD_RECEIPT_RELATIVE = f"{EXACT_ARCHIVE_PATH}/lifecycle-receipt.md"
COMPATIBILITY_RULE_ID = "exact-f5719e9-pre-runtime-checkpoint-v1"

REPAIR_CHANGE = "admit-legacy-miniprogram-gate-receipt"
REPAIR_ACTIVE_ROOT = f"openspec/changes/{REPAIR_CHANGE}"
REPAIR_ARCHIVE_RE = re.compile(
    rf"^openspec/changes/archive/[0-9]{{4}}-[0-9]{{2}}-[0-9]{{2}}-{REPAIR_CHANGE}$"
)
ADAPTER_RELATIVE = "tools/lifecycle-receipts/verify_legacy_miniprogram_receipt.py"
TEST_RELATIVE = "tools/lifecycle-receipts/tests/test_legacy_miniprogram_receipt.py"
REPAIR_CANONICAL_PATHS = {
    "openspec/specs/legacy-miniprogram-gate-receipt/spec.md",
    "openspec/specs/miniprogram-gate-lifecycle-profile/spec.md",
    "openspec/specs/loop-engineering-control-plane/spec.md",
}

EXPECTED_CONTROL_BLOBS = {
    "tools/lifecycle-receipts/verify_receipt.py": "f7b53f072b8f1f5f0ac6abd4cac65992ac7cca3d",
    "tools/lifecycle-receipts/receipt-schema-v1.json": "c0c9f57993b4a2d8b81a6d9e3aaee4c404f52465",
    "tools/lifecycle-receipts/profile_runner.py": "4dcd6c8be3e46cff968f27d859f354e066bc7538",
    "tools/lifecycle-receipts/mechanical-profiles-v1.json": "c61186f4bfb2b61eeb3799ebcfe29bd98b102793",
    "tools/lifecycle-receipts/mechanical-bindings-v1.json": "b323055fedbc86fcdb72cadc4f40a2aef481156c",
    "tools/lifecycle-receipts/profiles/miniprogram_user_regression_gate.py": "3ee4ef9013a9dfa56cdac4f65fe061de7cb9b4c6",
}


class LegacyReceiptError(ValueError):
    pass


def fail(message: str) -> None:
    raise LegacyReceiptError(message)


def git(repo: Path, *args: str, check: bool = True) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=True,
    )
    if check and result.returncode != 0:
        fail(f"git {' '.join(args)} failed: {(result.stderr or result.stdout).strip()}")
    return result.stdout


def git_success(repo: Path, *args: str) -> bool:
    return subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    ).returncode == 0


def git_blob_id(data: bytes) -> str:
    header = f"blob {len(data)}\0".encode("ascii")
    return hashlib.sha1(header + data).hexdigest()


def assert_exact_control_blobs(
    repo: Path, *, expected: dict[str, str] | None = None
) -> None:
    repo = repo.resolve()
    expected = expected or EXPECTED_CONTROL_BLOBS
    for relative, wanted in expected.items():
        path = repo / relative
        if not path.is_file() or path.is_symlink():
            fail(f"protected control path is not a regular file: {relative}")
        head_blob = git(repo, "rev-parse", f"HEAD:{relative}").strip()
        worktree_blob = git_blob_id(path.read_bytes())
        if head_blob != wanted or worktree_blob != wanted:
            fail(f"protected control blob mismatch: {relative}")


def _checkpoint_table(data: bytes) -> dict[str, str]:
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        fail(f"legacy checkpoint is not UTF-8: {exc}")
    fields: dict[str, str] = {}
    for line in text.splitlines():
        match = re.fullmatch(r"\|\s*([a-z][a-z0-9_]*)\s*\|\s*`?([^|`]*?)`?\s*\|", line)
        if match is None or match.group(1) in {"field"}:
            continue
        name = match.group(1)
        if name in fields:
            fail(f"duplicate legacy checkpoint field: {name}")
        fields[name] = match.group(2).strip()
    return fields


def compatibility_checkpoint_fields(
    data: bytes, standard_parser: Callable[[bytes], dict[str, str]]
) -> tuple[dict[str, str], dict[str, str] | None]:
    if EXACT_CHANGE.encode("utf-8") not in data:
        return standard_parser(data), None
    if hashlib.sha256(data).hexdigest() != EXACT_CHECKPOINT_SHA256:
        fail("legacy checkpoint SHA256 mismatch")
    fields = _checkpoint_table(data)
    expected = {
        "module": EXACT_CHANGE,
        "lifecycle": "IMPLEMENTING",
        "base_sha": "2299da6013c06f4ae7dcad535373c67b66c80ebb",
        "candidate_sha": "none",
    }
    if any(fields.get(name) != value for name, value in expected.items()):
        fail("legacy checkpoint recorded fields mismatch")
    return (
        {
            "state": "CANDIDATE",
            "candidate_sha": EXACT_CANDIDATE,
            "integrated_sha": "none",
            "archive_sha": "none",
        },
        {"state": fields["lifecycle"], "candidate_sha": fields["candidate_sha"]},
    )


def enumerate_with_exact_legacy(
    repo: Path,
    schema: dict[str, Any],
    parse_receipt: Callable[[bytes], dict[str, Any]],
    standard_enumerate: Callable[[Path, dict[str, Any]], list[tuple[Path, dict[str, Any]]]],
) -> list[tuple[Path, dict[str, Any]]]:
    repo = repo.resolve()
    legacy_path = repo / LEGACY_RECEIPT_RELATIVE
    standard_path = repo / STANDARD_RECEIPT_RELATIVE
    found = sorted(
        path
        for path in (repo / "openspec/changes/archive").glob(
            "*/legacy-lifecycle-receipt.md"
        )
        if path.exists() or path.is_symlink()
    )
    if not found:
        fail("missing exact legacy lifecycle receipt")
    if found != [legacy_path]:
        fail("legacy lifecycle receipt path is ambiguous")
    if standard_path.exists() or standard_path.is_symlink():
        fail("standard lifecycle receipt is forbidden for exact legacy change")
    if not legacy_path.is_file() or legacy_path.is_symlink():
        fail("legacy lifecycle receipt must be a regular non-symlink file")
    standard = standard_enumerate(repo, schema)
    if any(values.get("change_name") == EXACT_CHANGE for _, values in standard):
        fail("standard receipt set already contains exact legacy change")
    values = parse_receipt(legacy_path.read_bytes())
    if values.get("change_name") != EXACT_CHANGE:
        fail("legacy receipt change mismatch")
    return [*standard, (legacy_path, values)]


def _load_v1(repo: Path):
    path = repo / "tools/lifecycle-receipts/verify_receipt.py"
    spec = importlib.util.spec_from_file_location("protected_lifecycle_receipt_v1", path)
    if spec is None or spec.loader is None:
        fail("protected v1 verifier cannot be loaded")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    try:
        spec.loader.exec_module(module)
    except Exception as exc:
        sys.modules.pop(spec.name, None)
        fail(f"protected v1 verifier load failed: {exc}")
    return module


def _repair_archive_state(repo: Path) -> tuple[str, str, str]:
    tree = git(
        repo, "ls-tree", "-r", "--name-only", "HEAD", "--", "openspec/changes/archive"
    ).splitlines()
    candidates = [
        path[: -len("/proposal.md")]
        for path in tree
        if path.endswith("/proposal.md")
        and REPAIR_ARCHIVE_RE.fullmatch(path[: -len("/proposal.md")])
    ]
    if len(candidates) != 1:
        fail("repair archive path is missing or ambiguous")
    archive_root = candidates[0]
    proposal_path = f"{archive_root}/proposal.md"
    touches = git(repo, "log", "--format=%H", "--no-renames", "--", proposal_path).splitlines()
    if len(touches) != 1:
        fail("repair archive commit is missing, ambiguous, or later edited")
    archive_sha = touches[0]
    parents = git(repo, "rev-list", "--parents", "-n", "1", archive_sha).split()
    if len(parents) != 2:
        fail("repair archive must have one parent")
    candidate_sha = parents[1]
    if not git_success(repo, "merge-base", "--is-ancestor", archive_sha, "HEAD"):
        fail("repair archive is not an ancestor of current HEAD")

    active_paths = set(
        git(repo, "ls-tree", "-r", "--name-only", candidate_sha, "--", REPAIR_ACTIVE_ROOT).splitlines()
    )
    if not active_paths or f"{REPAIR_ACTIVE_ROOT}/proposal.md" not in active_paths:
        fail("repair candidate active change is missing")
    rows = git(
        repo, "diff-tree", "--no-commit-id", "--name-status", "-r", "-M", archive_sha
    ).splitlines()
    moved: set[str] = set()
    canonical: set[str] = set()
    for row in rows:
        parts = row.split("\t")
        if len(parts) == 3 and parts[0] == "R100":
            source, target = parts[1], parts[2]
            if not source.startswith(REPAIR_ACTIVE_ROOT + "/"):
                fail(f"repair archive contains unexpected rename: {row}")
            relative = source[len(REPAIR_ACTIVE_ROOT) + 1 :]
            if target != f"{archive_root}/{relative}":
                fail(f"repair archive changes relative path: {row}")
            moved.add(source)
            continue
        if len(parts) == 2 and parts[0] in {"A", "M"} and parts[1] in REPAIR_CANONICAL_PATHS:
            canonical.add(parts[1])
            continue
        fail(f"repair archive contains unexpected path: {row}")
    if moved != active_paths or canonical != REPAIR_CANONICAL_PATHS:
        fail("repair archive move or canonical delta mismatch")

    for relative in (ADAPTER_RELATIVE, TEST_RELATIVE):
        candidate_blob = git(repo, "rev-parse", f"{candidate_sha}:{relative}").strip()
        archive_blob = git(repo, "rev-parse", f"{archive_sha}:{relative}").strip()
        current_blob = git(repo, "rev-parse", f"HEAD:{relative}").strip()
        if candidate_blob != archive_blob or archive_blob != current_blob:
            fail(f"repair controlled source drift: {relative}")
    return candidate_sha, archive_sha, archive_root


def _assert_receipt_stage(repo: Path, repair_archive_sha: str, receipt_head: str) -> None:
    parents = git(repo, "rev-list", "--parents", "-n", "1", receipt_head).split()
    if parents != [receipt_head, repair_archive_sha]:
        fail("legacy receipt head parent is not exact repair archive")
    changed = git(
        repo,
        "diff-tree",
        "--no-commit-id",
        "--name-status",
        "-r",
        receipt_head,
    )
    if changed != f"A\t{LEGACY_RECEIPT_RELATIVE}\n":
        fail("legacy receipt head must add only the fixed legacy receipt")


def augment_payload(
    payload: dict[str, Any], recorded: dict[str, str]
) -> dict[str, Any]:
    expected = {
        "change_name": EXACT_CHANGE,
        "candidate_sha": EXACT_CANDIDATE,
        "archive_sha": EXACT_ARCHIVE,
        "receipt_head_verification": "PASS_DERIVED",
        "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
    }
    if any(payload.get(name) != value for name, value in expected.items()):
        fail("derived payload boundary mismatch")
    if recorded != {"state": "IMPLEMENTING", "candidate_sha": "none"}:
        fail("recorded legacy checkpoint boundary mismatch")
    result = dict(payload)
    result.update(
        {
            "compatibility_rule": COMPATIBILITY_RULE_ID,
            "recorded_checkpoint_state": recorded["state"],
            "recorded_checkpoint_candidate_sha": recorded["candidate_sha"],
        }
    )
    return result


def verify_legacy_change(
    repo: Path, *, module_cache_download: Path | None = None
) -> dict[str, Any]:
    repo = repo.resolve()
    assert_exact_control_blobs(repo)
    v1 = _load_v1(repo)
    original_enumerate = v1._enumerate_receipts
    original_checkpoint = v1._checkpoint_fields
    recorded_holder: dict[str, str] = {}

    def checkpoint_adapter(data: bytes) -> dict[str, str]:
        view, recorded = compatibility_checkpoint_fields(data, original_checkpoint)
        if recorded is not None:
            if recorded_holder:
                fail("legacy checkpoint compatibility invoked more than once")
            recorded_holder.update(recorded)
        return view

    def receipt_adapter(
        target_repo: Path, schema: dict[str, Any]
    ) -> list[tuple[Path, dict[str, Any]]]:
        return enumerate_with_exact_legacy(
            target_repo,
            schema,
            lambda data: v1.parse_receipt_bytes(data, schema),
            original_enumerate,
        )

    v1._checkpoint_fields = checkpoint_adapter
    v1._enumerate_receipts = receipt_adapter
    try:
        payload = v1.verify_change(
            repo,
            EXACT_CHANGE,
            module_cache_download=module_cache_download,
        )
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        fail(str(exc))
    finally:
        v1._checkpoint_fields = original_checkpoint
        v1._enumerate_receipts = original_enumerate
        sys.modules.pop(v1.__name__, None)
    if not recorded_holder:
        fail("exact legacy checkpoint compatibility was not exercised")
    _, repair_archive_sha, _ = _repair_archive_state(repo)
    _assert_receipt_stage(repo, repair_archive_sha, payload["receipt_head"])
    return augment_payload(payload, recorded_holder)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verify the one exact pre-convention Mini Program governance receipt"
    )
    parser.add_argument("--repo", default=".")
    parser.add_argument("--change", required=True)
    parser.add_argument("--module-cache-download")
    parser.add_argument("--json", action="store_true", dest="as_json")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    if args.change != EXACT_CHANGE:
        fail("legacy verifier accepts only the fixed governance change")
    payload = verify_legacy_change(
        Path(args.repo),
        module_cache_download=(
            Path(args.module_cache_download).resolve()
            if args.module_cache_download
            else None
        ),
    )
    if args.as_json:
        print(json.dumps(payload, ensure_ascii=False, sort_keys=True, separators=(",", ":")))
    else:
        print(payload)


if __name__ == "__main__":
    try:
        main()
    except LegacyReceiptError as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
