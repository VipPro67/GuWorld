package db

// Messaging, feedback and audit schema.

func schemaSocial() error {
	if err := execSchema("messages", `
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sender_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		receiver_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		channel VARCHAR(50) NOT NULL DEFAULT 'private' CHECK (channel IN ('private','world','general','sect','system')),
		sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE CASCADE
	);`); err != nil {
		return err
	}
	if err := execSchema("messages_sender_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages (sender_id);`); err != nil {
		return err
	}
	if err := execSchema("messages_receiver_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_receiver ON messages (receiver_id);`); err != nil {
		return err
	}
	if err := execSchema("messages_sect_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_sect ON messages (sect_id);`); err != nil {
		return err
	}
	if err := execSchema("messages_receiver_channel_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_receiver_channel_created ON messages (receiver_id, channel, created_at DESC);`); err != nil {
		return err
	}

	if err := execSchema("feedback", `
	CREATE TABLE IF NOT EXISTS feedback (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		category VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		total_votes INTEGER NOT NULL DEFAULT 0,
		developer_response TEXT DEFAULT NULL,
		rewarded BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("feedback_user_idx", `
	CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback (user_id);`); err != nil {
		return err
	}
	if err := execSchema("feedback_status_idx", `
	CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback (status);`); err != nil {
		return err
	}

	if err := execSchema("feedback_votes", `
	CREATE TABLE IF NOT EXISTS feedback_votes (
		feedback_id UUID NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		vote_type INTEGER NOT NULL,
		PRIMARY KEY (feedback_id, user_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("feedback_comments", `
	CREATE TABLE IF NOT EXISTS feedback_comments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		feedback_id UUID NOT NULL REFERENCES feedback(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		return err
	}
	if err := execSchema("feedback_comments_feedback_idx", `
	CREATE INDEX IF NOT EXISTS idx_feedback_comments_feedback ON feedback_comments (feedback_id);`); err != nil {
		return err
	}

	if err := execSchema("audit_logs", `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID REFERENCES players(id) ON DELETE SET NULL,
		action TEXT NOT NULL,
		entity_type TEXT,
		entity_id TEXT,
		request_summary TEXT,
		metadata JSONB DEFAULT '{}',
		ip_address TEXT,
		success BOOLEAN NOT NULL DEFAULT false,
		error_message TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("audit_logs_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_audit_logs_player ON audit_logs (player_id);`); err != nil {
		return err
	}
	if err := execSchema("audit_logs_action_idx", `
	CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);`); err != nil {
		return err
	}
	if err := execSchema("audit_logs_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs (created_at);`); err != nil {
		return err
	}

	return nil
}
