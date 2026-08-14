#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import re
import subprocess
import sys
import types


RUNNER_RELATIVE = Path("tools/lifecycle-receipts/profile_runner.py")
SHA_RE = re.compile(r"^[0-9a-f]{40}$")


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def load_candidate_runner(repo: Path, candidate: str) -> object:
    if SHA_RE.fullmatch(candidate) is None:
        fail("candidate must be a full SHA")
    result = subprocess.run(
        [
            "git",
            "-C",
            str(repo),
            "cat-file",
            "blob",
            f"{candidate}:{RUNNER_RELATIVE.as_posix()}",
        ],
        check=False,
        capture_output=True,
        timeout=30,
    )
    if result.returncode != 0:
        fail("candidate profile runner cannot be loaded")
    module_name = "archive_gate_candidate_profile_runner"
    module = types.ModuleType(module_name)
    module.__file__ = f"{candidate}:{RUNNER_RELATIVE.as_posix()}"
    sys.modules[module_name] = module
    try:
        exec(compile(result.stdout, module.__file__, "exec"), module.__dict__)
    except (OSError, ImportError, SyntaxError) as exc:
        fail(f"candidate profile runner cannot be loaded: {exc}")
    return module


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", required=True)
    parser.add_argument("--candidate", required=True)
    parser.add_argument("--archive", required=True)
    args = parser.parse_args()

    repo = Path(args.repo).resolve()
    if not repo.is_dir() or repo.is_symlink():
        fail("repository is missing or unsafe")
    try:
        runner = load_candidate_runner(repo, args.candidate)
        runner.validate_archive_commit(
            repo,
            args.candidate,
            args.archive,
            require_head=True,
        )
    except runner.ProfileError as exc:
        fail(str(exc))
    print("archive-gate=PASS")


if __name__ == "__main__":
    main()
