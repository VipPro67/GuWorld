-- ============================================================
-- PRE-FLIGHT: Migration players.gu_inventory -> player_gu
-- ============================================================
-- Chạy TRƯỚC khi deploy bản code mới (READ-ONLY, an toàn).
--
-- Cách chạy:
--   psql "postgres://gu_admin:gu_pass_123@localhost:5432/gu_db?sslmode=disable" \
--       -f backend/scripts/migrate_gu_inventory_preflight.sql
--
-- Kết quả:
--   (A) Số row sẽ bị BỎ QUA (mất dữ liệu âm thầm). Nếu > 0, xem chi tiết
--       bảng "problematic_rows_detail" bên dưới để quyết định.
--   (B) Rows gây lỗi cast -> PHẢI = 0 row, nếu không server KHÔNG start được.
--   (C) Snapshot tổng để so sánh sau khi migrate.
-- ============================================================

-- (A) ĐẾM ROWS SẼ BỊ BỎ QUA (mất dữ liệu âm thầm)
SELECT
  COUNT(*) AS total_gu_rows,
  COUNT(*) FILTER (WHERE elem->>'instance_id' IS NULL OR elem->>'instance_id' = '') AS null_instance_id,
  COUNT(*) FILTER (WHERE elem->>'gu_id' IS NULL OR elem->>'gu_id' = '') AS null_gu_id,
  COUNT(*) FILTER (WHERE NOT EXISTS (
    SELECT 1 FROM gu_dictionary gd
    WHERE gd.id = elem->>'gu_id'
      AND gd.rank = COALESCE(
        CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int END,
        1
      )
  )) AS not_in_dictionary,
  COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM player_gu pg WHERE pg.id = elem->>'instance_id')) AS duplicate_in_player_gu
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem;

-- (A.1) CHI TIẾT ROWS SẼ BỊ BỎ QUA (xem nếu (A) > 0)
SELECT p.id AS player_id, p.username,
       elem->>'instance_id' AS instance_id,
       elem->>'gu_id' AS gu_id,
       elem->>'rank' AS rank,
       CASE
         WHEN elem->>'instance_id' IS NULL OR elem->>'instance_id' = '' THEN 'NULL_EMPTY_INSTANCE_ID'
         WHEN elem->>'gu_id' IS NULL OR elem->>'gu_id' = '' THEN 'NULL_EMPTY_GU_ID'
         WHEN NOT EXISTS (
           SELECT 1 FROM gu_dictionary gd
           WHERE gd.id = elem->>'gu_id'
             AND gd.rank = COALESCE(
               CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int END,
               1
             )
         ) THEN 'NOT_IN_GU_DICTIONARY'
         WHEN EXISTS (SELECT 1 FROM player_gu pg WHERE pg.id = elem->>'instance_id') THEN 'DUPLICATE_IN_PLAYER_GU'
         ELSE 'OK'
       END AS skip_reason
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
WHERE elem->>'instance_id' IS NULL OR elem->>'instance_id' = ''
   OR elem->>'gu_id' IS NULL OR elem->>'gu_id' = ''
   OR NOT EXISTS (
        SELECT 1 FROM gu_dictionary gd
        WHERE gd.id = elem->>'gu_id'
          AND gd.rank = COALESCE(
            CASE WHEN elem->>'rank' ~ '^[0-9]+$' THEN (elem->>'rank')::int END,
            1
          )
      )
   OR EXISTS (SELECT 1 FROM player_gu pg WHERE pg.id = elem->>'instance_id');

-- (B) LIỆT KÊ ROWS GÂY LỖI CAST -> PHẢI = 0 ROWS (nếu có, server không start được)
SELECT p.id AS player_id, p.username,
       elem->>'instance_id' AS instance_id,
       elem->>'rank' AS rank,
       elem->>'hunger' AS hunger,
       elem->>'equipped' AS equipped,
       elem->>'is_vital' AS is_vital,
       elem->>'integration_time' AS integration_time,
       elem->>'growth_hours' AS growth_hours,
       elem->>'equipped_at' AS equipped_at
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem
WHERE (elem->>'rank' IS NOT NULL AND elem->>'rank' <> '' AND elem->>'rank' !~ '^[0-9]+$')
   OR (elem->>'hunger' IS NOT NULL AND elem->>'hunger' <> '' AND elem->>'hunger' !~ '^[0-9]+(\.[0-9]+)?$')
   OR (elem->>'integration_time' IS NOT NULL AND elem->>'integration_time' <> '' AND elem->>'integration_time' !~ '^[0-9]+(\.[0-9]+)?$')
   OR (elem->>'growth_hours' IS NOT NULL AND elem->>'growth_hours' <> '' AND elem->>'growth_hours' !~ '^[0-9]+(\.[0-9]+)?$')
   OR (elem->>'equipped' IS NOT NULL AND lower(elem->>'equipped') NOT IN ('true','false','t','f','yes','no','on','off','1','0'))
   OR (elem->>'is_vital' IS NOT NULL AND lower(elem->>'is_vital') NOT IN ('true','false','t','f','yes','no','on','off','1','0'))
   OR (elem->>'equipped_at' IS NOT NULL AND elem->>'equipped_at' <> '' AND elem->>'equipped_at' !~ '^\d{4}-\d{2}-\d{2}');

-- (C) SNAPSHOT TỔNG (so sánh sau khi migrate)
SELECT COUNT(*) AS player_count,
       COUNT(elem) AS gu_rows,
       COUNT(DISTINCT p.id) AS players_with_gus
FROM players p
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.gu_inventory,'[]'::jsonb)) AS elem;
