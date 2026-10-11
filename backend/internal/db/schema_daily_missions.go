package db

func schemaDailyMissions() error {
	if err := execSchema("player_daily_missions", `
	CREATE TABLE IF NOT EXISTS player_daily_missions (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		day_key DATE NOT NULL,
		mission_id VARCHAR(80) NOT NULL,
		rank_snapshot INTEGER NOT NULL,
		mission_type VARCHAR(40) NOT NULL,
		target_id VARCHAR(100) NOT NULL DEFAULT '',
		target_amount BIGINT NOT NULL,
		progress BIGINT NOT NULL DEFAULT 0,
		reward_type VARCHAR(30) NOT NULL,
		reward_amount BIGINT NOT NULL,
		completed_at TIMESTAMPTZ,
		claimed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, day_key, mission_id),
		CHECK (target_amount > 0),
		CHECK (progress >= 0),
		CHECK (reward_amount >= 0)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_player_daily_missions_day", `CREATE INDEX IF NOT EXISTS idx_player_daily_missions_day ON player_daily_missions(day_key);`); err != nil {
		return err
	}
	if err := execSchema("player_daily_mission_events", `
	CREATE TABLE IF NOT EXISTS player_daily_mission_events (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		day_key DATE NOT NULL,
		source_event_id VARCHAR(180) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_id, day_key, source_event_id)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_player_daily_mission_events_day", `CREATE INDEX IF NOT EXISTS idx_player_daily_mission_events_day ON player_daily_mission_events(day_key);`); err != nil {
		return err
	}
	if err := execSchema("player_daily_mission_rank_snapshots", `
	CREATE TABLE IF NOT EXISTS player_daily_mission_rank_snapshots (
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		day_key DATE NOT NULL,
		rank_snapshot INTEGER NOT NULL,
		PRIMARY KEY (player_id, day_key)
	);`); err != nil {
		return err
	}
	if err := execSchema("idx_player_daily_mission_rank_snapshots_day", `CREATE INDEX IF NOT EXISTS idx_player_daily_mission_rank_snapshots_day ON player_daily_mission_rank_snapshots(day_key);`); err != nil {
		return err
	}
	if err := execSchema("daily_mission_cycle_state", `
	CREATE TABLE IF NOT EXISTS daily_mission_cycle_state (
		id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
		day_key DATE NOT NULL
	);`); err != nil {
		return err
	}
	return nil
}
