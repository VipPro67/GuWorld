package db

// schemaModeration stores device-level access blocks (IP / browser
// fingerprint). Account bans live on players.is_banned; this table lets an
// admin or an auto-ban rule stop the offending device from registering a fresh
// account to evade the account ban.
func schemaModeration() error {
	return execSchema("player_access_blocks", `
	CREATE TABLE IF NOT EXISTS player_access_blocks (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		ip VARCHAR(50) NOT NULL DEFAULT '',
		fingerprint VARCHAR(255) NOT NULL DEFAULT '',
		reason TEXT NOT NULL DEFAULT '',
		banned_by UUID REFERENCES players(id) ON DELETE SET NULL,
		banned_until TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_player_access_blocks_fingerprint
		ON player_access_blocks (fingerprint) WHERE fingerprint <> '';
	CREATE INDEX IF NOT EXISTS idx_player_access_blocks_ip
		ON player_access_blocks (ip) WHERE ip <> '';`)
}
