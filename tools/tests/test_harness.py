#!/usr/bin/env python3
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from typing import Optional


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


class EntrypointRedTest(unittest.TestCase):
    def test_harness_entrypoint_exists(self) -> None:
        self.assertTrue(HARNESS.is_file(), "tools/harness is missing")


@unittest.skipUnless(HARNESS.is_file(), "requires tools/harness implementation")
class RepositoryContractTest(unittest.TestCase):
    def test_harness_has_only_four_thin_commands(self) -> None:
        result = run(str(HARNESS), "--help", cwd=REPO_ROOT, check=True)
        self.assertIn("{status,check,checkpoint,observe}", result.stdout)
        source = HARNESS.read_text(encoding="utf-8")
        for forbidden in ("git push", "openspec archive", "deploy", "send_message_to_thread"):
            self.assertNotIn(forbidden, source)

    def test_protected_governance_does_not_route_through_unjudged_harness(self) -> None:
        root_rules = (REPO_ROOT / "AGENTS.md").read_text(encoding="utf-8")
        runner = (REPO_ROOT / ".agents/skills/order-run-loop/SKILL.md").read_text(encoding="utf-8")
        for token in ("./tools/harness status", "./tools/harness checkpoint", "./tools/harness check", "./tools/harness observe"):
            self.assertNotIn(token, root_rules)
            self.assertNotIn(token, runner)
        for token in (
            "$order-plan-change",
            "$order-implement-tdd",
            "$order-verify-change",
            "$order-integrate-change",
            "at most two active change lane slots",
            "BLOCKED_EXTERNAL",
            "third consecutive occurrence",
            "lifecycle-receipt.md",
        ):
            self.assertIn(token, runner)

    def test_protected_governance_and_stage_skills_remain_byte_unchanged(self) -> None:
        paths = [
            "AGENTS.md",
            ".agents/skills/order-run-loop/SKILL.md",
            ".agents/skills/order-run-loop/references/self-evolution.md",
            ".agents/skills/order-plan-change/SKILL.md",
            ".agents/skills/order-implement-tdd/SKILL.md",
            ".agents/skills/order-verify-change/SKILL.md",
            ".agents/skills/order-integrate-change/SKILL.md",
        ]
        result = run(
            "git",
            "diff",
            "--quiet",
            "d817aeb3ac5de29d2695ed17ed6277b737ba3ee8",
            "--",
            *paths,
            cwd=REPO_ROOT,
        )
        self.assertEqual(result.returncode, 0, result.stderr)


@unittest.skipUnless(HARNESS.is_file(), "requires tools/harness implementation")
class HarnessCliTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory(prefix="order-harness-test-")
        self.repo = Path(self.temp.name) / "repo"
        self.repo.mkdir()
        run("git", "init", "-q", cwd=self.repo, check=True)
        run("git", "config", "user.name", "Harness Test", cwd=self.repo, check=True)
        run("git", "config", "user.email", "harness@example.invalid", cwd=self.repo, check=True)
        (self.repo / "README.md").write_text("base fixture\n", encoding="utf-8")
        (self.repo / "openspec" / "changes").mkdir(parents=True)
        (self.repo / "openspec" / "changes" / ".keep").write_text("\n", encoding="utf-8")
        run("git", "add", ".", cwd=self.repo, check=True)
        run("git", "commit", "-qm", "base fixture", cwd=self.repo, check=True)
        self.base = run("git", "rev-parse", "HEAD", cwd=self.repo, check=True).stdout.strip()
        self.write_change(
            "alpha",
            """## 1. Work

- [x] 1.1 completed task
- [ ] 1.2 current task
- [ ] 1.3 blocked task
- [ ] 1.4 todo task
""",
        )
        run("git", "add", ".", cwd=self.repo, check=True)
        run("git", "commit", "-qm", "fixture", cwd=self.repo, check=True)
        self.head = run("git", "rev-parse", "HEAD", cwd=self.repo, check=True).stdout.strip()

    def tearDown(self) -> None:
        self.temp.cleanup()

    def write_change(self, name: str, tasks: str) -> None:
        change = self.repo / "openspec" / "changes" / name
        change.mkdir(parents=True)
        (change / "tasks.md").write_text(tasks, encoding="utf-8")

    def harness(self, *args: str, cwd: Optional[Path] = None) -> subprocess.CompletedProcess[str]:
        return run(str(HARNESS), *args, cwd=cwd or self.repo)

    def checkpoint(self, *args: str) -> subprocess.CompletedProcess[str]:
        return self.harness(
            "checkpoint",
            "alpha",
            "--next",
            "continue focused work",
            "--evidence",
            "fixture evidence",
            *args,
        )

    def state_path(self) -> Path:
        common = run(
            "git", "rev-parse", "--path-format=absolute", "--git-common-dir", cwd=self.repo, check=True
        ).stdout.strip()
        return Path(common) / "codex-harness" / "state.json"

    def observations_path(self) -> Path:
        return self.state_path().with_name("observations.jsonl")

    def test_status_reports_unknown_and_json_is_clean(self) -> None:
        human = self.harness("status")
        self.assertEqual(human.returncode, 0, human.stderr)
        self.assertIn("alpha  UNKNOWN", human.stdout)
        machine = self.harness("status", "--json")
        self.assertEqual(machine.returncode, 0, machine.stderr)
        payload = json.loads(machine.stdout)
        self.assertEqual(payload["changes"][0]["name"], "alpha")
        self.assertEqual(payload["changes"][0]["lifecycle"], "UNKNOWN")
        self.assertNotIn("WHAT:", machine.stdout)

    def test_check_fails_closed_for_missing_checkpoint(self) -> None:
        result = self.harness("check")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("WHAT:", result.stderr)
        self.assertIn("WHY:", result.stderr)
        self.assertIn("FIX:", result.stderr)
        self.assertIn("alpha", result.stderr)

    def test_task_states_are_derived_without_copying_titles(self) -> None:
        result = self.checkpoint(
            "--state",
            "DRAFT",
            "--current-task",
            "1.2",
            "--blocked-task",
            "1.3",
            "--blocker-kind",
            "DEPENDENCY",
            "--blocker",
            "waiting for declared dependency",
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(self.harness("status", "--json").stdout)
        tasks = {task["id"]: task["state"] for task in payload["changes"][0]["tasks"]}
        self.assertEqual(
            tasks,
            {"1.1": "DONE", "1.2": "DOING", "1.3": "BLOCKED", "1.4": "TODO"},
        )
        stored = json.loads(self.state_path().read_text(encoding="utf-8"))
        serialized = json.dumps(stored, ensure_ascii=False)
        self.assertNotIn("completed task", serialized)
        self.assertNotIn("todo task", serialized)

    def test_invalid_task_reference_fails_without_rewriting_state(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        before = self.state_path().read_bytes()
        result = self.checkpoint("--state", "DRAFT", "--current-task", "9.9")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state_path().read_bytes(), before)
        self.assertIn("WHAT:", result.stderr)

    def test_illegal_lifecycle_jump_fails_without_rewriting_state(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        before = self.state_path().read_bytes()
        result = self.checkpoint("--state", "IMPLEMENTING")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state_path().read_bytes(), before)
        self.assertIn("illegal lifecycle transition", result.stderr)

    def test_legacy_integrated_bootstrap_requires_ancestry(self) -> None:
        direct = self.checkpoint(
            "--state",
            "INTEGRATED",
            "--candidate-sha",
            self.head,
            "--integrated-sha",
            self.head,
        )
        self.assertNotEqual(direct.returncode, 0)
        bad = self.checkpoint(
            "--state",
            "INTEGRATED",
            "--candidate-sha",
            "0" * 40,
            "--integrated-sha",
            self.head,
            "--bootstrap",
        )
        self.assertNotEqual(bad.returncode, 0)
        good = self.checkpoint(
            "--state",
            "INTEGRATED",
            "--candidate-sha",
            self.head,
            "--integrated-sha",
            self.head,
            "--bootstrap",
        )
        self.assertEqual(good.returncode, 0, good.stderr)
        self.assertEqual(self.harness("check").returncode, 0)

    def test_state_is_shared_by_a_second_worktree(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        sibling = Path(self.temp.name) / "sibling"
        run("git", "worktree", "add", "--detach", str(sibling), self.base, cwd=self.repo, check=True)
        status = self.harness("status", "--json", cwd=sibling)
        self.assertEqual(status.returncode, 0, status.stderr)
        payload = json.loads(status.stdout)
        self.assertEqual(payload["changes"][0]["lifecycle"], "DRAFT")
        self.assertEqual(payload["changes"][0]["source_worktree"], str(self.repo.resolve()))
        checked = self.harness("check", cwd=sibling)
        self.assertEqual(checked.returncode, 0, checked.stderr)

    def test_lock_prevents_checkpoint_overwrite(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        before = self.state_path().read_bytes()
        lock = self.state_path().with_name("write.lock")
        lock.write_text("held\n", encoding="utf-8")
        result = self.checkpoint("--state", "APPROVED")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state_path().read_bytes(), before)
        self.assertIn("another harness writer", result.stderr)
        lock.unlink()

    def test_observation_is_append_only_and_visible(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        good = self.harness(
            "observe",
            "--class",
            "checker",
            "--summary",
            "parser rejected a valid task",
            "--evidence",
            "reproduced by fixture alpha",
            "--next",
            "screen at the next module boundary",
        )
        self.assertEqual(good.returncode, 0, good.stderr)
        before = self.observations_path().read_bytes()
        bad = self.harness(
            "observe",
            "--class",
            "wish",
            "--summary",
            "invalid",
            "--evidence",
            "invalid",
            "--next",
            "invalid",
        )
        self.assertNotEqual(bad.returncode, 0)
        self.assertEqual(self.observations_path().read_bytes(), before)
        payload = json.loads(self.harness("status", "--json").stdout)
        self.assertEqual(payload["observations"]["count"], 1)
        self.assertEqual(payload["observations"]["by_class"], {"checker": 1})

    def test_corrupt_observations_fail_check_and_preserve_file(self) -> None:
        self.assertEqual(self.checkpoint("--state", "DRAFT").returncode, 0)
        self.observations_path().parent.mkdir(parents=True, exist_ok=True)
        self.observations_path().write_text("{broken\n", encoding="utf-8")
        before = self.observations_path().read_bytes()
        result = self.harness("check")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.observations_path().read_bytes(), before)
        self.assertIn("WHAT:", result.stderr)


if __name__ == "__main__":
    unittest.main()
