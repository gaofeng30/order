from __future__ import annotations

import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = Path(__file__).resolve().parents[3]
RUNNER_PATH = REPO_ROOT / "tools/lifecycle-receipts/profile_runner.py"
CHANGE_REL = Path("openspec/changes/allow-post-archive-bootstrap-binding")
ARCHIVE_REL = Path("openspec/changes/archive/2026-08-14-allow-post-archive-bootstrap-binding")
CANONICAL_REL = Path("openspec/specs/loop-engineering-control-plane/spec.md")
EXPECTED_REL = CHANGE_REL / "checks/expected-canonical-loop-engineering-control-plane-spec.md"
CHECKER_REL = CHANGE_REL / "checks/verify_archive.py"
BINDINGS_REL = Path("tools/lifecycle-receipts/mechanical-bindings-v1.json")
REGISTRY_REL = Path("tools/lifecycle-receipts/mechanical-profiles-v1.json")
WRAPPER_REL = Path("tools/lifecycle-receipts/profiles/lifecycle_receipt_control.py")
EXECUTOR_REL = Path("tools/lifecycle-receipts/profile_runner.py")
PROTECTED_PATHS = (
    REGISTRY_REL,
    BINDINGS_REL,
    Path("tools/lifecycle-receipts/profiles/self_evolution.py"),
    Path("tools/lifecycle-receipts/profiles/old_menu_artifact_fail.py"),
    Path("tools/lifecycle-receipts/profiles/menu_supersession.py"),
    WRAPPER_REL,
    EXECUTOR_REL,
    Path("tools/lifecycle-receipts/verify_receipt.py"),
    Path(".agents/skills/order-run-loop/SKILL.md"),
    Path(".agents/skills/order-run-loop/references/self-evolution.md"),
)
BOOTSTRAP_ID = "lifecycle-receipt-control-v1"
BOOTSTRAP_TARGET = "d0b70a077bcaa64c401837eb0e9b6f27035210a0"
TRUSTED_CHECKER_LOADER = """\
import re
import subprocess
import sys

candidate, archive, repo = sys.argv[1:4]
if re.fullmatch(r"[0-9a-f]{40}", candidate) is None:
    raise SystemExit(1)
if re.fullmatch(r"[0-9a-f]{40}", archive) is None:
    raise SystemExit(1)
path = "openspec/changes/allow-post-archive-bootstrap-binding/checks/verify_archive.py"
result = subprocess.run(
    ["git", "-C", repo, "cat-file", "blob", f"{candidate}:{path}"],
    check=False,
    capture_output=True,
    timeout=30,
)
if result.returncode != 0:
    sys.stderr.buffer.write(result.stderr)
    raise SystemExit(1)
source_name = f"{candidate}:{path}"
sys.argv = [source_name, "--repo", repo, "--candidate", candidate, "--archive", archive]
exec(compile(result.stdout, source_name, "exec"), {"__name__": "__main__", "__file__": source_name})
"""


spec = importlib.util.spec_from_file_location("post_archive_profile_runner", RUNNER_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load profile runner")
runner = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = runner
spec.loader.exec_module(runner)


def run_git(
    repo: Path,
    *args: str,
    check: bool = True,
    input_bytes: bytes | None = None,
) -> subprocess.CompletedProcess[bytes]:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=False,
        input=input_bytes,
        capture_output=True,
        timeout=30,
    )
    if check and result.returncode != 0:
        raise AssertionError(
            f"git {' '.join(args)} failed: "
            f"{(result.stderr or result.stdout).decode('utf-8', 'replace').strip()}"
        )
    return result


def git_text(repo: Path, *args: str) -> str:
    return run_git(repo, *args).stdout.decode("utf-8").strip()


class GitHistoryFixture:
    def __init__(self) -> None:
        self._temp = tempfile.TemporaryDirectory(prefix="bootstrap-binding-test-")
        self.repo = Path(self._temp.name) / "repo"
        self.repo.mkdir()
        run_git(self.repo, "init", "-q")
        run_git(self.repo, "config", "user.name", "Bootstrap Test")
        run_git(self.repo, "config", "user.email", "bootstrap@example.invalid")
        self._copy_candidate_tree()
        self._seed_historical_blobs()
        self.candidate = self._commit("candidate")
        self.archive_sha: str | None = None
        self.binding_sha: str | None = None

    def close(self) -> None:
        self._temp.cleanup()

    def _copy(self, relative: Path) -> None:
        source = REPO_ROOT / relative
        target = self.repo / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)

    def _copy_candidate_tree(self) -> None:
        shutil.copytree(REPO_ROOT / CHANGE_REL, self.repo / CHANGE_REL)
        self._copy(CANONICAL_REL)
        for relative in PROTECTED_PATHS:
            self._copy(relative)

    def _seed_historical_blobs(self) -> None:
        bindings = json.loads((REPO_ROOT / BINDINGS_REL).read_text(encoding="utf-8"))
        for binding in bindings["bindings"]:
            for field in ("tool_source_blob", "executor_source_blob"):
                blob = binding[field]
                data = run_git(REPO_ROOT, "cat-file", "blob", blob).stdout
                actual = run_git(self.repo, "hash-object", "-w", "--stdin", input_bytes=data)
                self_id = actual.stdout.decode("ascii").strip()
                if self_id != blob:
                    raise AssertionError("seeded historical blob ID mismatch")

    def _commit(self, message: str) -> str:
        run_git(self.repo, "add", "-A")
        run_git(self.repo, "commit", "-q", "-m", message)
        return git_text(self.repo, "rev-parse", "HEAD")

    def _bootstrap_binding(self) -> dict[str, str]:
        registry = json.loads((self.repo / REGISTRY_REL).read_text(encoding="utf-8"))
        profile = next(item for item in registry["profiles"] if item["profile_id"] == BOOTSTRAP_ID)
        return {
            "registry_version": "mechanical-profiles/v1",
            "profile_id": BOOTSTRAP_ID,
            "change_name": profile["change_name"],
            "target_sha": BOOTSTRAP_TARGET,
            "profile_definition_sha256": runner.sha256_json(profile),
            "tool_source_path": WRAPPER_REL.as_posix(),
            "tool_source_blob": git_text(self.repo, "rev-parse", f"HEAD:{WRAPPER_REL.as_posix()}"),
            "executor_source_path": EXECUTOR_REL.as_posix(),
            "executor_source_blob": git_text(
                self.repo, "rev-parse", f"HEAD:{EXECUTOR_REL.as_posix()}"
            ),
        }

    def _append_binding(self, mutation: tuple[str, str] | None = None) -> None:
        document = json.loads((self.repo / BINDINGS_REL).read_text(encoding="utf-8"))
        binding = self._bootstrap_binding()
        if mutation is not None:
            binding[mutation[0]] = mutation[1]
        document["bindings"].append(binding)
        (self.repo / BINDINGS_REL).write_text(
            json.dumps(document, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
        )

    def archive(
        self,
        *,
        canonical: str = "exact",
        mutate_path: Path | None = None,
        extra_path: str | None = None,
        wrong_parent: bool = False,
        merge: bool = False,
        bind_same_stage: bool = False,
        replace_authority: str | None = None,
    ) -> str:
        if wrong_parent:
            (self.repo / "intervening.txt").write_text("unverified\n", encoding="utf-8")
            self._commit("intervening")
        (self.repo / ARCHIVE_REL.parent).mkdir(parents=True, exist_ok=True)
        run_git(self.repo, "mv", CHANGE_REL.as_posix(), ARCHIVE_REL.as_posix())
        expected = (self.repo / ARCHIVE_REL / "checks/expected-canonical-loop-engineering-control-plane-spec.md").read_text(
            encoding="utf-8"
        )
        if canonical == "extra":
            expected += "unverified-extra\n"
        elif canonical == "missing":
            expected = expected[:-1]
        elif canonical == "reordered":
            lines = expected.splitlines(keepends=True)
            lines[-2], lines[-1] = lines[-1], lines[-2]
            expected = "".join(lines)
        (self.repo / CANONICAL_REL).write_text(expected, encoding="utf-8")
        if mutate_path is not None:
            path = self.repo / mutate_path
            path.write_text(path.read_text(encoding="utf-8") + "smuggled\n", encoding="utf-8")
        if extra_path is not None:
            path = self.repo / extra_path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("smuggled\n", encoding="utf-8")
        if bind_same_stage:
            self._append_binding()
        if replace_authority == "runner":
            (self.repo / EXECUTOR_REL).write_text(
                "class ProfileError(ValueError):\n"
                "    pass\n\n"
                "def validate_archive_commit(*args, **kwargs):\n"
                "    return 'trusted-by-A'\n",
                encoding="utf-8",
            )
        elif replace_authority == "checker":
            (self.repo / ARCHIVE_REL / "checks/verify_archive.py").write_text(
                "#!/usr/bin/env python3\nprint('archive-gate=PASS')\n",
                encoding="utf-8",
            )
        elif replace_authority is not None:
            raise AssertionError(f"unknown authority replacement: {replace_authority}")
        run_git(self.repo, "add", "-A")
        if merge:
            candidate_tree = git_text(self.repo, "rev-parse", f"{self.candidate}^{{tree}}")
            side = run_git(
                self.repo,
                "commit-tree",
                candidate_tree,
                "-p",
                self.candidate,
                input_bytes=b"side\n",
            ).stdout.decode("ascii").strip()
            tree = git_text(self.repo, "write-tree")
            archive = run_git(
                self.repo,
                "commit-tree",
                tree,
                "-p",
                self.candidate,
                "-p",
                side,
                input_bytes=b"merge archive\n",
            ).stdout.decode("ascii").strip()
            run_git(self.repo, "reset", "--hard", "-q", archive)
            self.archive_sha = archive
        else:
            self.archive_sha = self._commit("archive")
        return self.archive_sha

    def bind(
        self,
        *,
        mutation: tuple[str, str] | None = None,
        extra_path: bool = False,
    ) -> str:
        self._append_binding(mutation)
        if extra_path:
            (self.repo / "binding-smuggle.txt").write_text("smuggled\n", encoding="utf-8")
        self.binding_sha = self._commit("binding")
        return self.binding_sha

    def touch_binding_later(self) -> None:
        path = self.repo / BINDINGS_REL
        path.write_text(path.read_text(encoding="utf-8") + "\n", encoding="utf-8")
        self._commit("later binding edit")

    def drift_executor_later(self) -> None:
        path = self.repo / EXECUTOR_REL
        path.write_text(path.read_text(encoding="utf-8") + "# drift\n", encoding="utf-8")
        self._commit("later executor drift")

    def run_archive_gate(self) -> subprocess.CompletedProcess[str]:
        if self.archive_sha is None:
            raise AssertionError("archive not created")
        return subprocess.run(
            [
                sys.executable,
                "-c",
                TRUSTED_CHECKER_LOADER,
                self.candidate,
                self.archive_sha,
                str(self.repo),
            ],
            cwd=self.repo,
            env=dict(os.environ, PYTHONDONTWRITEBYTECODE="1"),
            check=False,
            capture_output=True,
            text=True,
            timeout=30,
        )

    def load(self) -> object:
        return runner.load_control_plane(
            self.repo,
            self.repo / REGISTRY_REL,
            self.repo / BINDINGS_REL,
            verify_git_blobs=True,
        )


class FixtureTestCase(unittest.TestCase):
    fixture: GitHistoryFixture

    def setUp(self) -> None:
        self.fixture = GitHistoryFixture()

    def tearDown(self) -> None:
        self.fixture.close()


class ArchiveGateTests(FixtureTestCase):
    def assert_archive_gate_rejected(self, **kwargs: object) -> None:
        self.fixture.archive(**kwargs)
        result = self.fixture.run_archive_gate()
        self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
        self.assertEqual(result.stdout, "")
        self.assertEqual(len(result.stderr.rstrip("\n").splitlines()), 1)

    def test_archive_gate_accepts_exact_fixture_bytes(self) -> None:
        self.fixture.archive()
        result = self.fixture.run_archive_gate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "archive-gate=PASS\n")
        self.assertEqual(result.stderr, "")

    def test_archive_gate_fixture_is_base_plus_approved_delta(self) -> None:
        canonical = (REPO_ROOT / CANONICAL_REL).read_text(encoding="utf-8").rstrip()
        delta = (
            REPO_ROOT
            / "openspec/changes/allow-post-archive-bootstrap-binding/specs/loop-engineering-control-plane/spec.md"
        ).read_text(encoding="utf-8")
        marker = "## ADDED Requirements\n\n"
        self.assertTrue(delta.startswith(marker))
        expected = canonical + "\n\n" + delta.removeprefix(marker).rstrip() + "\n"
        self.assertEqual((REPO_ROOT / EXPECTED_REL).read_text(encoding="utf-8"), expected)

    def test_archive_gate_rejects_extra_canonical_byte(self) -> None:
        self.assert_archive_gate_rejected(canonical="extra")

    def test_archive_gate_rejects_missing_canonical_byte(self) -> None:
        self.assert_archive_gate_rejected(canonical="missing")

    def test_archive_gate_rejects_reordered_canonical_bytes(self) -> None:
        self.assert_archive_gate_rejected(canonical="reordered")

    def test_archive_gate_rejects_smuggled_judge(self) -> None:
        self.assert_archive_gate_rejected(
            mutate_path=Path("tools/lifecycle-receipts/verify_receipt.py")
        )

    def test_archive_gate_rejects_smuggled_protected_control_sources(self) -> None:
        for relative in (
            REGISTRY_REL,
            BINDINGS_REL,
            WRAPPER_REL,
            EXECUTOR_REL,
            Path(".agents/skills/order-run-loop/SKILL.md"),
            Path(".agents/skills/order-run-loop/references/self-evolution.md"),
        ):
            with self.subTest(path=relative):
                fixture = GitHistoryFixture()
                try:
                    fixture.archive(mutate_path=relative)
                    result = fixture.run_archive_gate()
                    self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
                finally:
                    fixture.close()

    def test_archive_gate_rejects_a_controlled_runner_validator(self) -> None:
        self.assert_archive_gate_rejected(replace_authority="runner")

    def test_archive_gate_rejects_a_controlled_checker(self) -> None:
        self.assert_archive_gate_rejected(replace_authority="checker")

    def test_archive_gate_rejects_extra_path(self) -> None:
        self.assert_archive_gate_rejected(extra_path="smuggled.txt")

    def test_archive_gate_rejects_wrong_parent(self) -> None:
        self.assert_archive_gate_rejected(wrong_parent=True)

    def test_archive_gate_rejects_merge_parent(self) -> None:
        self.assert_archive_gate_rejected(merge=True)

    def test_archive_gate_rejects_non_r100_rename(self) -> None:
        self.assert_archive_gate_rejected(mutate_path=ARCHIVE_REL / "proposal.md")

    def test_archive_gate_rejects_wrong_head(self) -> None:
        self.fixture.archive()
        (self.fixture.repo / "later.txt").write_text("later\n", encoding="utf-8")
        self.fixture._commit("later")
        result = self.fixture.run_archive_gate()
        self.assertEqual(result.returncode, 1)
        self.assertIn("HEAD", result.stderr)

    def test_archive_gate_rejects_dirty_worktree(self) -> None:
        self.fixture.archive()
        (self.fixture.repo / "dirty.txt").write_text("dirty\n", encoding="utf-8")
        result = self.fixture.run_archive_gate()
        self.assertEqual(result.returncode, 1)
        self.assertIn("clean", result.stderr)


class BootstrapBindingTests(FixtureTestCase):
    def test_candidate_three_bindings_remain_valid(self) -> None:
        loaded = self.fixture.load()
        self.assertEqual(len(loaded.bindings), 3)
        self.assertEqual(list(runner.EXPECTED_PROFILE_TARGETS), [
            "self-evolution-v1",
            "old-menu-artifact-fail-v1",
            "menu-supersession-v1",
        ])

    def test_exact_later_bootstrap_binding_is_admitted(self) -> None:
        self.fixture.archive()
        self.fixture.bind()
        loaded = self.fixture.load()
        self.assertEqual(len(loaded.bindings), 4)
        self.assertEqual(loaded.bindings[BOOTSTRAP_ID]["target_sha"], BOOTSTRAP_TARGET)

    def test_premature_and_same_stage_bindings_fail_closed(self) -> None:
        self.fixture.bind()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

        self.fixture.close()
        self.fixture = GitHistoryFixture()
        self.fixture.archive(bind_same_stage=True)
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_binding_field_mutations_fail_closed(self) -> None:
        for field, value in (
            ("target_sha", "f" * 40),
            ("profile_definition_sha256", "0" * 64),
            ("tool_source_blob", "0" * 40),
            ("executor_source_blob", "0" * 40),
        ):
            with self.subTest(field=field):
                fixture = GitHistoryFixture()
                try:
                    fixture.archive()
                    fixture.bind(mutation=(field, value))
                    with self.assertRaises(runner.ProfileError):
                        fixture.load()
                finally:
                    fixture.close()

    def test_binding_source_path_mutations_fail_closed(self) -> None:
        for field, value in (
            ("tool_source_path", "tools/lifecycle-receipts/profiles/self_evolution.py"),
            ("executor_source_path", "tools/lifecycle-receipts/verify_receipt.py"),
        ):
            with self.subTest(field=field):
                fixture = GitHistoryFixture()
                try:
                    fixture.archive()
                    fixture.bind(mutation=(field, value))
                    with self.assertRaises(runner.ProfileError):
                        fixture.load()
                finally:
                    fixture.close()

    def test_duplicate_fifth_binding_fails_closed(self) -> None:
        self.fixture.archive()
        self.fixture._append_binding()
        self.fixture._append_binding()
        self.fixture._commit("duplicate fifth binding")
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_binding_commit_extra_path_fails_closed(self) -> None:
        self.fixture.archive()
        self.fixture.bind(extra_path=True)
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_later_binding_edit_fails_closed(self) -> None:
        self.fixture.archive()
        self.fixture.bind()
        self.fixture.touch_binding_later()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_current_executor_drift_fails_closed(self) -> None:
        self.fixture.archive()
        self.fixture.bind()
        self.fixture.drift_executor_later()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_receipt_supplied_commands_are_not_loader_inputs(self) -> None:
        signature = runner.validate_bindings_document.__annotations__
        self.assertNotIn("receipt_command", signature)
        self.assertNotIn("receipt_provenance", signature)


if __name__ == "__main__":
    unittest.main()
