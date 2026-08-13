#!/usr/bin/env python3
from __future__ import annotations

import argparse
import fnmatch
import hashlib
import importlib.util
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
from typing import Any


DEFAULT_SCHEMA = Path(__file__).with_name("receipt-schema-v1.json")
RECEIPT_ROOT = PurePosixPath("openspec/changes/archive")
RECEIPT_NAME = "lifecycle-receipt.md"
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
FIELD_RE = re.compile(r"^([a-z][a-z0-9_]*): (.*)$")
FENCE_RE = re.compile(r"^ {0,3}(`{3,}|~{3,})(.*)$")
TABLE_FIELD_RE = re.compile(r"^\|\s*([a-z][a-z0-9_]*)\s*\|\s*(.*?)\s*\|$")
OPEN_TASK_RE = re.compile(r"^- \[ \] ([0-9]+(?:\.[0-9]+)?)\b(.*)$")
OBSERVATION_CLASSES = {"candidate", "environment", "checker", "external"}
TASK_RESULTS = {"PASS", "FAIL", "NOT_RUN", "REQUIRED_DERIVED"}
PROFILE_RUNNER_PATH = Path(__file__).with_name("profile_runner.py")


class ReceiptError(ValueError):
    pass


def _fail(message: str) -> None:
    raise ReceiptError(message)


def load_schema(path: Path) -> dict[str, Any]:
    try:
        schema = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        _fail(f"schema load failed: {exc}")
    if schema.get("schema") != "archived-lifecycle-receipt/v1":
        _fail("unsupported receipt schema")
    fields = schema.get("fields")
    if not isinstance(fields, list) or not fields:
        _fail("schema fields must be a non-empty array")
    names = [field.get("name") for field in fields if isinstance(field, dict)]
    if len(names) != len(fields) or len(names) != len(set(names)):
        _fail("schema fields are invalid or duplicate")
    return schema


def _record_lines(data: bytes) -> list[str]:
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        _fail(f"receipt is not UTF-8: {exc}")
    records: list[str] = []
    offset = 0
    while offset < len(text):
        newline = text.find("\n", offset)
        if newline < 0:
            record = text[offset:]
            if "\r" in record:
                _fail("bare CR record terminator is not allowed")
            records.append(record)
            offset = len(text)
            continue
        record = text[offset:newline]
        if record.endswith("\r"):
            record = record[:-1]
        elif "\r" in record:
            _fail("bare CR record terminator is not allowed")
        records.append(record)
        offset = newline + 1
    return records


def _visible_lines(lines: list[str]) -> list[tuple[int, str]]:
    visible: list[tuple[int, str]] = []
    fence_char = ""
    fence_len = 0
    for number, line in enumerate(lines, start=1):
        match = FENCE_RE.match(line)
        if fence_char:
            stripped = line.lstrip(" ")
            if stripped.startswith(fence_char * fence_len) and not stripped[fence_len:].strip():
                fence_char = ""
                fence_len = 0
            continue
        if match:
            marker = match.group(1)
            fence_char = marker[0]
            fence_len = len(marker)
            continue
        visible.append((number, line))
    if fence_char:
        _fail("unterminated fenced block")
    return visible


def _json_value(name: str, raw: str) -> Any:
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        _fail(f"{name} is not valid JSON: {exc.msg}")


def _string_array(name: str, value: Any, minimum: int = 0) -> list[str]:
    if not isinstance(value, list) or any(not isinstance(item, str) for item in value):
        _fail(f"{name} must be a JSON string array")
    if len(value) < minimum:
        _fail(f"{name} has fewer than {minimum} items")
    if len(value) != len(set(value)):
        _fail(f"{name} contains duplicate items")
    return value


def _validate_field(field: dict[str, Any], raw: str) -> Any:
    name = field["name"]
    kind = field.get("type")
    if kind in {"string", "enum", "sha", "sha256", "sha_or_none"}:
        value: Any = raw
        if kind == "enum" and value not in field.get("values", []):
            _fail(f"{name} is not an allowed value")
        if kind == "sha" and SHA_RE.fullmatch(value) is None:
            _fail(f"{name} must be a full lowercase SHA")
        if kind == "sha256" and SHA256_RE.fullmatch(value) is None:
            _fail(f"{name} must be a full lowercase SHA256")
        if kind == "sha_or_none" and value != "none" and SHA_RE.fullmatch(value) is None:
            _fail(f"{name} must be none or a full lowercase SHA")
        expected = field.get("const")
        if expected is not None and value != expected:
            _fail(f"{name} must equal {expected}")
        pattern = field.get("pattern")
        if pattern is not None and re.fullmatch(pattern, value) is None:
            _fail(f"{name} does not match its schema")
        return value
    value = _json_value(name, raw)
    if kind == "json_string_array":
        return _string_array(name, value, int(field.get("min_items", 0)))
    if kind == "json_task_results":
        if not isinstance(value, dict) or any(
            not isinstance(task_id, str) or result not in TASK_RESULTS
            for task_id, result in value.items()
        ):
            _fail(f"{name} must map task IDs to valid results")
        return value
    if kind == "json_observation_counts":
        if not isinstance(value, dict) or set(value) != OBSERVATION_CLASSES:
            _fail(f"{name} must contain exactly the four observation classes")
        if any(not isinstance(count, int) or isinstance(count, bool) or count < 0 for count in value.values()):
            _fail(f"{name} counts must be non-negative integers")
        return value
    if kind == "json_sha_map":
        if not isinstance(value, dict) or any(
            not isinstance(key, str)
            or not key
            or PurePosixPath(key).is_absolute()
            or ".." in PurePosixPath(key).parts
            or not isinstance(sha, str)
            or SHA_RE.fullmatch(sha) is None
            for key, sha in value.items()
        ):
            _fail(f"{name} must map safe relative paths to full lowercase SHAs")
        return value
    if kind == "json_object":
        if not isinstance(value, dict):
            _fail(f"{name} must be a JSON object")
        return value
    _fail(f"unsupported field type for {name}: {kind}")


def _validate_form(values: dict[str, Any]) -> None:
    verdict = values["expected_historical_verdict"]
    if verdict == "VERIFIED":
        expected = {
            "failure_fingerprint": "none",
            "subsequent_gates": "PASS",
            "superseded_by": "none",
            "supersedes": "none",
            "integration_validation": "PASS",
            "archive_validation": "PASS",
        }
    elif verdict == "FAILED":
        expected = {
            "subsequent_gates": "NOT_RUN",
            "supersedes": "none",
            "integration_validation": "NOT_RUN",
            "archive_validation": "NOT_RUN",
        }
        if values["failure_fingerprint"] == "none" or not SHA_RE.fullmatch(values["superseded_by"]):
            _fail("FAILED requires an exact failure fingerprint and superseded_by SHA")
        if any(result != "NOT_RUN" for result in values["post_candidate_tasks_json"].values()):
            _fail("FAILED post-candidate tasks must remain NOT_RUN")
    else:
        expected = {
            "failure_fingerprint": "none",
            "subsequent_gates": "PASS",
            "superseded_by": "none",
            "integration_validation": "PASS",
            "archive_validation": "PASS",
        }
        if not SHA_RE.fullmatch(values["supersedes"]):
            _fail("VERIFIED_SUPERSESSION requires an exact supersedes SHA")
    for name, wanted in expected.items():
        if values[name] != wanted:
            _fail(f"{verdict} requires {name}={wanted}")
    if values["candidate_sha"] != values["integrated_sha"]:
        _fail("integrated_sha must equal candidate_sha under pure fast-forward")
    if values["superseded_by"] == values["candidate_sha"] or values["supersedes"] == values["candidate_sha"]:
        _fail("supersession links must reference a different exact candidate")


def parse_receipt_bytes(data: bytes, schema: dict[str, Any]) -> dict[str, Any]:
    visible = _visible_lines(_record_lines(data))
    fields_heading = schema["field_section"]
    retrospective_heading = schema["retrospective_section"]
    fields_positions = [i for i, (_, line) in enumerate(visible) if line == fields_heading]
    retrospective_positions = [i for i, (_, line) in enumerate(visible) if line == retrospective_heading]
    if len(fields_positions) != 1:
        _fail(f"receipt field section count must be 1, got {len(fields_positions)}")
    if len(retrospective_positions) != 1:
        _fail(f"retrospective section count must be 1, got {len(retrospective_positions)}")
    start = fields_positions[0]
    end = retrospective_positions[0]
    if end <= start:
        _fail("retrospective section must follow receipt fields")
    if not any(line for _, line in visible[end + 1 :]):
        _fail("retrospective body is missing")
    actual: list[tuple[str, str]] = []
    for number, line in visible[start + 1 : end]:
        if not line:
            continue
        match = FIELD_RE.fullmatch(line)
        if match is None:
            _fail(f"invalid receipt field line {number}")
        actual.append((match.group(1), match.group(2)))
    expected_names = [field["name"] for field in schema["fields"]]
    actual_names = [name for name, _ in actual]
    if actual_names != expected_names:
        missing = [name for name in expected_names if name not in actual_names]
        duplicate = sorted({name for name in actual_names if actual_names.count(name) > 1})
        unknown = [name for name in actual_names if name not in expected_names]
        _fail(f"receipt fields mismatch: missing={missing} duplicate={duplicate} unknown={unknown} order={actual_names}")
    values = {
        field["name"]: _validate_field(field, raw)
        for field, (_, raw) in zip(schema["fields"], actual)
    }
    _validate_form(values)
    return values


def _git(repo: Path, *args: str, binary: bool = False) -> Any:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=not binary,
    )
    if result.returncode != 0:
        stderr = result.stderr if not binary else result.stderr.decode("utf-8", "replace")
        stdout = result.stdout if not binary else result.stdout.decode("utf-8", "replace")
        _fail(f"git {' '.join(args)} failed: {(stderr or stdout).strip()}")
    return result.stdout


def _git_success(repo: Path, *args: str) -> bool:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return result.returncode == 0


def _commit(repo: Path, sha: str, label: str) -> None:
    if SHA_RE.fullmatch(sha) is None or not _git_success(repo, "cat-file", "-e", f"{sha}^{{commit}}"):
        _fail(f"{label} does not resolve to an exact commit")


def _blob(repo: Path, sha: str, path: str) -> bytes:
    return _git(repo, "show", f"{sha}:{path}", binary=True)


def _sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def _normalize_table_value(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == "`" and value[-1] == "`":
        return value[1:-1]
    return value


def _checkpoint_fields(data: bytes) -> dict[str, str]:
    visible = _visible_lines(_record_lines(data))
    headings = {"## Module base", "## Current Runtime"}
    positions = [index for index, (_, line) in enumerate(visible) if line in headings]
    if len(positions) != 1:
        _fail(f"checkpoint runtime section count must be 1, got {len(positions)}")
    start = positions[0] + 1
    end = len(visible)
    for index in range(start, len(visible)):
        if visible[index][1].startswith("## "):
            end = index
            break
    fields: dict[str, str] = {}
    for _, line in visible[start:end]:
        match = TABLE_FIELD_RE.fullmatch(line)
        if match is None:
            continue
        name = match.group(1)
        if name in fields:
            _fail(f"duplicate checkpoint field: {name}")
        fields[name] = _normalize_table_value(match.group(2))
    for required in ("state", "candidate_sha"):
        if required not in fields:
            _fail(f"missing checkpoint field: {required}")
    sha_pair = "integrated_sha" in fields and "archive_sha" in fields
    status_pair = "integration" in fields and "archive" in fields
    if not sha_pair and not status_pair:
        _fail("checkpoint is missing integration/archive candidate-stage fields")
    return fields


def _open_tasks(data: bytes) -> tuple[list[str], dict[str, str]]:
    identifiers: list[str] = []
    descriptions: dict[str, str] = {}
    for _, line in _visible_lines(_record_lines(data)):
        match = OPEN_TASK_RE.fullmatch(line)
        if match is None:
            continue
        task_id = match.group(1)
        if task_id in descriptions:
            _fail(f"duplicate open task ID: {task_id}")
        identifiers.append(task_id)
        descriptions[task_id] = match.group(2).strip()
    return identifiers, descriptions


def _candidate_paths(repo: Path, base_sha: str, candidate_sha: str) -> list[str]:
    output = _git(repo, "diff", "--name-only", f"{base_sha}..{candidate_sha}")
    return [line for line in output.splitlines() if line]


def _check_owned(paths: list[str], patterns: list[str]) -> None:
    for pattern in patterns:
        pure = PurePosixPath(pattern)
        if not pattern or pure.is_absolute() or ".." in pure.parts or "\\" in pattern:
            _fail(f"unsafe owned path pattern: {pattern}")
    outside = [
        path for path in paths if not any(fnmatch.fnmatchcase(path, pattern) for pattern in patterns)
    ]
    if outside:
        _fail(f"candidate paths outside owned paths: {outside}")


def _archive_paths(repo: Path, sha: str, prefix: str) -> set[str]:
    output = _git(repo, "ls-tree", "-r", "--name-only", sha, "--", prefix)
    return {line for line in output.splitlines() if line}


def _check_archive_diff(
    repo: Path, candidate_sha: str, archive_sha: str, change_name: str, archive_path: str
) -> None:
    output = _git(repo, "diff-tree", "--no-commit-id", "--name-status", "-r", "-M", archive_sha)
    active_prefix = f"openspec/changes/{change_name}/"
    archive_prefix = archive_path + "/"
    moved: set[str] = set()
    canonical_changed: set[str] = set()
    active_specs_prefix = active_prefix + "specs/"
    expected_canonical = {
        "openspec/specs/" + path[len(active_specs_prefix) :]
        for path in _archive_paths(repo, candidate_sha, active_specs_prefix)
    }
    for line in output.splitlines():
        parts = line.split("\t")
        status = parts[0]
        if status.startswith("R") and len(parts) == 3:
            source, target = parts[1], parts[2]
            if not source.startswith(active_prefix) or not target.startswith(archive_prefix):
                _fail(f"archive diff contains an unexpected rename: {line}")
            if source[len(active_prefix) :] != target[len(archive_prefix) :]:
                _fail(f"archive diff changed a relative path: {line}")
            moved.add(target[len(archive_prefix) :])
            continue
        if status in {"A", "M"} and len(parts) == 2 and parts[1].startswith("openspec/specs/"):
            canonical_changed.add(parts[1])
            continue
        _fail(f"archive diff contains an unexpected path: {line}")
    if not {"goal-checkpoint.md", "tasks.md"}.issubset(moved):
        _fail("archive diff is missing checkpoint/tasks moves")
    if canonical_changed != expected_canonical:
        _fail(
            "archive diff canonical delta mismatch: "
            f"expected={sorted(expected_canonical)} actual={sorted(canonical_changed)}"
        )


def _enumerate_receipts(repo: Path, schema: dict[str, Any]) -> list[tuple[Path, dict[str, Any]]]:
    root = repo / RECEIPT_ROOT
    if not root.is_dir():
        return []
    found: list[tuple[Path, dict[str, Any]]] = []
    for path in sorted(root.glob(f"*/{RECEIPT_NAME}")):
        if not path.is_file() or path.is_symlink():
            _fail(f"receipt path is not a regular file: {path}")
        found.append((path, parse_receipt_bytes(path.read_bytes(), schema)))
    return found


def _verify_change(
    repo: Path,
    change_name: str,
    schema_path: Path = DEFAULT_SCHEMA,
    *,
    derive_mechanical: bool,
    module_cache_download: Path | None = None,
) -> dict[str, Any]:
    repo = repo.resolve()
    schema = load_schema(schema_path.resolve())
    receipts = _enumerate_receipts(repo, schema)
    matches = [(path, values) for path, values in receipts if values["change_name"] == change_name]
    if not matches:
        _fail(f"missing lifecycle receipt for change: {change_name}")
    if len(matches) != 1:
        _fail(f"duplicate lifecycle receipts for change: {change_name}")
    receipt_file, values = matches[0]
    archive_path = values["archive_path"]
    receipt_path = receipt_file.relative_to(repo).as_posix()
    if receipt_file.parent.relative_to(repo).as_posix() != archive_path:
        _fail("receipt archive_path does not match its directory")
    if not PurePosixPath(archive_path).name.endswith(f"-{change_name}"):
        _fail("receipt archive_path does not match change_name")

    base_sha = values["repo_base_sha"]
    candidate_sha = values["candidate_sha"]
    integrated_sha = values["integrated_sha"]
    archive_sha = values["archive_sha"]
    for label, sha in (
        ("repo_base_sha", base_sha),
        ("candidate_sha", candidate_sha),
        ("integrated_sha", integrated_sha),
        ("archive_sha", archive_sha),
    ):
        _commit(repo, sha, label)
    if candidate_sha != integrated_sha:
        _fail("integrated_sha must equal candidate_sha")
    if not _git_success(repo, "merge-base", "--is-ancestor", base_sha, candidate_sha):
        _fail("repo base is not an ancestor of candidate")
    parents = _git(repo, "show", "-s", "--format=%P", archive_sha).strip().split()
    if parents != [integrated_sha]:
        _fail("archive parent must be the exact integrated candidate")
    if not _git_success(repo, "merge-base", "--is-ancestor", archive_sha, "HEAD"):
        _fail("archive_sha is not present in current HEAD")

    runner_path = ".agents/skills/order-run-loop/SKILL.md"
    if _git(repo, "rev-parse", f"{base_sha}:{runner_path}").strip() != values["runner_skill_git_blob"]:
        _fail("runner Skill blob does not match base")
    if _sha256(_blob(repo, base_sha, runner_path)) != values["runner_skill_sha256"]:
        _fail("runner Skill SHA256 does not match base")

    active_path = f"openspec/changes/{change_name}"
    candidate_checkpoint = _blob(repo, candidate_sha, f"{active_path}/goal-checkpoint.md")
    candidate_tasks = _blob(repo, candidate_sha, f"{active_path}/tasks.md")
    archived_checkpoint = _blob(repo, archive_sha, f"{archive_path}/goal-checkpoint.md")
    archived_tasks = _blob(repo, archive_sha, f"{archive_path}/tasks.md")
    if candidate_checkpoint != archived_checkpoint or candidate_tasks != archived_tasks:
        _fail("archived checkpoint/tasks differ from exact candidate bytes")
    if (repo / archive_path / "goal-checkpoint.md").read_bytes() != candidate_checkpoint:
        _fail("checkpoint was tampered after archive")
    if (repo / archive_path / "tasks.md").read_bytes() != candidate_tasks:
        _fail("tasks were tampered after archive")
    if _sha256(candidate_checkpoint) != values["candidate_checkpoint_sha256"]:
        _fail("candidate checkpoint SHA256 mismatch")
    if _sha256(candidate_tasks) != values["candidate_tasks_sha256"]:
        _fail("candidate tasks SHA256 mismatch")

    checkpoint = _checkpoint_fields(candidate_checkpoint)
    if checkpoint["state"].upper() != "CANDIDATE":
        _fail("candidate checkpoint state must be CANDIDATE")
    if "integrated_sha" in checkpoint:
        if checkpoint["integrated_sha"].lower() != "none" or checkpoint["archive_sha"].lower() != "none":
            _fail("candidate checkpoint contains later lifecycle SHAs")
    elif checkpoint["integration"].upper() != "NOT_RUN" or checkpoint["archive"].upper() != "NOT_RUN":
        _fail("candidate checkpoint claims later lifecycle stages")
    open_tasks, descriptions = _open_tasks(candidate_tasks)
    if open_tasks != values["candidate_open_tasks_json"]:
        _fail("candidate open task IDs do not match receipt")
    results = values["post_candidate_tasks_json"]
    if set(results) != set(open_tasks):
        _fail("post-candidate task keys do not match candidate open tasks")
    for task_id in open_tasks:
        description = descriptions[task_id].lower()
        derived = "receipt-head" in description or "receipt_head" in description
        if values["expected_historical_verdict"] == "FAILED":
            expected = "NOT_RUN"
        else:
            expected = "REQUIRED_DERIVED" if derived else "PASS"
        if results[task_id] != expected:
            _fail(f"post-candidate task {task_id} must be {expected}")

    _check_owned(_candidate_paths(repo, base_sha, candidate_sha), values["owned_paths_json"])
    _check_archive_diff(repo, candidate_sha, archive_sha, change_name, archive_path)
    if _git_success(repo, "cat-file", "-e", f"{archive_sha}:{active_path}/goal-checkpoint.md"):
        _fail("active change path still exists at archive_sha")
    if not _git_success(repo, "cat-file", "-e", f"{archive_sha}:{archive_path}/goal-checkpoint.md"):
        _fail("dated archive path is missing at archive_sha")
    if _git_success(repo, "cat-file", "-e", f"{archive_sha}:{receipt_path}"):
        _fail("receipt existed before its post-archive add commit")

    product_trees = values["product_trees_json"]
    for product_path, expected_tree in product_trees.items():
        for sha in (candidate_sha, archive_sha):
            actual = _git(repo, "rev-parse", f"{sha}:{product_path}").strip()
            if actual != expected_tree:
                _fail(f"product tree mismatch: {sha}:{product_path}")

    history = [line for line in _git(repo, "log", "--format=%H", "--", receipt_path).splitlines() if line]
    if len(history) != 1:
        _fail(f"receipt history must contain exactly one add commit, got {len(history)}")
    receipt_head = history[0]
    if not _git_success(repo, "merge-base", "--is-ancestor", archive_sha, receipt_head):
        _fail("archive_sha is not an ancestor of receipt-head")
    add_status = _git(
        repo, "diff-tree", "--root", "--no-commit-id", "--name-status", "-r", receipt_head, "--", receipt_path
    ).strip()
    if add_status != f"A\t{receipt_path}":
        _fail("receipt history does not begin with one exact add")
    if _blob(repo, receipt_head, receipt_path) != receipt_file.read_bytes():
        _fail("working receipt bytes differ from immutable receipt-head")
    if _git(repo, "status", "--porcelain").strip():
        _fail("receipt verification requires a clean worktree and index")

    if derive_mechanical:
        runner_spec = importlib.util.spec_from_file_location(
            "archived_lifecycle_profile_runner", PROFILE_RUNNER_PATH
        )
        if runner_spec is None or runner_spec.loader is None:
            _fail("controlled profile runner cannot be loaded")
        runner = importlib.util.module_from_spec(runner_spec)
        sys.modules[runner_spec.name] = runner
        try:
            runner_spec.loader.exec_module(runner)
            mechanical = runner.run_profile(
                repo,
                values["profile_id"],
                module_cache_download=module_cache_download,
            )
        except (OSError, ValueError, subprocess.SubprocessError) as exc:
            _fail(f"mechanical profile UNVERIFIED: {exc}")
        finally:
            sys.modules.pop(runner_spec.name, None)
        if mechanical.get("profile_binding_sha256") != values["profile_binding_sha256"]:
            _fail("receipt profile binding hash mismatch")
        expected_mechanical = (
            "EXPECTED_MECHANICAL_FAIL"
            if values["expected_historical_verdict"] == "FAILED"
            else "MECHANICAL_PASS"
        )
        if mechanical.get("mechanical_verification") != expected_mechanical:
            _fail("controlled profile did not reproduce the expected historical result")
        mechanical_result = mechanical["mechanical_verification"]
    else:
        mechanical_result = "NOT_RUN_FOR_STRUCTURAL_UNIT_TEST"

    return {
        "archive_path": archive_path,
        "archive_sha": archive_sha,
        "candidate_sha": candidate_sha,
        "change_name": change_name,
        "expected_historical_verdict": values["expected_historical_verdict"],
        "integrated_sha": integrated_sha,
        "integration_validation": values["integration_validation"],
        "archive_validation": values["archive_validation"],
        "lifecycle_state": values["lifecycle_state"],
        "product_trees": product_trees,
        "profile_id": values["profile_id"],
        "profile_binding_sha256": values["profile_binding_sha256"],
        "recorded_attestation": {
            "trust": values["recorded_attestation_trust"],
            "payload": values["recorded_attestation_json"],
        },
        "mechanical_verification": mechanical_result,
        "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
        "receipt_head": receipt_head,
        "receipt_head_result_persisted": False,
        "receipt_head_verification": "PASS_DERIVED",
        "superseded_by": values["superseded_by"],
        "supersedes": values["supersedes"],
    }


def verify_change(
    repo: Path,
    change_name: str,
    schema_path: Path = DEFAULT_SCHEMA,
    *,
    module_cache_download: Path | None = None,
) -> dict[str, Any]:
    """Verify structure and always derive the controlled mechanical result."""
    return _verify_change(
        repo,
        change_name,
        schema_path,
        derive_mechanical=True,
        module_cache_download=module_cache_download,
    )


def _verify_change_structure_for_tests(
    repo: Path,
    change_name: str,
    schema_path: Path = DEFAULT_SCHEMA,
) -> dict[str, Any]:
    """Exercise only the Git/receipt layer in synthetic unit repositories."""
    return _verify_change(
        repo,
        change_name,
        schema_path,
        derive_mechanical=False,
    )


def verify_delivery_chain(failed: dict[str, Any], replacement: dict[str, Any]) -> dict[str, Any]:
    if failed.get("expected_historical_verdict") != "FAILED":
        _fail("historical receipt must preserve FAILED verdict")
    if failed.get("mechanical_verification") != "EXPECTED_MECHANICAL_FAIL":
        _fail("historical receipt must reproduce the fixed mechanical failure")
    if failed.get("integration_validation") != "NOT_RUN" or failed.get("archive_validation") != "NOT_RUN":
        _fail("historical failed receipt later Gates must remain NOT_RUN")
    if replacement.get("expected_historical_verdict") != "VERIFIED_SUPERSESSION":
        _fail("replacement receipt must be VERIFIED_SUPERSESSION")
    if replacement.get("mechanical_verification") != "MECHANICAL_PASS":
        _fail("replacement controlled profile must mechanically pass")
    for field in ("integration_validation", "archive_validation"):
        if replacement.get(field) != "PASS":
            _fail(f"replacement {field} must record PASS")
    if failed.get("superseded_by") != replacement.get("candidate_sha"):
        _fail("failed receipt superseded_by does not match replacement candidate")
    if replacement.get("supersedes") != failed.get("candidate_sha"):
        _fail("replacement supersedes does not match failed candidate")
    if failed.get("product_trees") != replacement.get("product_trees"):
        _fail("product tree identity mismatch across supersession")
    for receipt in (failed, replacement):
        if receipt.get("receipt_head_verification") != "PASS_DERIVED":
            _fail("receipt chain requires derived receipt-head PASS")
    return {
        "mechanically_reproducible": True,
        "historical_change_verdict": "FAILED",
        "historical_candidate_sha": failed["candidate_sha"],
        "supersession_candidate_sha": replacement["candidate_sha"],
        "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
    }


def verify_chain(
    repo: Path,
    schema_path: Path = DEFAULT_SCHEMA,
    *,
    module_cache_download: Path | None = None,
) -> dict[str, Any]:
    failed = verify_change(
        repo,
        "connect-miniprogram-menu-catalog",
        schema_path,
        module_cache_download=module_cache_download,
    )
    replacement = verify_change(
        repo,
        "supersede-miniprogram-catalog-evidence",
        schema_path,
        module_cache_download=module_cache_download,
    )
    result = verify_delivery_chain(failed, replacement)
    result["receipts"] = [failed, replacement]
    return result


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Verify archived lifecycle receipts")
    parser.add_argument("--repo", default=".")
    parser.add_argument("--schema", default=str(DEFAULT_SCHEMA))
    selection = parser.add_mutually_exclusive_group(required=True)
    selection.add_argument("--change")
    selection.add_argument("--list", action="store_true")
    selection.add_argument("--chain", action="store_true")
    parser.add_argument("--json", action="store_true", dest="as_json")
    parser.add_argument("--module-cache-download")
    return parser.parse_args()


def main() -> None:
    args = _parse_args()
    repo = Path(args.repo).resolve()
    schema_path = Path(args.schema).resolve()
    module_cache_download = (
        Path(args.module_cache_download).resolve() if args.module_cache_download else None
    )
    if args.change:
        payload: dict[str, Any] = verify_change(
            repo,
            args.change,
            schema_path,
            module_cache_download=module_cache_download,
        )
    elif args.chain:
        payload = verify_chain(
            repo,
            schema_path,
            module_cache_download=module_cache_download,
        )
    else:
        schema = load_schema(schema_path)
        parsed = _enumerate_receipts(repo, schema)
        names = sorted(values["change_name"] for _, values in parsed)
        if len(names) != len(set(names)):
            _fail("duplicate lifecycle receipts found during enumeration")
        payload = {
            "receipts": [
                verify_change(
                    repo,
                    name,
                    schema_path,
                    module_cache_download=module_cache_download,
                )
                for name in names
            ]
        }
    if args.as_json:
        print(json.dumps(payload, ensure_ascii=False, sort_keys=True, separators=(",", ":")))
    else:
        print(payload)


if __name__ == "__main__":
    try:
        main()
    except ReceiptError as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
