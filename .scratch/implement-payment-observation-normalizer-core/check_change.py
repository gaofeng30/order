#!/usr/bin/env python3
from __future__ import annotations

import re
import subprocess
from pathlib import Path


BASE = "5e937f3599a16f4813d6021f4cd2dd637c3156a2"
ROOT = Path(__file__).resolve().parents[2]
SCRATCH = ".scratch/implement-payment-observation-normalizer-core/"
PACKAGE = "services/api/internal/paymentobservation/"


def git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def fail(message: str) -> None:
    raise SystemExit(f"PAYMENT_OBSERVATION_CHANGE_GATE=FAIL reason={message}")


if git("rev-parse", BASE) != BASE:
    fail("base-does-not-resolve")
if git("merge-base", BASE, "HEAD") != BASE:
    fail("base-is-not-ancestor")

changed = [line for line in git("diff", "--name-only", f"{BASE}...HEAD").splitlines() if line]
if not changed:
    fail("empty-candidate-diff")
outside = [path for path in changed if not (path.startswith(SCRATCH) or path.startswith(PACKAGE))]
if outside:
    fail("owned-path-violation")
if any(path in {"go.mod", "go.sum"} for path in changed):
    fail("module-dependency-changed")
if any("miniprogram-gates" in path for path in changed):
    fail("forbidden-miniprogram-gate")

required = {
    f"{SCRATCH}ticket.md",
    f"{SCRATCH}spec.md",
    f"{SCRATCH}tasks.md",
    f"{SCRATCH}mutation-test.sh",
    f"{SCRATCH}verify-mysql.sh",
    f"{SCRATCH}check_change.py",
    f"{PACKAGE}normalize.go",
    f"{PACKAGE}types.go",
    f"{PACKAGE}normalize_test.go",
}
if missing := sorted(required.difference(changed)):
    fail("missing-required-artifact")

production = "\n".join(
    (ROOT / PACKAGE / filename).read_text() for filename in ("types.go", "normalize.go")
)
observation_body = re.search(r"type Observation struct \{(?P<body>.*?)\n\}", production, re.S)
if observation_body is None or "Source" in observation_body.group("body"):
    fail("source-leaked-into-observation")
canonical_parts = production.split("func canonicalBytes(", 1)
if len(canonical_parts) != 2:
    fail("canonical-function-missing")
persistence_surface = observation_body.group("body") + canonical_parts[1]
for token in (
    "Payer",
    "OpenID",
    "Phone",
    "Certificate",
    "Signature",
    "RawBody",
    "NotificationID",
    "TradeStateDescription",
    "BankType",
    "Attach",
):
    if re.search(rf"\b{re.escape(token)}\b", persistence_surface):
        fail("forbidden-observation-data-surface")

for required_token in (
    "func Normalize(expected Expectation, input Input) (Observation, error)",
    '"order.payment-observation.v1"',
    "sha256.Sum256",
    'SourceCallback    Source = "CALLBACK"',
    'SourceActiveQuery Source = "ACTIVE_QUERY"',
    'ValidationRejectedMismatch Validation = "REJECTED_MISMATCH"',
    'ErrorUnsupportedTradeState  ErrorKind = "UNSUPPORTED_TRADE_STATE"',
):
    if required_token not in production:
        fail("missing-frozen-interface-token")

status = git("status", "--porcelain")
if status:
    fail("worktree-not-clean")

print(f"PAYMENT_OBSERVATION_CHANGE_GATE=PASS files={len(changed)} owned_only=yes pii_surface=minimal")
