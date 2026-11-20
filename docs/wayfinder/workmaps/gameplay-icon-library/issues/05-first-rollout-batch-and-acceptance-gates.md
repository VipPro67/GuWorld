# Set the first asset rollout batch and acceptance gates

Type: grilling
Status: resolved
Blocked by: [Specify in-game mapping and rollout strategy](https://github.com/VipPro67/GuWorld/issues/437)

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Set the first asset rollout batch and acceptance gates](https://github.com/VipPro67/GuWorld/issues/438).

## Question

Once asset-to-data mapping is clear, which categories and records should ship in the first batch and subsequent batches, and what objective acceptance gates should each batch meet before moving into the web game? Balance visible player impact, missing/weak existing coverage, asset production volume, and integration risk. Keep this to rollout order and completion gates; the pipeline and coverage rules are resolved in earlier tickets.

## Resolution

### Rollout order

1. Create grouped original artwork informed by representative samples of each asset group and the agreed 24 px readability rules.
2. First production batch delivered: 47 resource-point markers, the 44 distinct `produce_key` resource IDs produced by those points, and replacement artwork for the 35 Gu IDs that currently resolve to legacy/heuristic images. Rank variants share canonical artwork; item IDs that represent the same common material archetype may also share artwork, with rank/name shown in UI.

The exact IDs and their CSV provenance are listed in `research/first-rollout-batch-plan.json`. Grouped sheets, style-reference paths, prompts, and per-ID output paths are recorded in `art/game-icons/sources/manifest.json` and `frontend/src/data/game-icon-manifest.json`.
3. Subsequent Gu batches: prioritize Gu IDs appearing most often across inventory, wiki, and shop surfaces, then cover all remaining Gu IDs in manageable tranches. Keep shared item-category fallbacks for the long tail outside the selected resource identities.

### Acceptance gates

- Review grouped artwork by category and confirm the silhouettes remain readable at 24 px.
- Every delivered icon has a clear silhouette at 24 px and keeps the approved transparent pixel-art treatment.
- Every batch record is mapped by stable ID in the manifest; its PNG master and WebP runtime file exist and pass automated name, dimension, and transparency checks.
- The first tranche passes `cd frontend && npm run validate:game-icons:batch`, which checks all 126 planned IDs in addition to each file's metadata and image properties.
- No batch record is left on an unintended heuristic/category fallback, and existing valid CSV `image_url` overrides continue to resolve.
- Inventory and exploration map use the shared ID resolvers with existing fallback behavior retained for IDs outside this tranche.
