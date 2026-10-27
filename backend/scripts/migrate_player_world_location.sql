-- Run once during a maintenance window after stopping legacy movement writers
-- and draining realtime sessions. This is intentionally not run by InitSchema.
BEGIN;

ALTER TABLE players
    ADD COLUMN IF NOT EXISTS world_zone_id VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS world_instance_id VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS world_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS world_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS world_safe_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS world_safe_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS world_location_revision BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_players_world_location
    ON players (world_zone_id, world_instance_id) WHERE world_zone_id <> '';

-- Preserve the last committed realtime point when available. Join validates
-- map/content versions, map bounds and collision before restoring the point.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'player_world_state_v2'
          AND column_name = 'zone_id'
    ) THEN
        EXECUTE $backfill$
            UPDATE players p
            SET world_zone_id = s.zone_id,
                world_instance_id = s.instance_id,
                world_x = s.x,
                world_y = s.y,
                world_safe_x = s.safe_x,
                world_safe_y = s.safe_y,
                world_location_revision = GREATEST(p.world_location_revision, 1)
            FROM player_world_state_v2 s
            WHERE s.player_id = p.id
              AND p.world_zone_id = ''
        $backfill$;
    END IF;
END;
$$;

DO $$
BEGIN
    IF to_regclass(current_schema() || '.player_world_location') IS NOT NULL THEN
        EXECUTE $backfill$
            UPDATE players p
            SET current_region = l.region_id,
                world_zone_id = l.zone_id,
                world_instance_id = l.instance_id,
                world_x = l.x,
                world_y = l.y,
                world_safe_x = l.safe_x,
                world_safe_y = l.safe_y,
                world_location_revision = GREATEST(p.world_location_revision, l.revision, 1)
            FROM player_world_location l
            WHERE l.player_id = p.id
              AND p.world_zone_id = ''
        $backfill$;
        EXECUTE $preserve$
            INSERT INTO player_world_state_v2
                (player_id, hp, shield, map_version, content_version, updated_at)
            SELECT l.player_id, l.hp, l.shield, l.map_version, l.content_version, l.updated_at
            FROM player_world_location l
            ON CONFLICT (player_id) DO NOTHING
        $preserve$;
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION invalidate_world_location_after_legacy_move()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.current_region IS DISTINCT FROM OLD.current_region OR
        NEW.map_instance_id IS DISTINCT FROM OLD.map_instance_id OR
        NEW.x IS DISTINCT FROM OLD.x OR NEW.y IS DISTINCT FROM OLD.y) AND
       NEW.world_zone_id IS NOT DISTINCT FROM OLD.world_zone_id AND
       NEW.world_instance_id IS NOT DISTINCT FROM OLD.world_instance_id AND
       NEW.world_x IS NOT DISTINCT FROM OLD.world_x AND
       NEW.world_y IS NOT DISTINCT FROM OLD.world_y AND
       NEW.world_location_revision IS NOT DISTINCT FROM OLD.world_location_revision THEN
        NEW.world_zone_id := '';
        NEW.world_instance_id := '';
        NEW.world_location_revision := OLD.world_location_revision + 1;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg_players_world_location_legacy_invalidation ON players;
CREATE TRIGGER trg_players_world_location_legacy_invalidation
    BEFORE UPDATE OF current_region, map_instance_id, x, y ON players
    FOR EACH ROW EXECUTE FUNCTION invalidate_world_location_after_legacy_move();

DROP INDEX IF EXISTS idx_player_world_state_v2_zone;
ALTER TABLE player_world_state_v2
    DROP COLUMN IF EXISTS zone_id,
    DROP COLUMN IF EXISTS instance_id,
    DROP COLUMN IF EXISTS x,
    DROP COLUMN IF EXISTS y,
    DROP COLUMN IF EXISTS safe_x,
    DROP COLUMN IF EXISTS safe_y;

-- This table was an unused preparation for the previous migration proposal.
-- The player row is the selected location owner.
DROP TABLE IF EXISTS player_world_location;

COMMIT;
