from __future__ import annotations

import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]


class TrustBoundaryRedTests(unittest.TestCase):
    def test_receipt_schema_cannot_treat_attestation_or_provenance_as_pass(self) -> None:
        schema = json.loads((ROOT / "receipt-schema-v1.json").read_text(encoding="utf-8"))
        names = [field["name"] for field in schema["fields"]]
        self.assertNotIn("candidate_verification", names)
        self.assertNotIn("verifier_provenance", names)
        self.assertIn("recorded_attestation_json", names)
        self.assertIn("mechanical_verification", names)

    def test_receipt_cannot_supply_its_own_command_or_independent_verdict(self) -> None:
        schema = json.loads((ROOT / "receipt-schema-v1.json").read_text(encoding="utf-8"))
        names = {field["name"] for field in schema["fields"]}
        self.assertTrue(
            {"profile_id", "profile_binding_sha256", "recorded_attestation_json"}.issubset(names)
        )
        self.assertTrue({"command", "argv", "provenance", "independent_verdict"}.isdisjoint(names))

    def test_four_controlled_profiles_and_separate_bindings_exist(self) -> None:
        registry = json.loads(
            (ROOT / "mechanical-profiles-v1.json").read_text(encoding="utf-8")
        )
        identifiers = {profile["profile_id"] for profile in registry["profiles"]}
        self.assertEqual(
            identifiers,
            {
                "self-evolution-v1",
                "old-menu-artifact-fail-v1",
                "menu-supersession-v1",
                "lifecycle-receipt-control-v1",
            },
        )
        bindings = json.loads(
            (ROOT / "mechanical-bindings-v1.json").read_text(encoding="utf-8")
        )
        bound = {binding["profile_id"] for binding in bindings["bindings"]}
        self.assertEqual(
            bound,
            {"self-evolution-v1", "old-menu-artifact-fail-v1", "menu-supersession-v1"},
        )
        self.assertNotIn("lifecycle-receipt-control-v1", bound)


if __name__ == "__main__":
    unittest.main()
