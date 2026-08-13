#!/usr/bin/env python3
"""Run positive and negative fixtures for all seven checker contracts."""

from __future__ import annotations

import json
import sys
import tempfile
from pathlib import Path

from checker_contract import (
    archive_value,
    awk_contains,
    bounded_health_poll,
    literal_tool_roundtrip,
    parse_markdown_field,
    validate_safe_temp,
    zero_match_count,
)


class RegressionError(RuntimeError):
    pass


def require(condition: bool, message: str) -> None:
    if not condition:
        raise RegressionError(message)


def require_error(expected_fragment: str, action) -> None:
    try:
        action()
    except Exception as exc:  # fixture asserts a specific safe failure
        require(expected_fragment in str(exc), f"unexpected error: {exc}")
        return
    raise RegressionError(f"expected error containing {expected_fragment!r}")


def sequence_probe(states: list[bool]):
    iterator = iter(states)
    return lambda: next(iterator, states[-1])


def fixed_clock() -> float:
    return 0.0


def run(fixtures: dict) -> None:
    case = fixtures["zero_match"]
    positive = case["positive"]
    require(
        zero_match_count(positive["text"], positive["pattern"]) == positive["expected"],
        "zero-match positive fixture failed",
    )
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: zero_match_count(negative["text"], negative["pattern"]),
    )

    case = fixtures["markdown_field"]
    positive = case["positive"]
    require(
        parse_markdown_field(positive["markdown"], positive["key"]) == positive["expected"],
        "Markdown-field positive fixture failed",
    )
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: parse_markdown_field(negative["markdown"], negative["key"]),
    )

    case = fixtures["literal_backticks"]
    positive = case["positive"]
    require(
        literal_tool_roundtrip(positive["value"], positive["transport"]) == positive["value"],
        "literal-backtick argv fixture failed",
    )
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: literal_tool_roundtrip(negative["value"], negative["transport"]),
    )

    case = fixtures["awk_case"]
    for polarity in ("positive", "negative"):
        fixture = case[polarity]
        require(
            awk_contains(
                fixture["text"],
                fixture["needle"],
                case_sensitive=fixture["case_sensitive"],
            )
            is fixture["expected"],
            f"awk-case {polarity} fixture failed",
        )

    case = fixtures["bounded_health"]
    positive = case["positive"]
    require(
        bounded_health_poll(
            sequence_probe(positive["states"]),
            max_attempts=positive["max_attempts"],
            timeout_seconds=positive["timeout_seconds"],
            clock=fixed_clock,
        )
        == positive["expected_attempt"],
        "bounded-health positive fixture failed",
    )
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: bounded_health_poll(
            sequence_probe(negative["states"]),
            max_attempts=negative["max_attempts"],
            timeout_seconds=negative["timeout_seconds"],
            clock=fixed_clock,
        ),
    )

    case = fixtures["safe_temp"]
    positive = case["positive"]
    parent = Path(tempfile.gettempdir()).resolve()
    with tempfile.TemporaryDirectory(prefix=positive["prefix"], dir=parent) as raw_path:
        path = Path(raw_path)
        entries = validate_safe_temp(path, parent)
        require(len(entries) == positive["expected_entries"], "safe-temp positive fixture failed")
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: validate_safe_temp(Path(negative["target"]), parent),
    )

    case = fixtures["archive_newline"]
    positive = case["positive"]
    require(
        archive_value(positive["output"], positive["expected"]) == positive["expected"],
        "archive-newline positive fixture failed",
    )
    negative = case["negative"]
    require_error(
        negative["error"],
        lambda: archive_value(negative["output"], negative["expected"]),
    )


def main() -> int:
    fixture_path = Path(__file__).with_name("fixtures") / "checker-cases.json"
    try:
        fixtures = json.loads(fixture_path.read_text(encoding="utf-8"))
        run(fixtures)
    except Exception as exc:
        print(f"checker_regressions=FAIL: {exc}", file=sys.stderr)
        return 1
    print("checker_regressions=PASS: positive=7 negative=7")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
