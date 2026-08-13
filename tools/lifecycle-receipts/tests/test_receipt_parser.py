from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = ROOT / "verify_receipt.py"
SCHEMA_PATH = ROOT / "receipt-schema-v1.json"
CASES = json.loads(
    (Path(__file__).parent / "fixtures" / "receipt-cases.json").read_text(encoding="utf-8")
)

if not MODULE_PATH.is_file():
    raise FileNotFoundError(f"receipt checker is missing: {MODULE_PATH}")

spec = importlib.util.spec_from_file_location("lifecycle_receipt_checker", MODULE_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("cannot load receipt checker")
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


def canonical(value: object) -> str:
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def values(verdict: str) -> dict[str, str]:
    sha_a = "1" * 40
    sha_b = "2" * 40
    empty_trees = canonical({})
    common = {
        "receipt_schema": "archived-lifecycle-receipt/v1",
        "change_name": "demo-change",
        "archive_path": "openspec/changes/archive/2026-08-13-demo-change",
        "lifecycle_state": "ARCHIVED",
        "expected_historical_verdict": verdict,
        "failure_fingerprint": "none",
        "subsequent_gates": "PASS",
        "superseded_by": "none",
        "supersedes": "none",
        "repo_base_sha": "0" * 40,
        "runner_skill_git_blob": "3" * 40,
        "runner_skill_sha256": "4" * 64,
        "runner_version": "unversioned",
        "candidate_sha": sha_a,
        "integrated_sha": sha_a,
        "integration_validation": "PASS",
        "archive_sha": sha_b,
        "archive_validation": "PASS",
        "owned_paths_json": canonical(["openspec/changes/demo-change/**"]),
        "profile_id": "demo-change-v1",
        "profile_binding_sha256": "9" * 64,
        "recorded_attestation_trust": "UNTRUSTED_FOR_MECHANICAL_PASS",
        "recorded_attestation_json": canonical(
            {"claimed_verdict": "PASS", "provenance": "writer:self-attestation"}
        ),
        "mechanical_verification": "REQUIRED_DERIVED",
        "product_trees_json": empty_trees,
        "candidate_checkpoint_sha256": "5" * 64,
        "candidate_tasks_sha256": "6" * 64,
        "candidate_open_tasks_json": canonical(["6.1", "7.1"]),
        "post_candidate_tasks_json": canonical({"6.1": "PASS", "7.1": "REQUIRED_DERIVED"}),
        "receipt_head_verification": "REQUIRED_DERIVED",
        "observation_counts_json": canonical(
            {"candidate": 0, "checker": 1, "environment": 0, "external": 0}
        ),
        "draft_screen_decisions_json": canonical(["none"]),
        "retrospective_json": canonical(["all evidence classes reviewed"]),
        "unverified_boundaries_json": canonical([CASES["literal_value"]]),
    }
    if verdict == "FAILED":
        common.update(
            failure_fingerprint=CASES["old_failure_fingerprint"],
            subsequent_gates="NOT_RUN",
            superseded_by="7" * 40,
            integration_validation="NOT_RUN",
            archive_validation="NOT_RUN",
            post_candidate_tasks_json=canonical({"6.1": "NOT_RUN", "7.1": "NOT_RUN"}),
        )
    elif verdict == "VERIFIED_SUPERSESSION":
        common["supersedes"] = "8" * 40
    return common


def render(data: dict[str, str], *, fenced: bool = False) -> str:
    schema = checker.load_schema(SCHEMA_PATH)
    lines = ["# Archived lifecycle receipt", ""]
    if fenced:
        lines += ["```text", "## Receipt fields", "change_verdict: FAILED", "```", ""]
    lines.append("## Receipt fields")
    lines.extend(f"{field['name']}: {data[field['name']]}" for field in schema["fields"])
    lines += ["", "## Retrospective", "Machine fields above are authoritative.", ""]
    return "\n".join(lines)


class ParserTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.schema = checker.load_schema(SCHEMA_PATH)

    def parse(self, text: str, *, crlf: bool = False) -> dict[str, object]:
        if crlf:
            text = text.replace("\n", "\r\n")
        return checker.parse_receipt_bytes(text.encode(), self.schema)

    def test_all_three_forms_and_lf_crlf(self) -> None:
        for verdict in CASES["verdicts"]:
            with self.subTest(verdict=verdict):
                text = render(values(verdict))
                self.assertEqual(self.parse(text), self.parse(text, crlf=True))
                self.assertEqual(self.parse(text)["expected_historical_verdict"], verdict)

    def test_fenced_lookalike_and_literal_shell_text_are_data(self) -> None:
        parsed = self.parse(render(values("VERIFIED"), fenced=True))
        self.assertEqual(parsed["expected_historical_verdict"], "VERIFIED")
        self.assertEqual(
            parsed["recorded_attestation_trust"], "UNTRUSTED_FOR_MECHANICAL_PASS"
        )
        self.assertEqual(parsed["unverified_boundaries_json"], [CASES["literal_value"]])

    def test_missing_duplicate_unknown_out_of_order_and_malformed_fail(self) -> None:
        data = values("VERIFIED")
        base = render(data)
        line = f"runner_version: {data['runner_version']}\n"
        pair = f"runner_skill_sha256: {data['runner_skill_sha256']}\n{line}"
        cases = [
            base.replace(line, ""),
            base.replace(line, line + line),
            base.replace(line, "unknown_field: value\n" + line),
            base.replace(pair, line + f"runner_skill_sha256: {data['runner_skill_sha256']}\n"),
            base.replace("owned_paths_json: [", "owned_paths_json: {"),
        ]
        for receipt in cases:
            with self.subTest(receipt=receipt[:90]), self.assertRaises(checker.ReceiptError):
                self.parse(receipt)

    def test_stale_state_false_old_pass_and_persisted_head_fail(self) -> None:
        stale = values("VERIFIED")
        stale["lifecycle_state"] = "CANDIDATE"
        false_pass = values("FAILED")
        false_pass["integration_validation"] = "PASS"
        persisted = values("VERIFIED")
        persisted["receipt_head_verification"] = "PASS"
        for data in (stale, false_pass, persisted):
            with self.assertRaises(checker.ReceiptError):
                self.parse(render(data))

    def test_missing_duplicate_or_empty_retrospective_fails(self) -> None:
        base = render(values("VERIFIED"))
        cases = [
            base.replace("\n## Retrospective\n", "\n## Notes\n"),
            base + "\n## Retrospective\nagain\n",
            base.replace("Machine fields above are authoritative.\n", ""),
        ]
        for receipt in cases:
            with self.assertRaises(checker.ReceiptError):
                self.parse(receipt)


if __name__ == "__main__":
    unittest.main()
