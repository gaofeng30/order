#!/usr/bin/env python3
"""Verify immutable history and the current supersession writer marker."""

from pathlib import Path
import subprocess


BASE = "7d01fe22ded67aeded78cb7d03de87aa12416ada"
HISTORICAL = "6d77bdd6319722b7c71b4726c6159955da9a84b6"
OLD_ARCHIVE = Path("openspec/changes/archive/2026-08-13-connect-miniprogram-menu-catalog")
CHANGE = Path("openspec/changes/supersede-miniprogram-catalog-evidence")
EXPECTED_TREES = {
    "apps/wechat-miniprogram": "80d16424aefa0d4b9d4e451a1ebe5e8013627a8b",
    "services/api/internal/catalog": "1867e1cb94fd38b718641d28022e1cf2e386c85b",
    "services/api/internal/httpapi": "38f9f486156547cd547d2f3840566acfbbd4c0eb",
}


def run(repo: Path, *args: str, binary: bool = False):
    result = subprocess.run(
        ["git", *args], cwd=repo, capture_output=True, text=not binary
    )
    if result.returncode != 0:
        error = result.stderr if not binary else result.stderr.decode(errors="replace")
        raise SystemExit(f"git {' '.join(args)} failed: {error.strip()}")
    return result.stdout if binary else result.stdout.rstrip("\n")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(message)


def main() -> None:
    repo = Path(__file__).resolve().parents[4]
    old_files = [
        line
        for line in run(repo, "ls-tree", "-r", "--name-only", BASE, "--", str(OLD_ARCHIVE)).splitlines()
        if line
    ]
    require(len(old_files) == 6, f"historical archive file count mismatch: {len(old_files)}")
    for relative in old_files:
        require(
            (repo / relative).read_bytes() == run(repo, "show", f"{BASE}:{relative}", binary=True),
            f"historical archive changed: {relative}",
        )

    proposal = (repo / OLD_ARCHIVE / "proposal.md").read_text()
    tasks = (repo / OLD_ARCHIVE / "tasks.md").read_text()
    checkpoint = (repo / OLD_ARCHIVE / "goal-checkpoint.md").read_text()
    for token in ("状态：`DRAFT`", "candidate_sha=none", "ui_level_actual=NOT_RUN"):
        require(proposal.count(token) == 1, f"historical proposal field mismatch: {token}")
    require(
        checkpoint.count("| state | `CANDIDATE` |") == 1,
        "historical checkpoint CANDIDATE field mismatch",
    )
    for token in ("ui_level_actual: UI1", "4/4 artifacts done"):
        require(token in tasks, f"historical tasks completion field missing: {token}")

    for tree_path, expected in EXPECTED_TREES.items():
        for sha in (HISTORICAL, BASE):
            actual = run(repo, "rev-parse", f"{sha}:{tree_path}")
            require(actual == expected, f"tree mismatch: {sha}:{tree_path}={actual}")

    protected = subprocess.run(
        [
            "git", "diff", "--quiet", BASE, "--", "apps", "services", str(OLD_ARCHIVE),
            "openspec/specs", ".agents/skills", "tools", "AGENTS.md",
        ],
        cwd=repo,
    )
    require(protected.returncode == 0, "protected path differs from repository base")

    print("historical_artifact_consistency=FAIL")
    print("historical_subsequent_gates=NOT_RUN")
    print("historical_archive_bytes=UNCHANGED")
    print("current_product_trees=EXACT")
    current = (repo / CHANGE / "goal-checkpoint.md").read_text()
    require(
        current.count("| writer_gate | `PASS` |") == 1,
        "historical FAIL preserved; supersession writer Gate missing",
    )
    print("supersession_structure=PASS")


if __name__ == "__main__":
    main()
