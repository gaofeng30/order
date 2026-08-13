from __future__ import annotations

import copy
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


CHECKS_ROOT = Path(__file__).resolve().parent
VALIDATOR = CHECKS_ROOT / "verify_forward_test.py"
FALSE_GREEN_RESULT = CHECKS_ROOT / "fixtures/valid-forward-result.json"
NONEXISTENT_SHA = "1" * 40

EXPECTED_ROUTES = {
    "DRAFT": "$order-plan-change",
    "APPROVED": "$order-implement-tdd",
    "IMPLEMENTING": "$order-implement-tdd",
    "CANDIDATE": "$order-verify-change",
    "INDEPENDENT_VERIFIED": "$order-integrate-change",
    "INTEGRATED": "$order-integrate-change",
}

RECEIPT_FACTS = {
    "enable-run-loop-self-evolution": (
        "VERIFIED",
        "7a5e8bb261b994d68ce9af5eada347df6700c490",
        "94e04bf26e37e93299c26ef2c9c8aa7552619444",
        "none",
        "none",
        "self-evolution-v1",
    ),
    "connect-miniprogram-menu-catalog": (
        "FAILED",
        "6d77bdd6319722b7c71b4726c6159955da9a84b6",
        "7d01fe22ded67aeded78cb7d03de87aa12416ada",
        "109c8e828f6f5a10adff33ccdb73d4fd784b2f3d",
        "none",
        "old-menu-artifact-fail-v1",
    ),
    "supersede-miniprogram-catalog-evidence": (
        "VERIFIED_SUPERSESSION",
        "109c8e828f6f5a10adff33ccdb73d4fd784b2f3d",
        "510a84baddfb5b61200c159fd2041b7c512a92db",
        "none",
        "6d77bdd6319722b7c71b4726c6159955da9a84b6",
        "menu-supersession-v1",
    ),
}


def git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        capture_output=True,
        text=True,
    )
    if check and result.returncode != 0:
        raise AssertionError((result.stderr or result.stdout).strip())
    return result


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="") as stream:
        stream.write(content)


def commit(repo: Path, message: str) -> str:
    git(repo, "add", "-A")
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD").stdout.strip()


class SafeForwardRepo:
    def __init__(self) -> None:
        self.root = Path(tempfile.mkdtemp(prefix="forward-validator-test-")).resolve()
        temp_root = Path(tempfile.gettempdir()).resolve()
        if self.root.parent != temp_root or not self.root.name.startswith("forward-validator-test-"):
            raise RuntimeError("unsafe temporary workspace path")
        if self.root.is_symlink() or not self.root.is_dir():
            raise RuntimeError("temporary workspace is not a real directory")

        self.repo = self.root / "repo"
        self.repo.mkdir()
        git(self.repo, "init")
        git(self.repo, "config", "user.name", "Forward Validator Test")
        git(self.repo, "config", "user.email", "forward@example.invalid")
        write_text(self.repo / "README.md", "fixture seed\n")
        self.seed = commit(self.repo, "seed")

        self.receipts = []
        for name, (
            verdict,
            candidate,
            archive,
            superseded_by,
            supersedes,
            profile_id,
        ) in RECEIPT_FACTS.items():
            failed = verdict == "FAILED"
            self.receipts.append(
                {
                    "archive_path": f"openspec/changes/archive/2026-08-13-{name}",
                    "archive_sha": archive,
                    "candidate_sha": candidate,
                    "change_name": name,
                    "expected_historical_verdict": verdict,
                    "integrated_sha": candidate,
                    "integration_validation": "NOT_RUN" if failed else "PASS",
                    "archive_validation": "NOT_RUN" if failed else "PASS",
                    "lifecycle_state": "ARCHIVED",
                    "profile_id": profile_id,
                    "profile_binding_sha256": "a" * 64,
                    "recorded_attestation": {
                        "trust": "UNTRUSTED_FOR_MECHANICAL_PASS",
                        "payload": {"claimed_verdict": "FAIL" if failed else "PASS"},
                    },
                    "mechanical_verification": (
                        "EXPECTED_MECHANICAL_FAIL" if failed else "MECHANICAL_PASS"
                    ),
                    "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
                    "product_trees": {"product": self.seed},
                    "receipt_head": self.seed,
                    "receipt_head_result_persisted": False,
                    "receipt_head_verification": "PASS_DERIVED",
                    "superseded_by": superseded_by,
                    "supersedes": supersedes,
                }
            )
        self.delivery = {
            "mechanically_reproducible": True,
            "historical_change_verdict": "FAILED",
            "historical_candidate_sha": RECEIPT_FACTS[
                "connect-miniprogram-menu-catalog"
            ][1],
            "supersession_candidate_sha": RECEIPT_FACTS[
                "supersede-miniprogram-catalog-evidence"
            ][1],
            "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
            "receipts": [self.receipts[1], self.receipts[2]],
        }
        checker_data = {
            "receipts": {item["change_name"]: item for item in self.receipts},
            "delivery": self.delivery,
        }
        write_text(
            self.repo / "tools/lifecycle-receipts/checker-results.json",
            json.dumps(checker_data, sort_keys=True),
        )
        write_text(
            self.repo / "tools/lifecycle-receipts/verify_receipt.py",
            """#!/usr/bin/env python3
import argparse
import json
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--repo", required=True)
selection = parser.add_mutually_exclusive_group(required=True)
selection.add_argument("--change")
selection.add_argument("--chain", action="store_true")
parser.add_argument("--json", action="store_true")
args = parser.parse_args()
data = json.loads(Path(__file__).with_name("checker-results.json").read_text(encoding="utf-8"))
payload = data["delivery"] if args.chain else data["receipts"][args.change]
print(json.dumps(payload, sort_keys=True, separators=(",", ":")))
""",
        )
        self.candidate = commit(self.repo, "repo-local receipt checker")
        self.result_path = self.root / "forward-result.json"
        self.write_result(self.valid_result())
        git(self.repo, "checkout", "--detach", self.candidate)

    def valid_result(self) -> dict[str, object]:
        return {
            "candidate_sha": self.candidate,
            "repository_clean_before": True,
            "repository_clean_after": True,
            "verdict": "MECHANICAL_EVIDENCE_ONLY",
            "invalid_fixture_verdict": "NO-GO",
            "receipt_head_result_persisted": False,
            "receipts": copy.deepcopy(self.receipts),
            "delivery": copy.deepcopy(self.delivery),
            "routes": EXPECTED_ROUTES,
        }

    def write_result(self, payload: dict[str, object]) -> None:
        write_text(self.result_path, json.dumps(payload, sort_keys=True))

    def run_validator(self, candidate_sha: str | None = None) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                sys.executable,
                str(VALIDATOR),
                "--repo",
                str(self.repo),
                "--candidate-sha",
                candidate_sha or self.candidate,
                str(self.result_path),
            ],
            check=False,
            capture_output=True,
            text=True,
        )

    def close(self) -> None:
        root = self.root.resolve()
        temp_root = Path(tempfile.gettempdir()).resolve()
        if (
            root.parent != temp_root
            or not root.name.startswith("forward-validator-test-")
            or root.is_symlink()
            or not root.is_dir()
            or self.repo.parent != root
        ):
            raise RuntimeError("temporary cleanup validation failed")
        list(root.iterdir())
        shutil.rmtree(root)
        if root.exists():
            raise RuntimeError("temporary cleanup failed")


class ForwardValidatorTests(unittest.TestCase):
    def make_repo(self) -> SafeForwardRepo:
        fixture = SafeForwardRepo()
        self.addCleanup(fixture.close)
        return fixture

    def test_review_false_green_fixture_is_rejected_without_a_repo(self) -> None:
        result = subprocess.run(
            [
                sys.executable,
                str(VALIDATOR),
                "--candidate-sha",
                NONEXISTENT_SHA,
                str(FALSE_GREEN_RESULT),
            ],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertNotEqual(result.returncode, 0, result.stdout)

    def test_nonexistent_candidate_is_rejected(self) -> None:
        fixture = self.make_repo()
        payload = fixture.valid_result()
        payload["candidate_sha"] = NONEXISTENT_SHA
        fixture.write_result(payload)
        self.assertNotEqual(fixture.run_validator(NONEXISTENT_SHA).returncode, 0)

    def test_existing_candidate_must_equal_head(self) -> None:
        fixture = self.make_repo()
        payload = fixture.valid_result()
        payload["candidate_sha"] = fixture.seed
        fixture.write_result(payload)
        self.assertNotEqual(fixture.run_validator(fixture.seed).returncode, 0)

    def test_attached_head_is_rejected(self) -> None:
        fixture = self.make_repo()
        git(fixture.repo, "switch", "-")
        self.assertNotEqual(fixture.run_validator().returncode, 0)

    def test_dirty_repository_is_rejected(self) -> None:
        fixture = self.make_repo()
        write_text(fixture.repo / "dirty.txt", "untracked\n")
        self.assertNotEqual(fixture.run_validator().returncode, 0)

    def test_self_reported_fake_receipt_head_is_rejected(self) -> None:
        fixture = self.make_repo()
        payload = fixture.valid_result()
        receipts = payload["receipts"]
        assert isinstance(receipts, list)
        receipts[0]["receipt_head"] = "9" * 40
        fixture.write_result(payload)
        self.assertNotEqual(fixture.run_validator().returncode, 0)

    def test_writer_supplied_repo_local_checker_cannot_self_attest_pass(self) -> None:
        fixture = self.make_repo()
        result = fixture.run_validator()
        self.assertNotEqual(result.returncode, 0, result.stdout)


if __name__ == "__main__":
    unittest.main()
