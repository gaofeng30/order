#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path
import subprocess


TARGET = "6d77bdd6319722b7c71b4726c6159955da9a84b6"
CHANGE = Path("openspec/changes/connect-miniprogram-menu-catalog")
FINGERPRINT = "artifact-consistency|semantic|proposal-stale-draft-not-run|6d77bdd|verifier"


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


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
    require(head.returncode == 0 and head.stdout.strip() == TARGET, "wrong old-menu target")
    proposal = (repo / CHANGE / "proposal.md").read_text(encoding="utf-8")
    checkpoint = (repo / CHANGE / "goal-checkpoint.md").read_text(encoding="utf-8")
    tasks = (repo / CHANGE / "tasks.md").read_text(encoding="utf-8")
    for token in ("状态：`DRAFT`", "candidate_sha=none", "ui_level_actual=NOT_RUN"):
        require(proposal.count(token) == 1, f"old-menu proposal mismatch: {token}")
    require(
        checkpoint.count("| state | `CANDIDATE` |") == 1,
        "old-menu checkpoint no longer records CANDIDATE",
    )
    for token in ("ui_level_actual: UI1", "4/4 artifacts done"):
        require(tasks.count(token) >= 1, f"old-menu tasks mismatch: {token}")
    require("INDEPENDENT_VERIFIED" not in checkpoint, "old-menu checkpoint was falsely upgraded")
    print("old-menu-artifact-fail-v1=EXPECTED_MECHANICAL_FAIL")
    print(f"fingerprint={FINGERPRINT}")
    print("subsequent_gates=NOT_RUN")


if __name__ == "__main__":
    main()
