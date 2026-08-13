#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import subprocess
import sys


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def run(repo: Path, argv: list[str], timeout: int = 300) -> None:
    result = subprocess.run(argv, cwd=repo, check=False, capture_output=True, text=True, timeout=timeout)
    require(result.returncode == 0, (result.stderr or result.stdout).strip())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--temp-root", required=True)
    parser.add_argument("--git", required=True)
    parser.add_argument("--module-cache-download", required=True)
    args = parser.parse_args()
    repo = Path(args.repo).resolve()
    temp_root = Path(args.temp_root).resolve()
    require(repo.is_dir() and not repo.is_symlink(), "target repository is unsafe")
    require(temp_root.is_dir() and not temp_root.is_symlink(), "profile temp root is unsafe")
    suites = [
        [sys.executable, "-m", "unittest", "discover", "-s", "tools/lifecycle-receipts/tests", "-p", "test_*.py"],
        [sys.executable, "-m", "unittest", "openspec/changes/persist-archived-lifecycle-receipt/checks/test_verify_forward.py"],
        [sys.executable, "openspec/changes/persist-archived-lifecycle-receipt/checks/verify_runner_contract.py", "--repo", str(repo)],
        [sys.executable, "tools/lifecycle-receipts/profile_runner.py", "--repo", str(repo), "--all-historical", "--module-cache-download", args.module_cache_download, "--json"],
        [sys.executable, "tools/lifecycle-receipts/verify_receipt.py", "--repo", str(repo), "--chain", "--module-cache-download", args.module_cache_download, "--json"],
    ]
    for argv in suites:
        run(repo, argv)
    print("lifecycle-receipt-control-v1=MECHANICAL_PASS")


if __name__ == "__main__":
    main()
