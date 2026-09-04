package db

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
		rank INTEGER DEFAULT 1,
		sub_rank INTEGER DEFAULT 1,
		aptitude_type VARCHAR(10) NOT NULL,
		aptitude_percent INTEGER NOT NULL,
		aptitude_grade INTEGER DEFAULT 4,
		current_mana DOUBLE PRECISION NOT NULL,
		max_mana DOUBLE PRECISION NOT NULL,
		primeval_stones BIGINT DEFAULT 0,
		immortal_stones BIGINT DEFAULT 0,
		hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		base_hp DOUBLE PRECISION NOT NULL DEFAULT 100,
		current_region VARCHAR(50) NOT NULL DEFAULT 'hub_nam_cuong' REFERENCES regions(id),
		x INTEGER NOT NULL DEFAULT 50,
		y INTEGER NOT NULL DEFAULT 50,
		race VARCHAR(50) NOT NULL DEFAULT 'human' REFERENCES race_dictionary(key),
		gender VARCHAR(20) NOT NULL DEFAULT '',
		constitution VARCHAR(100) NOT NULL DEFAULT '' REFERENCES constitution_dictionary(key),
		path_masteries JSONB NOT NULL DEFAULT '{}',
		active_combat JSONB DEFAULT NULL,
		body_mods JSONB NOT NULL DEFAULT '[]',
		visited_subzones JSONB NOT NULL DEFAULT '[]',
		equipped_gus JSONB NOT NULL DEFAULT '[]',
		cleared_cells JSONB NOT NULL DEFAULT '{}',
		materials_inventory JSONB NOT NULL DEFAULT '{}',
		statuses JSONB NOT NULL DEFAULT '[]',
		earned_achievements JSONB NOT NULL DEFAULT '[]',
		growth_data JSONB NOT NULL DEFAULT '{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}',
		combat_settings JSONB NOT NULL DEFAULT '{}',
		immortal_state JSONB NOT NULL DEFAULT '{}',
		last_mana_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
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
		version INT NOT NULL DEFAULT 0
	);`); err != nil {
		return err
	}

	if err := execSchema("players_version_column", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='version') THEN
			ALTER TABLE players ADD COLUMN version INT NOT NULL DEFAULT 0;
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("players_fingerprint_columns", `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='register_fingerprint') THEN
			ALTER TABLE players ADD COLUMN register_fingerprint VARCHAR(255) NOT NULL DEFAULT '';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='register_ip') THEN
			ALTER TABLE players ADD COLUMN register_ip VARCHAR(50) NOT NULL DEFAULT '';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='last_login_fingerprint') THEN
			ALTER TABLE players ADD COLUMN last_login_fingerprint VARCHAR(255) NOT NULL DEFAULT '';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='last_login_ip') THEN
			ALTER TABLE players ADD COLUMN last_login_ip VARCHAR(50) NOT NULL DEFAULT '';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='last_login_time') THEN
			ALTER TABLE players ADD COLUMN last_login_time TIMESTAMP WITH TIME ZONE DEFAULT NULL;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='active_session_id') THEN
			ALTER TABLE players ADD COLUMN active_session_id VARCHAR(64) NOT NULL DEFAULT '';
		END IF;
	END $$;`); err != nil {
		return err
	}

	if err := execSchema("players_stones_type_upgrade", `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='primeval_stones' AND data_type='double precision') THEN
			ALTER TABLE players ALTER COLUMN primeval_stones TYPE BIGINT USING COALESCE(primeval_stones, 0)::BIGINT;
		END IF;
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='players' AND column_name='immortal_stones' AND data_type='double precision') THEN
			ALTER TABLE players ALTER COLUMN immortal_stones TYPE BIGINT USING COALESCE(immortal_stones, 0)::BIGINT;
		END IF;
	END $$;`); err != nil {
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

	// Google ids are NULL/empty for most accounts; the partial unique index
	// enforces uniqueness only for real google ids.
	if err := execSchema("players_google_id_idx", `
	CREATE UNIQUE INDEX IF NOT EXISTS players_google_id_idx ON players (google_id) WHERE google_id IS NOT NULL AND google_id != '';`); err != nil {
		return err
	}
	if err := execSchema("players_region_xy_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_region_xy ON players (current_region, x, y);`); err != nil {
		return err
	}
	if err := execSchema("players_party_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_party_id ON players (party_id);`); err != nil {
		return err
	}
	if err := execSchema("players_username_trgm_idx", `
	CREATE INDEX IF NOT EXISTS idx_players_username_trgm ON players USING gin (public.immutable_unaccent(username) gin_trgm_ops);`); err != nil {
		return err
	}

	return nil
}

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
		deposit_amount_stones INTEGER NOT NULL DEFAULT 0,
		milestone1_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone2_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone3_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone4_rewarded BOOLEAN NOT NULL DEFAULT false,
		milestone5_rewarded BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("referrals_milestone_cols", `
	ALTER TABLE referrals ADD COLUMN IF NOT EXISTS milestone4_rewarded BOOLEAN NOT NULL DEFAULT false;
	ALTER TABLE referrals ADD COLUMN IF NOT EXISTS milestone5_rewarded BOOLEAN NOT NULL DEFAULT false;
	`); err != nil {
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

	return nil
}
