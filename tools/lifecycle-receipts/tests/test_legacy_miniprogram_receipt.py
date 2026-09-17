from __future__ import annotations

import importlib.util
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[3]
MODULE_PATH = ROOT / "tools/lifecycle-receipts/verify_legacy_miniprogram_receipt.py"
CHECKPOINT_PATH = (
    "openspec/changes/enforce-miniprogram-user-regression-gate/goal-checkpoint.md"
)


def load_module():
    spec = importlib.util.spec_from_file_location("legacy_miniprogram_receipt", MODULE_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError("legacy verifier module spec is unavailable")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class LegacyCheckpointCompatibilityTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_module()
        cls.checkpoint = subprocess.run(
            [
                "git",
                "-C",
                str(ROOT),
                "show",
                f"{cls.module.EXACT_CANDIDATE}:{CHECKPOINT_PATH}",
            ],
            check=True,
            capture_output=True,
        ).stdout

    def test_exact_checkpoint_returns_narrow_view_and_recorded_facts(self) -> None:
        fallback_called = False

        def fallback(_: bytes):
            nonlocal fallback_called
            fallback_called = True
            return {"state": "STANDARD"}

        view, recorded = self.module.compatibility_checkpoint_fields(
            self.checkpoint, fallback
        )
        self.assertFalse(fallback_called)
        self.assertEqual(view["state"], "CANDIDATE")
        self.assertEqual(view["candidate_sha"], self.module.EXACT_CANDIDATE)
        self.assertEqual(view["integrated_sha"], "none")
        self.assertEqual(view["archive_sha"], "none")
        self.assertEqual(recorded, {"state": "IMPLEMENTING", "candidate_sha": "none"})

    def test_any_checkpoint_byte_change_fails_closed(self) -> None:
        changed = self.checkpoint.replace(b"IMPLEMENTING", b"CANDIDATE", 1)
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "legacy checkpoint SHA256 mismatch"
        ):
            self.module.compatibility_checkpoint_fields(changed, lambda _: {})

    def test_unrelated_checkpoint_uses_standard_parser(self) -> None:
        marker = {"state": "STANDARD"}
        view, recorded = self.module.compatibility_checkpoint_fields(
            b"# unrelated\n", lambda _: marker
        )
        self.assertIs(view, marker)
        self.assertIsNone(recorded)


class LegacyReceiptInputTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_module()

    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="legacy-receipt-input-")
        self.repo = Path(self.temporary.name)
        self.archive_root = self.repo / self.module.EXACT_ARCHIVE_PATH
        self.archive_root.mkdir(parents=True)

    def tearDown(self) -> None:
        self.temporary.cleanup()

    def write_legacy(self, path: Path | None = None) -> Path:
        destination = path or self.repo / self.module.LEGACY_RECEIPT_RELATIVE
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(b"receipt")
        return destination

    def enumerate(self, *, change: str | None = None):
        change = change or self.module.EXACT_CHANGE
        return self.module.enumerate_with_exact_legacy(
            self.repo,
            {},
            lambda _: {"change_name": change},
            lambda _repo, _schema: [],
        )

    def test_only_fixed_legacy_path_is_added(self) -> None:
        path = self.write_legacy()
        self.assertEqual(
            self.enumerate(),
            [(path.resolve(), {"change_name": self.module.EXACT_CHANGE})],
        )

    def test_missing_legacy_receipt_is_rejected(self) -> None:
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "missing exact legacy lifecycle receipt"
        ):
            self.enumerate()

    def test_standard_receipt_conflict_is_rejected(self) -> None:
        self.write_legacy()
        (self.archive_root / "lifecycle-receipt.md").write_bytes(b"standard")
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "standard lifecycle receipt is forbidden"
        ):
            self.enumerate()

    def test_duplicate_legacy_filename_is_rejected(self) -> None:
        self.write_legacy()
        self.write_legacy(
            self.repo
            / "openspec/changes/archive/2026-08-21-other/legacy-lifecycle-receipt.md"
        )
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "legacy lifecycle receipt path is ambiguous"
        ):
            self.enumerate()

    def test_wrong_change_inside_legacy_receipt_is_rejected(self) -> None:
        self.write_legacy()
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "legacy receipt change mismatch"
        ):
            self.enumerate(change="other-change")

    def test_symlink_legacy_receipt_is_rejected(self) -> None:
        target = self.archive_root / "payload"
        target.write_bytes(b"receipt")
        (self.repo / self.module.LEGACY_RECEIPT_RELATIVE).symlink_to(target)
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "regular non-symlink file"
        ):
            self.enumerate()


class ProtectedControlBlobTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_module()

    def test_exact_b_protected_blobs_are_accepted(self) -> None:
        self.module.assert_exact_control_blobs(ROOT)

    def test_wrong_expected_blob_is_rejected(self) -> None:
        expected = dict(self.module.EXPECTED_CONTROL_BLOBS)
        path = next(iter(expected))
        expected[path] = "0" * 40
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "protected control blob mismatch"
        ):
            self.module.assert_exact_control_blobs(ROOT, expected=expected)


class ResultBoundaryTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_module()

    def test_payload_preserves_original_state_and_excluded_claim(self) -> None:
        payload = self.module.augment_payload(
            {
                "change_name": self.module.EXACT_CHANGE,
                "candidate_sha": self.module.EXACT_CANDIDATE,
                "archive_sha": self.module.EXACT_ARCHIVE,
                "receipt_head_verification": "PASS_DERIVED",
                "actor_independence": "NOT_PROVEN_BY_MECHANICAL_REPLAY",
            },
            {"state": "IMPLEMENTING", "candidate_sha": "none"},
        )
        self.assertEqual(payload["recorded_checkpoint_state"], "IMPLEMENTING")
        self.assertEqual(payload["recorded_checkpoint_candidate_sha"], "none")
        self.assertEqual(
            payload["compatibility_rule"], self.module.COMPATIBILITY_RULE_ID
        )

    def test_wrong_derived_payload_is_rejected(self) -> None:
        with self.assertRaisesRegex(
            self.module.LegacyReceiptError, "derived payload boundary mismatch"
        ):
            self.module.augment_payload(
                {
                    "change_name": self.module.EXACT_CHANGE,
                    "candidate_sha": self.module.EXACT_CANDIDATE,
                    "archive_sha": self.module.EXACT_ARCHIVE,
                    "receipt_head_verification": "PASS_DERIVED",
                    "actor_independence": "PROVEN",
                },
                {"state": "IMPLEMENTING", "candidate_sha": "none"},
            )


class RepairStageFixture:
    def __init__(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix="legacy-repair-stage-")
        self.repo = Path(self.temporary.name).resolve()
        self.module = load_module()
        self.run("init", "-q")
        self.run("config", "user.email", "fixture@example.invalid")
        self.run("config", "user.name", "Fixture")
        for relative in (
            "openspec/specs/miniprogram-gate-lifecycle-profile/spec.md",
            "openspec/specs/loop-engineering-control-plane/spec.md",
        ):
            self.write(relative, "base\n")
        self.commit("base")
        for relative in (
            f"{self.module.REPAIR_ACTIVE_ROOT}/.openspec.yaml",
            f"{self.module.REPAIR_ACTIVE_ROOT}/proposal.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/design.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/goal-checkpoint.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/tasks.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/specs/legacy-miniprogram-gate-receipt/spec.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/specs/miniprogram-gate-lifecycle-profile/spec.md",
            f"{self.module.REPAIR_ACTIVE_ROOT}/specs/loop-engineering-control-plane/spec.md",
            self.module.ADAPTER_RELATIVE,
            self.module.TEST_RELATIVE,
        ):
            self.write(relative, f"candidate:{relative}\n")
        self.candidate = self.commit("candidate")
        self.archive_root = (
            "openspec/changes/archive/2026-08-21-"
            + self.module.REPAIR_CHANGE
        )

    def close(self) -> None:
        self.temporary.cleanup()

    def run(self, *args: str) -> str:
        return subprocess.run(
            ["git", "-C", str(self.repo), *args],
            check=True,
            capture_output=True,
            text=True,
        ).stdout.strip()

    def write(self, relative: str, value: str) -> None:
        path = self.repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(value, encoding="utf-8")

    def commit(self, message: str) -> str:
        self.run("add", "-A")
        self.run("commit", "-q", "-m", message)
        return self.run("rev-parse", "HEAD")

    def archive(self, *, extra_path: bool = False) -> str:
        source = self.repo / self.module.REPAIR_ACTIVE_ROOT
        target = self.repo / self.archive_root
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(source, target)
        self.write(
            "openspec/specs/legacy-miniprogram-gate-receipt/spec.md", "canonical\n"
        )
        self.write(
            "openspec/specs/miniprogram-gate-lifecycle-profile/spec.md", "modified\n"
        )
        self.write(
            "openspec/specs/loop-engineering-control-plane/spec.md", "modified\n"
        )
        if extra_path:
            self.write("smuggled.txt", "no\n")
        self.archive_sha = self.commit("archive")
        return self.archive_sha

    def receipt(self, *, extra_path: bool = False) -> str:
        self.write(self.module.LEGACY_RECEIPT_RELATIVE, "receipt\n")
        if extra_path:
            self.write("extra.txt", "no\n")
        self.receipt_sha = self.commit("receipt")
        return self.receipt_sha


class RepairStageValidationTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_module()

    def test_exact_candidate_archive_receipt_sequence_is_accepted(self) -> None:
        fixture = RepairStageFixture()
        try:
            archive = fixture.archive()
            receipt = fixture.receipt()
            candidate, actual_archive, root = self.module._repair_archive_state(
                fixture.repo
            )
            self.assertEqual(candidate, fixture.candidate)
            self.assertEqual(actual_archive, archive)
            self.assertEqual(root, fixture.archive_root)
            self.module._assert_receipt_stage(fixture.repo, archive, receipt)
        finally:
            fixture.close()

    def test_archive_extra_path_is_rejected(self) -> None:
        fixture = RepairStageFixture()
        try:
            fixture.archive(extra_path=True)
            with self.assertRaisesRegex(
                self.module.LegacyReceiptError,
                "repair archive contains unexpected path",
            ):
                self.module._repair_archive_state(fixture.repo)
        finally:
            fixture.close()

    def test_receipt_extra_path_is_rejected(self) -> None:
        fixture = RepairStageFixture()
        try:
            archive = fixture.archive()
            receipt = fixture.receipt(extra_path=True)
            with self.assertRaisesRegex(
                self.module.LegacyReceiptError,
                "legacy receipt head must add only",
            ):
                self.module._assert_receipt_stage(fixture.repo, archive, receipt)
        finally:
            fixture.close()

    def test_receipt_parent_must_be_exact_archive(self) -> None:
        fixture = RepairStageFixture()
        try:
            archive = fixture.archive()
            fixture.write("intermediate.txt", "no\n")
            fixture.commit("intermediate")
            receipt = fixture.receipt()
            with self.assertRaisesRegex(
                self.module.LegacyReceiptError,
                "legacy receipt head parent is not exact repair archive",
            ):
                self.module._assert_receipt_stage(fixture.repo, archive, receipt)
        finally:
            fixture.close()

    def test_adapter_drift_after_archive_is_rejected(self) -> None:
        fixture = RepairStageFixture()
        try:
            fixture.archive()
            fixture.receipt()
            fixture.write(self.module.ADAPTER_RELATIVE, "changed\n")
            fixture.commit("drift")
            with self.assertRaisesRegex(
                self.module.LegacyReceiptError, "repair controlled source drift"
            ):
                self.module._repair_archive_state(fixture.repo)
        finally:
            fixture.close()


if __name__ == "__main__":
    unittest.main()
