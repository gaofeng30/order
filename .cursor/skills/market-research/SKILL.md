---
name: market-research
description: Research current market prices, vendor fees, SaaS/service costs, freelance rates, and contract cost assumptions. Use when the user asks to price a project, compare market quotes, validate budget items, or create a quotation with sourced ranges and uncertainty notes.
---

# Market Research

## Use When

Use this skill for pricing, vendor cost checks, fee breakdowns, contract quotations, and market rate validation.

## Workflow

1. Define the pricing target:
   - Buyer type, region, project scope, timeframe, and whether prices should include tax.
   - Separate one-time fees, recurring fees, pass-through third-party costs, and optional services.

2. Source hierarchy:
   - Prefer official pricing pages, platform docs, government/regulated fee pages, and cloud vendor calculators.
   - Use vendor blogs, agency pages, marketplaces, and forum posts only as secondary market signals.
   - For volatile cloud or promo prices, record both activity price and normal renewal/upgrade risk.

3. Triangulate:
   - Find at least 2 independent sources for non-official market rates.
   - For each line item, produce a low/typical/high range when exact pricing is not fixed.
   - Mark whether the recommended price is cost pass-through, labor price, risk buffer, or negotiation anchor.

4. Separate quote logic:
   - Do not mix third-party fees into developer revenue unless the contract says the developer advances them.
   - Show both "client pays directly" and "developer includes in package" options when relevant.
   - Add tax/invoice treatment as a separate line.

5. Check reasonableness:
   - Compare total price against scope, timeline, maintenance burden, and delivery risk.
   - Flag suspiciously low prices, hidden renewal costs, and items that require client credentials or legal qualifications.
   - Ask the user to confirm assumptions that materially change the quote.

## Output Format

Use this structure:

1. Conclusion: recommended pricing stance and biggest risks.
2. Market evidence: sourced ranges with source type and confidence.
3. Proposed quote: line items, adopted price, who pays, and why.
4. Items to confirm with the user/client.
5. Contract wording notes for ambiguous or risky costs.

## Rules

- Always distinguish "market cost" from "your quoted price".
- Always identify recurring costs and renewal risks.
- Never present uncertain market prices as official facts.
- For legal, tax, invoice, and licensing questions, give practical drafting notes but recommend final review by a qualified professional.
