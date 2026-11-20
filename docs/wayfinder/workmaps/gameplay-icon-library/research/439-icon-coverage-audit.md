# Gameplay icon coverage audit

Context: [Inventory exact gameplay icon coverage and resource marker kinds](https://github.com/VipPro67/GuWorld/issues/439)

Read-only audit of the current catalog and frontend resolver paths. Counts are from the checked-in CSV/assets at audit time; runtime database imports can differ until the datasets are reimported.

## Gu catalog and artwork coverage

- `backend/data/Gu.csv`: 4,713 rows, 3,472 distinct `id` values, 466 IDs have multiple rank rows. Rank is a separate field for repeated IDs; the requested per-Gu-ID art target is therefore 3,472 identities, not 4,713 files.
- 51 distinct Gu IDs have non-empty `image_url` fields. `getGuIconUrl` currently accepts only ID/name and does not consult that field.
- Existing `frontend/public/assets/gu_icons/`: 438 root PNGs, 1,557 family-atlas images, and 15 custom images (2,010 files total across these directories).
- Evaluating current `getGuIconUrl(id, name)` over the 3,472 distinct IDs resolves 3,422 to shared family-atlas artwork, 15 to custom artwork, and 35 to legacy/heuristic artwork. Those 35 include `GU_ALL_PEOPLE_UNITED_I`, `GU_BLOOD_STAINED_ROBE_I`, `GU_CHAOS_CAVE_DRILL_I`, `GU_DANG_HUN_EXPLOSION_I`, `GU_FUTURE_SELF_I`, and `GU_YAMA_EMPEROR_I`; full current URL mapping can be regenerated from the resolver and catalog.
- The family mapping is generated in `frontend/src/lib/gu-family-icons.ts` and explicitly shares one icon across rank variants. The current resolver preserves the existing fallback chain, including deterministic fallback, for IDs without a custom/family match.

## Item and resource artwork coverage

- `backend/data/Item.csv`: 897 records: 826 Material, 62 Consumable, 8 Food, 1 Currency. No record currently has a non-empty `image_url`.
- The current item resolver maps the catalog to 50 shared icon keys, using 75 available image files in `frontend/public/assets/items/icons/`. No catalog item currently has a unique per-ID image URL. Most frequent resolver groups include `pelt` (97), `crystal` (88), `bone` (61), `vial` (58), and `dao-seal` (44).
- The approved coverage rule is unique art for distinctive resource/items and group fallbacks for the long tail. Exact records for the unique-art subset still need prioritization against the catalog.

## Mine and exploration markers

- `backend/data/ResourcePointTemplate.csv` contains 47 point types: 37 `MINERAL_VEIN`, 9 `BEAST_PASTURE`, and 1 `HERB_GARDEN`; 40 are `NORMAL` and 7 `FAMOUS`.
- Every template point type has an entry in `frontend/src/components/AperturePointsPanel.tsx`'s `POINT_GLYPHS`. The panel currently renders emoji per point type. Readiness is `state === "ACTIVE"` and total stock greater than zero; grade styles are separate UI badges, not point kinds.
- The 47 point IDs are: `HERB_VALLEY`, `BEAST_PASTURE`, `ORE_VEIN`, `SPIRIT_SPRING`, `GOLD_SAND_SHOAL`, `MOON_BOWL`, `YEAR_ESSENCE_POOL`, `DANG_HON_PEAK`, `JADE_SNAKE_CAVE`, `PHOENIX_PEAK`, `FROSTY_PEAK`, `CORPSE_RIDGE`, `QI_STONE_VEIN`, `REFINEMENT_IRON_MINE`, `FORMATION_STONE_VALLEY`, `RULE_STONE_FOREST`, `KILLING_IRON_MINE`, `WIND_STONE_CANYON`, `SPACE_CRYSTAL_RIFT`, `BLADE_STONE_QUARRY`, `SWORD_STONE_CLIFF`, `CREATION_STONE_FIELD`, `DESTRUCTION_IRON_MINE`, `BLOOD_CRYSTAL_POOL`, `LIGHTNING_CRYSTAL_CLIFF`, `THEFT_CRYSTAL_CAVE`, `DARK_CRYSTAL_ABYSS`, `CLOUD_CRYSTAL_TERRACE`, `LIGHT_CRYSTAL_CAVE`, `PHANTOM_CRYSTAL_VALLEY`, `STRENGTH_CRYSTAL_PEAK`, `TIME_SAND_DUNE`, `HEAVEN_SAND_SEA`, `DEATH_SAND_DUNE`, `STAR_SAND_DESERT`, `DREAM_PEARL_POOL`, `WISDOM_PEARL_POOL`, `LUCK_PEARL_POOL`, `INVESTIGATION_PEARL_POOL`, `SOUND_RESONANCE_CAVE`, `BONE_SPLINTER_FIELD`, `EMOTION_SILK_GROVE`, `INFORMATION_STONE_LIBRARY`, `PAINTING_INK_POOL`, `TRANSFORMATION_ESSENCE_POOL`, `LANGYA_REFINEMENT_PEAK`, `BLOOD_CRYSTAL_SEA`.
- `frontend/src/lib/worldResourceAssets.ts` has four image groups (`herb`, `crystal`, `food`, `gourd`). `ExplorationMap.tsx` uses them for `Resource` cells at 24×24 px; `PrimevalVein` uses a droplet glyph. Other world marker kinds currently include monster/boss sprites, a WildGu sparkle, portals, and inheritance altars.

## Integration pointers

- Gu: `backend/data/Gu.csv` → `frontend/src/lib/gu-icons.ts` → shared calls in Gu inventory, shops, trade, and other surfaces. CSV URLs exist but are not part of the resolver signature today.
- Items: `backend/data/Item.csv` → `frontend/src/lib/item-icons.ts` → shared calls in inventory, shops, trade, wiki, and crafting. The resolver already prioritizes valid `image_url` values.
- Exploration resources: item identity → `getWorldResourceGroup` / `WORLD_RESOURCE_ASSETS` → `ExplorationMap.tsx` map-cell icon.
- Aperture resource points: `ResourcePointTemplate.csv` → point `point_type` → `POINT_GLYPHS` in `AperturePointsPanel.tsx`; stock/state/grade remain UI overlays.

## Full item fallback group counts

All 897 catalog item IDs currently resolve through these shared icon keys; examples show representative IDs only. This table makes clear which records need a unique-art decision versus category fallback.

| Current icon key | Item count | Example IDs |
|---|---:|---|
| `pelt` | 97 | `dream_pearl_rank4`, `fire_crystal_rank1`, `phantom_crystal_rank2`, `poison_sand_rank2` |
| `crystal` | 88 | `creation_stone_rank1`, `creation_stone_rank2`, `creation_stone_rank4`, `creation_stone_rank7` |
| `bone` | 61 | `blade_stone_rank2`, `bone_marrow_rank2`, `bone_marrow_rank3`, `bone_marrow_rank4` |
| `vial` | 58 | `bone_marrow_rank9`, `food_essence_rank6`, `human_essence_rank1`, `human_essence_rank5` |
| `dao-seal` | 44 | `dao_material_blade`, `dao_material_blood`, `dao_material_bone`, `dao_material_cloud` |
| `blood` | 32 | `blood_crystal_rank2`, `blood_crystal_rank3`, `blood_crystal_rank4`, `blood_crystal_rank6` |
| `shadow` | 31 | `dark_crystal_rank2`, `dark_crystal_rank4`, `dark_crystal_rank5`, `dark_crystal_rank6` |
| `horn` | 29 | `blade_stone_rank3`, `destruction_beast_horn_rank1`, `destruction_beast_horn_rank2`, `destruction_beast_horn_rank3` |
| `ore` | 28 | `destruction_iron_rank1`, `destruction_iron_rank2`, `destruction_iron_rank3`, `destruction_iron_rank4` |
| `jade` | 25 | `food_essence_rank8`, `information_stone_rank8`, `poison_sand_rank8`, `qi_stone_rank8` |
| `pill` | 24 | `ITEM_HEALING_PELLET_SMALL`, `ITEM_HEALING_PELLET_MEDIUM`, `ITEM_HEALING_PELLET_LARGE`, `ITEM_LIFE_SAVING_PILL` |
| `sand` | 24 | `cloud_crystal_rank1`, `dark_crystal_rank3`, `death_sand_rank1`, `death_sand_rank3` |
| `pearl` | 23 | `creation_stone_rank9`, `dream_pearl_rank8`, `dream_pearl_rank9`, `earth_soil_rank9` |
| `blade` | 21 | `blade_stone_rank1`, `blade_stone_rank4`, `blade_stone_rank5`, `blade_stone_rank6` |
| `mushroom` | 20 | `phantom_crystal_rank1`, `qi_stone_rank1`, `refinement_iron_rank1`, `sound_resonance_rank1` |
| `trigram-token` | 18 | `QIAN_SKY_RANK6`, `KUN_EARTH_RANK6`, `LI_FIRE_RANK6`, `LY_FIRE_RANK6` |
| `animal-essence` | 16 | `poison_sand_rank1`, `rule_stone_rank1`, `soul_sand_rank1`, `space_crystal_rank1` |
| `light` | 16 | `light_crystal_rank2`, `light_crystal_rank3`, `light_crystal_rank5`, `light_crystal_rank6` |
| `fang` | 15 | `blood_crystal_rank1`, `bone_marrow_rank1`, `water_pearl_rank3`, `asura_blade_bone_rank2` |
| `clay` | 14 | `creation_stone_rank3`, `creation_stone_rank5`, `earth_soil_rank2`, `earth_soil_rank3` |
| `ice` | 14 | `ice_crystal_rank1`, `ice_crystal_rank2`, `ice_crystal_rank3`, `ice_crystal_rank4` |
| `insect` | 14 | `death_sand_rank2`, `dream_pearl_rank1`, `dream_pearl_rank2`, `dream_pearl_rank7` |
| `shard` | 14 | `strength_crystal_rank3`, `blood_bat_wing_rank3`, `heaven_tribulation_shard_rank1`, `heaven_tribulation_shard_rank2` |
| `cloud` | 13 | `cloud_crystal_rank2`, `cloud_crystal_rank4`, `cloud_crystal_rank5`, `cloud_crystal_rank6` |
| `pouch` | 13 | `food_essence_rank3`, `food_essence_rank4`, `food_essence_rank9`, `strength_crystal_rank5` |
| `shell` | 13 | `refinement_iron_rank3`, `asura_blade_bone_rank1`, `dark_python_hide_rank3`, `centipede_carapace_rank1` |
| `lightning` | 12 | `lightning_crystal_rank1`, `lightning_crystal_rank2`, `lightning_crystal_rank3`, `lightning_crystal_rank4` |
| `wind` | 12 | `wind_stone_rank3`, `wind_stone_rank7`, `wind_stone_rank8`, `wind_stone_rank9` |
| `wood` | 12 | `soul_sand_rank8`, `soul_sand_rank9`, `wood_crystal_rank6`, `wood_crystal_rank7` |
| `ember` | 11 | `fire_crystal_rank3`, `fire_crystal_rank4`, `fire_crystal_rank7`, `fire_crystal_rank8` |
| `water` | 11 | `water_pearl_rank5`, `water_pearl_rank6`, `water_pearl_rank8`, `water_pearl_rank9` |
| `root` | 10 | `ITEM_BLOOD_GINSENG`, `formation_herb_root_rank1`, `formation_herb_root_rank2`, `formation_herb_root_rank3` |
| `eye` | 9 | `hawk_investigation_eye_rank1`, `hawk_investigation_eye_rank2`, `hawk_investigation_eye_rank3`, `hawk_investigation_eye_rank4` |
| `hourglass` | 9 | `time_sand_rank5`, `time_sand_rank6`, `time_sand_rank7`, `time_toad_sand_rank5` |
| `silk` | 9 | `emotion_silk_rank1`, `emotion_silk_rank2`, `emotion_silk_rank3`, `emotion_silk_rank4` |
| `claw` | 8 | `iron_bear_paw_rank1`, `iron_bear_paw_rank3`, `iron_bear_paw_rank4`, `iron_bear_paw_rank5` |
| `formation` | 8 | `formation_stone_rank1`, `formation_stone_rank2`, `formation_stone_rank4`, `formation_stone_rank5` |
| `feather` | 5 | `cloud_crystal_rank3`, `fire_crystal_rank2`, `fire_crystal_rank5`, `asura_blade_bone_rank5` |
| `heart` | 3 | `blood_crystal_rank5`, `creation_stone_rank6`, `spirit_guardian_clay_rank7` |
| `chest` | 2 | `mine_chest`, `mine_chest_immortal` |
| `scale` | 2 | `fire_crystal_rank6`, `ice_cicada_wing_rank7` |
| `bagua-crystal` | 1 | `BAGUA_CRYSTALS` |
| `coin` | 1 | `arena_coin` |
| `egg` | 1 | `spring_egg` |
| `fruit` | 1 | `XUAN_RANK6` |
| `gear` | 1 | `enslavement_stone_rank3` |
| `nightfruit` | 1 | `XUAN_RANK1` |
| `potion` | 1 | `ITEM_IMMORTAL_SPRING_WATER` |
| `primordial-core` | 1 | `XUAN_RANK5` |
| `salve` | 1 | `ITEM_MEDICINAL_PASTE` |

## First rollout batch records

The exact first-batch IDs, CSV provenance, and per-record prompt seeds are recorded in [`first-rollout-batch-plan.json`](./first-rollout-batch-plan.json): 47 resource-point types, 44 unique production resource identities, and the 35 Gu IDs that miss custom, family, and direct icon coverage. All 126 are delivered. The batch validator confirms source CSV IDs, manifest paths, PNG masters, transparent WebP runtime files, dimensions, and prompt metadata. Grouped generation provenance is recorded in `art/game-icons/sources/manifest.json`.
