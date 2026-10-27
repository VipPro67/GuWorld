-- scripts/migrate_database_coordinates.sql
-- Production/Test DB migration script to convert all coordinates
-- to the local MMO coordinate system (1000x1000, 0..1000).

BEGIN;

-- 1. Ensure columns atlas_x, atlas_y exist in regions table
ALTER TABLE regions ADD COLUMN IF NOT EXISTS atlas_x INT NOT NULL DEFAULT 0;
ALTER TABLE regions ADD COLUMN IF NOT EXISTS atlas_y INT NOT NULL DEFAULT 0;

-- 2. Update players table (x, y)
WITH actual_players AS (
  SELECT p.id, r_actual.id as actual_region,
    GREATEST(10, LEAST(990, ROUND((p.x - r_actual.min_x) * 1000.0 / NULLIF(r_actual.width, 0))::int)) as new_x,
    GREATEST(10, LEAST(990, ROUND((p.y - r_actual.min_y) * 1000.0 / NULLIF(r_actual.height, 0))::int)) as new_y
  FROM players p
  JOIN regions r_actual ON p.x >= r_actual.min_x AND p.x < r_actual.max_x 
                       AND p.y >= r_actual.min_y AND p.y < r_actual.max_y 
                       AND r_actual.zone_type IN ('KHU_VUC', 'SAFE_ZONE')
  WHERE p.x > 1000 OR p.y > 1000
)
UPDATE players p
SET 
  current_region = ap.actual_region,
  x = ap.new_x,
  y = ap.new_y
FROM actual_players ap
WHERE p.id = ap.id;

-- 3. Update primeval_veins (Mỏ nguyên thạch / nguyên tuyền)
ALTER TABLE primeval_veins DROP CONSTRAINT IF EXISTS primeval_veins_region_id_x_y_key;

DELETE FROM primeval_veins WHERE expires_at <= NOW();

WITH raw_veins AS (
  SELECT pv.id, r_actual.id as actual_region,
    GREATEST(10, LEAST(990, ROUND((pv.x - r_actual.min_x) * 1000.0 / NULLIF(r_actual.width, 0))::int)) as base_x,
    GREATEST(10, LEAST(990, ROUND((pv.y - r_actual.min_y) * 1000.0 / NULLIF(r_actual.height, 0))::int)) as base_y
  FROM primeval_veins pv
  JOIN regions r_actual ON pv.x >= r_actual.min_x AND pv.x < r_actual.max_x 
                       AND pv.y >= r_actual.min_y AND pv.y < r_actual.max_y 
                       AND r_actual.zone_type IN ('KHU_VUC', 'SAFE_ZONE')
  WHERE pv.x > 1000 OR pv.y > 1000
),
disambiguated_veins AS (
  SELECT id, actual_region,
    base_x + (ROW_NUMBER() OVER (PARTITION BY actual_region, base_x, base_y ORDER BY id) - 1)::int as final_x,
    base_y as final_y
  FROM raw_veins
)
UPDATE primeval_veins pv
SET 
  region_id = dv.actual_region,
  x = dv.final_x,
  y = dv.final_y
FROM disambiguated_veins dv
WHERE pv.id = dv.id;

CREATE INDEX IF NOT EXISTS idx_primeval_veins_region ON primeval_veins (region_id, x, y);

-- 4. Update teleport_arrays (Trận pháp truyền tống của người chơi / tông môn)
ALTER TABLE teleport_arrays DROP CONSTRAINT IF EXISTS uq_teleport_array_cell;

WITH raw_arrays AS (
  SELECT ta.id, r_actual.id as actual_region,
    GREATEST(10, LEAST(990, ROUND((ta.x - r_actual.min_x) * 1000.0 / NULLIF(r_actual.width, 0))::int)) as base_x,
    GREATEST(10, LEAST(990, ROUND((ta.y - r_actual.min_y) * 1000.0 / NULLIF(r_actual.height, 0))::int)) as base_y
  FROM teleport_arrays ta
  JOIN regions r_actual ON ta.x >= r_actual.min_x AND ta.x < r_actual.max_x 
                       AND ta.y >= r_actual.min_y AND ta.y < r_actual.max_y 
                       AND r_actual.zone_type IN ('KHU_VUC', 'SAFE_ZONE')
  WHERE ta.x > 1000 OR ta.y > 1000
),
disambiguated_arrays AS (
  SELECT id, actual_region,
    base_x + (ROW_NUMBER() OVER (PARTITION BY actual_region, base_x, base_y ORDER BY id) - 1)::int as final_x,
    base_y as final_y
  FROM raw_arrays
)
UPDATE teleport_arrays ta
SET 
  region_id = da.actual_region,
  x = da.final_x,
  y = da.final_y
FROM disambiguated_arrays da
WHERE ta.id = da.id;

ALTER TABLE teleport_arrays ADD CONSTRAINT uq_teleport_array_cell UNIQUE (region_id, x, y);

-- 5. Update sects (Tông môn)
WITH actual_sects AS (
  SELECT s.id, r_actual.id as actual_region,
    GREATEST(10, LEAST(990, ROUND((s.map_x - r_actual.min_x) * 1000.0 / NULLIF(r_actual.width, 0))::int)) as new_x,
    GREATEST(10, LEAST(990, ROUND((s.map_y - r_actual.min_y) * 1000.0 / NULLIF(r_actual.height, 0))::int)) as new_y
  FROM sects s
  JOIN regions r_actual ON s.map_x >= r_actual.min_x AND s.map_x < r_actual.max_x 
                       AND s.map_y >= r_actual.min_y AND s.map_y < r_actual.max_y 
                       AND r_actual.zone_type IN ('KHU_VUC', 'SAFE_ZONE')
  WHERE s.map_x > 1000 OR s.map_y > 1000
)
UPDATE sects s
SET 
  region = ase.actual_region,
  map_x = ase.new_x,
  map_y = ase.new_y
FROM actual_sects ase
WHERE s.id = ase.id;

-- 6. Update shared_spatial_instance_members (Điểm quay về từ bí cảnh)
UPDATE shared_spatial_instance_members m
SET 
  return_x = GREATEST(10, LEAST(990, ROUND((m.return_x - r.min_x) * 1000.0 / NULLIF(r.width, 0))::int)),
  return_y = GREATEST(10, LEAST(990, ROUND((m.return_y - r.min_y) * 1000.0 / NULLIF(r.height, 0))::int))
FROM regions r
WHERE m.return_region_id = r.id
  AND (m.return_x > 1000 OR m.return_y > 1000)
  AND r.width > 0;

-- 7. Cập nhật bảng regions sang hệ tọa độ local 1000x1000
UPDATE regions
SET 
  atlas_x = min_x / 2,
  atlas_y = min_y / 2,
  min_x = 0,
  min_y = 0,
  max_x = 1000,
  max_y = 1000,
  width = 1000,
  height = 1000,
  entry_x = 500,
  entry_y = 500,
  safe_zone_x = 500,
  safe_zone_y = 500
WHERE zone_type IN ('KHU_VUC', 'SAFE_ZONE') AND width > 1000;

COMMIT;
