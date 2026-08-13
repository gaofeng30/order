#!/usr/bin/env python3
"""Portable checker primitives exercised by the self-evolution regressions."""

from __future__ import annotations

import re
import subprocess
import sys
from collections.abc import Callable
from pathlib import Path


def zero_match_count(text: str, pattern: str) -> int:
    try:
        compiled = re.compile(pattern)
    except re.error as exc:
        raise ValueError(f"invalid pattern: {exc}") from exc
    return sum(1 for line in text.splitlines() if compiled.search(line))


def parse_markdown_field(markdown: str, key: str) -> str:
    if re.fullmatch(r"[A-Za-z0-9_.-]+", key) is None:
        raise ValueError(f"invalid field key: {key!r}")

    field_pattern = re.compile(rf"^\s*{re.escape(key)}\s*:\s*(.*?)\s*$")
    table_pattern = re.compile(rf"^\s*\|\s*{re.escape(key)}\s*\|\s*(.*?)\s*\|\s*$")
    fence_character: str | None = None
    values: list[str] = []
    for line in markdown.splitlines():
        fence_match = re.match(r"^\s*(`{3,}|~{3,})", line)
        if fence_match:
            marker = fence_match.group(1)[0]
            if fence_character is None:
                fence_character = marker
            elif marker == fence_character:
                fence_character = None
            continue
        if fence_character is not None:
            continue
        match = field_pattern.fullmatch(line) or table_pattern.fullmatch(line)
        if match:
            values.append(match.group(1).strip())

    if not values:
        raise ValueError(f"missing field: {key}")
    if len(values) != 1:
        raise ValueError(f"duplicate field: {key}")
    return values[0]


def literal_tool_roundtrip(value: str, transport: str) -> str:
    if transport != "argv":
        raise ValueError(f"unsafe transport: {transport}")
    completed = subprocess.run(
        [sys.executable, "-c", "import sys; sys.stdout.write(sys.argv[1])", value],
        check=False,
        capture_output=True,
        text=True,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"tool invocation failed with exit {completed.returncode}")
    return completed.stdout


def awk_contains(text: str, needle: str, *, case_sensitive: bool) -> bool:
    haystack = "$0" if case_sensitive else "tolower($0)"
    query = "needle" if case_sensitive else "tolower(needle)"
    program = f"index({haystack}, {query}) {{ found=1 }} END {{ exit(found ? 0 : 1) }}"
    completed = subprocess.run(
        ["awk", "-v", f"needle={needle}", program],
        input=text,
        check=False,
        capture_output=True,
        text=True,
    )
    if completed.returncode == 0:
        return True
    if completed.returncode == 1:
        return False
    raise RuntimeError(f"awk invocation failed with exit {completed.returncode}: {completed.stderr.strip()}")


def bounded_health_poll(
    probe: Callable[[], bool],
    *,
    max_attempts: int,
    timeout_seconds: float,
    clock: Callable[[], float],
) -> int:
    if max_attempts <= 0:
        raise ValueError("max_attempts must be positive")
    if timeout_seconds <= 0:
        raise ValueError("timeout_seconds must be positive")

    started = clock()
    for attempt in range(1, max_attempts + 1):
        if clock() - started >= timeout_seconds:
            raise TimeoutError(
                f"health check did not recover within {attempt - 1} attempts/{timeout_seconds:g}s"
            )
        if probe():
            return attempt
    raise TimeoutError(
        f"health check did not recover within {max_attempts} attempts/{timeout_seconds:g}s"
    )


def validate_safe_temp(path: Path, expected_parent: Path) -> list[Path]:
    if not path.is_absolute():
        raise ValueError("unsafe temporary directory: target is not absolute")
    try:
        parent = expected_parent.resolve(strict=True)
        if path.is_symlink():
            raise ValueError("unsafe temporary directory: target is a symlink")
        target = path.resolve(strict=True)
    except OSError as exc:
        raise ValueError(f"unsafe temporary directory: cannot resolve target: {exc}") from exc
    if not parent.is_dir():
        raise ValueError("unsafe temporary directory: parent is not a directory")
    if target.parent != parent:
        raise ValueError("unsafe temporary directory: target is outside the expected parent")
    if not target.name.startswith("order-run-loop-"):
        raise ValueError("unsafe temporary directory: target name is not narrowly scoped")
    if not target.is_dir():
        raise ValueError("unsafe temporary directory: target is not a directory")
    entries = list(target.iterdir())
    if any(entry.is_symlink() for entry in entries):
        raise ValueError("unsafe temporary directory: target contains a symlink")
    return entries


def archive_value(output: str, expected: str) -> str:
    if output.endswith("\r\n"):
        value = output[:-2]
    elif output.endswith("\n"):
        value = output[:-1]
    else:
        raise ValueError("archive output missing trailing newline")
    if value != expected:
        raise ValueError(f"archive output mismatch: {value!r}")
    return value
