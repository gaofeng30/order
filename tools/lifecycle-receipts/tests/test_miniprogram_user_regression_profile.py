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
RUNNER_REL = Path("tools/lifecycle-receipts/profile_runner.py")
REGISTRY_REL = Path("tools/lifecycle-receipts/mechanical-profiles-v1.json")
BINDINGS_REL = Path("tools/lifecycle-receipts/mechanical-bindings-v1.json")
WRAPPER_REL = Path("tools/lifecycle-receipts/profiles/miniprogram_user_regression_gate.py")
TEST_REL = Path("tools/lifecycle-receipts/tests/test_miniprogram_user_regression_profile.py")
CHANGE_REL = Path("openspec/changes/add-miniprogram-gate-receipt-profile")
ARCHIVE_REL = Path("openspec/changes/archive/2026-08-21-add-miniprogram-gate-receipt-profile")
CHECKER_REL = CHANGE_REL / "checks/verify_archives.py"
EXPECTED_PROFILE_CANONICAL_REL = (
    CHANGE_REL / "checks/expected-canonical-miniprogram-gate-lifecycle-profile-spec.md"
)
CANONICAL_PROFILE_REL = Path("openspec/specs/miniprogram-gate-lifecycle-profile/spec.md")
TARGET = "f5719e98690d0b1301ed567c8b616e074846b445"
TARGET_ARCHIVE = "b53cb520c4cac0505803a9d1e6dcc1807ac34540"
BASE = TARGET_ARCHIVE
PROFILE_ID = "miniprogram-user-regression-gate-v1"
CURRENT_BINDING_IDS = [
    "self-evolution-v1",
    "old-menu-artifact-fail-v1",
    "menu-supersession-v1",
    "lifecycle-receipt-control-v1",
]


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


runner = load_module("miniprogram_profile_runner_under_test", REPO_ROOT / RUNNER_REL)


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
        timeout=60,
    )
    if check and result.returncode != 0:
        raise AssertionError(
            f"git {' '.join(args)} failed: "
            f"{(result.stderr or result.stdout).decode('utf-8', 'replace').strip()}"
        )
    return result


def git_text(repo: Path, *args: str) -> str:
    return run_git(repo, *args).stdout.decode("utf-8").strip()


def profile_definition() -> dict[str, object]:
    return {
        "profile_id": PROFILE_ID,
        "change_name": "enforce-miniprogram-user-regression-gate",
        "result_on_success": "MECHANICAL_PASS",
        "network": False,
        "write_scope": "temp-only",
        "isolation_contract": "trusted-wrapper-audited",
        "required_tools": {
            "python": "Python 3.14.6",
            "git": "git version 2.53.0",
            "node": "v25.8.1",
            "npm": "11.11.0",
            "go": "go version go1.26.5 darwin/arm64",
        },
        "excluded_claims": [
            "actor independence",
            "mutable OpenSpec CLI behavior",
            "user-directory Skill validation",
            "UI2 and UI3",
            "real WeChat execution",
        ],
        "timeout_seconds": 1200,
        "output_limit_bytes": 1048576,
        "steps": [
            {
                "step_id": "miniprogram-double-gate-regression",
                "argv": [
                    "{python}",
                    "{profile_tool}",
                    "--repo",
                    "{worktree}",
                    "--temp-root",
                    "{profile_temp}",
                    "--git",
                    "{git}",
                    "--python",
                    "{python}",
                    "--node",
                    "{node}",
                    "--npm",
                    "{npm}",
                    "--go",
                    "{go}",
                    "--module-cache-download",
                    "{module_cache_download}",
                ],
                "cwd": ".",
                "env_allowlist": [
                    "HOME",
                    "TMPDIR",
                    "XDG_CACHE_HOME",
                    "PATH",
                    "PYTHONDONTWRITEBYTECODE",
                    "LANG",
                    "LC_ALL",
                    "GOENV",
                    "GOTOOLCHAIN",
                    "GOPROXY",
                    "GOSUMDB",
                    "GOCACHE",
                    "GOFLAGS",
                    "GOMODCACHE",
                    "npm_config_cache",
                    "npm_config_offline",
                    "npm_config_audit",
                    "npm_config_fund",
                    "npm_config_update_notifier",
                ],
                "timeout_seconds": 1140,
                "expected_exit": 0,
                "expected_stdout_exact": f"{PROFILE_ID}=MECHANICAL_PASS\n",
                "output_limit_bytes": 1048576,
            }
        ],
    }


def five_profile_registry() -> dict[str, object]:
    document = json.loads((REPO_ROOT / REGISTRY_REL).read_text(encoding="utf-8"))
    if not any(item["profile_id"] == PROFILE_ID for item in document["profiles"]):
        document["profiles"].append(profile_definition())
    return document


class RegistryRedTests(unittest.TestCase):
    def test_01_unbound_fifth_profile_is_valid(self) -> None:
        loaded = runner.validate_registry_document(five_profile_registry())
        self.assertEqual(list(loaded), [*CURRENT_BINDING_IDS, PROFILE_ID])

    def test_02_current_four_bindings_remain_valid_with_unbound_profile(self) -> None:
        fixture = ProfileHistoryFixture()
        try:
            control = fixture.load()
            self.assertEqual(list(control.bindings), CURRENT_BINDING_IDS)
            self.assertNotIn(PROFILE_ID, control.bindings)
        finally:
            fixture.close()

    def test_03_premature_fifth_binding_fails_closed(self) -> None:
        registry = five_profile_registry()
        bindings = json.loads((REPO_ROOT / BINDINGS_REL).read_text(encoding="utf-8"))
        premature = {
            "registry_version": "mechanical-profiles/v1",
            "profile_id": PROFILE_ID,
            "change_name": "enforce-miniprogram-user-regression-gate",
            "target_sha": TARGET,
            "profile_definition_sha256": runner.sha256_json(profile_definition()),
            "tool_source_path": WRAPPER_REL.as_posix(),
            "tool_source_blob": "0" * 40,
            "executor_source_path": RUNNER_REL.as_posix(),
            "executor_source_blob": "0" * 40,
        }
        bindings["bindings"].append(premature)
        with tempfile.TemporaryDirectory(prefix="miniprogram-profile-premature-") as raw:
            root = Path(raw)
            registry_path = root / "registry.json"
            bindings_path = root / "bindings.json"
            registry_path.write_text(json.dumps(registry), encoding="utf-8")
            bindings_path.write_text(json.dumps(bindings), encoding="utf-8")
            with self.assertRaises(runner.ProfileError):
                runner.load_control_plane(
                    REPO_ROOT,
                    registry_path,
                    bindings_path,
                    verify_git_blobs=True,
                )

    def test_04_writer_uncommitted_accepts_only_the_unbound_profile_stage(self) -> None:
        control = runner.load_control_plane(
            REPO_ROOT,
            REPO_ROOT / REGISTRY_REL,
            REPO_ROOT / BINDINGS_REL,
            verify_git_blobs=False,
        )
        self.assertEqual(list(control.profiles), [*CURRENT_BINDING_IDS, PROFILE_ID])
        self.assertEqual(list(control.bindings), CURRENT_BINDING_IDS)


class ArchiveAuthorityRedTests(unittest.TestCase):
    def test_04_exact_governance_archive_is_accepted(self) -> None:
        self.assertTrue((REPO_ROOT / CHECKER_REL).is_file(), "archive authority is missing")
        checker = load_module("miniprogram_archive_checker", REPO_ROOT / CHECKER_REL)
        result = checker.validate_governance_archive(
            REPO_ROOT,
            authority_candidate=None,
            target=TARGET,
            archive=TARGET_ARCHIVE,
        )
        self.assertEqual(result, "2026-08-21-enforce-miniprogram-user-regression-gate")

    def test_05_wrong_governance_parent_is_rejected(self) -> None:
        self.assertTrue((REPO_ROOT / CHECKER_REL).is_file(), "archive authority is missing")
        checker = load_module("miniprogram_archive_checker_negative", REPO_ROOT / CHECKER_REL)
        with self.assertRaises(checker.ArchiveViolation):
            checker.validate_governance_archive(
                REPO_ROOT,
                authority_candidate=None,
                target=TARGET_ARCHIVE,
                archive=TARGET,
            )


class ProfileHistoryFixture:
    def __init__(self) -> None:
        self._temp = tempfile.TemporaryDirectory(prefix="miniprogram-profile-history-")
        self.repo = Path(self._temp.name) / "repo"
        subprocess.run(
            ["git", "clone", "-q", "--shared", "--no-checkout", str(REPO_ROOT), str(self.repo)],
            check=True,
            capture_output=True,
            timeout=60,
        )
        run_git(self.repo, "checkout", "-q", "--detach", BASE)
        run_git(self.repo, "config", "user.name", "Mini Program Profile Test")
        run_git(self.repo, "config", "user.email", "miniprogram-profile@example.invalid")
        self._copy_candidate_paths()
        self.candidate = self._commit("profile candidate")
        self.archive_sha: str | None = None
        self.binding_sha: str | None = None

    def close(self) -> None:
        self._temp.cleanup()

    def _copy(self, relative: Path) -> None:
        source = REPO_ROOT / relative
        if not source.is_file():
            raise AssertionError(f"candidate source missing: {relative}")
        target = self.repo / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)

    def _copy_candidate_paths(self) -> None:
        target_change = self.repo / CHANGE_REL
        if target_change.exists():
            shutil.rmtree(target_change)
        shutil.copytree(REPO_ROOT / CHANGE_REL, target_change)
        for relative in (REGISTRY_REL, RUNNER_REL, WRAPPER_REL, TEST_REL):
            self._copy(relative)

    def _commit(self, message: str) -> str:
        run_git(self.repo, "add", "-A")
        run_git(self.repo, "commit", "-q", "-m", message)
        return git_text(self.repo, "rev-parse", "HEAD")

    def archive(self, *, wrong_parent: bool = False, extra_path: bool = False) -> str:
        if wrong_parent:
            (self.repo / "intervening.txt").write_text("unverified\n", encoding="utf-8")
            self._commit("intervening")
        (self.repo / ARCHIVE_REL.parent).mkdir(parents=True, exist_ok=True)
        run_git(self.repo, "mv", CHANGE_REL.as_posix(), ARCHIVE_REL.as_posix())
        expected = self.repo / ARCHIVE_REL / "checks/expected-canonical-miniprogram-gate-lifecycle-profile-spec.md"
        canonical = self.repo / CANONICAL_PROFILE_REL
        canonical.parent.mkdir(parents=True, exist_ok=True)
        canonical.write_bytes(expected.read_bytes())
        if extra_path:
            (self.repo / "archive-smuggle.txt").write_text("smuggled\n", encoding="utf-8")
        self.archive_sha = self._commit("profile archive")
        return self.archive_sha

    def _binding(self) -> dict[str, str]:
        registry = json.loads((self.repo / REGISTRY_REL).read_text(encoding="utf-8"))
        profile = next(item for item in registry["profiles"] if item["profile_id"] == PROFILE_ID)
        return {
            "registry_version": "mechanical-profiles/v1",
            "profile_id": PROFILE_ID,
            "change_name": profile["change_name"],
            "target_sha": TARGET,
            "profile_definition_sha256": runner.sha256_json(profile),
            "tool_source_path": WRAPPER_REL.as_posix(),
            "tool_source_blob": git_text(self.repo, "rev-parse", f"{self.candidate}:{WRAPPER_REL.as_posix()}"),
            "executor_source_path": RUNNER_REL.as_posix(),
            "executor_source_blob": git_text(self.repo, "rev-parse", f"{self.candidate}:{RUNNER_REL.as_posix()}"),
        }

    def bind(self, *, mutate: tuple[str, str] | None = None, extra_path: bool = False) -> str:
        document = json.loads((self.repo / BINDINGS_REL).read_text(encoding="utf-8"))
        binding = self._binding()
        if mutate is not None:
            binding[mutate[0]] = mutate[1]
        document["bindings"].append(binding)
        (self.repo / BINDINGS_REL).write_text(
            json.dumps(document, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
        )
        if extra_path:
            (self.repo / "binding-smuggle.txt").write_text("smuggled\n", encoding="utf-8")
        self.binding_sha = self._commit("governance binding")
        return self.binding_sha

    def load(self):
        return runner.load_control_plane(
            self.repo,
            self.repo / REGISTRY_REL,
            self.repo / BINDINGS_REL,
            verify_git_blobs=True,
        )


class PostArchiveBindingRedTests(unittest.TestCase):
    def setUp(self) -> None:
        self.fixture = ProfileHistoryFixture()

    def tearDown(self) -> None:
        self.fixture.close()

    def test_06_exact_profile_archive_and_later_binding_are_admitted(self) -> None:
        self.fixture.archive()
        self.fixture.bind()
        loaded = self.fixture.load()
        self.assertEqual(list(loaded.bindings), [*CURRENT_BINDING_IDS, PROFILE_ID])

    def test_07_binding_before_profile_archive_is_rejected(self) -> None:
        self.fixture.bind()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_08_wrong_parent_archive_is_rejected(self) -> None:
        self.fixture.archive(wrong_parent=True)
        self.fixture.bind()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_09_archive_or_binding_smuggling_is_rejected(self) -> None:
        self.fixture.archive(extra_path=True)
        self.fixture.bind()
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

        self.fixture.close()
        self.fixture = ProfileHistoryFixture()
        self.fixture.archive()
        self.fixture.bind(extra_path=True)
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()

    def test_10_stale_binding_source_is_rejected(self) -> None:
        self.fixture.archive()
        self.fixture.bind(mutate=("tool_source_blob", "0" * 40))
        with self.assertRaises(runner.ProfileError):
            self.fixture.load()


class WrapperContractRedTests(unittest.TestCase):
    def test_11_wrapper_accepts_exact_clean_target_and_declares_full_matrix(self) -> None:
        self.assertTrue((REPO_ROOT / WRAPPER_REL).is_file(), "profile wrapper is missing")
        wrapper = load_module("miniprogram_profile_wrapper", REPO_ROOT / WRAPPER_REL)
        with tempfile.TemporaryDirectory(prefix="miniprogram-wrapper-target-") as raw:
            repo = (Path(raw) / "repo").resolve()
            subprocess.run(
                ["git", "clone", "-q", "--shared", "--no-checkout", str(REPO_ROOT), str(repo)],
                check=True,
                capture_output=True,
                timeout=60,
            )
            run_git(repo, "checkout", "-q", "--detach", TARGET)
            wrapper.validate_target(repo, shutil.which("git") or "git")
            commands = wrapper.build_commands(
                python=sys.executable,
                node=shutil.which("node") or "node",
                npm=shutil.which("npm") or "npm",
                go=shutil.which("go") or "go",
            )
            python_env = wrapper.controlled_python_environment()
        flattened = [item for command in commands for item in command]
        self.assertEqual(python_env["PYTHONDONTWRITEBYTECODE"], "1")
        self.assertIn("tools.tests.test_miniprogram_gate", flattened)
        self.assertIn("tools.tests.test_harness", flattened)
        self.assertIn("apps/wechat-miniprogram", flattened)
        self.assertIn("-race", flattened)
        self.assertIn("services/api/scripts/smoke.sh", flattened)

    def test_12_wrapper_rejects_dirty_or_wrong_target(self) -> None:
        self.assertTrue((REPO_ROOT / WRAPPER_REL).is_file(), "profile wrapper is missing")
        wrapper = load_module("miniprogram_profile_wrapper_negative", REPO_ROOT / WRAPPER_REL)
        with self.assertRaises(SystemExit):
            wrapper.validate_target(REPO_ROOT, shutil.which("git") or "git")


if __name__ == "__main__":
    unittest.main()
