package db

// Scouting schema: player_tracking_marks, world_residual_traces, and columns on players.

func schemaScouting() error {
	if err := execSchema("player_tracking_marks", `
	CREATE TABLE IF NOT EXISTS player_tracking_marks (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tracker_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		target_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		mark_type VARCHAR(30) NOT NULL DEFAULT 'wisdom',
		power FLOAT8 NOT NULL DEFAULT 0,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_tracking_marks_tracker ON player_tracking_marks(tracker_id, expires_at);
	CREATE INDEX IF NOT EXISTS idx_tracking_marks_target ON player_tracking_marks(target_id);`); err != nil {
		return err
	}

	if err := execSchema("world_residual_traces", `
	CREATE TABLE IF NOT EXISTS world_residual_traces (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
		x INT NOT NULL,
		y INT NOT NULL,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		action_type VARCHAR(30) NOT NULL DEFAULT 'move',
		dominant_path VARCHAR(30) NOT NULL DEFAULT '',
		dao_pulse_strength FLOAT8 NOT NULL DEFAULT 0,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_residual_traces_region_xy ON world_residual_traces(region_id, x, y, expires_at);`); err != nil {
		return err
	}

	return nil
}
