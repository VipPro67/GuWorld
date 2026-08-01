-- Fix bad rows in players.gu_inventory that fail the (gu_id, rank) -> gu_dictionary
-- FK check during migration to player_gu.
--
-- NOTE: The migration itself (db.go "migrate_player_gu_from_inventory") now
-- auto-normalizes rank 0 -> the dictionary's min rank, so Group A (starter rank-0
-- Gus, ~662 rows and growing while prod is live) is handled even without this
-- script. This script focuses on the rows the migration CANNOT repair:
--
--   B) remap Lotus family mislabels:
--        GU_HEAVENLY_ESSENCE_LOTUS     rank 5 -> GU_HEAVENLY_ESSENCE_LOTUS_5 (rank 5)  (9 Gus)
--        GU_HEAVENLY_ESSENCE_LOTUS     rank 9 -> GU_HEAVENLY_ESSENCE_LOTUS_I  (rank 9)  (4 Gus)
--        GU_HEAVENLY_ESSENCE_LOTUS_5   rank 6 -> GU_HEAVENLY_ESSENCE_LOTUS_I  (rank 6)  (14 Gus)
--   C) drop dead Gus not present in Gu.csv/gu_dictionary at all                        (17 Gus)
--        GU_SELF_RECOVERY (8), GU_SELF_RELIANCE (7), GU_HEALING_GRASS (2)
--
-- Total in group B+C: 27 + 17 = 44. Run BEFORE deploy (migration reads gu_inventory).
--
-- Run order: preview (SECTION 1), review output, then apply (SECTION 2).

-- ============================================================
-- SECTION 1: PREVIEW (read-only) — counts before & after
-- ============================================================

-- 1a) Rows that will change, grouped by transformation
SELECT
  CASE
    WHEN elem->>'gu_id' IN ('GU_SELF_RECOVERY','GU_SELF_RELIANCE','GU_HEALING_GRASS') THEN 'C) drop dead gu'
    WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' = '5' THEN 'B) lotus rank5 -> _5'
    WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' = '9' THEN 'B) lotus rank9 -> _I'
    WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS_5' AND elem->>'rank' = '6' THEN 'B) lotus_5 rank6 -> _I'
    WHEN elem->>'rank' = '0'
         AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' AND gd.rank = 1)
         THEN 'A) starter rank0 -> rank1'
  END AS transform,
  elem->>'gu_id' AS gu_id,
  elem->>'rank' AS old_rank,
  COUNT(*) AS cnt
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
WHERE (elem->>'gu_id' IN ('GU_SELF_RECOVERY','GU_SELF_RELIANCE','GU_HEALING_GRASS'))
   OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' IN ('5','9'))
   OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS_5' AND elem->>'rank' = '6')
   OR (elem->>'rank' = '0'
       AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' AND gd.rank = 1))
GROUP BY 1,2,3
ORDER BY 1, cnt DESC;

-- 1b) Sanity: total affected rows must equal 704
SELECT COUNT(*) AS total_affected
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
WHERE (elem->>'gu_id' IN ('GU_SELF_RECOVERY','GU_SELF_RELIANCE','GU_HEALING_GRASS'))
   OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' IN ('5','9'))
   OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS_5' AND elem->>'rank' = '6')
   OR (elem->>'rank' = '0'
       AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' AND gd.rank = 1));

-- 1c) Any OTHER not-in-dictionary rows we are NOT fixing? (must be 0 after apply, check before too)
SELECT elem->>'gu_id' AS gu_id, elem->>'rank' AS rank, COUNT(*) AS cnt
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
WHERE NOT EXISTS (
  SELECT 1 FROM gu_dictionary gd
  WHERE gd.id = elem->>'gu_id'
    AND gd.rank = COALESCE((elem->>'rank')::int, 1)
)
GROUP BY 1,2 ORDER BY 3 DESC;

-- ============================================================
-- SECTION 2: APPLY (transactional, idempotent)
-- ============================================================

-- BEGIN;
-- UPDATE players p
-- SET gu_inventory = (
--   SELECT COALESCE(jsonb_agg(newelem), '[]'::jsonb)
--   FROM (
--     SELECT
--       CASE
--         WHEN elem->>'rank' = '0'
--              AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' AND gd.rank = 1)
--           THEN elem || '{"rank": 1}'::jsonb
--         WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' = '5'
--           THEN elem || '{"gu_id": "GU_HEAVENLY_ESSENCE_LOTUS_5"}'::jsonb
--         WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' = '9'
--           THEN elem || '{"gu_id": "GU_HEAVENLY_ESSENCE_LOTUS_I"}'::jsonb
--         WHEN elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS_5' AND elem->>'rank' = '6'
--           THEN elem || '{"gu_id": "GU_HEAVENLY_ESSENCE_LOTUS_I"}'::jsonb
--         ELSE elem
--       END AS newelem
--     FROM jsonb_array_elements(p.gu_inventory) AS elem
--     WHERE elem->>'gu_id' NOT IN ('GU_SELF_RECOVERY','GU_SELF_RELIANCE','GU_HEALING_GRASS')
--   ) t
-- )
-- WHERE p.gu_inventory IS NOT NULL
--   AND EXISTS (
--     SELECT 1 FROM jsonb_array_elements(p.gu_inventory) elem
--     WHERE (elem->>'gu_id' IN ('GU_SELF_RECOVERY','GU_SELF_RELIANCE','GU_HEALING_GRASS'))
--        OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS' AND elem->>'rank' IN ('5','9'))
--        OR (elem->>'gu_id' = 'GU_HEAVENLY_ESSENCE_LOTUS_5' AND elem->>'rank' = '6')
--        OR (elem->>'rank' = '0'
--            AND EXISTS (SELECT 1 FROM gu_dictionary gd WHERE gd.id = elem->>'gu_id' AND gd.rank = 1))
--   );
-- COMMIT;

-- ============================================================
-- SECTION 3: VERIFY (after apply)
-- ============================================================

-- 3a) All not-in-dictionary rows must now be 0
-- SELECT elem->>'gu_id' AS gu_id, elem->>'rank' AS rank, COUNT(*) AS cnt
-- FROM players p
-- CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
-- WHERE NOT EXISTS (
--   SELECT 1 FROM gu_dictionary gd
--   WHERE gd.id = elem->>'gu_id'
--     AND gd.rank = COALESCE((elem->>'rank')::int, 1)
-- )
-- GROUP BY 1,2 ORDER BY 3 DESC;
-- -- expect: no rows

-- 3b) Total Gu rows after fix = 20.427 - 17 (dropped) = 20.410
-- SELECT COUNT(*) AS total_gus_after_fix
-- FROM players p
-- CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem;
-- -- expect: 20410
