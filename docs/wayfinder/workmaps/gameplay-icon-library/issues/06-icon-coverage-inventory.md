# Inventory exact gameplay icon coverage and resource marker kinds

Type: task
Status: resolved
Blocked by: none

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Inventory exact gameplay icon coverage and resource marker kinds](https://github.com/VipPro67/GuWorld/issues/439).

## Question

Inventory the exact records and UI paths needed to estimate the first asset batches. Inspect the current Gu catalog against its per-`Gu ID` coverage, identify item/resource records whose current icon resolver falls back to a generic category, and enumerate resource/deposit marker kinds and their render states in exploration. Record counts, representative ID lists, existing valid `image_url` coverage, and relevant frontend entry points in a concise report linked from this issue. Do not create or modify artwork/code.

## Resolution

Completed the read-only audit. The catalog has 3,472 Gu IDs, 897 items (no item `image_url` values), and 47 resource point types. Existing resolvers provide broad family/category fallbacks but not per-ID art for all records. Full counts, fallback group totals, marker IDs/states, and frontend pointers are in [the audit report](../research/439-icon-coverage-audit.md).
