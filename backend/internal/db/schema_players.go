package db

import "fmt"

const playerGoogleIDIndexSchema = `
	CREATE INDEX IF NOT EXISTS idx_players_google_id ON players (google_id)
	WHERE google_id IS NOT NULL;`

// Player/account schema. players is the hub most gameplay tables reference;
// it is created after parties so the players.party_id FK can be declared inline.
// The circular parties.leader_id -> players FK is added via ALTER below.

func schemaPlayers() error {
	if err := execSchema("players", `
	CREATE TABLE IF NOT EXISTS players (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL DEFAULT '',
		email VARCHAR(255) NOT NULL DEFAULT '' UNIQUE,
		google_id VARCHAR(255),
		ref_code VARCHAR(50) NOT NULL DEFAULT '' UNIQUE,
		rank INTEGER DEFAULT 0,
		sub_rank INTEGER DEFAULT 1,
		aptitude_type VARCHAR(10) NOT NULL,
		aptitude_percent INTEGER NOT NULL,
		aptitude_grade INTEGER DEFAULT 4,
		current_mana DOUBLE PRECISION NOT NULL,
		max_mana DOUBLE PRECISION NOT NULL,
		primeval_stones NUMERIC(30,3) NOT NULL DEFAULT 0,
		immortal_stones NUMERIC(30,3) NOT NULL DEFAULT 0,
		hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		base_hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		current_region VARCHAR(50) NOT NULL DEFAULT 'tan_thu_coc_nam_cuong' REFERENCES regions(id),
		map_instance_id UUID,
		x INTEGER NOT NULL DEFAULT 50,
		y INTEGER NOT NULL DEFAULT 50,
		world_zone_id VARCHAR(80) NOT NULL DEFAULT '',
		world_instance_id VARCHAR(80) NOT NULL DEFAULT '',
		world_x DOUBLE PRECISION NOT NULL DEFAULT 0,
		world_y DOUBLE PRECISION NOT NULL DEFAULT 0,
		world_safe_x DOUBLE PRECISION NOT NULL DEFAULT 0,
		world_safe_y DOUBLE PRECISION NOT NULL DEFAULT 0,
		world_location_revision BIGINT NOT NULL DEFAULT 0,
		race VARCHAR(50) NOT NULL DEFAULT 'human' REFERENCES race_dictionary(key),
		gender VARCHAR(20) NOT NULL DEFAULT '',
		constitution VARCHAR(100) NOT NULL DEFAULT '' REFERENCES constitution_dictionary(key),
		path_masteries JSONB NOT NULL DEFAULT '{}',
		body_mods JSONB NOT NULL DEFAULT '[]',
		visited_subzones JSONB NOT NULL DEFAULT '[]',
		equipped_gus JSONB NOT NULL DEFAULT '[]',
		cleared_cells JSONB NOT NULL DEFAULT '{}',
		materials_inventory JSONB NOT NULL DEFAULT '{}',
		statuses JSONB NOT NULL DEFAULT '[]',
		earned_achievements JSONB NOT NULL DEFAULT '[]',
		growth_data JSONB NOT NULL DEFAULT '{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}',
		combat_settings JSONB NOT NULL DEFAULT '{}',
		realtime_skill_loadout JSONB NOT NULL DEFAULT '[]'::jsonb,
		realtime_skill_loadout_configured BOOLEAN NOT NULL DEFAULT false,
		realtime_hotbar JSONB NOT NULL DEFAULT '[]'::jsonb,
		immortal_state JSONB NOT NULL DEFAULT '{}',
		last_mana_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_hunger_tick_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		aperture_stress DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		last_cultivate_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		wall_progress DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		human_qi DOUBLE PRECISION NOT NULL DEFAULT 0.0,
		next_tribulation_time TIMESTAMPTZ,
		lifespan DOUBLE PRECISION NOT NULL DEFAULT 100.0,
		time_speed_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		party_id UUID REFERENCES parties(id) ON DELETE SET NULL,
		arena_rank INTEGER,
		explore_count INTEGER NOT NULL DEFAULT 0,
		last_explore_reset TIMESTAMP NOT NULL DEFAULT NOW(),
		soul_hp INTEGER NOT NULL DEFAULT 10,
		max_soul INTEGER NOT NULL DEFAULT 10,
		soul_tier VARCHAR(30) NOT NULL DEFAULT 'NHAN_HON',
		needs_soul_refine BOOLEAN NOT NULL DEFAULT FALSE,
		selected_title VARCHAR(100) NOT NULL DEFAULT '',
		selected_frame VARCHAR(50) NOT NULL DEFAULT '',
		bio TEXT DEFAULT 'Hãy viết gì đó về bản thân...',
		fake_rank INTEGER DEFAULT 1,
		fake_sub_rank INTEGER DEFAULT 1,
		fake_vital_gu VARCHAR(50) DEFAULT '',
		is_disguised BOOLEAN DEFAULT true,
		is_admin BOOLEAN DEFAULT false,
		admin_permissions TEXT[] NOT NULL DEFAULT ARRAY['*']::TEXT[],
		needs_onboarding BOOLEAN NOT NULL DEFAULT false,
		avatar_url TEXT DEFAULT '',
		last_activity TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		current_thought DOUBLE PRECISION NOT NULL DEFAULT 100.0,
		max_thought DOUBLE PRECISION NOT NULL DEFAULT 100.0,
		thought_regen_rate DOUBLE PRECISION NOT NULL DEFAULT 1.0,
		register_fingerprint VARCHAR(255) NOT NULL DEFAULT '',
		register_ip VARCHAR(50) NOT NULL DEFAULT '',
		last_login_fingerprint VARCHAR(255) NOT NULL DEFAULT '',
		last_login_ip VARCHAR(50) NOT NULL DEFAULT '',
		last_login_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
		active_session_id VARCHAR(64) NOT NULL DEFAULT '',
		version INT NOT NULL DEFAULT 0,
		state_revision BIGINT NOT NULL DEFAULT 0,
		death_arena_title VARCHAR(100) DEFAULT '',
		martial_title VARCHAR(100) DEFAULT '',
		active_concealment JSONB NOT NULL DEFAULT '{}'::jsonb,
		last_sever_clues_at TIMESTAMPTZ DEFAULT NULL,
		last_resonance_scan_at TIMESTAMPTZ DEFAULT NULL,
		is_banned BOOLEAN NOT NULL DEFAULT false,
		ban_reason TEXT NOT NULL DEFAULT '',
		banned_until TIMESTAMPTZ,
		banned_by UUID REFERENCES players(id) ON DELETE SET NULL,
		banned_at TIMESTAMPTZ,
		is_chat_muted BOOLEAN NOT NULL DEFAULT false,
		chat_muted_until TIMESTAMPTZ
	);`); err != nil {
		return err
	}
	if err := execSchema("players_username_case_insensitive_unique", `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_players_username_lower_unique ON players (LOWER(username));`); err != nil {
		return fmt.Errorf("cannot enforce case-insensitive username uniqueness; resolve existing usernames that differ only by letter case first: %w", err)
	}
	if err := execSchema("players_realtime_skill_loadout", `
	ALTER TABLE players ADD COLUMN IF NOT EXISTS realtime_skill_loadout JSONB NOT NULL DEFAULT '[]'::jsonb;`); err != nil {
		return err
	}
	if err := execSchema("players_realtime_skill_loadout_configured", `
	ALTER TABLE players ADD COLUMN IF NOT EXISTS realtime_skill_loadout_configured BOOLEAN NOT NULL DEFAULT false;`); err != nil {
		return err
	}
	if err := execSchema("players_realtime_hotbar", `
	ALTER TABLE players ADD COLUMN IF NOT EXISTS realtime_hotbar JSONB NOT NULL DEFAULT '[]'::jsonb;`); err != nil {
		return err
	}
	if err := execSchema("players_map_instance_idx", `CREATE INDEX IF NOT EXISTS idx_players_map_instance_location ON players (map_instance_id, current_region, x, y);`); err != nil {
		return err
	}
	if err := execSchema("players_world_location_idx", `CREATE INDEX IF NOT EXISTS idx_players_world_location ON players (world_zone_id, world_instance_id) WHERE world_zone_id <> '';`); err != nil {
		return err
	}

	// Player-row writes serialize on the row itself, so this counter is ordered
	// by commit visibility for those writes.
	if err := execSchema("players_state_revision_trigger", `
	CREATE OR REPLACE FUNCTION bump_player_row_state_revision()
	RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN
		NEW.state_revision := OLD.state_revision + 1;
		RETURN NEW;
	END;
	$$;
	DROP TRIGGER IF EXISTS trg_players_state_revision ON players;
	CREATE TRIGGER trg_players_state_revision
		BEFORE UPDATE ON players
		FOR EACH ROW EXECUTE FUNCTION bump_player_row_state_revision();`); err != nil {
		return err
	}
	if err := execSchema("players_world_location_legacy_invalidation", `
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
		FOR EACH ROW EXECUTE FUNCTION invalidate_world_location_after_legacy_move();`); err != nil {
		return err
	}

	// Index to speed up the 30s world-index-sync scan on last_mana_update.
	if err := execSchema("players_last_mana_update_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_last_mana_update ON players (last_mana_update);
	`); err != nil {
		return err
	}
	if err := execSchema("players_last_activity_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_last_activity ON players (last_activity);
	`); err != nil {
		return err
	}

	// Complete the players <-> parties cycle: parties.leader_id was created
	// without a FK (see schemaParties) because players didn't exist yet.
	if err := execSchema("parties_leader_fk", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'parties_leader_id_fkey') THEN
			ALTER TABLE parties ADD CONSTRAINT parties_leader_id_fkey
				FOREIGN KEY (leader_id) REFERENCES players(id) ON DELETE CASCADE;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("players_region_xy_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_region_xy ON players (current_region, x, y);`); err != nil {
		return err
	}
	if err := execSchema("players_map_instance_pos_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_map_instance_pos ON players (map_instance_id, current_region, x, y);`); err != nil {
		return err
	}
	if err := execSchema("players_party_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_party_id ON players (party_id);`); err != nil {
		return err
	}
	if err := execSchema("players_created_at_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_created_at_id ON players (created_at DESC, id DESC);`); err != nil {
		return err
	}
	if err := execSchema("players_username_trgm_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_username_trgm ON players USING gin (public.immutable_unaccent(username) gin_trgm_ops);`); err != nil {
		return err
	}
	if err := execSchema("players_statuses_gin_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_statuses ON players USING gin (statuses);`); err != nil {
		return err
	}
	if err := execSchema("players_equipped_gus_gin_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_equipped_gus ON players USING gin (equipped_gus);`); err != nil {
		return err
	}
	if err := execSchema("players_path_masteries_gin_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_path_masteries ON players USING gin (path_masteries);`); err != nil {
		return err
	}
	if err := execSchema("players_body_mods_gin_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_body_mods ON players USING gin (body_mods);`); err != nil {
		return err
	}

	// Login hot path: WHERE LOWER(email) = $1 cannot use the raw UNIQUE(email)
	// btree, so every login seq-scans players without this expression index.
	if err := execSchema("players_email_lower_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_email_lower ON players (LOWER(email));`); err != nil {
		return err
	}

	// Google OAuth resolves an account by google_id before falling back to
	// email. Keep null-only password accounts out of the index while avoiding a
	// full players scan on every callback.
	if err := execSchema("players_google_id_idx", playerGoogleIDIndexSchema); err != nil {
		return err
	}

	// Login fingerprint limit check filters on (last_login_fingerprint,
	// last_login_time) twice per login. No index existed -> seq-scan per login.
	if err := execSchema("players_login_fp_time_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_login_fp_time ON players (last_login_fingerprint, last_login_time);`); err != nil {
		return err
	}

	// Admin ban list/filter only ever looks at the small set of banned rows.
	if err := execSchema("players_banned_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_banned ON players (banned_until) WHERE is_banned;`); err != nil {
		return err
	}

	return nil
}

// installPlayerCollectionRevisionTriggers attaches an ownership trigger to
// every table with a direct foreign key to players. Discovering the foreign
// keys keeps new player-owned collections covered without duplicating their
// schema here. Collection revisions deliberately live outside players: the
// old triggers updated the hot player row and inverted the lock order of APIs
// that mutate child rows before locking players, causing widespread 55P03
// lock timeouts.
func installPlayerCollectionRevisionTriggers() error {
	return execSchema("player_collection_state_revision_triggers", playerCollectionRevisionSchema)
}

const playerCollectionRevisionSchema = `
	CREATE TABLE IF NOT EXISTS player_collection_revisions (
		player_id UUID NOT NULL,
		shard SMALLINT NOT NULL,
		state_revision BIGINT NOT NULL DEFAULT 0,
		PRIMARY KEY (player_id, shard)
	);

	CREATE OR REPLACE FUNCTION delete_player_collection_revisions()
	RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN
		DELETE FROM player_collection_revisions WHERE player_id = OLD.id;
		RETURN OLD;
	END;
	$$;
	DROP TRIGGER IF EXISTS trg_delete_player_collection_revisions ON players;
	CREATE TRIGGER trg_delete_player_collection_revisions
		AFTER DELETE ON players
		FOR EACH ROW EXECUTE FUNCTION delete_player_collection_revisions();

	CREATE OR REPLACE FUNCTION bump_player_collection_state_revision()
	RETURNS trigger LANGUAGE plpgsql AS $$
	DECLARE
		owner_column text := TG_ARGV[0];
		old_owner uuid;
		new_owner uuid;
	BEGIN
		IF TG_OP <> 'INSERT' THEN
			old_owner := NULLIF(to_jsonb(OLD)->>owner_column, '')::uuid;
		END IF;
		IF TG_OP <> 'DELETE' THEN
			new_owner := NULLIF(to_jsonb(NEW)->>owner_column, '')::uuid;
		END IF;

		IF old_owner IS NOT NULL AND EXISTS (SELECT 1 FROM players WHERE id = old_owner) THEN
			INSERT INTO player_collection_revisions (player_id, shard, state_revision)
			VALUES (old_owner, mod(txid_current(), 64)::smallint, 1)
			ON CONFLICT (player_id, shard) DO UPDATE
			SET state_revision = player_collection_revisions.state_revision + 1;
		END IF;
		IF new_owner IS NOT NULL AND new_owner IS DISTINCT FROM old_owner
			AND EXISTS (SELECT 1 FROM players WHERE id = new_owner) THEN
			INSERT INTO player_collection_revisions (player_id, shard, state_revision)
			VALUES (new_owner, mod(txid_current(), 64)::smallint, 1)
			ON CONFLICT (player_id, shard) DO UPDATE
			SET state_revision = player_collection_revisions.state_revision + 1;
		END IF;
		IF TG_OP = 'DELETE' THEN
			RETURN OLD;
		END IF;
		RETURN NEW;
	END;
	$$;

	DO $$
	DECLARE fk record;
	BEGIN
		FOR fk IN
			SELECT c.conrelid::regclass AS table_name, c.conrelid AS table_oid, a.attname AS column_name,
			       'trg_state_revision_' || c.conrelid::oid || '_' || a.attnum AS trigger_name
			FROM pg_constraint c
			JOIN unnest(c.conkey) WITH ORDINALITY AS k(attnum, ord) ON true
			JOIN unnest(c.confkey) WITH ORDINALITY AS rk(attnum, ord) USING (ord)
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
			JOIN pg_attribute ra ON ra.attrelid = c.confrelid AND ra.attnum = rk.attnum
			WHERE c.contype = 'f'
			  AND c.confrelid = 'players'::regclass
			  AND ra.attname = 'id'
			  AND c.conrelid <> 'players'::regclass
			  AND c.conrelid <> 'player_collection_revisions'::regclass
		LOOP
			-- Only create missing triggers. DROP+CREATE on every boot takes an
			-- ACCESS EXCLUSIVE lock on ~60 hot player tables, blocking traffic
			-- during rolling restarts. The trigger name encodes the FK column,
			-- so an existing name always has the matching owner argument.
			IF NOT EXISTS (
				SELECT 1 FROM pg_trigger t
				WHERE t.tgrelid = fk.table_oid AND t.tgname = fk.trigger_name
			) THEN
				EXECUTE format(
					'CREATE TRIGGER %I AFTER INSERT OR UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION bump_player_collection_state_revision(%L)',
					fk.trigger_name, fk.table_name, fk.column_name
				);
			END IF;
		END LOOP;
	END $$;`

func schemaPlayerExtras() error {
	if err := execSchema("referrals", `
	CREATE TABLE IF NOT EXISTS referrals (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		referrer_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		referee_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
		referee_ip VARCHAR(50) NOT NULL DEFAULT '',
		referee_device TEXT NOT NULL DEFAULT '',
		is_suspicious BOOLEAN NOT NULL DEFAULT false,
		has_deposited BOOLEAN NOT NULL DEFAULT false,
		deposit_amount_stones BIGINT NOT NULL DEFAULT 0,
		milestone1_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone2_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone3_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone4_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone5_rewarded BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("referrals_referrer_idx", `
	CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals (referrer_id);`); err != nil {
		return err
	}

	if err := execSchema("leaderboard_settings", `
	CREATE TABLE IF NOT EXISTS leaderboard_settings (
		player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		show_on_leaderboard BOOLEAN NOT NULL DEFAULT TRUE
	);`); err != nil {
		return err
	}

	if err := execSchema("wallets", `
	CREATE TABLE IF NOT EXISTS wallets (
		user_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
		immortal_jade BIGINT NOT NULL DEFAULT 0,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	if err := execSchema("email_otps", `
	CREATE TABLE IF NOT EXISTS email_otps (
		email VARCHAR(255) PRIMARY KEY,
		otp VARCHAR(10) NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`); err != nil {
		return err
	}

	if err := execSchema("player_achievements", `
	CREATE TABLE IF NOT EXISTS player_achievements (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		achievement_id VARCHAR(50) NOT NULL,
		selected_title VARCHAR(100) NOT NULL DEFAULT '',
		selected_frame VARCHAR(50) NOT NULL DEFAULT '',
		unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, achievement_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("backfill_player_achievements", `
	INSERT INTO player_achievements(player_id, achievement_id)
	SELECT p.id, achievement->>'achievement_id'
	FROM players p
	CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(p.earned_achievements)='array' THEN p.earned_achievements ELSE '[]'::jsonb END) AS achievement
	WHERE COALESCE(achievement->>'achievement_id','') <> ''
	ON CONFLICT(player_id, achievement_id) DO NOTHING;`); err != nil {
		return err
	}

	return nil
}
