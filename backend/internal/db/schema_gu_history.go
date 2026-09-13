package db

// schemaGuHistory creates the gu_history_logs table and indexes.
func schemaGuHistory() error {
	if err := execSchema("gu_history_logs", `
	CREATE TABLE IF NOT EXISTS gu_history_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		instance_id VARCHAR(64) NOT NULL,
		gu_id VARCHAR(64) NOT NULL,
		rank INT NOT NULL,
		event_type VARCHAR(50) NOT NULL,
		actor_id UUID REFERENCES players(id) ON DELETE SET NULL,
		target_id UUID REFERENCES players(id) ON DELETE SET NULL,
		summary TEXT NOT NULL,
		metadata JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}

	if err := execSchema("gu_history_instance_idx", `
	CREATE INDEX IF NOT EXISTS idx_gu_history_instance_id ON gu_history_logs (instance_id, created_at DESC);`); err != nil {
		return err
	}

	if err := execSchema("gu_history_gu_id_idx", `
	CREATE INDEX IF NOT EXISTS idx_gu_history_gu_id ON gu_history_logs (gu_id, rank);`); err != nil {
		return err
	}

	return nil
}
