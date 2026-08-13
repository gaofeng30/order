from __future__ import annotations

import hashlib
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = ROOT / "verify_receipt.py"
SCHEMA_PATH = ROOT / "receipt-schema-v1.json"
spec = importlib.util.spec_from_file_location("lifecycle_receipt_git_checker", MODULE_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load receipt checker")
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)
SCHEMA = checker.load_schema(SCHEMA_PATH)


def git(repo: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        ["git", "-C", str(repo), *args], check=False, capture_output=True, text=True
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


def canonical(value: object) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def render(values: dict[str, str]) -> str:
    lines = ["# Archived lifecycle receipt", "", "## Receipt fields"]
    lines.extend(f"{field['name']}: {values[field['name']]}" for field in SCHEMA["fields"])
    lines.extend(["", "## Retrospective", "All immutable classifications were reviewed.", ""])
    return "\n".join(lines)


class SafeTempRepo:
    def __init__(self) -> None:
        self.root = Path(tempfile.mkdtemp(prefix="lifecycle-receipt-test-")).resolve()
        temp_root = Path(tempfile.gettempdir()).resolve()
        if self.root.parent != temp_root or not self.root.name.startswith("lifecycle-receipt-test-"):
            raise RuntimeError("unsafe temporary repository path")
        if self.root.is_symlink() or not self.root.is_dir():
            raise RuntimeError("temporary repository is not a real directory")
        git(self.root, "init")
        git(self.root, "config", "user.name", "Receipt Test")
        git(self.root, "config", "user.email", "receipt@example.invalid")

    def close(self) -> None:
        root = self.root.resolve()
        temp_root = Path(tempfile.gettempdir()).resolve()
        if (
            root.parent != temp_root
            or not root.name.startswith("lifecycle-receipt-test-")
            or root.is_symlink()
            or not root.is_dir()
        ):
            raise RuntimeError("temporary cleanup validation failed")
        list(root.iterdir())
        shutil.rmtree(root)
        if root.exists():
            raise RuntimeError("temporary cleanup failed")


class ReceiptRepo:
    def __init__(
        self,
        *,
        verdict: str = "VERIFIED",
        wrong_archive_parent: bool = False,
        forbidden_candidate_path: bool = False,
        archive_extra_path: bool = False,
        checkpoint_state: str = "CANDIDATE",
        receipt_overrides: dict[str, str] | None = None,
        add_receipt: bool = True,
    ) -> None:
        self.temp = SafeTempRepo()
        self.repo = self.temp.root
        self.change = "demo-change"
        self.active = Path("openspec/changes") / self.change
        self.archive = Path("openspec/changes/archive/2026-08-13-demo-change")
        self.receipt = self.archive / "lifecycle-receipt.md"
        self.checkpoint = """# Goal checkpoint

## Module base

| field | value |
| --- | --- |
| state | `{checkpoint_state}` |
| candidate_sha | `DERIVE_FROM_GIT_EXTERNAL_HANDOFF` |
| integrated_sha | `none` |
| archive_sha | `none` |
""".format(checkpoint_state=checkpoint_state)
        self.tasks = """## 1. Writer

- [x] 1.1 Complete writer work

## 2. Verification

- [ ] 2.1 Verify exact candidate

## 3. Receipt

- [ ] 3.1 Validate receipt-head as a derived Gate
"""
        runner = b"runner-v1\n"
        (self.repo / ".agents/skills/order-run-loop").mkdir(parents=True)
        (self.repo / ".agents/skills/order-run-loop/SKILL.md").write_bytes(runner)
        write_text(self.repo / "product/demo.txt", "same-product-tree\n")
        write_text(self.repo / "README.md", "base\n")
        self.base = commit(self.repo, "base")
        self.runner_blob = git(
            self.repo, "rev-parse", f"{self.base}:.agents/skills/order-run-loop/SKILL.md"
        ).stdout.strip()
        self.runner_sha256 = hashlib.sha256(runner).hexdigest()
        self.product_tree = git(self.repo, "rev-parse", f"{self.base}:product").stdout.strip()

        write_text(self.repo / self.active / "goal-checkpoint.md", self.checkpoint)
        write_text(self.repo / self.active / "tasks.md", self.tasks)
        write_text(self.repo / self.active / "proposal.md", "candidate artifact\n")
        write_text(self.repo / self.active / "specs/demo/spec.md", "delta\n")
        if forbidden_candidate_path:
            write_text(self.repo / "forbidden.txt", "outside ownership\n")
        self.candidate = commit(self.repo, "candidate")

        if wrong_archive_parent:
            write_text(self.repo / "between.txt", "intervening\n")
            commit(self.repo, "intervening")
        target = self.repo / self.archive
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(self.repo / self.active), str(target))
        write_text(self.repo / "openspec/specs/demo/spec.md", "canonical\n")
        if archive_extra_path:
            write_text(self.repo / "unexpected.txt", "not archive output\n")
        self.archive_sha = commit(self.repo, "archive")

        failed = verdict == "FAILED"
        supersession = verdict == "VERIFIED_SUPERSESSION"
        self.values = {
            "receipt_schema": "archived-lifecycle-receipt/v1",
            "change_name": self.change,
            "archive_path": self.archive.as_posix(),
            "lifecycle_state": "ARCHIVED",
            "expected_historical_verdict": verdict,
            "failure_fingerprint": "artifact-consistency|semantic|demo|candidate|verifier" if failed else "none",
            "subsequent_gates": "NOT_RUN" if failed else "PASS",
            "superseded_by": "7" * 40 if failed else "none",
            "supersedes": "8" * 40 if supersession else "none",
            "repo_base_sha": self.base,
            "runner_skill_git_blob": self.runner_blob,
            "runner_skill_sha256": self.runner_sha256,
            "runner_version": "unversioned",
            "candidate_sha": self.candidate,
            "integrated_sha": self.candidate,
            "integration_validation": "NOT_RUN" if failed else "PASS",
            "archive_sha": self.archive_sha,
            "archive_validation": "NOT_RUN" if failed else "PASS",
            "owned_paths_json": canonical(["openspec/changes/demo-change/**"]),
            "profile_id": "demo-change-v1",
            "profile_binding_sha256": "9" * 64,
            "recorded_attestation_trust": "UNTRUSTED_FOR_MECHANICAL_PASS",
            "recorded_attestation_json": canonical(
                {"claimed_verdict": "PASS", "provenance": "writer:self-attestation"}
            ),
            "mechanical_verification": "REQUIRED_DERIVED",
            "product_trees_json": canonical({"product": self.product_tree}),
            "candidate_checkpoint_sha256": hashlib.sha256(self.checkpoint.encode()).hexdigest(),
            "candidate_tasks_sha256": hashlib.sha256(self.tasks.encode()).hexdigest(),
            "candidate_open_tasks_json": canonical(["2.1", "3.1"]),
            "post_candidate_tasks_json": canonical(
                {"2.1": "NOT_RUN", "3.1": "NOT_RUN"}
                if failed
                else {"2.1": "PASS", "3.1": "REQUIRED_DERIVED"}
            ),
            "receipt_head_verification": "REQUIRED_DERIVED",
            "observation_counts_json": canonical(
                {"candidate": 0, "checker": 1, "environment": 0, "external": 0}
            ),
            "draft_screen_decisions_json": canonical(["none"]),
            "retrospective_json": canonical(["classification complete"]),
            "unverified_boundaries_json": canonical(["none"]),
        }
        if receipt_overrides:
            self.values.update(receipt_overrides)
        if add_receipt:
            write_text(self.repo / self.receipt, render(self.values))
            self.receipt_head = commit(self.repo, "receipt")
        else:
            self.receipt_head = ""

    def close(self) -> None:
        self.temp.close()


class GitReceiptTests(unittest.TestCase):
    def make_repo(self, **kwargs: object) -> ReceiptRepo:
        fixture = ReceiptRepo(**kwargs)
        self.addCleanup(fixture.close)
        return fixture

    def test_all_three_forms_have_valid_git_history(self) -> None:
        for verdict in ("VERIFIED", "FAILED", "VERIFIED_SUPERSESSION"):
            with self.subTest(verdict=verdict):
                fixture = self.make_repo(verdict=verdict)
                result = checker._verify_change_structure_for_tests(
                    fixture.repo, fixture.change, SCHEMA_PATH
                )
                self.assertEqual(result["expected_historical_verdict"], verdict)
                self.assertEqual(result["receipt_head"], fixture.receipt_head)
                self.assertEqual(result["receipt_head_verification"], "PASS_DERIVED")

    def test_cli_list_and_change_fail_closed_without_controlled_binding(self) -> None:
        fixture = self.make_repo()
        for selection in (["--change", fixture.change], ["--list"]):
            result = subprocess.run(
                [sys.executable, str(MODULE_PATH), "--repo", str(fixture.repo), *selection, "--json"],
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0, result.stdout)

    def test_chain_cli_is_deterministic(self) -> None:
        self.assertIn("--chain", subprocess.run(
            [sys.executable, str(MODULE_PATH), "--help"],
            check=True,
            capture_output=True,
            text=True,
        ).stdout)

    def test_missing_duplicate_and_later_edited_receipts_fail(self) -> None:
        missing = self.make_repo(add_receipt=False)
        with self.assertRaisesRegex(checker.ReceiptError, "missing"):
            checker._verify_change_structure_for_tests(missing.repo, missing.change, SCHEMA_PATH)

        duplicate = self.make_repo()
        duplicate_path = duplicate.repo / "openspec/changes/archive/2026-08-14-other/lifecycle-receipt.md"
        write_text(duplicate_path, render(duplicate.values))
        commit(duplicate.repo, "duplicate")
        with self.assertRaisesRegex(checker.ReceiptError, "duplicate"):
            checker._verify_change_structure_for_tests(duplicate.repo, duplicate.change, SCHEMA_PATH)

        edited = self.make_repo()
        write_text(edited.repo / edited.receipt, render(edited.values) + "later\n")
        commit(edited.repo, "edit receipt")
        with self.assertRaisesRegex(checker.ReceiptError, "history|immutable"):
            checker._verify_change_structure_for_tests(edited.repo, edited.change, SCHEMA_PATH)

    def test_wrong_parent_archive_extra_and_forbidden_candidate_path_fail(self) -> None:
        fixtures = [
            (self.make_repo(wrong_archive_parent=True), "parent"),
            (self.make_repo(archive_extra_path=True), "archive diff"),
            (self.make_repo(forbidden_candidate_path=True), "owned"),
        ]
        for fixture, reason in fixtures:
            with self.subTest(reason=reason), self.assertRaisesRegex(checker.ReceiptError, reason):
                checker._verify_change_structure_for_tests(
                    fixture.repo, fixture.change, SCHEMA_PATH
                )

    def test_checkpoint_tasks_digest_state_and_open_task_mismatches_fail(self) -> None:
        overrides = [
            {"candidate_checkpoint_sha256": "0" * 64},
            {"candidate_tasks_sha256": "0" * 64},
            {"candidate_open_tasks_json": canonical(["2.1"])},
        ]
        for override in overrides:
            fixture = self.make_repo(receipt_overrides=override)
            with self.subTest(override=override), self.assertRaises(checker.ReceiptError):
                checker._verify_change_structure_for_tests(
                    fixture.repo, fixture.change, SCHEMA_PATH
                )

        tampered = self.make_repo()
        write_text(tampered.repo / tampered.archive / "tasks.md", tampered.tasks + "tampered\n")
        commit(tampered.repo, "tamper tasks")
        with self.assertRaisesRegex(checker.ReceiptError, "tasks|tamper"):
            checker._verify_change_structure_for_tests(
                tampered.repo, tampered.change, SCHEMA_PATH
            )

        wrong_state = self.make_repo(checkpoint_state="IMPLEMENTING")
        with self.assertRaisesRegex(checker.ReceiptError, "state"):
            checker._verify_change_structure_for_tests(
                wrong_state.repo, wrong_state.change, SCHEMA_PATH
            )

        wrong_archive_path = self.make_repo(
            receipt_overrides={
                "archive_path": "openspec/changes/archive/2026-08-14-demo-change"
            }
        )
        with self.assertRaisesRegex(checker.ReceiptError, "archive_path"):
            checker._verify_change_structure_for_tests(
                wrong_archive_path.repo,
                wrong_archive_path.change,
                SCHEMA_PATH,
            )

    def test_product_tree_mismatch_fails(self) -> None:
        fixture = self.make_repo(receipt_overrides={"product_trees_json": canonical({"product": "9" * 40})})
        with self.assertRaisesRegex(checker.ReceiptError, "product tree"):
            checker._verify_change_structure_for_tests(
                fixture.repo, fixture.change, SCHEMA_PATH
            )

    def test_dirty_worktree_fails(self) -> None:
        fixture = self.make_repo()
        write_text(fixture.repo / "dirty.txt", "uncommitted\n")
        with self.assertRaisesRegex(checker.ReceiptError, "clean worktree"):
            checker._verify_change_structure_for_tests(
                fixture.repo, fixture.change, SCHEMA_PATH
            )


class ChainTests(unittest.TestCase):
    def test_chain_requires_old_fail_reciprocal_links_passes_and_equal_trees(self) -> None:
        failed = {
            "change_name": "old-menu",
            "expected_historical_verdict": "FAILED",
            "candidate_sha": "1" * 40,
            "superseded_by": "2" * 40,
            "supersedes": "none",
            "mechanical_verification": "EXPECTED_MECHANICAL_FAIL",
            "integration_validation": "NOT_RUN",
            "archive_validation": "NOT_RUN",
            "product_trees": {"app": "3" * 40},
            "receipt_head_verification": "PASS_DERIVED",
        }
        replacement = {
            "change_name": "replacement",
            "expected_historical_verdict": "VERIFIED_SUPERSESSION",
            "candidate_sha": "2" * 40,
            "superseded_by": "none",
            "supersedes": "1" * 40,
            "mechanical_verification": "MECHANICAL_PASS",
            "integration_validation": "PASS",
            "archive_validation": "PASS",
            "product_trees": {"app": "3" * 40},
            "receipt_head_verification": "PASS_DERIVED",
        }
        result = checker.verify_delivery_chain(failed, replacement)
        self.assertTrue(result["mechanically_reproducible"])
        self.assertEqual(result["historical_change_verdict"], "FAILED")
        mutations = [
            (failed, dict(replacement, supersedes="9" * 40)),
            (failed, dict(replacement, product_trees={"app": "4" * 40})),
            (dict(failed, expected_historical_verdict="VERIFIED"), replacement),
            (failed, dict(replacement, archive_validation="NOT_RUN")),
            (failed, dict(replacement, receipt_head_verification="FAIL_DERIVED")),
        ]
        for old, new in mutations:
            with self.subTest(old=old, new=new), self.assertRaises(checker.ReceiptError):
                checker.verify_delivery_chain(old, new)


if __name__ == "__main__":
    unittest.main()
