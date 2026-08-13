#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import subprocess
import sys


TARGET = "7a5e8bb261b994d68ce9af5eada347df6700c490"
CHANGE = Path("openspec/changes/enable-run-loop-self-evolution")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def run(argv: list[str], cwd: Path, expected: str) -> None:
    result = subprocess.run(
        argv, cwd=cwd, check=False, capture_output=True, text=True, timeout=90
    )
    require(result.returncode == 0, (result.stderr or result.stdout).strip())
    require(result.stdout.strip() == expected, f"unexpected controlled output: {result.stdout!r}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--temp-root", required=True)
    parser.add_argument("--git", required=True)
    args = parser.parse_args()
    repo = Path(args.repo).resolve()
    temp_root = Path(args.temp_root).resolve()
    require(repo.is_dir() and not repo.is_symlink(), "target repository is unsafe")
    require(temp_root.is_dir() and not temp_root.is_symlink(), "profile temp root is unsafe")
    head = subprocess.run(
        [args.git, "-C", str(repo), "rev-parse", "HEAD"],
        check=False,
        capture_output=True,
        text=True,
        timeout=10,
    )
    require(head.returncode == 0 and head.stdout.strip() == TARGET, "wrong self-evolution target")
    checks = repo / CHANGE / "checks"
    run(
        [sys.executable, str(checks / "verify_contract.py"), "--repo", str(repo)],
        repo,
        "contract_check=PASS",
    )
    run(
        [sys.executable, str(checks / "run_checker_regressions.py")],
        repo,
        "checker_regressions=PASS: positive=7 negative=7",
    )
    print("self-evolution-v1=MECHANICAL_PASS")


if __name__ == "__main__":
    main()
