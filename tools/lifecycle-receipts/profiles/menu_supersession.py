#!/usr/bin/env python3
from __future__ import annotations

import argparse
from io import BytesIO
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import tarfile


TARGET = "109c8e828f6f5a10adff33ccdb73d4fd784b2f3d"
BASE = "7d01fe22ded67aeded78cb7d03de87aa12416ada"
LEGACY_BASE = "94e04bf26e37e93299c26ef2c9c8aa7552619444"
CHANGE = "openspec/changes/supersede-miniprogram-catalog-evidence"
EXPECTED_TREES = {
    "apps/wechat-miniprogram": "80d16424aefa0d4b9d4e451a1ebe5e8013627a8b",
    "services/api/internal/catalog": "1867e1cb94fd38b718641d28022e1cf2e386c85b",
    "services/api/internal/httpapi": "38f9f486156547cd547d2f3840566acfbbd4c0eb",
}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def run(
    argv: list[str],
    cwd: Path,
    *,
    expected_exit: int = 0,
    timeout: int = 180,
    env: dict[str, str] | None = None,
) -> str:
    result = subprocess.run(
        argv,
        cwd=cwd,
        env=env,
        check=False,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    output = result.stdout + result.stderr
    require(result.returncode == expected_exit, f"{' '.join(argv)} exit={result.returncode}: {output[-2000:]}")
    require(len(output.encode("utf-8")) <= 1_048_576, "child output exceeds controlled bound")
    return output


def run_stdout(
    argv: list[str], cwd: Path, *, timeout: int, env: dict[str, str]
) -> str:
    result = subprocess.run(
        argv,
        cwd=cwd,
        env=env,
        check=False,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    require(result.returncode == 0, f"{' '.join(argv)} exit={result.returncode}: {(result.stderr or result.stdout)[-2000:]}")
    require(
        len(result.stdout.encode("utf-8")) + len(result.stderr.encode("utf-8")) <= 1_048_576,
        "child output exceeds controlled bound",
    )
    return result.stdout


def git(git_binary: str, repo: Path, *args: str, binary: bool = False):
    result = subprocess.run(
        [git_binary, "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=not binary,
        timeout=60,
    )
    output = result.stdout if binary else result.stdout + result.stderr
    require(result.returncode == 0, f"git {' '.join(args)} failed")
    return result.stdout if binary else result.stdout.strip()


def validate_path_ancestry(path: Path, *, directory: bool = True) -> Path:
    lexical = Path(os.path.abspath(os.fspath(path.expanduser())))
    current = Path(lexical.anchor)
    for part in lexical.parts[1:]:
        current /= part
        require(not current.is_symlink(), f"symbolic-link component: {current}")
    try:
        resolved = lexical.resolve(strict=True)
    except OSError as exc:
        raise SystemExit(f"unsafe path {path}: {exc}") from exc
    expected_kind = resolved.is_dir() if directory else resolved.is_file()
    require(expected_kind, f"unsafe filesystem type: {path}")
    return resolved


def validate_temp_tree_no_links(path: Path) -> None:
    validate_path_ancestry(path)
    for item in path.rglob("*"):
        require(not item.is_symlink(), f"symbolic link in controlled tree: {item}")


def proxy_escape(value: str) -> str:
    return "".join(f"!{char.lower()}" if "A" <= char <= "Z" else char for char in value)


def decode_json_stream(raw: str, label: str) -> list[dict[str, object]]:
    decoder = json.JSONDecoder()
    offset = 0
    values: list[dict[str, object]] = []
    while offset < len(raw):
        while offset < len(raw) and raw[offset].isspace():
            offset += 1
        if offset == len(raw):
            break
        value, offset = decoder.raw_decode(raw, offset)
        require(isinstance(value, dict), f"{label} contains a non-object")
        values.append(value)
    require(values, f"{label} is empty")
    return values


def controlled_go_environment(
    temp_root: Path,
    go_binary: str,
    cache_download: Path,
) -> tuple[dict[str, str], dict[str, str]]:
    validate_path_ancestry(temp_root)
    cache_download = validate_path_ancestry(cache_download)
    go_path = validate_path_ancestry(Path(go_binary), directory=False)
    home = temp_root / "go-home"
    tmp = temp_root / "go-tmp"
    gomodcache = temp_root / "gomodcache"
    gocache = temp_root / "gocache"
    for path in (home, tmp, gomodcache, gocache):
        path.mkdir()
    common = {
        "HOME": str(home),
        "TMPDIR": str(tmp),
        "PATH": os.pathsep.join((str(go_path.parent), "/usr/bin", "/bin")),
        "LANG": "C",
        "LC_ALL": "C",
        "GOENV": "off",
        "GOTOOLCHAIN": "local",
        "GOSUMDB": "off",
        "GOFLAGS": "-modcacherw",
        "GOMODCACHE": str(gomodcache),
        "GOCACHE": str(gocache),
    }
    populate = {**common, "GOPROXY": cache_download.as_uri()}
    offline = {**common, "GOPROXY": "off"}
    return populate, offline


def module_identity(fact: dict[str, object], label: str) -> tuple[str, str] | None:
    require(not fact.get("Error"), f"{label} contains an Error")
    module_path = fact.get("Path")
    require(
        isinstance(module_path, str)
        and module_path
        and not any(char.isspace() or ord(char) < 32 for char in module_path),
        f"{label} module path is invalid",
    )
    if fact.get("Main") is True:
        return None
    version = fact.get("Version")
    require(
        isinstance(version, str)
        and version.startswith("v")
        and not any(char.isspace() or ord(char) < 32 for char in version),
        f"{label} module version is invalid",
    )
    require(not fact.get("Replace"), f"{label} replacement is outside the fixed profile")
    return module_path, version


def validate_source_artifacts(
    cache_download: Path,
    build_modules: dict[str, str],
    package_modules: dict[str, str],
) -> None:
    cache_download = validate_path_ancestry(cache_download)
    for module_path, version in sorted(package_modules.items()):
        require(build_modules.get(module_path) == version, "package module is outside MVS build list")
        base = cache_download.joinpath(
            *proxy_escape(module_path).split("/"), "@v", proxy_escape(version)
        )
        validate_path_ancestry(Path(f"{base}.mod"), directory=False)
        validate_path_ancestry(Path(f"{base}.zip"), directory=False)
        ziphash = validate_path_ancestry(Path(f"{base}.ziphash"), directory=False)
        require(
            re.fullmatch(r"h1:[A-Za-z0-9+/]{43}=\n?", ziphash.read_text(encoding="ascii"))
            is not None,
            f"invalid local proxy ziphash: {module_path}@{version}",
        )


def safe_extract_app(archive: bytes, destination: Path) -> None:
    with tarfile.open(fileobj=BytesIO(archive), mode="r:") as stream:
        for member in stream.getmembers():
            pure = PurePosixPath(member.name)
            require(not pure.is_absolute() and ".." not in pure.parts, "unsafe Git archive path")
            require(member.isdir() or member.isfile(), "Git archive contains a non-file entry")
            target = destination.joinpath(*pure.parts)
            require(destination == target.resolve() or destination in target.resolve().parents, "archive escape")
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            source = stream.extractfile(member)
            require(source is not None, "Git archive file cannot be read")
            target.write_bytes(source.read())


def check_legacy_red(repo: Path, temp_root: Path, git_binary: str, node: str) -> None:
    archive = git(
        git_binary,
        repo,
        "archive",
        "--format=tar",
        LEGACY_BASE,
        "apps/wechat-miniprogram",
        binary=True,
    )
    overlay = temp_root / "legacy-overlay"
    overlay.mkdir()
    safe_extract_app(archive, overlay)
    tests = overlay / "apps/wechat-miniprogram/tests"
    tests.mkdir(parents=True, exist_ok=True)
    for relative in ("tests/page-harness.js", "tests/catalog-ui1.test.js"):
        source = repo / "apps/wechat-miniprogram" / relative
        target = overlay / "apps/wechat-miniprogram" / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, target)
    output = run(
        [node, "--test", "--test-name-pattern=legacy behavior boundary", "tests/catalog-ui1.test.js"],
        overlay / "apps/wechat-miniprogram",
        expected_exit=1,
        timeout=60,
    )
    assertions = (
        "request count = 0, want 1",
        "network failure request count = 0, want 1",
        "fallback product id = p001, want null",
    )
    observed = re.findall(r"AssertionError \[ERR_ASSERTION\]: ([^\r\n]+)", output)
    require(observed == list(assertions), f"legacy Red assertion messages mismatch: {observed}")
    require("Cannot find module" not in output, "legacy Red was replaced by missing-module failure")
    validate_temp_tree_no_links(overlay)


def check_static(repo: Path, node: str) -> None:
    app = repo / "apps/wechat-miniprogram"
    js_files = sorted(app.rglob("*.js"))
    json_files = sorted(app.rglob("*.json"))
    require(len(js_files) == 51, f"JS count mismatch: {len(js_files)}")
    require(len(json_files) == 43, f"JSON count mismatch: {len(json_files)}")
    for path in js_files:
        run([node, "--check", str(path)], repo, timeout=20)
    for path in [*json_files, repo / "project.config.json"]:
        json.loads(path.read_text(encoding="utf-8"))
    package = json.loads((app / "package.json").read_text(encoding="utf-8"))
    lock = json.loads((app / "package-lock.json").read_text(encoding="utf-8"))
    for key in ("dependencies", "devDependencies", "optionalDependencies", "peerDependencies"):
        require(not package.get(key), f"package dependency field is not empty: {key}")
    require(set(lock.get("packages", {})) == {""}, "package lock contains dependency packages")
    require(not lock.get("dependencies"), "package lock dependencies are not empty")


def check_scope(repo: Path, git_binary: str) -> None:
    paths = git(git_binary, repo, "diff", "--name-only", f"{BASE}..{TARGET}").splitlines()
    require(len(paths) == 7, f"supersession changed path count mismatch: {len(paths)}")
    require(all(path.startswith(CHANGE + "/") for path in paths), "supersession escaped owned path")
    forbidden = re.compile(
        r"BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY|Authorization:\s*Bearer|Cookie:\s*[^<]|api[_-]?key\s*[:=]\s*['\"][^<]",
        re.IGNORECASE,
    )
    for path in paths:
        text = (repo / path).read_text(encoding="utf-8")
        require(forbidden.search(text) is None, f"sensitive material pattern in {path}")
    for tree_path, expected in EXPECTED_TREES.items():
        require(git(git_binary, repo, "rev-parse", f"{TARGET}:{tree_path}") == expected, f"tree mismatch: {tree_path}")


def check_go(
    repo: Path,
    temp_root: Path,
    go_binary: str,
    cache_download: Path,
) -> None:
    cache_download = validate_path_ancestry(cache_download)
    require(run([go_binary, "version"], repo).strip() == "go version go1.26.5 darwin/arm64", "Go version mismatch")
    populate_env, offline_env = controlled_go_environment(
        temp_root, go_binary, cache_download
    )
    gomodcache = Path(populate_env["GOMODCACHE"])
    gocache = Path(populate_env["GOCACHE"])
    module = repo / "services/api"
    build_json = run_stdout(
        [go_binary, "list", "-m", "-json", "all"],
        module,
        env=populate_env,
        timeout=180,
    )
    build_modules: dict[str, str] = {}
    main_count = 0
    for fact in decode_json_stream(build_json, "MVS build list"):
        identity = module_identity(fact, "MVS build list")
        if identity is None:
            main_count += 1
            continue
        module_path, version = identity
        require(module_path not in build_modules, f"duplicate MVS module: {module_path}")
        build_modules[module_path] = version
    require(main_count == 1 and build_modules, "MVS build list main/module count mismatch")
    dependency_json = run_stdout(
        [
            go_binary,
            "list",
            "-deps",
            "-test",
            "-json",
            "./internal/catalog",
            "./internal/httpapi",
        ],
        module,
        env=populate_env,
        timeout=300,
    )
    package_modules: dict[str, str] = {}
    for package in decode_json_stream(dependency_json, "target test package closure"):
        require(
            not package.get("Error") and not package.get("DepsErrors"),
            f"target test dependency graph contains an error: {package.get('ImportPath')}",
        )
        module_fact = package.get("Module") if isinstance(package, dict) else None
        if (
            isinstance(module_fact, dict)
            and not module_fact.get("Main")
            and isinstance(module_fact.get("Path"), str)
            and isinstance(module_fact.get("Version"), str)
        ):
            identity = module_identity(module_fact, "target test package closure")
            require(identity is not None, "external package module cannot be main")
            module_path, version = identity
            prior = package_modules.setdefault(module_path, version)
            require(prior == version, f"package closure module version conflict: {module_path}")
    require(package_modules, "target package dependency closure is empty")
    validate_source_artifacts(cache_download, build_modules, package_modules)
    run([go_binary, "mod", "verify"], module, env=offline_env, timeout=120)
    run(
        [go_binary, "test", "./internal/catalog", "./internal/httpapi", "-count=1"],
        module,
        env=offline_env,
        timeout=300,
    )
    validate_temp_tree_no_links(gomodcache)
    validate_temp_tree_no_links(gocache)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--temp-root", required=True)
    parser.add_argument("--git", required=True)
    parser.add_argument("--python", required=True)
    parser.add_argument("--node", required=True)
    parser.add_argument("--npm", required=True)
    parser.add_argument("--go", required=True)
    parser.add_argument("--module-cache-download", required=True)
    args = parser.parse_args()
    repo = Path(args.repo).resolve()
    temp_root = Path(args.temp_root).resolve()
    cache_download = validate_path_ancestry(Path(args.module_cache_download))
    require(repo.is_dir() and not repo.is_symlink(), "target repository is unsafe")
    require(temp_root.is_dir() and not temp_root.is_symlink(), "profile temp root is unsafe")
    require(git(args.git, repo, "rev-parse", "HEAD") == TARGET, "wrong supersession target")

    structure = run(
        [
            args.python,
            "openspec/changes/supersede-miniprogram-catalog-evidence/checks/verify_supersession.py",
        ],
        repo,
    )
    for token in (
        "historical_artifact_consistency=FAIL",
        "historical_subsequent_gates=NOT_RUN",
        "historical_archive_bytes=UNCHANGED",
        "current_product_trees=EXACT",
        "supersession_structure=PASS",
    ):
        require(token in structure, f"supersession structure output missing: {token}")
    check_legacy_red(repo, temp_root, args.git, args.node)
    ui = run([args.npm, "test", "--prefix", "apps/wechat-miniprogram"], repo, timeout=180)
    summaries: dict[str, list[str]] = {"tests": [], "pass": [], "fail": []}
    for line in ui.splitlines():
        parts = line.strip().split()
        if len(parts) >= 2 and parts[-2] in summaries and parts[-1].isdecimal():
            summaries[parts[-2]].append(parts[-1])
    for label, expected in (("tests", "13"), ("pass", "13"), ("fail", "0")):
        observed = summaries[label]
        require(observed == [expected], f"UI1 {label} result mismatch: {observed}")
    check_static(repo, args.node)
    check_scope(repo, args.git)
    check_go(repo, temp_root, args.go, cache_download)
    print("menu-supersession-v1=MECHANICAL_PASS")


if __name__ == "__main__":
    main()
