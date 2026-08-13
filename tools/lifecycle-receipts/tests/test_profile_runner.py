from __future__ import annotations

import copy
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
RUNNER_PATH = ROOT / "profile_runner.py"

if not RUNNER_PATH.is_file():
    raise FileNotFoundError(f"controlled profile runner is missing: {RUNNER_PATH}")

spec = importlib.util.spec_from_file_location("lifecycle_profile_runner", RUNNER_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load profile runner")
runner = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = runner
spec.loader.exec_module(runner)


class RegistryContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.registry_path = ROOT / "mechanical-profiles-v1.json"
        self.bindings_path = ROOT / "mechanical-bindings-v1.json"
        self.registry = json.loads(self.registry_path.read_text(encoding="utf-8"))
        self.bindings = json.loads(self.bindings_path.read_text(encoding="utf-8"))

    def test_exact_registry_and_three_historical_bindings_validate(self) -> None:
        loaded = runner.load_control_plane(
            ROOT.parent.parent,
            self.registry_path,
            self.bindings_path,
            verify_git_blobs=False,
        )
        self.assertEqual(len(loaded.profiles), 4)
        self.assertEqual(len(loaded.bindings), 3)
        self.assertNotIn("lifecycle-receipt-control-v1", loaded.bindings)

    def test_each_definition_contract_mutation_fails_closed(self) -> None:
        profile = self.registry["profiles"][0]
        mutations = {
            "target-result": ("result_on_success", "UNVERIFIED"),
            "argv": ("steps", []),
            "network": ("network", True),
            "write": ("write_scope", "repository"),
            "timeout": ("timeout_seconds", 0),
            "output": ("output_limit_bytes", 0),
            "isolation": ("isolation_contract", "os-enforced"),
        }
        for label, (field, value) in mutations.items():
            altered = copy.deepcopy(self.registry)
            altered["profiles"][0][field] = value
            with self.subTest(label=label), self.assertRaises(runner.ProfileError):
                runner.validate_registry_document(altered)

        altered = copy.deepcopy(self.registry)
        altered["profiles"][0]["steps"][0]["argv"] = "python -c pass"
        with self.assertRaises(runner.ProfileError):
            runner.validate_registry_document(altered)

    def test_binding_target_definition_tool_and_executor_mutations_fail_closed(self) -> None:
        for field, value in (
            ("target_sha", "f" * 40),
            ("profile_definition_sha256", "0" * 64),
            ("tool_source_blob", "0" * 40),
            ("executor_source_blob", "0" * 40),
        ):
            altered = copy.deepcopy(self.bindings)
            altered["bindings"][0][field] = value
            with self.subTest(field=field), self.assertRaises(runner.ProfileError):
                runner.validate_bindings_document(
                    altered,
                    runner.validate_registry_document(self.registry),
                    ROOT.parent.parent,
                    verify_git_blobs=False,
                )

    def test_receipt_supplied_profile_override_is_not_an_input(self) -> None:
        signature = runner.run_profile.__annotations__
        self.assertNotIn("receipt_command", signature)
        self.assertNotIn("receipt_provenance", signature)


class BoundedExecutorTests(unittest.TestCase):
    def test_literal_shell_text_is_one_argv_value(self) -> None:
        with tempfile.TemporaryDirectory(prefix="profile-argv-test-") as directory:
            root = Path(directory)
            marker = root / "executed"
            literal = f"`touch {marker}`;$(touch {marker})"
            result = runner.run_bounded_argv(
                [sys.executable, "-c", "import sys;print(sys.argv[1])", literal],
                cwd=root,
                env={"HOME": str(root), "TMPDIR": str(root)},
                timeout_seconds=2,
                output_limit_bytes=4096,
                output_root=root,
            )
            self.assertEqual(result.returncode, 0)
            self.assertEqual(result.stdout.strip(), literal)
            self.assertFalse(marker.exists())

    def test_timeout_and_output_overflow_fail_closed(self) -> None:
        with tempfile.TemporaryDirectory(prefix="profile-bound-test-") as directory:
            root = Path(directory)
            with self.assertRaisesRegex(runner.ProfileError, "timeout"):
                runner.run_bounded_argv(
                    [sys.executable, "-c", "import time;time.sleep(5)"],
                    cwd=root,
                    env={"HOME": str(root), "TMPDIR": str(root)},
                    timeout_seconds=0.1,
                    output_limit_bytes=4096,
                    output_root=root,
                )
            with self.assertRaisesRegex(runner.ProfileError, "output"):
                runner.run_bounded_argv(
                    [sys.executable, "-c", "print('x'*10000)"],
                    cwd=root,
                    env={"HOME": str(root), "TMPDIR": str(root)},
                    timeout_seconds=2,
                    output_limit_bytes=128,
                    output_root=root,
                )

    def test_unsafe_link_is_rejected_before_cleanup(self) -> None:
        with tempfile.TemporaryDirectory(prefix="profile-link-test-") as directory:
            root = Path(directory)
            target = root / "target"
            target.mkdir()
            (root / "escape").symlink_to(target, target_is_directory=True)
            with self.assertRaisesRegex(runner.ProfileError, "symbolic link"):
                runner.assert_safe_tree(root)


if __name__ == "__main__":
    unittest.main()
