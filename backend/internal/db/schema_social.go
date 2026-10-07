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
		channel VARCHAR(50) NOT NULL DEFAULT 'private' CHECK (channel IN ('private','world','general','sect','system','region')),
		sect_id VARCHAR(50) REFERENCES sects(id) ON DELETE CASCADE,
		chat_scope VARCHAR(120) NOT NULL DEFAULT ''
	);`); err != nil {
		return err
	}
	if err := execSchema("messages_chat_scope", `
	ALTER TABLE messages ADD COLUMN IF NOT EXISTS chat_scope VARCHAR(120) NOT NULL DEFAULT '';
	ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_channel_check;
	ALTER TABLE messages ADD CONSTRAINT messages_channel_check CHECK (channel IN ('private','world','general','sect','system','region'));
	CREATE INDEX IF NOT EXISTS idx_messages_region_chat ON messages (chat_scope, created_at DESC) WHERE channel = 'region';`); err != nil {
		return err
	}
	if err := execSchema("private_chat_connections", `
	CREATE TABLE IF NOT EXISTS private_chat_connections (
		player_a UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		player_b UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (player_a, player_b),
		CHECK (player_a < player_b)
	);
	INSERT INTO private_chat_connections (player_a, player_b)
	SELECT DISTINCT LEAST(sender_id, receiver_id), GREATEST(sender_id, receiver_id)
	FROM messages
	WHERE channel = 'private'
	  AND sender_id <> receiver_id
	  AND sender_id::text NOT IN ('00000000-0000-0000-0000-000000000000','11111111-1111-1111-1111-111111111111')
	  AND receiver_id::text NOT IN ('00000000-0000-0000-0000-000000000000','11111111-1111-1111-1111-111111111111')
	ON CONFLICT DO NOTHING;
	CREATE INDEX IF NOT EXISTS idx_private_messages_sender_receiver_created ON messages (sender_id, receiver_id, created_at DESC) WHERE channel = 'private';
	CREATE INDEX IF NOT EXISTS idx_private_messages_receiver_sender_created ON messages (receiver_id, sender_id, created_at DESC) WHERE channel = 'private';`); err != nil {
		return err
	}
	if err := execSchema("messages_sender_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages (sender_id);`); err != nil {
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
	if err := execSchema("messages_sect_channel_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_messages_sect_channel_created ON messages (receiver_id, channel, sect_id, created_at DESC);`); err != nil {
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
		actor_username TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		effective_roles JSONB NOT NULL DEFAULT '[]',
		effective_permission TEXT NOT NULL DEFAULT '',
		entity_type TEXT,
		entity_id TEXT,
		change_data JSONB NOT NULL DEFAULT '{}',
		reason TEXT NOT NULL DEFAULT '',
		request_id TEXT NOT NULL DEFAULT '',
		request_summary TEXT,
		metadata JSONB DEFAULT '{}',
		ip_address TEXT,
		success BOOLEAN NOT NULL DEFAULT false,
		error_message TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
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
	// Composite index for the most common query: player's logs ordered by time DESC.
	// Covers WHERE player_id = $1 ORDER BY created_at DESC without merging two indexes.
	if err := execSchema("audit_logs_player_created_idx", `
	CREATE INDEX IF NOT EXISTS idx_audit_logs_player_created ON audit_logs (player_id, created_at DESC);`); err != nil {
		return err
	}

	if err := execSchema("notifications", `
	CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`); err != nil {
		return err
	}
	if err := execSchema("notifications_player_idx", `
	CREATE INDEX IF NOT EXISTS idx_notifications_player ON notifications (player_id, created_at DESC);`); err != nil {
		return err
	}
	if err := execSchema("notifications_unread_idx", `
	CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications (player_id, is_read) WHERE is_read = FALSE;`); err != nil {
		return err
	}

	return nil
}
