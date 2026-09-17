#!/usr/bin/env python3
"""Fail-closed Mini Program TDD and user-regression gate validation."""

import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence, Set, Tuple


MARKER_PATH = "tools/miniprogram_gate.py"
MANIFEST_NAME = "miniprogram-gates.json"
MINIPROGRAM_PREFIX = "apps/wechat-miniprogram/"
FULL_SHA_RE = re.compile(r"^[0-9a-f]{40}$")
TASK_RE = re.compile(r"^\s*- \[([ xX])\]\s+([0-9]+(?:\.[0-9]+)*)\s+(.+?)\s*$")
UI_RANK = {"UI0": 0, "UI1": 1, "UI2": 2, "UI3": 3}
NATIVE_PATTERNS = {
    "wx.login": re.compile(r"\bwx\.login\b"),
    "getPhoneNumber": re.compile(r"getPhoneNumber"),
    "wx.requestPayment": re.compile(r"\bwx\.requestPayment\b"),
    "wx.scanCode": re.compile(r"\bwx\.scanCode\b"),
    "wx.requestSubscribeMessage": re.compile(r"\bwx\.requestSubscribeMessage\b"),
}
HIGH_STATES = {"INDEPENDENT_VERIFIED", "INTEGRATED", "ARCHIVED"}


class GateViolation(ValueError):
    pass


def _git(root: Path, *args: str, allow_failure: bool = False) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        ["git", *args],
        cwd=str(root),
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode != 0 and not allow_failure:
        raise GateViolation(
            f"git {' '.join(args)} failed with exit {result.returncode}: {result.stderr.strip()}"
        )
    return result


def _commit_exists(root: Path, sha: str) -> bool:
    if not FULL_SHA_RE.fullmatch(sha):
        return False
    return _git(root, "cat-file", "-e", f"{sha}^{{commit}}", allow_failure=True).returncode == 0


def _blob_exists(root: Path, sha: str, path: str) -> bool:
    return _git(root, "cat-file", "-e", f"{sha}:{path}", allow_failure=True).returncode == 0


def _show_bytes(root: Path, sha: str, path: str) -> Optional[bytes]:
    result = subprocess.run(
        ["git", "show", f"{sha}:{path}"],
        cwd=str(root),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return result.stdout if result.returncode == 0 else None


def _candidate_parent(root: Path, candidate_sha: str) -> str:
    result = _git(root, "rev-parse", f"{candidate_sha}^")
    parent = result.stdout.strip()
    if not FULL_SHA_RE.fullmatch(parent):
        raise GateViolation("candidate parent is not a full commit SHA")
    return parent


def _activation_enabled(root: Path, candidate_sha: Optional[str]) -> bool:
    ref = _candidate_parent(root, candidate_sha) if candidate_sha else _git(root, "rev-parse", "HEAD").stdout.strip()
    return _blob_exists(root, ref, MARKER_PATH)


def _candidate_diff_paths(root: Path, candidate_sha: str) -> List[str]:
    main = _git(root, "show-ref", "--verify", "--hash", "refs/heads/main", allow_failure=True)
    base = _candidate_parent(root, candidate_sha)
    if main.returncode == 0:
        merge_base = _git(root, "merge-base", candidate_sha, main.stdout.strip(), allow_failure=True)
        if merge_base.returncode == 0 and merge_base.stdout.strip() != candidate_sha:
            base = merge_base.stdout.strip()
    result = _git(root, "diff", "--name-only", f"{base}...{candidate_sha}")
    return [line for line in result.stdout.splitlines() if line]


def _candidate_diff_text(root: Path, base_sha: str, candidate_sha: str) -> str:
    return _git(
        root,
        "diff",
        "--unified=0",
        f"{base_sha}...{candidate_sha}",
        "--",
        "apps/wechat-miniprogram",
    ).stdout


def _manifest_path(change: str) -> str:
    return f"openspec/changes/{change}/{MANIFEST_NAME}"


def _tasks_path(change: str) -> str:
    return f"openspec/changes/{change}/tasks.md"


def _proposal_path(change: str) -> str:
    return f"openspec/changes/{change}/proposal.md"


def _load_json_bytes(raw: bytes, label: str) -> Dict[str, Any]:
    try:
        value = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise GateViolation(f"{label} is not valid UTF-8 JSON: {exc}")
    if not isinstance(value, dict):
        raise GateViolation(f"{label} must be a JSON object")
    return value


def _exact_keys(value: Dict[str, Any], expected: Set[str], label: str) -> None:
    actual = set(value)
    if actual != expected:
        missing = sorted(expected - actual)
        extra = sorted(actual - expected)
        raise GateViolation(f"{label} fields mismatch: missing={missing} extra={extra}")


def _nonempty_string(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value.strip():
        raise GateViolation(f"{label} must be a non-empty string")
    if "\n" in value or "\r" in value:
        raise GateViolation(f"{label} must be one line")
    return value.strip()


def _string_list(value: Any, label: str) -> List[str]:
    if not isinstance(value, list) or not value:
        raise GateViolation(f"{label} must be a non-empty list")
    result = [_nonempty_string(item, f"{label} item") for item in value]
    if len(result) != len(set(result)):
        raise GateViolation(f"{label} contains duplicate values")
    return result


def _parse_tasks(raw: bytes) -> Dict[str, bool]:
    try:
        text = raw.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise GateViolation(f"tasks.md is not UTF-8: {exc}")
    tasks: Dict[str, bool] = {}
    for line in text.splitlines():
        match = TASK_RE.match(line)
        if not match:
            continue
        task_id = match.group(2)
        if task_id in tasks:
            raise GateViolation(f"tasks.md repeats task {task_id}")
        tasks[task_id] = match.group(1).lower() == "x"
    if not tasks:
        raise GateViolation("tasks.md contains no parseable checkbox tasks")
    return tasks


def _parse_manifest(raw: bytes, change: str) -> Dict[str, Any]:
    manifest = _load_json_bytes(raw, "Mini Program gate manifest")
    _exact_keys(
        manifest,
        {"schema_version", "change", "base_sha", "tdd", "user_regression"},
        "manifest",
    )
    if manifest["schema_version"] != 1:
        raise GateViolation("manifest schema_version must be 1")
    if manifest["change"] != change:
        raise GateViolation(f"manifest change must equal {change}")
    if not isinstance(manifest["base_sha"], str) or not FULL_SHA_RE.fullmatch(manifest["base_sha"]):
        raise GateViolation("manifest base_sha must be a full lowercase commit SHA")

    tdd = manifest["tdd"]
    if not isinstance(tdd, dict):
        raise GateViolation("manifest tdd must be an object")
    _exact_keys(tdd, {"red", "green", "refactor"}, "manifest tdd")
    task_ids: List[str] = []
    for phase in ("red", "green", "refactor"):
        record = tdd[phase]
        if not isinstance(record, dict):
            raise GateViolation(f"manifest tdd.{phase} must be an object")
        _exact_keys(record, {"task_id", "command"}, f"manifest tdd.{phase}")
        task_ids.append(_nonempty_string(record["task_id"], f"manifest tdd.{phase}.task_id"))
        _nonempty_string(record["command"], f"manifest tdd.{phase}.command")
    if len(task_ids) != len(set(task_ids)):
        raise GateViolation("manifest TDD phases must reference distinct task IDs")

    user = manifest["user_regression"]
    if not isinstance(user, dict):
        raise GateViolation("manifest user_regression must be an object")
    _exact_keys(
        user,
        {"minimum_ui_level", "native_capabilities", "scenarios"},
        "manifest user_regression",
    )
    minimum = user["minimum_ui_level"]
    if minimum not in UI_RANK or UI_RANK[minimum] < UI_RANK["UI2"]:
        raise GateViolation("ordinary Mini Program user regression requires at least UI2")
    native = user["native_capabilities"]
    if not isinstance(native, list) or any(not isinstance(item, str) or not item.strip() for item in native):
        raise GateViolation("native_capabilities must be a list of non-empty strings")
    if len(native) != len(set(native)):
        raise GateViolation("native_capabilities contains duplicates")
    if native and UI_RANK[minimum] < UI_RANK["UI3"]:
        raise GateViolation("native Mini Program capabilities require UI3 user regression")

    scenarios = user["scenarios"]
    if not isinstance(scenarios, list) or len(scenarios) < 2:
        raise GateViolation("user regression requires at least primary and regression scenarios")
    scenario_ids: List[str] = []
    kinds: Set[str] = set()
    for index, scenario in enumerate(scenarios):
        if not isinstance(scenario, dict):
            raise GateViolation(f"scenario {index} must be an object")
        _exact_keys(scenario, {"id", "kind", "steps", "expected"}, f"scenario {index}")
        scenario_ids.append(_nonempty_string(scenario["id"], f"scenario {index}.id"))
        kind = _nonempty_string(scenario["kind"], f"scenario {index}.kind")
        if kind not in {"primary", "regression"}:
            raise GateViolation(f"scenario {index}.kind must be primary or regression")
        kinds.add(kind)
        _string_list(scenario["steps"], f"scenario {index}.steps")
        _string_list(scenario["expected"], f"scenario {index}.expected")
    if len(scenario_ids) != len(set(scenario_ids)):
        raise GateViolation("user regression scenario IDs must be unique")
    if kinds != {"primary", "regression"}:
        raise GateViolation("user regression requires both primary and regression scenarios")
    return manifest


def _validate_candidate_binding(
    root: Path,
    change: str,
    candidate_sha: str,
    manifest: Dict[str, Any],
) -> Tuple[Dict[str, bool], Set[str]]:
    base_sha = manifest["base_sha"]
    if not _commit_exists(root, base_sha):
        raise GateViolation("manifest base_sha does not resolve to a commit")
    ancestor = _git(root, "merge-base", "--is-ancestor", base_sha, candidate_sha, allow_failure=True)
    if ancestor.returncode != 0:
        raise GateViolation("manifest base_sha is not an ancestor of candidate_sha")
    paths = _git(root, "diff", "--name-only", f"{base_sha}...{candidate_sha}").stdout.splitlines()
    if not any(path.startswith(MINIPROGRAM_PREFIX) for path in paths):
        raise GateViolation("manifest candidate diff does not touch apps/wechat-miniprogram")
    tasks_raw = _show_bytes(root, candidate_sha, _tasks_path(change))
    if tasks_raw is None:
        raise GateViolation("candidate is missing tasks.md")
    tasks = _parse_tasks(tasks_raw)
    diff_text = _candidate_diff_text(root, base_sha, candidate_sha)
    detected = {name for name, pattern in NATIVE_PATTERNS.items() if pattern.search(diff_text)}
    declared = set(manifest["user_regression"]["native_capabilities"])
    missing = sorted(detected - declared)
    if missing:
        raise GateViolation(f"manifest omits detected native capabilities: {', '.join(missing)}")
    if (detected or declared) and UI_RANK[manifest["user_regression"]["minimum_ui_level"]] < UI_RANK["UI3"]:
        raise GateViolation("native Mini Program candidate requires UI3 user regression")
    return tasks, detected


def _planned_miniprogram(change_path: Path) -> bool:
    if (change_path / MANIFEST_NAME).is_file():
        return True
    proposal = change_path / "proposal.md"
    if not proposal.is_file():
        return False
    return "apps/wechat-miniprogram" in proposal.read_text(encoding="utf-8")


def validate_change(
    root: Path,
    change: str,
    change_path: Path,
    lifecycle: str,
    candidate_sha: Optional[str],
    blocker: Optional[Dict[str, Any]],
    user_regression: Optional[Dict[str, Any]],
) -> Optional[Dict[str, Any]]:
    if not _activation_enabled(root, candidate_sha):
        return None

    manifest_raw: Optional[bytes]
    if candidate_sha:
        paths = _candidate_diff_paths(root, candidate_sha)
        manifest_raw = _show_bytes(root, candidate_sha, _manifest_path(change))
        is_miniprogram = any(path.startswith(MINIPROGRAM_PREFIX) for path in paths) or manifest_raw is not None
    else:
        is_miniprogram = _planned_miniprogram(change_path)
        manifest_file = change_path / MANIFEST_NAME
        manifest_raw = manifest_file.read_bytes() if manifest_file.is_file() else None
    if not is_miniprogram:
        if user_regression is not None:
            raise GateViolation("user regression receipt is only valid for a Mini Program change")
        return None
    if manifest_raw is None:
        raise GateViolation("Mini Program change is missing miniprogram-gates.json manifest")

    manifest = _parse_manifest(manifest_raw, change)
    tasks: Dict[str, bool]
    detected: Set[str] = set()
    if candidate_sha:
        if not _commit_exists(root, candidate_sha):
            raise GateViolation("candidate_sha is not a full existing commit")
        tasks, detected = _validate_candidate_binding(root, change, candidate_sha, manifest)
    else:
        tasks_file = change_path / "tasks.md"
        if not tasks_file.is_file():
            raise GateViolation("Mini Program change is missing tasks.md")
        tasks = _parse_tasks(tasks_file.read_bytes())

    if lifecycle in {"CANDIDATE", "INDEPENDENT_VERIFIED", "INTEGRATED", "ARCHIVED"}:
        for phase in ("red", "green", "refactor"):
            task_id = manifest["tdd"][phase]["task_id"]
            if task_id not in tasks:
                raise GateViolation(f"manifest {phase} task {task_id} does not exist")
            if not tasks[task_id]:
                raise GateViolation(f"manifest {phase} task {task_id} is not complete")

    scenario_ids = [item["id"] for item in manifest["user_regression"]["scenarios"]]
    summary = {
        "required": True,
        "minimum_ui_level": manifest["user_regression"]["minimum_ui_level"],
        "scenario_ids": scenario_ids,
        "native_capabilities": sorted(set(manifest["user_regression"]["native_capabilities"]) | detected),
    }
    if lifecycle not in HIGH_STATES and user_regression is not None:
        raise GateViolation("user regression receipt is premature before independent verification")
    if lifecycle in HIGH_STATES:
        if blocker is not None:
            kind = blocker.get("kind") if isinstance(blocker, dict) else "unknown"
            raise GateViolation(f"Mini Program promotion has unresolved blocker: {kind}")
        if candidate_sha is None:
            raise GateViolation("Mini Program promotion requires candidate_sha")
        _validate_stored_receipt(change, candidate_sha, manifest, user_regression)
        summary["receipt"] = user_regression
    return summary


def _validate_stored_receipt(
    change: str,
    candidate_sha: str,
    manifest: Dict[str, Any],
    record: Optional[Dict[str, Any]],
) -> None:
    if not isinstance(record, dict):
        raise GateViolation("Mini Program promotion requires an exact user regression receipt")
    expected = {
        "candidate_sha",
        "ui_level",
        "scenario_ids",
        "environment",
        "unverified_boundary",
        "receipt_sha256",
    }
    _exact_keys(record, expected, "stored user regression receipt")
    if record["candidate_sha"] != candidate_sha:
        raise GateViolation("stored user regression receipt belongs to a stale candidate SHA")
    minimum = manifest["user_regression"]["minimum_ui_level"]
    if record["ui_level"] not in UI_RANK or UI_RANK[record["ui_level"]] < UI_RANK[minimum]:
        raise GateViolation(f"user regression receipt must reach {minimum}")
    expected_ids = sorted(item["id"] for item in manifest["user_regression"]["scenarios"])
    if record["scenario_ids"] != expected_ids:
        raise GateViolation("stored user regression receipt does not cover the exact scenario set")
    _nonempty_string(record["environment"], "stored receipt environment")
    _nonempty_string(record["unverified_boundary"], "stored receipt unverified_boundary")
    if not isinstance(record["receipt_sha256"], str) or not re.fullmatch(r"[0-9a-f]{64}", record["receipt_sha256"]):
        raise GateViolation("stored receipt SHA256 is invalid")


def load_receipt(
    path: Path,
    change: str,
    candidate_sha: str,
    manifest: Dict[str, Any],
) -> Dict[str, Any]:
    if path.is_symlink() or not path.is_file():
        raise GateViolation("user regression receipt must be a real regular file, not a symlink")
    raw = path.read_bytes()
    if not raw or len(raw) > 65536:
        raise GateViolation("user regression receipt must be between 1 and 65536 bytes")
    receipt = _load_json_bytes(raw, "user regression receipt")
    _exact_keys(
        receipt,
        {
            "schema_version",
            "change",
            "candidate_sha",
            "ui_level",
            "environment",
            "scenarios",
            "result",
            "unverified_boundary",
        },
        "user regression receipt",
    )
    if receipt["schema_version"] != 1:
        raise GateViolation("user regression receipt schema_version must be 1")
    if receipt["change"] != change:
        raise GateViolation(f"user regression receipt change must equal {change}")
    if receipt["candidate_sha"] != candidate_sha:
        raise GateViolation("user regression receipt belongs to a stale candidate SHA")
    minimum = manifest["user_regression"]["minimum_ui_level"]
    ui_level = receipt["ui_level"]
    if ui_level not in UI_RANK or UI_RANK[ui_level] < UI_RANK[minimum]:
        raise GateViolation(f"user regression receipt must reach {minimum}")
    if receipt["result"] != "PASS":
        raise GateViolation("user regression receipt total result must be PASS")
    environment = _nonempty_string(receipt["environment"], "receipt environment")
    boundary = _nonempty_string(receipt["unverified_boundary"], "receipt unverified_boundary")
    scenarios = receipt["scenarios"]
    if not isinstance(scenarios, list):
        raise GateViolation("receipt scenarios must be a list")
    actual_ids: List[str] = []
    for index, scenario in enumerate(scenarios):
        if not isinstance(scenario, dict):
            raise GateViolation(f"receipt scenario {index} must be an object")
        _exact_keys(scenario, {"id", "result"}, f"receipt scenario {index}")
        actual_ids.append(_nonempty_string(scenario["id"], f"receipt scenario {index}.id"))
        if scenario["result"] != "PASS":
            raise GateViolation(f"receipt scenario {index} result must be PASS")
    expected_ids = sorted(item["id"] for item in manifest["user_regression"]["scenarios"])
    if sorted(actual_ids) != expected_ids or len(actual_ids) != len(set(actual_ids)):
        raise GateViolation("user regression receipt does not cover the exact scenario set")
    return {
        "candidate_sha": candidate_sha,
        "ui_level": ui_level,
        "scenario_ids": expected_ids,
        "environment": environment,
        "unverified_boundary": boundary,
        "receipt_sha256": hashlib.sha256(raw).hexdigest(),
    }


def candidate_manifest(root: Path, change: str, candidate_sha: str) -> Dict[str, Any]:
    raw = _show_bytes(root, candidate_sha, _manifest_path(change))
    if raw is None:
        raise GateViolation("Mini Program change is missing miniprogram-gates.json manifest")
    return _parse_manifest(raw, change)


def _discover_root() -> Path:
    result = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode != 0:
        raise GateViolation("current directory is not a Git worktree")
    return Path(result.stdout.strip()).resolve()


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="Validate Mini Program OpenSpec quality gates")
    subparsers = parser.add_subparsers(dest="command", required=True)
    plan = subparsers.add_parser("plan")
    plan.add_argument("--change", required=True)
    candidate = subparsers.add_parser("candidate")
    candidate.add_argument("--change", required=True)
    candidate.add_argument("--candidate-sha", required=True)
    receipt = subparsers.add_parser("receipt")
    receipt.add_argument("--change", required=True)
    receipt.add_argument("--candidate-sha", required=True)
    receipt.add_argument("--receipt", required=True)
    args = parser.parse_args(argv)
    root = _discover_root()
    change_path = root / "openspec" / "changes" / args.change
    try:
        if args.command == "plan":
            result = validate_change(root, args.change, change_path, "APPROVED", None, None, None)
        elif args.command == "candidate":
            result = validate_change(
                root,
                args.change,
                change_path,
                "CANDIDATE",
                args.candidate_sha,
                None,
                None,
            )
        else:
            manifest = candidate_manifest(root, args.change, args.candidate_sha)
            record = load_receipt(Path(args.receipt), args.change, args.candidate_sha, manifest)
            result = validate_change(
                root,
                args.change,
                change_path,
                "INDEPENDENT_VERIFIED",
                args.candidate_sha,
                None,
                record,
            )
        required = bool(result and result.get("required"))
        print(f"MINIPROGRAM_GATE=PASS required={str(required).lower()}")
        return 0
    except GateViolation as exc:
        print(f"MINIPROGRAM_GATE=FAIL reason={exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
