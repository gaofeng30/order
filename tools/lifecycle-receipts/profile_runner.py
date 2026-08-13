#!/usr/bin/env python3
from __future__ import annotations

import argparse
from dataclasses import dataclass
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import selectors
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Any


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
PROFILE_RESULTS = {"MECHANICAL_PASS", "EXPECTED_MECHANICAL_FAIL"}
EXPECTED_PROFILE_TARGETS = {
    "self-evolution-v1": "7a5e8bb261b994d68ce9af5eada347df6700c490",
    "old-menu-artifact-fail-v1": "6d77bdd6319722b7c71b4726c6159955da9a84b6",
    "menu-supersession-v1": "109c8e828f6f5a10adff33ccdb73d4fd784b2f3d",
}
EXPECTED_TOOL_PATHS = {
    "self-evolution-v1": "tools/lifecycle-receipts/profiles/self_evolution.py",
    "old-menu-artifact-fail-v1": "tools/lifecycle-receipts/profiles/old_menu_artifact_fail.py",
    "menu-supersession-v1": "tools/lifecycle-receipts/profiles/menu_supersession.py",
    "lifecycle-receipt-control-v1": "tools/lifecycle-receipts/profiles/lifecycle_receipt_control.py",
}
ALLOWED_ENV_KEYS = {
    "GOCACHE",
    "GOENV",
    "GOFLAGS",
    "GOMODCACHE",
    "GOPROXY",
    "GOSUMDB",
    "GOTOOLCHAIN",
    "HOME",
    "LANG",
    "LC_ALL",
    "PATH",
    "PYTHONDONTWRITEBYTECODE",
    "TMPDIR",
    "XDG_CACHE_HOME",
    "npm_config_audit",
    "npm_config_cache",
    "npm_config_fund",
    "npm_config_offline",
    "npm_config_update_notifier",
}
PROFILE_KEYS = {
    "profile_id",
    "change_name",
    "result_on_success",
    "network",
    "write_scope",
    "isolation_contract",
    "required_tools",
    "excluded_claims",
    "timeout_seconds",
    "output_limit_bytes",
    "steps",
}
STEP_KEYS = {
    "step_id",
    "argv",
    "cwd",
    "env_allowlist",
    "timeout_seconds",
    "expected_exit",
    "expected_stdout_exact",
    "output_limit_bytes",
}
BINDING_KEYS = {
    "registry_version",
    "profile_id",
    "change_name",
    "target_sha",
    "profile_definition_sha256",
    "tool_source_path",
    "tool_source_blob",
    "executor_source_path",
    "executor_source_blob",
}


class ProfileError(ValueError):
    pass


def fail(message: str) -> None:
    raise ProfileError(message)


def canonical_json(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def sha256_json(value: object) -> str:
    return hashlib.sha256(canonical_json(value).encode("utf-8")).hexdigest()


def git_blob_id(data: bytes) -> str:
    header = f"blob {len(data)}\0".encode("ascii")
    return hashlib.sha1(header + data).hexdigest()  # Git object contract requires SHA-1 here.


def load_json(path: Path, label: str) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        fail(f"{label} load failed: {exc}")
    if not isinstance(value, dict):
        fail(f"{label} must be a JSON object")
    return value


def _safe_relative(value: str, label: str) -> None:
    path = PurePosixPath(value)
    if not value or path.is_absolute() or ".." in path.parts or "\\" in value:
        fail(f"{label} must be a safe repository-relative path")


def _positive_int(value: object, label: str, maximum: int) -> int:
    if not isinstance(value, int) or isinstance(value, bool) or value <= 0 or value > maximum:
        fail(f"{label} must be an integer in 1..{maximum}")
    return value


def validate_registry_document(document: dict[str, Any]) -> dict[str, dict[str, Any]]:
    if set(document) != {"registry_version", "profiles"}:
        fail("registry top-level fields mismatch")
    if document["registry_version"] != "mechanical-profiles/v1":
        fail("unsupported mechanical profile registry")
    profiles = document["profiles"]
    if not isinstance(profiles, list) or len(profiles) != 4:
        fail("registry must contain exactly four profiles")
    result: dict[str, dict[str, Any]] = {}
    for profile in profiles:
        if not isinstance(profile, dict) or set(profile) != PROFILE_KEYS:
            fail("profile fields mismatch")
        identifier = profile["profile_id"]
        if not isinstance(identifier, str) or identifier in result:
            fail("profile_id is invalid or duplicate")
        if identifier not in EXPECTED_TOOL_PATHS:
            fail(f"unexpected profile_id: {identifier}")
        if not isinstance(profile["change_name"], str) or not profile["change_name"]:
            fail(f"{identifier} change_name is invalid")
        if profile["result_on_success"] not in PROFILE_RESULTS:
            fail(f"{identifier} result_on_success is invalid")
        if profile["network"] is not False or profile["write_scope"] != "temp-only":
            fail(f"{identifier} must declare network=false and write_scope=temp-only")
        if profile["isolation_contract"] != "trusted-wrapper-audited":
            fail(f"{identifier} requires unavailable or false isolation")
        required = profile["required_tools"]
        if not isinstance(required, dict) or not required or any(
            not isinstance(key, str) or not isinstance(value, str) or not value
            for key, value in required.items()
        ):
            fail(f"{identifier} required_tools is invalid")
        excluded = profile["excluded_claims"]
        if not isinstance(excluded, list) or not excluded or any(
            not isinstance(item, str) or not item for item in excluded
        ) or len(excluded) != len(set(excluded)):
            fail(f"{identifier} excluded_claims is invalid")
        _positive_int(profile["timeout_seconds"], f"{identifier} timeout", 1200)
        _positive_int(profile["output_limit_bytes"], f"{identifier} output limit", 1_048_576)
        steps = profile["steps"]
        if not isinstance(steps, list) or not steps:
            fail(f"{identifier} must contain steps")
        step_ids: set[str] = set()
        for step in steps:
            if not isinstance(step, dict) or set(step) != STEP_KEYS:
                fail(f"{identifier} step fields mismatch")
            step_id = step["step_id"]
            if not isinstance(step_id, str) or not step_id or step_id in step_ids:
                fail(f"{identifier} step_id is invalid or duplicate")
            step_ids.add(step_id)
            argv = step["argv"]
            if not isinstance(argv, list) or not argv or any(
                not isinstance(item, str) or not item for item in argv
            ):
                fail(f"{identifier}/{step_id} argv must be a non-empty string array")
            _safe_relative(step["cwd"], f"{identifier}/{step_id} cwd")
            env_allowlist = step["env_allowlist"]
            if not isinstance(env_allowlist, list) or any(
                key not in ALLOWED_ENV_KEYS for key in env_allowlist
            ) or len(env_allowlist) != len(set(env_allowlist)):
                fail(f"{identifier}/{step_id} env allowlist is invalid")
            _positive_int(step["timeout_seconds"], f"{identifier}/{step_id} timeout", 1200)
            if step["timeout_seconds"] > profile["timeout_seconds"]:
                fail(f"{identifier}/{step_id} timeout exceeds profile bound")
            if step["expected_exit"] != 0:
                fail(f"{identifier}/{step_id} expected exit must be 0")
            if not isinstance(step["expected_stdout_exact"], str):
                fail(f"{identifier}/{step_id} expected stdout must be a string")
            _positive_int(
                step["output_limit_bytes"], f"{identifier}/{step_id} output limit", 1_048_576
            )
            if step["output_limit_bytes"] > profile["output_limit_bytes"]:
                fail(f"{identifier}/{step_id} output limit exceeds profile bound")
        result[identifier] = profile
    if set(result) != set(EXPECTED_TOOL_PATHS):
        fail("four predeclared profile IDs are required")
    return result


def _git(repo: Path, *args: str, check: bool = True, binary: bool = False) -> Any:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=not binary,
        timeout=30,
    )
    if check and result.returncode != 0:
        stderr = result.stderr if not binary else result.stderr.decode("utf-8", "replace")
        stdout = result.stdout if not binary else result.stdout.decode("utf-8", "replace")
        fail(f"git {' '.join(args)} failed: {(stderr or stdout).strip()}")
    return result.stdout if check else result


def validate_bindings_document(
    document: dict[str, Any],
    profiles: dict[str, dict[str, Any]],
    repo: Path,
    *,
    verify_git_blobs: bool,
) -> dict[str, dict[str, Any]]:
    if set(document) != {"bindings_version", "bindings"}:
        fail("bindings top-level fields mismatch")
    if document["bindings_version"] != "mechanical-bindings/v1":
        fail("unsupported mechanical bindings registry")
    bindings = document["bindings"]
    if not isinstance(bindings, list):
        fail("bindings must be an array")
    result: dict[str, dict[str, Any]] = {}
    for binding in bindings:
        if not isinstance(binding, dict) or set(binding) != BINDING_KEYS:
            fail("binding fields mismatch")
        identifier = binding["profile_id"]
        if identifier in result or identifier not in profiles:
            fail("binding profile_id is unknown or duplicate")
        profile = profiles[identifier]
        if binding["registry_version"] != "mechanical-profiles/v1":
            fail(f"{identifier} binding registry version mismatch")
        if binding["change_name"] != profile["change_name"]:
            fail(f"{identifier} binding change mismatch")
        expected_target = EXPECTED_PROFILE_TARGETS.get(identifier)
        if expected_target is None:
            fail("bootstrap profile must remain unbound in the candidate")
        if binding["target_sha"] != expected_target:
            fail(f"{identifier} target SHA mismatch")
        if binding["profile_definition_sha256"] != sha256_json(profile):
            fail(f"{identifier} profile definition hash mismatch")
        expected_tool = EXPECTED_TOOL_PATHS[identifier]
        if binding["tool_source_path"] != expected_tool:
            fail(f"{identifier} tool source path mismatch")
        if binding["executor_source_path"] != "tools/lifecycle-receipts/profile_runner.py":
            fail(f"{identifier} executor source path mismatch")
        for field in ("tool_source_blob", "executor_source_blob"):
            if not isinstance(binding[field], str) or SHA_RE.fullmatch(binding[field]) is None:
                fail(f"{identifier} {field} must be a Git blob ID")
        for path_field, blob_field in (
            ("tool_source_path", "tool_source_blob"),
            ("executor_source_path", "executor_source_blob"),
        ):
            path = repo / binding[path_field]
            if not path.is_file() or path.is_symlink():
                fail(f"{identifier} bound source is missing or unsafe: {binding[path_field]}")
            if git_blob_id(path.read_bytes()) != binding[blob_field]:
                fail(f"{identifier} {blob_field} does not match source bytes")
            if verify_git_blobs:
                head_blob = _git(repo, "rev-parse", f"HEAD:{binding[path_field]}").strip()
                if head_blob != binding[blob_field]:
                    fail(f"{identifier} bound source is not exact at HEAD")
        result[identifier] = binding
    if set(result) != set(EXPECTED_PROFILE_TARGETS):
        fail("candidate must contain exactly the three historical bindings")
    return result


@dataclass(frozen=True)
class ControlPlane:
    profiles: dict[str, dict[str, Any]]
    bindings: dict[str, dict[str, Any]]


def load_control_plane(
    repo: Path,
    registry_path: Path,
    bindings_path: Path,
    *,
    verify_git_blobs: bool,
) -> ControlPlane:
    repo = repo.resolve()
    profiles = validate_registry_document(load_json(registry_path, "profile registry"))
    bindings = validate_bindings_document(
        load_json(bindings_path, "profile bindings"), profiles, repo, verify_git_blobs=verify_git_blobs
    )
    return ControlPlane(profiles=profiles, bindings=bindings)


@dataclass(frozen=True)
class BoundedResult:
    returncode: int
    stdout: str


def _kill_group(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


def run_bounded_argv(
    argv: list[str],
    *,
    cwd: Path,
    env: dict[str, str],
    timeout_seconds: float,
    output_limit_bytes: int,
    output_root: Path,
) -> BoundedResult:
    if not argv or any(not isinstance(item, str) or not item for item in argv):
        fail("argv must be a non-empty string array")
    cwd = cwd.resolve()
    output_root = output_root.resolve()
    if cwd != output_root and output_root not in cwd.parents:
        fail("subprocess cwd escapes validated output root")
    if any(key not in ALLOWED_ENV_KEYS for key in env):
        fail("subprocess environment contains a non-allowlisted key")
    process = subprocess.Popen(
        argv,
        cwd=cwd,
        env=env,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )
    assert process.stdout is not None
    selector = selectors.DefaultSelector()
    selector.register(process.stdout, selectors.EVENT_READ)
    output = bytearray()
    deadline = time.monotonic() + timeout_seconds
    try:
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                _kill_group(process)
                process.wait(timeout=5)
                fail("profile subprocess timeout")
            events = selector.select(min(remaining, 0.05))
            for key, _ in events:
                chunk = os.read(key.fileobj.fileno(), 65536)
                if chunk:
                    output.extend(chunk)
                    if len(output) > output_limit_bytes:
                        _kill_group(process)
                        process.wait(timeout=5)
                        fail("profile subprocess output limit exceeded")
                else:
                    selector.unregister(key.fileobj)
            if process.poll() is not None and not selector.get_map():
                break
        return BoundedResult(process.returncode, output.decode("utf-8", "replace"))
    finally:
        selector.close()
        if process.poll() is None:
            _kill_group(process)
            process.wait(timeout=5)
        process.stdout.close()


def assert_safe_tree(root: Path) -> None:
    root = root.resolve()
    if not root.is_dir() or root.is_symlink():
        fail("temporary root is not a real directory")
    for current, directories, files in os.walk(root, followlinks=False):
        current_path = Path(current)
        for name in [*directories, *files]:
            path = current_path / name
            if path.is_symlink():
                fail(f"symbolic link found in temporary tree: {path.relative_to(root)}")


def _validate_temp_root(root: Path) -> Path:
    temp_parent = Path(tempfile.gettempdir()).resolve()
    resolved = root.resolve()
    if (
        resolved.parent != temp_parent
        or not resolved.name.startswith("lifecycle-profile-")
        or not resolved.is_dir()
        or resolved.is_symlink()
    ):
        fail("unsafe mechanical-profile temporary root")
    return resolved


def _minimal_environment(temp_root: Path, binaries: dict[str, str]) -> dict[str, str]:
    home = temp_root / "home"
    tmp = temp_root / "tmp"
    cache = temp_root / "cache"
    for path in (home, tmp, cache, temp_root / "gocache", temp_root / "gomodcache", temp_root / "npm"):
        path.mkdir(parents=True, exist_ok=True)
    path_dirs = sorted({str(Path(value).parent) for value in binaries.values()})
    path_dirs.extend(["/usr/bin", "/bin"])
    return {
        "HOME": str(home),
        "TMPDIR": str(tmp),
        "XDG_CACHE_HOME": str(cache),
        "PATH": os.pathsep.join(dict.fromkeys(path_dirs)),
        "PYTHONDONTWRITEBYTECODE": "1",
        "LANG": "C",
        "LC_ALL": "C",
        "GOENV": "off",
        "GOTOOLCHAIN": "local",
        "GOPROXY": "off",
        "GOSUMDB": "off",
        "GOCACHE": str(temp_root / "gocache"),
        "GOFLAGS": "-modcacherw",
        "GOMODCACHE": str(temp_root / "gomodcache"),
        "npm_config_cache": str(temp_root / "npm"),
        "npm_config_offline": "true",
        "npm_config_audit": "false",
        "npm_config_fund": "false",
        "npm_config_update_notifier": "false",
    }


def _run_version(argv: list[str], env: dict[str, str] | None = None) -> str:
    result = subprocess.run(
        argv, check=False, capture_output=True, text=True, timeout=10, env=env
    )
    return result.stdout.strip() if result.returncode == 0 else ""


def _resolve_existing_without_links(path: Path, label: str, *, directory: bool) -> Path:
    lexical = Path(os.path.abspath(os.fspath(path.expanduser())))
    current = Path(lexical.anchor)
    try:
        for part in lexical.parts[1:]:
            current /= part
            if current.is_symlink():
                fail(f"{label} contains a symbolic-link component: {current}")
        resolved = lexical.resolve(strict=True)
    except OSError as exc:
        fail(f"{label} cannot be resolved safely: {exc}")
    expected_kind = resolved.is_dir() if directory else resolved.is_file()
    if not expected_kind:
        fail(f"{label} has the wrong filesystem type: {resolved}")
    return resolved


def _resolve_go_binary(expected: str) -> str:
    launcher = shutil.which("go")
    if not launcher:
        fail("local Go launcher is unavailable")
    local_env = dict(os.environ, GOTOOLCHAIN="local")
    if _run_version([launcher, "version"], local_env) == expected:
        return str(Path(launcher).resolve())
    match = re.fullmatch(r"go version (go[0-9.]+) (darwin)/(arm64)", expected)
    if match is None:
        fail("required Go version contract is unsupported")
    cache = subprocess.run(
        [launcher, "env", "GOMODCACHE"],
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
        env=local_env,
    )
    if cache.returncode != 0 or not cache.stdout.strip():
        fail("local Go module cache cannot be resolved")
    version, os_name, arch = match.groups()
    candidate = _resolve_existing_without_links(
        Path(cache.stdout.strip()).expanduser()
        / f"golang.org/toolchain@v0.0.1-{version}.{os_name}-{arch}"
        / "bin/go",
        "exact cached local Go toolchain",
        directory=False,
    )
    if not os.access(candidate, os.X_OK):
        fail(f"exact cached local Go toolchain is unavailable: {candidate}")
    if _run_version([str(candidate), "version"], local_env) != expected:
        fail("cached local Go toolchain version mismatch")
    return str(candidate)


def _resolve_binaries(required: dict[str, str]) -> dict[str, str]:
    names = {"python": sys.executable, "git": shutil.which("git")}
    for name in required:
        if name not in names:
            names[name] = (
                _resolve_go_binary(required[name]) if name == "go" else shutil.which(name)
            )
    if any(not value for value in names.values()):
        fail("required local tool is unavailable")
    return {key: str(Path(value).resolve()) for key, value in names.items() if value}


def _local_module_cache_download(go_binary: str) -> Path:
    result = subprocess.run(
        [go_binary, "env", "GOMODCACHE"],
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    if result.returncode != 0 or not result.stdout.strip():
        fail("local Go module cache cannot be resolved")
    return _resolve_existing_without_links(
        Path(result.stdout.strip()).expanduser() / "cache" / "download",
        "local Go module proxy cache",
        directory=True,
    )


def _check_versions(required: dict[str, str], binaries: dict[str, str], temp_root: Path) -> None:
    commands = {
        "python": [binaries["python"], "--version"],
        "git": [binaries["git"], "--version"],
        "node": [binaries.get("node", ""), "--version"],
        "npm": [binaries.get("npm", ""), "--version"],
        "go": [binaries.get("go", ""), "version"],
    }
    env = _minimal_environment(temp_root, binaries)
    for name, expected in required.items():
        result = run_bounded_argv(
            commands[name],
            cwd=temp_root,
            env={key: env[key] for key in ("HOME", "TMPDIR", "PATH", "LANG", "LC_ALL")},
            timeout_seconds=10,
            output_limit_bytes=4096,
            output_root=temp_root,
        )
        if result.returncode != 0 or result.stdout.strip() != expected:
            fail(f"{name} version mismatch: {result.stdout.strip()!r}")


def _repo_clean_detached(repo: Path, target_sha: str, label: str, git_binary: str) -> None:
    head = subprocess.run(
        [git_binary, "-C", str(repo), "rev-parse", "HEAD"], check=False, capture_output=True, text=True
    )
    if head.returncode != 0 or head.stdout.strip() != target_sha:
        fail(f"{label} HEAD is not exact target")
    symbolic = subprocess.run(
        [git_binary, "-C", str(repo), "symbolic-ref", "-q", "HEAD"],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if symbolic.returncode != 1:
        fail(f"{label} HEAD must be detached")
    status = subprocess.run(
        [git_binary, "-C", str(repo), "status", "--porcelain=v1", "--untracked-files=all"],
        check=False,
        capture_output=True,
        text=True,
    )
    if status.returncode != 0 or status.stdout:
        fail(f"{label} target worktree must be clean")


def _expand(value: str, replacements: dict[str, str]) -> str:
    for key, replacement in replacements.items():
        value = value.replace("{" + key + "}", replacement)
    if re.search(r"\{[a-z_]+\}", value):
        fail(f"unknown profile argv placeholder: {value}")
    return value


def binding_sha256(binding: dict[str, Any]) -> str:
    return sha256_json(binding)


def run_profile(
    repo: Path,
    profile_id: str,
    *,
    registry_path: Path | None = None,
    bindings_path: Path | None = None,
    writer_uncommitted: bool = False,
    module_cache_download: Path | None = None,
) -> dict[str, Any]:
    repo = repo.resolve()
    registry_path = registry_path or repo / "tools/lifecycle-receipts/mechanical-profiles-v1.json"
    bindings_path = bindings_path or repo / "tools/lifecycle-receipts/mechanical-bindings-v1.json"
    control = load_control_plane(
        repo, registry_path, bindings_path, verify_git_blobs=not writer_uncommitted
    )
    if profile_id not in control.bindings:
        fail(f"profile is not exact-target-bound: {profile_id}")
    profile = control.profiles[profile_id]
    binding = control.bindings[profile_id]
    binaries = _resolve_binaries(profile["required_tools"])
    raw_root = Path(tempfile.mkdtemp(prefix="lifecycle-profile-"))
    root = _validate_temp_root(raw_root)
    try:
        _check_versions(profile["required_tools"], binaries, root)
        env_all = _minimal_environment(root, binaries)
        worktree = root / "worktree"
        clone = run_bounded_argv(
            [binaries["git"], "clone", "--no-hardlinks", "--no-checkout", str(repo), str(worktree)],
            cwd=root,
            env={key: env_all[key] for key in ("HOME", "TMPDIR", "PATH", "LANG", "LC_ALL")},
            timeout_seconds=60,
            output_limit_bytes=65536,
            output_root=root,
        )
        if clone.returncode != 0:
            fail(f"local clone failed: {clone.stdout.strip()}")
        checkout = run_bounded_argv(
            [binaries["git"], "-C", str(worktree), "checkout", "--detach", binding["target_sha"]],
            cwd=root,
            env={key: env_all[key] for key in ("HOME", "TMPDIR", "PATH", "LANG", "LC_ALL")},
            timeout_seconds=60,
            output_limit_bytes=65536,
            output_root=root,
        )
        if checkout.returncode != 0:
            fail(f"target checkout failed: {checkout.stdout.strip()}")
        _repo_clean_detached(worktree, binding["target_sha"], "before", binaries["git"])

        profile_tool = root / "profile-tool.py"
        if writer_uncommitted:
            source = repo / binding["tool_source_path"]
            tool_bytes = source.read_bytes()
        else:
            tool_bytes = _git(repo, "cat-file", "blob", binding["tool_source_blob"], binary=True)
        if git_blob_id(tool_bytes) != binding["tool_source_blob"]:
            fail("bound profile tool blob content mismatch")
        profile_tool.write_bytes(tool_bytes)
        profile_temp = root / "profile-temp"
        profile_temp.mkdir()
        source_cache = ""
        if "go" in binaries:
            source_cache = str(
                _resolve_existing_without_links(
                    module_cache_download,
                    "explicit local Go module proxy cache",
                    directory=True,
                )
                if module_cache_download is not None
                else _local_module_cache_download(binaries["go"])
            )
        replacements = {
            "python": binaries["python"],
            "git": binaries["git"],
            "node": binaries.get("node", ""),
            "npm": binaries.get("npm", ""),
            "go": binaries.get("go", ""),
            "worktree": str(worktree),
            "temp_root": str(root),
            "profile_temp": str(profile_temp),
            "profile_tool": str(profile_tool),
            "module_cache_download": source_cache,
        }
        step_results: list[dict[str, Any]] = []
        for step in profile["steps"]:
            cwd = worktree / step["cwd"]
            if not cwd.is_dir() or cwd.is_symlink():
                fail(f"profile step cwd is missing or unsafe: {step['cwd']}")
            argv = [_expand(item, replacements) for item in step["argv"]]
            env = {key: env_all[key] for key in step["env_allowlist"]}
            result = run_bounded_argv(
                argv,
                cwd=cwd,
                env=env,
                timeout_seconds=step["timeout_seconds"],
                output_limit_bytes=step["output_limit_bytes"],
                output_root=root,
            )
            if result.returncode != step["expected_exit"]:
                tail = result.stdout[-4096:].strip()
                fail(
                    f"{profile_id}/{step['step_id']} exit mismatch: "
                    f"{result.returncode}; output={tail}"
                )
            if result.stdout != step["expected_stdout_exact"]:
                fail(f"{profile_id}/{step['step_id']} output mismatch: {result.stdout!r}")
            step_results.append({"step_id": step["step_id"], "result": "PASS"})
        _repo_clean_detached(worktree, binding["target_sha"], "after", binaries["git"])
        return {
            "profile_id": profile_id,
            "target_sha": binding["target_sha"],
            "profile_binding_sha256": binding_sha256(binding),
            "mechanical_verification": profile["result_on_success"],
            "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
            "steps": step_results,
        }
    finally:
        assert_safe_tree(root)
        shutil.rmtree(root)
        if root.exists():
            fail("mechanical-profile temporary cleanup failed")


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run controlled archived-lifecycle profiles")
    parser.add_argument("--repo", default=".")
    parser.add_argument("--profile")
    parser.add_argument("--all-historical", action="store_true")
    parser.add_argument("--writer-uncommitted", action="store_true")
    parser.add_argument("--module-cache-download")
    parser.add_argument("--json", action="store_true", dest="as_json")
    return parser.parse_args()


def main() -> None:
    args = _parse_args()
    if bool(args.profile) == bool(args.all_historical):
        fail("select exactly one profile or --all-historical")
    repo = Path(args.repo).resolve()
    identifiers = [args.profile] if args.profile else list(EXPECTED_PROFILE_TARGETS)
    results = [
        run_profile(
            repo,
            identifier,
            writer_uncommitted=args.writer_uncommitted,
            module_cache_download=(
                Path(args.module_cache_download) if args.module_cache_download else None
            ),
        )
        for identifier in identifiers
    ]
    payload: object = results[0] if args.profile else {"profiles": results}
    if args.as_json:
        print(canonical_json(payload))
    else:
        print(payload)


if __name__ == "__main__":
    try:
        main()
    except ProfileError as exc:
        print(f"UNVERIFIED: {exc}", file=sys.stderr)
        raise SystemExit(1)
