#!/usr/bin/env python3
"""Verify the retired PRD pointer and the complete 0818 baseline delta."""

from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]
POINTER = ROOT / "docs/product/online-ordering-system-prd.md"
DELTA = (
    ROOT
    / "openspec/changes/adopt-0818-prd-baseline/specs/mvp-product-baseline/spec.md"
)


def fail(message: str) -> None:
    print(f"BASELINE_CHECK=FAIL {message}")
    raise SystemExit(1)


def require(text: str, token: str, source: Path) -> None:
    if token not in text:
        fail(f"missing required token {token!r} in {source.relative_to(ROOT)}")


def main() -> None:
    for path in (POINTER, DELTA):
        if not path.is_file():
            fail(f"missing required file {path.relative_to(ROOT)}")

    pointer = POINTER.read_text(encoding="utf-8")
    pointer_lines = pointer.splitlines()
    if len(pointer_lines) > 40:
        fail(
            "retired PRD must be a thin pointer with at most 40 lines; "
            f"found {len(pointer_lines)} lines"
        )

    pointer_tokens = (
        "已废止",
        "唯一有效产品基线",
        "docs/product/online-ordering-system-prd-0818.md",
        "§1–§14",
        "docs/product/online-ordering-system-prd-0818-review.md",
        "仅作为客户裁决证据",
        "## 13.2 外部 Gate",
        "openspec/specs/mvp-product-baseline/spec.md",
        "External readiness follows one twelve-gate chain",
        "十二 Gate",
    )
    for token in pointer_tokens:
        require(pointer, token, POINTER)

    target = "docs/product/online-ordering-system-prd-0818.md"
    if pointer.count(target) != 1:
        fail(f"retired PRD must point to the 0818 PRD exactly once; found {pointer.count(target)}")

    forbidden_headings = (
        "## 1. 项目背景",
        "## 3. 一期范围与文档依据",
        "## 5. 用户端功能",
        "## 6. 商户后台功能",
        "## 7. 订单、库存、支付和履约规则",
        "## 12. 正式业务不变量",
        "# 15. 前端实现规格",
    )
    for heading in forbidden_headings:
        if heading in pointer:
            fail(f"retired PRD still contains active legacy body heading {heading!r}")

    delta = DELTA.read_text(encoding="utf-8")
    for index in range(1, 17):
        marker = f"（I{index}）"
        count = delta.count(marker)
        if count != 1:
            fail(
                f"invariant marker {marker} must appear exactly once; found {count}"
            )

    removed_requirements = (
        "### Requirement: Inventory is keyed by service date, meal period, and product",
        "### Requirement: Order submission uses a bounded atomic soft hold",
        "### Requirement: Orders use one nine-state production state machine",
        "### Requirement: Employee price is an optional fixed per-product amount",
        "### Requirement: Every first-phase order uses one fixed pickup slot",
        "### Requirement: Merchant permissions use four server-enforced roles",
    )
    require(delta, "## REMOVED Requirements", DELTA)
    for requirement in removed_requirements:
        require(delta, requirement, DELTA)

    deprecated_terms = (
        "数量库存",
        "软预占",
        "即时取餐",
        "接单",
        "九态",
        "四角色",
        "会员等级",
        "优惠券",
        "逐商品员工价",
    )
    for term in deprecated_terms:
        require(delta, term, DELTA)

    blocker_mappings = (
        "P1 阻塞营业状态操作归属",
        "P2 阻塞跨营业日未取订单运营处置时限",
        "P3 阻塞 PC 后台扫码登录会话与设备信任",
        "P4 阻塞附加手机号数量模型",
        "P5 阻塞全局折扣率生效时机",
    )
    for mapping in blocker_mappings:
        require(delta, mapping, DELTA)

    print(
        "BASELINE_CHECK=PASS "
        f"pointer_lines={len(pointer_lines)} invariants=16 removed_requirements=6 blockers=5"
    )


if __name__ == "__main__":
    main()
