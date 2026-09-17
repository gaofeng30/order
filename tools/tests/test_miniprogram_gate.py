#!/usr/bin/env python3
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from typing import Dict, List, Optional


REPO_ROOT = Path(__file__).resolve().parents[2]
HARNESS = REPO_ROOT / "tools" / "harness"


def run(*argv: str, cwd: Path, check: bool = False) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        [*argv],
        cwd=cwd,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if check and result.returncode != 0:
        raise AssertionError(
            f"command failed: {argv}\nstdout={result.stdout}\nstderr={result.stderr}"
        )
    return result


class MiniprogramGateLifecycleTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory(prefix="order-miniprogram-gate-")
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        run("git", "init", "-q", cwd=self.repo, check=True)
        run("git", "branch", "-M", "main", cwd=self.repo, check=True)
        run("git", "config", "user.name", "Gate Test", cwd=self.repo, check=True)
        run("git", "config", "user.email", "gate@example.invalid", cwd=self.repo, check=True)
        (self.repo / "README.md").write_text("base\n", encoding="utf-8")
        (self.repo / "openspec" / "changes").mkdir(parents=True)
        run("git", "add", ".", cwd=self.repo, check=True)
        run("git", "commit", "-qm", "initial", cwd=self.repo, check=True)

    def tearDown(self) -> None:
        self.temp.cleanup()

    def activate_marker(self) -> str:
        marker = self.repo / "tools" / "miniprogram_gate.py"
        marker.parent.mkdir(parents=True, exist_ok=True)
        marker.write_text("# activation marker\n", encoding="utf-8")
        run("git", "add", str(marker.relative_to(self.repo)), cwd=self.repo, check=True)
        run("git", "commit", "-qm", "activate gate", cwd=self.repo, check=True)
        return self.sha()

    def sha(self, ref: str = "HEAD") -> str:
        return run("git", "rev-parse", ref, cwd=self.repo, check=True).stdout.strip()

    def harness(self, *args: str) -> subprocess.CompletedProcess[str]:
        return run(str(HARNESS), *args, cwd=self.repo)

    def tasks(self, incomplete: Optional[str] = None) -> str:
        stages = {
            "red": ("2.1", "red behavior"),
            "green": ("3.1", "green behavior"),
            "refactor": ("4.1", "refactor behavior"),
        }
        lines = ["## Gate tasks", ""]
        for stage, (task_id, title) in stages.items():
            marker = " " if stage == incomplete else "x"
            lines.append(f"- [{marker}] {task_id} {title}")
        lines.extend(["- [ ] 5.1 independent verification", ""])
        return "\n".join(lines)

    def manifest(
        self,
        base_sha: str,
        minimum: str = "UI2",
        native: Optional[List[str]] = None,
    ) -> Dict[str, object]:
        return {
            "schema_version": 1,
            "change": "alpha",
            "base_sha": base_sha,
            "tdd": {
                "red": {"task_id": "2.1", "command": "node --test red.test.js"},
                "green": {"task_id": "3.1", "command": "node --test red.test.js"},
                "refactor": {"task_id": "4.1", "command": "node --test red.test.js"},
            },
            "user_regression": {
                "minimum_ui_level": minimum,
                "native_capabilities": native or [],
                "scenarios": [
                    {
                        "id": "primary-journey",
                        "kind": "primary",
                        "steps": ["open the changed user journey"],
                        "expected": ["the intended user result is visible"],
                    },
                    {
                        "id": "adjacent-regression",
                        "kind": "regression",
                        "steps": ["open the adjacent unchanged journey"],
                        "expected": ["the adjacent user result remains unchanged"],
                    },
                ],
            },
        }

    def create_candidate(
        self,
        *,
        marker: bool = True,
        manifest: Optional[Dict[str, object]] = None,
        incomplete: Optional[str] = None,
        source: str = "Page({});\n",
    ) -> str:
        base_sha = self.activate_marker() if marker else self.sha()
        change = self.repo / "openspec" / "changes" / "alpha"
        change.mkdir(parents=True)
        (change / "proposal.md").write_text(
            "## Impact\n\nOwned paths: `apps/wechat-miniprogram/**`.\n",
            encoding="utf-8",
        )
        (change / "tasks.md").write_text(self.tasks(incomplete), encoding="utf-8")
        if manifest is not None:
            data = dict(manifest)
            data["base_sha"] = base_sha
            (change / "miniprogram-gates.json").write_text(
                json.dumps(data, ensure_ascii=False, indent=2) + "\n",
                encoding="utf-8",
            )
        app = self.repo / "apps" / "wechat-miniprogram" / "app.js"
        app.parent.mkdir(parents=True)
        app.write_text(source, encoding="utf-8")
        run("git", "add", ".", cwd=self.repo, check=True)
        run("git", "commit", "-qm", "candidate", cwd=self.repo, check=True)
        self.base_sha = base_sha
        self.candidate_sha = self.sha()
        return self.candidate_sha

    def advance_to_implementing(self) -> None:
        for state in ("DRAFT", "APPROVED", "IMPLEMENTING"):
            result = self.harness(
                "checkpoint",
                "alpha",
                "--state",
                state,
                "--next",
                "continue",
                "--evidence",
                f"{state} fixture evidence",
            )
            self.assertEqual(result.returncode, 0, result.stderr)

    def candidate_checkpoint(self, **extra: str) -> subprocess.CompletedProcess[str]:
        argv = [
            "checkpoint",
            "alpha",
            "--state",
            "CANDIDATE",
            "--candidate-sha",
            self.candidate_sha,
            "--next",
            "verify",
            "--evidence",
            "candidate fixture evidence",
        ]
        for key, value in extra.items():
            argv.extend([f"--{key.replace('_', '-')}", value])
        return self.harness(*argv)

    def write_receipt(
        self,
        *,
        candidate_sha: Optional[str] = None,
        ui_level: str = "UI2",
        scenario_ids: Optional[List[str]] = None,
    ) -> Path:
        ids = scenario_ids or ["primary-journey", "adjacent-regression"]
        receipt = {
            "schema_version": 1,
            "change": "alpha",
            "candidate_sha": candidate_sha or self.candidate_sha,
            "ui_level": ui_level,
            "environment": "controlled test runtime",
            "scenarios": [{"id": scenario_id, "result": "PASS"} for scenario_id in ids],
            "result": "PASS",
            "unverified_boundary": "outside this controlled fixture",
        }
        path = self.root / "user-regression-receipt.json"
        path.write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
        return path

    def independent_checkpoint(self, receipt: Optional[Path] = None) -> subprocess.CompletedProcess[str]:
        argv = [
            "checkpoint",
            "alpha",
            "--state",
            "INDEPENDENT_VERIFIED",
            "--candidate-sha",
            self.candidate_sha,
            "--next",
            "integrate",
            "--evidence",
            "repository verifier fixture evidence",
        ]
        if receipt is not None:
            argv.extend(["--user-regression-receipt", str(receipt)])
        return self.harness(*argv)

    def valid_manifest(self, minimum: str = "UI2", native: Optional[List[str]] = None) -> Dict[str, object]:
        return self.manifest("0" * 40, minimum=minimum, native=native)

    def prepare_valid_candidate(self, minimum: str = "UI2", native: Optional[List[str]] = None, source: str = "Page({});\n") -> None:
        self.create_candidate(manifest=self.valid_manifest(minimum, native), source=source)
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_candidate_without_manifest_is_rejected(self) -> None:
        self.create_candidate(manifest=None)
        self.advance_to_implementing()
        before = self.harness("status", "--json").stdout
        result = self.candidate_checkpoint()
        self.assertNotEqual(
            result.returncode,
            0,
            "Harness accepted a Mini Program candidate with no gate manifest",
        )
        self.assertIn("manifest", result.stderr.lower())
        after = json.loads(self.harness("status", "--json").stdout)
        self.assertEqual(after["changes"][0]["lifecycle"], "IMPLEMENTING")
        self.assertIn('"IMPLEMENTING"', before)

    def test_incomplete_tdd_task_is_rejected(self) -> None:
        self.create_candidate(manifest=self.valid_manifest(), incomplete="green")
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("green", result.stderr.lower())

    def test_ordinary_ui1_is_rejected(self) -> None:
        self.create_candidate(manifest=self.valid_manifest("UI1"))
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("UI2", result.stderr)

    def test_native_ui2_is_rejected(self) -> None:
        self.create_candidate(
            manifest=self.valid_manifest("UI2", ["wx.login"]),
            source="wx.login({});\n",
        )
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("UI3", result.stderr)

    def test_omitted_native_capability_is_rejected(self) -> None:
        self.create_candidate(manifest=self.valid_manifest("UI3"), source="wx.login({});\n")
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("wx.login", result.stderr)

    def test_missing_user_regression_receipt_is_rejected(self) -> None:
        self.prepare_valid_candidate()
        result = self.independent_checkpoint()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("receipt", result.stderr.lower())

    def test_stale_and_partial_receipts_are_rejected(self) -> None:
        self.prepare_valid_candidate()
        for receipt in (
            self._prepared_receipt(candidate_sha=self.base_sha),
            self._prepared_receipt(scenario_ids=["primary-journey"]),
        ):
            result = self.independent_checkpoint(receipt)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(
                json.loads(self.harness("status", "--json").stdout)["changes"][0]["lifecycle"],
                "CANDIDATE",
            )

    def _prepared_receipt(
        self,
        candidate_sha: Optional[str] = None,
        scenario_ids: Optional[List[str]] = None,
    ) -> Path:
        if not hasattr(self, "candidate_sha"):
            self.prepare_valid_candidate()
        path = self.write_receipt(candidate_sha=candidate_sha, scenario_ids=scenario_ids)
        copy = self.root / f"receipt-{len(list(self.root.glob('receipt-*.json')))}.json"
        copy.write_bytes(path.read_bytes())
        return copy

    def test_exact_ui2_receipt_advances_and_reports_gate(self) -> None:
        self.prepare_valid_candidate()
        result = self.independent_checkpoint(self.write_receipt())
        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(self.harness("status", "--json").stdout)
        self.assertEqual(payload["changes"][0]["lifecycle"], "INDEPENDENT_VERIFIED")
        self.assertEqual(payload["changes"][0]["user_regression"]["ui_level"], "UI2")
        self.assertEqual(payload["changes"][0]["user_regression"]["candidate_sha"], self.candidate_sha)

    def test_exact_ui3_native_receipt_advances(self) -> None:
        self.prepare_valid_candidate("UI3", ["wx.login"], "wx.login({});\n")
        result = self.independent_checkpoint(self.write_receipt(ui_level="UI3"))
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_external_blocker_rejects_promotion(self) -> None:
        self.create_candidate(manifest=self.valid_manifest())
        self.advance_to_implementing()
        result = self.candidate_checkpoint(
            blocker="real platform assets unavailable",
            blocker_kind="BLOCKED_EXTERNAL",
            blocked_task="5.1",
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        result = self.independent_checkpoint(self.write_receipt())
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("BLOCKED_EXTERNAL", result.stderr)

    def test_hand_injected_candidate_receipt_fails_check(self) -> None:
        self.prepare_valid_candidate()
        common = Path(
            run(
                "git",
                "rev-parse",
                "--path-format=absolute",
                "--git-common-dir",
                cwd=self.repo,
                check=True,
            ).stdout.strip()
        )
        state_path = common / "codex-harness" / "state.json"
        state = json.loads(state_path.read_text(encoding="utf-8"))
        state["changes"]["alpha"]["user_regression"] = {
            "candidate_sha": self.candidate_sha,
            "ui_level": "UI2",
            "scenario_ids": ["adjacent-regression", "primary-journey"],
            "environment": "manually injected state",
            "unverified_boundary": "not loaded through the receipt gate",
            "receipt_sha256": "0" * 64,
        }
        state_path.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")
        result = self.harness("check")
        self.assertNotEqual(
            result.returncode,
            0,
            "Harness accepted a receipt injected before independent verification",
        )
        self.assertIn("premature", result.stderr.lower())

    def test_candidate_change_invalidates_old_receipt(self) -> None:
        self.prepare_valid_candidate()
        result = self.independent_checkpoint(self.write_receipt())
        self.assertEqual(result.returncode, 0, result.stderr)
        result = self.harness(
            "checkpoint", "alpha", "--state", "IMPLEMENTING", "--candidate-sha", self.candidate_sha,
            "--next", "repair", "--evidence", "candidate invalidated",
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        app = self.repo / "apps" / "wechat-miniprogram" / "app.js"
        app.write_text("Page({ data: { repaired: true } });\n", encoding="utf-8")
        run("git", "add", str(app.relative_to(self.repo)), cwd=self.repo, check=True)
        run("git", "commit", "-qm", "replacement", cwd=self.repo, check=True)
        old_candidate = self.candidate_sha
        self.candidate_sha = self.sha()
        result = self.candidate_checkpoint()
        self.assertEqual(result.returncode, 0, result.stderr)
        stale = self.write_receipt(candidate_sha=old_candidate)
        result = self.independent_checkpoint(stale)
        self.assertNotEqual(result.returncode, 0)

    def test_pre_marker_candidate_keeps_existing_behavior(self) -> None:
        self.create_candidate(marker=False, manifest=None)
        self.advance_to_implementing()
        result = self.candidate_checkpoint()
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
