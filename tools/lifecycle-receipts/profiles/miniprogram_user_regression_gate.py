#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import subprocess


TARGET = "f5719e98690d0b1301ed567c8b616e074846b445"
BASE = "2299da6013c06f4ae7dcad535373c67b66c80ebb"
PROFILE_ID = "miniprogram-user-regression-gate-v1"
PROTECTED_TREES = (
    "apps",
    "services",
    "tools/lifecycle-receipts",
    ".agents/skills/order-run-loop/SKILL.md",
    ".agents/skills/order-run-loop/references/self-evolution.md",
    "docs/product/online-ordering-system-prd-0818.md",
)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def run(
    argv: list[str],
    cwd: Path,
    *,
    timeout: int = 300,
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
    require(
        result.returncode == 0,
        f"{' '.join(argv)} exit={result.returncode}: {output[-2000:]}",
    )
    require(len(output.encode("utf-8")) <= 1_048_576, "child output exceeds bound")
    return output


def git(git_binary: str, repo: Path, *args: str) -> str:
    result = subprocess.run(
        [git_binary, "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=True,
        timeout=60,
    )
    require(result.returncode == 0, f"git {' '.join(args)} failed")
    return result.stdout.strip()


def validate_real_path(path: Path, *, directory: bool) -> Path:
    lexical = Path(os.path.abspath(os.fspath(path.expanduser())))
    current = Path(lexical.anchor)
    for part in lexical.parts[1:]:
        current /= part
        require(not current.is_symlink(), f"symbolic-link component: {current}")
    try:
        resolved = lexical.resolve(strict=True)
    except OSError as exc:
        raise SystemExit(f"unsafe path {path}: {exc}") from exc
    require(resolved.is_dir() if directory else resolved.is_file(), f"wrong path type: {path}")
    return resolved


def validate_target(repo: Path, git_binary: str) -> None:
    repo = validate_real_path(repo, directory=True)
    require(git(git_binary, repo, "rev-parse", "HEAD") == TARGET, "target HEAD mismatch")
    symbolic = subprocess.run(
        [git_binary, "-C", str(repo), "symbolic-ref", "-q", "HEAD"],
        check=False,
        capture_output=True,
        timeout=30,
    )
    require(symbolic.returncode == 1, "target must be detached")
    require(
        git(git_binary, repo, "status", "--porcelain=v1", "--untracked-files=all") == "",
        "target worktree must be clean",
    )


def build_commands(*, python: str, node: str, npm: str, go: str) -> list[list[str]]:
    return [
        [
            python,
            "-m",
            "unittest",
            "-v",
            "tools.tests.test_miniprogram_gate",
            "tools.tests.test_harness",
        ],
        [npm, "test", "--prefix", "apps/wechat-miniprogram"],
        [go, "test", "./..."],
        [go, "test", "-race", "./..."],
        [go, "vet", "./..."],
        [go, "build", "./..."],
        ["bash", "services/api/scripts/smoke.sh"],
    ]


def controlled_python_environment() -> dict[str, str]:
    environment = dict(os.environ)
    environment["PYTHONDONTWRITEBYTECODE"] = "1"
    return environment


def validate_scope(repo: Path, git_binary: str) -> None:
    changed = git(git_binary, repo, "diff", "--name-only", f"{BASE}..{TARGET}").splitlines()
    require(changed, "target diff is empty")
    required = {
        "tools/miniprogram_gate.py",
        "tools/tests/test_miniprogram_gate.py",
        "tools/harness",
        "openspec/changes/enforce-miniprogram-user-regression-gate/proposal.md",
    }
    require(required.issubset(set(changed)), "target control paths are incomplete")
    require(
        not any(path.startswith(("apps/", "services/")) for path in changed),
        "target changes product code",
    )
    for path in PROTECTED_TREES:
        require(
            git(git_binary, repo, "rev-parse", f"{BASE}:{path}")
            == git(git_binary, repo, "rev-parse", f"{TARGET}:{path}"),
            f"protected object changed: {path}",
        )
    forbidden = re.compile(
        r"BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY|Authorization:\s*Bearer\s+[A-Za-z0-9]|Cookie:\s*[A-Za-z0-9]|api[_-]?key\s*[:=]\s*['\"][A-Za-z0-9]",
        re.IGNORECASE,
    )
    for path in changed:
        source = repo / path
        if source.is_file():
            try:
                text = source.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            require(forbidden.search(text) is None, f"sensitive pattern in {path}")


def validate_static(repo: Path, node: str) -> None:
    app = repo / "apps/wechat-miniprogram"
    for path in sorted(app.rglob("*.js")):
        run([node, "--check", str(path)], repo, timeout=30)
    for path in [*sorted(app.rglob("*.json")), repo / "project.config.json"]:
        json.loads(path.read_text(encoding="utf-8"))


def controlled_go_environment(temp_root: Path, cache_download: Path) -> tuple[dict[str, str], dict[str, str]]:
    temp_root = validate_real_path(temp_root, directory=True)
    cache_download = validate_real_path(cache_download, directory=True)
    for path in (
        temp_root / "home",
        temp_root / "tmp",
        temp_root / "gomodcache",
        temp_root / "gocache",
    ):
        path.mkdir(exist_ok=True)
    common = dict(os.environ)
    common.update(
        {
            "GOENV": "off",
            "GOTOOLCHAIN": "local",
            "GOSUMDB": "off",
            "GOFLAGS": "-modcacherw",
            "HOME": str(temp_root / "home"),
            "TMPDIR": str(temp_root / "tmp"),
            "GOMODCACHE": str(temp_root / "gomodcache"),
            "GOCACHE": str(temp_root / "gocache"),
        }
    )
    populate = {**common, "GOPROXY": cache_download.as_uri()}
    offline = {**common, "GOPROXY": "off"}
    return populate, offline


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
    repo = validate_real_path(Path(args.repo), directory=True)
    temp_root = validate_real_path(Path(args.temp_root), directory=True)
    cache_download = validate_real_path(Path(args.module_cache_download), directory=True)
    for binary in (args.git, args.python, args.node, args.npm, args.go):
        validate_real_path(Path(binary), directory=False)
    validate_target(repo, args.git)
    validate_scope(repo, args.git)
    validate_static(repo, args.node)
    commands = build_commands(
        python=args.python,
        node=args.node,
        npm=args.npm,
        go=args.go,
    )
    run(commands[0], repo, timeout=300, env=controlled_python_environment())
    run(commands[1], repo, timeout=300)
    module = repo / "services/api"
    populate_env, offline_env = controlled_go_environment(temp_root, cache_download)
    run([args.go, "mod", "download"], module, timeout=300, env=populate_env)
    gofmt = validate_real_path(Path(args.go).with_name("gofmt"), directory=False)
    require(
        run([str(gofmt), "-l", "services/api"], repo, timeout=120).strip() == "",
        "Go source is not formatted",
    )
    for command in commands[2:6]:
        run(command, module, timeout=900, env=offline_env)
    run(commands[6], repo, timeout=300, env=offline_env)
    validate_target(repo, args.git)
    print(f"{PROFILE_ID}=MECHANICAL_PASS")


if __name__ == "__main__":
    main()
