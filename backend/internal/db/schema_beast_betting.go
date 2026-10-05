package db

// Beast betting schema for entertainment.
func schemaBeastBetting() error {
	if err := execSchema("beast_betting_rounds", `
	CREATE TABLE IF NOT EXISTS beast_betting_rounds (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','LOCKED','FINISHED')),
		winner VARCHAR(20) NOT NULL DEFAULT '',
		team_a_name VARCHAR(50) NOT NULL DEFAULT 'Tả Thanh Long',
		team_b_name VARCHAR(50) NOT NULL DEFAULT 'Hữu Bạch Hổ',
		total_bet_a DOUBLE PRECISION DEFAULT 0,
		total_bet_b DOUBLE PRECISION DEFAULT 0,
		battle_log TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE UNIQUE DEFAULT CURRENT_TIMESTAMP,
		locked_at TIMESTAMP WITH TIME ZONE NULL,
		resolved_at TIMESTAMP WITH TIME ZONE NULL
	);`); err != nil {
		return err
	}

	if err := execSchema("beast_bets", `
	CREATE TABLE IF NOT EXISTS beast_bets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		round_id UUID NOT NULL REFERENCES beast_betting_rounds(id) ON DELETE CASCADE,
		player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		team VARCHAR(20) NOT NULL CHECK (team IN ('team_a','team_b')),
		amount DOUBLE PRECISION NOT NULL CHECK (amount > 0),
		payout DOUBLE PRECISION DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_player_round_bet UNIQUE (player_id, round_id)
	);`); err != nil {
		return err
	}

	if err := execSchema("idx_beast_bets_round", `
	CREATE INDEX IF NOT EXISTS idx_beast_bets_round ON beast_bets(round_id);`); err != nil {
		return err
	}

	if err := execSchema("idx_beast_bets_player", `
	CREATE INDEX IF NOT EXISTS idx_beast_bets_player ON beast_bets(player_id);`); err != nil {
		return err
	}

	return nil
}
